package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FirmUsage tracks current usage metrics for a firm (cached/denormalized)
type FirmUsage struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Firm relationship
	FirmID string `gorm:"type:uuid;not null;uniqueIndex" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	// Current usage counts
	CurrentUsers        int   `gorm:"not null;default:0" json:"current_users"`
	CurrentStorageBytes int64 `gorm:"not null;default:0" json:"current_storage_bytes"`
	CurrentCases        int   `gorm:"not null;default:0" json:"current_cases"`

	// Last recalculation timestamp
	LastCalculatedAt time.Time `gorm:"not null" json:"last_calculated_at"`
}

// BeforeCreate hook to generate UUID
func (fu *FirmUsage) BeforeCreate(tx *gorm.DB) error {
	if fu.ID == "" {
		fu.ID = uuid.New().String()
	}
	if fu.LastCalculatedAt.IsZero() {
		fu.LastCalculatedAt = time.Now()
	}
	return nil
}

// TableName specifies the table name
func (FirmUsage) TableName() string {
	return "firm_usages"
}

// FormatStorageUsed returns human-readable storage usage
func (fu *FirmUsage) FormatStorageUsed() string {
	return FormatBytes(fu.CurrentStorageBytes)
}

// FormatBytes converts bytes to human-readable format
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

// IsStale checks if the usage data is older than 1 hour
func (fu *FirmUsage) IsStale() bool {
	return time.Since(fu.LastCalculatedAt) > time.Hour
}
