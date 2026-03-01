package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `Manage Hubbiott configuration including viewing,
editing, and validating configuration settings.`,
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View current configuration",
	Long:  `Display the current configuration settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		if cfgFile == "" {
			fmt.Println("No config file specified, using defaults and environment variables")
		} else {
			fmt.Printf("Config file: %s\n", cfgFile)
		}
		fmt.Println("\nCurrent configuration:")
		settings := viper.AllSettings()
		if len(settings) == 0 {
			fmt.Println("  (no settings configured)")
		} else {
			for key, value := range settings {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new configuration file",
	Long:  `Create a new configuration file with default values.`,
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		configPath := home + "/.hubbiott.yaml"
		if cfgFile != "" {
			configPath = cfgFile
		}

		// Check if file already exists
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config file already exists at %s\n", configPath)
			return
		}

		// Create default config
		defaultConfig := `# Hubbiott Configuration

# Server settings
server:
  port: 8080
  host: "0.0.0.0"

# Discord bot settings
discord:
  token: ""
  guild_id: ""

# Database settings
database:
  url: ""

# GitHub settings
github:
  webhook_secret: ""
`
		err = os.WriteFile(configPath, []byte(defaultConfig), 0644)
		cobra.CheckErr(err)

		fmt.Printf("Created config file at %s\n", configPath)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configInitCmd)
}
