package main

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"log"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	if err := db.Initialize(cfg.DBPath, cfg.Environment); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Starting slug migration for existing firms...")

	// Fetch all firms without slugs
	var firms []models.Firm
	if err := db.DB.Where("slug = ? OR slug IS NULL", "").Find(&firms).Error; err != nil {
		log.Fatalf("Failed to fetch firms: %v", err)
	}

	if len(firms) == 0 {
		log.Println("No firms need slug migration. All firms already have slugs.")
		return
	}

	log.Printf("Found %d firms without slugs. Generating slugs...\n", len(firms))

	// Generate and update slugs
	for i, firm := range firms {
		slug := generateSlug(db.DB, firm.Name)
		firm.Slug = slug

		if err := db.DB.Model(&firm).Update("slug", slug).Error; err != nil {
			log.Printf("Failed to update slug for firm %s (ID: %s): %v\n", firm.Name, firm.ID, err)
			continue
		}

		log.Printf("[%d/%d] Generated slug '%s' for firm '%s'\n", i+1, len(firms), slug, firm.Name)
	}

	log.Println("Slug migration completed successfully!")
}

// generateSlug creates a URL-friendly slug from the firm name
func generateSlug(tx *gorm.DB, name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove special characters (keep only alphanumeric and hyphens)
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	slug = reg.ReplaceAllString(slug, "")

	// Remove consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	// Limit to 50 characters
	if len(slug) > 50 {
		slug = slug[:50]
		slug = strings.TrimRight(slug, "-")
	}

	// Ensure uniqueness
	originalSlug := slug
	counter := 1
	for {
		var count int64
		tx.Model(&models.Firm{}).Where("slug = ?", slug).Count(&count)
		if count == 0 {
			break
		}
		slug = originalSlug + "-" + strconv.Itoa(counter)
		counter++
	}

	return slug
}
