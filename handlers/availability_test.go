package handlers

import (
	"law_flow_app_go/models"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAvailabilityPageHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-avail", Name: "Availability Firm"}
	database.Create(firm)
	user := &models.User{ID: "user-avail", Name: "Availability User", Email: "avail@test.com", FirmID: stringToPtr(firm.ID), Role: "lawyer"}
	database.Create(user)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/availability", nil)
		c.Set("user", user)
		c.Set("firm", firm)

		err := AvailabilityPageHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestGetAvailabilityHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-get-avail", Name: "Get Availability Firm"}
	database.Create(firm)
	lawyer := &models.User{ID: "lawyer-get-avail", Name: "Lawyer", Email: "lawyer-get-avail@test.com", FirmID: stringToPtr(firm.ID), Role: "lawyer"}
	database.Create(lawyer)

	// Create some availability slots
	database.Create(&models.Availability{
		ID: "avail-1", LawyerID: lawyer.ID, DayOfWeek: 1,
		StartTime: "09:00", EndTime: "12:00", IsActive: true,
	})
	database.Create(&models.Availability{
		ID: "avail-2", LawyerID: lawyer.ID, DayOfWeek: 2,
		StartTime: "14:00", EndTime: "17:00", IsActive: true,
	})

	t.Run("Success", func(t *testing.T) {
		_, c, _ := setupEcho(http.MethodGet, "/api/availability", nil)
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := GetAvailabilityHandler(c)
		assert.NoError(t, err)
	})
}

func TestCreateAvailabilityHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-create-avail", Name: "Create Availability Firm"}
	database.Create(firm)
	lawyer := &models.User{ID: "lawyer-create-avail", Name: "Lawyer", Email: "lawyer-create-avail@test.com", FirmID: stringToPtr(firm.ID), Role: "lawyer"}
	database.Create(lawyer)

	t.Run("Success", func(t *testing.T) {
		f := url.Values{}
		f.Add("day_of_week", "1")
		f.Add("start_time", "09:00")
		f.Add("end_time", "12:00")

		_, c, rec := setupEcho(http.MethodPost, "/api/availability", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := CreateAvailabilityHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		// Verify availability was created
		var avail models.Availability
		database.Where("lawyer_id = ? AND day_of_week = ?", lawyer.ID, 1).First(&avail)
		assert.Equal(t, "09:00", avail.StartTime)
		assert.Equal(t, "12:00", avail.EndTime)
	})

	t.Run("Missing required fields", func(t *testing.T) {
		f := url.Values{}
		f.Add("day_of_week", "")
		f.Add("start_time", "")
		f.Add("end_time", "")

		_, c, _ := setupEcho(http.MethodPost, "/api/availability", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := CreateAvailabilityHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})

	t.Run("Invalid day of week", func(t *testing.T) {
		f := url.Values{}
		f.Add("day_of_week", "8")
		f.Add("start_time", "09:00")
		f.Add("end_time", "12:00")

		_, c, _ := setupEcho(http.MethodPost, "/api/availability", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := CreateAvailabilityHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})

	t.Run("Invalid time format", func(t *testing.T) {
		f := url.Values{}
		f.Add("day_of_week", "1")
		f.Add("start_time", "invalid")
		f.Add("end_time", "12:00")

		_, c, _ := setupEcho(http.MethodPost, "/api/availability", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := CreateAvailabilityHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})

	t.Run("End time before start time", func(t *testing.T) {
		f := url.Values{}
		f.Add("day_of_week", "1")
		f.Add("start_time", "12:00")
		f.Add("end_time", "09:00")

		_, c, _ := setupEcho(http.MethodPost, "/api/availability", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := CreateAvailabilityHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})
}

func TestUpdateAvailabilityHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-update-avail", Name: "Update Availability Firm"}
	database.Create(firm)
	lawyer := &models.User{ID: "lawyer-update-avail", Name: "Lawyer", Email: "lawyer-update-avail@test.com", FirmID: stringToPtr(firm.ID), Role: "lawyer"}
	database.Create(lawyer)

	availability := &models.Availability{
		ID: "avail-update", LawyerID: lawyer.ID, DayOfWeek: 1,
		StartTime: "09:00", EndTime: "12:00", IsActive: true,
	}
	database.Create(availability)

	t.Run("Success", func(t *testing.T) {
		f := url.Values{}
		f.Add("start_time", "10:00")
		f.Add("end_time", "14:00")
		f.Add("is_active", "true")

		_, c, rec := setupEcho(http.MethodPost, "/api/availability/avail-update", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.SetParamNames("id")
		c.SetParamValues("avail-update")
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := UpdateAvailabilityHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify availability was updated
		var updatedAvail models.Availability
		database.First(&updatedAvail, "id = ?", "avail-update")
		assert.Equal(t, "10:00", updatedAvail.StartTime)
		assert.Equal(t, "14:00", updatedAvail.EndTime)
	})

	t.Run("Availability not found", func(t *testing.T) {
		f := url.Values{}
		f.Add("start_time", "10:00")
		f.Add("end_time", "14:00")

		_, c, _ := setupEcho(http.MethodPost, "/api/availability/nonexistent", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.SetParamNames("id")
		c.SetParamValues("nonexistent")
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := UpdateAvailabilityHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusNotFound, err.(*echo.HTTPError).Code)
	})
}

func TestDeleteAvailabilityHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-delete-avail", Name: "Delete Availability Firm"}
	database.Create(firm)
	lawyer := &models.User{ID: "lawyer-delete-avail", Name: "Lawyer", Email: "lawyer-delete-avail@test.com", FirmID: stringToPtr(firm.ID), Role: "lawyer"}
	database.Create(lawyer)

	availability := &models.Availability{
		ID: "avail-delete", LawyerID: lawyer.ID, DayOfWeek: 1,
		StartTime: "09:00", EndTime: "12:00", IsActive: true,
	}
	database.Create(availability)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodDelete, "/api/availability/avail-delete", nil)
		c.SetParamNames("id")
		c.SetParamValues("avail-delete")
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := DeleteAvailabilityHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)

		// Verify availability was deleted
		var deletedAvail models.Availability
		err = database.First(&deletedAvail, "id = ?", "avail-delete").Error
		assert.Error(t, err)
	})

	t.Run("Availability not found", func(t *testing.T) {
		_, c, _ := setupEcho(http.MethodDelete, "/api/availability/nonexistent", nil)
		c.SetParamNames("id")
		c.SetParamValues("nonexistent")
		c.Set("user", lawyer)
		c.Set("firm", firm)

		err := DeleteAvailabilityHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusNotFound, err.(*echo.HTTPError).Code)
	})
}
