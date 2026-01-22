package handlers

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"law_flow_app_go/templates/partials"
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

	if err := query.Preload("DocumentType").First(&user, "id = ?", id).Error; err != nil {
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
	firm := middleware.GetCurrentFirm(c)

	// Only admins can create users (enforced by route middleware)
	user := new(models.User)

	// Read form values
	user.Name = c.FormValue("name")
	user.Email = c.FormValue("email")
	user.Password = c.FormValue("password")
	user.Role = c.FormValue("role")

	// Handle optional fields
	if address := c.FormValue("address"); address != "" {
		user.Address = &address
	}
	if phoneNumber := c.FormValue("phone_number"); phoneNumber != "" {
		user.PhoneNumber = &phoneNumber
	}
	if documentTypeID := c.FormValue("document_type_id"); documentTypeID != "" {
		user.DocumentTypeID = &documentTypeID
	}
	if documentNumber := c.FormValue("document_number"); documentNumber != "" {
		user.DocumentNumber = &documentNumber
	}

	// Handle checkbox - only present if checked
	isActiveStr := c.FormValue("is_active")
	user.IsActive = isActiveStr == "true"

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

	// Validate password strength
	if err := services.ValidatePassword(user.Password); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Hash password
	hashedPassword, err := services.HashPassword(user.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to hash password",
		})
	}
	user.Password = hashedPassword

	if err := db.DB.Create(user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create user",
		})
	}

	// Create default availability for lawyers and admins
	if user.Role == "lawyer" || user.Role == "admin" {
		if err := services.CreateDefaultAvailability(user.ID); err != nil {
			// Log error but don't fail user creation
			services.LogSecurityEvent("AVAILABILITY_SEED_FAILED", user.ID, "Failed to create default availability: "+err.Error())
		}
	}

	// Log security event
	services.LogSecurityEvent("USER_CREATED", currentUser.ID, "Created user: "+user.ID)

	// Log audit event
	services.LogAuditEvent(db.DB, services.AuditContext{
		UserID:    currentUser.ID,
		UserName:  currentUser.Name,
		UserRole:  currentUser.Role,
		FirmID:    firm.ID,
		FirmName:  firm.Name,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}, models.AuditActionCreate, "user", user.ID, user.Name, "Created new user", nil, user)

	// Send welcome email asynchronously (non-blocking)
	cfg := config.Load()
	if user.Email != "" {
		userName := user.Name
		if userName == "" {
			userName = user.Email
		}
		userLang := user.Language
		if userLang == "" {
			userLang = "es"
		}
		email := services.BuildWelcomeEmail(user.Email, userName, userLang)
		services.SendEmailAsync(cfg, email)
	}

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// Close modal and reload users table
		c.Response().Header().Set("HX-Trigger", "reload-users")
		return c.NoContent(http.StatusOK)
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
	firm := middleware.GetCurrentFirm(c)

	// Capture old values for audit
	oldValues := map[string]interface{}{
		"name":             user.Name,
		"email":            user.Email,
		"role":             user.Role,
		"is_active":        user.IsActive,
		"address":          user.Address,
		"phone_number":     user.PhoneNumber,
		"document_type_id": user.DocumentTypeID,
		"document_number":  user.DocumentNumber,
	}

	// Read form values
	name := c.FormValue("name")
	email := c.FormValue("email")
	role := c.FormValue("role")
	isActiveStr := c.FormValue("is_active")

	// Update fields
	if name != "" {
		user.Name = name
	}
	if email != "" {
		user.Email = email
	}
	if role != "" {
		user.Role = role
	}
	user.IsActive = isActiveStr == "true"

	// Handle optional fields
	if address := c.FormValue("address"); address != "" {
		user.Address = &address
	}
	if phoneNumber := c.FormValue("phone_number"); phoneNumber != "" {
		user.PhoneNumber = &phoneNumber
	}
	if documentTypeID := c.FormValue("document_type_id"); documentTypeID != "" {
		user.DocumentTypeID = &documentTypeID
	}
	if documentNumber := c.FormValue("document_number"); documentNumber != "" {
		user.DocumentNumber = &documentNumber
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

	// Log audit event
	services.LogAuditEvent(db.DB, services.AuditContext{
		UserID:    currentUser.ID,
		UserName:  currentUser.Name,
		UserRole:  currentUser.Role,
		FirmID:    firm.ID,
		FirmName:  firm.Name,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}, models.AuditActionUpdate, "user", user.ID, user.Name, "Updated user details", oldValues, user)

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// Close modal and reload users table
		c.Response().Header().Set("HX-Trigger", "reload-users")
		return c.NoContent(http.StatusOK)
	}

	// Don't return password in response
	user.Password = ""
	return c.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user (admin only)
func DeleteUser(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

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

	// Log audit event
	services.LogAuditEvent(db.DB, services.AuditContext{
		UserID:    currentUser.ID,
		UserName:  currentUser.Name,
		UserRole:  currentUser.Role,
		FirmID:    firm.ID,
		FirmName:  firm.Name,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}, models.AuditActionDelete, "user", user.ID, user.Name, "Deleted user", user, nil)

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.NoContent(http.StatusOK)
	}

	return c.JSON(http.StatusNoContent, nil)
}

// UsersPageHandler renders the users management page
func UsersPageHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	csrfToken := middleware.GetCSRFToken(c)

	// Render the users page
	component := pages.Users(c.Request().Context(), "User Management", csrfToken, currentUser, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetUsersListHTMX returns the users table with optional filters
func GetUsersListHTMX(c echo.Context) error {
	var users []models.User
	currentUser := middleware.GetCurrentUser(c)

	// Get filter parameters
	roleFilter := c.QueryParam("role")
	statusFilter := c.QueryParam("status")

	// Scope query to current user's firm
	query := middleware.GetFirmScopedQuery(c, db.DB)

	// Apply role filter
	if roleFilter != "" {
		query = query.Where("role = ?", roleFilter)
	}

	// Apply status filter
	if statusFilter != "" {
		if statusFilter == "active" {
			query = query.Where("is_active = ?", true)
		} else if statusFilter == "inactive" {
			query = query.Where("is_active = ?", false)
		}
	}

	// Order by created_at descending
	query = query.Order("created_at DESC")

	if err := query.Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch users",
		})
	}

	// Render the users table
	component := partials.UsersTable(c.Request().Context(), users, currentUser.Role)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetUserFormNew returns the form modal for creating a new user
func GetUserFormNew(c echo.Context) error {
	// Render the form modal with empty user
	component := partials.UserFormModal(c.Request().Context(), nil, false)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetUserFormEdit returns the form modal for editing an existing user
func GetUserFormEdit(c echo.Context) error {
	id := c.Param("id")
	var user models.User

	// Scope query to current user's firm
	query := middleware.GetFirmScopedQuery(c, db.DB)

	if err := query.Preload("DocumentType").First(&user, "id = ?", id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	// Check authorization
	if !middleware.CanAccessUser(c, user.ID) {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	// Render the form modal with user data
	component := partials.UserFormModal(c.Request().Context(), &user, true)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetUserDeleteConfirm returns the delete confirmation modal
func GetUserDeleteConfirm(c echo.Context) error {
	id := c.Param("id")
	var user models.User

	// Scope query to current user's firm
	query := middleware.GetFirmScopedQuery(c, db.DB)

	if err := query.First(&user, "id = ?", id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	// Render the delete confirmation modal
	component := partials.DeleteConfirmModal(c.Request().Context(), user)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
