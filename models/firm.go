package models

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Firm struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name         string `gorm:"not null" json:"name"`
	Slug         string `gorm:"uniqueIndex;not null" json:"slug"`
	Country      string `gorm:"not null" json:"country"`
	Timezone     string `gorm:"not null;default:UTC" json:"timezone"`
	Address      string `json:"address"`
	City         string `json:"city"`
	Phone        string `json:"phone"`
	Description  string `gorm:"type:text" json:"description"`
	BillingEmail string `gorm:"not null" json:"billing_email"`
	InfoEmail    string `json:"info_email"`
	NoreplyEmail string `json:"noreply_email"`

	// Availability settings
	BufferMinutes int `gorm:"not null;default:15" json:"buffer_minutes"` // Buffer between appointments (30, 45, or 60 min)

	// Relationships
	Users []User `gorm:"foreignKey:FirmID" json:"-"`
}

// BeforeCreate hook to generate UUID and slug
func (f *Firm) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	if f.Slug == "" {
		f.Slug = generateSlug(tx, f.Name)
	}
	return nil
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
		tx.Model(&Firm{}).Where("slug = ?", slug).Count(&count)
		if count == 0 {
			break
		}
		slug = originalSlug + "-" + strconv.Itoa(counter)
		counter++
	}

	return slug
}

// TableName specifies the table name for Firm model
func (Firm) TableName() string {
	return "firms"
}
