package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	// Server
	Port        int
	Environment string // development, staging, production
	GinMode     string // debug, release

	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Encryption
	EncryptionKey  string // base64 32 bytes
	EncryptionSalt string // base64 16 bytes

	// JWT
	JWTSecret string // base64 64 bytes
	JWTExpiry time.Duration

	// OAuth (Uber)
	UberClientID     string
	UberClientSecret string
	UberRedirectURI  string
	// UberUseMock selects MockUberClient over the real HTTP client. Defaults
	// to true because partner.accounts/partner.trips/partner.payments scope
	// access is still pending Uber's approval (see
	// .specs/features/api-endpoints/uber-integration.md) — flip to false
	// once access is granted.
	UberUseMock bool

	// Redis
	RedisHost     string
	RedisPort     int
	RedisPassword string

	// Logging
	LogLevel  string // debug, info, warn, error
	LogFormat string // json

	// Security
	CORSOrigins       []string
	RateLimitEnabled  bool
	RateLimitRequests int
	RateLimitWindow   int // seconds

	// WhatsApp (fase 2)
	WhatsAppBusinessAPIKey string
	WhatsAppWebhookSecret  string

	// Sentry
	SentryDSN string
}

// Load reads configuration from environment variables, applying sane defaults
// for anything not set. It does not read the .env file itself — callers are
// responsible for loading it (e.g. via godotenv) before calling Load.
func Load() *Config {
	environment := getEnv("ENVIRONMENT", "development")

	return &Config{
		Port:        getEnvInt("PORT", 8080),
		Environment: environment,
		// GIN_MODE always wins when set explicitly; otherwise it follows
		// ENVIRONMENT so production never defaults to gin's verbose debug
		// mode by accident.
		GinMode: getEnv("GIN_MODE", defaultGinMode(environment)),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 3306),
		DBUser:     getEnv("DB_USER", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "pilot_db"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		EncryptionKey:  getEnv("ENCRYPTION_KEY", ""),
		EncryptionSalt: getEnv("ENCRYPTION_SALT", ""),

		JWTSecret: getEnv("JWT_SECRET", ""),
		JWTExpiry: getEnvDuration("JWT_EXPIRY", 24*time.Hour),

		UberClientID:     getEnv("UBER_CLIENT_ID", ""),
		UberClientSecret: getEnv("UBER_CLIENT_SECRET", ""),
		UberRedirectURI:  getEnv("UBER_REDIRECT_URI", ""),
		UberUseMock:      getEnvBool("UBER_USE_MOCK", true),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		CORSOrigins:       getEnvList("CORS_ORIGINS", nil),
		RateLimitEnabled:  getEnvBool("RATE_LIMIT_ENABLED", true),
		RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   getEnvInt("RATE_LIMIT_WINDOW", 60),

		WhatsAppBusinessAPIKey: getEnv("WHATSAPP_BUSINESS_API_KEY", ""),
		WhatsAppWebhookSecret:  getEnv("WHATSAPP_WEBHOOK_SECRET", ""),

		SentryDSN: getEnv("SENTRY_DSN", ""),
	}
}

// defaultGinMode maps our ENVIRONMENT values onto gin's mode constants:
// only "development" gets gin's verbose debug logging.
func defaultGinMode(environment string) string {
	if environment == "development" {
		return "debug"
	}
	return "release"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if durVal, err := time.ParseDuration(value); err == nil {
			return durVal
		}
	}
	return defaultValue
}

func getEnvList(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
