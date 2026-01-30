package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/superadmin"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SuperadminSecurityDashboardHandler renders the security monitoring dashboard
func SuperadminSecurityDashboardHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	csrfToken := middleware.GetCSRFToken(c)

	// 1. Fetch active alerts from memory monitor
	alerts := services.Monitor.GetRecentAlerts()

	// 2. Fetch recent security-related audit logs from DB
	// We look for logs where ResourceType is 'SECURITY_EVENT' or Action is related to security
	var logs []models.AuditLog
	if err := db.DB.Where("resource_type = ? OR action IN ?", "SECURITY_EVENT", []string{"LOGIN", "LOGOUT", "FAILED_LOGIN", "ACCOUNT_LOCKED"}).
		Order("created_at DESC").
		Limit(20).
		Find(&logs).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch security logs")
	}

	// Render
	component := superadmin.SecurityDashboard(c.Request().Context(), "Security Dashboard | Superadmin", csrfToken, user, "/superadmin/security", alerts, logs)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
