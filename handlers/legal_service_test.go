package handlers

import (
	"encoding/json"
	"law_flow_app_go/models"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServicesPageHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-s1", Name: "Service Firm"}
	database.Create(firm)

	admin := &models.User{ID: "admin-s1", Name: "Admin", Email: "admin-s1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/services", nil)
		c.Set("user", admin)
		c.Set("firm", firm)

		err := ServicesPageHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestGetServicesHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-s2", Name: "Service Firm 2"}
	database.Create(firm)

	admin := &models.User{ID: "admin-s2", Name: "Admin 2", Email: "admin-s2@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	client := &models.User{ID: "client-s2", Name: "Client 2", Email: "client-s2@test.com", FirmID: stringToPtr(firm.ID), Role: "client"}
	database.Create(client)

	// Create some services
	database.Create(&models.LegalService{
		ID:            "service-s2-1",
		FirmID:        firm.ID,
		ServiceNumber: "SVC-2026-00001",
		Title:         "Service 1",
		ClientID:      client.ID,
		Status:        models.ServiceStatusIntake,
		CreatedAt:     time.Now(),
	})

	t.Run("Admin sees all", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/services", nil)
		c.Set("user", admin)
		c.Set("firm", firm)

		err := GetServicesHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 1)
	})

	t.Run("Client sees own", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/services", nil)
		c.Set("user", client)
		c.Set("firm", firm)

		err := GetServicesHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 1)
	})
}

func TestCreateServiceHandler(t *testing.T) {
	database := setupTestDB(t)
	country := &models.Country{ID: "country-1", Name: "Test Country", Code: "TST"}
	database.Create(country)
	firm := &models.Firm{ID: "firm-s3", Name: "Service Firm 3", CountryID: country.ID, Slug: "firm-s3"}
	database.Create(firm)

	plan := &models.Plan{ID: "plan-pro", Name: "Pro Plan", Tier: "professional", MaxCases: 10}
	database.Create(plan)
	database.Create(&models.FirmSubscription{FirmID: firm.ID, PlanID: plan.ID, Status: "active"})
	database.Create(&models.FirmUsage{FirmID: firm.ID, CurrentCases: 0})

	admin := &models.User{ID: "admin-s3", Name: "Admin 3", Email: "admin3-s@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	client := &models.User{ID: "client-s3", Name: "Client 3", Email: "client3-s@test.com", FirmID: stringToPtr(firm.ID), Role: "client"}
	database.Create(client)

	cat := &models.ChoiceCategory{ID: "cat-1", FirmID: firm.ID, Key: models.ChoiceCategoryKeyServiceType, Name: "Service Types", Country: country.Name}
	database.Create(cat)
	stype := &models.ChoiceOption{ID: "stype-1", CategoryID: cat.ID, Code: "CONTRACT", Label: "Contract Review", IsActive: true}
	database.Create(stype)

	t.Run("Success", func(t *testing.T) {
		f := url.Values{}
		f.Add("title", "New Service")
		f.Add("client_id", client.ID)
		f.Add("service_type_id", stype.ID)
		f.Add("description", "Description")
		f.Add("objective", "Objective")
		f.Add("assigned_to_id", admin.ID)

		_, c, rec := setupEcho(http.MethodPost, "/api/services", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CreateServiceHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
	})
}

func TestUpdateServiceHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-s4", Name: "Service Firm 4"}
	database.Create(firm)

	admin := &models.User{ID: "admin-s4", Name: "Admin 4", Email: "admin4-s@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	service := &models.LegalService{
		ID:            "service-s4",
		FirmID:        firm.ID,
		ServiceNumber: "SVC-2026-00004",
		Title:         "Original Title",
		ClientID:      "client-1",
		CreatedAt:     time.Now(),
	}
	database.Create(service)

	t.Run("Success", func(t *testing.T) {
		f := url.Values{}
		f.Add("title", "Updated Title")
		f.Add("description", "Updated Description")

		_, c, rec := setupEcho(http.MethodPut, "/api/services/service-s4", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.SetParamNames("id")
		c.SetParamValues("service-s4")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := UpdateServiceHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		database.First(service, "id = ?", "service-s4")
		assert.Equal(t, "Updated Title", service.Title)
	})
}

func TestGetServiceDetailHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-sd1", Name: "Detail Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-sd1", Name: "Admin", Email: "admin-sd1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)
	client := &models.User{ID: "client-sd1", Name: "Client", Email: "client-sd1@test.com", FirmID: stringToPtr(firm.ID), Role: "client"}
	database.Create(client)

	service := &models.LegalService{
		ID:            "service-sd1",
		FirmID:        firm.ID,
		ServiceNumber: "SVC-2026-00005",
		Title:         "Detail Title",
		ClientID:      client.ID,
		CreatedAt:     time.Now(),
	}
	database.Create(service)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/services/service-sd1", nil)
		c.SetParamNames("id")
		c.SetParamValues("service-sd1")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := GetServiceDetailHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Detail Title")
	})
}

func TestGetServiceTimelineHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-st1", Name: "Timeline Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-st1", Name: "Admin", Email: "admin-st1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	service := &models.LegalService{
		ID:            "service-st1",
		FirmID:        firm.ID,
		ServiceNumber: "SVC-2026-00006",
		Title:         "Timeline Title",
		ClientID:      "client-1",
		CreatedAt:     time.Now(),
	}
	database.Create(service)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/services/service-st1/timeline", nil)
		c.SetParamNames("id")
		c.SetParamValues("service-st1")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := GetServiceTimelineHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestDeleteServiceHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-ds1", Name: "Delete Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-ds1", Name: "Admin", Email: "admin-ds1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	service := &models.LegalService{
		ID:            "service-ds1",
		FirmID:        firm.ID,
		ServiceNumber: "SVC-2026-00007",
		Title:         "Delete Title",
		ClientID:      "client-1",
		CreatedAt:     time.Now(),
	}
	database.Create(service)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodDelete, "/api/services/service-ds1", nil)
		c.SetParamNames("id")
		c.SetParamValues("service-ds1")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := DeleteServiceHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)

		var deleted models.LegalService
		err = database.First(&deleted, "id = ?", "service-ds1").Error
		assert.Error(t, err) // Should be record not found
	})
}

func TestUpdateServiceStatusHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-su1", Name: "Status Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-su1", Name: "Admin", Email: "admin-su1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	service := &models.LegalService{
		ID:            "service-su1",
		FirmID:        firm.ID,
		ServiceNumber: "SVC-2026-00008",
		Title:         "Status Title",
		ClientID:      "client-1",
		Status:        models.ServiceStatusIntake,
		CreatedAt:     time.Now(),
	}
	database.Create(service)

	t.Run("Success", func(t *testing.T) {
		f := url.Values{}
		f.Add("status", models.ServiceStatusInProgress)

		_, c, rec := setupEcho(http.MethodPost, "/api/services/service-su1/status", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.SetParamNames("id")
		c.SetParamValues("service-su1")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := UpdateServiceStatusHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		database.First(service, "id = ?", "service-su1")
		assert.Equal(t, models.ServiceStatusInProgress, service.Status)
	})
}

func TestCreateServiceModalHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-sm1", Name: "Modal Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-sm1", Name: "Admin", Email: "admin-sm1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/services/modal", nil)
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CreateServiceModalHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestGetUpdateServiceModalHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-um1", Name: "Update Modal Firm"}
	database.Create(firm)
	admin := &models.User{ID: "admin-um1", Name: "Admin", Email: "admin-um1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	service := &models.LegalService{
		ID:            "service-um1",
		FirmID:        firm.ID,
		ServiceNumber: "SVC-2026-00009",
		Title:         "Update Modal Title",
		ClientID:      "client-1",
		CreatedAt:     time.Now(),
	}
	database.Create(service)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/services/service-um1/edit", nil)
		c.SetParamNames("id")
		c.SetParamValues("service-um1")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := GetUpdateServiceModalHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
