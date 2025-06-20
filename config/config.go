package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("Unable to determine user home directory")
	}

	err = os.MkdirAll(fmt.Sprintf("%s/.404skill", homeDir), os.ModePerm)
	if err != nil {
		panic("Unable to create .404skill directory")
	}

	ConfigFilePath = fmt.Sprintf("%s/.404skill/config.yml", homeDir)
}

var ConfigFilePath string

// Config represents the application configuration
type Config struct {
	Username           string          `yaml:"username"`
	Password           string          `yaml:"password"`
	AccessToken        string          `yaml:"access_token"`
	LastUpdated        time.Time       `yaml:"last_updated"`
	DownloadedProjects map[string]bool `yaml:"downloaded_projects"`
}

// readConfig reads the configuration from the config file
// This is private - use ConfigManager methods instead
func readConfig() (Config, error) {
	var config Config
	data, err := os.ReadFile(ConfigFilePath)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(data, &config)
	return config, err
}

// writeConfig writes the configuration to the config file
// This is private - use ConfigManager methods instead
func writeConfig(config Config) error {
	data, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFilePath, data, 0600)
}

// isTokenExpired checks if a token has expired (24 hour expiry)
func isTokenExpired(lastUpdated time.Time) bool {
	return time.Since(lastUpdated) >= 24*time.Hour
}

// SimpleConfigWriter provides config writing functionality without circular dependencies
type SimpleConfigWriter struct{}

// UpdateAuthConfig updates authentication-related configuration while preserving other settings
func (s *SimpleConfigWriter) UpdateAuthConfig(username, password, accessToken string) error {
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
