package handlers

import (
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestLoginHandler(t *testing.T) {
	_, c, rec := setupEcho(http.MethodGet, "/login", nil)

	err := LoginHandler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLoginPostHandler(t *testing.T) {
	setup := func(t *testing.T, email, password string) (*gorm.DB, *models.User, *models.Firm) {
		database := setupTestDB(t)
		firm := &models.Firm{ID: "firm-" + email, Name: "Firm " + email}
		database.Create(firm)
		hashedPassword, _ := services.HashPassword(password)
		user := &models.User{
			ID:       "user-" + email,
			Email:    email,
			Password: hashedPassword,
			Name:     "Test " + email,
			IsActive: true,
			FirmID:   stringToPtr(firm.ID),
		}
		database.Create(user)
		return database, user, firm
	}

	t.Run("Valid credentials", func(t *testing.T) {
		email := "valid@test.com"
		password := "pass123456789"
		_, _, _ = setup(t, email, password)

		f := url.Values{}
		f.Add("email", email)
		f.Add("password", password)

		_, c, rec := setupEcho(http.MethodPost, "/login", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := LoginPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/dashboard", rec.Header().Get("Location"))
	})

	t.Run("Invalid credentials", func(t *testing.T) {
		email := "invalid@test.com"
		password := "pass123456789"
		_, _, _ = setup(t, email, password)

		f := url.Values{}
		f.Add("email", email)
		f.Add("password", "wrong")

		_, c, rec := setupEcho(http.MethodPost, "/login", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := LoginPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})

	t.Run("HTMX request error", func(t *testing.T) {
		email := "htmx@test.com"
		password := "pass123456789"
		_, _, _ = setup(t, email, password)

		f := url.Values{}
		f.Add("email", email)
		f.Add("password", "wrong")

		_, c, rec := setupEcho(http.MethodPost, "/login", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Request().Header.Set("HX-Request", "true")

		err := LoginPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid email or password")
	})

	t.Run("Deactivated user", func(t *testing.T) {
		email := "inactive@test.com"
		password := "pass123456789"
		db, user, _ := setup(t, email, password)
		user.IsActive = false
		db.Save(user)

		f := url.Values{}
		f.Add("email", email)
		f.Add("password", password)

		_, c, rec := setupEcho(http.MethodPost, "/login", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := LoginPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})

	t.Run("Locked user", func(t *testing.T) {
		email := "locked@test.com"
		password := "pass123456789"
		db, user, _ := setup(t, email, password)
		until := time.Now().Add(1 * time.Hour)
		user.LockoutUntil = &until
		db.Save(user)

		f := url.Values{}
		f.Add("email", email)
		f.Add("password", password)

		_, c, rec := setupEcho(http.MethodPost, "/login", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := LoginPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})

	t.Run("Superadmin user", func(t *testing.T) {
		email := "super@test.com"
		password := "pass123456789"
		db, user, _ := setup(t, email, password)
		user.Role = "superadmin"
		db.Save(user)

		f := url.Values{}
		f.Add("email", email)
		f.Add("password", password)

		_, c, rec := setupEcho(http.MethodPost, "/login", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := LoginPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/superadmin", rec.Header().Get("Location"))
	})

	t.Run("User without firm", func(t *testing.T) {
		email := "nofirm@test.com"
		password := "pass123456789"
		db, user, _ := setup(t, email, password)
		user.FirmID = nil
		db.Save(user)

		f := url.Values{}
		f.Add("email", email)
		f.Add("password", password)

		_, c, rec := setupEcho(http.MethodPost, "/login", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := LoginPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/firm/setup", rec.Header().Get("Location"))
	})

	t.Run("Login failure increments", func(t *testing.T) {
		email := "fail@test.com"
		password := "pass123456789"
		db, user, _ := setup(t, email, password)

		f := url.Values{}
		f.Add("email", email)
		f.Add("password", "wrong")

		_, c, rec := setupEcho(http.MethodPost, "/login", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := LoginPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)

		db.First(user, "id = ?", user.ID)
		assert.Equal(t, 1, user.FailedLoginAttempts)
	})

	t.Run("Lockout trigger", func(t *testing.T) {
		email := "lockout@test.com"
		password := "pass123456789"
		db, user, _ := setup(t, email, password)
		user.FailedLoginAttempts = 4
		db.Save(user)

		f := url.Values{}
		f.Add("email", email)
		f.Add("password", "wrong")

		_, c, rec := setupEcho(http.MethodPost, "/login", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := LoginPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)

		db.First(user, "id = ?", user.ID)
		assert.NotNil(t, user.LockoutUntil)
		assert.True(t, time.Now().Before(*user.LockoutUntil))
		assert.Equal(t, 0, user.FailedLoginAttempts)
	})

	t.Run("Empty inputs", func(t *testing.T) {
		f := url.Values{}
		f.Add("email", "")
		f.Add("password", "")

		_, c, rec := setupEcho(http.MethodPost, "/login", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := LoginPostHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})
}

func TestGetLockoutDuration(t *testing.T) {
	assert.Equal(t, 15*time.Minute, getLockoutDuration(0))
	assert.Equal(t, 30*time.Minute, getLockoutDuration(1))
	assert.Equal(t, 1*time.Hour, getLockoutDuration(2))
	assert.Equal(t, 24*time.Hour, getLockoutDuration(3))
	assert.Equal(t, 24*time.Hour, getLockoutDuration(10))
}

func TestLogoutHandler(t *testing.T) {
	_ = setupTestDB(t)
	_, c, rec := setupEcho(http.MethodGet, "/logout", nil)

	// Set user to cover audit log
	user := &models.User{ID: "user-logout", Name: "Logout User"}
	c.Set("user", user)

	err := LogoutHandler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestGetCurrentUserHandler(t *testing.T) {
	_ = setupTestDB(t)
	_, c, rec := setupEcho(http.MethodGet, "/api/me", nil)

	t.Run("Unauthorized", func(t *testing.T) {
		err := GetCurrentUserHandler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, err.(*echo.HTTPError).Code)
	})

	t.Run("Authorized", func(t *testing.T) {
		user := &models.User{ID: "user-1", Name: "Test"}
		c.Set("user", user)

		err := GetCurrentUserHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "user-1")
	})
}
