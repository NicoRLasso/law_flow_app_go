package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/partials"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// GetServiceActivitiesHandler list
func GetServiceActivitiesHandler(c echo.Context) error {
	serviceID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	var activities []models.ServiceActivity
	var total int64

	query := db.DB.Where("firm_id = ? AND service_id = ?", currentFirm.ID, serviceID)

	if err := query.Model(&models.ServiceActivity{}).Count(&total).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count activities")
	}

	if err := query.Order("occurred_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&activities).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch activities")
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	component := partials.ServiceActivityTable(c.Request().Context(), activities, page, totalPages, limit, int(total), serviceID, currentUser.Role != "client")
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetServiceActivityForm returns the modal for adding a new activity
func GetServiceActivityForm(c echo.Context) error {
	serviceID := c.Param("id")
	component := partials.AddServiceActivityModal(c.Request().Context(), serviceID)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetServiceActivityEditModalHandler returns the edit modal for an activity
func GetServiceActivityEditModalHandler(c echo.Context) error {
	serviceID := c.Param("id")
	activityID := c.Param("aid")
	currentFirm := middleware.GetCurrentFirm(c)

	var activity models.ServiceActivity
	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, activityID, serviceID).First(&activity).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Activity not found")
	}

	component := partials.EditServiceActivityModal(c.Request().Context(), activity, serviceID)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// CreateServiceActivityHandler
func CreateServiceActivityHandler(c echo.Context) error {
	serviceID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	activityType := c.FormValue("activity_type")
	content := c.FormValue("content")
	title := c.FormValue("title")

	if title == "" {
		title = "Activity"
	}

	activity := models.ServiceActivity{
		FirmID:       currentFirm.ID,
		ServiceID:    serviceID,
		ActivityType: activityType,
		Title:        title,
		Content:      content,
		CreatedByID:  currentUser.ID,
	}

	now := time.Now()
	activity.OccurredAt = &now

	if dateStr := c.FormValue("occurred_at"); dateStr != "" {
		if t, err := time.Parse("2006-01-02T15:04", dateStr); err == nil {
			activity.OccurredAt = &t
		} else if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			activity.OccurredAt = &t
		}
	}

	if durationStr := c.FormValue("duration"); durationStr != "" {
		if d, err := strconv.Atoi(durationStr); err == nil {
			activity.Duration = &d
		}
	}

	if err := db.DB.Create(&activity).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity")
	}

	// Trigger refreshes
	c.Response().Header().Set("HX-Trigger", `{"refreshTimeline": true, "refreshSummary": true}`)

	return GetServiceActivitiesHandler(c)
}

// UpdateServiceActivityHandler
func UpdateServiceActivityHandler(c echo.Context) error {
	serviceID := c.Param("id")
	activityID := c.Param("aid")
	currentFirm := middleware.GetCurrentFirm(c)

	var activity models.ServiceActivity
	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, activityID, serviceID).First(&activity).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Activity not found")
	}

	activity.Content = c.FormValue("content")
	activity.Title = c.FormValue("title")

	if durationStr := c.FormValue("duration"); durationStr != "" {
		if d, err := strconv.Atoi(durationStr); err == nil {
			activity.Duration = &d
		}
	}

	if err := db.DB.Save(&activity).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update activity")
	}

	// Trigger refreshes
	c.Response().Header().Set("HX-Trigger", `{"refreshTimeline": true, "refreshSummary": true}`)

	return GetServiceActivitiesHandler(c)
}

// DeleteServiceActivityHandler
func DeleteServiceActivityHandler(c echo.Context) error {
	serviceID := c.Param("id")
	activityID := c.Param("aid")
	currentFirm := middleware.GetCurrentFirm(c)

	var activity models.ServiceActivity
	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, activityID, serviceID).First(&activity).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Activity not found")
	}

	isTimeEntry := activity.ActivityType == models.ActivityTypeTimeEntry

	if err := db.DB.Delete(&activity).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete")
	}

	if isTimeEntry {
		services.UpdateServiceActualHours(db.DB, serviceID)
	}

	// Trigger refreshes
	c.Response().Header().Set("HX-Trigger", `{"refreshTimeline": true, "refreshSummary": true}`)

	return GetServiceActivitiesHandler(c)
}
