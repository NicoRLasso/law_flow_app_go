package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"law_flow_app_go/templates/partials"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// CaseRequestsPageHandler renders the case requests dashboard page
func CaseRequestsPageHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	component := pages.CaseRequests("Case Requests | Law Flow", user, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// PublicCaseRequestHandler renders the public case request form
func PublicCaseRequestHandler(c echo.Context) error {
	slug := c.Param("slug")

	// Find firm by slug
	var firm models.Firm
	if err := db.DB.Where("slug = ?", slug).First(&firm).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Firm not found")
	}

	// Render public form template
	component := pages.PublicCaseRequest(firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// PublicCaseRequestPostHandler handles public case request form submission
func PublicCaseRequestPostHandler(c echo.Context) error {
	slug := c.Param("slug")

	// Find firm by slug
	var firm models.Firm
	if err := db.DB.Where("slug = ?", slug).First(&firm).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Firm not found")
	}

	// Parse form data
	name := strings.TrimSpace(c.FormValue("name"))
	email := strings.TrimSpace(c.FormValue("email"))
	phone := strings.TrimSpace(c.FormValue("phone"))
	documentType := strings.TrimSpace(c.FormValue("document_type"))
	documentNumber := strings.TrimSpace(c.FormValue("document_number"))
	description := strings.TrimSpace(c.FormValue("description"))
	priority := strings.TrimSpace(c.FormValue("priority"))

	// Set default priority if not provided
	if priority == "" {
		priority = models.PriorityMedium
	}

	// Validate required fields
	if name == "" || email == "" || phone == "" || documentType == "" || documentNumber == "" || description == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="error-message">All fields are required</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "All fields are required")
	}

	// Validate document type
	if !models.IsValidDocumentType(documentType) {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="error-message">Invalid document type</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid document type")
	}

	// Validate priority
	if !models.IsValidPriority(priority) {
		priority = models.PriorityMedium
	}

	// Create case request
	caseRequest := models.CaseRequest{
		FirmID:         firm.ID,
		Name:           name,
		Email:          email,
		Phone:          phone,
		DocumentType:   documentType,
		DocumentNumber: documentNumber,
		Description:    description,
		Priority:       priority,
		Status:         models.StatusPending,
		IPAddress:      c.RealIP(),
		UserAgent:      c.Request().UserAgent(),
	}

	// Handle optional file upload
	file, err := c.FormFile("file")
	if err == nil && file != nil {
		// Validate PDF upload
		if err := services.ValidatePDFUpload(file); err != nil {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusBadRequest, `<div class="error-message">`+err.Error()+`</div>`)
			}
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		// Get upload directory from config
		uploadDir := "uploads" // Default, should come from config in production

		// Save file
		uploadResult, err := services.SaveUploadedFile(file, uploadDir, firm.ID)
		if err != nil {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusInternalServerError, `<div class="error-message">Failed to upload file</div>`)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to upload file")
		}

		// Store file metadata
		caseRequest.FileName = uploadResult.FileName
		caseRequest.FileOriginalName = uploadResult.FileOriginalName
		caseRequest.FilePath = uploadResult.FilePath
		caseRequest.FileSize = uploadResult.FileSize
	}

	// Save to database
	if err := db.DB.Create(&caseRequest).Error; err != nil {
		// Cleanup uploaded file on error
		if caseRequest.FilePath != "" {
			services.DeleteUploadedFile(caseRequest.FilePath)
		}
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="error-message">Failed to submit request</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to submit request")
	}

	// Redirect to success page
	return c.Redirect(http.StatusSeeOther, "/firm/"+firm.Slug+"/request/success")
}

// GetCaseRequestsHandler returns a list of case requests for the current firm
func GetCaseRequestsHandler(c echo.Context) error {
	// Get query parameters for filtering
	status := c.QueryParam("status")
	priority := c.QueryParam("priority")

	// Build firm-scoped query
	query := middleware.GetFirmScopedQuery(c, db.DB)

	// Apply filters
	if status != "" && models.IsValidStatus(status) {
		query = query.Where("status = ?", status)
	}
	if priority != "" && models.IsValidPriority(priority) {
		query = query.Where("priority = ?", priority)
	}

	// Fetch requests
	var requests []models.CaseRequest
	if err := query.Order("created_at DESC").Find(&requests).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch requests")
	}

	// Check if HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		component := partials.CaseRequestList(requests)
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, requests)
}

// GetCaseRequestHandler returns a single case request
func GetCaseRequestHandler(c echo.Context) error {
	id := c.Param("id")

	// Fetch request with firm-scoping
	var request models.CaseRequest
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.Preload("ReviewedBy").First(&request, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	return c.JSON(http.StatusOK, request)
}

// DownloadCaseRequestFileHandler serves the uploaded file
func DownloadCaseRequestFileHandler(c echo.Context) error {
	id := c.Param("id")

	// Fetch request with firm-scoping
	var request models.CaseRequest
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.First(&request, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	// Check if file exists
	if request.FilePath == "" {
		return echo.NewHTTPError(http.StatusNotFound, "No file attached to this request")
	}

	// Verify file path is within upload directory (security check)
	uploadDir := "uploads" // Should come from config
	absUploadDir, err := filepath.Abs(uploadDir)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify file path")
	}
	absFilePath, err := filepath.Abs(request.FilePath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify file path")
	}
	if !strings.HasPrefix(absFilePath, absUploadDir) {
		return echo.NewHTTPError(http.StatusForbidden, "Invalid file path")
	}

	// Serve file
	return c.File(request.FilePath)
}

// UpdateCaseRequestStatusHandler updates the status of a case request
func UpdateCaseRequestStatusHandler(c echo.Context) error {
	id := c.Param("id")

	// Parse request body
	var payload struct {
		Status string `json:"status"`
	}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate status
	if !models.IsValidStatus(payload.Status) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid status")
	}

	// Fetch request with firm-scoping
	var request models.CaseRequest
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.First(&request, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	// Get current user
	currentUser := middleware.GetCurrentUser(c)

	// Update status
	now := time.Now()
	request.Status = payload.Status
	request.ReviewedByID = &currentUser.ID
	request.ReviewedAt = &now

	if err := db.DB.Save(&request).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update status")
	}

	return c.JSON(http.StatusOK, request)
}
