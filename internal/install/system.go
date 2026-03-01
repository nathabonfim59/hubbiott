// Package install provides service installation functionality for Hubbiott.
package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nathabonfim59/hubbiott/internal/wizard"
)

// System paths for system-mode installation.
const (
	SystemBinaryPath  = "/usr/local/bin/hubbiott"
	SystemConfigDir   = "/etc/hubbiott"
	SystemServiceDir  = "/etc/systemd/system"
	SystemServiceUser = "hubbiott"
)

// SystemInstaller handles system-mode installation of Hubbiott.
type SystemInstaller struct {
	config *wizard.Config
}

// NewSystemInstaller creates a new SystemInstaller with the given configuration.
func NewSystemInstaller(config *wizard.Config) *SystemInstaller {
	return &SystemInstaller{
		config: config,
	}
}

// Install performs the system-mode installation.
// It requires root privileges and installs the binary system-wide,
// creates the config directory, and sets up a systemd system service.
func (s *SystemInstaller) Install() (*Result, error) {
	// Check for root privileges
	if err := s.checkRootPrivileges(); err != nil {
		return nil, err
	}

	// Define installation paths
	binaryPath := SystemBinaryPath
	configDir := SystemConfigDir
	configPath := filepath.Join(configDir, "config.yaml")
	servicePath := filepath.Join(SystemServiceDir, "hubbiott.service")
	workingDir := configDir

	// Step 1: Create hubbiott system user if not exists
	if err := s.ensureSystemUser(); err != nil {
		return nil, fmt.Errorf("failed to create system user: %w", err)
	}

	// Step 2: Install binary to /usr/local/bin/hubbiott
	if err := s.installBinary(binaryPath); err != nil {
		return nil, fmt.Errorf("failed to install binary: %w", err)
	}

	// Step 3: Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Step 4: Set config directory ownership to hubbiott user
	if err := s.setOwnership(configDir); err != nil {
		return nil, fmt.Errorf("failed to set config directory ownership: %w", err)
	}

	// Step 5: Generate and write config file
	if err := s.config.GenerateConfigFile(configPath); err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	// Step 6: Set config file ownership and permissions
	if err := s.setOwnership(configPath); err != nil {
		return nil, fmt.Errorf("failed to set config file ownership: %w", err)
	}
	// Set restrictive permissions on config file (contains secrets)
	if err := os.Chmod(configPath, 0600); err != nil {
		return nil, fmt.Errorf("failed to set config file permissions: %w", err)
	}

	// Step 7: Create systemd system service file
	if err := s.createServiceFile(servicePath, binaryPath, workingDir); err != nil {
		return nil, fmt.Errorf("failed to create service file: %w", err)
	}

	// Step 8: Reload systemd daemon
	if err := Systemctl("daemon-reload"); err != nil {
		return nil, fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Step 9: Enable the service
	if err := Systemctl("enable", "hubbiott.service"); err != nil {
		return nil, fmt.Errorf("failed to enable service: %w", err)
	}

	// Step 10: Start the service
	if err := Systemctl("start", "hubbiott.service"); err != nil {
		return nil, fmt.Errorf("failed to start service: %w", err)
	}

	return &Result{
		BinaryPath:  binaryPath,
		ConfigPath:  configPath,
		ServicePath: servicePath,
		ServiceName: "hubbiott.service",
	}, nil
}

// checkRootPrivileges verifies that the installer is running with root privileges.
func (s *SystemInstaller) checkRootPrivileges() error {
	// Check if running as root (uid 0)
	if os.Geteuid() != 0 {
		return fmt.Errorf("system installation requires root privileges. Please run with sudo")
	}
	return nil
}

// ensureSystemUser creates the hubbiott system user if it doesn't exist.
func (s *SystemInstaller) ensureSystemUser() error {
	// Check if user already exists
	if _, err := exec.LookPath("id"); err == nil {
		cmd := exec.Command("id", "-u", SystemServiceUser)
		if err := cmd.Run(); err == nil {
			// User already exists
			return nil
		}
	}

	// Try to create the system user
	// Use useradd if available, fall back to adduser
	if _, err := exec.LookPath("useradd"); err == nil {
		cmd := exec.Command("useradd",
			"--system",                     // Create as system user
			"--no-create-home",             // Don't create home directory
			"--shell", "/usr/sbin/nologin", // No login shell
			"--comment", "Hubbiott Service",
			SystemServiceUser,
		)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create system user: %w\n%s", err, output)
		}
		return nil
	}

	if _, err := exec.LookPath("adduser"); err == nil {
		cmd := exec.Command("adduser",
			"--system",
			"--no-create-home",
			"--disabled-password",
			"--gecos", "Hubbiott Service",
			SystemServiceUser,
		)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create system user: %w\n%s", err, output)
		}
		return nil
	}

	// Neither useradd nor adduser available - warn but continue
	// The service file will still work with the User= directive
	// if the user was created manually or through other means
	fmt.Printf("Warning: Could not create system user '%s'. ", SystemServiceUser)
	fmt.Printf("Please ensure the user exists or create it manually:\n")
	fmt.Printf("  sudo useradd --system --no-create-home %s\n", SystemServiceUser)

	return nil
}

// installBinary copies the current binary to the system installation location.
func (s *SystemInstaller) installBinary(destPath string) error {
	// Get current binary path
	currentBinary, err := GetCurrentBinaryPath()
	if err != nil {
		return err
	}

	// Check if already installed
	if _, err := os.Stat(destPath); err == nil {
		// File exists, stop the service if running
		_ = Systemctl("stop", "hubbiott.service")
	}

	// Copy the binary
	if err := CopyFile(currentBinary, destPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Make sure it's executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Set ownership to root
	if err := s.setOwnership(destPath); err != nil {
		return fmt.Errorf("failed to set binary ownership: %w", err)
	}

	return nil
}

// createServiceFile creates the systemd system service file.
func (s *SystemInstaller) createServiceFile(servicePath, binaryPath, workingDir string) error {
	// Get service template for system mode
	serviceContent, err := GetServiceTemplate(ModeSystem, binaryPath, workingDir)
	if err != nil {
		return fmt.Errorf("failed to generate service template: %w", err)
	}

	// Ensure the systemd directory exists
	if err := os.MkdirAll(filepath.Dir(servicePath), 0755); err != nil {
		return fmt.Errorf("failed to create systemd directory: %w", err)
	}

	// Write the service file
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	return nil
}

// setOwnership sets the ownership of a file or directory to root:hubbiott.
// For the binary, it sets root:root. For config files, root:hubbiott.
func (s *SystemInstaller) setOwnership(path string) error {
	// Use chown to set ownership
	cmd := exec.Command("chown", "root:root", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set ownership: %w\n%s", err, output)
	}

	// For config directory and files, also set group to hubbiott for read access
	if strings.HasPrefix(path, SystemConfigDir) {
		cmd := exec.Command("chown", "root:"+SystemServiceUser, path)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set group ownership: %w\n%s", err, output)
		}
	}

	return nil
}

// Uninstall removes the system-mode installation.
func (s *SystemInstaller) Uninstall() error {
	// Check for root privileges
	if err := s.checkRootPrivileges(); err != nil {
		return err
	}

	// Stop and disable the service
	_ = Systemctl("stop", "hubbiott.service")
	_ = Systemctl("disable", "hubbiott.service")

	// Remove service file
	servicePath := filepath.Join(SystemServiceDir, "hubbiott.service")
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload systemd
	_ = Systemctl("daemon-reload")

	// Remove binary
	if err := os.Remove(SystemBinaryPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove binary: %w", err)
	}

	// Remove config directory (with user confirmation in a real implementation)
	// For now, we'll leave the config directory in place to preserve user data
	fmt.Printf("Note: Configuration directory %s has been preserved.\n", SystemConfigDir)
	fmt.Printf("To remove it manually: sudo rm -rf %s\n", SystemConfigDir)

	return nil
}
