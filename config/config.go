package config

import (
	"404skill-cli/auth"
	"404skill-cli/supabase"
	"context"
	"fmt"
	"io/ioutil"
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

// ReadConfig reads the configuration from the config file
func ReadConfig() (Config, error) {
	var config Config
	data, err := ioutil.ReadFile(ConfigFilePath)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(data, &config)
	return config, err
}

func WriteConfig(config Config) error {
	data, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(ConfigFilePath, data, 0600)
}

func IsTokenExpired(lastUpdated time.Time) bool {
	return time.Since(lastUpdated) >= 1*time.Hour
}

// ConfigTokenProvider implements the TokenProvider interface
type ConfigTokenProvider struct{}

// NewConfigTokenProvider creates a new ConfigTokenProvider
func NewConfigTokenProvider() *ConfigTokenProvider {
	return &ConfigTokenProvider{}
}

// GetToken implements the TokenProvider interface
func (p *ConfigTokenProvider) GetToken() (string, error) {
	config, err := ReadConfig()
	if err != nil {
		return "", err
	}

	if IsTokenExpired(config.LastUpdated) || config.AccessToken == "" {
		client, err := supabase.NewSupabaseClient()
		if err != nil {
			return "", fmt.Errorf("failed to create supabase client: %w", err)
		}

		authProvider := auth.NewSupabaseAuth(client)
		accessToken, err := authProvider.SignIn(context.Background(), config.Username, config.Password)
		if err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}

		cfg := Config{
			Username:           config.Username,
			Password:           config.Password,
			AccessToken:        accessToken,
			LastUpdated:        time.Now(),
			DownloadedProjects: config.DownloadedProjects,
		}

		if err := WriteConfig(cfg); err != nil {
			return "", fmt.Errorf("failed to save new token: %w", err)
		}

		return accessToken, nil
	}

	return config.AccessToken, nil
}
