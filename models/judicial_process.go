package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JudicialProcess represents a legal process tracked from the external Judicial Branch API
type JudicialProcess struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// External Identifiers
	ProcessID string `gorm:"not null;index" json:"process_id"`
	Radicado  string `gorm:"not null;index" json:"radicado"` // Main identifier/filing number

	// Country-Specific Details (JSON)
	// Stores: Office, Judge, Department, ProcessType, Subjects, IsPrivado, etc.
	Details JSONMap `gorm:"type:text" json:"details"`

	// Status & Tracking
	IsPrivado    bool      `json:"is_privado"` // Keep as boolean for quick access if needed, or move to Details
	LastTracking time.Time `json:"last_tracking"`
	Status       string    `gorm:"not null;default:ACTIVE" json:"status"`

	// Dates
	LastActivityDate time.Time `json:"last_activity_date"`

	// Relationships
	CaseID string `gorm:"type:uuid;not null;index" json:"case_id"`
	Case   Case   `gorm:"foreignKey:CaseID" json:"case,omitempty"`

	Actions []JudicialProcessAction `gorm:"foreignKey:JudicialProcessID;constraint:OnDelete:CASCADE" json:"actions,omitempty"`
}

// JSONMap is a helper for storing JSON data in text columns
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = JSONMap{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &j)
}

// JudicialProcessAction represents an action (actuacion) within the process
type JudicialProcessAction struct {
	ID        string    `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	JudicialProcessID string `gorm:"type:uuid;not null;index" json:"judicial_process_id"`

	// Fields from API (Actuaciones)
	ExternalID string    `gorm:"index" json:"external_id"`    // Unique ID in external system
	Type       string    `json:"type"`                        // Action Type/Title
	Annotation string    `gorm:"type:text" json:"annotation"` // Description/Content
	ActionDate time.Time `json:"action_date"`                 // When it happened

	HasDocuments bool    `json:"has_documents"`
	Metadata     JSONMap `gorm:"type:text" json:"metadata"` // Stores extra dates, flags, etc.
}

// BeforeCreate hook to generate UUID
func (j *JudicialProcess) BeforeCreate(tx *gorm.DB) error {
	if j.ID == "" {
		j.ID = uuid.New().String()
	}
	return nil
}

// BeforeCreate hook to generate UUID
func (a *JudicialProcessAction) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// TableName overrides
func (JudicialProcess) TableName() string {
	return "judicial_processes"
}

func (JudicialProcessAction) TableName() string {
	return "judicial_process_actions"
}
