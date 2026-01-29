package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/partials"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// GetServiceMilestonesHandler returns the list of milestones for a service
func GetServiceMilestonesHandler(c echo.Context) error {
	serviceID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	// Verify access
	service, err := services.GetServiceByID(db.DB, currentFirm.ID, serviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Service not found")
	}

	if currentUser.Role == "client" && service.ClientID != currentUser.ID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	milestones, err := services.GetMilestonesByService(db.DB, serviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch milestones")
	}

	progress, err := services.GetMilestoneProgress(db.DB, serviceID)
	if err != nil {
		progress = &services.MilestoneProgress{} // Fallback
	}

	// Note: partials.ServiceMilestoneList will be created in Phase 4
	component := partials.ServiceMilestoneList(c.Request().Context(), milestones, progress, currentUser.Role != "client")
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// CreateMilestoneHandler adds a new milestone
func CreateMilestoneHandler(c echo.Context) error {
	serviceID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	// Verify access (Lawyer/Admin only)
	if currentUser.Role == "client" {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	title := c.FormValue("title")
	description := c.FormValue("description")
	dueDateVal := c.FormValue("due_date")

	if title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Title is required")
	}

	// Determine sort order (append to end)
	var maxOrder int
	db.DB.Model(&models.ServiceMilestone{}).Where("service_id = ?", serviceID).Select("COALESCE(MAX(sort_order), 0)").Scan(&maxOrder)

	milestone := models.ServiceMilestone{
		FirmID:      currentFirm.ID,
		ServiceID:   serviceID,
		Title:       title,
		Description: &description,
		SortOrder:   maxOrder + 1,
		Status:      models.MilestoneStatusPending,
	}

	if dueDateVal != "" {
		if t, err := time.Parse("2006-01-02", dueDateVal); err == nil {
			milestone.DueDate = &t
		}
	}

	if err := db.DB.Create(&milestone).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create milestone")
	}

	// Audit Log
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionCreate,
		"ServiceMilestone", milestone.ID, milestone.Title,
		"Milestone created", nil, milestone)

	// Return updated list
	return GetServiceMilestonesHandler(c)
}

// UpdateMilestoneHandler updates an existing milestone
func UpdateMilestoneHandler(c echo.Context) error {
	serviceID := c.Param("id")
	milestoneID := c.Param("mid")
	// currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	var milestone models.ServiceMilestone
	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, milestoneID, serviceID).First(&milestone).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Milestone not found")
	}

	milestone.Title = c.FormValue("title")

	desc := c.FormValue("description")
	milestone.Description = &desc

	if dueDateVal := c.FormValue("due_date"); dueDateVal != "" {
		if t, err := time.Parse("2006-01-02", dueDateVal); err == nil {
			milestone.DueDate = &t
		}
	} else {
		milestone.DueDate = nil
	}

	if err := db.DB.Save(&milestone).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update milestone")
	}

	// Audit Log
	// ...

	return GetServiceMilestonesHandler(c)
}

// CompleteMilestoneHandler toggles milestone completion
func CompleteMilestoneHandler(c echo.Context) error {
	serviceID := c.Param("id")
	milestoneID := c.Param("mid")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	var milestone models.ServiceMilestone
	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, milestoneID, serviceID).First(&milestone).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Milestone not found")
	}

	isComplete := c.FormValue("is_complete") == "true"

	var err error
	if isComplete {
		err = services.CompleteMilestone(db.DB, milestoneID, currentUser.ID)
	} else {
		err = services.ResetMilestone(db.DB, milestoneID)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update status")
	}

	// Audit Log
	action := models.AuditActionUpdate
	msg := "Milestone completed"
	if !isComplete {
		msg = "Milestone reset"
	}
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, action,
		"ServiceMilestone", milestone.ID, milestone.Title,
		msg, nil, nil)

	return GetServiceMilestonesHandler(c)
}

// DeleteMilestoneHandler deletes a milestone
func DeleteMilestoneHandler(c echo.Context) error {
	serviceID := c.Param("id")
	milestoneID := c.Param("mid")
	currentFirm := middleware.GetCurrentFirm(c)

	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, milestoneID, serviceID).Delete(&models.ServiceMilestone{}).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete milestone")
	}

	// Audit Log
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionDelete,
		"ServiceMilestone", milestoneID, "",
		"Milestone deleted", nil, nil)

	return GetServiceMilestonesHandler(c)
}

// ReorderMilestonesHandler handles drag-and-drop reordering
func ReorderMilestonesHandler(c echo.Context) error {
	serviceID := c.Param("id")
	// Parse ids from form: ids[]=1&ids[]=2...
	c.Request().ParseForm()
	ids := c.Request().Form["ids[]"]

	if len(ids) > 0 {
		if err := services.ReorderMilestones(db.DB, serviceID, ids); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to reorder")
		}
	}

	return GetServiceMilestonesHandler(c)
}

// Add this to GetServiceMilestonesHandler if needed to handle reordering UI updates specific logic
