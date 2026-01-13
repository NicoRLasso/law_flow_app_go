package handlers

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetUsers returns all users in the current user's firm
func GetUsers(c echo.Context) error {
	var users []models.User

	// Scope query to current user's firm
	query := middleware.GetFirmScopedQuery(c, db.DB)

	if err := query.Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch users",
		})
	}

	return c.JSON(http.StatusOK, users)
}

// GetUser returns a single user by ID
func GetUser(c echo.Context) error {
	id := c.Param("id")
	var user models.User

	// Scope query to current user's firm
	query := middleware.GetFirmScopedQuery(c, db.DB)

	if err := query.First(&user, "id = ?", id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	// Check authorization (admins can view all, others only themselves)
	if !middleware.CanAccessUser(c, user.ID) {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	return c.JSON(http.StatusOK, user)
}

// CreateUser creates a new user (admin only)
func CreateUser(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)

	// Only admins can create users (enforced by route middleware)
	user := new(models.User)

	if err := c.Bind(user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if user.Email == "" || user.Password == "" || user.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name, email, and password are required",
		})
	}

	// Validate role
	validRoles := map[string]bool{
		"admin": true, "lawyer": true, "staff": true, "client": true,
	}
	if user.Role == "" {
		user.Role = "staff" // Default to staff
	} else if !validRoles[user.Role] {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid role. Must be one of: admin, lawyer, staff, client",
		})
	}

	// Force user to be in the same firm as creator
	user.FirmID = currentUser.FirmID

	// Hash password
	hashedPassword, err := services.HashPassword(user.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to hash password",
		})
	}
	user.Password = hashedPassword

	// Set IsActive to true by default
	user.IsActive = true

	if err := db.DB.Create(user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create user",
		})
	}

	// Log security event
	services.LogSecurityEvent("USER_CREATED", currentUser.ID, "Created user: "+user.ID)

	// Send welcome email asynchronously (non-blocking)
	cfg := config.Load()
	if user.Email != "" {
		userName := user.Name
		if userName == "" {
			userName = user.Email
		}
		email := services.BuildWelcomeEmail(user.Email, userName)
		services.SendEmailAsync(cfg, email)
	}

	// Don't return password in response
	user.Password = ""
	return c.JSON(http.StatusCreated, user)
}

// UpdateUser updates an existing user
func UpdateUser(c echo.Context) error {
	id := c.Param("id")
	var user models.User

	// Scope query to current user's firm
	query := middleware.GetFirmScopedQuery(c, db.DB)

	if err := query.First(&user, "id = ?", id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	// Check authorization
	if !middleware.CanModifyUser(c, user.ID) {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	// Store original values that shouldn't be changed by non-admins
	originalFirmID := user.FirmID
	originalRole := user.Role
	originalPassword := user.Password
	currentUser := middleware.GetCurrentUser(c)

	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Non-admins cannot change firm or role
	if currentUser.Role != "admin" {
		user.FirmID = originalFirmID
		user.Role = originalRole
	}

	// Validate role if admin is changing it
	if currentUser.Role == "admin" && user.Role != "" {
		validRoles := map[string]bool{
			"admin": true, "lawyer": true, "staff": true, "client": true,
		}
		if !validRoles[user.Role] {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid role. Must be one of: admin, lawyer, staff, client",
			})
		}
	}

	// Don't allow updating password through this endpoint
	// (should have separate password change endpoint)
	user.Password = originalPassword

	if err := db.DB.Save(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update user",
		})
	}

	// Log security event if admin modified another user
	if currentUser.ID != user.ID {
		services.LogSecurityEvent("USER_MODIFIED", currentUser.ID, "Modified user: "+user.ID)
	}

	// Don't return password in response
	user.Password = ""
	return c.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user (admin only)
func DeleteUser(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	var user models.User

	// Scope query to current user's firm
	query := middleware.GetFirmScopedQuery(c, db.DB)

	if err := query.First(&user, "id = ?", id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	// Prevent admins from deleting themselves
	if user.ID == currentUser.ID {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Cannot delete your own account",
		})
	}

	// Soft delete (GORM's default with DeletedAt field)
	if err := db.DB.Delete(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete user",
		})
	}

	// Log security event
	services.LogSecurityEvent("USER_DELETED", currentUser.ID, "Deleted user: "+user.ID)

	return c.JSON(http.StatusNoContent, nil)
}
