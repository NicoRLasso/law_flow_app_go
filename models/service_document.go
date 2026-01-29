package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service document type constants
const (
	ServiceDocTypeIntake      = "INTAKE"      // Initial request documents from client
	ServiceDocTypeWorking     = "WORKING"     // Work in progress documents
	ServiceDocTypeDeliverable = "DELIVERABLE" // Final output/deliverable
	ServiceDocTypeReference   = "REFERENCE"   // Reference materials
)

// ServiceDocument represents a document attached to a legal service
type ServiceDocument struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Scoping
	FirmID    string       `gorm:"type:uuid;not null;index" json:"firm_id"`
	Firm      Firm         `gorm:"foreignKey:FirmID" json:"firm,omitempty"`
	ServiceID string       `gorm:"type:uuid;not null;index:idx_service_doc" json:"service_id"`
	Service   LegalService `gorm:"foreignKey:ServiceID" json:"service,omitempty"`

	// File metadata
	FileName         string `gorm:"not null" json:"file_name"`
	FileOriginalName string `gorm:"not null" json:"file_original_name"`
	FilePath         string `gorm:"not null" json:"-"` // Not exposed in JSON for security
	FileSize         int64  `gorm:"not null" json:"file_size"`
	MimeType         string `json:"mime_type,omitempty"`

	// Document classification
	DocumentType string  `gorm:"not null;default:REFERENCE" json:"document_type"`
	Description  *string `gorm:"type:text" json:"description,omitempty"`
	IsPublic     bool    `gorm:"default:false" json:"is_public"` // If true, clients can view this document

	// Template link (if generated from template)
	GeneratedFromTemplateID *string           `gorm:"type:uuid" json:"generated_from_template_id,omitempty"`
	GeneratedFromTemplate   *DocumentTemplate `gorm:"foreignKey:GeneratedFromTemplateID" json:"-"`

	// Upload tracking
	UploadedByID *string `gorm:"type:uuid" json:"uploaded_by_id,omitempty"`
	UploadedBy   *User   `gorm:"foreignKey:UploadedByID" json:"uploaded_by,omitempty"`
}

// BeforeCreate hook to generate UUID
func (d *ServiceDocument) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for ServiceDocument model
func (ServiceDocument) TableName() string {
	return "service_documents"
}

// GetDownloadURL returns a safe download URL for this document
func (d *ServiceDocument) GetDownloadURL() string {
	return "/api/services/" + d.ServiceID + "/documents/" + d.ID + "/download"
}

// IsIntake checks if the document is an intake document
func (d *ServiceDocument) IsIntake() bool {
	return d.DocumentType == ServiceDocTypeIntake
}

// IsDeliverable checks if the document is a deliverable
func (d *ServiceDocument) IsDeliverable() bool {
	return d.DocumentType == ServiceDocTypeDeliverable
}

// IsWorking checks if the document is a working document
func (d *ServiceDocument) IsWorking() bool {
	return d.DocumentType == ServiceDocTypeWorking
}

// IsReference checks if the document is a reference document
func (d *ServiceDocument) IsReference() bool {
	return d.DocumentType == ServiceDocTypeReference
}

// IsValidServiceDocType checks if the document type is valid
func IsValidServiceDocType(docType string) bool {
	validTypes := []string{
		ServiceDocTypeIntake,
		ServiceDocTypeWorking,
		ServiceDocTypeDeliverable,
		ServiceDocTypeReference,
	}
	for _, t := range validTypes {
		if t == docType {
			return true
		}
	}
	return false
}

// GetServiceDocTypeDisplayName returns human-readable document type name
func GetServiceDocTypeDisplayName(docType string) string {
	names := map[string]string{
		ServiceDocTypeIntake:      "Intake Document",
		ServiceDocTypeWorking:     "Working Document",
		ServiceDocTypeDeliverable: "Deliverable",
		ServiceDocTypeReference:   "Reference",
	}
	if name, ok := names[docType]; ok {
		return name
	}
	return docType
}
