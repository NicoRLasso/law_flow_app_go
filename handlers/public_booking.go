package handlers

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// PublicBookingPageHandler renders the public booking page
func PublicBookingPageHandler(c echo.Context) error {
	firmSlug := c.Param("slug")

	// Get firm by slug
	var firm models.Firm
	if err := db.DB.Where("slug = ?", firmSlug).First(&firm).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Firm not found")
	}

	// Get lawyers with availability
	lawyers, err := services.GetFirmLawyersWithAvailability(firm.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch lawyers")
	}

	// Get active appointment types
	aptTypes, err := services.GetActiveAppointmentTypes(firm.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch appointment types")
	}

	component := pages.PublicBooking(c.Request().Context(), firm, lawyers, aptTypes)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// PublicGetLawyersHandler returns available lawyers for booking (JSON)
func PublicGetLawyersHandler(c echo.Context) error {
	firmSlug := c.Param("slug")

	// Get firm by slug
	var firm models.Firm
	if err := db.DB.Where("slug = ?", firmSlug).First(&firm).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Firm not found")
	}

	lawyers, err := services.GetFirmLawyersWithAvailability(firm.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch lawyers")
	}

	return c.JSON(http.StatusOK, lawyers)
}

// PublicGetSlotsHandler returns available slots for a lawyer/date (JSON)
func PublicGetSlotsHandler(c echo.Context) error {
	firmSlug := c.Param("slug")

	// Get firm by slug
	var firm models.Firm
	if err := db.DB.Where("slug = ?", firmSlug).First(&firm).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Firm not found")
	}

	lawyerID := c.QueryParam("lawyer_id")
	dateStr := c.QueryParam("date")
	aptTypeID := c.QueryParam("type_id")

	if lawyerID == "" || dateStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "lawyer_id and date are required")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid date format")
	}

	// Get slot duration (default 60, or from appointment type)
	slotDuration := 60
	if aptTypeID != "" {
		aptType, err := services.GetAppointmentTypeByID(aptTypeID)
		if err == nil {
			slotDuration = aptType.DurationMinutes
		}
	}

	slots, err := services.GetAvailableSlots(lawyerID, date, slotDuration, firm.Timezone)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get slots")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"slots":    slots,
		"date":     dateStr,
		"timezone": firm.Timezone,
	})
}

// PublicSubmitBookingHandler handles public appointment booking
func PublicSubmitBookingHandler(c echo.Context) error {
	firmSlug := c.Param("slug")

	// Get firm by slug
	var firm models.Firm
	if err := db.DB.Where("slug = ?", firmSlug).First(&firm).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Firm not found")
	}

	var req struct {
		LawyerID          string  `json:"lawyer_id" form:"lawyer_id"`
		AppointmentTypeID string  `json:"appointment_type_id" form:"appointment_type_id"`
		StartTime         string  `json:"start_time" form:"start_time"` // RFC3339
		EndTime           string  `json:"end_time" form:"end_time"`     // RFC3339
		ClientName        string  `json:"client_name" form:"client_name"`
		ClientEmail       string  `json:"client_email" form:"client_email"`
		ClientPhone       string  `json:"client_phone" form:"client_phone"`
		Notes             *string `json:"notes" form:"notes"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	if req.LawyerID == "" || req.StartTime == "" || req.ClientName == "" || req.ClientEmail == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required fields")
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid start_time format")
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid end_time format")
	}

	// Verify lawyer belongs to firm
	var lawyer models.User
	if err := db.DB.First(&lawyer, "id = ? AND firm_id = ? AND role IN (?)", req.LawyerID, firm.ID, []string{"lawyer", "admin"}).Error; err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid lawyer")
	}

	// Create appointment
	var aptTypeID *string
	if req.AppointmentTypeID != "" {
		aptTypeID = &req.AppointmentTypeID
	}

	var phone *string
	if req.ClientPhone != "" {
		phone = &req.ClientPhone
	}

	apt := &models.Appointment{
		FirmID:            firm.ID,
		LawyerID:          req.LawyerID,
		AppointmentTypeID: aptTypeID,
		ClientName:        req.ClientName,
		ClientEmail:       req.ClientEmail,
		ClientPhone:       phone,
		StartTime:         startTime.UTC(),
		EndTime:           endTime.UTC(),
		Status:            models.AppointmentStatusScheduled,
		Notes:             req.Notes,
	}

	if err := services.CreateAppointment(apt); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	// Load appointment with relations
	apt, _ = services.GetAppointmentByID(apt.ID)

	// Send confirmation emails asynchronously
	cfg := c.Get("config").(*config.Config)

	// Email to client
	go func() {
		typeName := ""
		if apt.AppointmentType != nil {
			typeName = apt.AppointmentType.Name
		}
		meetingURL := ""
		if apt.MeetingURL != nil {
			meetingURL = *apt.MeetingURL
		}

		// Default to "es" for public booking
		clientLang := "es"

		email := services.BuildAppointmentConfirmationEmail(apt.ClientEmail, services.AppointmentConfirmationEmailData{
			ClientName:      apt.ClientName,
			FirmName:        firm.Name,
			Date:            apt.StartTime.Format("Monday, January 2, 2006"),
			Time:            apt.StartTime.Format("3:04 PM"),
			Duration:        apt.Duration(),
			LawyerName:      apt.Lawyer.Name,
			AppointmentType: typeName,
			MeetingURL:      meetingURL,
			ManageLink:      cfg.AppURL + "/appointment/" + apt.BookingToken,
		}, clientLang)
		if err := services.SendEmail(cfg, email); err != nil {
			// Log error but don't fail
		}
	}()

	// Email to lawyer
	go func() {
		typeName := ""
		if apt.AppointmentType != nil {
			typeName = apt.AppointmentType.Name
		}
		phone := ""
		if apt.ClientPhone != nil {
			phone = *apt.ClientPhone
		}
		notes := ""
		if apt.Notes != nil {
			notes = *apt.Notes
		}

		// Load lawyer language
		lawyerLang := apt.Lawyer.Language
		if lawyerLang == "" {
			lawyerLang = "es"
		}

		email := services.BuildLawyerAppointmentNotificationEmail(apt.Lawyer.Email, services.LawyerAppointmentNotificationEmailData{
			LawyerName:      apt.Lawyer.Name,
			ClientName:      apt.ClientName,
			ClientEmail:     apt.ClientEmail,
			ClientPhone:     phone,
			Date:            apt.StartTime.Format("Monday, January 2, 2006"),
			Time:            apt.StartTime.Format("3:04 PM"),
			Duration:        apt.Duration(),
			AppointmentType: typeName,
			Notes:           notes,
		}, lawyerLang)
		if err := services.SendEmail(cfg, email); err != nil {
			// Log error but don't fail
		}
	}()

	// Return success with booking token for confirmation page
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message":       "Appointment booked successfully",
		"appointment":   apt,
		"booking_token": apt.BookingToken,
	})
}

// PublicAppointmentDetailHandler shows appointment details via token
func PublicAppointmentDetailHandler(c echo.Context) error {
	token := c.Param("token")

	var apt models.Appointment
	if err := db.DB.Preload("Lawyer").Preload("Firm").Preload("AppointmentType").
		Where("booking_token = ?", token).First(&apt).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Appointment not found")
	}

	component := pages.PublicAppointmentDetail(c.Request().Context(), apt)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// PublicCancelAppointmentHandler cancels an appointment via token
func PublicCancelAppointmentHandler(c echo.Context) error {
	token := c.Param("token")

	var apt models.Appointment
	if err := db.DB.Preload("Lawyer").Preload("Firm").
		Where("booking_token = ?", token).First(&apt).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Appointment not found")
	}

	if !apt.IsCancellable() {
		return echo.NewHTTPError(http.StatusBadRequest, "Appointment cannot be cancelled")
	}

	var req struct {
		Reason string `json:"reason" form:"reason"`
	}
	c.Bind(&req)

	// Update appointment
	now := time.Now()
	updates := map[string]interface{}{
		"status":       models.AppointmentStatusCancelled,
		"cancelled_at": now,
	}
	if req.Reason != "" {
		updates["cancellation_reason"] = req.Reason
	}

	if err := db.DB.Model(&apt).Updates(updates).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to cancel")
	}

	// Send cancellation email
	cfg := c.Get("config").(*config.Config)
	go func() {
		var firm models.Firm
		db.DB.First(&firm, "id = ?", apt.FirmID)

		// Default to "es" for public cancellation
		email := services.BuildAppointmentCancelledEmail(apt.ClientEmail, services.AppointmentCancelledEmailData{
			ClientName:         apt.ClientName,
			FirmName:           firm.Name,
			Date:               apt.StartTime.Format("Monday, January 2, 2006"),
			Time:               apt.StartTime.Format("3:04 PM"),
			LawyerName:         apt.Lawyer.Name,
			CancellationReason: req.Reason,
			BookingLink:        cfg.AppURL + "/firm/" + firm.Slug + "/book",
		}, "es")
		services.SendEmail(cfg, email)
	}()

	return c.JSON(http.StatusOK, map[string]string{"message": "Appointment cancelled"})
}
