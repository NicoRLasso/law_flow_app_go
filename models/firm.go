package models

import (
	"fmt"
	"math/rand"
	"regexp"
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

	Name            string `gorm:"not null" json:"name"`
	Slug            string `gorm:"uniqueIndex;not null" json:"slug"`
	Country         string `gorm:"not null" json:"country"`
	Timezone        string `gorm:"not null;default:UTC" json:"timezone"`
	Address         string `json:"address"`
	City            string `json:"city"`
	Phone           string `json:"phone"`
	Description     string `gorm:"type:text" json:"description"`
	BillingEmail    string `gorm:"not null" json:"billing_email"`
	InfoEmail       string `json:"info_email"`
	NoreplyEmail    string `gorm:"not null" json:"noreply_email"`
	EmailSenderName string `gorm:"not null" json:"email_sender_name"`
	IsActive        bool   `gorm:"not null;default:true" json:"is_active"`

	// Branding
	LogoURL string `json:"logo_url"` // Path to firm's logo image

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
		f.Slug = GenerateSlug(tx, f.Name)
	}
	return nil
}

// GenerateSlug creates a URL-friendly slug from the firm name with a random suffix
func GenerateSlug(tx *gorm.DB, name string) string {
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

	// Limit base slug length to 40 characters to leave room for suffix
	if len(slug) > 40 {
		slug = slug[:40]
		slug = strings.TrimRight(slug, "-")
	}

	// Generate a random 6-character alphanumeric suffix
	suffix := generateRandomString(6)

	// Combine base slug and suffix
	finalSlug := fmt.Sprintf("%s-%s", slug, suffix)

	// Ensure uniqueness (extremely unlikely to collide but safe to check)
	for {
		var count int64
		tx.Model(&Firm{}).Where("slug = ?", finalSlug).Count(&count)
		if count == 0 {
			break
		}
		// Regenerate suffix if collision occurs
		suffix = generateRandomString(6)
		finalSlug = fmt.Sprintf("%s-%s", slug, suffix)
	}

	return finalSlug
}

// generateRandomString generates a random alphanumeric string of length n
func generateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// TableName specifies the table name for Firm model
func (Firm) TableName() string {
	return "firms"
}
