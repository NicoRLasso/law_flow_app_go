package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Complexity levels for case subtypes
const (
	ComplexityLow    = "low"
	ComplexityMedium = "medium"
	ComplexityHigh   = "high"
)

// CaseSubtype represents the bottom-level legal classification (e.g., Divorcio, Custodia)
type CaseSubtype struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Firm relationship
	FirmID string `gorm:"type:uuid;not null;index:idx_case_sub_firm_branch;index:idx_case_sub_firm_country" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	// Branch relationship
	BranchID string     `gorm:"type:uuid;not null;index:idx_case_sub_firm_branch;index:idx_case_sub_branch_active" json:"branch_id"`
	Branch   CaseBranch `gorm:"foreignKey:BranchID" json:"branch,omitempty"`

	// Subtype metadata
	Country     string `gorm:"not null;index:idx_case_sub_firm_country" json:"country"` // Full country name
	Code        string `gorm:"not null" json:"code"`                                    // Unique code
	Name        string `gorm:"not null" json:"name"`                                    // Display name
	Description string `gorm:"type:text" json:"description"`
	Order       int    `gorm:"not null;default:0" json:"order"` // For sorting within branch
	IsActive    bool   `gorm:"not null;default:true;index:idx_case_sub_branch_active" json:"is_active"`
	IsSystem    bool   `gorm:"not null;default:false" json:"is_system"` // Prevents deletion of system subtypes

	// Optional metadata
	TypicalDurationDays *int   `json:"typical_duration_days,omitempty"` // Typical case duration in days
	ComplexityLevel     string `json:"complexity_level,omitempty"`      // low, medium, high
}

// BeforeCreate hook to generate UUID
func (cs *CaseSubtype) BeforeCreate(tx *gorm.DB) error {
	if cs.ID == "" {
		cs.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for CaseSubtype model
func (CaseSubtype) TableName() string {
	return "case_subtypes"
}

// GetDomain is a convenience method to access domain through branch
func (cs *CaseSubtype) GetDomain(db *gorm.DB) (*CaseDomain, error) {
	if cs.Branch.ID != "" && cs.Branch.Domain.ID != "" {
		return &cs.Branch.Domain, nil
	}

	var branch CaseBranch
	if err := db.Preload("Domain").First(&branch, "id = ?", cs.BranchID).Error; err != nil {
		return nil, err
	}

	return &branch.Domain, nil
}
