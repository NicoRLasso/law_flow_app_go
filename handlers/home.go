package handlers

import (
	"law_flow_app_go/templates/pages"

	"github.com/labstack/echo/v4"
)

// HomeHandler handles the home page request
func HomeHandler(c echo.Context) error {
	// Render the home page template
	component := pages.Home("Law Flow App - Home")
	return component.Render(c.Request().Context(), c.Response().Writer)
}
