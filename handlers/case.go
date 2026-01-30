package handlers

import (
	"context"
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

	component := pages.Cases(c.Request().Context(), "Cases | LexLegal Cloud", csrfToken, user, firm)
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

	// Build firm-scoped query
	query := middleware.GetFirmScopedQuery(c, db.DB)

	// Apply role-based filter
	if currentUser.Role == "lawyer" {
		// Lawyers see cases assigned to them OR where they are collaborators
		query = query.Where(
			db.DB.Where("assigned_to_id = ?", currentUser.ID).
				Or("EXISTS (SELECT 1 FROM case_collaborators WHERE case_collaborators.case_id = cases.id AND case_collaborators.user_id = ?)", currentUser.ID),
		)
	} else if currentUser.Role == "client" {
		// Clients see only their own cases
		query = query.Where("client_id = ?", currentUser.ID)
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
	// Apply role-based filter
	if currentUser.Role == "lawyer" {
		// Lawyers see cases assigned to them OR where they are collaborators
		query = query.Where(
			db.DB.Where("assigned_to_id = ?", currentUser.ID).
				Or("EXISTS (SELECT 1 FROM case_collaborators WHERE case_collaborators.case_id = cases.id AND case_collaborators.user_id = ?)", currentUser.ID),
		)
	} else if currentUser.Role == "client" {
		// Clients see only their own cases
		query = query.Where("client_id = ?", currentUser.ID)
	}

	// Fetch case with all relationships
	var caseRecord models.Case
	if err := query.
		Preload("Client").
		Preload("Client.DocumentType").
		Preload("AssignedTo").
		Preload("Domain").
		Preload("Branch").
		Preload("Subtypes").
		Preload("Documents").
		Preload("Collaborators").
		Preload("OpposingParty").
		Preload("OpposingParty.DocumentType").
		First(&caseRecord, "id = ?", id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Render detail page
	csrfToken := middleware.GetCSRFToken(c)
	component := pages.CaseDetail(c.Request().Context(), "Case Details | LexLegal Cloud", csrfToken, currentUser, currentFirm, caseRecord)
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
	} else if currentUser.Role == "client" {
		caseQuery = caseQuery.Where("client_id = ?", currentUser.ID)
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
	} else if currentUser.Role == "client" {
		caseQuery = caseQuery.Where("client_id = ?", currentUser.ID)
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

	// Check if using R2 storage (file path is a storage key, not a local path)
	if _, ok := services.Storage.(*services.R2Storage); ok {
		// Generate signed URL for R2 download (valid for 15 minutes)
		signedURL, err := services.Storage.GetSignedURL(context.Background(), document.FilePath, 15*time.Minute)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate download URL")
		}
		return c.Redirect(http.StatusTemporaryRedirect, signedURL)
	}

	// Local storage: verify file path is within upload directory (security check)
	uploadDir := "uploads"
	absUploadDir, err := filepath.Abs(uploadDir)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify file path")
	}
	localPath := filepath.Join(uploadDir, document.FilePath)
	absFilePath, err := filepath.Abs(localPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify file path")
	}
	if !strings.HasPrefix(absFilePath, absUploadDir) {
		return echo.NewHTTPError(http.StatusForbidden, "Invalid file path")
	}

	// Set the Content-Disposition header to suggest the original filename
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+document.FileOriginalName+"\"")
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	c.Response().Header().Set("X-Download-Options", "noopen")
	c.Response().Header().Set("X-Permitted-Cross-Domain-Policies", "none")

	// Serve file from local storage
	return c.File(localPath)
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
	} else if currentUser.Role == "client" {
		caseQuery = caseQuery.Where("client_id = ?", currentUser.ID)
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

	// Get file from storage (works for both R2 and local)
	reader, contentType, err := services.Storage.Get(context.Background(), document.FilePath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve file")
	}
	defer reader.Close()

	// Set headers for inline PDF display
	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Content-Disposition", "inline; filename=\""+document.FileOriginalName+"\"")
	c.Response().Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")

	// Stream the file to the response
	return c.Stream(http.StatusOK, contentType, reader)
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
	} else if currentUser.Role == "client" {
		caseQuery = caseQuery.Where("client_id = ?", currentUser.ID)
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

	// Check storage limit before uploading
	limitResult, err := services.CanUploadFile(db.DB, currentFirm.ID, file.Size)
	if err != nil {
		if err == services.ErrStorageLimitReached {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusForbidden, `
					<div class="p-4 bg-warning/20 text-warning rounded-lg">
						<p class="font-bold">Storage Limit Reached</p>
						<p class="text-sm">`+limitResult.Message+`</p>
						<a href="/firm/settings#subscription" class="btn btn-sm btn-primary mt-2">Upgrade Plan</a>
					</div>
				`)
			}
			return echo.NewHTTPError(http.StatusForbidden, limitResult.Message)
		}
		if err == services.ErrSubscriptionExpired {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusForbidden, `
					<div class="p-4 bg-error/20 text-error rounded-lg">
						<p class="font-bold">Subscription Expired</p>
						<p class="text-sm">Your subscription has expired. Please renew to continue.</p>
						<a href="/firm/settings#subscription" class="btn btn-sm btn-primary mt-2">Renew Now</a>
					</div>
				`)
			}
			return echo.NewHTTPError(http.StatusForbidden, "Subscription has expired")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check storage limits")
	}

	// Validate file
	if err := services.ValidateDocumentUpload(file); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">`+err.Error()+`</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Generate storage key and upload file
	storageKey := services.GenerateCaseDocumentKey(currentFirm.ID, caseID, file.Filename)
	uploadResult, err := services.Storage.Upload(context.Background(), file, storageKey)
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
		FileOriginalName: file.Filename,
		FilePath:         uploadResult.Key, // Storage key for R2 or local path
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
		services.Storage.Delete(context.Background(), uploadResult.Key)
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to save document</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save document")
	}

	// Update storage usage
	if err := services.UpdateFirmUsageAfterStorageChange(db.DB, currentFirm.ID, uploadResult.FileSize); err != nil {
		// Log but don't fail - usage will be recalculated on next check
		services.LogSecurityEvent(db.DB, "USAGE_UPDATE_FAILED", currentUser.ID, "Failed to update storage: "+err.Error())
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
			<div class="p-4 bg-green-500/20 text-green-400 rounded-lg mb-4" x-init="setTimeout(() => { closeUploadModal(); htmx.trigger('#case-documents-list', 'loadDocuments'); }, 1000)">
				Document uploaded successfully!
			</div>
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
	} else if currentUser.Role == "client" {
		caseQuery = caseQuery.Where("client_id = ?", currentUser.ID)
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

// CreateCaseModalHandler renders the create case modal
func CreateCaseModalHandler(c echo.Context) error {
	currentFirm := middleware.GetCurrentFirm(c)

	// Fetch clients
	var clients []models.User
	if err := db.DB.Where("firm_id = ? AND role = ?", currentFirm.ID, "client").Order("name ASC").Find(&clients).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch clients")
	}

	// Fetch lawyers (for assignment)
	var lawyers []models.User
	if err := db.DB.Where("firm_id = ? AND role IN (?, ?)", currentFirm.ID, "lawyer", "admin").Order("name ASC").Find(&lawyers).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch lawyers")
	}

	// Fetch domains for classification
	var domains []models.CaseDomain
	if err := db.DB.Where("firm_id = ? AND is_active = ?", currentFirm.ID, true).Order("`order` ASC, name ASC").Find(&domains).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch domains")
	}

	currentUser := middleware.GetCurrentUser(c)

	component := partials.CaseCreateModal(c.Request().Context(), currentUser, clients, lawyers, domains)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// CreateCaseHandler handles the creation of a new case
func CreateCaseHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	// Check case limit before creating
	limitResult, err := services.CanAddCase(db.DB, currentFirm.ID)
	if err != nil {
		if err == services.ErrCaseLimitReached {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusForbidden, `
					<div class="alert alert-warning shadow-lg">
						<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
						</svg>
						<div>
							<h3 class="font-bold">Case Limit Reached</h3>
							<div class="text-xs">`+limitResult.Message+`</div>
						</div>
						<a href="/firm/settings#subscription" class="btn btn-sm btn-primary">Upgrade Plan</a>
					</div>
				`)
			}
			return echo.NewHTTPError(http.StatusForbidden, limitResult.Message)
		}
		if err == services.ErrSubscriptionExpired {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusForbidden, `
					<div class="alert alert-error shadow-lg">
						<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"/>
						</svg>
						<div>
							<h3 class="font-bold">Subscription Expired</h3>
							<div class="text-xs">Your subscription has expired. Please renew to continue.</div>
						</div>
						<a href="/firm/settings#subscription" class="btn btn-sm btn-primary">Renew Now</a>
					</div>
				`)
			}
			return echo.NewHTTPError(http.StatusForbidden, "Subscription has expired")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check subscription limits")
	}

	// Parse form
	clientID := c.FormValue("client_id")
	clientRole := c.FormValue("client_role")
	title := c.FormValue("title")
	filingNumber := c.FormValue("filing_number")
	description := c.FormValue("description")
	assignedToID := c.FormValue("assigned_to_id")

	// Classification
	domainID := c.FormValue("domain_id")
	branchID := c.FormValue("branch_id")
	subtypeIDs := c.Request().Form["subtype_ids[]"]

	// Validation
	if clientID == "" || clientRole == "" || description == "" || domainID == "" || branchID == "" || assignedToID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required fields")
	}

	// Length Validation
	if len(title) > 255 {
		return echo.NewHTTPError(http.StatusBadRequest, "Title must be less than 255 characters")
	}
	if len(filingNumber) > 24 {
		return echo.NewHTTPError(http.StatusBadRequest, "Filing number must be less than 24 characters")
	}
	if len(description) > 5000 {
		return echo.NewHTTPError(http.StatusBadRequest, "Description must be less than 5000 characters")
	}

	// Generate unique case number
	caseNumber, err := services.EnsureUniqueCaseNumber(db.DB, currentFirm.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate case number")
	}

	now := time.Now()

	// Create Case Model
	newCase := models.Case{
		FirmID:          currentFirm.ID,
		ClientID:        clientID,
		CaseNumber:      caseNumber,
		CaseType:        "General", // Default value as we use classification now
		Description:     description,
		Status:          models.CaseStatusOpen,
		OpenedAt:        now,
		StatusChangedBy: &currentUser.ID,
		StatusChangedAt: &now,
		DomainID:        &domainID,
		BranchID:        &branchID,
	}

	if title != "" {
		newCase.Title = &title
	}
	if filingNumber != "" {
		newCase.FilingNumber = &filingNumber
	}
	if clientRole != "" {
		newCase.ClientRole = &clientRole
	}
	if assignedToID != "" {
		newCase.AssignedToID = &assignedToID
	}

	tx := db.DB.Begin()

	if err := tx.Create(&newCase).Error; err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create case: "+err.Error())
	}

	// Link subtypes if provided
	if len(subtypeIDs) > 0 {
		var subtypes []models.CaseSubtype
		if err := tx.Where("id IN ?", subtypeIDs).Find(&subtypes).Error; err != nil {
			tx.Rollback()
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch subtypes")
		}
		if err := tx.Model(&newCase).Association("Subtypes").Append(subtypes); err != nil {
			tx.Rollback()
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to link subtypes")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to commit case")
	}

	// Update usage cache
	if err := services.UpdateFirmUsageAfterCaseChange(db.DB, currentFirm.ID, 1); err != nil {
		// Log but don't fail - usage will be recalculated on next check
		services.LogSecurityEvent(db.DB, "USAGE_UPDATE_FAILED", currentUser.ID, "Failed to update case count: "+err.Error())
	}

	// Audit logging
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(
		db.DB,
		auditCtx,
		models.AuditActionCreate,
		"Case",
		newCase.ID,
		newCase.CaseNumber,
		"Case created manually",
		nil,
		newCase,
	)

	// Trigger reload of table via HTMX header
	c.Response().Header().Set("HX-Trigger", "reload-cases")

	return c.NoContent(http.StatusOK)
}
