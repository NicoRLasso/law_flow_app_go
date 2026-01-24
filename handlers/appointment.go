package handlers

import (
	"fmt"
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"law_flow_app_go/templates/partials"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// AppointmentsPageHandler renders the appointments page
func AppointmentsPageHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	csrfToken := middleware.GetCSRFToken(c)

	component := pages.Appointments(c.Request().Context(), "Appointments | LexLegal Cloud", csrfToken, user, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetAppointmentsHandler returns appointments for the current lawyer or firm
func GetAppointmentsHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	// Parse date range from query params (default to current week)
	startStr := c.QueryParam("start")
	endStr := c.QueryParam("end")

	var startDate, endDate time.Time
	var err error

	if startStr != "" {
		startDate, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid start date format")
		}
	} else {
		// Default to 1 year ago to show past appointments too
		startDate = time.Now().AddDate(-1, 0, 0)
		startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	}

	if endStr != "" {
		endDate, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid end date format")
		}
		endDate = endDate.Add(24 * time.Hour) // Include the end date
	} else {
		// Default to 1 year from now to show all upcoming appointments
		endDate = time.Now().AddDate(1, 0, 0)
	}

	var appointments []models.Appointment

	// Admins see all firm appointments, lawyers see their own
	if user.Role == "admin" {
		appointments, err = services.GetFirmAppointments(*user.FirmID, startDate, endDate)
	} else {
		appointments, err = services.GetLawyerAppointments(user.ID, startDate, endDate)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch appointments")
	}

	// Check if HTMX request - return HTML table
	if c.Request().Header.Get("HX-Request") == "true" {
		// For now, simple pagination (page 1, all results)
		component := partials.AppointmentTable(c.Request().Context(), appointments, 1, 1, len(appointments), len(appointments))
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, appointments)
}

// GetAvailableSlotsHandler returns available slots for a lawyer on a specific date
func GetAvailableSlotsHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	// Get lawyer ID (for admin, can specify lawyer; otherwise use current user)
	lawyerID := c.QueryParam("lawyer_id")
	if lawyerID == "" {
		lawyerID = user.ID
	}

	// Parse date
	dateStr := c.QueryParam("date")
	if dateStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Date is required")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid date format (use YYYY-MM-DD)")
	}

	// Get firm timezone
	var firm models.Firm
	if err := db.DB.First(&firm, "id = ?", *user.FirmID).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch firm")
	}

	// TODO: Get slot duration from firm settings (default 60 min for now)
	slotDuration := 60

	slots, err := services.GetAvailableSlots(lawyerID, date, slotDuration, firm.Timezone)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate slots")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"slots":    slots,
		"date":     dateStr,
		"timezone": firm.Timezone,
	})
}

// CreateAppointmentHandler creates a new appointment
func CreateAppointmentHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	var req struct {
		LawyerID          string  `json:"lawyer_id" form:"lawyer_id"`
		ClientID          string  `json:"client_id" form:"client_id"`
		AppointmentTypeID string  `json:"appointment_type_id" form:"appointment_type_id"`
		StartTime         string  `json:"start_time" form:"start_time"` // RFC3339 format
		EndTime           string  `json:"end_time" form:"end_time"`     // RFC3339 format
		Notes             *string `json:"notes" form:"notes"`
		CaseID            *string `json:"case_id" form:"case_id"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	if req.LawyerID == "" || req.StartTime == "" || req.EndTime == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "lawyer_id, start_time, and end_time are required")
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid start_time format (use RFC3339)")
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid end_time format (use RFC3339)")
	}

	// Resolve ClientID from CaseID if provided
	if req.CaseID != nil && *req.CaseID != "" {
		var kase models.Case
		if err := db.DB.First(&kase, "id = ?", *req.CaseID).Error; err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid case ID")
		}
		req.ClientID = kase.ClientID
	} else if req.ClientID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "client_id or case_id is required")
	}

	// Verify client exists and has role 'client'
	var client models.User
	if err := db.DB.First(&client, "id = ? AND role = ?", req.ClientID, "client").Error; err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid client")
	}

	// Verify lawyer exists and belongs to the same firm
	var lawyer models.User
	if err := db.DB.First(&lawyer, "id = ? AND firm_id = ? AND role IN (?)", req.LawyerID, *user.FirmID, []string{"lawyer", "admin"}).Error; err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid lawyer")
	}

	// Create appointment with client snapshot
	apt := &models.Appointment{
		LawyerID:    req.LawyerID,
		ClientID:    &req.ClientID,
		ClientName:  client.Name,
		ClientEmail: client.Email,
		ClientPhone: client.DocumentNumber, // Using DocumentNumber as phone if available
		StartTime:   startTime.UTC(),
		EndTime:     endTime.UTC(),
		Status:      models.AppointmentStatusScheduled,
		Notes:       req.Notes,
		CaseID:      req.CaseID,
		FirmID:      *user.FirmID,
	}

	// Set appointment type if provided
	if req.AppointmentTypeID != "" {
		apt.AppointmentTypeID = &req.AppointmentTypeID
	}

	if err := services.CreateAppointment(apt); err != nil {
		// For HTMX requests, return error as HTML
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusConflict, fmt.Sprintf(`<div class="text-red-500 text-sm">%s</div>`, err.Error()))
		}
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	// Send confirmation emails asynchronously
	cfg := c.Get("config").(*config.Config)

	// Get firm for email data
	var firm models.Firm
	db.DB.First(&firm, "id = ?", apt.FirmID)

	// Generate ICS file
	firmEmail := firm.InfoEmail
	if firmEmail == "" {
		firmEmail = firm.NoreplyEmail
	}
	icsContent, err := services.GenerateAppointmentICS(apt, firm.Name, firmEmail, firm.Timezone)
	if err != nil {
		// Log error but continue with email sending
		fmt.Printf("Error generating ICS: %v\n", err)
	}

	// Send confirmation to client
	clientEmailData := services.AppointmentConfirmationEmailData{
		ClientName:      client.Name,
		FirmName:        firm.Name,
		Date:            apt.StartTime.Format("January 2, 2006"),
		Time:            apt.StartTime.Format("3:04 PM"),
		Duration:        int(apt.EndTime.Sub(apt.StartTime).Minutes()),
		LawyerName:      lawyer.Name,
		AppointmentType: "", // Added to fix potential missing field if struct changed or just empty
	}
	if apt.AppointmentType != nil {
		clientEmailData.AppointmentType = apt.AppointmentType.Name
	}

	clientLang := client.Language
	if clientLang == "" {
		clientLang = "es"
	}
	clientEmail := services.BuildAppointmentConfirmationEmail(client.Email, clientEmailData, clientLang)

	// Attach ICS if generated successfully
	if len(icsContent) > 0 {
		clientEmail.Attachments = append(clientEmail.Attachments, services.Attachment{
			Filename: "appointment.ics",
			Content:  icsContent,
		})
	}

	services.SendEmailAsync(cfg, clientEmail)

	// Notify lawyer about new appointment
	lawyerEmailData := services.LawyerAppointmentNotificationEmailData{
		LawyerName:      lawyer.Name,
		ClientName:      client.Name,
		ClientEmail:     client.Email,
		ClientPhone:     "",
		Date:            apt.StartTime.Format("January 2, 2006"),
		Time:            apt.StartTime.Format("3:04 PM"),
		Duration:        int(apt.EndTime.Sub(apt.StartTime).Minutes()),
		AppointmentType: "",
	}
	if apt.AppointmentType != nil {
		lawyerEmailData.AppointmentType = apt.AppointmentType.Name
	}

	if client.PhoneNumber != nil {
		lawyerEmailData.ClientPhone = *client.PhoneNumber
	}
	if apt.Notes != nil {
		lawyerEmailData.Notes = *apt.Notes
	}
	lawyerLang := lawyer.Language
	if lawyerLang == "" {
		lawyerLang = "es"
	}
	lawyerEmail := services.BuildLawyerAppointmentNotificationEmail(lawyer.Email, lawyerEmailData, lawyerLang)

	// Attach ICS to lawyer email as well
	if len(icsContent) > 0 {
		lawyerEmail.Attachments = append(lawyerEmail.Attachments, services.Attachment{
			Filename: "appointment.ics",
			Content:  icsContent,
		})
	}

	services.SendEmailAsync(cfg, lawyerEmail)

	// For HTMX requests, return success with trigger to reload table
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", "reload-appointments")
		return c.HTML(http.StatusCreated, `<div class="text-green-500 text-sm">Appointment created successfully!</div>`)
	}

	// Reload with relationships for API response
	apt, _ = services.GetAppointmentByID(apt.ID)

	// Audit logging (Create Appointment)
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionCreate, "Appointment", apt.ID, fmt.Sprintf("Appointment with %s", apt.ClientName), "Created appointment", nil, apt)

	return c.JSON(http.StatusCreated, apt)
}

// GetAppointmentHandler returns a single appointment
func GetAppointmentHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	id := c.Param("id")
	apt, err := services.GetAppointmentByID(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Appointment not found")
	}

	// Verify access (same firm)
	if apt.FirmID != *user.FirmID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	return c.JSON(http.StatusOK, apt)
}

// UpdateAppointmentStatusHandler updates the status of an appointment
func UpdateAppointmentStatusHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	id := c.Param("id")
	apt, err := services.GetAppointmentByID(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Appointment not found")
	}

	// Verify access (same firm)
	if apt.FirmID != *user.FirmID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	var req struct {
		Status string `json:"status" form:"status"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if !models.IsValidAppointmentStatus(req.Status) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid status")
	}

	if err := services.UpdateAppointmentStatus(id, req.Status); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update status")
	}

	apt, _ = services.GetAppointmentByID(id)

	// Audit logging (Update Status)
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionUpdate, "Appointment", apt.ID, fmt.Sprintf("Appointment with %s", apt.ClientName), "Updated appointment status to "+req.Status, nil, apt)

	return c.JSON(http.StatusOK, apt)
}

// CancelAppointmentHandler cancels an appointment
func CancelAppointmentHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	id := c.Param("id")
	apt, err := services.GetAppointmentByID(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Appointment not found")
	}

	// Verify access (same firm)
	if apt.FirmID != *user.FirmID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	if err := services.CancelAppointment(id); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusOK, fmt.Sprintf(`<div class="text-red-500">Error: %s</div>`, err.Error()))
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Audit logging (Cancel)
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionUpdate, "Appointment", id, fmt.Sprintf("Appointment with %s", apt.ClientName), "Cancelled appointment", nil, map[string]string{"status": "cancelled"})

	if c.Request().Header.Get("HX-Request") == "true" {
		// Reload the appointment to get updated status
		updatedApt, _ := services.GetAppointmentByID(id)
		if updatedApt == nil {
			return c.NoContent(http.StatusNotFound)
		}
		return partials.AppointmentRow(c.Request().Context(), *updatedApt).Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Appointment cancelled"})
}

// RescheduleAppointmentHandler reschedules an appointment
func RescheduleAppointmentHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	id := c.Param("id")
	apt, err := services.GetAppointmentByID(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Appointment not found")
	}

	// Verify access (same firm)
	if apt.FirmID != *user.FirmID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	var req struct {
		StartTime string `json:"start_time" form:"start_time"`
		EndTime   string `json:"end_time" form:"end_time"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid start_time format")
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid end_time format")
	}

	if err := services.RescheduleAppointment(id, startTime.UTC(), endTime.UTC()); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	apt, _ = services.GetAppointmentByID(id)

	// Audit logging (Reschedule)
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionUpdate, "Appointment", id, fmt.Sprintf("Appointment with %s", apt.ClientName), "Rescheduled appointment", nil, apt)

	return c.JSON(http.StatusOK, apt)
}

// GetClientsForAppointmentHandler returns clients that can be booked
func GetClientsForAppointmentHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	var clients []models.User
	if err := db.DB.Where("firm_id = ? AND role = ? AND is_active = ?", *user.FirmID, "client", true).
		Order("name asc").Find(&clients).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch clients")
	}

	// Return HTML options for select
	html := "<option value=''>Select a client...</option>"
	for _, client := range clients {
		html += fmt.Sprintf("<option value='%s'>%s</option>", client.ID, client.Name)
	}
	return c.HTML(http.StatusOK, html)
}

// GetLawyersForAppointmentHandler returns lawyers that can receive appointments
func GetLawyersForAppointmentHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	var lawyers []models.User
	if err := db.DB.Where("firm_id = ? AND role IN (?) AND is_active = ?", *user.FirmID, []string{"lawyer", "admin"}, true).
		Order("name asc").Find(&lawyers).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch lawyers")
	}

	// Return HTML options for select
	html := "<option value=''>Select a lawyer...</option>"
	for _, lawyer := range lawyers {
		html += fmt.Sprintf("<option value='%s'>%s</option>", lawyer.ID, lawyer.Name)
	}
	return c.HTML(http.StatusOK, html)
}

// GetCasesForAppointmentHandler returns cases that the user can book appointments for
func GetCasesForAppointmentHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	var cases []models.Case
	query := db.DB.Where("firm_id = ? AND status = ?", *user.FirmID, models.CaseStatusOpen).Preload("Client")

	// If not admin, valid only assigned cases or where collaborator
	if user.Role != "admin" {
		query = query.Where("assigned_to_id = ? OR id IN (SELECT case_id FROM case_collaborators WHERE user_id = ?)", user.ID, user.ID)
	}

	if err := query.Order("created_at desc").Find(&cases).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch cases")
	}

	// Return HTML options for select
	html := "<option value=''>Select a case...</option>"
	for _, kase := range cases {
		label := kase.CaseNumber
		if kase.Title != nil && *kase.Title != "" {
			label += " - " + *kase.Title
		}
		if kase.Client.Name != "" {
			label += " (" + kase.Client.Name + ")"
		}
		html += fmt.Sprintf("<option value='%s'>%s</option>", kase.ID, label)
	}
	return c.HTML(http.StatusOK, html)
}
