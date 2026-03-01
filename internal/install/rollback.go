// Package install provides service installation functionality for Hubbiott.
package install

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

// RollbackAction is a function that performs a rollback for a specific step.
type RollbackAction func() error

// RollbackStep represents a single installation step with its rollback action.
type RollbackStep struct {
	Name        string         // Human-readable name of the step
	Action      RollbackAction // Function to execute for rollback
	Completed   bool           // Whether this step was completed successfully
	SkipOnError bool           // Whether to skip this rollback if an error occurs
}

// Rollback manages installation steps and their rollback actions.
// It tracks completed steps and can execute rollbacks in reverse order on failure.
type Rollback struct {
	mu    sync.Mutex
	steps []RollbackStep
}

// NewRollback creates a new Rollback instance.
func NewRollback() *Rollback {
	return &Rollback{
		steps: make([]RollbackStep, 0),
	}
}

// AddStep registers a new installation step with its rollback action.
// Steps should be added in the order they are executed.
// The rollback action will be executed in reverse order during rollback.
func (r *Rollback) AddStep(name string, rollbackAction RollbackAction) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.steps = append(r.steps, RollbackStep{
		Name:        name,
		Action:      rollbackAction,
		Completed:   false,
		SkipOnError: false,
	})
}

// MarkCompleted marks the last added step as completed.
// This should be called after a step successfully completes.
func (r *Rollback) MarkCompleted() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.steps) > 0 {
		r.steps[len(r.steps)-1].Completed = true
	}
}

// Execute runs all rollback actions in reverse order for completed steps.
// It continues even if individual rollback actions fail, collecting all errors.
// Returns a multi-error if any rollbacks failed, or nil if all succeeded.
func (r *Rollback) Execute() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errors []error

	// Iterate in reverse order
	for i := len(r.steps) - 1; i >= 0; i-- {
		step := r.steps[i]

		// Only rollback completed steps
		if !step.Completed {
			continue
		}

		// Skip if marked to skip on error
		if step.SkipOnError {
			continue
		}

		if step.Action != nil {
			if err := step.Action(); err != nil {
				errors = append(errors, fmt.Errorf("rollback failed for step %q: %w", step.Name, err))
			}
		}
	}

	if len(errors) > 0 {
		return &RollbackError{Errors: errors}
	}

	return nil
}

// RollbackError holds multiple errors from rollback operations.
type RollbackError struct {
	Errors []error
}

// Error implements the error interface.
func (e *RollbackError) Error() string {
	if len(e.Errors) == 0 {
		return "rollback failed"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("rollback failed: %v", e.Errors[0])
	}
	return fmt.Sprintf("rollback failed with %d errors: %v", len(e.Errors), e.Errors[0])
}

// Unwrap returns the first error for compatibility with errors.Is/As.
func (e *RollbackError) Unwrap() error {
	if len(e.Errors) > 0 {
		return e.Errors[0]
	}
	return nil
}

// Common rollback actions

// RollbackFileRemoval creates a rollback action that removes a file.
func RollbackFileRemoval(path string) RollbackAction {
	return func() error {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove file %s: %w", path, err)
		}
		return nil
	}
}

// RollbackDirRemoval creates a rollback action that removes a directory.
func RollbackDirRemoval(path string) RollbackAction {
	return func() error {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			// If directory is not empty, try to remove contents first
			if os.IsExist(err) {
				entries, readErr := os.ReadDir(path)
				if readErr == nil && len(entries) == 0 {
					return os.Remove(path)
				}
			}
			return fmt.Errorf("failed to remove directory %s: %w", path, err)
		}
		return nil
	}
}

// RollbackServiceStop creates a rollback action that stops a systemd service.
func RollbackServiceStop(serviceName string, userMode bool) RollbackAction {
	return func() error {
		var err error
		if userMode {
			err = SystemctlUser("stop", serviceName)
		} else {
			err = Systemctl("stop", serviceName)
		}
		if err != nil {
			return fmt.Errorf("failed to stop service %s: %w", serviceName, err)
		}
		return nil
	}
}

// RollbackServiceDisable creates a rollback action that disables a systemd service.
func RollbackServiceDisable(serviceName string, userMode bool) RollbackAction {
	return func() error {
		var err error
		if userMode {
			err = SystemctlUser("disable", serviceName)
		} else {
			err = Systemctl("disable", serviceName)
		}
		if err != nil {
			return fmt.Errorf("failed to disable service %s: %w", serviceName, err)
		}
		return nil
	}
}

// RollbackDaemonReload creates a rollback action that reloads the systemd daemon.
func RollbackDaemonReload(userMode bool) RollbackAction {
	return func() error {
		var err error
		if userMode {
			err = SystemctlUser("daemon-reload")
		} else {
			err = Systemctl("daemon-reload")
		}
		if err != nil {
			return fmt.Errorf("failed to reload systemd daemon: %w", err)
		}
		return nil
	}
}

// RollbackUserRemoval creates a rollback action that removes a system user.
func RollbackUserRemoval(username string) RollbackAction {
	return func() error {
		// Try userdel first, then deluser
		if _, err := exec.LookPath("userdel"); err == nil {
			cmd := exec.Command("userdel", username)
			if output, err := cmd.CombinedOutput(); err != nil {
				// Ignore "does not exist" errors
				if !isUserNotExistError(string(output)) {
					return fmt.Errorf("failed to remove user %s: %w\n%s", username, err, output)
				}
			}
			return nil
		}

		if _, err := exec.LookPath("deluser"); err == nil {
			cmd := exec.Command("deluser", username)
			if output, err := cmd.CombinedOutput(); err != nil {
				// Ignore "does not exist" errors
				if !isUserNotExistError(string(output)) {
					return fmt.Errorf("failed to remove user %s: %w\n%s", username, err, output)
				}
			}
			return nil
		}

		// Neither command available, nothing to do
		return nil
	}
}

// isUserNotExistError checks if the output indicates the user doesn't exist.
func isUserNotExistError(output string) bool {
	return contains(output, "does not exist") ||
		contains(output, "not found") ||
		contains(output, "no such user")
}

// contains checks if s contains substr (case-insensitive helper).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			subc := substr[j]
			// Simple lowercase comparison
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if subc >= 'A' && subc <= 'Z' {
				subc += 32
			}
			if sc != subc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// NoopRollback creates a rollback action that does nothing.
// Useful for steps that don't require explicit rollback.
func NoopRollback() RollbackAction {
	return func() error {
		return nil
	}
}
