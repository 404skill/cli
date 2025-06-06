package config

import (
	"context"
	"fmt"
	"time"

	"404skill-cli/auth"
	"404skill-cli/supabase"
)

// ConfigManager handles configuration operations
type ConfigManager struct{}

// NewConfigManager creates a new config manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{}
}

// HasCredentials checks if the config has stored credentials
func (c *ConfigManager) HasCredentials() bool {
	cfg, err := readConfig()
	if err != nil {
		return false
	}
	return cfg.Username != "" && cfg.Password != ""
}

// GetDownloadedProjects returns a map of downloaded project IDs
func (c *ConfigManager) GetDownloadedProjects() map[string]bool {
	cfg, err := readConfig()
	if err != nil {
		return make(map[string]bool)
	}
	if cfg.DownloadedProjects == nil {
		return make(map[string]bool)
	}
	return cfg.DownloadedProjects
}

// IsProjectDownloaded checks if a project has been downloaded
func (c *ConfigManager) IsProjectDownloaded(projectID string) bool {
	cfg, err := readConfig()
	if err != nil {
		return false
	}
	return cfg.DownloadedProjects != nil && cfg.DownloadedProjects[projectID]
}

// UpdateDownloadedProject marks a project as downloaded
func (c *ConfigManager) UpdateDownloadedProject(projectID string) error {
	cfg, err := readConfig()
	if err != nil {
		return err
	}
	if cfg.DownloadedProjects == nil {
		cfg.DownloadedProjects = make(map[string]bool)
	}
	cfg.DownloadedProjects[projectID] = true
	return writeConfig(cfg)
}

// UpdateAuthConfig updates authentication-related configuration while preserving other settings
func (c *ConfigManager) UpdateAuthConfig(username, password, accessToken string) error {
	// Read existing config to preserve DownloadedProjects and other data
	cfg, err := readConfig()
	if err != nil {
		// If config doesn't exist, create new one
		cfg = Config{}
	}

	// Update only the auth-related fields
	cfg.Username = username
	cfg.Password = password
	cfg.AccessToken = accessToken
	cfg.LastUpdated = time.Now()

	// Ensure DownloadedProjects map exists
	if cfg.DownloadedProjects == nil {
		cfg.DownloadedProjects = make(map[string]bool)
	}

	return writeConfig(cfg)
}

// GetToken gets a valid access token, refreshing it if necessary
// This implements the TokenProvider interface functionality
func (c *ConfigManager) GetToken() (string, error) {
	config, err := readConfig()
	if err != nil {
		return "", err
	}

	if isTokenExpired(config.LastUpdated) || config.AccessToken == "" {
		client, err := supabase.NewSupabaseClient()
		if err != nil {
			return "", fmt.Errorf("failed to create supabase client: %w", err)
		}

		authProvider := auth.NewSupabaseAuth(client)
		accessToken, err := authProvider.SignIn(context.Background(), config.Username, config.Password)
		if err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}

		// Use our own UpdateAuthConfig method to save the new token
		if err := c.UpdateAuthConfig(config.Username, config.Password, accessToken); err != nil {
			return "", fmt.Errorf("failed to save new token: %w", err)
		}

		return accessToken, nil
	}

	return config.AccessToken, nil
}
