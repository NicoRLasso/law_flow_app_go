package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/tursodatabase/libsql-client-go/libsql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// DatabaseConfig holds the configuration for database connection
type DatabaseConfig struct {
	DBPath           string
	Environment      string
	TursoDatabaseURL string
	TursoAuthToken   string
}

// Initialize sets up the database connection (Turso or local SQLite)
func Initialize(dbPath string, environment string) error {
	return InitializeWithConfig(DatabaseConfig{
		DBPath:      dbPath,
		Environment: environment,
	})
}

// InitializeWithConfig sets up the database connection with full configuration
func InitializeWithConfig(cfg DatabaseConfig) error {
	var err error

	// Determine log level based on environment
	logLevel := logger.Info
	if cfg.Environment == "production" {
		logLevel = logger.Warn
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	// Check if Turso is configured
	if cfg.TursoDatabaseURL != "" && cfg.TursoAuthToken != "" {
		DB, err = connectTurso(cfg.TursoDatabaseURL, cfg.TursoAuthToken, gormConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to Turso database: %w", err)
		}
		log.Println("Database connection established (Turso)")
	} else {
		DB, err = connectLocalSQLite(cfg.DBPath, gormConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to local SQLite database: %w", err)
		}
		log.Println("Database connection established (Local SQLite with WAL mode)")
	}

	return nil
}

// connectTurso establishes a connection to Turso database
func connectTurso(databaseURL, authToken string, gormConfig *gorm.Config) (*gorm.DB, error) {
	connector, err := libsql.NewConnector(databaseURL, libsql.WithAuthToken(authToken))
	if err != nil {
		return nil, fmt.Errorf("failed to create Turso connector: %w", err)
	}

	sqlDB := sql.OpenDB(connector)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping Turso database: %w", err)
	}

	// Use GORM with the existing sql.DB connection
	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GORM with Turso: %w", err)
	}

	return db, nil
}

// connectLocalSQLite establishes a connection to local SQLite database
func connectLocalSQLite(dbPath string, gormConfig *gorm.Config) (*gorm.DB, error) {
	// Enable WAL mode for better concurrency support
	dsn := dbPath + "?_journal_mode=WAL"

	db, err := gorm.Open(sqlite.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	return db, nil
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
