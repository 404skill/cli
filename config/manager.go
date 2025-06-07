package config

import (
	"fmt"
	"time"
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
func (c *ConfigManager) GetToken() (string, error) {
	config, err := readConfig()
	if err != nil {
		return "", err
	}

	if isTokenExpired(config.LastUpdated) || config.AccessToken == "" {
		return "", fmt.Errorf("token expired - please log in again")
	}

	return config.AccessToken, nil
}
