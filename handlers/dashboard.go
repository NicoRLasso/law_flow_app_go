package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/templates/pages"
	"time"

	"github.com/labstack/echo/v4"
)

// DashboardHandler renders the main dashboard
func DashboardHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	csrfToken := middleware.GetCSRFToken(c)

	stats := pages.DashboardStats{}
	db := db.DB

	// 1. Active Cases (Status = OPEN)
	if err := db.Model(&models.Case{}).
		Where("firm_id = ? AND status = ?", firm.ID, models.CaseStatusOpen).
		Count(&stats.ActiveCases).Error; err != nil {
		c.Logger().Error("Failed to count active cases:", err)
	}

	// 2. Total Clients (Role = client)
	// Note: We count users with role 'client' associated with the firm
	// OR users that are clients in cases of this firm (if we want to be more specific, but role check is faster)
	if err := db.Model(&models.User{}).
		Where("firm_id = ? AND role = 'client'", firm.ID).
		Count(&stats.TotalClients).Error; err != nil {
		c.Logger().Error("Failed to count clients:", err)
	}

	// 3. Completed This Month (Status = CLOSED, StatusChangedAt in current month)
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if err := db.Model(&models.Case{}).
		Where("firm_id = ? AND status = ? AND status_changed_at >= ?", firm.ID, models.CaseStatusClosed, startOfMonth).
		Count(&stats.CompletedMonthly).Error; err != nil {
		c.Logger().Error("Failed to count monthly completed cases:", err)
	}

	// 4. Pending Tasks / Upcoming Appointments (Next 7 days)
	// We'll treat appointments as "tasks" for now
	nextWeek := now.AddDate(0, 0, 7)
	if err := db.Model(&models.Appointment{}).
		Where("firm_id = ? AND start_time BETWEEN ? AND ? AND status IN ?",
			firm.ID, now, nextWeek, []string{models.AppointmentStatusScheduled, models.AppointmentStatusConfirmed}).
		Count(&stats.PendingTasks).Error; err != nil {
		c.Logger().Error("Failed to count pending tasks:", err)
	}

	// 5. Recent Cases (Last 5 updated)
	if err := db.Model(&models.Case{}).
		Where("firm_id = ?", firm.ID).
		Preload("Client"). // Preload Client to show name
		Order("updated_at DESC").
		Limit(5).
		Find(&stats.RecentCases).Error; err != nil {
		c.Logger().Error("Failed to fetch recent cases:", err)
	}

	// 6. Upcoming Appointments (Next 5)
	if err := db.Model(&models.Appointment{}).
		Where("firm_id = ? AND start_time > ? AND status IN ?",
					firm.ID, now, []string{models.AppointmentStatusScheduled, models.AppointmentStatusConfirmed}).
		Preload("Client"). // Preload Client/User info if needed
		Preload("Case").   // Preload Case info if useful
		Order("start_time ASC").
		Limit(5).
		Find(&stats.UpcomingAppointments).Error; err != nil {
		c.Logger().Error("Failed to fetch upcoming appointments:", err)
	}

	component := pages.Dashboard(c.Request().Context(), "Dashboard | Law Flow", csrfToken, user, firm, stats)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
