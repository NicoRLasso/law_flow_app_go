package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// LoginHandler renders the login page
func LoginHandler(c echo.Context) error {
	component := pages.Login("Login | Law Flow")
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// LoginPostHandler handles the login form submission
func LoginPostHandler(c echo.Context) error {
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")

	// Validate input
	if email == "" || password == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500 text-sm mt-2">Email and password are required</div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Find user by email with firm preloaded
	var user models.User
	err := db.DB.Preload("Firm").Where("email = ?", email).First(&user).Error
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusUnauthorized, `<div class="text-red-500 text-sm mt-2">Invalid email or password</div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Verify password
	if !services.VerifyPassword(user.Password, password) {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusUnauthorized, `<div class="text-red-500 text-sm mt-2">Invalid email or password</div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Check if user is active
	if !user.IsActive {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusForbidden, `<div class="text-red-500 text-sm mt-2">Your account has been deactivated</div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Get client IP and user agent
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	// Create session (use empty string for FirmID if user doesn't have one yet)
	firmID := ""
	if user.FirmID != nil {
		firmID = *user.FirmID
	}
	session, err := services.CreateSession(db.DB, user.ID, firmID, ipAddress, userAgent)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create session")
	}

	// Set session cookie
	cookie := &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    session.Token,
		Path:     "/",
		MaxAge:   int(services.DefaultSessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	db.DB.Save(&user)

	// Check if user has a firm
	if !user.HasFirm() {
		// User needs to set up their firm first
		if c.Request().Header.Get("HX-Request") == "true" {
			c.Response().Header().Set("HX-Redirect", "/firm/setup")
			return c.NoContent(http.StatusOK)
		}
		return c.Redirect(http.StatusSeeOther, "/firm/setup")
	}

	// Redirect to dashboard
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/dashboard")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusSeeOther, "/dashboard")
}

// LogoutHandler handles user logout
func LogoutHandler(c echo.Context) error {
	// Get session cookie
	cookie, err := c.Cookie(middleware.SessionCookieName)
	if err == nil {
		// Delete session from database
		services.DeleteSession(db.DB, cookie.Value)
	}

	// Clear session cookie
	clearCookie := &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(clearCookie)

	// Redirect to login
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/login")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusSeeOther, "/login")
}

// GetCurrentUserHandler returns the current user info as JSON
func GetCurrentUserHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Not authenticated")
	}

	// Return user with firm info
	user.Firm = firm
	return c.JSON(http.StatusOK, user)
}
