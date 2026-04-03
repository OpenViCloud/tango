package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"tango/internal/orchestrator"
	"tango/internal/orchestrator/compose"

	"github.com/spf13/cobra"
)

var (
	serviceComposeFile string
	serviceProjectName string
	serviceLogsTail    int
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage Docker Compose services",
}

func newDriver() orchestrator.Driver {
	composeFile := serviceComposeFile
	projectName := serviceProjectName

	// Fall back to daemon config if flags not set
	if composeFile == "" || projectName == "" {
		daemonCfg := orchestrator.LoadConfig()
		if composeFile == "" {
			composeFile = daemonCfg.ComposeFile
		}
		if projectName == "" {
			projectName = daemonCfg.ProjectName
		}
	}

	return compose.New(composeFile, projectName)
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all services and their status",
	Run: func(cmd *cobra.Command, args []string) {
		driver := newDriver()
		defer driver.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Check Docker first
		if err := driver.Ping(ctx); err != nil {
			fmt.Println("Docker is not reachable:", err)
			os.Exit(1)
		}

		services, err := driver.ListServices(ctx)
		if err != nil {
			fmt.Println("Failed to list services:", err)
			os.Exit(1)
		}

		if len(services) == 0 {
			fmt.Println("No services found.")
			return
		}

		fmt.Printf("%-20s %-12s %-12s %-15s %s\n", "SERVICE", "STATE", "HEALTH", "IMAGE", "PORTS")
		fmt.Printf("%-20s %-12s %-12s %-15s %s\n", "-------", "-----", "------", "-----", "-----")
		for _, svc := range services {
			image := svc.Image
			if len(image) > 15 {
				image = image[:12] + "..."
			}
			ports := ""
			if len(svc.Ports) > 0 {
				ports = svc.Ports[0]
				if len(svc.Ports) > 1 {
					ports += fmt.Sprintf(" (+%d)", len(svc.Ports)-1)
				}
			}
			fmt.Printf("%-20s %-12s %-12s %-15s %s\n",
				svc.Name, svc.State, svc.Health, image, ports)
		}
	},
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show detailed status for a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		driver := newDriver()
		defer driver.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		svc, err := driver.ServiceStatus(ctx, args[0])
		if err != nil {
			fmt.Println("Failed to get status:", err)
			os.Exit(1)
		}

		fmt.Printf("Service     : %s\n", svc.Name)
		fmt.Printf("State       : %s\n", svc.State)
		fmt.Printf("Health      : %s\n", svc.Health)
		fmt.Printf("Container   : %s\n", svc.ContainerID)
		fmt.Printf("Image       : %s\n", svc.Image)
		if svc.Uptime > 0 {
			fmt.Printf("Uptime      : %s\n", formatDuration(svc.Uptime))
		}
		if svc.ExitCode != 0 {
			fmt.Printf("Exit Code   : %d\n", svc.ExitCode)
		}
		if len(svc.Ports) > 0 {
			fmt.Printf("Ports       : %s\n", svc.Ports[0])
			for _, p := range svc.Ports[1:] {
				fmt.Printf("              %s\n", p)
			}
		}
	},
}

var serviceRestartCmd = &cobra.Command{
	Use:   "restart <name>",
	Short: "Restart a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		driver := newDriver()
		defer driver.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		fmt.Printf("Restarting %s...\n", args[0])
		if err := driver.RestartService(ctx, args[0]); err != nil {
			fmt.Println("Failed to restart:", err)
			os.Exit(1)
		}
		fmt.Printf("Service %s restarted.\n", args[0])
	},
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		driver := newDriver()
		defer driver.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		fmt.Printf("Stopping %s...\n", args[0])
		if err := driver.StopService(ctx, args[0]); err != nil {
			fmt.Println("Failed to stop:", err)
			os.Exit(1)
		}
		fmt.Printf("Service %s stopped.\n", args[0])
	},
}

var serviceStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a stopped service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		driver := newDriver()
		defer driver.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		fmt.Printf("Starting %s...\n", args[0])
		if err := driver.StartService(ctx, args[0]); err != nil {
			fmt.Println("Failed to start:", err)
			os.Exit(1)
		}
		fmt.Printf("Service %s started.\n", args[0])
	},
}

var serviceLogsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show recent logs for a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		driver := newDriver()
		defer driver.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		reader, err := driver.ServiceLogs(ctx, args[0], serviceLogsTail)
		if err != nil {
			fmt.Println("Failed to get logs:", err)
			os.Exit(1)
		}
		defer func() {
			if err := reader.Close(); err != nil {
				slog.Debug("service logs stream closed", "err", err)
			}
		}()

		io.Copy(os.Stdout, reader)
	},
}

func init() {
	serviceCmd.PersistentFlags().StringVarP(&serviceComposeFile, "file", "f", "", "Path to docker-compose.yml")
	serviceCmd.PersistentFlags().StringVarP(&serviceProjectName, "project", "p", "", "Compose project name")
	serviceLogsCmd.Flags().IntVar(&serviceLogsTail, "tail", 50, "Number of log lines to show")

	serviceCmd.AddCommand(serviceListCmd, serviceStatusCmd, serviceRestartCmd, serviceStopCmd, serviceStartCmd, serviceLogsCmd)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}
