package handlers

import (
	"law_flow_app_go/config"
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

const dummyHash = "$2a$10$123456789012345678901uABCDEFGHIJabcdefghijABCDEFGHIJabc" // valid-looking bcrypt hash length

func init() {
	// Generate a real dummy hash at startup to ensure consistent timing
	// We ignore error here as it should not fail in normal operation
	hash, _ := services.HashPassword("dummy_password_for_timing_mitigation")
	if hash != "" {
		globalDummyHash = hash
	}
}

// Package level variable to hold the dummy hash
var globalDummyHash string = "$2a$10$X7.G.t8./.t.t.t.t.t.t.t.t.t.t.t.t.t.t.t.t.t.t.t.t" // Fallback

// LoginHandler renders the login page
func LoginHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	component := pages.Login(c.Request().Context(), "Login | Law Flow", csrfToken)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// LoginPostHandler handles the login form submission
func LoginPostHandler(c echo.Context) error {
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")

	// Validate input
	if email == "" || password == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="bg-red-500/10 border border-red-500/20 text-red-400 px-4 py-3 rounded-xl flex items-center gap-3 transition-all animate-in fade-in slide-in-from-top-2"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">Email and password are required</span></div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Find user by email with firm preloaded
	var user models.User
	err := db.DB.Preload("Firm").Where("email = ?", email).First(&user).Error
	if err != nil {
		// Timing attack mitigation:
		// Determine a valid hash to use (either from found user or dummy) to ensure VerifyPassword is always called
		// However, since we are in the "user not found" block, we MUST use the dummy hash.
		// We perform the check against the provided password.
		services.VerifyPassword(globalDummyHash, password)

		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="bg-red-500/10 border border-red-500/20 text-red-400 px-4 py-3 rounded-xl flex items-center gap-3 transition-all animate-in fade-in slide-in-from-top-2"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">Invalid email or password</span></div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Verify password
	if !services.VerifyPassword(user.Password, password) {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="bg-red-500/10 border border-red-500/20 text-red-400 px-4 py-3 rounded-xl flex items-center gap-3 transition-all animate-in fade-in slide-in-from-top-2"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">Invalid email or password</span></div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Check if user is active
	if !user.IsActive {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="bg-red-500/10 border border-red-500/20 text-red-400 px-4 py-3 rounded-xl flex items-center gap-3 transition-all animate-in fade-in slide-in-from-top-2"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">Your account has been deactivated</span></div>`)
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

	// Get config
	cfg := c.Get("config").(*config.Config)
	isProduction := cfg.Environment == "production"

	// Set session cookie
	cookie := &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    session.Token,
		Path:     "/",
		MaxAge:   int(services.DefaultSessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)

	// Set language cookie if user has a preference
	if user.Language != "" {
		middleware.SetLanguageCookie(c, user.Language)
	}

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	db.DB.Save(&user)

	// Check if user is superadmin - redirect to superadmin dashboard
	if user.IsSuperadmin() {
		if c.Request().Header.Get("HX-Request") == "true" {
			c.Response().Header().Set("HX-Redirect", "/superadmin")
			return c.NoContent(http.StatusOK)
		}
		return c.Redirect(http.StatusSeeOther, "/superadmin")
	}

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

	// Get config
	cfg := c.Get("config").(*config.Config)
	isProduction := cfg.Environment == "production"

	// Clear session cookie
	clearCookie := &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProduction,
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
