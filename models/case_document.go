package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CaseDocument represents a document attached to a case or case request
type CaseDocument struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Firm relationship (for scoping)
	FirmID string `gorm:"type:uuid;not null;index" json:"firm_id"`
	Firm   Firm   `gorm:"foreignKey:FirmID" json:"firm,omitempty"`

	// Relationships - can belong to either CaseRequest OR Case (or both)
	CaseRequestID *string      `gorm:"type:uuid;index" json:"case_request_id,omitempty"`
	CaseRequest   *CaseRequest `gorm:"foreignKey:CaseRequestID" json:"case_request,omitempty"`

	CaseID *string `gorm:"type:uuid;index" json:"case_id,omitempty"`
	Case   *Case   `gorm:"foreignKey:CaseID" json:"case,omitempty"`

	// File metadata
	FileName         string `gorm:"not null" json:"file_name"`
	FileOriginalName string `gorm:"not null" json:"file_original_name"`
	FilePath         string `gorm:"not null" json:"-"` // Not exposed in JSON for security
	FileSize         int64  `gorm:"not null" json:"file_size"`
	MimeType         string `json:"mime_type,omitempty"`

	// Document metadata
	DocumentType string  `json:"document_type,omitempty"` // e.g., "initial_request", "evidence", "contract", etc.
	Description  *string `gorm:"type:text" json:"description,omitempty"`
	IsPublic     bool    `gorm:"default:false" json:"is_public"` // If true, clients can view this document

	// Upload tracking
	UploadedByID *string `gorm:"type:uuid" json:"uploaded_by_id,omitempty"`
	UploadedBy   *User   `gorm:"foreignKey:UploadedByID" json:"uploaded_by,omitempty"`
}

// BeforeCreate hook to generate UUID
func (d *CaseDocument) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// GetDownloadURL returns a safe download URL for this document
func (d *CaseDocument) GetDownloadURL() string {
	if d.CaseID != nil {
		return "/api/cases/" + *d.CaseID + "/documents/" + d.ID + "/download"
	}
	if d.CaseRequestID != nil {
		return "/api/case-requests/" + *d.CaseRequestID + "/documents/" + d.ID + "/download"
	}
	return ""
}
