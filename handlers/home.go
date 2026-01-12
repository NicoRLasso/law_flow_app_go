package handlers

import (
	"law_flow_app_go/templates/pages"

	"github.com/labstack/echo/v4"
)

// LandingHandler handles the landing page request with Kinetic Typography design
func LandingHandler(c echo.Context) error {
	// Render the landing page template
	component := pages.Landing("Law Flow - Modern Legal Practice Management")
	return component.Render(c.Request().Context(), c.Response().Writer)
}
