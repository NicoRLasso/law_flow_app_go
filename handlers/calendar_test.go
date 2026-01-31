package handlers

import (
	"law_flow_app_go/models"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestCalendarPageHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-calendar", Name: "Calendar Firm"}
	database.Create(firm)
	user := &models.User{ID: "user-calendar", Name: "Calendar User", Email: "calendar@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(user)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/calendar", nil)
		c.Set("user", user)
		c.Set("firm", firm)

		err := CalendarPageHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestGetCalendarEventsHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-cal-events", Name: "Calendar Events Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-cal-events", Name: "Admin", Email: "admin-cal-events@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)
	lawyer := &models.User{ID: "lawyer-cal-events", Name: "Lawyer", Email: "lawyer-cal-events@test.com", FirmID: stringToPtr(firm.ID), Role: "lawyer"}
	database.Create(lawyer)

	// Create some appointments
	now := time.Now()
	database.Create(&models.Appointment{
		ID: "appt-cal-1", FirmID: firm.ID, LawyerID: lawyer.ID, ClientID: stringToPtr("client-1"),
		ClientName: "Client 1", ClientEmail: "client1@test.com", StartTime: now, EndTime: now.Add(1 * time.Hour), Status: "SCHEDULED",
	})
	database.Create(&models.Appointment{
		ID: "appt-cal-2", FirmID: firm.ID, LawyerID: lawyer.ID, ClientID: stringToPtr("client-2"),
		ClientName: "Client 2", ClientEmail: "client2@test.com", StartTime: now.Add(2 * time.Hour), EndTime: now.Add(3 * time.Hour), Status: "SCHEDULED",
	})

	t.Run("Success", func(t *testing.T) {
		startDate := time.Now().Format("2006-01-02")
		endDate := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
		_, c, rec := setupEcho(http.MethodGet, "/api/calendar/events?start="+startDate+"&end="+endDate, nil)
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CalendarEventsHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Missing date parameters", func(t *testing.T) {
		_, c, _ := setupEcho(http.MethodGet, "/api/calendar/events", nil)
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CalendarEventsHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})

	t.Run("Invalid date format", func(t *testing.T) {
		_, c, _ := setupEcho(http.MethodGet, "/api/calendar/events?start=invalid&end=invalid", nil)
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CalendarEventsHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})
}
