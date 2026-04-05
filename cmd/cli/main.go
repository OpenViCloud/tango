package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tango/internal/orchestrator"
	"tango/internal/orchestrator/compose"
	"tango/internal/orchestrator/daemon"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
)

// ── Root ──────────────────────────────────────────

var rootCmd = &cobra.Command{
	Use:   "tango",
	Short: "Manage Docker services, health monitoring, and runtime operations",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(renderRootScreen(cmd))
	},
}

// ── tango version ──────────────────────────────────

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tango version %s (%s)\n", version, commit)
	},
}

// ── tango status ───────────────────────────────────

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show overall orchestration status",
	Run: func(cmd *cobra.Command, args []string) {
		status, err := daemon.ReadStatus()
		if err != nil {
			if daemon.IsRunning() {
				fmt.Println("⚠️  Daemon is running, but no status is available yet.")
				return
			}
			fmt.Println("❌ Daemon is not running and no status file is available.")
			os.Exit(1)
		}

		fmt.Printf("State      : %s\n", status.State)
		fmt.Printf("Daemon PID : %d\n", status.PID)
		fmt.Printf("Last Check : %s\n", status.LastCheck.Format("2006-01-02 15:04:05"))
		fmt.Printf("Services   : %d tracked\n", len(status.Services))

		if status.State != "ok" {
			os.Exit(1)
		}
	},
}

// ── tango uninstall ───────────────────────────────

var uninstallPurge bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Tango CLI, daemon, and all config files",
	Run: func(cmd *cobra.Command, args []string) {
		binaryPath, _ := os.Executable()
		binaryPath, _ = filepath.EvalSymlinks(binaryPath)
		configDir := orchestrator.ConfigDir()
		daemonCfg := orchestrator.LoadConfig()
		runtimeDir := runtimeDirFromComposeFile(daemonCfg.ComposeFile)

		fmt.Println("⚠️  This will remove:")
		fmt.Printf("   • Binary:  %s\n", binaryPath)
		fmt.Printf("   • Config:  %s/\n", configDir)
		fmt.Println("   • Daemon:  system service (systemd/launchd)")
		fmt.Println("   • Docker:  all Tango containers (volumes kept)")
		if uninstallPurge {
			if runtimeDir != "" {
				fmt.Printf("   • Runtime: %s/\n", runtimeDir)
			}
			fmt.Println("   • Docker:  volumes and images removed (--purge)")
		} else {
			if runtimeDir != "" {
				fmt.Printf("   Runtime directory will NOT be removed: %s/\n", runtimeDir)
			}
			fmt.Println("   Use --purge to also remove volumes, images, and the runtime directory.")
		}
		fmt.Println()
		fmt.Print("Type 'yes' to confirm: ")

		var confirm string
		fmt.Scanln(&confirm)

		if confirm != "yes" {
			fmt.Println("Cancelled.")
			return
		}

		// 1. Stop Tango containers (always). With --purge also remove volumes and images.
		{
			msg := "Stopping containers... "
			if uninstallPurge {
				msg = "Stopping containers and removing volumes/images... "
			}
			fmt.Print(msg)
			driver := compose.New(daemonCfg.ComposeFile, daemonCfg.ProjectName)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			if err := driver.Down(ctx, uninstallPurge); err != nil {
				fmt.Printf("failed (%v)\n", err)
			} else {
				fmt.Println("done")
			}
			cancel()
			driver.Close()
		}

		// 2. Stop and remove daemon service
		fmt.Print("Removing daemon service... ")
		if err := daemon.Uninstall(); err != nil {
			fmt.Printf("skipped (%v)\n", err)
		} else {
			fmt.Println("done")
		}

		// 3. Remove config directory
		fmt.Print("Removing config files... ")
		if err := os.RemoveAll(configDir); err != nil {
			fmt.Printf("failed (%v)\n", err)
		} else {
			fmt.Println("done")
		}

		// 4. Remove runtime directory after purge tears down the stack.
		if uninstallPurge && runtimeDir != "" {
			fmt.Print("Removing runtime files... ")
			if err := os.RemoveAll(runtimeDir); err != nil {
				fmt.Printf("failed (%v)\n", err)
			} else {
				fmt.Println("done")
			}
		}

		// 5. Remove binary (self-delete)
		fmt.Print("Removing binary... ")
		if err := os.Remove(binaryPath); err != nil {
			fmt.Printf("failed (%v)\n", err)
			fmt.Println("   You can remove it manually: rm", binaryPath)
		} else {
			fmt.Println("done")
		}

		fmt.Println()
		fmt.Println("✅ Tango has been uninstalled.")
	},
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallPurge, "purge", false, "Also remove all Docker containers, volumes, and images")
}

func runtimeDirFromComposeFile(composeFile string) string {
	if composeFile == "" {
		return ""
	}

	clean := filepath.Clean(composeFile)
	dir := filepath.Dir(clean)
	if dir == "." || dir == "/" {
		return ""
	}

	// Only remove known Tango runtime directories, not arbitrary compose parents.
	if dir == "/opt/tango" || strings.HasPrefix(dir, "/opt/tango/") {
		return dir
	}

	return ""
}

// ── main ──────────────────────────────────────────

func main() {
	rootCmd.AddCommand(versionCmd, statusCmd, daemonCmd, serviceCmd, uninstallCmd, swarmCmd)
	rootCmd.Long = renderRootScreen(rootCmd)
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
