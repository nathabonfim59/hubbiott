package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information (can be set at build time)
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Display the version, git commit, and build date of Hubbiott.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Hubbiott %s\n", Version)
		fmt.Printf("  Git commit: %s\n", GitCommit)
		fmt.Printf("  Build date: %s\n", BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
