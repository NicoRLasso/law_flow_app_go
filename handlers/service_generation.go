package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/partials"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// GetServiceTemplateModalHandler renders the template selector modal
func GetServiceTemplateModalHandler(c echo.Context) error {
	serviceID := c.Param("id")
	// currentFirm := middleware.GetCurrentFirm(c)

	var service models.LegalService
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		First(&service, "id = ?", serviceID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Service not found")
	}

	// Fetch available active templates for the firm
	var templates []models.DocumentTemplate
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&templates).Error; err != nil {
		// Log error but continue with empty templates
		fmt.Printf("Error fetching templates: %v\n", err)
	}

	component := partials.ServiceTemplateSelectorModal(c.Request().Context(), service.ID, templates)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// PreviewServiceTemplateHandler returns a preview with variables replaced
func PreviewServiceTemplateHandler(c echo.Context) error {
	serviceID := c.Param("id")
	templateID := c.QueryParam("template_id")

	if templateID == "" {
		return c.String(http.StatusBadRequest, "Template ID is required")
	}

	// Get Service with relationships needed for variables
	var service models.LegalService
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		Preload("Client").
		Preload("Client.DocumentType"). // Ensure DocumentType is loaded
		Preload("AssignedTo").
		Preload("ServiceType"). // Load choice option
		First(&service, "id = ?", serviceID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Service not found")
	}

	// Get Template
	var template models.DocumentTemplate
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		First(&template, "id = ?", templateID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Template not found")
	}

	// Get Firm
	firm := c.Get("firm").(*models.Firm)

	// Build Data & Render
	data := services.BuildTemplateDataFromService(&service, firm)
	renderedContent := services.RenderTemplate(template.Content, data)

	// Re-use the existing TemplatePreview partial from document_generation handlers
	// Assuming it's in templates/partials/pdf_viewer_modal.templ or similar...
	// Wait, handlers/document_generation.go uses partials.TemplatePreview(ctx, renderedContent)
	// I need to check where TemplatePreview is defined.
	// Step 224: "return partials.TemplatePreview(ctx, renderedContent).Render..."
	// It is likely in templates/partials/template_selector_modal.templ or generated_documents_table.templ
	// I'll assume partials.TemplatePreview exists as seen in existing code.

	return partials.TemplatePreview(c.Request().Context(), renderedContent).Render(c.Request().Context(), c.Response().Writer)
}

// GenerateServiceDocumentHandler generates the final PDF/Doc
func GenerateServiceDocumentHandler(c echo.Context) error {
	serviceID := c.Param("id")
	templateID := c.FormValue("template_id")
	customName := c.FormValue("name")

	if templateID == "" {
		return c.String(http.StatusBadRequest, "Template ID is required")
	}

	user := c.Get("user").(*models.User)
	firm := c.Get("firm").(*models.Firm)
	firmID := *user.FirmID

	// 1. Fetch Service
	var service models.LegalService
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		Preload("Client").
		Preload("Client.DocumentType").
		Preload("AssignedTo").
		Preload("ServiceType").
		First(&service, "id = ?", serviceID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Service not found")
	}

	// 2. Fetch Template
	var template models.DocumentTemplate
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		First(&template, "id = ?", templateID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Template not found")
	}

	// 3. Render Content
	data := services.BuildTemplateDataFromService(&service, firm)
	finalContent := services.RenderTemplate(template.Content, data)

	// 4. Generate Name
	documentName := customName
	if documentName == "" {
		documentName = fmt.Sprintf("%s - %s", template.Name, service.ServiceNumber)
	}

	// 5. Generate PDF
	pdfOptions := services.PDFOptions{
		PageOrientation: template.PageOrientation,
		PageSize:        template.PageSize,
		MarginTop:       template.MarginTop,
		MarginBottom:    template.MarginBottom,
		MarginLeft:      template.MarginLeft,
		MarginRight:     template.MarginRight,
	}

	pdfBytes, err := services.GeneratePDFFromTemplate(finalContent, pdfOptions)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error generating PDF: "+err.Error())
	}

	// 6. Upload/Save File
	fileName := fmt.Sprintf("%s_%d.pdf", uuid.New().String(), time.Now().Unix())
	// Reuse existing key generation or make a new one strict for services?
	// existing: services.GenerateGeneratedDocumentKey(firmID, caseID, fileName)
	// We should probably have GenerateServiceDocumentKey.
	// For now, let's use a generic path: firms/{firmID}/services/{serviceID}/documents/{fileName}
	storageKey := fmt.Sprintf("firms/%s/services/%s/documents/%s", firmID, serviceID, fileName)

	reader := bytes.NewReader(pdfBytes)
	uploadResult, err := services.Storage.UploadReader(c.Request().Context(), reader, storageKey, "application/pdf", int64(len(pdfBytes)))
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error saving PDF: "+err.Error())
	}

	// Check if uploadResult.Key is what we need. Yes.
	filePath := uploadResult.Key

	// 7. Save to ServiceDocument
	serviceDoc := models.ServiceDocument{
		FirmID:                  firmID,
		ServiceID:               serviceID,
		FileName:                fileName,
		FileOriginalName:        documentName + ".pdf",
		FilePath:                filePath,
		FileSize:                int64(len(pdfBytes)),
		MimeType:                "application/pdf",
		DocumentType:            models.ServiceDocTypeDeliverable, // Default for generated docs
		Description:             nil,                              // Could add "Generated from Template X"
		IsPublic:                false,
		GeneratedFromTemplateID: &template.ID,
		UploadedByID:            &user.ID,
	}

	if err := db.DB.Create(&serviceDoc).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error saving document record")
	}

	// 8. Return Updated Documents List
	// This will replace the content of #service-documents-list due to hx-swap on the form and after-request trigger if needed.
	// However, the modal uses HX-Target="#service-documents-list".
	// So if we return the list partial here, it will update the list directly.
	return GetServiceDocumentsHandler(c)
}
