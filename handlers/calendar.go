package handlers

import (
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// CalendarPageHandler renders the calendar view
func CalendarPageHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	// Fetch appointment types for filter/legend (optional)
	aptTypes, _ := services.GetActiveAppointmentTypes(*user.FirmID)
	firm := middleware.GetCurrentFirm(c)
	csrfToken := middleware.GetCSRFToken(c)

	component := pages.CalendarPage(c.Request().Context(), user, firm, csrfToken, aptTypes)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// CalendarEventsHandler returns appointments as FullCalendar events
func CalendarEventsHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	startStr := c.QueryParam("start")
	endStr := c.QueryParam("end")

	if startStr == "" || endStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "start and end dates are required")
	}

	// FullCalendar sends ISO8601 string, but let's be safe with parsing
	// Often it's like 2023-10-01T00:00:00-05:00
	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		// Try YYYY-MM-DD
		startTime, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid start date format")
		}
	}

	endTime, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		// Try YYYY-MM-DD
		endTime, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid end date format")
		}
	}

	// Fetch appointments
	// If admin/staff, fetch for whole firm? Or just their own?
	// For now, let's fetch for the firm if admin, otherwise just for the lawyer
	var appointments []models.Appointment
	var dbErr error

	if user.Role == "admin" || user.Role == "staff" {
		appointments, dbErr = services.GetFirmAppointments(*user.FirmID, startTime, endTime)
	} else {
		appointments, dbErr = services.GetLawyerAppointments(user.ID, startTime, endTime)
	}

	if dbErr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch events")
	}

	// Map to FullCalendar event objects
	events := make([]map[string]interface{}, 0)
	for _, apt := range appointments {
		color := "#3B82F6" // Default blue
		if apt.AppointmentType != nil && apt.AppointmentType.Color != "" {
			color = apt.AppointmentType.Color
		}

		title := apt.ClientName
		if apt.AppointmentType != nil {
			title = apt.AppointmentType.Name + " - " + apt.ClientName
		} else {
			title = "Appointment - " + apt.ClientName
		}

		// Also check blocked dates if we want to show them?
		// Maybe later.

		events = append(events, map[string]interface{}{
			"id":              apt.ID,
			"title":           title,
			"start":           apt.StartTime.Format(time.RFC3339),
			"end":             apt.EndTime.Format(time.RFC3339),
			"backgroundColor": color,
			"borderColor":     color,
			"extendedProps": map[string]interface{}{
				"clientName": apt.ClientName,
				"status":     apt.Status,
				"notes":      apt.Notes,
				"lawyer":     apt.Lawyer.Name,
			},
		})
	}

	// Also fetch blocked dates to show as background events or gray events
	// For simplicity, let's just stick to appointments for now in this MVP step.
	// Blocked dates overlap visualization is a nice-to-have optimization.

	return c.JSON(http.StatusOK, events)
}
