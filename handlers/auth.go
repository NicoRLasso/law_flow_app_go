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

// getLockoutDuration returns the lockout duration based on the lockout count (exponential backoff)
// 1st lockout: 15 minutes, 2nd: 30 minutes, 3rd: 1 hour, 4th+: 24 hours
func getLockoutDuration(lockoutCount int) time.Duration {
	switch lockoutCount {
	case 0:
		return 15 * time.Minute
	case 1:
		return 30 * time.Minute
	case 2:
		return 1 * time.Hour
	default:
		return 24 * time.Hour
	}
}

// LoginHandler renders the login page
func LoginHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	component := pages.Login(c.Request().Context(), "Login | LexLegal Cloud", csrfToken)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// LoginPostHandler handles the login form submission
func LoginPostHandler(c echo.Context) error {
	email := strings.TrimSpace(strings.ToLower(c.FormValue("email")))
	password := c.FormValue("password")

	// Validate input
	if email == "" || password == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="bg-red-500/10 border border-red-500/20  px-4 py-3 rounded-xl flex items-center gap-3 transition-all animate-in fade-in slide-in-from-top-2"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">Email and password are required</span></div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Find user by email with firm preloaded
	// Note: We do NOT pre-check lockout status separately to avoid timing attacks
	var user models.User
	err := db.DB.Preload("Firm").Where("email = ?", email).First(&user).Error

	// Always perform password verification to ensure constant timing
	// This prevents username enumeration via timing attacks
	hashToVerify := globalDummyHash
	if err == nil {
		hashToVerify = user.Password
	}
	passwordValid := services.VerifyPassword(hashToVerify, password)

	// Handle user not found (after password check to maintain constant timing)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="bg-red-500/10 border border-red-500/20  px-4 py-3 rounded-xl flex items-center gap-3 transition-all animate-in fade-in slide-in-from-top-2"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">Invalid email or password</span></div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Check if account is locked (after password verification for constant timing)
	if user.LockoutUntil != nil && time.Now().Before(*user.LockoutUntil) {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="bg-red-500/10 border border-red-500/20  px-4 py-3 rounded-xl flex items-center gap-3 transition-all animate-in fade-in slide-in-from-top-2"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">Account is locked. Try again later.</span></div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Verify password result
	if !passwordValid {
		services.Monitor.TrackFailedLogin(c.RealIP()) // Track security event
		// Increment failed login attempts
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= 5 {
			// Apply exponential backoff for lockout duration
			lockoutDuration := getLockoutDuration(user.LockoutCount)
			lockoutTime := time.Now().Add(lockoutDuration)
			user.LockoutUntil = &lockoutTime
			user.LockoutCount++ // Increment lockout count for next time
			user.FailedLoginAttempts = 0

			// Log security event for excessive failed attempts
			services.LogSecurityEvent(db.DB, "ACCOUNT_LOCKED", user.ID, "Account locked due to excessive failed login attempts")
		}
		db.DB.Save(&user)

		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="bg-red-500/10 border border-red-500/20  px-4 py-3 rounded-xl flex items-center gap-3 transition-all animate-in fade-in slide-in-from-top-2"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">Invalid email or password</span></div>`)
		}
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	// Reset failed attempts and lockout on successful login
	if user.FailedLoginAttempts > 0 || user.LockoutUntil != nil {
		user.FailedLoginAttempts = 0
		user.LockoutUntil = nil
		// Reset lockout count on successful login (user has proven they know the password)
		user.LockoutCount = 0
		db.DB.Save(&user)
	}

	// Check if user is active
	if !user.IsActive {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="bg-red-500/10 border border-red-500/20  px-4 py-3 rounded-xl flex items-center gap-3 transition-all animate-in fade-in slide-in-from-top-2"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">Your account has been deactivated</span></div>`)
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

	// Audit logging (Login)
	auditCtx := services.AuditContext{
		UserID:    user.ID,
		UserName:  user.Name,
		UserRole:  user.Role,
		FirmID:    firmID,
		FirmName:  "", // Will be updated if firm exists
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
	if user.Firm != nil {
		auditCtx.FirmName = user.Firm.Name
	}
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionLogin, "User", user.ID, user.Name, "User logged in", nil, nil)

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
	// Audit logging (Logout) - capture user before session deletion
	if user := middleware.GetCurrentUser(c); user != nil {
		firmID := ""
		if user.FirmID != nil {
			firmID = *user.FirmID
		}
		// Try to get firm name if possible, or leave empty if not critical
		firmName := ""
		if user.Firm != nil {
			firmName = user.Firm.Name
		}

		auditCtx := services.AuditContext{
			UserID:    user.ID,
			UserName:  user.Name,
			UserRole:  user.Role,
			FirmID:    firmID,
			FirmName:  firmName,
			IPAddress: c.RealIP(),
			UserAgent: c.Request().UserAgent(),
		}
		services.LogAuditEvent(db.DB, auditCtx, models.AuditActionLogout, "User", user.ID, user.Name, "User logged out", nil, nil)
	}

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
