package config

import (
    "os"
)

func GetBaseURL() string {
    env := os.Getenv("ENV")
    if env == "production" {
        return os.Getenv("BASE_URL_PROD")
    }
    return os.Getenv("BASE_URL_DEV")
} 