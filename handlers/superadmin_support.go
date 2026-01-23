package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/templates/superadmin"
	"net/http"
	"time"

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

// SuperadminSupportDetailHandler renders a single support ticket
func SuperadminSupportDetailHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	csrfToken := middleware.GetCSRFToken(c)
	id := c.Param("id")

	var ticket models.SupportTicket
	if err := db.DB.Preload("User").Preload("RespondedBy").First(&ticket, "id = ?", id).Error; err != nil {
		c.Logger().Error("Failed to fetch ticket:", err)
		return echo.NewHTTPError(http.StatusNotFound, "Ticket not found")
	}

	component := superadmin.SupportDetail(c.Request().Context(), "Ticket Details | Superadmin", csrfToken, user, ticket)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// SuperadminUpdateTicketStatusHandler updates the ticket status
func SuperadminUpdateTicketStatusHandler(c echo.Context) error {
	id := c.Param("id")
	status := c.FormValue("status")
	// allowed statuses: open, in_progress, resolved, closed
	validStatuses := map[string]bool{
		"open":        true,
		"in_progress": true,
		"resolved":    true,
		"closed":      true,
	}

	if !validStatuses[status] {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid status")
	}

	if err := db.DB.Model(&models.SupportTicket{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		c.Logger().Error("Failed to update ticket status:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update ticket")
	}

	return c.Redirect(http.StatusSeeOther, "/superadmin/support/"+id)
}

// SuperadminTakeTicketHandler assigns the ticket to the current user and sets status to in_progress
func SuperadminTakeTicketHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	id := c.Param("id")

	updates := map[string]interface{}{
		"responded_by_id": user.ID,
		"status":          "in_progress",
	}

	if err := db.DB.Model(&models.SupportTicket{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.Logger().Error("Failed to take ticket:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to take ticket")
	}

	return c.Redirect(http.StatusSeeOther, "/superadmin/support/"+id)
}

// SuperadminReplyTicketHandler handles the reply submission
func SuperadminReplyTicketHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	id := c.Param("id")
	response := c.FormValue("response")

	if response == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Response cannot be empty")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"response":        response,
		"responded_by_id": user.ID,
		"responded_at":    now,
		"status":          "resolved", // Auto-resolve on reply? Or leave as in_progress? Let's auto-resolve for now or make it optional. User request implies "response... resolved". Let's stick to update separately or assume reply implies progress/resolve.

	}
	// Explicitly set status to resolved when replying, as per common support flows.
	updates["status"] = "resolved"

	if err := db.DB.Model(&models.SupportTicket{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.Logger().Error("Failed to save ticket response:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save response")
	}

	// TODO: Notify user via email about the reply? (Optional enhancement)

	return c.Redirect(http.StatusSeeOther, "/superadmin/support/"+id)
}
