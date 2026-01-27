package config

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	// MinSessionSecretLength is the minimum required length for session secret in production
	MinSessionSecretLength = 32
)

type Config struct {
	ServerPort  string
	DBPath      string
	Environment string
	UploadDir   string
	// Email (Resend)
	ResendAPIKey  string
	EmailFrom     string
	EmailFromName string
	EmailTestMode bool // When true, emails are logged to console instead of sent
	// Other
	AllowedOrigins   []string
	AppURL           string
	SessionSecret    string
	TursoDatabaseURL string
	TursoAuthToken   string
	// Cloudflare Turnstile
	TurnstileSiteKey   string
	TurnstileSecretKey string
	// Cloudflare R2 Storage
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2PublicURL       string
}

func Load() *Config {
	// Load .env file (ignore error if not present - use system env vars)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	environment := getEnv("ENVIRONMENT", "development")
	sessionSecret := getEnv("SESSION_SECRET", "")

	// Validate session secret - this will fatal in production if invalid
	ValidateSessionSecret(sessionSecret, environment)

	// In development, generate a secure secret if none provided
	if sessionSecret == "" && environment != "production" {
		sessionSecret = GenerateSecureSecret()
		log.Println("[INFO] Generated temporary session secret for development. Set SESSION_SECRET env var for persistence.")
	}

	return &Config{
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		DBPath:             getEnv("DB_PATH", "db/app.db"),
		Environment:        environment,
		UploadDir:          getEnv("UPLOAD_DIR", "static/uploads"),
		ResendAPIKey:       getEnv("RESEND_API_KEY", ""),
		EmailFrom:          getEnv("EMAIL_FROM", "noreply@lexlegalcloud.org"),
		EmailFromName:      getEnv("EMAIL_FROM_NAME", "lexlegalcloud App"),
		EmailTestMode:      getEnvBool("EMAIL_TEST_MODE", true), // Default true for safety
		AllowedOrigins:     strings.Split(getEnv("ALLOWED_ORIGINS", "*"), ","),
		AppURL:             getEnv("APP_URL", "http://localhost:8080"),
		SessionSecret:      sessionSecret,
		TursoDatabaseURL:   getEnv("TURSO_DATABASE_URL", ""),
		TursoAuthToken:     getEnv("TURSO_AUTH_TOKEN", ""),
		TurnstileSiteKey:   getEnv("TURNSTILE_SITE_KEY", ""),
		TurnstileSecretKey: getEnv("TURNSTILE_SECRET_KEY", ""),
		R2AccountID:        getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:      getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey:  getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName:       getEnv("R2_BUCKET_NAME", ""),
		R2PublicURL:        getEnv("R2_PUBLIC_URL", ""),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Printf("Using default value for %s: %s", key, defaultValue)
		return defaultValue
	}
	return value
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	// Accept common boolean representations
	switch strings.ToLower(value) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// ValidateSessionSecret validates the session secret meets security requirements
// In production, it must be at least 32 bytes and not a known insecure default
func ValidateSessionSecret(secret string, environment string) error {
	// Known insecure defaults that must be rejected
	insecureDefaults := []string{
		"dev-secret-change-in-production",
		"change-me",
		"secret",
		"development",
		"test",
		"",
	}

	for _, insecure := range insecureDefaults {
		if strings.EqualFold(secret, insecure) {
			if environment == "production" {
				log.Fatal("[CRITICAL] SESSION_SECRET is set to an insecure default value. Generate a secure random secret with: openssl rand -base64 32")
			}
			log.Printf("[WARNING] SESSION_SECRET is set to an insecure default value. This is acceptable only in development.")
			return nil
		}
	}

	if environment == "production" {
		if len(secret) < MinSessionSecretLength {
			log.Fatalf("[CRITICAL] SESSION_SECRET must be at least %d characters in production (current: %d). Generate with: openssl rand -base64 32", MinSessionSecretLength, len(secret))
		}
	}

	return nil
}

// GenerateSecureSecret generates a cryptographically secure random secret
// This is used only for development when no secret is provided
func GenerateSecureSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("[WARNING] Failed to generate secure secret: %v", err)
		return ""
	}
	return base64.StdEncoding.EncodeToString(bytes)
}
