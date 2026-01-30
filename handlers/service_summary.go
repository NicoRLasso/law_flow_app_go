package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetServiceSummaryHandler returns the summary tab content for a service
func GetServiceSummaryHandler(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	service, err := services.GetServiceByID(db.DB, currentFirm.ID, id)
	if err != nil {
		if err == services.ErrServiceNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Service not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve service")
	}

	// Security check for clients
	if currentUser.Role == "client" && service.ClientID != currentUser.ID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	// Fetch total expenses for summary
	totalExpenses, _ := services.GetServiceTotalExpenses(db.DB, service.ID)

	// Fetch timeline events
	timeline := buildServiceTimeline(service)

	component := pages.ServiceSummaryTab(c.Request().Context(), service, currentUser, totalExpenses, timeline)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
