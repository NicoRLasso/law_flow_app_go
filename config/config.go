package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort     string
	DBPath         string
	Environment    string
	UploadDir      string
	SMTPHost       string
	SMTPPort       string
	SMTPUsername   string
	SMTPPassword   string
	EmailFrom      string
	EmailFromName  string
	AllowedOrigins []string
	AppURL         string
	SessionSecret  string
}

func Load() *Config {
	// Load .env file (ignore error if not present - use system env vars)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	return &Config{
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		DBPath:         getEnv("DB_PATH", "db/app.db"),
		Environment:    getEnv("ENVIRONMENT", "development"),
		UploadDir:      getEnv("UPLOAD_DIR", "uploads"),
		SMTPHost:       getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:       getEnv("SMTP_PORT", "587"),
		SMTPUsername:   getEnv("SMTP_USERNAME", ""),
		SMTPPassword:   getEnv("SMTP_PASSWORD", ""),
		EmailFrom:      getEnv("EMAIL_FROM", "noreply@lawflowapp.com"),
		EmailFromName:  getEnv("EMAIL_FROM_NAME", "LawFlow App"),
		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "*"), ","),
		AppURL:         getEnv("APP_URL", "http://localhost:8080"),
		SessionSecret:  getEnv("SESSION_SECRET", ""),
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
