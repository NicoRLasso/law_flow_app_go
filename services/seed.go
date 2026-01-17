package services

import (
	"law_flow_app_go/models"
	"log"
	"os"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeedSuperadminFromEnv creates a superadmin user from environment variables
// Only runs if SUPERADMIN_EMAIL and SUPERADMIN_PASSWORD are set
// and no superadmin user exists yet
func SeedSuperadminFromEnv(db *gorm.DB) error {
	email := os.Getenv("SUPERADMIN_EMAIL")
	password := os.Getenv("SUPERADMIN_PASSWORD")
	name := os.Getenv("SUPERADMIN_NAME")

	// Skip if env vars not set
	if email == "" || password == "" {
		return nil
	}

	if name == "" {
		name = "Superadmin"
	}

	// Check if superadmin already exists
	var count int64
	if err := db.Model(&models.User{}).Where("role = ?", "superadmin").Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		log.Println("[SEED] Superadmin user already exists, skipping seed")
		return nil
	}

	// Check if a user with this email already exists
	var existingUser models.User
	if err := db.Where("email = ?", email).First(&existingUser).Error; err == nil {
		log.Printf("[SEED] User with email %s already exists, skipping superadmin seed", email)
		return nil
	}

	// Hash password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}

	// Create superadmin user (no firm association)
	user := &models.User{
		ID:       uuid.New().String(),
		Name:     name,
		Email:    email,
		Password: hashedPassword,
		Role:     "superadmin",
		IsActive: true,
		FirmID:   nil, // Superadmins have no firm
	}

	if err := db.Create(user).Error; err != nil {
		return err
	}

	log.Printf("[SEED] Created superadmin user: %s", email)
	return nil
}
