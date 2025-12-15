package internal

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Env         string
	Port        int
	LogLevel    string
	DatabaseUrl string
}

func NewConfig() (*Config, error) {
	cfg := &Config{
		Env:      getEnv("ENV", "development"),
		Port:     getEnvInt("PORT", 8080),
		LogLevel: getEnv("LOG_LEVEL", "debug"),
	}

	// Required
	cfg.DatabaseUrl = os.Getenv("DATABASE_URL")
	if cfg.DatabaseUrl == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}
