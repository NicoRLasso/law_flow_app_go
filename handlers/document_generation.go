package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/partials"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// GetGenerateDocumentTabHandler returns the generate document tab content for case detail
func GetGenerateDocumentTabHandler(c echo.Context) error {
	caseID := c.Param("id")
	ctx := context.Background()

	// Verify case belongs to firm
	var caseRecord models.Case
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		Preload("Client").
		Preload("AssignedTo").
		Preload("Domain").
		Preload("Branch").
		Preload("Subtypes").
		First(&caseRecord, "id = ?", caseID).Error; err != nil {
		return c.String(http.StatusNotFound, "Case not found")
	}

	// Get available templates
	var templates []models.DocumentTemplate
	middleware.GetFirmScopedQuery(c, db.DB).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&templates)

	// Get generated documents for this case
	var generatedDocs []models.GeneratedDocument
	middleware.GetFirmScopedQuery(c, db.DB).
		Where("case_id = ?", caseID).
		Preload("Template").
		Preload("GeneratedBy").
		Order("created_at DESC").
		Find(&generatedDocs)

	return partials.GenerateDocumentTab(ctx, caseRecord, templates, generatedDocs).Render(c.Request().Context(), c.Response().Writer)
}

// PreviewTemplateHandler renders a template preview with case data
func PreviewTemplateHandler(c echo.Context) error {
	caseID := c.Param("id")
	templateID := c.QueryParam("template_id")

	if templateID == "" {
		return c.String(http.StatusBadRequest, "Template ID is required")
	}

	// Get case with all relationships
	var caseRecord models.Case
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		Preload("Client").
		Preload("Client.DocumentType").
		Preload("AssignedTo").
		Preload("Domain").
		Preload("Branch").
		Preload("Subtypes").
		First(&caseRecord, "id = ?", caseID).Error; err != nil {
		return c.String(http.StatusNotFound, "Case not found")
	}

	// Get template
	var template models.DocumentTemplate
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&template, "id = ?", templateID).Error; err != nil {
		return c.String(http.StatusNotFound, "Template not found")
	}

	// Get firm
	firm := c.Get("firm").(*models.Firm)

	// Build template data and render
	data := services.BuildTemplateDataFromCase(&caseRecord, firm)
	renderedContent := services.RenderTemplate(template.Content, data)

	// Return rendered HTML for preview
	ctx := context.Background()
	return partials.TemplatePreview(ctx, renderedContent).Render(c.Request().Context(), c.Response().Writer)
}

// GenerateDocumentHandler generates a PDF document from a template
func GenerateDocumentHandler(c echo.Context) error {
	caseID := c.Param("id")
	templateID := c.FormValue("template_id")
	documentName := c.FormValue("name")
	finalContent := c.FormValue("content") // Optional: edited content from preview

	user := c.Get("user").(*models.User)
	firm := c.Get("firm").(*models.Firm)
	firmID := *user.FirmID

	if templateID == "" {
		return c.String(http.StatusBadRequest, "Template ID is required")
	}

	// Get case with all relationships
	var caseRecord models.Case
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		Preload("Client").
		Preload("Client.DocumentType").
		Preload("AssignedTo").
		Preload("Domain").
		Preload("Branch").
		Preload("Subtypes").
		First(&caseRecord, "id = ?", caseID).Error; err != nil {
		return c.String(http.StatusNotFound, "Case not found")
	}

	// Get template
	var template models.DocumentTemplate
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&template, "id = ?", templateID).Error; err != nil {
		return c.String(http.StatusNotFound, "Template not found")
	}

	// If no custom content provided, render from template
	if finalContent == "" {
		data := services.BuildTemplateDataFromCase(&caseRecord, firm)
		finalContent = services.RenderTemplate(template.Content, data)
	}

	// Generate document name if not provided
	if documentName == "" {
		documentName = fmt.Sprintf("%s - %s", template.Name, caseRecord.CaseNumber)
	}

	// Generate PDF
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

	// Save PDF to filesystem
	uploadDir := filepath.Join("uploads", "firms", firmID, "cases", caseID, "generated")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return c.String(http.StatusInternalServerError, "Error creating directory")
	}

	fileName := fmt.Sprintf("%s_%d.pdf", uuid.New().String(), time.Now().Unix())
	filePath := filepath.Join(uploadDir, fileName)

	if err := os.WriteFile(filePath, pdfBytes, 0644); err != nil {
		return c.String(http.StatusInternalServerError, "Error saving PDF")
	}

	// Create GeneratedDocument record
	generatedDoc := models.GeneratedDocument{
		FirmID:          firmID,
		TemplateID:      template.ID,
		TemplateVersion: template.Version,
		CaseID:          caseID,
		Name:            documentName,
		FinalContent:    finalContent,
		FileName:        fileName,
		FilePath:        filePath,
		FileSize:        int64(len(pdfBytes)),
		GeneratedByID:   user.ID,
	}

	if err := db.DB.Create(&generatedDoc).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error saving document record")
	}

	// Auto-archive: Create a CaseDocument record
	archiveDoc := models.CaseDocument{
		FirmID:           firmID,
		CaseID:           &caseID,
		FileName:         fileName,
		FileOriginalName: documentName + ".pdf",
		FilePath:         filePath,
		FileSize:         int64(len(pdfBytes)),
		MimeType:         "application/pdf",
		DocumentType:     "generated",
		UploadedByID:     &user.ID,
		IsPublic:         false,
	}

	if err := db.DB.Create(&archiveDoc).Error; err != nil {
		// Non-fatal: log but continue
		fmt.Printf("Warning: could not auto-archive document: %v\n", err)
	} else {
		// Link the archive document
		generatedDoc.CaseDocumentID = &archiveDoc.ID
		db.DB.Save(&generatedDoc)
	}

	// Return updated generated documents list
	return GetGeneratedDocumentsHandler(c)
}

// GetGeneratedDocumentsHandler returns the list of generated documents for a case
func GetGeneratedDocumentsHandler(c echo.Context) error {
	caseID := c.Param("id")
	ctx := context.Background()

	var docs []models.GeneratedDocument
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		Where("case_id = ?", caseID).
		Preload("Template").
		Preload("GeneratedBy").
		Order("created_at DESC").
		Find(&docs).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error fetching documents")
	}

	return partials.GeneratedDocumentsTable(ctx, docs, caseID).Render(c.Request().Context(), c.Response().Writer)
}

// DownloadGeneratedDocumentHandler serves a generated document for download
func DownloadGeneratedDocumentHandler(c echo.Context) error {
	caseID := c.Param("id")
	docID := c.Param("docId")

	var doc models.GeneratedDocument
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		Where("case_id = ?", caseID).
		First(&doc, "id = ?", docID).Error; err != nil {
		return c.String(http.StatusNotFound, "Document not found")
	}

	// Check file exists
	if _, err := os.Stat(doc.FilePath); os.IsNotExist(err) {
		return c.String(http.StatusNotFound, "File not found")
	}

	return c.Attachment(doc.FilePath, doc.Name+".pdf")
}
