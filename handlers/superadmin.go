package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/superadmin"
	superadmin_partials "law_flow_app_go/templates/superadmin/partials"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SuperadminDashboardHandler renders the superadmin dashboard
func SuperadminDashboardHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	// Stats: Total Firms
	var totalFirms int64
	db.DB.Model(&models.Firm{}).Count(&totalFirms)

	// Stats: Total Users
	var totalUsers int64
	db.DB.Model(&models.User{}).Count(&totalUsers)

	// Stats: Active Users
	var activeUsers int64
	db.DB.Model(&models.User{}).Where("is_active = ?", true).Count(&activeUsers)

	// Stats: Users Pending Setup (FirmID is null)
	var pendingUsers int64
	db.DB.Model(&models.User{}).Where("firm_id IS NULL").Count(&pendingUsers)

	// Recent Firms (Top 5)
	var recentFirms []models.Firm
	if err := db.DB.Order("created_at DESC").Limit(5).Find(&recentFirms).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch recent firms")
	}

	// Recent Users (Top 5) with Firm preloaded
	var recentUsers []models.User
	if err := db.DB.Preload("Firm").Order("created_at DESC").Limit(5).Find(&recentUsers).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch recent users")
	}

	// Render the superadmin dashboard
	component := superadmin.Dashboard(
		c.Request().Context(),
		"Superadmin Dashboard",
		csrfToken,
		currentUser,
		c.Request().URL.Path,
		totalFirms,
		totalUsers,
		activeUsers,
		pendingUsers,
		recentFirms,
		recentUsers,
	)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminUsersPageHandler renders the users management page
func SuperadminUsersPageHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	// Fetch all firms for filter
	var firms []models.Firm
	if err := db.DB.Order("name ASC").Find(&firms).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch firms")
	}

	// Fetch initial users (limit 50)
	var users []models.User
	if err := db.DB.Preload("Firm").Order("created_at DESC").Limit(50).Find(&users).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch users")
	}

	component := superadmin.UsersPage(
		c.Request().Context(),
		"User Management",
		csrfToken,
		currentUser,
		c.Request().URL.Path,
		users,
		firms,
	)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminGetUsersListHTMX returns the filtered users table
func SuperadminGetUsersListHTMX(c echo.Context) error {
	query := db.DB.Model(&models.User{}).Preload("Firm")

	// Filters
	if search := c.QueryParam("search"); search != "" {
		searchLike := "%" + search + "%"
		query = query.Where("name LIKE ? OR email LIKE ?", searchLike, searchLike)
	}
	if role := c.QueryParam("role"); role != "" {
		query = query.Where("role = ?", role)
	}
	if firmID := c.QueryParam("firm_id"); firmID != "" {
		if firmID == "none" {
			query = query.Where("firm_id IS NULL")
		} else {
			query = query.Where("firm_id = ?", firmID)
		}
	}
	if status := c.QueryParam("status"); status != "" {
		if status == "active" {
			query = query.Where("is_active = ?", true)
		} else if status == "inactive" {
			query = query.Where("is_active = ?", false)
		}
	}

	var users []models.User
	if err := query.Order("created_at DESC").Limit(50).Find(&users).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch users")
	}

	component := superadmin_partials.UsersTable(c.Request().Context(), users)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminGetUserFormNew renders the create user modal
func SuperadminGetUserFormNew(c echo.Context) error {
	var firms []models.Firm
	if err := db.DB.Order("name ASC").Find(&firms).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch firms")
	}

	component := superadmin_partials.UserFormModal(c.Request().Context(), nil, firms, false)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminGetUserFormEdit renders the edit user modal
func SuperadminGetUserFormEdit(c echo.Context) error {
	id := c.Param("id")
	var user models.User
	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}

	var firms []models.Firm
	if err := db.DB.Order("name ASC").Find(&firms).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch firms")
	}

	component := superadmin_partials.UserFormModal(c.Request().Context(), &user, firms, true)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminCreateUserHandler creates a new user (Updated to handle modal)
func SuperadminCreateUserHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)

	name := c.FormValue("name")
	email := c.FormValue("email")
	password := c.FormValue("password")
	role := c.FormValue("role")
	isActive := c.FormValue("is_active") == "true"
	firmID := c.FormValue("firm_id")

	// Basic validation
	if name == "" || email == "" || password == "" {
		return c.String(http.StatusBadRequest, "<div class='text-red-400'>Name, email, and password are required</div>")
	}

	// Password strength
	if err := services.ValidatePassword(password); err != nil {
		return c.String(http.StatusBadRequest, "<div class='text-red-400'>"+err.Error()+"</div>")
	}

	hashedPassword, err := services.HashPassword(password)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to hash password")
	}

	user := &models.User{
		Name:     name,
		Email:    email,
		Password: hashedPassword,
		Role:     role,
		IsActive: isActive,
	}

	if firmID != "" {
		user.FirmID = &firmID
	}

	if err := db.DB.Create(user).Error; err != nil {
		return c.String(http.StatusBadRequest, "<div class='text-red-400'>Failed to create user. Email might be taken.</div>")
	}

	services.LogSecurityEvent("SUPERADMIN_USER_CREATED", currentUser.ID, "Created user: "+user.ID)

	// Return updated list
	// We need to fetch and render the list again to close modal and show new data
	// But simply returning the table HTMX will replace the modal target?
	// The modal target is #users-table-container.
	// So returning the table will replace the table content, AND we need to close the modal.
	// To close the modal, we can return OOB swap to empty the modal container.

	// Fetch updated users
	return SuperadminGetUsersListHTMX(c)
	// Note: This replaces the table. But the modal background is in #modal-container.
	// If the modal form targets #users-table-container, it replaces the table.
	// The modal itself (which is in #modal-container) remains open?
	// Ah, the UserFormModal template has `hx-target="#users-table-container"`.
	// The modal is OVER the table. If I update the table, the modal is still there.
	// I need to close the modal too.
	// I can use `hx-swap-oob` to close the modal.
	// I will return the table AND an OOB swap for #modal-container.
}

// SuperadminUpdateUser updates a user
func SuperadminUpdateUser(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	var user models.User
	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}

	user.Name = c.FormValue("name")
	user.Email = c.FormValue("email")
	user.Role = c.FormValue("role")
	user.IsActive = c.FormValue("is_active") == "true"

	firmID := c.FormValue("firm_id")
	if firmID != "" {
		user.FirmID = &firmID
	} else {
		user.FirmID = nil
	}

	if password := c.FormValue("password"); password != "" {
		if err := services.ValidatePassword(password); err != nil {
			return c.String(http.StatusBadRequest, "<div class='text-red-400'>"+err.Error()+"</div>")
		}
		hashed, _ := services.HashPassword(password)
		user.Password = hashed
	}

	if err := db.DB.Save(&user).Error; err != nil {
		return c.String(http.StatusBadRequest, "<div class='text-red-400'>Failed to update user</div>")
	}

	services.LogSecurityEvent("SUPERADMIN_USER_UPDATED", currentUser.ID, "Updated user: "+user.ID)

	// Return updated list + close modal via OOB
	c.Response().Header().Set("HX-Trigger", "closeModal") // Or I can use OOB

	return SuperadminGetUsersListHTMX(c)
}

// SuperadminToggleUserActive toggles user status
func SuperadminToggleUserActive(c echo.Context) error {
	id := c.Param("id")
	var user models.User
	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}

	user.IsActive = !user.IsActive
	db.DB.Save(&user)

	return SuperadminGetUsersListHTMX(c)
}

// SuperadminGetUserDeleteConfirm renders delete confirmation
func SuperadminGetUserDeleteConfirm(c echo.Context) error {
	id := c.Param("id")
	var user models.User
	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}

	component := superadmin_partials.UserDeleteConfirmModal(c.Request().Context(), &user)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminDeleteUser soft deletes a user
func SuperadminDeleteUser(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	if err := db.DB.Delete(&models.User{}, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete user")
	}

	services.LogSecurityEvent("SUPERADMIN_USER_DELETED", currentUser.ID, "Deleted user: "+id)

	return SuperadminGetUsersListHTMX(c)
}

// --- Firm Management Handlers ---

// SuperadminFirmsPageHandler renders the firms management page
func SuperadminFirmsPageHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	// Fetch firms (limit 50 initially)
	var firms []models.Firm
	if err := db.DB.Order("created_at DESC").Limit(50).Find(&firms).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch firms")
	}

	component := superadmin.FirmsPage(
		c.Request().Context(),
		"Firm Management",
		csrfToken,
		currentUser,
		c.Request().URL.Path,
		firms,
	)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminGetFirmsListHTMX returns the filtered firms table
func SuperadminGetFirmsListHTMX(c echo.Context) error {
	query := db.DB.Model(&models.Firm{})

	// Filters
	if search := c.QueryParam("search"); search != "" {
		searchLike := "%" + search + "%"
		query = query.Where("name LIKE ? OR billing_email LIKE ?", searchLike, searchLike)
	}

	if status := c.QueryParam("status"); status != "" {
		if status == "active" {
			query = query.Where("is_active = ?", true)
		} else if status == "inactive" {
			query = query.Where("is_active = ?", false)
		}
	}

	var firms []models.Firm
	if err := query.Order("created_at DESC").Limit(50).Find(&firms).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch firms")
	}

	component := superadmin_partials.FirmsTable(c.Request().Context(), firms)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminGetFirmFormNew renders the create firm modal
func SuperadminGetFirmFormNew(c echo.Context) error {
	component := superadmin_partials.FirmDetailModal(c.Request().Context(), &models.Firm{})
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminGetFirmFormEdit renders the edit firm modal
func SuperadminGetFirmFormEdit(c echo.Context) error {
	id := c.Param("id")
	var firm models.Firm
	if err := db.DB.First(&firm, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Firm not found")
	}

	component := superadmin_partials.FirmDetailModal(c.Request().Context(), &firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminCreateFirmHandler creates a new firm
func SuperadminCreateFirmHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)

	name := c.FormValue("name")
	billingEmail := c.FormValue("billing_email")
	country := c.FormValue("country")

	// Basic Validation
	if name == "" || billingEmail == "" || country == "" {
		return c.String(http.StatusBadRequest, "<div class='text-red-400'>Name, billing email and country are required</div>")
	}

	firm := &models.Firm{
		Name:         name,
		BillingEmail: billingEmail,
		NoreplyEmail: c.FormValue("noreply_email"),
		Country:      country,
		City:         c.FormValue("city"),
		Address:      c.FormValue("address"),
		IsActive:     c.FormValue("is_active") == "true",
	}

	if err := db.DB.Create(firm).Error; err != nil {
		return c.String(http.StatusBadRequest, "<div class='text-red-400'>Failed to create firm: "+err.Error()+"</div>")
	}

	services.LogSecurityEvent("SUPERADMIN_FIRM_CREATED", currentUser.ID, "Created firm: "+firm.ID)

	return SuperadminGetFirmsListHTMX(c)
}

// SuperadminUpdateFirm updates a firm
func SuperadminUpdateFirm(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	var firm models.Firm
	if err := db.DB.First(&firm, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Firm not found")
	}

	firm.Name = c.FormValue("name")
	firm.BillingEmail = c.FormValue("billing_email")
	firm.NoreplyEmail = c.FormValue("noreply_email")
	firm.Country = c.FormValue("country")
	firm.City = c.FormValue("city")
	firm.Address = c.FormValue("address")
	firm.IsActive = c.FormValue("is_active") == "true"

	if err := db.DB.Save(&firm).Error; err != nil {
		return c.String(http.StatusBadRequest, "<div class='text-red-400'>Failed to update firm: "+err.Error()+"</div>")
	}

	services.LogSecurityEvent("SUPERADMIN_FIRM_UPDATED", currentUser.ID, "Updated firm: "+firm.ID)

	// Return updated list, relying on client-side modal removal (like user update)
	// But unlike user updated, we are replacing the LIST only.
	// We need to close the modal too.
	// The modal is in #modal-container. The list is #firms-table-container.
	// If the form target is #firms-table-container, the modal stays open.
	// We can add an OOB swap to close the modal or use a trigger.
	c.Response().Header().Set("HX-Trigger", "closeModal") // Requires JS handling or OOB

	return SuperadminGetFirmsListHTMX(c)
}

// SuperadminToggleFirmActive toggles firm status
func SuperadminToggleFirmActive(c echo.Context) error {
	id := c.Param("id")
	var firm models.Firm
	if err := db.DB.First(&firm, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Firm not found")
	}

	firm.IsActive = !firm.IsActive
	db.DB.Save(&firm)

	return SuperadminGetFirmsListHTMX(c)
}

// SuperadminGetFirmDeleteConfirm renders delete confirmation for firm
func SuperadminGetFirmDeleteConfirm(c echo.Context) error {
	id := c.Param("id")
	var firm models.Firm
	if err := db.DB.First(&firm, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Firm not found")
	}

	component := superadmin_partials.FirmDeleteConfirmModal(c.Request().Context(), &firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminDeleteFirm soft deletes a firm
func SuperadminDeleteFirm(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	if err := db.DB.Delete(&models.Firm{}, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete firm")
	}

	services.LogSecurityEvent("SUPERADMIN_FIRM_DELETED", currentUser.ID, "Deleted firm: "+id)

	return SuperadminGetFirmsListHTMX(c)
}
