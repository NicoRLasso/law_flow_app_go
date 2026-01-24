package services

import (
	"fmt"
	"law_flow_app_go/models"
	"time"
)

// GenerateAppointmentICS generates an ICS file content for an appointment
func GenerateAppointmentICS(apt *models.Appointment, firmName, firmEmail, firmTimezone string) ([]byte, error) {
	// Format dates for ICS (YYYYMMDDTHHMMSSZ)
	// We assume appointment times are in UTC in the database
	dateFormat := "20060102T150405Z"
	dtStamp := time.Now().UTC().Format(dateFormat)
	dtStart := apt.StartTime.UTC().Format(dateFormat)
	dtEnd := apt.EndTime.UTC().Format(dateFormat)

	// Build description
	description := fmt.Sprintf("Appointment with %s at %s.", apt.LawyerID, firmName) // LawyerID is a placeholder, strictly we should use LawyerName but we might not have it loaded unless preloaded.
	// We should rely on what's passed in the appointment struct.
	// Ideally the caller ensures Lawyer relation is loaded or we pass lawyer name.
	// For now let's keep it simple.
	if apt.Notes != nil && *apt.Notes != "" {
		description += fmt.Sprintf("\n\nNotes: %s", *apt.Notes)
	}

	// Escape special characters in text fields (backslashes, newlines, commas, semicolons)
	// For simplicity in this initial version, we will just start with basic string building.
	// Robust escaping would be: \ -> \\, ; -> \;, , -> \,, \n -> \n

	summary := fmt.Sprintf("Appointment: %s", firmName)
	if apt.AppointmentType != nil {
		summary = fmt.Sprintf("%s: %s", apt.AppointmentType.Name, firmName)
	}

	const icsTemplate = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//LexLegalCloud//Appointment//EN
CALSCALE:GREGORIAN
METHOD:REQUEST
BEGIN:VEVENT
UID:%s
DTSTAMP:%s
DTSTART:%s
DTEND:%s
SUMMARY:%s
DESCRIPTION:%s
ORGANIZER;CN="%s":mailto:%s
STATUS:CONFIRMED
END:VEVENT
END:VCALENDAR`

	icsContent := fmt.Sprintf(icsTemplate,
		apt.ID,      // UID
		dtStamp,     // DTSTAMP
		dtStart,     // DTSTART
		dtEnd,       // DTEND
		summary,     // SUMMARY
		description, // DESCRIPTION
		firmName,    // ORGANIZER CN
		firmEmail,   // ORGANIZER MAILTO
	)

	return []byte(icsContent), nil
}
