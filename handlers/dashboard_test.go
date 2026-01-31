package handlers

import (
	"law_flow_app_go/models"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDashboardHandler(t *testing.T) {
	database := setupTestDB(t)

	// Setup data
	firm := &models.Firm{ID: "firm-dash", Name: "Dash Firm"}
	database.Create(firm)

	admin := &models.User{
		ID:     "admin-dash",
		Name:   "Admin Dash",
		Email:  "admin@dash.com",
		Role:   "admin",
		FirmID: stringToPtr(firm.ID),
	}
	database.Create(admin)

	client := &models.User{
		ID:     "client-dash",
		Name:   "Client Dash",
		Email:  "client@dash.com",
		Role:   "client",
		FirmID: stringToPtr(firm.ID),
	}
	database.Create(client)

	lawyer := &models.User{
		ID:     "lawyer-dash",
		Name:   "Lawyer Dash",
		Email:  "lawyer@dash.com",
		Role:   "lawyer",
		FirmID: stringToPtr(firm.ID),
	}
	database.Create(lawyer)

	// Create a notification
	database.Create(&models.Notification{
		FirmID:  firm.ID,
		UserID:  stringToPtr(admin.ID),
		Type:    models.NotificationTypeSystem,
		Title:   "New Alert",
		Message: "Something happened",
	})

	// Create some cases
	database.Create(&models.Case{
		ID:           "case-1",
		FirmID:       firm.ID,
		CaseNumber:   "CASE-001",
		Status:       models.CaseStatusOpen,
		ClientID:     client.ID,
		AssignedToID: stringToPtr(lawyer.ID),
		OpenedAt:     time.Now(),
	})

	database.Create(&models.Case{
		ID:              "case-2",
		FirmID:          firm.ID,
		CaseNumber:      "CASE-002",
		Status:          models.CaseStatusClosed,
		ClientID:        client.ID,
		OpenedAt:        time.Now().AddDate(0, -1, 0),
		StatusChangedAt: func() *time.Time { t := time.Now(); return &t }(),
	})

	// Create an appointment for lawyer
	database.Create(&models.Appointment{
		ID:          "appt-1",
		FirmID:      firm.ID,
		LawyerID:    lawyer.ID,
		ClientID:    stringToPtr(client.ID),
		ClientName:  client.Name,
		ClientEmail: client.Email,
		StartTime:   time.Now().Add(1 * time.Hour),
		EndTime:     time.Now().Add(2 * time.Hour),
		Status:      models.AppointmentStatusScheduled,
	})

	t.Run("Admin Dashboard", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/dashboard", nil)
		c.Set("user", admin)
		c.Set("firm", firm)

		err := DashboardHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Dashboard")
	})

	t.Run("Lawyer Dashboard", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/dashboard", nil)
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := DashboardHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Client Dashboard", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/dashboard", nil)
		c.Set("user", client)
		c.Set("firm", firm)

		err := DashboardHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
