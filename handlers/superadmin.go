package handlers

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/superadmin"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SuperadminDashboardHandler renders the superadmin dashboard
func SuperadminDashboardHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	// Get all firms
	var firms []models.Firm
	if err := db.DB.Order("created_at DESC").Find(&firms).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch firms")
	}

	// Get user counts per firm
	type FirmStats struct {
		FirmID    string
		UserCount int64
	}
	var firmStats []FirmStats
	db.DB.Model(&models.User{}).
		Select("firm_id, count(*) as user_count").
		Where("firm_id IS NOT NULL").
		Group("firm_id").
		Scan(&firmStats)

	// Create a map for quick lookup
	statsMap := make(map[string]int64)
	for _, stat := range firmStats {
		statsMap[stat.FirmID] = stat.UserCount
	}

	// Render the superadmin dashboard
	component := superadmin.Dashboard(c.Request().Context(), "Superadmin Dashboard", csrfToken, currentUser, firms, statsMap)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminCreateUserHandler creates a new user without a firm
func SuperadminCreateUserHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)

	// Read form values
	name := c.FormValue("name")
	email := c.FormValue("email")
	password := c.FormValue("password")

	// Validate required fields
	if name == "" || email == "" || password == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="bg-red-500/10 border border-red-500/20 text-red-400 px-4 py-3 rounded-xl flex items-center gap-3"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">Name, email, and password are required</span></div>`)
		}
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name, email, and password are required",
		})
	}

	// Check if email already exists
	var existingUser models.User
	if err := db.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="bg-red-500/10 border border-red-500/20 text-red-400 px-4 py-3 rounded-xl flex items-center gap-3"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">A user with this email already exists</span></div>`)
		}
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "A user with this email already exists",
		})
	}

	// Validate password strength
	if err := services.ValidatePassword(password); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="bg-red-500/10 border border-red-500/20 text-red-400 px-4 py-3 rounded-xl flex items-center gap-3"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span class="text-sm font-medium">`+err.Error()+`</span></div>`)
		}
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Hash password
	hashedPassword, err := services.HashPassword(password)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to hash password")
	}

	// Create user without firm (FirmID = nil)
	// Role defaults to "admin" since they will set up their own firm
	user := &models.User{
		Name:     name,
		Email:    email,
		Password: hashedPassword,
		Role:     "admin",
		IsActive: true,
		FirmID:   nil, // No firm assigned - user will set up on first login
	}

	if err := db.DB.Create(user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create user")
	}

	// Log security event
	services.LogSecurityEvent("SUPERADMIN_USER_CREATED", currentUser.ID, "Created user without firm: "+user.ID)

	// Send welcome email asynchronously
	cfg := config.Load()
	email_obj := services.BuildWelcomeEmail(user.Email, user.Name)
	services.SendEmailAsync(cfg, email_obj)

	// Return success response
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", "reload-dashboard")
		return c.HTML(http.StatusOK, `<div class="bg-green-500/10 border border-green-500/20 text-green-400 px-4 py-3 rounded-xl flex items-center gap-3"><svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg><span class="text-sm font-medium">User created successfully! They can now log in and set up their firm.</span></div>`)
	}

	user.Password = "" // Don't return password
	return c.JSON(http.StatusCreated, user)
}
