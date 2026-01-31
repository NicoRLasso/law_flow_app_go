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

func TestFirmSettingsPageHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-settings", Name: "Settings Firm"}
	database.Create(firm)
	user := &models.User{ID: "user-settings", Name: "Settings User", Email: "settings@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(user)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/firm/settings", nil)
		c.Set("user", user)
		c.Set("firm", firm)

		err := FirmSettingsPageHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestUpdateFirmSettingsHandler(t *testing.T) {
	database := setupTestDB(t)
	country := &models.Country{ID: "country-update", Name: "Test Country", Code: "TST"}
	database.Create(country)
	firm := &models.Firm{ID: "firm-update-settings", Name: "Update Settings Firm", CountryID: country.ID}
	database.Create(firm)
	user := &models.User{ID: "user-update-settings", Name: "Update Settings User", Email: "update-settings@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(user)

	t.Run("Success", func(t *testing.T) {
		f := url.Values{}
		f.Add("update_type", "general")
		f.Add("name", "Updated Firm Name")
		f.Add("country_id", country.ID)
		f.Add("address", "123 Updated St")
		f.Add("phone", "1234567890")
		f.Add("billing_email", "updated@firm.com")

		_, c, rec := setupEcho(http.MethodPost, "/firm/settings", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)
		c.Set("firm", firm)

		err := UpdateFirmHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify firm was updated
		var updatedFirm models.Firm
		database.First(&updatedFirm, "id = ?", firm.ID)
		assert.Equal(t, "Updated Firm Name", updatedFirm.Name)
	})

	t.Run("Name too long", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", strings.Repeat("a", 256))

		_, c, _ := setupEcho(http.MethodPost, "/firm/settings", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)
		c.Set("firm", firm)

		err := UpdateFirmHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})

	t.Run("Address too long", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "Test Firm")
		f.Add("address", strings.Repeat("a", 256))

		_, c, _ := setupEcho(http.MethodPost, "/firm/settings", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)
		c.Set("firm", firm)

		err := UpdateFirmHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})

	t.Run("Phone too long", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "Test Firm")
		f.Add("phone", strings.Repeat("1", 21))

		_, c, _ := setupEcho(http.MethodPost, "/firm/settings", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)
		c.Set("firm", firm)

		err := UpdateFirmHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})

	t.Run("Email too long", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "Test Firm")
		f.Add("email", strings.Repeat("a", 321)+"@test.com")

		_, c, _ := setupEcho(http.MethodPost, "/firm/settings", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)
		c.Set("firm", firm)

		err := UpdateFirmHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, err.(*echo.HTTPError).Code)
	})
}

func TestFirmSetupPageHandler(t *testing.T) {
	database := setupTestDB(t)
	user := &models.User{ID: "user-setup", Name: "Setup User", Email: "setup@test.com", FirmID: nil, Role: "admin"}
	database.Create(user)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/firm/setup", nil)
		c.Set("user", user)

		err := FirmSetupHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestCreateFirmHandler(t *testing.T) {
	database := setupTestDB(t)
	country := &models.Country{ID: "country-create", Name: "Test Country", Code: "TST"}
	database.Create(country)
	user := &models.User{ID: "user-create-firm", Name: "Create Firm User", Email: "create-firm@test.com", FirmID: nil, Role: "admin"}
	database.Create(user)

	t.Run("Success", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "New Firm")
		f.Add("country_id", country.ID)
		f.Add("address", "123 New St")
		f.Add("phone", "1234567890")
		f.Add("billing_email", "new@firm.com")

		_, c, rec := setupEcho(http.MethodPost, "/firm/setup", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := FirmSetupPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/dashboard", rec.Header().Get("Location"))

		// Verify firm was created
		var firm models.Firm
		database.Where("name = ?", "New Firm").First(&firm)
		assert.Equal(t, "New Firm", firm.Name)
	})

	t.Run("Missing required fields", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "")

		_, c, rec := setupEcho(http.MethodPost, "/firm/setup", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := FirmSetupPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
	})

	t.Run("Name too long", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", strings.Repeat("a", 256))

		_, c, rec := setupEcho(http.MethodPost, "/firm/setup", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := FirmSetupPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
	})
}
