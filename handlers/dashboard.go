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
	csrfToken := middleware.GetCSRFToken(c)

	component := pages.Dashboard(c.Request().Context(), "Dashboard | Law Flow", csrfToken, user, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
