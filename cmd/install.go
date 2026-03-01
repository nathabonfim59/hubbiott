package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
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

	// TODO: Implement service installation based on mode
	fmt.Printf("\nInstallation mode: %s\n", config.InstallMode)
	fmt.Println("\nService installation is not yet implemented.")
	fmt.Println("For now, you can run the server manually with: hubbiott serve")
}
