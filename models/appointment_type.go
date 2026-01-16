package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AppointmentType represents a configurable appointment type per firm
type AppointmentType struct {
	ID        string         `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	FirmID          string `gorm:"type:uuid;index;not null" json:"firm_id"`
	Name            string `gorm:"size:100;not null" json:"name"` // "Initial Consultation", "Follow-up"
	Description     string `gorm:"type:text" json:"description,omitempty"`
	DurationMinutes int    `gorm:"default:60" json:"duration_minutes"`    // Default 60 min
	Color           string `gorm:"size:7;default:'#3B82F6'" json:"color"` // Hex color for calendar
	IsActive        bool   `gorm:"default:true;index" json:"is_active"`
	Order           int    `gorm:"default:0" json:"order"` // Display ordering

	// Relationships
	Firm         Firm          `gorm:"foreignKey:FirmID" json:"firm,omitempty"`
	Appointments []Appointment `gorm:"foreignKey:AppointmentTypeID" json:"appointments,omitempty"`
}

// BeforeCreate hook to generate UUID
func (at *AppointmentType) BeforeCreate(tx *gorm.DB) error {
	if at.ID == "" {
		at.ID = uuid.New().String()
	}
	return nil
}

// TableName specifies the table name
func (AppointmentType) TableName() string {
	return "appointment_types"
}

// Default appointment types for new firms
var DefaultAppointmentTypes = []struct {
	Name            string
	Description     string
	DurationMinutes int
	Color           string
	Order           int
}{
	{"Initial Consultation", "First meeting with a new client", 60, "#3B82F6", 1},
	{"Follow-up", "Follow-up meeting with existing client", 30, "#10B981", 2},
	{"Case Review", "Detailed case review and strategy", 90, "#8B5CF6", 3},
	{"Document Signing", "Contract or document signing session", 30, "#F59E0B", 4},
	{"Court Preparation", "Preparing client for court appearance", 60, "#EF4444", 5},
}

// CreateDefaultAppointmentTypes creates default types for a firm
func CreateDefaultAppointmentTypes(db *gorm.DB, firmID string) error {
	for _, t := range DefaultAppointmentTypes {
		apt := &AppointmentType{
			FirmID:          firmID,
			Name:            t.Name,
			Description:     t.Description,
			DurationMinutes: t.DurationMinutes,
			Color:           t.Color,
			Order:           t.Order,
			IsActive:        true,
		}
		if err := db.Create(apt).Error; err != nil {
			return err
		}
	}
	return nil
}
