package handlers

import (
	"law_flow_app_go/middleware"
	"law_flow_app_go/templates/pages"

	"github.com/labstack/echo/v4"
)

// ToolsPageHandler renders the tools page
func ToolsPageHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	csrfToken := middleware.GetCSRFToken(c)

	component := pages.Tools(c.Request().Context(), "Tools | LexLegal Cloud", csrfToken, currentUser, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
