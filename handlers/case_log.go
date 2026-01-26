package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/templates/partials"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

func render(c echo.Context, component templ.Component) error {
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// GetCaseLogsHandler handles fetching case logs with optional filters
func GetCaseLogsHandler(c echo.Context) error {
	caseID := c.Param("id")
	entryType := c.QueryParam("type")
	search := c.QueryParam("search")

	// Pagination
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

	// Verify case belongs to firm
	var caseRecord models.Case
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&caseRecord, "id = ?", caseID).Error; err != nil {
		return c.String(http.StatusNotFound, "Case not found")
	}

	var logs []models.CaseLog
	var total int64
	query := middleware.GetFirmScopedQuery(c, db.DB).Model(&models.CaseLog{}).Where("case_id = ?", caseID)

	if entryType != "" && entryType != "all" {
		query = query.Where("entry_type = ?", entryType)
	}

	if search != "" {
		likeSearch := "%" + search + "%"
		query = query.Where("title LIKE ? OR content LIKE ?", likeSearch, likeSearch)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error counting logs")
	}

	// Fetch paginated
	if err := query.Order("occurred_at DESC, created_at DESC").Limit(limit).Offset((page - 1) * limit).Find(&logs).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error fetching logs")
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// If targeting the container directly (e.g. from filter/search), return just the list
	if c.Request().Header.Get("HX-Target") == "case-logs-container" {
		return render(c, partials.CaseLogList(context.Background(), logs, caseID, page, totalPages, limit, int(total)))
	}

	return render(c, partials.CaseLogTable(context.Background(), logs, caseID, page, totalPages, limit, int(total)))
}

// GetCaseLogFormHandler returns the form for creating a new log entry
func GetCaseLogFormHandler(c echo.Context) error {
	caseID := c.Param("id")

	// We need to fetch documents to populate the select dropdown
	// Verify case belongs to firm
	var caseRecord models.Case
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&caseRecord, "id = ?", caseID).Error; err != nil {
		return c.String(http.StatusNotFound, "Case not found")
	}

	// We need to fetch documents to populate the select dropdown (firm scoped)
	var documents []models.CaseDocument
	if err := middleware.GetFirmScopedQuery(c, db.DB).Where("case_id = ?", caseID).Find(&documents).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error fetching documents")
	}

	return render(c, partials.CaseLogModal(c.Request().Context(), models.CaseLog{CaseID: caseID}, documents, true)) // true = isNew
}

// CreateCaseLogHandler handles the creation of a new log entry
func CreateCaseLogHandler(c echo.Context) error {
	caseID := c.Param("id")
	user := c.Get("user").(*models.User)
	firmID := user.FirmID

	// Verify case belongs to firm
	var caseRecord models.Case
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&caseRecord, "id = ?", caseID).Error; err != nil {
		return c.String(http.StatusNotFound, "Case not found")
	}

	entryType := c.FormValue("entry_type")
	title := c.FormValue("title")
	content := c.FormValue("content")
	documentIDStr := c.FormValue("document_id")
	contactName := c.FormValue("contact_name")
	contactPhone := c.FormValue("contact_phone")
	occurredAtStr := c.FormValue("occurred_at")
	durationStr := c.FormValue("duration")

	// Validation
	if title == "" {
		return c.String(http.StatusBadRequest, "Title is required")
	}

	logEntry := models.CaseLog{
		FirmID:      *firmID,
		CaseID:      caseID,
		EntryType:   entryType,
		Title:       title,
		Content:     content,
		CreatedByID: user.ID,
	}

	if documentIDStr != "" {
		logEntry.DocumentID = &documentIDStr
	}

	if contactName != "" {
		logEntry.ContactName = &contactName
	}

	if contactPhone != "" {
		logEntry.ContactPhone = &contactPhone
	}

	if occurredAtStr != "" {
		// Assuming HTML datetime-local input: "2006-01-02T15:04"
		parsedTime, err := time.Parse("2006-01-02T15:04", occurredAtStr)
		if err == nil {
			logEntry.OccurredAt = &parsedTime
		}
	} else {
		now := time.Now()
		logEntry.OccurredAt = &now
	}

	if durationStr != "" {
		duration, err := strconv.Atoi(durationStr)
		if err == nil {
			logEntry.Duration = &duration
		}
	}

	if err := db.DB.Create(&logEntry).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error creating log entry")
	}

	return fetchAndRenderLogs(c, caseID)
}

// GetCaseLogHandler returns a specific log entry (for editing usually, reusing the modal)
func GetCaseLogHandler(c echo.Context) error {
	id := c.Param("logId")
	caseID := c.Param("id")

	var logEntry models.CaseLog
	// Use firm-scoped query to prevent IDOR
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&logEntry, "id = ?", id).Error; err != nil {
		return c.String(http.StatusNotFound, "Log entry not found")
	}

	var documents []models.CaseDocument
	if err := middleware.GetFirmScopedQuery(c, db.DB).Where("case_id = ?", caseID).Find(&documents).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error fetching documents")
	}

	return render(c, partials.CaseLogModal(c.Request().Context(), logEntry, documents, false)) // false = isNew
}

// GetCaseLogViewHandler returns a read-only view modal for a specific log entry
func GetCaseLogViewHandler(c echo.Context) error {
	id := c.Param("logId")

	var logEntry models.CaseLog
	// Use firm-scoped query to prevent IDOR, preload Document if present
	if err := middleware.GetFirmScopedQuery(c, db.DB).Preload("Document").First(&logEntry, "id = ?", id).Error; err != nil {
		return c.String(http.StatusNotFound, "Log entry not found")
	}

	return render(c, partials.CaseLogViewModal(c.Request().Context(), logEntry))
}

// helper to fetch logs and render the table
// helper to fetch logs and render the table
func fetchAndRenderLogs(c echo.Context, caseID string) error {
	var logs []models.CaseLog
	var total int64
	page := 1
	limit := 10 // Default limit for re-render after action

	// Use firm-scoped query
	query := middleware.GetFirmScopedQuery(c, db.DB).Model(&models.CaseLog{}).Where("case_id = ?", caseID)

	if err := query.Count(&total).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error counting logs")
	}

	if err := query.Order("occurred_at DESC, created_at DESC").Limit(limit).Offset(0).Find(&logs).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error fetching logs")
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Always return list for updates/creations/deletions as they target the container
	return render(c, partials.CaseLogList(context.Background(), logs, caseID, page, totalPages, limit, int(total)))
}

// UpdateCaseLogHandler updates an existing log entry
func UpdateCaseLogHandler(c echo.Context) error {
	id := c.Param("logId")
	// caseID := c.Param("id")

	var logEntry models.CaseLog
	// Use firm-scoped query to prevent IDOR
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&logEntry, "id = ?", id).Error; err != nil {
		return c.String(http.StatusNotFound, "Log entry not found")
	}

	entryType := c.FormValue("entry_type")
	title := c.FormValue("title")
	content := c.FormValue("content")
	documentIDStr := c.FormValue("document_id")
	contactName := c.FormValue("contact_name")
	contactPhone := c.FormValue("contact_phone")
	occurredAtStr := c.FormValue("occurred_at")
	durationStr := c.FormValue("duration")

	if title == "" {
		return c.String(http.StatusBadRequest, "Title is required")
	}

	logEntry.EntryType = entryType
	logEntry.Title = title
	logEntry.Content = content

	if documentIDStr != "" {
		logEntry.DocumentID = &documentIDStr
	} else {
		logEntry.DocumentID = nil
	}

	if contactName != "" {
		logEntry.ContactName = &contactName
	} else {
		logEntry.ContactName = nil
	}

	if contactPhone != "" {
		logEntry.ContactPhone = &contactPhone
	} else {
		logEntry.ContactPhone = nil
	}

	if occurredAtStr != "" {
		parsedTime, err := time.Parse("2006-01-02T15:04", occurredAtStr)
		if err == nil {
			logEntry.OccurredAt = &parsedTime
		}
	}

	if durationStr != "" {
		duration, err := strconv.Atoi(durationStr)
		if err == nil {
			logEntry.Duration = &duration
		}
	} else {
		logEntry.Duration = nil
	}

	if err := db.DB.Save(&logEntry).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error updating log entry")
	}

	// For update, the modal targets #case-logs-container, so we need to return the list
	// Ideally we would just replace the card, but list refresh is safer for ordering
	return fetchAndRenderLogs(c, logEntry.CaseID)
}

// DeleteCaseLogHandler deletes a log entry
func DeleteCaseLogHandler(c echo.Context) error {
	id := c.Param("logId")

	// Use firm-scoped query to ensure user owns the log entry
	var logEntry models.CaseLog
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&logEntry, "id = ?", id).Error; err != nil {
		return c.String(http.StatusNotFound, "Log entry not found")
	}

	if err := db.DB.Delete(&logEntry).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error deleting log entry")
	}

	// Make sure to return the updated list so HTMX can swap the container content
	return fetchAndRenderLogs(c, logEntry.CaseID)
}
