package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"tango/internal/orchestrator"
	"tango/internal/orchestrator/compose"
	"tango/internal/orchestrator/daemon"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the health-check daemon",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon in the background",
	Run: func(cmd *cobra.Command, args []string) {
		if daemon.IsRunning() {
			fmt.Println("Daemon is already running (PID:", daemon.ReadPID(), ")")
			return
		}

		// Launch "tango daemon run" as a detached process
		exe, err := os.Executable()
		if err != nil {
			fmt.Println("Failed to resolve executable:", err)
			os.Exit(1)
		}

		// Ensure log directory exists
		os.MkdirAll(orchestrator.ConfigDir(), 0700)
		logFile, err := os.OpenFile(orchestrator.LogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			fmt.Println("Failed to open log file:", err)
			os.Exit(1)
		}

		proc := exec.Command(exe, "daemon", "run")
		proc.Stdout = logFile
		proc.Stderr = logFile
		detachProcess(proc)

		if err := proc.Start(); err != nil {
			fmt.Println("Failed to start daemon:", err)
			os.Exit(1)
		}
		logFile.Close()

		// Wait briefly for PID file
		for i := 0; i < 10; i++ {
			time.Sleep(200 * time.Millisecond)
			if daemon.IsRunning() {
				fmt.Println("Daemon started (PID:", daemon.ReadPID(), ")")
				fmt.Println("Log file:", orchestrator.LogPath())
				return
			}
		}

		fmt.Println("Daemon process started but PID not confirmed. Check logs:", orchestrator.LogPath())
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	Run: func(cmd *cobra.Command, args []string) {
		pid := daemon.ReadPID()
		if pid == 0 || !daemon.IsRunning() {
			fmt.Println("Daemon is not running.")
			return
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Println("Failed to find process:", err)
			os.Exit(1)
		}

		if err := stopProcess(process); err != nil {
			fmt.Println("Failed to send signal:", err)
			os.Exit(1)
		}

		// Wait for PID file to disappear
		for i := 0; i < 20; i++ {
			time.Sleep(250 * time.Millisecond)
			if !daemon.IsRunning() {
				fmt.Println("Daemon stopped.")
				return
			}
		}

		fmt.Println("Daemon may still be stopping. PID:", pid)
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon and service health status",
	Run: func(cmd *cobra.Command, args []string) {
		if !daemon.IsRunning() {
			fmt.Println("Daemon is not running.")

			// Still try to show last known status
			status, err := daemon.ReadStatus()
			if err == nil && len(status.Services) > 0 {
				fmt.Println("\nLast known status (stale):")
				printDaemonStatus(status)
			}
			return
		}

		status, err := daemon.ReadStatus()
		if err != nil {
			fmt.Println("Daemon is running (PID:", daemon.ReadPID(), ") but no status available yet.")
			return
		}

		fmt.Println("Daemon running (PID:", status.PID, ")")
		fmt.Println("Last check:", status.LastCheck.Format("2006-01-02 15:04:05"))
		fmt.Println("State:", status.State)
		fmt.Println()
		printDaemonStatus(status)
	},
}

func printDaemonStatus(status *daemon.Status) {
	if len(status.Services) == 0 {
		fmt.Println("No services tracked.")
		return
	}

	fmt.Printf("%-20s %-12s %-12s %-8s %s\n", "SERVICE", "STATE", "HEALTH", "RETRIES", "EXHAUSTED")
	fmt.Printf("%-20s %-12s %-12s %-8s %s\n", "-------", "-----", "------", "-------", "---------")
	for _, svc := range status.Services {
		exhausted := ""
		if svc.Exhausted {
			exhausted = "YES"
		}
		fmt.Printf("%-20s %-12s %-12s %-8d %s\n",
			svc.Name, svc.State, svc.Health, svc.RestartCount, exhausted)
	}
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install daemon as a system service (launchd/systemd)",
	Run: func(cmd *cobra.Command, args []string) {
		daemonCfg := orchestrator.LoadConfig()

		if err := daemon.Install(daemonCfg.ComposeFile, daemonCfg.ProjectName); err != nil {
			fmt.Println("Failed to install service:", err)
			os.Exit(1)
		}

		fmt.Println("Daemon installed as system service.")
		fmt.Println("It will start automatically on boot and restart on failure.")
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the daemon system service",
	Run: func(cmd *cobra.Command, args []string) {
		if err := daemon.Uninstall(); err != nil {
			fmt.Println("Failed to uninstall service:", err)
			os.Exit(1)
		}
		fmt.Println("Daemon service removed.")
	},
}

// daemonRunCmd is the hidden command that actually runs the daemon loop.
var daemonRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the daemon (internal, use 'daemon start' instead)",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		daemonCfg := orchestrator.LoadConfig()

		// Setup logger
		logFile, err := os.OpenFile(orchestrator.LogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		var handler slog.Handler
		if err == nil {
			handler = slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: slog.LevelDebug})
		} else {
			handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
		}
		logger := slog.New(handler)

		// Create driver
		driver := compose.New(daemonCfg.ComposeFile, daemonCfg.ProjectName)
		defer driver.Close()

		d := daemon.New(driver, daemonCfg, logger)
		if err := d.Run(context.Background()); err != nil {
			logger.Error("daemon exited with error", "err", err)
			os.Exit(1)
		}
	},
}

func init() {
	daemonCmd.AddCommand(daemonStartCmd, daemonStopCmd, daemonStatusCmd, daemonInstallCmd, daemonUninstallCmd, daemonRunCmd)
}
