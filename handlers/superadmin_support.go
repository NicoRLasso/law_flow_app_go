package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/templates/superadmin"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SuperadminSupportPageHandler renders the list of all support tickets
func SuperadminSupportPageHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	var tickets []models.SupportTicket
	// Fetch all tickets, ordered by latest first, preload User
	if err := db.DB.Preload("User").Order("created_at desc").Find(&tickets).Error; err != nil {
		c.Logger().Error("Failed to fetch support tickets:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load tickets")
	}

	component := superadmin.Support(c.Request().Context(), "Support Tickets | Superadmin", csrfToken, user, tickets)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminResolveTicketHandler marks a ticket as resolved
func SuperadminResolveTicketHandler(c echo.Context) error {
	id := c.Param("id")

	if err := db.DB.Model(&models.SupportTicket{}).Where("id = ?", id).Update("status", "resolved").Error; err != nil {
		c.Logger().Error("Failed to resolve ticket:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update ticket")
	}

	return c.Redirect(http.StatusSeeOther, "/superadmin/support")
}
