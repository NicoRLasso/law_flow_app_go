package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/templates/pages"
	"net/http"

	"github.com/labstack/echo/v4"
)

// PublicCaseRequestSuccessHandler renders the success page after form submission
func PublicCaseRequestSuccessHandler(c echo.Context) error {
	slug := c.Param("slug")

	// Find firm by slug
	var firm models.Firm
	if err := db.DB.Where("slug = ?", slug).First(&firm).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Firm not found")
	}

	// Render success template
	csrfToken := middleware.GetCSRFToken(c)
	component := pages.CaseRequestSuccess(c.Request().Context(), csrfToken, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
