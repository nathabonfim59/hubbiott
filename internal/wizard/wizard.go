// Package wizard provides an interactive configuration wizard for Hubbiott.
// It uses charmbracelet/huh for a beautiful terminal-based form.
package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/nathabonfim59/hubbiott/embed"
)

// Config holds the configuration values collected from the wizard.
type Config struct {
	// Installation mode
	InstallMode string // "user" or "system"

	// Discord settings
	DiscordToken   string
	DiscordGuildID string

	// Database settings
	DatabaseURL       string
	DatabaseAuthToken string

	// API settings
	APIKey string

	// GitHub settings
	GitHubToken         string
	GitHubWebhookSecret string
}

// Wizard manages the interactive configuration process.
type Wizard struct {
	config Config
}

// New creates a new Wizard instance.
func New() *Wizard {
	return &Wizard{}
}

// Run executes the interactive wizard and returns the collected configuration.
func (w *Wizard) Run() (*Config, error) {
	// Installation mode selection
	modeOptions := []huh.Option[string]{
		{Key: "User installation (runs as current user, user-level service)", Value: "user"},
		{Key: "System installation (runs as root/daemon, system-wide service)", Value: "system"},
	}

	// Database type selection
	dbOptions := []huh.Option[string]{
		{Key: "Local SQLite (file:./hubbiott.db)", Value: "sqlite"},
		{Key: "Turso Cloud (libsql://...)", Value: "turso"},
	}

	var dbType string

	// Build the form
	form := huh.NewForm(
		// Installation mode group
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Installation Mode").
				Description("Choose how Hubbiott will be installed on your system").
				Options(modeOptions...).
				Value(&w.config.InstallMode),
		),

		// Discord configuration group
		huh.NewGroup(
			huh.NewNote().
				Title("Discord Bot Configuration").
				Description("Configure your Discord bot credentials.\nYou can obtain these from the Discord Developer Portal."),
			huh.NewInput().
				Title("Discord Bot Token").
				Description("The bot token from Discord Developer Portal").
				Placeholder("OTk5OTk5OTk5OTk5OTk5OTk5.Gxxxxxxxxx.xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx").
				Value(&w.config.DiscordToken).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("discord token is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Discord Guild ID").
				Description("The Discord server ID where the bot will operate").
				Placeholder("1234567890123456789").
				Value(&w.config.DiscordGuildID).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("guild ID is required")
					}
					return nil
				}),
		),

		// Database configuration group
		huh.NewGroup(
			huh.NewNote().
				Title("Database Configuration").
				Description("Configure the database connection for storing data."),
			huh.NewSelect[string]().
				Title("Database Type").
				Description("Choose your database backend").
				Options(dbOptions...).
				Value(&dbType),
			huh.NewInput().
				Title("Database URL").
				Description("For SQLite: file:./hubbiott.db | For Turso: libsql://your-db.turso.io").
				Placeholder("file:./hubbiott.db").
				Value(&w.config.DatabaseURL).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("database URL is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Database Auth Token").
				Description("Required for Turso. Leave empty for local SQLite.").
				Placeholder("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...").
				Value(&w.config.DatabaseAuthToken),
		),

		// API configuration group
		huh.NewGroup(
			huh.NewNote().
				Title("API Configuration").
				Description("Configure the API security settings."),
			huh.NewInput().
				Title("API Key").
				Description("Secret key for API authentication (generate a secure random string)").
				Placeholder("your-secure-api-key-here").
				Value(&w.config.APIKey).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("API key is required")
					}
					if len(s) < 16 {
						return fmt.Errorf("API key should be at least 16 characters for security")
					}
					return nil
				}),
		),

		// GitHub configuration group
		huh.NewGroup(
			huh.NewNote().
				Title("GitHub Integration").
				Description("Configure GitHub integration for webhooks and automation."),
			huh.NewInput().
				Title("GitHub Personal Access Token").
				Description("Optional. Required for GitHub API operations.").
				Placeholder("ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx").
				Value(&w.config.GitHubToken),
			huh.NewInput().
				Title("GitHub Webhook Secret").
				Description("Secret for verifying webhook payloads (recommended)").
				Placeholder("your-webhook-secret").
				Value(&w.config.GitHubWebhookSecret).
				Validate(func(s string) error {
					// Webhook secret is optional but recommended
					return nil
				}),
		),
	).WithTheme(huh.ThemeCatppuccin())

	// Run the form
	err := form.Run()
	if err != nil {
		return nil, fmt.Errorf("wizard form failed: %w", err)
	}

	// Set default database URL if not provided based on db type
	if w.config.DatabaseURL == "" && dbType == "sqlite" {
		w.config.DatabaseURL = "file:./hubbiott.db"
	}

	return &w.config, nil
}

// GenerateConfigFile generates a configuration file from the template.
func (c *Config) GenerateConfigFile(destPath string) error {
	// Create the template data
	tmplData := map[string]string{
		"DiscordToken":        c.DiscordToken,
		"DiscordGuildID":      c.DiscordGuildID,
		"DatabaseURL":         c.DatabaseURL,
		"DatabaseAuthToken":   c.DatabaseAuthToken,
		"APIKey":              c.APIKey,
		"GitHubToken":         c.GitHubToken,
		"GitHubWebhookSecret": c.GitHubWebhookSecret,
	}

	// Get the template
	tmpl := embed.ConfigTemplate

	// Simple template replacement
	result := tmpl
	for key, value := range tmplData {
		result = strings.ReplaceAll(result, "{{."+key+"}}", value)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write the config file
	if err := os.WriteFile(destPath, []byte(result), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the default config file path based on install mode.
func (c *Config) GetConfigPath() (string, error) {
	switch c.InstallMode {
	case "system":
		return "/etc/hubbiott/config.yaml", nil
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		return filepath.Join(home, ".config", "hubbiott", "config.yaml"), nil
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		return filepath.Join(home, ".config", "hubbiott", "config.yaml"), nil
	}
}

// Summary returns a human-readable summary of the configuration.
func (c *Config) Summary() string {
	var sb strings.Builder

	sb.WriteString("Configuration Summary:\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	sb.WriteString(fmt.Sprintf("Installation Mode: %s\n", c.InstallMode))

	sb.WriteString("\nDiscord:\n")
	sb.WriteString(fmt.Sprintf("  Token: %s...%s\n", c.DiscordToken[:10], c.DiscordToken[len(c.DiscordToken)-4:]))
	sb.WriteString(fmt.Sprintf("  Guild ID: %s\n", c.DiscordGuildID))

	sb.WriteString("\nDatabase:\n")
	sb.WriteString(fmt.Sprintf("  URL: %s\n", c.DatabaseURL))
	if c.DatabaseAuthToken != "" {
		sb.WriteString("  Auth Token: (configured)\n")
	} else {
		sb.WriteString("  Auth Token: (not set)\n")
	}

	sb.WriteString("\nAPI:\n")
	sb.WriteString(fmt.Sprintf("  Key: %s...%s\n", c.APIKey[:4], c.APIKey[len(c.APIKey)-4:]))

	sb.WriteString("\nGitHub:\n")
	if c.GitHubToken != "" {
		sb.WriteString("  Token: (configured)\n")
	} else {
		sb.WriteString("  Token: (not set)\n")
	}
	if c.GitHubWebhookSecret != "" {
		sb.WriteString("  Webhook Secret: (configured)\n")
	} else {
		sb.WriteString("  Webhook Secret: (not set)\n")
	}

	return sb.String()
}
