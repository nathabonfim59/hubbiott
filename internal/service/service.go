// Package service provides systemd service management for Hubbiott.
package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// ServiceName is the default systemd service name.
	ServiceName = "hubbiott.service"
)

// Mode represents the service mode (user or system).
type Mode string

const (
	// ModeUser manages user-level systemd services.
	ModeUser Mode = "user"
	// ModeSystem manages system-level systemd services (requires root).
	ModeSystem Mode = "system"
)

// Status represents the current state of a service.
type Status struct {
	// Name is the service name.
	Name string
	// Active indicates if the service is currently running.
	Active bool
	// State is the current systemd state (e.g., "running", "stopped", "failed").
	State string
	// SubState is the detailed systemd sub-state.
	SubState string
	// Enabled indicates if the service is enabled for auto-start.
	Enabled bool
	// PID is the process ID (0 if not running).
	PID int
	// Uptime is the service uptime in seconds (0 if not running).
	Uptime int
}

// LogEntry represents a journal log entry.
type LogEntry struct {
	Timestamp string
	Message   string
	Priority  string
}

// Manager manages systemd services for Hubbiott.
type Manager struct {
	// Mode determines whether to manage user or system services.
	Mode Mode
	// ServiceName is the name of the systemd service (default: hubbiott.service).
	ServiceName string
}

// NewManager creates a new service manager.
// If mode is empty, it auto-detects based on installation.
func NewManager(mode Mode) *Manager {
	m := &Manager{
		Mode:        mode,
		ServiceName: ServiceName,
	}
	// Auto-detect mode if not specified
	if m.Mode == "" {
		m.Mode = detectMode()
	}
	return m
}

// detectMode determines if Hubbiott is installed as user or system service.
func detectMode() Mode {
	home, err := os.UserHomeDir()
	if err == nil {
		userServicePath := filepath.Join(home, ".config", "systemd", "user", ServiceName)
		if _, err := os.Stat(userServicePath); err == nil {
			return ModeUser
		}
	}

	systemServicePath := filepath.Join("/etc/systemd/system", ServiceName)
	if _, err := os.Stat(systemServicePath); err == nil {
		return ModeSystem
	}

	// Default to user mode
	return ModeUser
}

// systemctl runs a systemctl command with appropriate flags.
func (m *Manager) systemctl(args ...string) error {
	var cmd *exec.Cmd
	if m.Mode == ModeUser {
		cmd = exec.Command("systemctl", append([]string{"--user"}, args...)...)
	} else {
		cmd = exec.Command("systemctl", args...)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s failed: %w\n%s", strings.Join(args, " "), err, output)
	}
	return nil
}

// systemctlOutput runs a systemctl command and returns its output.
func (m *Manager) systemctlOutput(args ...string) (string, error) {
	var cmd *exec.Cmd
	if m.Mode == ModeUser {
		cmd = exec.Command("systemctl", append([]string{"--user"}, args...)...)
	} else {
		cmd = exec.Command("systemctl", args...)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("systemctl %s failed: %w", strings.Join(args, " "), err)
	}
	return string(output), nil
}

// Start starts the service.
func (m *Manager) Start() error {
	return m.systemctl("start", m.ServiceName)
}

// Stop stops the service.
func (m *Manager) Stop() error {
	return m.systemctl("stop", m.ServiceName)
}

// Restart restarts the service.
func (m *Manager) Restart() error {
	return m.systemctl("restart", m.ServiceName)
}

// Status returns the current status of the service.
func (m *Manager) Status() (*Status, error) {
	status := &Status{
		Name: m.ServiceName,
	}

	// Get active state
	output, err := m.systemctlOutput("show", m.ServiceName,
		"--property=ActiveState,SubState,MainPID,ActiveEnterTimestampMonotonic")
	if err != nil {
		// Service might not exist or not loaded
		status.Active = false
		status.State = "unknown"
		status.SubState = "unknown"
		return status, nil
	}

	// Parse the output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		switch key {
		case "ActiveState":
			status.State = value
			status.Active = value == "active"
		case "SubState":
			status.SubState = value
		case "MainPID":
			status.PID, _ = strconv.Atoi(value)
		}
	}

	// Check if enabled
	enabledOutput, err := m.systemctlOutput("is-enabled", m.ServiceName)
	if err == nil {
		status.Enabled = strings.TrimSpace(enabledOutput) == "enabled"
	} else {
		status.Enabled = false
	}

	return status, nil
}

// Enable enables the service to start on boot.
func (m *Manager) Enable() error {
	return m.systemctl("enable", m.ServiceName)
}

// Disable disables the service from starting on boot.
func (m *Manager) Disable() error {
	return m.systemctl("disable", m.ServiceName)
}

// IsRunning returns true if the service is currently active.
func (m *Manager) IsRunning() bool {
	status, err := m.Status()
	if err != nil {
		return false
	}
	return status.Active
}

// Logs returns recent journal logs for the service.
// If lines is 0, it defaults to 50.
func (m *Manager) Logs(lines int) ([]LogEntry, error) {
	if lines == 0 {
		lines = 50
	}

	var cmd *exec.Cmd
	if m.Mode == ModeUser {
		cmd = exec.Command("journalctl", "--user", "-u", m.ServiceName,
			"-n", strconv.Itoa(lines), "--no-pager",
			"-o", "json")
	} else {
		cmd = exec.Command("journalctl", "-u", m.ServiceName,
			"-n", strconv.Itoa(lines), "--no-pager",
			"-o", "json")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("journalctl failed: %w", err)
	}

	// Parse JSON output (simplified - each line is a JSON object)
	var entries []LogEntry
	lines2 := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines2 {
		if line == "" {
			continue
		}
		entry := parseJournalEntry(line)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries, nil
}

// LogsRaw returns raw journal log output as a string.
// This is a simpler alternative to Logs() for display purposes.
func (m *Manager) LogsRaw(lines int) (string, error) {
	if lines == 0 {
		lines = 50
	}

	var cmd *exec.Cmd
	if m.Mode == ModeUser {
		cmd = exec.Command("journalctl", "--user", "-u", m.ServiceName,
			"-n", strconv.Itoa(lines), "--no-pager")
	} else {
		cmd = exec.Command("journalctl", "-u", m.ServiceName,
			"-n", strconv.Itoa(lines), "--no-pager")
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("journalctl failed: %w", err)
	}

	return string(output), nil
}

// parseJournalEntry parses a JSON journal entry line.
func parseJournalEntry(line string) *LogEntry {
	// Simple JSON parsing for common fields
	// Format: {"__REALTIME_TIMESTAMP":"...", "MESSAGE":"...", "PRIORITY":"..."}
	entry := &LogEntry{}

	// Extract MESSAGE
	if idx := strings.Index(line, `"MESSAGE":`); idx != -1 {
		start := strings.Index(line[idx:], `"`) + idx + 1
		if start > idx {
			rest := line[start:]
			end := strings.Index(rest, `"`)
			if end != -1 {
				entry.Message = rest[:end]
				// Unescape basic escape sequences
				entry.Message = strings.ReplaceAll(entry.Message, "\\n", "\n")
				entry.Message = strings.ReplaceAll(entry.Message, "\\t", "\t")
			}
		}
	}

	// Extract PRIORITY
	if idx := strings.Index(line, `"PRIORITY":`); idx != -1 {
		start := strings.Index(line[idx:], `"`) + idx + 1
		if start > idx {
			rest := line[start:]
			end := strings.Index(rest, `"`)
			if end != -1 {
				entry.Priority = rest[:end]
			}
		}
	}

	// Extract timestamp
	if idx := strings.Index(line, `"__REALTIME_TIMESTAMP":`); idx != -1 {
		start := strings.Index(line[idx:], `"`) + idx + 1
		if start > idx {
			rest := line[start:]
			end := strings.Index(rest, `"`)
			if end != -1 {
				entry.Timestamp = rest[:end]
			}
		}
	}

	return entry
}

// FollowLogs follows the journal logs in real-time.
// This blocks until the context is cancelled.
func (m *Manager) FollowLogs() error {
	var cmd *exec.Cmd
	if m.Mode == ModeUser {
		cmd = exec.Command("journalctl", "--user", "-u", m.ServiceName, "-f")
	} else {
		cmd = exec.Command("journalctl", "-u", m.ServiceName, "-f")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// DaemonReload reloads the systemd daemon to pick up service file changes.
func (m *Manager) DaemonReload() error {
	return m.systemctl("daemon-reload")
}

// ResetFailed resets the failed state of the service.
func (m *Manager) ResetFailed() error {
	return m.systemctl("reset-failed", m.ServiceName)
}

// Exists checks if the service file exists.
func (m *Manager) Exists() bool {
	var servicePath string
	if m.Mode == ModeUser {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		servicePath = filepath.Join(home, ".config", "systemd", "user", m.ServiceName)
	} else {
		servicePath = filepath.Join("/etc/systemd/system", m.ServiceName)
	}

	_, err := os.Stat(servicePath)
	return err == nil
}
