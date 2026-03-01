// Package install provides service installation functionality for Hubbiott.
package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nathabonfim59/hubbiott/embed"
)

// InstallMode represents the installation mode.
type InstallMode string

const (
	// ModeUser installs Hubbiott for the current user only.
	ModeUser InstallMode = "user"
	// ModeSystem installs Hubbiott system-wide (requires root).
	ModeSystem InstallMode = "system"
)

// Paths holds the installation paths for Hubbiott.
type Paths struct {
	// BinaryPath is where the hubbiott binary will be installed.
	BinaryPath string
	// ConfigDir is the directory for configuration files.
	ConfigDir string
	// ConfigFile is the path to the main config file.
	ConfigFile string
	// ServiceFile is the path to the systemd service file.
	ServiceFile string
	// WorkingDir is the working directory for the service.
	WorkingDir string
}

// Installer provides common installation functionality.
type Installer struct {
	Paths Paths
	Mode  InstallMode
}

// Result holds the result of an installation.
type Result struct {
	BinaryPath  string
	ConfigPath  string
	ServicePath string
	ServiceName string
}

// ErrNotSupported is returned when an installation mode is not supported.
var ErrNotSupported = fmt.Errorf("installation mode not supported on this system")

// ErrAlreadyInstalled is returned when Hubbiott is already installed.
var ErrAlreadyInstalled = fmt.Errorf("hubbiott is already installed")

// GetServiceTemplate returns the appropriate systemd service template for the install mode.
func GetServiceTemplate(mode InstallMode, binaryPath, workingDir string) (string, error) {
	tmpl := embed.SystemdServiceTemplate

	// Build template data based on mode
	var tmplData map[string]string

	switch mode {
	case ModeUser:
		// User service doesn't need User/Group fields
		// We'll modify the template to handle this
		tmplData = map[string]string{
			"ServiceUser":  "", // Will be handled specially
			"ServiceGroup": "",
			"WorkingDir":   workingDir,
			"BinaryPath":   binaryPath,
		}
	case ModeSystem:
		// System service needs user/group
		tmplData = map[string]string{
			"ServiceUser":  "hubbiott",
			"ServiceGroup": "hubbiott",
			"WorkingDir":   workingDir,
			"BinaryPath":   binaryPath,
		}
	default:
		return "", ErrNotSupported
	}

	// Replace template variables
	result := tmpl
	for key, value := range tmplData {
		result = strings.ReplaceAll(result, "{{."+key+"}}", value)
	}

	// For user mode, we need to modify the service file
	// Remove User= and Group= lines, and change WantedBy to default.target
	if mode == ModeUser {
		lines := strings.Split(result, "\n")
		var filteredLines []string
		for _, line := range lines {
			// Skip User and Group directives for user services
			if strings.HasPrefix(line, "User=") || strings.HasPrefix(line, "Group=") {
				continue
			}
			// Change WantedBy for user services
			if strings.HasPrefix(line, "WantedBy=") {
				line = "WantedBy=default.target"
			}
			filteredLines = append(filteredLines, line)
		}
		result = strings.Join(filteredLines, "\n")
	}

	return result, nil
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Get source file info for permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	// Preserve permissions
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// SystemctlUser runs a systemctl command with --user flag.
func SystemctlUser(args ...string) error {
	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl --user %s failed: %w\n%s", strings.Join(args, " "), err, output)
	}
	return nil
}

// Systemctl runs a systemctl command (system-level).
func Systemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s failed: %w\n%s", strings.Join(args, " "), err, output)
	}
	return nil
}

// GetCurrentBinaryPath returns the path to the current executable.
func GetCurrentBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get current binary path: %w", err)
	}
	// Resolve symlinks
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("failed to resolve binary path: %w", err)
	}
	return exe, nil
}
