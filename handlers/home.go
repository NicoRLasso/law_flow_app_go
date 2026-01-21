package handlers

import (
	"law_flow_app_go/middleware"
	"law_flow_app_go/templates/pages"

	"github.com/labstack/echo/v4"
)

// LandingHandler handles the landing page request with Kinetic Typography design
func LandingHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	// Render the landing page template
	component := pages.Landing(c.Request().Context(), "LexLegal Cloud - Modern Legal Practice Management", csrfToken)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
