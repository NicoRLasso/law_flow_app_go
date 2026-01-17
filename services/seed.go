package services

import (
	"law_flow_app_go/models"
	"log"
	"os"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeedAdminFromEnv creates an initial admin user from environment variables
// Only runs if ADMIN_EMAIL and ADMIN_PASSWORD are set AND no users exist
func SeedAdminFromEnv(db *gorm.DB) error {
	email := os.Getenv("ADMIN_EMAIL")
	password := os.Getenv("ADMIN_PASSWORD")
	name := os.Getenv("ADMIN_NAME")

	// Skip if env vars not set
	if email == "" || password == "" {
		return nil
	}

	if name == "" {
		name = "Admin"
	}

	// Check if any users exist
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		log.Println("[SEED] Users already exist, skipping admin seed")
		return nil
	}

	// Hash password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}

	// Create admin user
	user := &models.User{
		ID:       uuid.New().String(),
		Name:     name,
		Email:    email,
		Password: hashedPassword,
		Role:     "admin",
		IsActive: true,
	}

	if err := db.Create(user).Error; err != nil {
		return err
	}

	log.Printf("[SEED] ✅ Created admin user: %s", email)
	return nil
}
