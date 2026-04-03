package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"tango/internal/orchestrator"
)

const launchdLabel = "com.tango.daemon"
const systemdUnit = "tango-daemon.service"

// Install generates and installs the appropriate service file for the current OS.
func Install(composeFile, projectName string) error {
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve binary path: %w", err)
	}
	binaryPath, err = filepath.EvalSymlinks(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return installLaunchd(binaryPath, composeFile, projectName)
	case "linux":
		return installSystemd(binaryPath, composeFile, projectName)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// Uninstall removes the service file and stops the daemon.
func Uninstall() error {
	switch runtime.GOOS {
	case "darwin":
		return uninstallLaunchd()
	case "linux":
		return uninstallSystemd()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// ── macOS launchd ──────────────────────────────────

func launchdPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
}

func installLaunchd(binaryPath, composeFile, projectName string) error {
	logPath := orchestrator.LogPath()

	var envEntries string
	if composeFile != "" {
		envEntries += fmt.Sprintf(`
		<key>TANGO_COMPOSE_FILE</key>
		<string>%s</string>`, composeFile)
	}
	if projectName != "" {
		envEntries += fmt.Sprintf(`
		<key>TANGO_PROJECT_NAME</key>
		<string>%s</string>`, projectName)
	}

	var envSection string
	if envEntries != "" {
		envSection = fmt.Sprintf(`
	<key>EnvironmentVariables</key>
	<dict>%s
	</dict>`, envEntries)
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>daemon</string>
		<string>run</string>
	</array>%s
	<key>KeepAlive</key>
	<true/>
	<key>RunAtLoad</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>
`, launchdLabel, binaryPath, envSection, logPath, logPath)

	plistPath := launchdPlistPath()
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return err
	}

	cmd := exec.Command("launchctl", "load", plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

func uninstallLaunchd() error {
	plistPath := launchdPlistPath()

	cmd := exec.Command("launchctl", "unload", plistPath)
	cmd.Run() // ignore error if not loaded

	return os.Remove(plistPath)
}

// ── Linux systemd ──────────────────────────────────

func systemdUnitPath() string {
	return filepath.Join("/etc", "systemd", "system", systemdUnit)
}

func installSystemd(binaryPath, composeFile, projectName string) error {
	if os.Geteuid() != 0 {
		return errors.New("system-wide daemon install requires root privileges")
	}

	serviceUser, serviceHome, err := serviceUserAndHome()
	if err != nil {
		return err
	}
	logPath := filepath.Join(serviceHome, ".config", "tango", "daemon.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return fmt.Errorf("create daemon log dir: %w", err)
	}

	var envLines string
	if composeFile != "" {
		envLines += fmt.Sprintf("Environment=TANGO_COMPOSE_FILE=%s\n", composeFile)
	}
	if projectName != "" {
		envLines += fmt.Sprintf("Environment=TANGO_PROJECT_NAME=%s\n", projectName)
	}
	envLines += fmt.Sprintf("Environment=HOME=%s\n", serviceHome)

	unit := fmt.Sprintf(`[Unit]
Description=Tango Daemon - Docker health check and auto-restart
Wants=docker.service network-online.target
After=docker.service network-online.target

[Service]
Type=simple
ExecStart=%s daemon run
%sRestart=on-failure
RestartSec=5
User=%s
Group=%s
StandardOutput=append:%s
StandardError=append:%s

[Install]
WantedBy=multi-user.target
`, binaryPath, envLines, serviceUser, serviceUser, logPath, logPath)

	unitPath := systemdUnitPath()
	if err := os.MkdirAll(filepath.Dir(unitPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(unitPath, []byte(unit), 0644); err != nil {
		return err
	}

	// Reload and enable
	cmds := [][]string{
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", "--now", systemdUnit},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s failed: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(output)), err)
		}
	}

	return nil
}

func uninstallSystemd() error {
	if os.Geteuid() != 0 {
		return errors.New("system-wide daemon uninstall requires root privileges")
	}

	cmd := exec.Command("systemctl", "disable", "--now", systemdUnit)
	cmd.Run() // ignore error

	return os.Remove(systemdUnitPath())
}

func serviceUserAndHome() (string, string, error) {
	username := os.Getenv("SUDO_USER")
	if username == "" {
		username = os.Getenv("USER")
	}
	if username == "" {
		return "", "", errors.New("could not determine service user")
	}

	u, err := user.Lookup(username)
	if err != nil {
		return "", "", fmt.Errorf("lookup service user %q: %w", username, err)
	}
	if u.HomeDir == "" {
		return "", "", fmt.Errorf("user %q does not have a home directory", username)
	}

	return username, u.HomeDir, nil
}
