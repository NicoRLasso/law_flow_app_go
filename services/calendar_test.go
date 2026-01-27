package services

import (
	"law_flow_app_go/models"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAppointmentICS(t *testing.T) {
	// Setup test data
	now := time.Now().UTC()
	startTime := now.Add(1 * time.Hour)
	endTime := now.Add(2 * time.Hour)
	notes := "Test appointment notes"

	apt := &models.Appointment{
		ID:        "test-apt-id",
		StartTime: startTime,
		EndTime:   endTime,
		Notes:     &notes,
		LawyerID:  "lawyer-id",
		AppointmentType: &models.AppointmentType{
			Name: "Initial Consultation",
		},
	}

	firmName := "LexLegal Cloud"
	firmEmail := "support@lexlegal.com"
	firmTimezone := "UTC"

	// Execute
	icsBytes, err := GenerateAppointmentICS(apt, firmName, firmEmail, firmTimezone)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, icsBytes)

	icsContent := string(icsBytes)

	// Basic structure checks
	assert.True(t, strings.HasPrefix(icsContent, "BEGIN:VCALENDAR"), "Should start with BEGIN:VCALENDAR")
	assert.True(t, strings.HasSuffix(icsContent, "END:VCALENDAR"), "Should end with END:VCALENDAR")
	assert.Contains(t, icsContent, "BEGIN:VEVENT")
	assert.Contains(t, icsContent, "END:VEVENT")

	// UID check
	assert.Contains(t, icsContent, "UID:test-apt-id")

	// Time checks (ICS format)
	dateFormat := "20060102T150405Z"
	assert.Contains(t, icsContent, "DTSTART:"+startTime.Format(dateFormat))
	assert.Contains(t, icsContent, "DTEND:"+endTime.Format(dateFormat))

	// Summary and Description
	assert.Contains(t, icsContent, "SUMMARY:Initial Consultation: LexLegal Cloud")
	assert.Contains(t, icsContent, "DESCRIPTION:Appointment with lawyer-id at LexLegal Cloud.\n\nNotes: Test appointment notes")

	// Organizer
	assert.Contains(t, icsContent, "ORGANIZER;CN=\"LexLegal Cloud\":mailto:support@lexlegal.com")
}

func TestGenerateAppointmentICS_NoNotes(t *testing.T) {
	// Setup test data (minimal)
	startTime := time.Now().UTC()
	endTime := startTime.Add(30 * time.Minute)

	apt := &models.Appointment{
		ID:        "simple-apt",
		StartTime: startTime,
		EndTime:   endTime,
		LawyerID:  "L1",
	}

	// Execute
	icsBytes, err := GenerateAppointmentICS(apt, "Firm X", "x@firm.com", "UTC")

	// Assert
	assert.NoError(t, err)
	icsContent := string(icsBytes)

	assert.Contains(t, icsContent, "SUMMARY:Appointment: Firm X")
	assert.Contains(t, icsContent, "DESCRIPTION:Appointment with L1 at Firm X.")
	assert.NotContains(t, icsContent, "Notes:")
}
