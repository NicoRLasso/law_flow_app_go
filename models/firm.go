package models

import (
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
	Country      string `gorm:"not null" json:"country"`
	Timezone     string `gorm:"not null;default:UTC" json:"timezone"`
	Address      string `json:"address"`
	City         string `json:"city"`
	Phone        string `json:"phone"`
	Description  string `gorm:"type:text" json:"description"`
	BillingEmail string `gorm:"not null" json:"billing_email"`
	InfoEmail    string `json:"info_email"`
	NoreplyEmail string `json:"noreply_email"`

	// Relationships
	Users []User `gorm:"foreignKey:FirmID" json:"-"`
}

// BeforeCreate hook to generate UUID
func (f *Firm) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for Firm model
func (Firm) TableName() string {
	return "firms"
}
