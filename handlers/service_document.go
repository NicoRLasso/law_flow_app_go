package handlers

import (
	"context"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/partials"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// GetServiceDocumentsHandler lists all documents for a service
func GetServiceDocumentsHandler(c echo.Context) error {
	serviceID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	// Verify access
	service, err := services.GetServiceByID(db.DB, currentFirm.ID, serviceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Service not found")
	}
	if currentUser.Role == "client" && service.ClientID != currentUser.ID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	var documents []models.ServiceDocument
	query := db.DB.Where("firm_id = ? AND service_id = ?", currentFirm.ID, serviceID)

	// Clients only see public documents
	if currentUser.Role == "client" {
		query = query.Where("is_public = ?", true)
	}

	// Filter parameters
	if keyword := c.QueryParam("keyword"); keyword != "" {
		query = query.Where("file_original_name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if docType := c.QueryParam("document_type"); docType != "" {
		query = query.Where("document_type = ?", docType)
	}

	var total int64
	if err := query.Model(&models.ServiceDocument{}).Count(&total).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count documents")
	}

	// Pagination parameters
	page := 1
	limit := 10
	if p, err := strconv.Atoi(c.QueryParam("page")); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(c.QueryParam("limit")); err == nil && l > 0 {
		limit = l
	}
	offset := (page - 1) * limit
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	if err := query.Preload("UploadedBy").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&documents).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch documents")
	}

	// Note: partials.ServiceDocumentTable will be created in Phase 4
	component := partials.ServiceDocumentTable(c.Request().Context(), documents, page, totalPages, limit, int(total), serviceID, currentUser.Role != "client")
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// UploadServiceDocumentHandler handles file uploads
func UploadServiceDocumentHandler(c echo.Context) error {
	serviceID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	// Verify service exists
	if _, err := services.GetServiceByID(db.DB, currentFirm.ID, serviceID); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Service not found")
	}

	// Get form
	description := c.FormValue("description")
	docType := c.FormValue("document_type")
	if docType == "" {
		docType = "REFERENCE" // Default
	}

	// Get visibility setting
	isPublic := c.FormValue("is_public") == "true" || c.FormValue("is_public") == "on"

	// Get file
	file, err := c.FormFile("file")
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">File is required</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "File is required")
	}

	// Limits check
	if _, err := services.CanUploadFile(db.DB, currentFirm.ID, file.Size); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusForbidden, `<div class="p-4 bg-warning/20 text-warning rounded-lg">Storage limit reached</div>`)
		}
		return echo.NewHTTPError(http.StatusForbidden, "Storage limit reached")
	}

	// Upload
	key := services.GenerateServiceDocumentKey(currentFirm.ID, serviceID, file.Filename)
	uploadResult, err := services.Storage.Upload(context.Background(), file, key)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Upload failed</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Upload failed")
	}

	// Record in DB
	doc := models.ServiceDocument{
		FirmID:           currentFirm.ID,
		ServiceID:        serviceID,
		FileName:         uploadResult.FileName,
		FileOriginalName: file.Filename,
		FilePath:         uploadResult.Key,
		FileSize:         uploadResult.FileSize,
		MimeType:         uploadResult.MimeType,
		DocumentType:     docType,
		Description:      &description,
		IsPublic:         isPublic,
		UploadedByID:     &currentUser.ID,
	}

	if currentUser.Role == "client" {
		// Clients uploading -> default behavior? Maybe notify lawyer?
		// For now just save.
	}

	if err := db.DB.Create(&doc).Error; err != nil {
		// Rollback storage?
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save document record")
	}

	// Update storage usage
	services.UpdateFirmUsageAfterStorageChange(db.DB, currentFirm.ID, uploadResult.FileSize)

	// Audit
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionCreate,
		"ServiceDocument", doc.ID, doc.FileOriginalName,
		"Document uploaded", nil, doc)

	return GetServiceDocumentsHandler(c)
}

// DownloadServiceDocumentHandler serves the file
func DownloadServiceDocumentHandler(c echo.Context) error {
	serviceID := c.Param("id")
	docID := c.Param("did")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	var doc models.ServiceDocument
	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, docID, serviceID).First(&doc).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Document not found")
	}

	// Client access check
	if currentUser.Role == "client" && !doc.IsPublic {
		// Also verify client owns the service
		var service models.LegalService
		db.DB.Select("client_id").First(&service, "id = ?", serviceID)
		if service.ClientID != currentUser.ID {
			return echo.NewHTTPError(http.StatusForbidden, "Access denied")
		}
		if !doc.IsPublic {
			// Actually clients should generally see docs they uploaded?
			// but if lawyer uploaded as private, client cant see.
			// assuming 'IsPublic' implies 'Visible to Client'.
			return echo.NewHTTPError(http.StatusForbidden, "Document is private")
		}
	}

	// Audit download
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionDownload,
		"ServiceDocument", doc.ID, doc.FileOriginalName,
		"Document downloaded", nil, nil)

	// Handle R2 signed url vs local
	if _, ok := services.Storage.(*services.R2Storage); ok {
		url, err := services.Storage.GetSignedURL(context.Background(), doc.FilePath, 15*time.Minute)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get download URL")
		}
		return c.Redirect(http.StatusTemporaryRedirect, url)
	}

	// Local
	reader, contentType, err := services.Storage.Get(context.Background(), doc.FilePath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read file")
	}
	defer reader.Close()

	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+doc.FileOriginalName+"\"")
	return c.Stream(http.StatusOK, contentType, reader)
}

// DeleteServiceDocumentHandler deletes a document
func DeleteServiceDocumentHandler(c echo.Context) error {
	serviceID := c.Param("id")
	docID := c.Param("did")
	currentFirm := middleware.GetCurrentFirm(c)

	var doc models.ServiceDocument
	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, docID, serviceID).First(&doc).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Document not found")
	}

	// Delete from storage
	services.Storage.Delete(context.Background(), doc.FilePath)

	// Delete from DB
	if err := db.DB.Delete(&doc).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete record")
	}

	// Audit
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionDelete,
		"ServiceDocument", doc.ID, doc.FileOriginalName,
		"Document deleted", nil, nil)

	return GetServiceDocumentsHandler(c)
}

// ToggleServiceDocumentVisibilityHandler toggles document visibility
func ToggleServiceDocumentVisibilityHandler(c echo.Context) error {
	serviceID := c.Param("id")
	docID := c.Param("did")
	currentFirm := middleware.GetCurrentFirm(c)

	// Verify access
	var doc models.ServiceDocument
	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, docID, serviceID).First(&doc).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Document not found")
	}

	// Toggle
	doc.IsPublic = !doc.IsPublic
	if err := db.DB.Save(&doc).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update visibility")
	}

	// Audit
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionVisibilityChange,
		"ServiceDocument", doc.ID, doc.FileOriginalName,
		"Visibility toggled", map[string]interface{}{"old": !doc.IsPublic}, map[string]interface{}{"new": doc.IsPublic})

	db.DB.Preload("UploadedBy").First(&doc, "id = ?", doc.ID)
	component := partials.ServiceDocumentRow(c.Request().Context(), doc, serviceID)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// ViewServiceDocumentHandler serves a PDF document for inline viewing
func ViewServiceDocumentHandler(c echo.Context) error {
	serviceID := c.Param("id")
	docID := c.Param("did")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	var doc models.ServiceDocument
	if err := db.DB.Where("firm_id = ? AND id = ? AND service_id = ?", currentFirm.ID, docID, serviceID).First(&doc).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Document not found")
	}

	// Client access check
	if currentUser.Role == "client" && !doc.IsPublic {
		var service models.LegalService
		db.DB.Select("client_id").First(&service, "id = ?", serviceID)
		if service.ClientID != currentUser.ID || !doc.IsPublic {
			return echo.NewHTTPError(http.StatusForbidden, "Access denied")
		}
	}

	// Validate it's a PDF
	if !strings.HasSuffix(strings.ToLower(doc.FileOriginalName), ".pdf") {
		return echo.NewHTTPError(http.StatusBadRequest, "Only PDF files can be viewed inline")
	}

	// Audit view
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(db.DB, auditCtx, models.AuditActionView,
		"ServiceDocument", doc.ID, doc.FileOriginalName,
		"Document viewed inline", nil, nil)

	// Get from storage (works for both R2 and local)
	reader, contentType, err := services.Storage.Get(context.Background(), doc.FilePath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve file")
	}
	defer reader.Close()

	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Content-Disposition", "inline; filename=\""+doc.FileOriginalName+"\"")

	// Stream the file to the response
	return c.Stream(http.StatusOK, contentType, reader)
}
