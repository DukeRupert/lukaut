package internal

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Env         string
	Port        int
	LogLevel    string
	DatabaseUrl string

	// SMTP Configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string

	// Application base URL (for email links)
	BaseURL string
}

func NewConfig() (*Config, error) {
	// Load .env file if it exists (ignored in production)
	_ = godotenv.Load()

	cfg := &Config{
		Env:      getEnv("ENV", "development"),
		Port:     getEnvInt("PORT", 8080),
		LogLevel: getEnv("LOG_LEVEL", "debug"),

		// SMTP defaults for Mailhog (development)
		SMTPHost:     getEnv("SMTP_HOST", "localhost"),
		SMTPPort:     getEnvInt("SMTP_PORT", 1025),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "noreply@lukaut.com"),
		SMTPFromName: getEnv("SMTP_FROM_NAME", "Lukaut"),

		// Base URL defaults to localhost for development
		BaseURL: getEnv("BASE_URL", "http://localhost:8080"),
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
