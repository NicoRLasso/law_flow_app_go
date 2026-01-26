package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

// DeleteCaseDocumentHandler handles the deletion of a case document
func DeleteCaseDocumentHandler(c echo.Context) error {
	caseID := c.Param("id")
	docID := c.Param("docId")
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)

	// Only admin and lawyer can delete documents
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

	// Fetch document metadata for audit logging before deletion
	var document models.CaseDocument
	if err := db.DB.Where("id = ? AND firm_id = ?", docID, currentFirm.ID).First(&document).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Document not found")
	}

	// Store file size before deletion for usage update
	deletedFileSize := document.FileSize

	// Perform deletion
	if err := services.DeleteCaseDocument(docID, currentUser.ID, currentFirm.ID); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Failed to delete document</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete document")
	}

	// Update storage usage (decrease by deleted file size)
	if err := services.UpdateFirmUsageAfterStorageChange(db.DB, currentFirm.ID, -deletedFileSize); err != nil {
		// Log but don't fail - usage will be recalculated on next check
		services.LogSecurityEvent("USAGE_UPDATE_FAILED", currentUser.ID, "Failed to update storage after delete: "+err.Error())
	}

	// Audit logging (Delete)
	auditCtx := middleware.GetAuditContext(c)
	services.LogAuditEvent(
		db.DB,
		auditCtx,
		models.AuditActionDelete,
		"CaseDocument",
		docID,
		document.FileOriginalName,
		"Document deleted",
		document, // Old state
		nil,      // New state
	)

	// Return empty string to remove the row from the table (HTMX swap)
	return c.String(http.StatusOK, "")
}
