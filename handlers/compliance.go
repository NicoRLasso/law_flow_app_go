package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages/compliance"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// ComplianceDashboardHandler renders the main compliance center dashboard
func ComplianceDashboardHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)
	firm := c.Get("firm").(*models.Firm)
	ctx := c.Request().Context()

	// Get consent logs count
	var consentCount int64
	db.DB.Model(&models.ConsentLog{}).Where("firm_id = ?", firm.ID).Count(&consentCount)

	// Get pending ARCO requests
	var pendingRequests int64
	db.DB.Model(&models.SubjectRightsRequest{}).
		Where("firm_id = ? AND status = ?", firm.ID, models.SubjectRequestStatusPending).
		Count(&pendingRequests)

	// Get recent audit logs
	var recentAuditLogs []models.AuditLog
	db.DB.Where("firm_id = ?", firm.ID).
		Order("created_at DESC").
		Limit(10).
		Find(&recentAuditLogs)

	data := compliance.DashboardData{
		User:            user,
		Firm:            firm,
		ConsentCount:    consentCount,
		PendingRequests: pendingRequests,
		RecentAuditLogs: recentAuditLogs,
	}

	return render(c, compliance.Dashboard(ctx, data))
}

// GetComplianceConsentLogsHandler returns paginated consent logs for a firm
func GetComplianceConsentLogsHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)
	firm := c.Get("firm").(*models.Firm)
	ctx := c.Request().Context()

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit := 20

	var consents []models.ConsentLog
	var total int64

	query := db.DB.Model(&models.ConsentLog{}).Where("firm_id = ?", firm.ID)
	query.Count(&total)

	if err := query.Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&consents).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch consent logs")
	}

	totalPages := (int(total) + limit - 1) / limit

	// If HTMX request, return just the table fragment
	if c.Request().Header.Get("HX-Request") == "true" {
		return render(c, compliance.ConsentLogTable(ctx, consents, page, totalPages, int(total)))
	}

	// Otherwise return full page
	data := compliance.ConsentsPageData{
		User:     user,
		Firm:     firm,
		Consents: consents,
		Page:     page,
		Total:    int(total),
		Pages:    totalPages,
	}
	return render(c, compliance.ConsentsPage(ctx, data))
}

// GetComplianceARCORequestsHandler returns paginated ARCO requests for a firm
func GetComplianceARCORequestsHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)
	firm := c.Get("firm").(*models.Firm)
	ctx := c.Request().Context()

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit := 20

	statusFilter := c.QueryParam("status")
	var status *models.SubjectRequestStatus
	if statusFilter != "" {
		s := models.SubjectRequestStatus(statusFilter)
		status = &s
	}

	requests, total, err := services.GetSubjectRightsRequests(db.DB, firm.ID, status, page, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch requests")
	}

	totalPages := (int(total) + limit - 1) / limit

	// If HTMX request, return just the table fragment
	if c.Request().Header.Get("HX-Request") == "true" {
		return render(c, compliance.ARCORequestTable(ctx, requests, page, totalPages, int(total)))
	}

	// Otherwise return full page
	data := compliance.ARCOPageData{
		User:     user,
		Firm:     firm,
		Requests: requests,
		Page:     page,
		Total:    int(total),
		Pages:    totalPages,
	}
	return render(c, compliance.ARCOPage(ctx, data))
}

// CreateComplianceARCORequestHandler creates a new ARCO request (for clients)
func CreateComplianceARCORequestHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)

	requestType := models.SubjectRequestType(c.FormValue("request_type"))
	justification := c.FormValue("justification")

	if justification == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Justification is required")
	}

	_, err := services.CreateSubjectRightsRequest(
		db.DB,
		user.ID,
		user.Email,
		user.FirmID,
		requestType,
		justification,
		c.RealIP(),
		c.Request().UserAgent(),
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create request")
	}

	// Return success message or redirect
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", "arcoRequestCreated")
		return c.HTML(http.StatusOK, `<div class="p-4 bg-green-100 text-green-800 rounded-lg">Request submitted successfully</div>`)
	}

	return c.Redirect(http.StatusSeeOther, "/admin/compliance")
}

// ResolveComplianceARCORequestHandler resolves an ARCO request (approve/deny)
func ResolveComplianceARCORequestHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)
	firm := c.Get("firm").(*models.Firm)

	requestID := c.Param("id")
	action := c.FormValue("action") // "approve" or "deny"
	response := c.FormValue("response")

	var status models.SubjectRequestStatus
	if action == "approve" {
		status = models.SubjectRequestStatusApproved
	} else {
		status = models.SubjectRequestStatusDenied
	}

	if err := services.ResolveSubjectRightsRequest(db.DB, requestID, user.ID, status, response); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to resolve request")
	}

	// Log the action using existing audit pattern
	description := fmt.Sprintf("ARCO request %s: %s", action, requestID)
	auditLog := models.AuditLog{
		UserID:       &user.ID,
		UserName:     user.Name,
		UserRole:     user.Role,
		FirmID:       &firm.ID,
		FirmName:     firm.Name,
		ResourceType: "subject_rights_request",
		ResourceID:   requestID,
		Action:       models.AuditActionUpdate,
		Description:  description,
	}
	db.DB.Create(&auditLog)

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", "arcoRequestResolved")
		// Return updated table
		return GetComplianceARCORequestsHandler(c)
	}

	return c.Redirect(http.StatusSeeOther, "/admin/compliance/arco")
}

// ExportComplianceUserDataHandler generates a ZIP file with all user data (Right of Portability)
func ExportComplianceUserDataHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)
	firm := c.Get("firm").(*models.Firm)

	// Create ZIP buffer
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// 1. User profile data
	profileData := map[string]interface{}{
		"id":              user.ID,
		"name":            user.Name,
		"email":           user.Email,
		"phone":           user.PhoneNumber,
		"document_number": user.DocumentNumber,
		"address":         user.Address,
		"created_at":      user.CreatedAt,
	}
	profileJSON, _ := json.MarshalIndent(profileData, "", "  ")
	profileFile, _ := zipWriter.Create("profile.json")
	profileFile.Write(profileJSON)

	// 2. Cases where user is client
	var cases []models.Case
	db.DB.Where("client_id = ?", user.ID).Find(&cases)
	casesJSON, _ := json.MarshalIndent(cases, "", "  ")
	casesFile, _ := zipWriter.Create("cases.json")
	casesFile.Write(casesJSON)

	// 3. Consent history
	consents, _ := services.GetUserConsents(db.DB, user.ID)
	consentsJSON, _ := json.MarshalIndent(consents, "", "  ")
	consentsFile, _ := zipWriter.Create("consent_history.json")
	consentsFile.Write(consentsJSON)

	// 4. Services
	var legalServices []models.LegalService
	db.DB.Where("client_id = ?", user.ID).Find(&legalServices)
	servicesJSON, _ := json.MarshalIndent(legalServices, "", "  ")
	servicesFile, _ := zipWriter.Create("services.json")
	servicesFile.Write(servicesJSON)

	zipWriter.Close()

	// Log the export action
	description := "User data export (Right of Portability)"
	auditLog := models.AuditLog{
		UserID:       &user.ID,
		UserName:     user.Name,
		UserRole:     user.Role,
		FirmID:       &firm.ID,
		FirmName:     firm.Name,
		ResourceType: "user",
		ResourceID:   user.ID,
		Action:       models.AuditActionView,
		Description:  description,
	}
	db.DB.Create(&auditLog)

	// Return ZIP file
	c.Response().Header().Set("Content-Type", "application/zip")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=my_data_%s.zip", time.Now().Format("2006-01-02")))

	return c.Blob(http.StatusOK, "application/zip", buf.Bytes())
}

// GetComplianceAuditLogsHandler returns paginated audit logs for compliance view
func GetComplianceAuditLogsHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)
	firm := c.Get("firm").(*models.Firm)
	ctx := c.Request().Context()

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit := 50

	var logs []models.AuditLog
	var total int64

	query := db.DB.Model(&models.AuditLog{}).Where("firm_id = ?", firm.ID)
	query.Count(&total)

	if err := query.Order("created_at DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&logs).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch audit logs")
	}

	totalPages := (int(total) + limit - 1) / limit

	// If HTMX request, return just the table fragment
	if c.Request().Header.Get("HX-Request") == "true" {
		return render(c, compliance.AuditLogTable(ctx, logs, page, totalPages, int(total)))
	}

	// Otherwise return full page
	data := compliance.AuditPageData{
		User:  user,
		Firm:  firm,
		Logs:  logs,
		Page:  page,
		Total: int(total),
		Pages: totalPages,
	}
	return render(c, compliance.AuditPage(ctx, data))
}
