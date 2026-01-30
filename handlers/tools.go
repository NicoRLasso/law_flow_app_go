package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"

	"github.com/labstack/echo/v4"
)

// ToolsPageHandler renders the tools page
func ToolsPageHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	csrfToken := middleware.GetCSRFToken(c)

	// Fetch Colombia's departments for the radicado builder
	departments, err := services.GetDepartmentsByCountryCode(db.DB, "COL")
	if err != nil {
		departments = []models.Department{} // Empty list on error
	}

	component := pages.Tools(c.Request().Context(), "Tools | LexLegal Cloud", csrfToken, currentUser, firm, departments)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
