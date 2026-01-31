package handlers

import (
	"law_flow_app_go/models"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAppointmentsPageHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-appt", Name: "Appointment Firm"}
	database.Create(firm)
	user := &models.User{ID: "user-appt", Name: "Appointment User", Email: "appt@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(user)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/appointments", nil)
		c.Set("user", user)
		c.Set("firm", firm)

		err := AppointmentsPageHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestGetAppointmentsHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-get-appt", Name: "Get Appointment Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-appt", Name: "Admin", Email: "admin-appt@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)
	lawyer := &models.User{ID: "lawyer-appt", Name: "Lawyer", Email: "lawyer-appt@test.com", FirmID: stringToPtr(firm.ID), Role: "lawyer"}
	database.Create(lawyer)

	// Create some appointments
	now := time.Now()
	database.Create(&models.Appointment{
		ID: "appt-1", FirmID: firm.ID, LawyerID: lawyer.ID, ClientID: stringToPtr("client-1"),
		ClientName: "Client 1", ClientEmail: "client1@test.com", StartTime: now, EndTime: now.Add(1 * time.Hour), Status: "SCHEDULED",
	})
	database.Create(&models.Appointment{
		ID: "appt-2", FirmID: firm.ID, LawyerID: lawyer.ID, ClientID: stringToPtr("client-2"),
		ClientName: "Client 2", ClientEmail: "client2@test.com", StartTime: now.Add(2 * time.Hour), EndTime: now.Add(3 * time.Hour), Status: "SCHEDULED",
	})

	t.Run("Admin sees all firm appointments", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/appointments", nil)
		c.Set("user", admin)

		err := GetAppointmentsHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Lawyer sees their own appointments", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/appointments", nil)
		c.Set("user", lawyer)

		err := GetAppointmentsHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Unauthorized user", func(t *testing.T) {
		_, c, _ := setupEcho(http.MethodGet, "/api/appointments", nil)

		err := GetAppointmentsHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, err.(*echo.HTTPError).Code)
	})

	t.Run("Invalid start date format", func(t *testing.T) {
		_, c, _ := setupEcho(http.MethodGet, "/api/appointments?start=invalid-date", nil)
		c.Set("user", admin)

		err := GetAppointmentsHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})

	t.Run("Invalid end date format", func(t *testing.T) {
		_, c, _ := setupEcho(http.MethodGet, "/api/appointments?end=invalid-date", nil)
		c.Set("user", admin)

		err := GetAppointmentsHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})

	t.Run("Valid date range", func(t *testing.T) {
		startDate := time.Now().Format("2006-01-02")
		endDate := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
		_, c, rec := setupEcho(http.MethodGet, "/api/appointments?start="+startDate+"&end="+endDate, nil)
		c.Set("user", admin)

		err := GetAppointmentsHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Pagination", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/appointments?page=1&limit=5", nil)
		c.Set("user", admin)

		err := GetAppointmentsHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("HTMX request", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/appointments", nil)
		c.Request().Header.Set("HX-Request", "true")
		c.Set("user", admin)

		err := GetAppointmentsHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestCreateAppointmentHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-create-appt", Name: "Create Appointment Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-create-appt", Name: "Admin", Email: "admin-create-appt@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)
	client := &models.User{ID: "client-create-appt", Name: "Client", Email: "client-create-appt@test.com", FirmID: stringToPtr(firm.ID), Role: "client"}
	database.Create(client)

	// Create availability for the lawyer (all days to handle timezone shifts)
	for i := 0; i < 7; i++ {
		database.Create(&models.Availability{
			LawyerID:  admin.ID,
			DayOfWeek: i,
			StartTime: "00:00",
			EndTime:   "23:59",
			IsActive:  true,
		})
	}

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		f := url.Values{}
		f.Add("lawyer_id", admin.ID)
		f.Add("client_id", client.ID) // Also need client_id as per handler logic (or case_id)
		f.Add("start_time", now.Format(time.RFC3339))
		f.Add("end_time", now.Add(1*time.Hour).Format(time.RFC3339))
		f.Add("notes", "Test appointment")

		_, c, rec := setupEcho(http.MethodPost, "/api/appointments", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CreateAppointmentHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("Missing required fields", func(t *testing.T) {
		f := url.Values{}
		f.Add("client_name", "")
		f.Add("client_email", "")

		_, c, _ := setupEcho(http.MethodPost, "/api/appointments", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CreateAppointmentHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})

	t.Run("Invalid date format", func(t *testing.T) {
		f := url.Values{}
		f.Add("client_name", "New Client")
		f.Add("client_email", "newclient@test.com")
		f.Add("start_time", "invalid-date")
		f.Add("end_time", "invalid-date")

		_, c, _ := setupEcho(http.MethodPost, "/api/appointments", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CreateAppointmentHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})
}

func TestUpdateAppointmentStatusHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-update-appt", Name: "Update Appointment Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-update-appt", Name: "Admin", Email: "admin-update-appt@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)
	client := &models.User{ID: "client-update-appt", Name: "Client", Email: "client-update-appt@test.com", FirmID: stringToPtr(firm.ID), Role: "client"}
	database.Create(client)

	now := time.Now()
	appointment := &models.Appointment{
		ID: "appt-update", FirmID: firm.ID, LawyerID: admin.ID, ClientID: stringToPtr(client.ID),
		ClientName: "Original Client", ClientEmail: "original@test.com", StartTime: now, EndTime: now.Add(1 * time.Hour), Status: "SCHEDULED",
	}
	database.Create(appointment)

	t.Run("Success", func(t *testing.T) {
		f := url.Values{}
		f.Add("status", "COMPLETED")

		_, c, rec := setupEcho(http.MethodPost, "/api/appointments/appt-update/status", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.SetParamNames("id")
		c.SetParamValues("appt-update")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := UpdateAppointmentStatusHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify appointment was updated
		var updatedAppt models.Appointment
		database.First(&updatedAppt, "id = ?", "appt-update")
		assert.Equal(t, "COMPLETED", updatedAppt.Status)
	})

	t.Run("Appointment not found", func(t *testing.T) {
		f := url.Values{}
		f.Add("status", "COMPLETED")

		_, c, _ := setupEcho(http.MethodPost, "/api/appointments/nonexistent/status", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.SetParamNames("id")
		c.SetParamValues("nonexistent")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := UpdateAppointmentStatusHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusNotFound, err.(*echo.HTTPError).Code)
	})
}

func TestCancelAppointmentHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-delete-appt", Name: "Delete Appointment Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-delete-appt", Name: "Admin", Email: "admin-delete-appt@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)
	client := &models.User{ID: "client-delete-appt", Name: "Client", Email: "client-delete-appt@test.com", FirmID: stringToPtr(firm.ID), Role: "client"}
	database.Create(client)

	now := time.Now()
	appointment := &models.Appointment{
		ID: "appt-delete", FirmID: firm.ID, LawyerID: admin.ID, ClientID: stringToPtr(client.ID),
		ClientName: "Delete Client", ClientEmail: "delete@test.com", StartTime: now, EndTime: now.Add(1 * time.Hour), Status: "SCHEDULED",
	}
	database.Create(appointment)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodPost, "/api/appointments/appt-delete/cancel", nil)
		c.SetParamNames("id")
		c.SetParamValues("appt-delete")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CancelAppointmentHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify appointment was cancelled
		var cancelledAppt models.Appointment
		database.First(&cancelledAppt, "id = ?", "appt-delete")
		assert.Equal(t, "CANCELLED", cancelledAppt.Status)
	})

	t.Run("Appointment not found", func(t *testing.T) {
		_, c, _ := setupEcho(http.MethodPost, "/api/appointments/nonexistent/cancel", nil)
		c.SetParamNames("id")
		c.SetParamValues("nonexistent")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CancelAppointmentHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusNotFound, err.(*echo.HTTPError).Code)
	})
}
