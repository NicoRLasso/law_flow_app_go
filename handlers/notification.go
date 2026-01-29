package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

func GetNotificationsHandler(c echo.Context) error {
	// Implementation for getting notifications list (HTMX partial)
	// For now, we are rendering it inline in the dashboard, so this might be for polling updates
	// But the plan asked to create it.
	// We'll return an empty 200 OK for now as the dashboard renders it initially.
	return c.NoContent(http.StatusOK)
}

func MarkNotificationReadHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	notificationID := c.Param("id")

	service := services.NewNotificationService(db.DB)
	if err := service.MarkAsRead(notificationID, user.ID, firm.ID); err != nil {
		return c.String(http.StatusInternalServerError, "Error marking as read")
	}

	// Return empty string to remove the notification from the UI (hx-swap="outerHTML")
	return c.String(http.StatusOK, "")
}

func MarkAllNotificationsReadHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	service := services.NewNotificationService(db.DB)
	if err := service.MarkAllAsRead(firm.ID, user.ID); err != nil {
		return c.String(http.StatusInternalServerError, "Error marking all as read")
	}

	// Return updated notifications section (empty if all read, or just the header with 0 count)
	// Ideally we re-render the notifications section with 0 count.
	// For simplicity, we can return an empty div or a "No new notifications" message.
	// But to keep it consistent with the template logic, we might want to return the whole section again?
	// The template logic is: if len(stats.Notifications) > 0 { ... }
	// So if we return empty string, the whole section disappears.
	return c.String(http.StatusOK, "")
}
