package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"law_flow_app_go/templates/partials"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// AuditLogsPageHandler renders the audit logs dashboard page
func AuditLogsPageHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	// lang := middleware.GetLocale(c) // Not needed as we pass context

	return pages.AuditLogs(c.Request().Context(), user, firm).Render(c.Request().Context(), c.Response())
}

// GetAuditLogsHandler returns filtered and paginated audit logs
func GetAuditLogsHandler(c echo.Context) error {
	firm := middleware.GetCurrentFirm(c)
	if firm == nil {
		return echo.NewHTTPError(http.StatusForbidden, "No firm context")
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 20

	// Parse filters
	filters := services.AuditLogFilters{
		UserID:       c.QueryParam("user_id"),
		ResourceType: c.QueryParam("resource_type"),
		Action:       c.QueryParam("action"),
		SearchQuery:  c.QueryParam("search"),
	}

	if dateFrom := c.QueryParam("date_from"); dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			filters.DateFrom = t
		}
	}
	if dateTo := c.QueryParam("date_to"); dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			filters.DateTo = t.Add(24*time.Hour - time.Second) // End of day
		}
	}

	// Default to today if no date filters provided
	if filters.DateFrom.IsZero() && filters.DateTo.IsZero() {
		now := time.Now()
		filters.DateFrom = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		filters.DateTo = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	}

	logs, total, err := services.GetFirmAuditLogs(db.DB, firm.ID, filters, page, pageSize)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch audit logs")
	}

	// lang := middleware.GetLocale(c)
	return partials.AuditLogTable(c.Request().Context(), logs, total, page, pageSize).
		Render(c.Request().Context(), c.Response())
}

// GetResourceHistoryHandler returns the audit history for a specific resource
func GetResourceHistoryHandler(c echo.Context) error {
	resourceType := c.Param("type")
	resourceID := c.Param("id")

	logs, err := services.GetResourceAuditHistory(db.DB, resourceType, resourceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch history")
	}

	// lang := middleware.GetLocale(c)
	return partials.AuditTimeline(c.Request().Context(), logs).Render(c.Request().Context(), c.Response())
}
