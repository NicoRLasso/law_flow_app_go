package handlers

import (
	"fmt"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"law_flow_app_go/templates/partials"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// GetCaseSummaryHandler returns the summary tab content for a case
func GetCaseSummaryHandler(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	caseRecord, err := services.GetCaseByID(db.DB, currentFirm.ID, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Security check for clients
	if currentUser.Role == "client" && caseRecord.ClientID != currentUser.ID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	// Fetch timeline events (using buildCaseTimeline)
	timeline := buildCaseTimeline(caseRecord)

	component := pages.CaseResumenTab(c.Request().Context(), *caseRecord, currentUser, timeline)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetCaseTimelineHandler returns paginated timeline events for a case
func GetCaseTimelineHandler(c echo.Context) error {
	id := c.Param("id")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit := 8

	currentFirm := middleware.GetCurrentFirm(c)
	caseRecord, err := services.GetCaseByID(db.DB, currentFirm.ID, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	allEvents := buildCaseTimeline(caseRecord)
	total := len(allEvents)
	totalPages := (total + limit - 1) / limit

	var events []models.TimelineEvent
	start := (page - 1) * limit
	if start < total {
		end := start + limit
		if end > total {
			end = total
		}
		events = allEvents[start:end]
	}

	// Use the generic TimelineList component
	apiURL := fmt.Sprintf("/api/cases/%s/timeline", id)
	component := partials.TimelineList(c.Request().Context(), events, page, totalPages, total, id, apiURL, "#case-timeline-events-container", limit)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// buildCaseTimeline creates a procedural timeline of case events
func buildCaseTimeline(caseRecord *models.Case) []models.TimelineEvent {
	var events []models.TimelineEvent

	// 1. Case Opened (Always First)
	events = append(events, models.TimelineEvent{
		Date:        caseRecord.OpenedAt,
		Type:        "case_opened",
		Title:       "Case Opened",
		Description: "Case was opened in the system",
	})

	// 2. Milestones (in sort_order as preloaded)
	for _, milestone := range caseRecord.Milestones {
		desc := ""
		if milestone.Description != nil {
			desc = *milestone.Description
		}
		date := caseRecord.OpenedAt
		if milestone.DueDate != nil {
			date = *milestone.DueDate
		} else if milestone.Status == models.MilestoneStatusCompleted && milestone.UpdatedAt.After(caseRecord.OpenedAt) {
			date = milestone.UpdatedAt
		}

		events = append(events, models.TimelineEvent{
			Date:        date,
			Type:        "milestone",
			Title:       milestone.Title,
			Description: desc,
			Status:      milestone.Status,
			IsCompleted: milestone.Status == models.MilestoneStatusCompleted,
		})
	}

	// 3. Case Closed (if exists)
	if caseRecord.ClosedAt != nil {
		events = append(events, models.TimelineEvent{
			Date:        *caseRecord.ClosedAt,
			Type:        "case_closed",
			Title:       "Case Closed",
			Description: "Case was closed",
			IsCompleted: true,
		})
	}

	return events
}
