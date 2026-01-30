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

// Helper to verify case access and return the case
func verifyCaseAccess(c echo.Context, caseID string) (*models.Case, error) {
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	query := db.DB.Where("firm_id = ? AND id = ?", currentFirm.ID, caseID)

	if currentUser.Role == "lawyer" {
		query = query.Where(
			db.DB.Where("assigned_to_id = ?", currentUser.ID).
				Or("EXISTS (SELECT 1 FROM case_collaborators WHERE case_collaborators.case_id = cases.id AND case_collaborators.user_id = ?)", currentUser.ID),
		)
	} else if currentUser.Role == "client" {
		query = query.Where("client_id = ?", currentUser.ID)
	}

	var caseRecord models.Case
	if err := query.First(&caseRecord).Error; err != nil {
		return nil, err
	}
	return &caseRecord, nil
}

// GetCaseMilestonesHandler returns the list of milestones for a case
func GetCaseMilestonesHandler(c echo.Context) error {
	caseID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	_, err := verifyCaseAccess(c, caseID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	milestones, err := services.GetMilestonesByCase(db.DB, caseID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch milestones")
	}

	progress, err := services.GetCaseMilestoneProgress(db.DB, caseID)
	if err != nil {
		progress = &services.MilestoneProgress{}
	}

	component := partials.CaseMilestoneList(c.Request().Context(), milestones, progress, currentUser.Role != "client", caseID)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// CreateCaseMilestoneHandler adds a new milestone to a case
func CreateCaseMilestoneHandler(c echo.Context) error {
	caseID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	if currentUser.Role == "client" {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	_, err := verifyCaseAccess(c, caseID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	title := c.FormValue("title")
	description := c.FormValue("description")
	dueDateVal := c.FormValue("due_date")

	if title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Title is required")
	}

	var maxOrder int
	db.DB.Model(&models.CaseMilestone{}).Where("case_id = ?", caseID).Select("COALESCE(MAX(sort_order), 0)").Scan(&maxOrder)

	milestone := models.CaseMilestone{
		FirmID:      currentFirm.ID,
		CaseID:      caseID,
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

	// Trigger timeline and summary refresh
	c.Response().Header().Set("HX-Trigger", "refreshTimeline,refreshSummary")

	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionCreate,
		"CaseMilestone", milestone.ID, milestone.Title,
		"Case milestone created", nil, milestone)

	return GetCaseMilestonesHandler(c)
}

// UpdateCaseMilestoneHandler updates an existing milestone
func UpdateCaseMilestoneHandler(c echo.Context) error {
	caseID := c.Param("id")
	milestoneID := c.Param("mid")
	currentFirm := middleware.GetCurrentFirm(c)

	var milestone models.CaseMilestone
	if err := db.DB.Where("firm_id = ? AND id = ? AND case_id = ?", currentFirm.ID, milestoneID, caseID).First(&milestone).Error; err != nil {
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

	// Trigger timeline and summary refresh
	c.Response().Header().Set("HX-Trigger", "refreshTimeline,refreshSummary")

	return GetCaseMilestonesHandler(c)
}

// CompleteCaseMilestoneHandler toggles milestone completion
func CompleteCaseMilestoneHandler(c echo.Context) error {
	caseID := c.Param("id")
	milestoneID := c.Param("mid")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	_, err := verifyCaseAccess(c, caseID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	var milestone models.CaseMilestone
	if err := db.DB.Where("firm_id = ? AND id = ? AND case_id = ?", currentFirm.ID, milestoneID, caseID).First(&milestone).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Milestone not found")
	}

	isComplete := c.FormValue("is_complete") == "true"

	if isComplete {
		err = services.CompleteCaseMilestone(db.DB, milestoneID, currentUser.ID)
	} else {
		err = services.ResetCaseMilestone(db.DB, milestoneID)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update status")
	}

	// Trigger timeline and summary refresh
	c.Response().Header().Set("HX-Trigger", "refreshTimeline,refreshSummary")

	auditCtx := middleware.GetAuditContext(c)
	msg := "Case milestone completed"
	if !isComplete {
		msg = "Case milestone reset"
	}
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionUpdate,
		"CaseMilestone", milestone.ID, milestone.Title,
		msg, nil, nil)

	return GetCaseMilestonesHandler(c)
}

// DeleteCaseMilestoneHandler deletes a milestone
func DeleteCaseMilestoneHandler(c echo.Context) error {
	caseID := c.Param("id")
	milestoneID := c.Param("mid")
	currentFirm := middleware.GetCurrentFirm(c)

	var milestone models.CaseMilestone
	if err := db.DB.Where("firm_id = ? AND id = ? AND case_id = ?", currentFirm.ID, milestoneID, caseID).First(&milestone).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Milestone not found")
	}

	if err := db.DB.Delete(&milestone).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete milestone")
	}

	// Trigger timeline and summary refresh
	c.Response().Header().Set("HX-Trigger", "refreshTimeline,refreshSummary")

	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionDelete,
		"CaseMilestone", milestone.ID, milestone.Title,
		"Case milestone deleted", nil, nil)

	return GetCaseMilestonesHandler(c)
}

// ReorderCaseMilestonesHandler handles drag-and-drop reordering
func ReorderCaseMilestonesHandler(c echo.Context) error {
	caseID := c.Param("id")
	c.Request().ParseForm()
	ids := c.Request().Form["ids[]"]

	if len(ids) > 0 {
		if err := services.ReorderCaseMilestones(db.DB, caseID, ids); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to reorder")
		}
	}

	// Trigger timeline and summary refresh
	c.Response().Header().Set("HX-Trigger", "refreshTimeline,refreshSummary")

	return GetCaseMilestonesHandler(c)
}
