package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CaseDomain represents the top-level legal classification (e.g., Derecho Público, Privado)
type CaseDomain struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Firm relationship
	FirmID string `gorm:"type:uuid;not null;index:idx_case_dom_firm_country" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	// Domain metadata
	Country     string `gorm:"not null;index:idx_case_dom_firm_country;index:idx_case_dom_country_active" json:"country"` // Full country name
	Code        string `gorm:"not null" json:"code"`                                                                      // Unique code (e.g., "PUBLICO")
	Name        string `gorm:"not null" json:"name"`                                                                      // Display name
	Description string `gorm:"type:text" json:"description"`
	Order       int    `gorm:"not null;default:0" json:"order"` // For sorting
	IsActive    bool   `gorm:"not null;default:true;index:idx_case_dom_country_active" json:"is_active"`
	IsSystem    bool   `gorm:"not null;default:false" json:"is_system"` // Prevents deletion of system domains

	// Relationships
	Branches []CaseBranch `gorm:"foreignKey:DomainID" json:"branches,omitempty"`
}

// BeforeCreate hook to generate UUID
func (cd *CaseDomain) BeforeCreate(tx *gorm.DB) error {
	if cd.ID == "" {
		cd.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for CaseDomain model
func (CaseDomain) TableName() string {
	return "case_domains"
}
