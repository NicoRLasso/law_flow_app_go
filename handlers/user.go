package handlers

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/services/i18n"
	"law_flow_app_go/templates/pages"
	"law_flow_app_go/templates/partials"
	"net/http"
	"strconv"
	"strings"

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

	// Check role first to determine if this is a billable user
	role := c.FormValue("role")
	if role == "" {
		role = "staff" // Default to staff
	}

	// Only check user limits for billable roles (admin, lawyer, staff)
	// Clients don't count towards the user limit
	if role == "admin" || role == "lawyer" || role == "staff" {
		limitResult, err := services.CanAddUser(db.DB, firm.ID)
		if err != nil {
			if err == services.ErrUserLimitReached {
				if c.Request().Header.Get("HX-Request") == "true" {
					title := i18n.T(c.Request().Context(), "subscription.errors.user_limit_title")
					message := i18n.T(c.Request().Context(), limitResult.TranslationKey, limitResult.TranslationArgs)
					btnText := i18n.T(c.Request().Context(), "subscription.errors.upgrade_plan")

					return c.HTML(http.StatusForbidden, `
						<div class="alert alert-warning shadow-lg">
							<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
							</svg>
							<div>
								<h3 class="font-bold">`+title+`</h3>
								<div class="text-xs">`+message+`</div>
							</div>
							<a href="/firm/settings#subscription" class="btn btn-sm btn-primary">`+btnText+`</a>
						</div>
					`)
				}
				return echo.NewHTTPError(http.StatusForbidden, i18n.T(c.Request().Context(), limitResult.TranslationKey, limitResult.TranslationArgs))
			}
			if err == services.ErrSubscriptionExpired {
				if c.Request().Header.Get("HX-Request") == "true" {
					// Use keys if available in limitResult, or fallback
					key := "subscription.errors.subscription_expired"
					if limitResult != nil && limitResult.TranslationKey != "" {
						key = limitResult.TranslationKey
					}

					title := i18n.T(c.Request().Context(), "subscription.errors.subscription_expired_title")
					message := i18n.T(c.Request().Context(), key)
					btnText := i18n.T(c.Request().Context(), "subscription.errors.renew_now")

					return c.HTML(http.StatusForbidden, `
						<div class="alert alert-error shadow-lg">
							<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"/>
							</svg>
							<div>
								<h3 class="font-bold">`+title+`</h3>
								<div class="text-xs">`+message+`</div>
							</div>
							<a href="/firm/settings#subscription" class="btn btn-sm btn-primary">`+btnText+`</a>
						</div>
					`)
				}
				// Use keys if available in limitResult
				key := "subscription.errors.subscription_expired"
				if limitResult != nil && limitResult.TranslationKey != "" {
					key = limitResult.TranslationKey
				}
				return echo.NewHTTPError(http.StatusForbidden, i18n.T(c.Request().Context(), key))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check subscription limits")
		}
	} else if role == "client" {
		limitResult, err := services.CanAddClient(db.DB, firm.ID)
		if err != nil {
			if err == services.ErrClientLimitReached {
				if c.Request().Header.Get("HX-Request") == "true" {
					title := i18n.T(c.Request().Context(), "subscription.errors.client_limit_title")
					message := i18n.T(c.Request().Context(), limitResult.TranslationKey, limitResult.TranslationArgs)
					btnText := i18n.T(c.Request().Context(), "subscription.errors.upgrade_plan")

					return c.HTML(http.StatusForbidden, `
						<div class="alert alert-warning shadow-lg">
							<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
							</svg>
							<div>
								<h3 class="font-bold">`+title+`</h3>
								<div class="text-xs">`+message+`</div>
							</div>
							<a href="/firm/settings#subscription" class="btn btn-sm btn-primary">`+btnText+`</a>
						</div>
					`)
				}
				return echo.NewHTTPError(http.StatusForbidden, i18n.T(c.Request().Context(), limitResult.TranslationKey, limitResult.TranslationArgs))
			}
			// Handle other errors same as standard user
			if err == services.ErrSubscriptionExpired {
				if c.Request().Header.Get("HX-Request") == "true" {
					// Use keys if available in limitResult
					key := "subscription.errors.subscription_expired"
					if limitResult != nil && limitResult.TranslationKey != "" {
						key = limitResult.TranslationKey
					}

					title := i18n.T(c.Request().Context(), "subscription.errors.subscription_expired_title")
					message := i18n.T(c.Request().Context(), key)
					btnText := i18n.T(c.Request().Context(), "subscription.errors.renew_now")

					return c.HTML(http.StatusForbidden, `
						<div class="alert alert-error shadow-lg">
							<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"/>
							</svg>
							<div>
								<h3 class="font-bold">`+title+`</h3>
								<div class="text-xs">`+message+`</div>
							</div>
							<a href="/firm/settings#subscription" class="btn btn-sm btn-primary">`+btnText+`</a>
						</div>
					`)
				}
				// Use keys if available
				key := "subscription.errors.subscription_expired"
				if limitResult != nil && limitResult.TranslationKey != "" {
					key = limitResult.TranslationKey
				}
				return echo.NewHTTPError(http.StatusForbidden, i18n.T(c.Request().Context(), key))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check subscription limits")
		}
	}

	// Only admins can create users (enforced by route middleware)
	user := new(models.User)

	// Read form values
	user.Name = c.FormValue("name")
	user.Email = strings.ToLower(strings.TrimSpace(c.FormValue("email")))
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

	// Length Validation
	if len(user.Name) > 255 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Name must be less than 255 characters"})
	}
	if len(user.Email) > 320 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email must be less than 320 characters"})
	}
	if len(user.Password) > 72 {
		if c.Request().Header.Get("HX-Request") == "true" {
			return partials.UserFormModal(c.Request().Context(), user, false, "Password must be less than 72 characters").Render(c.Request().Context(), c.Response().Writer)
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Password must be less than 72 characters"})
	}
	if user.Address != nil && len(*user.Address) > 255 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Address must be less than 255 characters"})
	}
	if user.PhoneNumber != nil && len(*user.PhoneNumber) > 20 {
		if c.Request().Header.Get("HX-Request") == "true" {
			return partials.UserFormModal(c.Request().Context(), user, false, "Phone number must be less than 20 characters").Render(c.Request().Context(), c.Response().Writer)
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Phone number must be less than 20 characters"})
	}
	if user.DocumentNumber != nil && len(*user.DocumentNumber) > 50 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Document number must be less than 50 characters"})
	}

	// Validate required fields
	if user.Email == "" || user.Password == "" || user.Name == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return partials.UserFormModal(c.Request().Context(), user, false, "Name, email, and password are required").Render(c.Request().Context(), c.Response().Writer)
		}
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name, email, and password are required",
		})
	}

	// Validate role (already validated and set at the beginning)
	validRoles := map[string]bool{
		"admin": true, "lawyer": true, "staff": true, "client": true,
	}
	if !validRoles[user.Role] {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid role. Must be one of: admin, lawyer, staff, client",
		})
	}

	// Force user to be in the same firm as creator
	user.FirmID = currentUser.FirmID

	// Validate password strength
	if err := services.ValidatePassword(user.Password); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return partials.UserFormModal(c.Request().Context(), user, false, err.Error()).Render(c.Request().Context(), c.Response().Writer)
		}
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Hash password
	hashedPassword, err := services.HashPassword(user.Password)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return partials.UserFormModal(c.Request().Context(), user, false, "Failed to hash password").Render(c.Request().Context(), c.Response().Writer)
		}
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

	// Update usage cache for billable users
	if user.Role == "admin" || user.Role == "lawyer" || user.Role == "staff" {
		if err := services.UpdateFirmUsageAfterUserChange(db.DB, firm.ID, 1); err != nil {
			// Log but don't fail - usage will be recalculated on next check
			services.LogSecurityEvent(db.DB, "USAGE_UPDATE_FAILED", currentUser.ID, "Failed to update user count: "+err.Error())
		}
	}

	// Create default availability for lawyers and admins
	if user.Role == "lawyer" || user.Role == "admin" {
		if err := services.CreateDefaultAvailability(db.DB, user.ID); err != nil {
			// Log error but don't fail user creation
			services.LogSecurityEvent(db.DB, "AVAILABILITY_SEED_FAILED", user.ID, "Failed to create default availability: "+err.Error())
		}
	}

	// Log security event
	services.LogSecurityEvent(db.DB, "USER_CREATED", currentUser.ID, "Created user: "+user.ID)

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
	email := strings.ToLower(strings.TrimSpace(c.FormValue("email")))
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

	// Length Validation
	if len(name) > 255 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Name must be less than 255 characters"})
	}
	if len(email) > 320 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email must be less than 320 characters"})
	}

	// Handle optional fields
	if address := c.FormValue("address"); address != "" {
		if len(address) > 255 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Address must be less than 255 characters"})
		}
		user.Address = &address
	}
	if phoneNumber := c.FormValue("phone_number"); phoneNumber != "" {
		if len(phoneNumber) > 20 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Phone number must be less than 20 characters"})
		}
		user.PhoneNumber = &phoneNumber
	}
	if documentTypeID := c.FormValue("document_type_id"); documentTypeID != "" {
		user.DocumentTypeID = &documentTypeID
	}
	if documentNumber := c.FormValue("document_number"); documentNumber != "" {
		if len(documentNumber) > 50 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Document number must be less than 50 characters"})
		}
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
		services.LogSecurityEvent(db.DB, "USER_MODIFIED", currentUser.ID, "Modified user: "+user.ID)
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

	// Update usage cache for billable users
	if user.Role == "admin" || user.Role == "lawyer" || user.Role == "staff" {
		if err := services.UpdateFirmUsageAfterUserChange(db.DB, firm.ID, -1); err != nil {
			// Log but don't fail - usage will be recalculated on next check
			services.LogSecurityEvent(db.DB, "USAGE_UPDATE_FAILED", currentUser.ID, "Failed to update user count: "+err.Error())
		}
	}

	// Log security event
	services.LogSecurityEvent(db.DB, "USER_DELETED", currentUser.ID, "Deleted user: "+user.ID)

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
	currentUser := middleware.GetCurrentUser(c)

	// Get filter parameters
	roleFilter := c.QueryParam("role")
	statusFilter := c.QueryParam("status")

	// Get pagination parameters
	page := 1
	limit := 10
	if pageParam := c.QueryParam("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

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

	// Get total count
	var total int64
	if err := query.Model(&models.User{}).Count(&total).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Failed to count users")
	}

	// Calculate pagination
	offset := (page - 1) * limit
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Fetch paginated users
	var users []models.User
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch users",
		})
	}

	// Render the users table
	component := partials.UsersTable(c.Request().Context(), users, currentUser.Role, page, totalPages, limit, int(total))
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetUserFormNew returns the form modal for creating a new user
func GetUserFormNew(c echo.Context) error {
	// Render the form modal with empty user
	component := partials.UserFormModal(c.Request().Context(), nil, false, "")
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
	component := partials.UserFormModal(c.Request().Context(), &user, true, "")
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
