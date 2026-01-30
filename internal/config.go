package internal

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

	// Storage Configuration
	StorageProvider string // "local" or "r2"

	// Local Storage (development)
	LocalStoragePath string // Base directory for local file storage
	LocalStorageURL  string // Base URL for accessing local files

	// R2 Storage (production)
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2PublicURL       string // Optional custom domain URL

	// Worker Configuration
	WorkerEnabled      bool
	WorkerConcurrency  int
	WorkerPollInterval time.Duration
	WorkerJobTimeout   time.Duration

	// AI Provider Configuration
	AIProvider       string // "anthropic" or "mock"
	AnthropicAPIKey  string
	AnthropicModel   string
	AIMaxRetries     int
	AIRetryBaseDelay time.Duration
	AIRequestTimeout time.Duration

	// Invite code system (MVP testing)
	InviteCodesEnabled bool     // Enable/disable invite code requirement
	ValidInviteCodes   []string // List of valid codes to accept

	// Admin access control
	AdminEmails []string // List of email addresses with admin access

	// Stripe Billing Configuration
	// These are required when billing is enabled in production.
	// In development, billing handlers function as stubs if these are empty.
	StripeSecretKey     string // Stripe API secret key (sk_test_... or sk_live_...)
	StripeWebhookSecret string // Stripe webhook signing secret (whsec_...)

	// Stripe Price IDs for subscription plans
	StripeStarterMonthlyPriceID      string
	StripeStarterYearlyPriceID       string
	StripeProfessionalMonthlyPriceID string
	StripeProfessionalYearlyPriceID  string

	// Metrics endpoint authentication
	// If both are empty, the /metrics endpoint will be unprotected (not recommended)
	MetricsUsername string
	MetricsPassword string
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

		// Storage defaults to local filesystem for development
		StorageProvider:  getEnv("STORAGE_PROVIDER", "local"),
		LocalStoragePath: getEnv("LOCAL_STORAGE_PATH", "./storage"),
		LocalStorageURL:  getEnv("LOCAL_STORAGE_URL", "http://localhost:8080/files"),

		// R2 configuration (production only)
		R2AccountID:       getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName:      getEnv("R2_BUCKET_NAME", ""),
		R2PublicURL:       getEnv("R2_PUBLIC_URL", ""),

		// Worker defaults
		WorkerEnabled:      getEnvBool("WORKER_ENABLED", true),
		WorkerConcurrency:  getEnvInt("WORKER_CONCURRENCY", 2),
		WorkerPollInterval: getEnvDuration("WORKER_POLL_INTERVAL", 5*time.Second),
		WorkerJobTimeout:   getEnvDuration("WORKER_JOB_TIMEOUT", 5*time.Minute),

		// AI provider defaults
		AIProvider:       getEnv("AI_PROVIDER", "mock"),
		AnthropicAPIKey:  getEnv("ANTHROPIC_API_KEY", ""),
		AnthropicModel:   getEnv("ANTHROPIC_MODEL", "claude-3-5-sonnet-20241022"),
		AIMaxRetries:     getEnvInt("AI_MAX_RETRIES", 3),
		AIRetryBaseDelay: getEnvDuration("AI_RETRY_BASE_DELAY", 1*time.Second),
		AIRequestTimeout: getEnvDuration("AI_REQUEST_TIMEOUT", 60*time.Second),

		// Invite code defaults (enabled by default for MVP testing)
		InviteCodesEnabled: getEnvBool("INVITE_CODES_ENABLED", true),

		// Stripe billing (optional — stubs work without these)
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),

		// Stripe price IDs (optional — required when billing is enabled)
		StripeStarterMonthlyPriceID:      getEnv("STRIPE_STARTER_MONTHLY_PRICE_ID", ""),
		StripeStarterYearlyPriceID:       getEnv("STRIPE_STARTER_YEARLY_PRICE_ID", ""),
		StripeProfessionalMonthlyPriceID: getEnv("STRIPE_PROFESSIONAL_MONTHLY_PRICE_ID", ""),
		StripeProfessionalYearlyPriceID:  getEnv("STRIPE_PROFESSIONAL_YEARLY_PRICE_ID", ""),

		// Metrics authentication
		MetricsUsername: getEnv("METRICS_USERNAME", ""),
		MetricsPassword: getEnv("METRICS_PASSWORD", ""),
	}

	// Parse invite codes from comma-separated environment variable
	inviteCodesStr := getEnv("VALID_INVITE_CODES", "")
	if inviteCodesStr != "" {
		codes := strings.Split(inviteCodesStr, ",")
		for _, code := range codes {
			trimmed := strings.TrimSpace(strings.ToUpper(code))
			if trimmed != "" {
				cfg.ValidInviteCodes = append(cfg.ValidInviteCodes, trimmed)
			}
		}
	}

	// Parse admin emails from comma-separated environment variable
	adminEmailsStr := getEnv("ADMIN_EMAILS", "")
	if adminEmailsStr != "" {
		emails := strings.Split(adminEmailsStr, ",")
		for _, email := range emails {
			trimmed := strings.TrimSpace(strings.ToLower(email))
			if trimmed != "" {
				cfg.AdminEmails = append(cfg.AdminEmails, trimmed)
			}
		}
	}

	// Required
	cfg.DatabaseUrl = os.Getenv("DATABASE_URL")
	if cfg.DatabaseUrl == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	// Validate storage configuration
	if cfg.StorageProvider == "r2" {
		if cfg.R2AccountID == "" {
			return nil, fmt.Errorf("R2_ACCOUNT_ID is required when STORAGE_PROVIDER is 'r2'")
		}
		if cfg.R2AccessKeyID == "" {
			return nil, fmt.Errorf("R2_ACCESS_KEY_ID is required when STORAGE_PROVIDER is 'r2'")
		}
		if cfg.R2SecretAccessKey == "" {
			return nil, fmt.Errorf("R2_SECRET_ACCESS_KEY is required when STORAGE_PROVIDER is 'r2'")
		}
		if cfg.R2BucketName == "" {
			return nil, fmt.Errorf("R2_BUCKET_NAME is required when STORAGE_PROVIDER is 'r2'")
		}
	} else if cfg.StorageProvider != "local" {
		return nil, fmt.Errorf("STORAGE_PROVIDER must be either 'local' or 'r2', got: %s", cfg.StorageProvider)
	}

	// Validate AI provider configuration
	if cfg.AIProvider == "anthropic" {
		if cfg.AnthropicAPIKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY is required when AI_PROVIDER is 'anthropic'")
		}
	} else if cfg.AIProvider != "mock" {
		return nil, fmt.Errorf("AI_PROVIDER must be either 'anthropic' or 'mock', got: %s", cfg.AIProvider)
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

func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}
