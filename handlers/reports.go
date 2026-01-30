package handlers

import (
	"encoding/csv"
	"fmt"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// ExportReportHandler handles CSV export requests
func ExportReportHandler(c echo.Context) error {
	firm := middleware.GetCurrentFirm(c)
	if firm == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	reportType := c.FormValue("report_type")
	startDateStr := c.FormValue("start_date")
	endDateStr := c.FormValue("end_date")
	clientID := c.FormValue("client_id")
	lawyerID := c.FormValue("lawyer_id")
	status := c.FormValue("status")

	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_report_%s.csv", reportType, time.Now().Format("20060102_150405")))

	writer := csv.NewWriter(c.Response().Writer)
	defer writer.Flush()

	switch reportType {
	case "cases":
		return exportCases(firm.ID, startDateStr, endDateStr, clientID, lawyerID, status, writer)
	case "services":
		return exportServices(firm.ID, startDateStr, endDateStr, clientID, lawyerID, status, writer)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid report type")
	}
}

func exportCases(firmID, startDate, endDate, clientID, lawyerID, status string, w *csv.Writer) error {
	// Header
	header := []string{
		"Case Number", "Title", "Status", "Opened At", "Closed At",
		"Client Name", "Client Email", "Assigned To", "Description",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	query := db.DB.Model(&models.Case{}).Where("firm_id = ?", firmID).
		Preload("Client").Preload("AssignedTo")

	// Apply filters
	if startDate != "" {
		query = query.Where("opened_at >= ?", startDate)
	}
	if endDate != "" {
		// Add time to make it inclusive of the end date
		query = query.Where("opened_at <= ?", endDate+" 23:59:59")
	}
	if clientID != "" {
		query = query.Where("client_id = ?", clientID)
	}
	if lawyerID != "" {
		query = query.Where("assigned_to_id = ?", lawyerID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	rows, err := query.Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var c models.Case
		db.DB.ScanRows(rows, &c)

		// Manual Preload hack if rows.Next() is used without Find() slices
		// Actually, using Find() with batching is safer for CSV export to avoid memory bloat,
		// but GORM's FindInBatches is also an option. For simplicity and standard usage:
		// We'll re-query or just use a loop with offsets if dataset is huge.
		// For now, standard Find() into slice might be okay for typical law firm size, but let's be robust.
	}
	// Re-approach: Use FindInBatches to be memory efficient

	var cases []models.Case
	// Ignoring the rows iterator for a cleaner batch approach
	batchSize := 100
	result := query.FindInBatches(&cases, batchSize, func(tx *gorm.DB, batch int) error {
		for _, c := range cases {
			closedAt := ""
			if c.ClosedAt != nil {
				closedAt = c.ClosedAt.Format("2006-01-02")
			}
			assignedTo := ""
			if c.AssignedTo != nil {
				assignedTo = c.AssignedTo.Name
			}

			record := []string{
				c.CaseNumber,
				safeString(c.Title),
				c.Status,
				c.OpenedAt.Format("2006-01-02"),
				closedAt,
				c.Client.Name,
				c.Client.Email,
				assignedTo,
				c.Description,
			}
			if err := w.Write(record); err != nil {
				return err
			}
		}
		return nil
	})

	return result.Error
}

func exportServices(firmID, startDate, endDate, clientID, lawyerID, status string, w *csv.Writer) error {
	// Header
	header := []string{
		"Service Number", "Title", "Status", "Priority", "Created At",
		"Client Name", "Assigned To", "Actual Hours", "Estimated Hours",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	query := db.DB.Model(&models.LegalService{}).Where("firm_id = ?", firmID).
		Preload("Client").Preload("AssignedTo")

	// Apply filters
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}
	if clientID != "" {
		query = query.Where("client_id = ?", clientID)
	}
	if lawyerID != "" {
		query = query.Where("assigned_to_id = ?", lawyerID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var services []models.LegalService
	batchSize := 100
	result := query.FindInBatches(&services, batchSize, func(tx *gorm.DB, batch int) error {
		for _, s := range services {
			assignedTo := ""
			if s.AssignedTo != nil {
				assignedTo = s.AssignedTo.Name
			}
			estHours := ""
			if s.EstimatedHours != nil {
				estHours = fmt.Sprintf("%.2f", *s.EstimatedHours)
			}

			record := []string{
				s.ServiceNumber,
				s.Title,
				s.Status,
				s.Priority,
				s.CreatedAt.Format("2006-01-02"),
				s.Client.Name,
				assignedTo,
				fmt.Sprintf("%.2f", s.ActualHours),
				estHours,
			}
			if err := w.Write(record); err != nil {
				return err
			}
		}
		return nil
	})

	return result.Error
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
