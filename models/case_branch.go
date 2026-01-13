package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CaseBranch represents the middle-level legal classification (e.g., Derecho Civil, Penal)
type CaseBranch struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Firm relationship
	FirmID string `gorm:"type:uuid;not null;index:idx_case_br_firm_domain;index:idx_case_br_firm_country" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	// Domain relationship
	DomainID string     `gorm:"type:uuid;not null;index:idx_case_br_firm_domain;index:idx_case_br_domain_active" json:"domain_id"`
	Domain   CaseDomain `gorm:"foreignKey:DomainID" json:"domain,omitempty"`

	// Branch metadata
	Country     string `gorm:"not null;index:idx_case_br_firm_country" json:"country"` // Full country name
	Code        string `gorm:"not null" json:"code"`                                   // Unique code (e.g., "CIVIL")
	Name        string `gorm:"not null" json:"name"`                                   // Display name
	Description string `gorm:"type:text" json:"description"`
	Order       int    `gorm:"not null;default:0" json:"order"` // For sorting within domain
	IsActive    bool   `gorm:"not null;default:true;index:idx_case_br_domain_active" json:"is_active"`
	IsSystem    bool   `gorm:"not null;default:false" json:"is_system"` // Prevents deletion of system branches

	// Relationships
	Subtypes []CaseSubtype `gorm:"foreignKey:BranchID" json:"subtypes,omitempty"`
}

// BeforeCreate hook to generate UUID
func (cb *CaseBranch) BeforeCreate(tx *gorm.DB) error {
	if cb.ID == "" {
		cb.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for CaseBranch model
func (CaseBranch) TableName() string {
	return "case_branches"
}
