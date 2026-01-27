package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetAppointmentTypesHandler returns all appointment types for the firm
func GetAppointmentTypesHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	types, err := services.GetAppointmentTypes(db.DB, *user.FirmID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch appointment types")
	}

	return c.JSON(http.StatusOK, types)
}

// GetActiveAppointmentTypesHandler returns only active appointment types (for booking)
func GetActiveAppointmentTypesHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	types, err := services.GetActiveAppointmentTypes(db.DB, *user.FirmID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch appointment types")
	}

	return c.JSON(http.StatusOK, types)
}

// CreateAppointmentTypeHandler creates a new appointment type
func CreateAppointmentTypeHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	var req struct {
		Name            string `json:"name" form:"name"`
		Description     string `json:"description" form:"description"`
		DurationMinutes int    `json:"duration_minutes" form:"duration_minutes"`
		Color           string `json:"color" form:"color"`
		Order           int    `json:"order" form:"order"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Name is required")
	}

	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 60 // Default to 60 minutes
	}

	if req.Color == "" {
		req.Color = "#3B82F6" // Default blue
	}

	aptType := &models.AppointmentType{
		FirmID:          *user.FirmID,
		Name:            req.Name,
		Description:     req.Description,
		DurationMinutes: req.DurationMinutes,
		Color:           req.Color,
		Order:           req.Order,
		IsActive:        true,
	}

	if err := services.CreateAppointmentType(db.DB, aptType); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create appointment type")
	}

	return c.JSON(http.StatusCreated, aptType)
}

// UpdateAppointmentTypeHandler updates an appointment type
func UpdateAppointmentTypeHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	id := c.Param("id")
	aptType, err := services.GetAppointmentTypeByID(db.DB, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Appointment type not found")
	}

	// Verify same firm
	if aptType.FirmID != *user.FirmID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	var req struct {
		Name            *string `json:"name" form:"name"`
		Description     *string `json:"description" form:"description"`
		DurationMinutes *int    `json:"duration_minutes" form:"duration_minutes"`
		Color           *string `json:"color" form:"color"`
		Order           *int    `json:"order" form:"order"`
		IsActive        *bool   `json:"is_active" form:"is_active"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.DurationMinutes != nil {
		updates["duration_minutes"] = *req.DurationMinutes
	}
	if req.Color != nil {
		updates["color"] = *req.Color
	}
	if req.Order != nil {
		updates["order"] = *req.Order
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := services.UpdateAppointmentType(db.DB, id, updates); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update appointment type")
	}

	aptType, _ = services.GetAppointmentTypeByID(db.DB, id)
	return c.JSON(http.StatusOK, aptType)
}

// DeleteAppointmentTypeHandler deletes an appointment type
func DeleteAppointmentTypeHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	id := c.Param("id")
	aptType, err := services.GetAppointmentTypeByID(db.DB, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Appointment type not found")
	}

	// Verify same firm
	if aptType.FirmID != *user.FirmID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	if err := services.DeleteAppointmentType(db.DB, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete appointment type")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Appointment type deleted"})
}
