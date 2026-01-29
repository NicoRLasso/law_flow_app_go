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
	activeCasesQuery := db.Model(&models.Case{}).Where("firm_id = ? AND status = ?", firm.ID, models.CaseStatusOpen)
	if user.Role == "client" {
		activeCasesQuery = activeCasesQuery.Where("client_id = ?", user.ID)
	}

	if err := activeCasesQuery.Count(&stats.ActiveCases).Error; err != nil {
		c.Logger().Error("Failed to count active cases:", err)
	}

	// 2. Total Clients (Role = client)
	// Note: We count users with role 'client' associated with the firm
	// OR users that are clients in cases of this firm (if we want to be more specific, but role check is faster)
	// Only show for non-client users
	if user.Role != "client" {
		if err := db.Model(&models.User{}).
			Where("firm_id = ? AND role = 'client'", firm.ID).
			Count(&stats.TotalClients).Error; err != nil {
			c.Logger().Error("Failed to count clients:", err)
		}
	}

	// 3. Completed This Month (Status = CLOSED, StatusChangedAt in current month)
	// Only show for non-client users or adapt for client's completed cases
	if user.Role != "client" {
		now := time.Now()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		if err := db.Model(&models.Case{}).
			Where("firm_id = ? AND status = ? AND status_changed_at >= ?", firm.ID, models.CaseStatusClosed, startOfMonth).
			Count(&stats.CompletedMonthly).Error; err != nil {
			c.Logger().Error("Failed to count monthly completed cases:", err)
		}
	} else {
		// For clients, maybe show THEIR completed cases this month? Or just hide it.
		// Let's zero it out for now as per plan "hide or zero out"
		stats.CompletedMonthly = 0
	}

	// 4. Pending Tasks / Upcoming Appointments (Next 7 days)
	// We'll treat appointments as "tasks" for now
	now := time.Now()
	nextWeek := now.AddDate(0, 0, 7)

	pendingTasksQuery := db.Model(&models.Appointment{}).
		Where("firm_id = ? AND start_time BETWEEN ? AND ? AND status IN ?",
			firm.ID, now, nextWeek, []string{models.AppointmentStatusScheduled, models.AppointmentStatusConfirmed})

	if user.Role == "client" {
		// Clients see their own appointments
		pendingTasksQuery = pendingTasksQuery.Where("client_id = ?", user.ID)
	} else if user.Role == "lawyer" {
		// Lawyers see appointments assigned to them or where they are attendee
		pendingTasksQuery = pendingTasksQuery.Where("lawyer_id = ?", user.ID)
	}

	if err := pendingTasksQuery.Count(&stats.PendingTasks).Error; err != nil {
		c.Logger().Error("Failed to count pending tasks:", err)
	}

	// 5. Recent Cases (Last 5 updated)
	recentCasesQuery := db.Model(&models.Case{}).Where("firm_id = ?", firm.ID)

	if user.Role == "client" {
		recentCasesQuery = recentCasesQuery.Where("client_id = ?", user.ID)
	} else if user.Role == "lawyer" {
		recentCasesQuery = recentCasesQuery.Where(
			db.Where("assigned_to_id = ?", user.ID).
				Or("EXISTS (SELECT 1 FROM case_collaborators WHERE case_collaborators.case_id = cases.id AND case_collaborators.user_id = ?)", user.ID),
		)
	}

	if err := recentCasesQuery.
		Preload("Client"). // Preload Client to show name
		Order("updated_at DESC").
		Limit(5).
		Find(&stats.RecentCases).Error; err != nil {
		c.Logger().Error("Failed to fetch recent cases:", err)
	}

	// 6. Upcoming Appointments (Next 5)
	upcomingApptsQuery := db.Model(&models.Appointment{}).
		Where("firm_id = ? AND start_time > ? AND status IN ?",
			firm.ID, now, []string{models.AppointmentStatusScheduled, models.AppointmentStatusConfirmed})

	if user.Role == "client" {
		upcomingApptsQuery = upcomingApptsQuery.Where("client_id = ?", user.ID)
	} else if user.Role == "lawyer" {
		upcomingApptsQuery = upcomingApptsQuery.Where("lawyer_id = ?", user.ID)
	}

	if err := upcomingApptsQuery.
		Preload("Client"). // Preload Client/User info if needed
		Preload("Case").   // Preload Case info if useful
		Order("start_time ASC").
		Limit(5).
		Find(&stats.UpcomingAppointments).Error; err != nil {
		c.Logger().Error("Failed to fetch upcoming appointments:", err)
	}

	// Fetch unread notifications
	var notifications []models.Notification
	if err := db.Where("firm_id = ? AND (user_id IS NULL OR user_id = ?) AND read_at IS NULL", firm.ID, user.ID).
		Order("created_at DESC").
		Limit(5).
		Find(&notifications).Error; err != nil {
		c.Logger().Error("Failed to fetch notifications:", err)
	}
	stats.Notifications = notifications

	if err := db.Model(&models.Notification{}).
		Where("firm_id = ? AND (user_id IS NULL OR user_id = ?) AND read_at IS NULL", firm.ID, user.ID).
		Count(&stats.UnreadCount).Error; err != nil {
		c.Logger().Error("Failed to count unread notifications:", err)
	}

	component := pages.Dashboard(c.Request().Context(), "Dashboard | LexLegal Cloud", csrfToken, user, firm, stats)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
