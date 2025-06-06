package config

import (
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

// readConfig reads the configuration from the config file
// This is private - use ConfigManager methods instead
func readConfig() (Config, error) {
	var config Config
	data, err := ioutil.ReadFile(ConfigFilePath)
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
	return ioutil.WriteFile(ConfigFilePath, data, 0600)
}

// isTokenExpired checks if a token has expired (24 hour expiry)
func isTokenExpired(lastUpdated time.Time) bool {
	return time.Since(lastUpdated) >= 24*time.Hour
}
