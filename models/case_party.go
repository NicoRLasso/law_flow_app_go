package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CaseParty represents an opposing party (contraparte) in a case
// Each case can have at most one contraparte
type CaseParty struct {
	ID        string    `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Case relationship (unique - one party per case)
	CaseID string `gorm:"type:uuid;not null;uniqueIndex" json:"case_id"`
	Case   Case   `gorm:"foreignKey:CaseID" json:"case,omitempty"`

	// Party type (DEMANDANTE or DEMANDADO - opposite of client's role)
	PartyType string `gorm:"size:20;not null" json:"party_type"`

	// Party information
	Name           string  `gorm:"not null" json:"name"`
	Email          *string `json:"email,omitempty"`
	Phone          *string `json:"phone,omitempty"`
	DocumentTypeID *string `gorm:"type:uuid" json:"document_type_id,omitempty"`
	DocumentNumber *string `json:"document_number,omitempty"`

	// Relations
	DocumentType *ChoiceOption `gorm:"foreignKey:DocumentTypeID" json:"document_type,omitempty"`
}

// BeforeCreate hook to generate UUID
func (cp *CaseParty) BeforeCreate(tx *gorm.DB) error {
	if cp.ID == "" {
		cp.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name for CaseParty model
func (CaseParty) TableName() string {
	return "case_parties"
}

// GetPartyTypeDisplayName returns the display name for the party type
func (cp *CaseParty) GetPartyTypeDisplayName() string {
	if cp.PartyType == ClientRoleDemandante {
		return "Demandante"
	}
	return "Demandado"
}
