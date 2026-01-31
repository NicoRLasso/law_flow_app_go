package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ConsentType represents the type of consent given
type ConsentType string

const (
	ConsentTypeDataProcessing ConsentType = "DATA_PROCESSING" // Authorization for data treatment (Law 1581)
	ConsentTypeMarketing      ConsentType = "MARKETING"       // Optional marketing communications
	ConsentTypeCookies        ConsentType = "COOKIES"         // Cookie acceptance
)

// ConsentLog represents an immutable record of a user's consent to data processing.
// Required by Law 1581 of 2012 (Colombia) - Habeas Data.
type ConsentLog struct {
	ID        string    `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time `gorm:"index:idx_consent_created_at" json:"created_at"`

	// Subject identification
	UserID    string `gorm:"type:uuid;not null;index:idx_consent_user" json:"user_id"`
	UserEmail string `gorm:"not null" json:"user_email"` // Denormalized for historical accuracy

	// Firm scope (optional, for clients associated with a firm)
	FirmID *string `gorm:"type:uuid;index:idx_consent_firm" json:"firm_id,omitempty"`

	// Consent details
	ConsentType   ConsentType `gorm:"not null;index:idx_consent_type" json:"consent_type"`
	Granted       bool        `gorm:"not null" json:"granted"` // true = granted, false = revoked
	PolicyVersion string      `gorm:"not null" json:"policy_version"`
	PolicyText    string      `gorm:"type:text;not null" json:"policy_text"` // Snapshot of policy at time of consent

	// Request metadata (for legal evidence)
	IPAddress string `gorm:"not null" json:"ip_address"`
	UserAgent string `gorm:"not null" json:"user_agent"`

	// Relationships (for reading, not for data integrity)
	User *User `gorm:"foreignKey:UserID" json:"-"`
	Firm *Firm `gorm:"foreignKey:FirmID" json:"-"`
}

// BeforeCreate generates UUID and prevents modification
func (c *ConsentLog) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// BeforeUpdate prevents modification of consent logs (immutability)
func (c *ConsentLog) BeforeUpdate(tx *gorm.DB) error {
	return gorm.ErrRecordNotFound // Prevent any updates
}

// BeforeDelete prevents deletion of consent logs (immutability)
func (c *ConsentLog) BeforeDelete(tx *gorm.DB) error {
	return gorm.ErrRecordNotFound // Prevent any deletes
}

// TableName specifies the table name
func (ConsentLog) TableName() string {
	return "consent_logs"
}
