package handlers

import (
	"fmt"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/services/i18n"
	"law_flow_app_go/templates/pages"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// Helper to render error message with data-error attribute so modal doesn't close
func availabilityErrorHTML(msg string) string {
	return fmt.Sprintf(`<div class="text-red-500 text-sm" data-error="true">%s</div>`, msg)
}

// Helper to render success message
func availabilitySuccessHTML(msg string) string {
	return fmt.Sprintf(`<div class="text-green-500 text-sm">%s</div>`, msg)
}

// AvailabilityPageHandler renders the availability settings page
func AvailabilityPageHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	// Check if user is a lawyer or admin
	if currentUser.Role != "lawyer" && currentUser.Role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "Only lawyers and admins can access availability settings")
	}

	// Get lawyer ID (for admins viewing their own availability, use their ID)
	lawyerID := currentUser.ID

	// Fallback: create default availability if none exists (for users created before availability seeding was added to user creation)
	hasSlots, err := services.HasAvailabilitySlots(lawyerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check availability slots")
	}

	if !hasSlots {
		if err := services.CreateDefaultAvailability(lawyerID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create default availability")
		}
	}

	// Get availability slots
	slots, err := services.GetLawyerAvailability(lawyerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load availability")
	}

	// Get blocked dates
	blockedDates, err := services.GetAllBlockedDates(lawyerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load blocked dates")
	}

	csrfToken := middleware.GetCSRFToken(c)
	component := pages.AvailabilitySettings(c.Request().Context(), "Availability Settings | LexLegal Cloud", csrfToken, currentUser, firm, slots, blockedDates)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetAvailabilityHandler returns availability slots for the current lawyer
func GetAvailabilityHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	slots, err := services.GetLawyerAvailability(currentUser.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load availability")
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		component := pages.AvailabilitySchedule(c.Request().Context(), slots)
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, slots)
}

// CreateAvailabilityHandler creates a new availability slot
func CreateAvailabilityHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	ctx := c.Request().Context()

	dayOfWeek, err := strconv.Atoi(c.FormValue("day_of_week"))
	if err != nil || dayOfWeek < 0 || dayOfWeek > 6 {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, availabilityErrorHTML(i18n.T(ctx, "availability.errors.invalid_day")))
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid day of week")
	}

	startTime := strings.TrimSpace(c.FormValue("start_time"))
	endTime := strings.TrimSpace(c.FormValue("end_time"))

	if startTime == "" || endTime == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, availabilityErrorHTML(i18n.T(ctx, "availability.errors.times_required")))
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Start and end time are required")
	}

	if startTime >= endTime {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, availabilityErrorHTML(i18n.T(ctx, "availability.errors.end_after_start")))
		}
		return echo.NewHTTPError(http.StatusBadRequest, "End time must be after start time")
	}

	// Check for overlaps
	overlaps, err := services.CheckAvailabilityOverlap(currentUser.ID, dayOfWeek, startTime, endTime, "")
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, availabilityErrorHTML(i18n.T(ctx, "availability.errors.check_failed")))
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check availability")
	}
	if overlaps {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, availabilityErrorHTML(i18n.T(ctx, "availability.errors.slot_overlaps")))
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Slot overlaps with existing availability")
	}

	slot := &models.Availability{
		LawyerID:  currentUser.ID,
		DayOfWeek: dayOfWeek,
		StartTime: startTime,
		EndTime:   endTime,
		IsActive:  true,
	}

	if err := services.CreateAvailabilitySlot(slot); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, availabilityErrorHTML(i18n.T(ctx, "availability.errors.create_failed")))
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create slot")
	}

	// For HTMX, reload the availability list
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", "availability-updated")
	}

	// Audit logging
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionCreate, "Availability", slot.ID, "Availability Slot", "Created availability slot", nil, slot)

	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, availabilitySuccessHTML(i18n.T(ctx, "availability.success.slot_added")))
	}

	return c.JSON(http.StatusCreated, slot)
}

// UpdateAvailabilityHandler updates an existing availability slot
func UpdateAvailabilityHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	ctx := c.Request().Context()
	slotID := c.Param("id")

	slot, err := services.GetAvailabilityByID(slotID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Slot not found")
	}

	// Verify ownership
	if slot.LawyerID != currentUser.ID && currentUser.Role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	// Update fields
	if dayStr := c.FormValue("day_of_week"); dayStr != "" {
		dayOfWeek, err := strconv.Atoi(dayStr)
		if err == nil && dayOfWeek >= 0 && dayOfWeek <= 6 {
			slot.DayOfWeek = dayOfWeek
		}
	}

	if startTime := strings.TrimSpace(c.FormValue("start_time")); startTime != "" {
		slot.StartTime = startTime
	}

	if endTime := strings.TrimSpace(c.FormValue("end_time")); endTime != "" {
		slot.EndTime = endTime
	}

	if isActiveStr := c.FormValue("is_active"); isActiveStr != "" {
		slot.IsActive = isActiveStr == "true" || isActiveStr == "on" || isActiveStr == "1"
	}

	if slot.StartTime >= slot.EndTime {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, availabilityErrorHTML(i18n.T(ctx, "availability.errors.end_after_start")))
		}
		return echo.NewHTTPError(http.StatusBadRequest, "End time must be after start time")
	}

	// Check for overlaps (excluding current slot)
	overlaps, err := services.CheckAvailabilityOverlap(slot.LawyerID, slot.DayOfWeek, slot.StartTime, slot.EndTime, slot.ID)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, availabilityErrorHTML(i18n.T(ctx, "availability.errors.check_failed")))
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check availability")
	}
	if overlaps {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, availabilityErrorHTML(i18n.T(ctx, "availability.errors.slot_overlaps")))
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Slot overlaps with existing availability")
	}

	if err := services.UpdateAvailabilitySlot(slot); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, availabilityErrorHTML(i18n.T(ctx, "availability.errors.update_failed")))
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update slot")
	}

	// Audit logging
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionUpdate, "Availability", slot.ID, "Availability Slot", "Updated availability slot", nil, slot)

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", "availability-updated")
		return c.HTML(http.StatusOK, availabilitySuccessHTML(i18n.T(ctx, "availability.success.slot_updated")))
	}

	return c.JSON(http.StatusOK, slot)
}

// deleteEntityConfig holds configuration for the generic delete handler
type deleteEntityConfig struct {
	EntityName  string
	AuditType   string
	TriggerName string
	FetchFunc   func(id string) (interface{}, string, error) // Returns entity, ownerID, error
	DeleteFunc  func(id string) error
}

// handleDeleteEntity is a generic helper for deleting entities with ownership check and audit logging
func handleDeleteEntity(c echo.Context, id string, cfg deleteEntityConfig) error {
	currentUser := middleware.GetCurrentUser(c)

	entity, ownerID, err := cfg.FetchFunc(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, cfg.EntityName+" not found")
	}

	// Verify ownership
	if ownerID != currentUser.ID && currentUser.Role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	if err := cfg.DeleteFunc(id); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, fmt.Sprintf(`<div class="text-red-500 text-sm">Failed to delete %s</div>`, strings.ToLower(cfg.EntityName)))
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete "+strings.ToLower(cfg.EntityName))
	}

	// Audit logging
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionDelete, cfg.AuditType, id, cfg.EntityName, "Deleted "+strings.ToLower(cfg.EntityName), entity, nil)

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", cfg.TriggerName)
		return c.HTML(http.StatusOK, "")
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteAvailabilityHandler deletes an availability slot
func DeleteAvailabilityHandler(c echo.Context) error {
	slotID := c.Param("id")

	cfg := deleteEntityConfig{
		EntityName:  "Availability Slot",
		AuditType:   "Availability",
		TriggerName: "availability-updated",
		FetchFunc: func(id string) (interface{}, string, error) {
			slot, err := services.GetAvailabilityByID(id)
			if err != nil {
				return nil, "", err
			}
			return slot, slot.LawyerID, nil
		},
		DeleteFunc: services.DeleteAvailabilitySlot,
	}

	return handleDeleteEntity(c, slotID, cfg)
}

// GetBlockedDatesHandler returns blocked dates for the current lawyer
func GetBlockedDatesHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	blockedDates, err := services.GetAllBlockedDates(currentUser.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load blocked dates")
	}
	// Check for HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		component := pages.BlockedDatesList(c.Request().Context(), blockedDates)
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, blockedDates)
}

// CreateBlockedDateHandler creates a new blocked date (Range Only)
func CreateBlockedDateHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	reason := strings.TrimSpace(c.FormValue("reason"))

	startDateStr := strings.TrimSpace(c.FormValue("start_date"))
	endDateStr := strings.TrimSpace(c.FormValue("end_date"))

	if startDateStr == "" || endDateStr == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="text-red-500 text-sm">Start and end dates are required</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Start and end dates are required")
	}

	sDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid start date")
	}
	eDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid end date")
	}

	if eDate.Before(sDate) {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="text-red-500 text-sm">End date must be after start date</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "End date must be after start date")
	}

	startAt := sDate
	endAt := eDate.Add(24 * time.Hour).Add(-1 * time.Second) // End of last day

	blockedDate := &models.BlockedDate{
		LawyerID:  currentUser.ID,
		StartAt:   startAt,
		EndAt:     endAt,
		Reason:    reason,
		IsFullDay: true, // Simplified to always be full day
	}

	if err := services.CreateBlockedDate(blockedDate); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="text-red-500 text-sm">Failed to save block</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save block")
	}

	// Audit logging
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionCreate, "BlockedDate", blockedDate.ID, "Blocked Date", "Created blocked date: "+reason, nil, blockedDate)

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", "blocked-dates-updated")
		return c.HTML(http.StatusOK, `<div class="text-green-500 text-sm">Blocked successfully!</div>`)
	}

	return c.JSON(http.StatusCreated, blockedDate)
}

// DeleteBlockedDateHandler deletes a blocked date
func DeleteBlockedDateHandler(c echo.Context) error {
	dateID := c.Param("id")

	cfg := deleteEntityConfig{
		EntityName:  "Blocked Date",
		AuditType:   "BlockedDate",
		TriggerName: "blocked-dates-updated",
		FetchFunc: func(id string) (interface{}, string, error) {
			date, err := services.GetBlockedDateByID(id)
			if err != nil {
				return nil, "", err
			}
			return date, date.LawyerID, nil
		},
		DeleteFunc: services.DeleteBlockedDate,
	}

	return handleDeleteEntity(c, dateID, cfg)
}

// UpdateBufferSettingsHandler updates the firm's buffer time settings
func UpdateBufferSettingsHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)

	// Only admins can update buffer settings
	if currentUser.Role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "Only admins can update buffer settings")
	}

	bufferMinutes, err := strconv.Atoi(c.FormValue("buffer_minutes"))
	if err != nil || (bufferMinutes != 15 && bufferMinutes != 30 && bufferMinutes != 45 && bufferMinutes != 60) {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="text-red-500 text-sm">`+i18n.T(c.Request().Context(), "availability.errors.invalid_buffer")+`</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Buffer must be 15, 30, 45, or 60 minutes")
	}

	firm := middleware.GetCurrentFirm(c)
	if err := db.DB.Model(&firm).Update("buffer_minutes", bufferMinutes).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, `<div class="text-red-500 text-sm">Failed to update buffer settings</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update buffer settings")
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, `<div class="text-green-500 text-sm">Buffer settings updated!</div>`)
	}

	return c.JSON(http.StatusOK, map[string]int{"buffer_minutes": bufferMinutes})
}

// CheckOverlapHandler validates if a time slot overlaps with existing ones
func CheckOverlapHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)

	dayOfWeek, _ := strconv.Atoi(c.FormValue("day_of_week"))
	startTime := strings.TrimSpace(c.FormValue("start_time"))
	endTime := strings.TrimSpace(c.FormValue("end_time"))
	excludeSlotID := c.FormValue("exclude_slot_id")

	// If inputs are incomplete, don't show warning yet
	if startTime == "" || endTime == "" {
		return c.NoContent(http.StatusOK)
	}

	// Basic validation
	if startTime >= endTime {
		return c.HTML(http.StatusOK, `<div class="text-red-500 text-sm mt-1">End time must be after start time</div>`)
	}

	overlaps, err := services.CheckAvailabilityOverlap(currentUser.ID, dayOfWeek, startTime, endTime, excludeSlotID)
	if err != nil {
		// Log error but don't show specific error to user during typing validation
		return c.NoContent(http.StatusOK)
	}

	if overlaps {
		return c.HTML(http.StatusOK, `<div class="text-red-500 text-sm mt-1">Warning: This time overlaps with an existing slot!</div>`)
	}

	return c.NoContent(http.StatusOK)
}

// CheckBlockedDateOverlapHandler validates if a blocked date overlaps with existing ones
func CheckBlockedDateOverlapHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)

	startDateStr := strings.TrimSpace(c.FormValue("start_date"))
	endDateStr := strings.TrimSpace(c.FormValue("end_date"))

	if startDateStr == "" || endDateStr == "" {
		return c.NoContent(http.StatusOK)
	}

	sDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return c.NoContent(http.StatusOK)
	}
	eDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return c.NoContent(http.StatusOK)
	}

	// Range is always full days
	startAt := sDate
	endAt := eDate.Add(24 * time.Hour).Add(-1 * time.Second)

	excludeID := c.FormValue("exclude_id")

	overlaps, err := services.CheckBlockedDateOverlap(currentUser.ID, startAt, endAt, excludeID)
	if err != nil {
		return c.NoContent(http.StatusOK)
	}

	if overlaps {
		return c.HTML(http.StatusOK, `<div class="text-red-500 text-sm mt-1">Warning: Overlaps with existing block!</div>`)
	}

	return c.NoContent(http.StatusOK)
}
