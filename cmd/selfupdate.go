package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nathabonfim59/hubbiott/internal/update"
	"github.com/spf13/cobra"
)

var (
	// Flags for selfupdate command
	selfupdateCheckOnly bool
	selfupdateVersion   string
	selfupdateTimeout   time.Duration
)

// selfupdateCmd represents the selfupdate command
var selfupdateCmd = &cobra.Command{
	Use:   "selfupdate",
	Short: "Update hubbiott to the latest version",
	Long: `Update hubbiott to the latest version from GitHub releases.

This command will:
  1. Check for the latest release on GitHub
  2. Download the new binary for your platform
  3. Verify the checksum (if available)
  4. Replace the current binary atomically

Examples:
  # Update to the latest version
  hubbiott selfupdate

  # Check if an update is available without updating
  hubbiott selfupdate --check

  # Update to a specific version
  hubbiott selfupdate --version 1.2.3`,
	Run: func(cmd *cobra.Command, args []string) {
		updater := update.NewUpdater()
		ctx, cancel := context.WithTimeout(context.Background(), selfupdateTimeout)
		defer cancel()

		currentVersion := Version

		// If checking only
		if selfupdateCheckOnly {
			checkUpdate(ctx, updater, currentVersion)
			return
		}

		// If specific version requested
		if selfupdateVersion != "" {
			updateToVersion(ctx, updater, selfupdateVersion)
			return
		}

		// Perform update
		performUpdate(ctx, updater, currentVersion)
	},
}

func init() {
	rootCmd.AddCommand(selfupdateCmd)

	selfupdateCmd.Flags().BoolVar(&selfupdateCheckOnly, "check", false, "Only check for updates, don't install")
	selfupdateCmd.Flags().StringVar(&selfupdateVersion, "version", "", "Update to a specific version")
	selfupdateCmd.Flags().DurationVar(&selfupdateTimeout, "timeout", 5*time.Minute, "Timeout for the update operation")
}

func checkUpdate(ctx context.Context, updater *update.Updater, currentVersion string) {
	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Println("Checking for updates...")

	release, err := updater.CheckForUpdate(ctx, currentVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	if release == nil {
		fmt.Println("Already up to date!")
		return
	}

	fmt.Printf("Update available: %s\n", release.Version)
	fmt.Printf("Published: %s\n", release.PublishedAt.Format("2006-01-02"))
	if release.ReleaseNotes != "" {
		fmt.Println("\nRelease notes:")
		fmt.Println(release.ReleaseNotes)
	}
}

func updateToVersion(ctx context.Context, updater *update.Updater, version string) {
	fmt.Printf("Updating to version %s...\n", version)

	release, err := updater.GetReleaseByVersion(ctx, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting release: %v\n", err)
		os.Exit(1)
	}

	// Download the new binary
	fmt.Println("Downloading...")
	archivePath, err := updater.DownloadBinary(ctx, release.DownloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(archivePath)

	// Verify checksum if available
	if release.Checksum != "" {
		fmt.Println("Verifying checksum...")
		if err := updater.VerifyChecksum(archivePath, release.Checksum); err != nil {
			fmt.Fprintf(os.Stderr, "Checksum verification failed: %v\n", err)
			os.Exit(1)
		}
	}

	// Extract the binary
	fmt.Println("Extracting...")
	newBinaryPath, err := updater.ExtractBinary(archivePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Extraction failed: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(newBinaryPath)

	// Replace the current binary
	fmt.Println("Installing...")
	if err := updater.ReplaceBinary(newBinaryPath); err != nil {
		fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully updated to version %s!\n", release.Version)
	fmt.Println("Please restart hubbiott to use the new version.")
}

func performUpdate(ctx context.Context, updater *update.Updater, currentVersion string) {
	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Println("Checking for updates...")

	release, err := updater.Update(ctx, currentVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
		os.Exit(1)
	}

	if release == nil {
		fmt.Println("Already up to date!")
		return
	}

	fmt.Printf("\nSuccessfully updated to version %s!\n", release.Version)
	fmt.Println("Please restart hubbiott to use the new version.")
}
