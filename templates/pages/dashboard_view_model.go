package pages

import (
	"law_flow_app_go/models"
)

// DashboardStats holds the data for the dashboard
type DashboardStats struct {
	ActiveCases          int64
	TotalClients         int64
	CompletedMonthly     int64
	PendingTasks         int64
	RecentCases          []models.Case
	UpcomingAppointments []models.Appointment
	Notifications        []models.Notification
	UnreadCount          int64
}
