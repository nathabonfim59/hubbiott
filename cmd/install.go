package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/nathabonfim59/hubbiott/internal/install"
	"github.com/nathabonfim59/hubbiott/internal/wizard"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Hubbiott as a system service",
	Long: `Install Hubbiott as a system service (systemd, launchd, etc.)
to run automatically on system startup.

This command will run an interactive wizard to collect configuration
settings before installing the service.`,
	Run: func(cmd *cobra.Command, args []string) {
		runInstall()
	},
}

var skipWizard bool

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().BoolVar(&skipWizard, "skip-wizard", false, "Skip the interactive wizard and use existing config")
}

func runInstall() {
	fmt.Println("╭─────────────────────────────────────────────╮")
	fmt.Println("│     Hubbiott Installation Wizard            │")
	fmt.Println("╰─────────────────────────────────────────────╯")
	fmt.Println()

	// Run the wizard
	w := wizard.New()
	config, err := w.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Display summary
	fmt.Println()
	fmt.Println(config.Summary())

	// Get config file path
	configPath, err := config.GetConfigPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting config path: %v\n", err)
		os.Exit(1)
	}

	// Confirm installation
	var confirm bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Proceed with installation?").
				Description(fmt.Sprintf("Config will be saved to: %s", configPath)).
				Value(&confirm),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := confirmForm.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !confirm {
		fmt.Println("Installation cancelled.")
		os.Exit(0)
	}

	// Generate config file
	fmt.Printf("\nWriting configuration to %s...\n", configPath)
	if err := config.GenerateConfigFile(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Configuration saved!")

	// Install service based on mode
	fmt.Printf("\nInstalling Hubbiott (%s mode)...\n", config.InstallMode)

	var result *install.Result
	switch config.InstallMode {
	case "user":
		installer := install.NewUserInstaller(config)
		result, err = installer.Install()
	case "system":
		installer := install.NewSystemInstaller(config)
		result, err = installer.Install()
	default:
		fmt.Fprintf(os.Stderr, "Unknown installation mode: %s\n", config.InstallMode)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during installation: %v\n", err)
		os.Exit(1)
	}

	// Display success message
	fmt.Println()
	fmt.Println("╭─────────────────────────────────────────────╮")
	fmt.Println("│         Installation Complete!              │")
	fmt.Println("╰─────────────────────────────────────────────╯")
	fmt.Println()
	fmt.Printf("Binary:     %s\n", result.BinaryPath)
	fmt.Printf("Config:     %s\n", result.ConfigPath)
	fmt.Printf("Service:    %s\n", result.ServicePath)
	fmt.Println()
	fmt.Println("The hubbiott service has been started and enabled.")
	fmt.Println()
	fmt.Println("Useful commands:")
	if config.InstallMode == "system" {
		fmt.Println("  sudo systemctl status hubbiott    # Check service status")
		fmt.Println("  sudo journalctl -u hubbiott       # View service logs")
		fmt.Println("  sudo systemctl restart hubbiott   # Restart service")
	} else {
		fmt.Println("  systemctl --user status hubbiott    # Check service status")
		fmt.Println("  systemctl --user logs hubbiott      # View service logs")
		fmt.Println("  systemctl --user restart hubbiott   # Restart service")
	}
}
