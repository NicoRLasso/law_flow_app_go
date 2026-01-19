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
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// CasesPageHandler renders the cases page
func CasesPageHandler(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	csrfToken := middleware.GetCSRFToken(c)

	component := pages.Cases(c.Request().Context(), "Cases | Law Flow", csrfToken, user, firm)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetCasesHandler returns a list of cases with filtering and pagination
func GetCasesHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)

	// Get query parameters for filtering
	status := c.QueryParam("status")
	assignedTo := c.QueryParam("assigned_to")
	dateFrom := c.QueryParam("date_from")
	dateTo := c.QueryParam("date_to")
	keyword := c.QueryParam("keyword")

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

	// Apply role-based filter
	if currentUser.Role == "lawyer" {
		// Lawyers see cases assigned to them OR where they are collaborators
		query = query.Where(
			db.DB.Where("assigned_to_id = ?", currentUser.ID).
				Or("EXISTS (SELECT 1 FROM case_collaborators WHERE case_collaborators.case_id = cases.id AND case_collaborators.user_id = ?)", currentUser.ID),
		)
	}
	// Admins see all cases (no additional filter)

	// Apply historical filter
	historical := c.QueryParam("historical")
	if historical == "true" {
		query = query.Where("is_historical = ?", true)
	} else {
		// By default, show only non-historical cases
		query = query.Where("is_historical = ? OR is_historical IS NULL", false)
	}

	// Apply status filter
	if status != "" && models.IsValidCaseStatus(status) {
		query = query.Where("status = ?", status)
	}

	// Apply assigned_to filter (admin only)
	if assignedTo != "" && currentUser.Role == "admin" {
		query = query.Where("assigned_to_id = ?", assignedTo)
	}

	// Apply date range filters
	if dateFrom != "" {
		if parsedDate, err := time.Parse("2006-01-02", dateFrom); err == nil {
			query = query.Where("opened_at >= ?", parsedDate)
		}
	}
	if dateTo != "" {
		if parsedDate, err := time.Parse("2006-01-02", dateTo); err == nil {
			// Add 24 hours to include the entire day
			endOfDay := parsedDate.Add(24 * time.Hour)
			query = query.Where("opened_at < ?", endOfDay)
		}
	}

	// Apply keyword search
	if keyword != "" {
		keyword = "%" + keyword + "%"
		query = query.Where(
			db.DB.Where("case_number LIKE ?", keyword).
				Or("title LIKE ?", keyword).
				Or("description LIKE ?", keyword).
				Or("EXISTS (SELECT 1 FROM users WHERE users.id = cases.client_id AND users.name LIKE ?)", keyword),
		)
	}

	// Get total count
	var total int64
	if err := query.Model(&models.Case{}).Count(&total).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count cases")
	}

	// Calculate pagination
	offset := (page - 1) * limit
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Fetch paginated cases with preloading
	var cases []models.Case
	if err := query.
		Preload("Client").
		Preload("AssignedTo").
		Preload("Domain").
		Preload("Branch").
		Preload("Subtypes").
		Preload("Documents").
		Preload("Collaborators").
		Order("opened_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&cases).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch cases")
	}

	// Check if HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		component := partials.CaseTable(c.Request().Context(), cases, page, totalPages, limit, int(total))
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	// Return JSON with pagination metadata
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": cases,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetCaseDetailHandler returns a case detail page
func GetCaseDetailHandler(c echo.Context) error {
	id := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	// Build firm-scoped query
	query := middleware.GetFirmScopedQuery(c, db.DB)

	// Apply role-based filter
	if currentUser.Role == "lawyer" {
		// Lawyers see cases assigned to them OR where they are collaborators
		query = query.Where(
			db.DB.Where("assigned_to_id = ?", currentUser.ID).
				Or("EXISTS (SELECT 1 FROM case_collaborators WHERE case_collaborators.case_id = cases.id AND case_collaborators.user_id = ?)", currentUser.ID),
		)
	}

	// Fetch case with all relationships
	var caseRecord models.Case
	if err := query.
		Preload("Client").
		Preload("AssignedTo").
		Preload("Domain").
		Preload("Branch").
		Preload("Subtypes").
		Preload("Documents").
		Preload("Collaborators").
		First(&caseRecord, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Render detail page
	csrfToken := middleware.GetCSRFToken(c)
	component := pages.CaseDetail(c.Request().Context(), "Case Details | Law Flow", csrfToken, currentUser, currentFirm, caseRecord)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetLawyersForFilterHandler returns a list of lawyers for the filter dropdown (admin only)
func GetLawyersForFilterHandler(c echo.Context) error {
	// Build firm-scoped query
	query := middleware.GetFirmScopedQuery(c, db.DB)

	// Fetch active lawyers and admins
	var users []models.User
	if err := query.
		Where("role IN (?, ?)", "lawyer", "admin").
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&users).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch lawyers")
	}

	// Return JSON
	return c.JSON(http.StatusOK, users)
}

// GetCaseDocumentsHandler returns case documents with filtering and pagination
func GetCaseDocumentsHandler(c echo.Context) error {
	caseID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	// Get query parameters for filtering
	documentType := c.QueryParam("document_type")
	keyword := c.QueryParam("keyword")

	// Get pagination parameters
	page := 1
	limit := 10
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

	// First verify the case exists and user has access
	caseQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if currentUser.Role == "lawyer" {
		caseQuery = caseQuery.Where("assigned_to_id = ?", currentUser.ID)
	}

	var caseRecord models.Case
	if err := caseQuery.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Build documents query
	query := middleware.GetFirmScopedQuery(c, db.DB).
		Where("case_id = ?", caseID)

	// Apply document type filter
	if documentType != "" {
		query = query.Where("document_type = ?", documentType)
	}

	// Apply keyword search
	if keyword != "" {
		keyword = "%" + keyword + "%"
		query = query.Where(
			db.DB.Where("file_original_name LIKE ?", keyword).
				Or("description LIKE ?", keyword),
		)
	}

	// Get total count
	var total int64
	if err := query.Model(&models.CaseDocument{}).Count(&total).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count documents")
	}

	// Calculate pagination
	offset := (page - 1) * limit
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Fetch paginated documents
	var documents []models.CaseDocument
	if err := query.
		Preload("UploadedBy").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&documents).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch documents")
	}

	// Check if HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		component := partials.CaseDocumentTable(c.Request().Context(), documents, page, totalPages, limit, int(total), caseID)
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	// Return JSON with pagination metadata
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": documents,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// DownloadCaseDocumentHandler serves a case document for download
func DownloadCaseDocumentHandler(c echo.Context) error {
	caseID := c.Param("id")
	docID := c.Param("docId")
	currentUser := middleware.GetCurrentUser(c)

	// First verify the case exists and user has access
	caseQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if currentUser.Role == "lawyer" {
		caseQuery = caseQuery.Where("assigned_to_id = ?", currentUser.ID)
	}

	var caseRecord models.Case
	if err := caseQuery.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Fetch document with firm-scoping
	var document models.CaseDocument
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.First(&document, "id = ? AND case_id = ?", docID, caseID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Document not found")
	}

	// Check if file exists
	if document.FilePath == "" {
		return echo.NewHTTPError(http.StatusNotFound, "No file attached to this document")
	}

	// Verify file path is within upload directory (security check)
	uploadDir := "uploads"
	absUploadDir, err := filepath.Abs(uploadDir)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify file path")
	}
	absFilePath, err := filepath.Abs(document.FilePath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify file path")
	}
	if !strings.HasPrefix(absFilePath, absUploadDir) {
		return echo.NewHTTPError(http.StatusForbidden, "Invalid file path")
	}

	// Set the Content-Disposition header to suggest the original filename
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+document.FileOriginalName+"\"")

	// Audit logging (Download)
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(
		db.DB,
		auditCtx,
		models.AuditActionDownload,
		"CaseDocument",
		document.ID,
		document.FileOriginalName,
		"Document downloaded",
		nil,
		nil,
	)

	// Serve file
	return c.File(document.FilePath)
}

// ViewCaseDocumentHandler serves a PDF document for inline viewing
func ViewCaseDocumentHandler(c echo.Context) error {
	caseID := c.Param("id")
	docID := c.Param("docId")
	currentUser := middleware.GetCurrentUser(c)

	// First verify the case exists and user has access
	caseQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if currentUser.Role == "lawyer" {
		caseQuery = caseQuery.Where("assigned_to_id = ?", currentUser.ID)
	}

	var caseRecord models.Case
	if err := caseQuery.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Fetch document with firm-scoping
	var document models.CaseDocument
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.First(&document, "id = ? AND case_id = ?", docID, caseID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Document not found")
	}

	// Check if file exists
	if document.FilePath == "" {
		return echo.NewHTTPError(http.StatusNotFound, "No file attached to this document")
	}

	// Validate it's a PDF
	if !strings.HasSuffix(strings.ToLower(document.FileOriginalName), ".pdf") {
		return echo.NewHTTPError(http.StatusBadRequest, "Only PDF files can be viewed inline")
	}

	// Verify file path is within upload directory (security check)
	uploadDir := "uploads"
	absUploadDir, err := filepath.Abs(uploadDir)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify file path")
	}
	absFilePath, err := filepath.Abs(document.FilePath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify file path")
	}
	if !strings.HasPrefix(absFilePath, absUploadDir) {
		return echo.NewHTTPError(http.StatusForbidden, "Invalid file path")
	}

	// Set headers for inline display
	c.Response().Header().Set("Content-Type", "application/pdf")
	c.Response().Header().Set("Content-Disposition", "inline; filename=\""+document.FileOriginalName+"\"")

	// Audit logging (View)
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(
		db.DB,
		auditCtx,
		models.AuditActionView,
		"CaseDocument",
		document.ID,
		document.FileOriginalName,
		"Document viewed",
		nil,
		nil,
	)

	// Serve file
	return c.File(document.FilePath)
}

// UploadCaseDocumentHandler handles document uploads for a case
func UploadCaseDocumentHandler(c echo.Context) error {
	caseID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	// First verify the case exists and user has access
	caseQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if currentUser.Role == "lawyer" {
		caseQuery = caseQuery.Where("assigned_to_id = ?", currentUser.ID)
	}

	var caseRecord models.Case
	if err := caseQuery.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusNotFound, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Case not found</div>`)
		}
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Get form values
	documentType := c.FormValue("document_type")
	description := c.FormValue("description")

	// Validate document type
	if documentType == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Document type is required</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Document type is required")
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">File is required</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "File is required")
	}

	// Validate file
	if err := services.ValidateDocumentUpload(file); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">`+err.Error()+`</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Save file
	uploadDir := "uploads"
	uploadResult, err := services.SaveCaseDocument(file, uploadDir, currentFirm.ID, caseID)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to upload file</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to upload file")
	}

	// Get visibility setting (defaults to false/private)
	isPublic := c.FormValue("is_public") == "true" || c.FormValue("is_public") == "on"

	// Create document record
	document := models.CaseDocument{
		FirmID:           currentFirm.ID,
		CaseID:           &caseID,
		FileName:         uploadResult.FileName,
		FileOriginalName: uploadResult.FileOriginalName,
		FilePath:         uploadResult.FilePath,
		FileSize:         uploadResult.FileSize,
		MimeType:         uploadResult.MimeType,
		DocumentType:     documentType,
		UploadedByID:     &currentUser.ID,
		IsPublic:         isPublic,
	}

	if description != "" {
		document.Description = &description
	}

	// Save to database
	if err := db.DB.Create(&document).Error; err != nil {
		// Clean up uploaded file on database error
		services.DeleteUploadedFile(uploadResult.FilePath)
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to save document</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save document")
	}

	// Audit logging (Upload)
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(
		db.DB,
		auditCtx,
		models.AuditActionCreate,
		"CaseDocument",
		document.ID,
		document.FileOriginalName,
		"Document uploaded",
		nil,
		document,
	)

	// Return success message and trigger document list reload
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, `
			<div class="p-4 bg-green-500/20 text-green-400 rounded-lg mb-4">
				Document uploaded successfully!
			</div>
			<script>
				setTimeout(function() {
					closeUploadModal();
					htmx.trigger('#case-documents-list', 'loadDocuments');
				}, 1000);
			</script>
		`)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "Document uploaded successfully",
		"document": document,
	})
}

// ToggleDocumentVisibilityHandler toggles a document's public/private visibility
func ToggleDocumentVisibilityHandler(c echo.Context) error {
	caseID := c.Param("id")
	docID := c.Param("docId")
	currentUser := middleware.GetCurrentUser(c)

	// Only admin and lawyer can toggle visibility
	if currentUser.Role != "admin" && currentUser.Role != "lawyer" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusForbidden, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Permission denied</div>`)
		}
		return echo.NewHTTPError(http.StatusForbidden, "Permission denied")
	}

	// First verify the case exists and user has access
	caseQuery := middleware.GetFirmScopedQuery(c, db.DB)
	if currentUser.Role == "lawyer" {
		caseQuery = caseQuery.Where(
			db.DB.Where("assigned_to_id = ?", currentUser.ID).
				Or("EXISTS (SELECT 1 FROM case_collaborators WHERE case_collaborators.case_id = cases.id AND case_collaborators.user_id = ?)", currentUser.ID),
		)
	}

	var caseRecord models.Case
	if err := caseQuery.First(&caseRecord, "id = ?", caseID).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusNotFound, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Case not found</div>`)
		}
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Fetch document with firm-scoping
	var document models.CaseDocument
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.First(&document, "id = ? AND case_id = ?", docID, caseID).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusNotFound, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Document not found</div>`)
		}
		return echo.NewHTTPError(http.StatusNotFound, "Document not found")
	}

	// Toggle the visibility
	document.IsPublic = !document.IsPublic
	if err := db.DB.Save(&document).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to update visibility</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update visibility")
	}

	// Return updated document row for HTMX
	if c.Request().Header.Get("HX-Request") == "true" {
		// Preload uploader for display
		db.DB.Preload("UploadedBy").First(&document, "id = ?", docID)
		component := partials.CaseDocumentRow(c.Request().Context(), document, caseID)
		return component.Render(c.Request().Context(), c.Response().Writer)
	}

	// Audit logging (Visibility Change)
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(
		db.DB,
		auditCtx,
		models.AuditActionVisibilityChange,
		"CaseDocument",
		document.ID,
		document.FileOriginalName,
		"Document visibility changed",
		map[string]bool{"is_public": !document.IsPublic}, // Old value
		map[string]bool{"is_public": document.IsPublic},  // New value
	)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Visibility updated",
		"is_public": document.IsPublic,
	})
}
