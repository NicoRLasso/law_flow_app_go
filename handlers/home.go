package handlers

import (
	"law_flow_app_go/templates"

	"github.com/labstack/echo/v4"
)

// HomeHandler handles the home page request
func HomeHandler(c echo.Context) error {
	// Render the index template
	component := templates.Index("Law Flow App - Home")
	return component.Render(c.Request().Context(), c.Response().Writer)
}
