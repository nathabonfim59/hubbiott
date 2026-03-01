package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/nathabonfim59/hubbiott/internal/service"
	"github.com/spf13/cobra"
)

var (
	// Flags for service command
	serviceMode     string
	serviceFollow   bool
	serviceLines    int
	serviceSystem   bool
	serviceUserMode bool
)

// serviceCmd represents the service command
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage Hubbiott systemd service",
	Long: `Manage the Hubbiott systemd service.

This command provides subcommands to start, stop, restart, and check
the status of the Hubbiott systemd service. It supports both user-level
and system-level services.

Examples:
  hubbiott service start          # Start the service
  hubbiott service stop           # Stop the service
  hubbiott service status         # Check service status
  hubbiott service logs           # View recent logs
  hubbiott service logs -f        # Follow logs in real-time
  hubbiott service logs -n 100    # View last 100 log lines
  hubbiott service --system start # Start system service (requires sudo)`,
}

// serviceStartCmd represents the service start command
var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Hubbiott service",
	Long:  `Start the Hubbiott systemd service.`,
	Run: func(cmd *cobra.Command, args []string) {
		mgr := getServiceManager()
		if err := mgr.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service started successfully")
	},
}

// serviceStopCmd represents the service stop command
var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Hubbiott service",
	Long:  `Stop the Hubbiott systemd service.`,
	Run: func(cmd *cobra.Command, args []string) {
		mgr := getServiceManager()
		if err := mgr.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service stopped successfully")
	},
}

// serviceRestartCmd represents the service restart command
var serviceRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the Hubbiott service",
	Long:  `Restart the Hubbiott systemd service.`,
	Run: func(cmd *cobra.Command, args []string) {
		mgr := getServiceManager()
		if err := mgr.Restart(); err != nil {
			fmt.Fprintf(os.Stderr, "Error restarting service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service restarted successfully")
	},
}

// serviceStatusCmd represents the service status command
var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Hubbiott service status",
	Long:  `Display the current status of the Hubbiott systemd service.`,
	Run: func(cmd *cobra.Command, args []string) {
		mgr := getServiceManager()
		status, err := mgr.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting service status: %v\n", err)
			os.Exit(1)
		}

		// Display status
		fmt.Printf("Service: %s\n", status.Name)
		fmt.Printf("Mode:    %s\n", getServiceMode())
		fmt.Printf("State:   %s (%s)\n", status.State, status.SubState)
		fmt.Printf("Running: %t\n", status.Active)
		fmt.Printf("Enabled: %t\n", status.Enabled)
		if status.PID > 0 {
			fmt.Printf("PID:     %d\n", status.PID)
		}
	},
}

// serviceEnableCmd represents the service enable command
var serviceEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable Hubbiott service to start on boot",
	Long:  `Enable the Hubbiott systemd service to start automatically on boot.`,
	Run: func(cmd *cobra.Command, args []string) {
		mgr := getServiceManager()
		if err := mgr.Enable(); err != nil {
			fmt.Fprintf(os.Stderr, "Error enabling service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service enabled successfully")
	},
}

// serviceDisableCmd represents the service disable command
var serviceDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable Hubbiott service from starting on boot",
	Long:  `Disable the Hubbiott systemd service from starting automatically on boot.`,
	Run: func(cmd *cobra.Command, args []string) {
		mgr := getServiceManager()
		if err := mgr.Disable(); err != nil {
			fmt.Fprintf(os.Stderr, "Error disabling service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service disabled successfully")
	},
}

// serviceLogsCmd represents the service logs command
var serviceLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View Hubbiott service logs",
	Long: `View logs from the Hubbiott systemd service using journalctl.

Use -f to follow logs in real-time, or -n to specify the number of lines.`,
	Run: func(cmd *cobra.Command, args []string) {
		mgr := getServiceManager()

		if serviceFollow {
			// Follow logs in real-time
			if err := mgr.FollowLogs(); err != nil {
				fmt.Fprintf(os.Stderr, "Error following logs: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Get logs
		logs, err := mgr.LogsRaw(serviceLines)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting logs: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(logs)
	},
}

// getServiceMode returns the service mode based on flags.
func getServiceMode() service.Mode {
	if serviceSystem {
		return service.ModeSystem
	}
	if serviceUserMode {
		return service.ModeUser
	}
	// Auto-detect
	return ""
}

// getServiceManager creates a service manager with the appropriate mode.
func getServiceManager() *service.Manager {
	return service.NewManager(getServiceMode())
}

func init() {
	rootCmd.AddCommand(serviceCmd)

	// Add subcommands
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceRestartCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	serviceCmd.AddCommand(serviceEnableCmd)
	serviceCmd.AddCommand(serviceDisableCmd)
	serviceCmd.AddCommand(serviceLogsCmd)

	// Mode flags (mutually exclusive)
	serviceCmd.PersistentFlags().BoolVar(&serviceSystem, "system", false, "Manage system service (requires root/sudo)")
	serviceCmd.PersistentFlags().BoolVar(&serviceUserMode, "user", false, "Manage user service")

	// Mark flags as mutually exclusive
	serviceCmd.MarkFlagsMutuallyExclusive("system", "user")

	// Log-specific flags
	serviceLogsCmd.Flags().BoolVarP(&serviceFollow, "follow", "f", false, "Follow logs in real-time")
	serviceLogsCmd.Flags().IntVarP(&serviceLines, "lines", "n", 50, "Number of log lines to show")
}

// Helper function for strconv (avoid unused import)
var _ = strconv.Itoa
