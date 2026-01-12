package db

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Initialize sets up the database connection and runs migrations
func Initialize(dbPath string) error {
	var err error

	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection established")
	return nil
}

// AutoMigrate runs database migrations for the provided models
func AutoMigrate(models ...interface{}) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	err := DB.AutoMigrate(models...)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations completed")
	return nil
}

// Close closes the database connection
func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	return sqlDB.Close()
}
