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
	currentFirm := middleware.GetCurrentFirm(c)

	var activities []models.ServiceActivity
	if err := db.DB.Where("firm_id = ? AND service_id = ?", currentFirm.ID, serviceID).
		Order("occurred_at DESC").
		Find(&activities).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch activities")
	}

	component := partials.ServiceActivityList(c.Request().Context(), activities)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetServiceActivityForm returns the modal for adding a new activity
func GetServiceActivityForm(c echo.Context) error {
	serviceID := c.Param("id")
	component := partials.AddServiceActivityModal(c.Request().Context(), serviceID)
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

	// Update service hours if time entry
	if activity.ActivityType == models.ActivityTypeTimeEntry {
		services.UpdateServiceActualHours(db.DB, serviceID)
	}

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

	if activity.ActivityType == models.ActivityTypeTimeEntry {
		services.UpdateServiceActualHours(db.DB, serviceID)
	}

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

	return GetServiceActivitiesHandler(c)
}
