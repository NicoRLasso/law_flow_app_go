package handlers

import (
	"law_flow_app_go/middleware"
	"law_flow_app_go/templates/pages"

	"github.com/labstack/echo/v4"
)

// DashboardHandler renders the main dashboard
func DashboardHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	component := pages.Dashboard("Dashboard | Law Flow", user, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
