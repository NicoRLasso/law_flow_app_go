package handlers

import (
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestProfileSettingsPageHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-profile", Name: "Profile Firm"}
	database.Create(firm)
	user := &models.User{ID: "user-profile", Name: "Profile User", Email: "profile@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(user)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/profile/settings", nil)
		c.Set("user", user)
		c.Set("firm", firm)

		err := ProfileSettingsPageHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestUpdateProfileHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-update", Name: "Update Firm"}
	database.Create(firm)
	user := &models.User{ID: "user-update", Name: "Update User", Email: "update@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(user)

	t.Run("Success", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "Updated Name")
		f.Add("email", "updated@test.com")
		f.Add("language", "en")
		f.Add("phone_number", "1234567890")
		f.Add("address", "123 Test St")

		_, c, rec := setupEcho(http.MethodPost, "/profile/update", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := UpdateProfileHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify user was updated
		var updatedUser models.User
		database.First(&updatedUser, "id = ?", user.ID)
		assert.Equal(t, "Updated Name", updatedUser.Name)
		assert.Equal(t, "updated@test.com", updatedUser.Email)
		assert.Equal(t, "en", updatedUser.Language)
	})

	t.Run("Missing required fields", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "")
		f.Add("email", "")
		f.Add("language", "")

		_, c, _ := setupEcho(http.MethodPost, "/profile/update", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := UpdateProfileHandler(c)
		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, he.Code)
	})

	t.Run("Name too long", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", strings.Repeat("a", 256))
		f.Add("email", "test@test.com")
		f.Add("language", "en")

		_, c, _ := setupEcho(http.MethodPost, "/profile/update", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := UpdateProfileHandler(c)
		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, he.Code)
	})

	t.Run("Email too long", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "Test")
		f.Add("email", strings.Repeat("a", 321)+"@test.com")
		f.Add("language", "en")

		_, c, _ := setupEcho(http.MethodPost, "/profile/update", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := UpdateProfileHandler(c)
		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, he.Code)
	})

	t.Run("Phone number too long", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "Test")
		f.Add("email", "test@test.com")
		f.Add("language", "en")
		f.Add("phone_number", strings.Repeat("1", 21))

		_, c, _ := setupEcho(http.MethodPost, "/profile/update", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := UpdateProfileHandler(c)
		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, he.Code)
	})

	t.Run("Address too long", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "Test")
		f.Add("email", "test@test.com")
		f.Add("language", "en")
		f.Add("address", strings.Repeat("a", 256))

		_, c, _ := setupEcho(http.MethodPost, "/profile/update", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := UpdateProfileHandler(c)
		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, he.Code)
	})

	t.Run("Document number too long", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "Test")
		f.Add("email", "test@test.com")
		f.Add("language", "en")
		f.Add("document_number", strings.Repeat("1", 51))

		_, c, _ := setupEcho(http.MethodPost, "/profile/update", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := UpdateProfileHandler(c)
		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, he.Code)
	})

	t.Run("HTMX request error", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "")
		f.Add("email", "")
		f.Add("language", "")

		_, c, rec := setupEcho(http.MethodPost, "/profile/update", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Request().Header.Set("HX-Request", "true")
		c.Set("user", user)

		err := UpdateProfileHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Name, email, and language are required")
	})
}

func TestChangePasswordHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-password", Name: "Password Firm"}
	database.Create(firm)
	password, _ := services.HashPassword("oldpassword123")

	t.Run("Success", func(t *testing.T) {
		user := &models.User{ID: "user-pwd-success", Name: "Password User", Email: "pwd-success@test.com", Password: password, FirmID: stringToPtr(firm.ID), Role: "admin"}
		database.Create(user)

		f := url.Values{}
		f.Add("current_password", "oldpassword123")
		f.Add("new_password", "NewPassword123!")
		f.Add("confirm_password", "NewPassword123!")

		_, c, rec := setupEcho(http.MethodPost, "/profile/change-password", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := ChangePasswordHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify password was changed
		var updatedUser models.User
		database.First(&updatedUser, "id = ?", user.ID)
		assert.NotEqual(t, password, updatedUser.Password)
	})

	t.Run("Wrong current password", func(t *testing.T) {
		user := &models.User{ID: "user-pwd-wrong", Name: "Password User", Email: "pwd-wrong@test.com", Password: password, FirmID: stringToPtr(firm.ID), Role: "admin"}
		database.Create(user)

		f := url.Values{}
		f.Add("current_password", "wrongpassword")
		f.Add("new_password", "NewPassword123!")
		f.Add("confirm_password", "NewPassword123!")

		_, c, _ := setupEcho(http.MethodPost, "/profile/change-password", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := ChangePasswordHandler(c)
		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, he.Code)
	})

	t.Run("Passwords do not match", func(t *testing.T) {
		user := &models.User{ID: "user-pwd-mismatch", Name: "Password User", Email: "pwd-mismatch@test.com", Password: password, FirmID: stringToPtr(firm.ID), Role: "admin"}
		database.Create(user)

		f := url.Values{}
		f.Add("current_password", "oldpassword123")
		f.Add("new_password", "NewPassword123!")
		f.Add("confirm_password", "differentpassword")

		_, c, _ := setupEcho(http.MethodPost, "/profile/change-password", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := ChangePasswordHandler(c)
		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, he.Code)
	})

	t.Run("New password too short", func(t *testing.T) {
		user := &models.User{ID: "user-pwd-short", Name: "Password User", Email: "pwd-short@test.com", Password: password, FirmID: stringToPtr(firm.ID), Role: "admin"}
		database.Create(user)

		f := url.Values{}
		f.Add("current_password", "oldpassword123")
		f.Add("new_password", "short")
		f.Add("confirm_password", "short")

		_, c, _ := setupEcho(http.MethodPost, "/profile/change-password", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := ChangePasswordHandler(c)
		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, he.Code)
	})

	t.Run("Missing fields", func(t *testing.T) {
		user := &models.User{ID: "user-pwd-missing", Name: "Password User", Email: "pwd-missing@test.com", Password: password, FirmID: stringToPtr(firm.ID), Role: "admin"}
		database.Create(user)

		f := url.Values{}
		f.Add("current_password", "")
		f.Add("new_password", "")
		f.Add("confirm_password", "")

		_, c, _ := setupEcho(http.MethodPost, "/profile/change-password", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", user)

		err := ChangePasswordHandler(c)
		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, he.Code)
	})
}
