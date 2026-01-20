package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/services/i18n"
	"law_flow_app_go/templates/components"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// GetSubtypesTabHandler returns the classifications tab content with domain/branch/subtype hierarchy
func GetSubtypesTabHandler(c echo.Context) error {
	ctx := c.Request().Context()
	user := middleware.GetCurrentUser(c)

	// Get all domains for the firm with branches and subtypes preloaded
	var domains []models.CaseDomain
	if err := db.DB.
		Where("firm_id = ?", *user.FirmID).
		Order("`order` ASC, name ASC").
		Preload("Branches", func(d *gorm.DB) *gorm.DB {
			return d.Order("`order` ASC, name ASC")
		}).
		Preload("Branches.Subtypes", func(d *gorm.DB) *gorm.DB {
			return d.Order("`order` ASC, name ASC")
		}).
		Find(&domains).Error; err != nil {
		return c.String(http.StatusInternalServerError, i18n.T(ctx, "common.error"))
	}

	return components.SubtypeList(ctx, domains).Render(c.Request().Context(), c.Response().Writer)
}

// GetSubtypesForBranchHandler returns subtypes for a specific branch (JSON for Alpine.js)
func GetSubtypesForBranchHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	branchID := c.Param("branchId")

	subtypes, err := services.GetCaseSubtypes(db.DB, *user.FirmID, branchID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch subtypes"})
	}

	return c.JSON(http.StatusOK, subtypes)
}

// GetSubtypeFormHandler returns the modal form for creating/editing a subtype
func GetSubtypeFormHandler(c echo.Context) error {
	ctx := c.Request().Context()
	user := middleware.GetCurrentUser(c)
	subtypeID := c.Param("id")
	branchID := c.QueryParam("branch_id")

	var subtype *models.CaseSubtype
	var branch models.CaseBranch

	if subtypeID != "" {
		// Editing existing subtype
		subtype = &models.CaseSubtype{}
		if err := db.DB.
			Preload("Branch").
			Preload("Branch.Domain").
			First(subtype, "id = ? AND firm_id = ?", subtypeID, *user.FirmID).Error; err != nil {
			return c.String(http.StatusNotFound, i18n.T(ctx, "common.not_found"))
		}
		branch = subtype.Branch
	} else {
		// Creating new subtype - need branch info
		if branchID == "" {
			return c.String(http.StatusBadRequest, "branch_id is required")
		}
		if err := db.DB.
			Preload("Domain").
			First(&branch, "id = ? AND firm_id = ?", branchID, *user.FirmID).Error; err != nil {
			return c.String(http.StatusNotFound, i18n.T(ctx, "common.not_found"))
		}
	}

	return components.SubtypeFormModal(ctx, subtype, branch).Render(c.Request().Context(), c.Response().Writer)
}

// CreateSubtypeHandler creates a new subtype
func CreateSubtypeHandler(c echo.Context) error {
	ctx := c.Request().Context()
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	branchID := c.FormValue("branch_id")
	name := strings.TrimSpace(c.FormValue("name"))
	code := strings.TrimSpace(c.FormValue("code"))
	description := strings.TrimSpace(c.FormValue("description"))
	orderStr := c.FormValue("order")
	durationStr := c.FormValue("typical_duration_days")
	complexity := c.FormValue("complexity_level")

	// Validate required fields
	if branchID == "" || name == "" || code == "" {
		return c.String(http.StatusBadRequest, i18n.T(ctx, "common.required_fields"))
	}

	// Validate branch exists and belongs to firm
	var branch models.CaseBranch
	if err := db.DB.First(&branch, "id = ? AND firm_id = ?", branchID, *user.FirmID).Error; err != nil {
		return c.String(http.StatusBadRequest, i18n.T(ctx, "common.not_found"))
	}

	// Parse order
	order := 0
	if orderStr != "" {
		if o, err := strconv.Atoi(orderStr); err == nil {
			order = o
		}
	}

	// Parse duration
	var duration *int
	if durationStr != "" {
		if d, err := strconv.Atoi(durationStr); err == nil {
			duration = &d
		}
	}

	subtype := models.CaseSubtype{
		FirmID:              *user.FirmID,
		BranchID:            branchID,
		Country:             branch.Country,
		Code:                strings.ToUpper(code),
		Name:                name,
		Description:         description,
		Order:               order,
		IsActive:            true,
		IsSystem:            false,
		TypicalDurationDays: duration,
		ComplexityLevel:     complexity,
	}

	if err := db.DB.Create(&subtype).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return c.String(http.StatusBadRequest, i18n.T(ctx, "common.duplicate_code"))
		}
		return c.String(http.StatusInternalServerError, i18n.T(ctx, "common.error"))
	}

	// Log audit
	services.LogAuditEvent(db.DB, services.AuditContext{
		UserID:    user.ID,
		UserName:  user.Name,
		UserRole:  user.Role,
		FirmID:    firm.ID,
		FirmName:  firm.Name,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}, models.AuditActionCreate, "case_subtype", subtype.ID, subtype.Name, "Created case subtype", nil, subtype)

	// Return refreshed subtypes list
	return GetSubtypesTabHandler(c)
}

// UpdateSubtypeHandler updates an existing subtype
func UpdateSubtypeHandler(c echo.Context) error {
	ctx := c.Request().Context()
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	subtypeID := c.Param("id")

	var subtype models.CaseSubtype
	if err := db.DB.First(&subtype, "id = ? AND firm_id = ?", subtypeID, *user.FirmID).Error; err != nil {
		return c.String(http.StatusNotFound, i18n.T(ctx, "common.not_found"))
	}

	// Capture old values for audit
	oldValues := map[string]interface{}{
		"name":        subtype.Name,
		"code":        subtype.Code,
		"description": subtype.Description,
	}

	// Get form values
	name := strings.TrimSpace(c.FormValue("name"))
	code := strings.TrimSpace(c.FormValue("code"))
	description := strings.TrimSpace(c.FormValue("description"))
	orderStr := c.FormValue("order")
	durationStr := c.FormValue("typical_duration_days")
	complexity := c.FormValue("complexity_level")

	// Validate required fields
	if name == "" || code == "" {
		return c.String(http.StatusBadRequest, i18n.T(ctx, "common.required_fields"))
	}

	// Parse order
	order := subtype.Order
	if orderStr != "" {
		if o, err := strconv.Atoi(orderStr); err == nil {
			order = o
		}
	}

	// Parse duration
	var duration *int
	if durationStr != "" {
		if d, err := strconv.Atoi(durationStr); err == nil {
			duration = &d
		}
	}

	// Update fields
	subtype.Name = name
	subtype.Code = strings.ToUpper(code)
	subtype.Description = description
	subtype.Order = order
	subtype.TypicalDurationDays = duration
	subtype.ComplexityLevel = complexity

	if err := db.DB.Save(&subtype).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return c.String(http.StatusBadRequest, i18n.T(ctx, "common.duplicate_code"))
		}
		return c.String(http.StatusInternalServerError, i18n.T(ctx, "common.error"))
	}

	// Log audit
	services.LogAuditEvent(db.DB, services.AuditContext{
		UserID:    user.ID,
		UserName:  user.Name,
		UserRole:  user.Role,
		FirmID:    firm.ID,
		FirmName:  firm.Name,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}, models.AuditActionUpdate, "case_subtype", subtype.ID, subtype.Name, "Updated case subtype", oldValues, subtype)

	// Return refreshed subtypes list
	return GetSubtypesTabHandler(c)
}

// ToggleSubtypeActiveHandler toggles the active status of a subtype
func ToggleSubtypeActiveHandler(c echo.Context) error {
	ctx := c.Request().Context()
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	subtypeID := c.Param("id")

	var subtype models.CaseSubtype
	if err := db.DB.First(&subtype, "id = ? AND firm_id = ?", subtypeID, *user.FirmID).Error; err != nil {
		return c.String(http.StatusNotFound, i18n.T(ctx, "common.not_found"))
	}

	oldActive := subtype.IsActive
	subtype.IsActive = !subtype.IsActive
	if err := db.DB.Save(&subtype).Error; err != nil {
		return c.String(http.StatusInternalServerError, i18n.T(ctx, "common.error"))
	}

	// Log audit
	action := "Activated case subtype"
	if !subtype.IsActive {
		action = "Deactivated case subtype"
	}
	services.LogAuditEvent(db.DB, services.AuditContext{
		UserID:    user.ID,
		UserName:  user.Name,
		UserRole:  user.Role,
		FirmID:    firm.ID,
		FirmName:  firm.Name,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}, models.AuditActionUpdate, "case_subtype", subtype.ID, subtype.Name, action, map[string]interface{}{"is_active": oldActive}, map[string]interface{}{"is_active": subtype.IsActive})

	// Return refreshed subtypes list
	return GetSubtypesTabHandler(c)
}

// DeleteSubtypeHandler soft deletes a subtype (non-system only)
func DeleteSubtypeHandler(c echo.Context) error {
	ctx := c.Request().Context()
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	subtypeID := c.Param("id")

	var subtype models.CaseSubtype
	if err := db.DB.First(&subtype, "id = ? AND firm_id = ?", subtypeID, *user.FirmID).Error; err != nil {
		return c.String(http.StatusNotFound, i18n.T(ctx, "common.not_found"))
	}

	// Prevent deletion of system subtypes
	if subtype.IsSystem {
		return c.String(http.StatusForbidden, i18n.T(ctx, "settings.classifications.cannot_delete_system"))
	}

	if err := db.DB.Delete(&subtype).Error; err != nil {
		return c.String(http.StatusInternalServerError, i18n.T(ctx, "common.error"))
	}

	// Log audit
	services.LogAuditEvent(db.DB, services.AuditContext{
		UserID:    user.ID,
		UserName:  user.Name,
		UserRole:  user.Role,
		FirmID:    firm.ID,
		FirmName:  firm.Name,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}, models.AuditActionDelete, "case_subtype", subtype.ID, subtype.Name, "Deleted case subtype", subtype, nil)

	// Return refreshed subtypes list
	return GetSubtypesTabHandler(c)
}
