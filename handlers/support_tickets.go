package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/templates/components"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetSupportTicketsHandler fetches support tickets via HTMX
func GetSupportTicketsHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	if user == nil || firm == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	var tickets []models.SupportTicket
	query := db.DB.Model(&models.SupportTicket{}).Preload("User").Preload("RespondedBy")

	// Filter by Status
	status := c.QueryParam("status")
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	isFirmAdmin := user.Role == "admin"

	if isFirmAdmin {
		// Admin: See all tickets from users in the same firm
		// We join with users table to filter by firm_id
		query = query.Joins("JOIN users ON users.id = support_tickets.user_id").
			Where("users.firm_id = ?", firm.ID)

		// Filter by User (only for admin)
		filterUserID := c.QueryParam("user_id")
		if filterUserID != "" && filterUserID != "all" {
			query = query.Where("support_tickets.user_id = ?", filterUserID)
		}
	} else {
		// Non-Admin: See only own tickets
		query = query.Where("support_tickets.user_id = ?", user.ID)
	}

	// Order by latest first
	if err := query.Order("support_tickets.created_at desc").Find(&tickets).Error; err != nil {
		c.Logger().Error("Failed to fetch tickets:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load tickets")
	}

	component := components.SupportTicketsList(c.Request().Context(), tickets, isFirmAdmin)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
