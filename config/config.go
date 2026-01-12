package config

import (
	"log"
	"os"
)

type Config struct {
	ServerPort string
	DBPath     string
	Environment string
}

func Load() *Config {
	return &Config{
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		DBPath:      getEnv("DB_PATH", "db/app.db"),
		Environment: getEnv("ENVIRONMENT", "development"),
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
