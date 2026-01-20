package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/services/i18n"
	"law_flow_app_go/templates/components"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// GetSubtypesTabHandler returns the initial classifications tab with domains loaded
func GetSubtypesTabHandler(c echo.Context) error {
	ctx := c.Request().Context()
	user := middleware.GetCurrentUser(c)

	// Get all domains for the firm (only domains, no relations needed yet)
	var domains []models.CaseDomain
	if err := db.DB.
		Where("firm_id = ?", *user.FirmID).
		Order("`order` ASC, name ASC").
		Find(&domains).Error; err != nil {
		return c.String(http.StatusInternalServerError, i18n.T(ctx, "common.error"))
	}

	return components.SubtypeManager(ctx, domains).Render(c.Request().Context(), c.Response().Writer)
}

// GetBranchesForDomainHandler return branches options for a domain
func GetBranchesForDomainHandler(c echo.Context) error {
	ctx := c.Request().Context()
	user := middleware.GetCurrentUser(c)
	domainID := c.QueryParam("domain_id")

	var branches []models.CaseBranch
	if domainID != "" {
		if err := db.DB.
			Where("firm_id = ? AND domain_id = ?", *user.FirmID, domainID).
			Order("`order` ASC, name ASC").
			Find(&branches).Error; err != nil {
			return c.String(http.StatusInternalServerError, i18n.T(ctx, "common.error"))
		}
	}

	return components.BranchOptions(ctx, branches, nil).Render(c.Request().Context(), c.Response().Writer)
}

// GetSubtypeOptionsHandler return valid subtypes for a branch as a multi-select dropdown
func GetSubtypeOptionsHandler(c echo.Context) error {
	ctx := c.Request().Context()
	user := middleware.GetCurrentUser(c)
	branchID := c.QueryParam("branch_id")

	// If no branch selected, return empty hint in component
	if branchID == "" {
		return components.SubtypeCheckboxes(ctx, []models.CaseSubtype{}, []string{}).Render(ctx, c.Response().Writer)
	}

	var subtypes []models.CaseSubtype
	if err := db.DB.
		Where("firm_id = ? AND branch_id = ?", *user.FirmID, branchID).
		Where("is_active = ?", true).
		Order("`order` ASC, name ASC").
		Find(&subtypes).Error; err != nil {
		return c.String(http.StatusInternalServerError, i18n.T(ctx, "common.error"))
	}

	// We don't have selected IDs here (this endpoint is for fetching options on change)
	// If needed for edit, we might need to pass them or let the frontend handle re-selection if possible
	// But usually this is used when changing branches, so selection is cleared.
	// For initial load in edit modal, the component is rendered directly with selected IDs.
	return components.SubtypeCheckboxes(ctx, subtypes, []string{}).Render(ctx, c.Response().Writer)
}

// GetSubtypesForBranchHandler returns the table of subtypes for a branch
func GetSubtypesForBranchHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	branchID := c.QueryParam("branch_id")

	return renderSubtypesList(c, *user.FirmID, branchID)
}

// GetSubtypeCheckboxesHandler returns the checkbox list of subtypes for a branch
func GetSubtypeCheckboxesHandler(c echo.Context) error {
	ctx := c.Request().Context()
	user := middleware.GetCurrentUser(c)
	branchID := c.QueryParam("branch_id")

	// Optional: get currently selected IDs if editing (not implemented in this flow as modal manages checked state via JS or pre-render,
	// but strictly speaking subsequent fetches lose that state unless passed back.
	// For now, we assume this is just for fetching options when changing branch in Create/Edit modal).
	var selectedIDs []string

	if branchID == "" {
		return components.SubtypeCheckboxes(ctx, []models.CaseSubtype{}, selectedIDs).Render(ctx, c.Response().Writer)
	}

	var subtypes []models.CaseSubtype
	if err := db.DB.
		Where("firm_id = ? AND branch_id = ?", *user.FirmID, branchID).
		Order("name ASC").
		Find(&subtypes).Error; err != nil {
		return c.String(http.StatusInternalServerError, i18n.T(ctx, "common.error"))
	}

	return components.SubtypeCheckboxes(ctx, subtypes, selectedIDs).Render(ctx, c.Response().Writer)
}

// renderSubtypesList is a helper to fetch subtypes and render the table component
func renderSubtypesList(c echo.Context, firmID, branchID string) error {
	ctx := c.Request().Context()

	// If no branch selected, return empty or hint
	if branchID == "" {
		return c.NoContent(http.StatusOK)
	}

	var subtypes []models.CaseSubtype
	// Fetch all subtypes (active and inactive) for the branch
	if err := db.DB.
		Where("firm_id = ?", firmID).
		Where("branch_id = ?", branchID).
		Order("`order` ASC, name ASC").
		Find(&subtypes).Error; err != nil {
		return c.String(http.StatusInternalServerError, i18n.T(ctx, "common.error"))
	}

	return components.SubtypesTable(ctx, subtypes, branchID).Render(ctx, c.Response().Writer)
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

// GetSubtypeViewHandler returns the modal view for a subtype
func GetSubtypeViewHandler(c echo.Context) error {
	ctx := c.Request().Context()
	user := middleware.GetCurrentUser(c)
	subtypeID := c.Param("id")

	var subtype models.CaseSubtype
	if err := db.DB.
		Preload("Branch").
		Preload("Branch.Domain").
		First(&subtype, "id = ? AND firm_id = ?", subtypeID, *user.FirmID).Error; err != nil {
		return c.String(http.StatusNotFound, i18n.T(ctx, "common.not_found"))
	}

	return components.SubtypeViewModal(ctx, subtype).Render(c.Request().Context(), c.Response().Writer)
}

// CreateSubtypeHandler creates a new subtype
func CreateSubtypeHandler(c echo.Context) error {
	ctx := c.Request().Context()
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	branchID := c.FormValue("branch_id")
	name := strings.TrimSpace(c.FormValue("name"))
	codeRaw := strings.TrimSpace(c.FormValue("code"))
	description := strings.TrimSpace(c.FormValue("description"))

	// Sanitize Code: Uppercase, letters only (we'll just replace spaces with _, usually strictly letters only means alphanumeric but user said letters and spaces->_)
	// Interpreting "Code * solo reciba letras y que los espacios los convierta en _" literally:
	// If JS handles the input restriction, backend should just ensure format.
	// We'll replace spaces with _ and uppercase it.
	code := strings.ToUpper(strings.ReplaceAll(codeRaw, " ", "_"))

	// Validate required fields
	if branchID == "" || name == "" || code == "" {
		return c.String(http.StatusBadRequest, i18n.T(ctx, "common.required_fields"))
	}

	// Validate branch exists and belongs to firm
	var branch models.CaseBranch
	if err := db.DB.First(&branch, "id = ? AND firm_id = ?", branchID, *user.FirmID).Error; err != nil {
		return c.String(http.StatusBadRequest, i18n.T(ctx, "common.not_found"))
	}

	// Auto-increment Order
	var maxOrder int
	// Find max order for this branch
	db.DB.Model(&models.CaseSubtype{}).
		Where("firm_id = ? AND branch_id = ?", *user.FirmID, branchID).
		Select("COALESCE(MAX(`order`), 0)").
		Scan(&maxOrder)

	newOrder := maxOrder + 1

	subtype := models.CaseSubtype{
		FirmID:      *user.FirmID,
		BranchID:    branchID,
		Country:     branch.Country,
		Code:        code,
		Name:        name,
		Description: description,
		Order:       newOrder,
		IsActive:    true,
		IsSystem:    false,
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

	// Return refreshed subtypes list for the branch
	return renderSubtypesList(c, *user.FirmID, branchID)
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
	codeRaw := strings.TrimSpace(c.FormValue("code"))
	description := strings.TrimSpace(c.FormValue("description"))

	// Sanitize Code
	code := strings.ToUpper(strings.ReplaceAll(codeRaw, " ", "_"))

	// Validate required fields
	if name == "" || code == "" {
		return c.String(http.StatusBadRequest, i18n.T(ctx, "common.required_fields"))
	}

	// Update fields
	subtype.Name = name
	subtype.Code = code
	subtype.Description = description
	// Order is preserved (not updated via this form)

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
	return renderSubtypesList(c, *user.FirmID, subtype.BranchID)
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
	return renderSubtypesList(c, *user.FirmID, subtype.BranchID)
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

	branchID := subtype.BranchID // Save for return

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
	return renderSubtypesList(c, *user.FirmID, branchID)
}
