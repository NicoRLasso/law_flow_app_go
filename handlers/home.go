package handlers

import (
	"law_flow_app_go/middleware"
	"law_flow_app_go/templates/pages"

	"github.com/labstack/echo/v4"
)

// LandingHandler handles the landing page request with Kinetic Typography design
func LandingHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("landing")
	component := pages.Landing(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
