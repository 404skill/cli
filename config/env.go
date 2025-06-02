package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

var (
	enbeddedEnv     string
	enbeddedBaseURL string
)

func GetBaseURL() (string, error) {
	if enbeddedEnv != "" {
		return enbeddedBaseURL, nil
	}
	if err := godotenv.Load(); err != nil {
		return "", fmt.Errorf("failed to load environment: %w", err)
	}
	env := os.Getenv("ENV")
	if env == "production" {
		return os.Getenv("BASE_URL_PROD"), nil
	}
	return os.Getenv("BASE_URL_DEV"), nil
}
