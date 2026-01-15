package handlers

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"law_flow_app_go/templates/partials"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// CaseRequestsPageHandler renders the case requests dashboard page
func CaseRequestsPageHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)

	component := pages.CaseRequests(c.Request().Context(), "Case Requests | Law Flow", user, firm)
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

	// Fetch document type options
	documentTypes, err := services.GetChoiceOptions(db.DB, firm.ID, "document_type")
	if err != nil {
		c.Logger().Errorf("Failed to fetch document types for firm %s: %v", firm.ID, err)
		// Continue with empty slice - form will show no options
	}

	// Fetch priority options
	priorities, err := services.GetChoiceOptions(db.DB, firm.ID, "priority")
	if err != nil {
		c.Logger().Errorf("Failed to fetch priorities for firm %s: %v", firm.ID, err)
		// Continue with empty slice - form will show no options
	}

	// Render public form template
	component := pages.PublicCaseRequest(c.Request().Context(), firm, documentTypes, priorities)
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
	if !services.ValidateChoiceOption(db.DB, firm.ID, "document_type", documentType) {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="error-message">Invalid document type</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid document type")
	}

	// Validate priority
	if !services.ValidateChoiceOption(db.DB, firm.ID, "priority", priority) {
		priority = models.PriorityMedium
	}

	// Resolve document type code to UUID
	var documentTypeID *string
	var docTypeOption models.ChoiceOption
	if err := db.DB.Where("firm_id = ? AND category = ? AND code = ?", firm.ID, "document_type", documentType).
		First(&docTypeOption).Error; err == nil {
		documentTypeID = &docTypeOption.ID
	}

	// Create case request
	caseRequest := models.CaseRequest{
		FirmID:         firm.ID,
		Name:           name,
		Email:          email,
		Phone:          phone,
		DocumentType:   documentType,   // Legacy: store code
		DocumentTypeID: documentTypeID, // New: store UUID reference
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

	// Get pagination parameters
	page := 1
	limit := 20
	if pageParam := c.QueryParam("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Build firm-scoped query
	query := middleware.GetFirmScopedQuery(c, db.DB)

	// Filter out rejected requests older than 24 hours
	// Logic: Show if (Status != Rejected) OR (Status == Rejected AND UpdatedAt > 24h ago)
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
	query = query.Where("(status != ? OR (status = ? AND updated_at > ?))",
		models.StatusRejected, models.StatusRejected, twentyFourHoursAgo)

	// Apply filters
	if status != "" && models.IsValidStatus(status) {
		query = query.Where("status = ?", status)
	}
	if priority != "" && models.IsValidPriority(priority) {
		query = query.Where("priority = ?", priority)
	}

	// Get total count
	var total int64
	if err := query.Model(&models.CaseRequest{}).Count(&total).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count requests")
	}

	// Calculate pagination
	offset := (page - 1) * limit
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Fetch paginated requests
	var requests []models.CaseRequest
	if err := query.Preload("DocumentTypeOption").Order("created_at DESC").Limit(limit).Offset(offset).Find(&requests).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch requests")
	}

	// Check if HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		component := partials.CaseRequestTable(c.Request().Context(), requests, page, totalPages, limit, int(total))
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	// Return JSON with pagination metadata
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": requests,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetCaseRequestHandler returns a single case request
func GetCaseRequestHandler(c echo.Context) error {
	id := c.Param("id")

	// Fetch request with firm-scoping
	var request models.CaseRequest
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.Preload("ReviewedBy").Preload("DocumentTypeOption").First(&request, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	return c.JSON(http.StatusOK, request)
}

// GetCaseRequestDetailHandler returns a case request detail modal
func GetCaseRequestDetailHandler(c echo.Context) error {
	id := c.Param("id")

	// Fetch request with firm-scoping
	var request models.CaseRequest
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.Preload("ReviewedBy").Preload("DocumentTypeOption").First(&request, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	// Render detail modal
	component := partials.CaseRequestDetailModal(c.Request().Context(), request)
	return component.Render(c.Request().Context(), c.Response().Writer)
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
		Status        string `json:"status" form:"status"`
		RejectionNote string `json:"rejection_note" form:"rejection_note"`
	}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate status
	if !models.IsValidStatus(payload.Status) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid status")
	}

	// Validate rejection note if status is rejected
	if payload.Status == models.StatusRejected && strings.TrimSpace(payload.RejectionNote) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Rejection note is required when rejecting a request")
	}

	// Fetch request with firm-scoping and firm information
	var request models.CaseRequest
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.Preload("Firm").First(&request, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	// IMMUTABLE REJECTION: Prevent changing status if already rejected
	if request.Status == models.StatusRejected {
		if c.Request().Header.Get("HX-Request") == "true" {
			// For HTMX, we might want to show a toast or alert, but for now 400 is fine
			return c.HTML(http.StatusBadRequest, "Cannot change status of a rejected request")
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot change status of a rejected request")
	}

	// Get current user
	currentUser := middleware.GetCurrentUser(c)

	// Update status
	now := time.Now()
	request.Status = payload.Status
	request.ReviewedByID = &currentUser.ID
	request.ReviewedAt = &now

	// Update rejection note if provided
	if payload.Status == models.StatusRejected {
		request.RejectionNote = strings.TrimSpace(payload.RejectionNote)

		// DOCUMENTATION REMOVAL: Delete file if exists
		if request.FilePath != "" {
			// Verify file path is within upload directory (security check)
			uploadDir := "uploads"
			absUploadDir, err := filepath.Abs(uploadDir)
			if err == nil {
				absFilePath, err := filepath.Abs(request.FilePath)
				if err == nil && strings.HasPrefix(absFilePath, absUploadDir) {
					// Delete file
					if err := services.DeleteUploadedFile(request.FilePath); err != nil {
						c.Logger().Errorf("Failed to delete file %s: %v", request.FilePath, err)
					}
				}
			}
			// Clear metadata
			request.FilePath = ""
			request.FileName = ""
			request.FileOriginalName = ""
			request.FileSize = 0
		}
	}

	if err := db.DB.Save(&request).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update status")
	}

	// Send rejection email if status is rejected
	if payload.Status == models.StatusRejected {
		// Get config and build rejection email
		cfg := c.Get("config").(*config.Config)

		// Determine contact email (prefer InfoEmail, fallback to BillingEmail)
		firmEmail := request.Firm.InfoEmail
		if firmEmail == "" {
			firmEmail = request.Firm.BillingEmail
		}

		// Build rejection email
		emailMsg := services.BuildCaseRequestRejectionEmail(
			request.Email,
			request.Name,
			request.Firm.Name,
			request.RejectionNote,
			firmEmail,
			request.Firm.Phone,
		)

		// Send email asynchronously
		services.SendEmailAsync(cfg, emailMsg)
	}

	// Return success (NoContent for HTMX to avoid swapping JSON, JSON for API)
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.NoContent(http.StatusOK)
	}

	return c.JSON(http.StatusOK, request)
}

// DeleteCaseRequestHandler deletes a case request and its associated file
func DeleteCaseRequestHandler(c echo.Context) error {
	id := c.Param("id")

	// Fetch request with firm-scoping
	var request models.CaseRequest
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.First(&request, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	// Delete associated file from disk if exists
	if request.FilePath != "" {
		// Verify file path is within upload directory (security check)
		uploadDir := "uploads"
		absUploadDir, err := filepath.Abs(uploadDir)
		if err == nil {
			absFilePath, err := filepath.Abs(request.FilePath)
			if err == nil && strings.HasPrefix(absFilePath, absUploadDir) {
				// Delete file, log error but don't fail the request
				if err := services.DeleteUploadedFile(request.FilePath); err != nil {
					// Log error but continue with database deletion
					c.Logger().Errorf("Failed to delete file %s: %v", request.FilePath, err)
				}
			}
		}
	}

	// Soft delete the database record
	if err := db.DB.Delete(&request).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete request")
	}

	// Return success for HTMX
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.NoContent(http.StatusOK)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Request deleted successfully"})
}
