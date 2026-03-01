// Package install provides service installation functionality for Hubbiott.
package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nathabonfim59/hubbiott/internal/wizard"
)

// UserInstaller handles user-mode installation of Hubbiott.
type UserInstaller struct {
	config *wizard.Config
}

// NewUserInstaller creates a new UserInstaller with the given configuration.
func NewUserInstaller(config *wizard.Config) *UserInstaller {
	return &UserInstaller{
		config: config,
	}
}

// Install performs the user-mode installation.
// It installs the binary, creates config directory, and sets up systemd user service.
// On failure, it automatically rolls back any completed steps.
func (u *UserInstaller) Install() (*Result, error) {
	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Define installation paths
	binaryPath := filepath.Join(home, ".local", "bin", "hubbiott")
	configDir := filepath.Join(home, ".config", "hubbiott")
	configPath := filepath.Join(configDir, "config.yaml")
	serviceDir := filepath.Join(home, ".config", "systemd", "user")
	servicePath := filepath.Join(serviceDir, "hubbiott.service")
	workingDir := configDir

	// Initialize rollback handler
	rollback := NewRollback()

	// Helper function to handle errors with rollback
	handleError := func(err error) (*Result, error) {
		if err != nil {
			rollbackErr := rollback.Execute()
			if rollbackErr != nil {
				return nil, fmt.Errorf("%w (rollback also failed: %v)", err, rollbackErr)
			}
		}
		return nil, err
	}

	// Step 1: Install binary to ~/.local/bin/hubbiott
	rollback.AddStep("binary installation", RollbackFileRemoval(binaryPath))
	if err := u.installBinary(binaryPath); err != nil {
		return handleError(fmt.Errorf("failed to install binary: %w", err))
	}
	rollback.MarkCompleted()

	// Step 2: Create config directory
	rollback.AddStep("config directory creation", RollbackDirRemoval(configDir))
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return handleError(fmt.Errorf("failed to create config directory: %w", err))
	}
	rollback.MarkCompleted()

	// Step 3: Generate and write config file
	rollback.AddStep("config file creation", RollbackFileRemoval(configPath))
	if err := u.config.GenerateConfigFile(configPath); err != nil {
		return handleError(fmt.Errorf("failed to write config file: %w", err))
	}
	rollback.MarkCompleted()

	// Step 4: Create systemd user service directory
	rollback.AddStep("systemd user directory creation", RollbackDirRemoval(serviceDir))
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return handleError(fmt.Errorf("failed to create systemd user directory: %w", err))
	}
	rollback.MarkCompleted()

	// Step 5: Create systemd user service file
	rollback.AddStep("service file creation", RollbackFileRemoval(servicePath))
	if err := u.createServiceFile(servicePath, binaryPath, workingDir); err != nil {
		return handleError(fmt.Errorf("failed to create service file: %w", err))
	}
	rollback.MarkCompleted()

	// Step 6: Reload systemd daemon
	rollback.AddStep("systemd daemon reload", RollbackDaemonReload(true))
	if err := SystemctlUser("daemon-reload"); err != nil {
		return handleError(fmt.Errorf("failed to reload systemd daemon: %w", err))
	}
	rollback.MarkCompleted()

	// Step 7: Enable the service
	rollback.AddStep("service enablement", RollbackServiceDisable("hubbiott.service", true))
	if err := SystemctlUser("enable", "hubbiott.service"); err != nil {
		return handleError(fmt.Errorf("failed to enable service: %w", err))
	}
	rollback.MarkCompleted()

	// Step 8: Start the service
	rollback.AddStep("service start", RollbackServiceStop("hubbiott.service", true))
	if err := SystemctlUser("start", "hubbiott.service"); err != nil {
		return handleError(fmt.Errorf("failed to start service: %w", err))
	}
	rollback.MarkCompleted()

	return &Result{
		BinaryPath:  binaryPath,
		ConfigPath:  configPath,
		ServicePath: servicePath,
		ServiceName: "hubbiott.service",
	}, nil
}

// installBinary copies the current binary to the installation location.
func (u *UserInstaller) installBinary(destPath string) error {
	// Get current binary path
	currentBinary, err := GetCurrentBinaryPath()
	if err != nil {
		return err
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create binary directory: %w", err)
	}

	// Check if already installed
	if _, err := os.Stat(destPath); err == nil {
		// File exists, we'll overwrite it
		// First stop the service if running
		_ = SystemctlUser("stop", "hubbiott.service")
	}

	// Copy the binary
	if err := CopyFile(currentBinary, destPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Make sure it's executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	return nil
}

// createServiceFile creates the systemd user service file.
func (u *UserInstaller) createServiceFile(servicePath, binaryPath, workingDir string) error {
	// Get service template for user mode
	serviceContent, err := GetServiceTemplate(ModeUser, binaryPath, workingDir)
	if err != nil {
		return fmt.Errorf("failed to generate service template: %w", err)
	}

	// Write the service file
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	return nil
}

// Uninstall removes the user-mode installation.
func (u *UserInstaller) Uninstall() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	binaryPath := filepath.Join(home, ".local", "bin", "hubbiott")
	servicePath := filepath.Join(home, ".config", "systemd", "user", "hubbiott.service")

	// Stop and disable the service
	_ = SystemctlUser("stop", "hubbiott.service")
	_ = SystemctlUser("disable", "hubbiott.service")

	// Remove service file
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload systemd
	_ = SystemctlUser("daemon-reload")

	// Remove binary
	if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove binary: %w", err)
	}

	return nil
}
