package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Page orientation constants
const (
	OrientationPortrait  = "portrait"
	OrientationLandscape = "landscape"
)

// Page size constants
const (
	PageSizeLetter = "letter"
	PageSizeLegal  = "legal"
	PageSizeA4     = "A4"
)

// DocumentTemplate represents a reusable document template for generating legal documents
type DocumentTemplate struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Firm relationship (multi-tenant scoping)
	FirmID string `gorm:"type:uuid;not null;index" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	// Template identification
	Name        string  `gorm:"not null" json:"name"`
	Description *string `gorm:"type:text" json:"description,omitempty"`

	// Category relationship
	CategoryID *string           `gorm:"type:uuid;index" json:"category_id,omitempty"`
	Category   *TemplateCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`

	// Content (HTML with {{variable}} placeholders)
	Content string `gorm:"type:text;not null" json:"content"`

	// Versioning
	Version int `gorm:"not null;default:1" json:"version"`

	// Status
	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	// Created by
	CreatedByID string `gorm:"type:uuid;not null" json:"created_by_id"`
	CreatedBy   User   `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`

	// PDF Settings
	PageOrientation string `gorm:"not null;default:portrait" json:"page_orientation"`
	PageSize        string `gorm:"not null;default:letter" json:"page_size"`
	MarginTop       int    `gorm:"not null;default:72" json:"margin_top"` // 72 points = 1 inch
	MarginBottom    int    `gorm:"not null;default:72" json:"margin_bottom"`
	MarginLeft      int    `gorm:"not null;default:72" json:"margin_left"`
	MarginRight     int    `gorm:"not null;default:72" json:"margin_right"`
}

// BeforeCreate hook to generate UUID
func (t *DocumentTemplate) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for DocumentTemplate model
func (DocumentTemplate) TableName() string {
	return "document_templates"
}

// IsValidOrientation checks if the orientation is valid
func IsValidOrientation(orientation string) bool {
	return orientation == OrientationPortrait || orientation == OrientationLandscape
}

// IsValidPageSize checks if the page size is valid
func IsValidPageSize(size string) bool {
	return size == PageSizeLetter || size == PageSizeLegal || size == PageSizeA4
}
