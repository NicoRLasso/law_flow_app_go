package services

import (
	"errors"
	"law_flow_app_go/models"
	"time"

	"gorm.io/gorm"
)

// CaseMilestone-related errors
var (
	ErrCaseMilestoneNotFound = errors.New("case milestone not found")
)

// GetMilestonesByCase retrieves all milestones for a case ordered by sort_order
func GetMilestonesByCase(db *gorm.DB, caseID string) ([]models.CaseMilestone, error) {
	var milestones []models.CaseMilestone
	err := db.Where("case_id = ?", caseID).
		Preload("OutputDocument").
		Preload("Completer").
		Order("sort_order ASC").
		Find(&milestones).Error
	return milestones, err
}

// GetCaseMilestoneByID retrieves a milestone by ID
func GetCaseMilestoneByID(db *gorm.DB, milestoneID string) (*models.CaseMilestone, error) {
	var milestone models.CaseMilestone
	err := db.Preload("OutputDocument").
		Preload("Completer").
		First(&milestone, "id = ?", milestoneID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCaseMilestoneNotFound
		}
		return nil, err
	}
	return &milestone, nil
}

// GetCaseMilestoneProgress returns completion statistics for a case's milestones
func GetCaseMilestoneProgress(db *gorm.DB, caseID string) (*MilestoneProgress, error) {
	progress := &MilestoneProgress{}

	// Get all milestone counts by status
	type statusCount struct {
		Status string
		Count  int
	}
	var counts []statusCount

	err := db.Model(&models.CaseMilestone{}).
		Where("case_id = ?", caseID).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&counts).Error

	if err != nil {
		return nil, err
	}

	for _, c := range counts {
		progress.Total += c.Count
		switch c.Status {
		case models.MilestoneStatusCompleted:
			progress.Completed = c.Count
		case models.MilestoneStatusSkipped:
			progress.Skipped = c.Count
		case models.MilestoneStatusPending, models.MilestoneStatusInProgress:
			progress.Pending += c.Count
		}
	}

	// Calculate percentage (completed + skipped count as done)
	if progress.Total > 0 {
		done := progress.Completed + progress.Skipped
		progress.Percent = (done * 100) / progress.Total
	}

	return progress, nil
}

// CompleteCaseMilestone marks a milestone as completed
func CompleteCaseMilestone(db *gorm.DB, milestoneID, userID string) error {
	now := time.Now()
	return db.Model(&models.CaseMilestone{}).
		Where("id = ?", milestoneID).
		Updates(map[string]interface{}{
			"status":       models.MilestoneStatusCompleted,
			"completed_at": now,
			"completed_by": userID,
		}).Error
}

// ResetCaseMilestone resets a milestone to pending status
func ResetCaseMilestone(db *gorm.DB, milestoneID string) error {
	return db.Model(&models.CaseMilestone{}).
		Where("id = ?", milestoneID).
		Updates(map[string]interface{}{
			"status":       models.MilestoneStatusPending,
			"completed_at": nil,
			"completed_by": nil,
		}).Error
}

// ReorderCaseMilestones updates the sort order of milestones
func ReorderCaseMilestones(db *gorm.DB, caseID string, milestoneIDs []string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for i, id := range milestoneIDs {
			if err := tx.Model(&models.CaseMilestone{}).
				Where("id = ? AND case_id = ?", id, caseID).
				Update("sort_order", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// LinkDocumentToCaseMilestone links a document as the output of a case milestone
func LinkDocumentToCaseMilestone(db *gorm.DB, milestoneID, documentID string) error {
	return db.Model(&models.CaseMilestone{}).
		Where("id = ?", milestoneID).
		Update("output_document_id", documentID).Error
}

// CreateDefaultCaseMilestones creates default milestones for a new case
func CreateDefaultCaseMilestones(db *gorm.DB, caseRecord *models.Case) error {
	// Mock milestones for cases (exactly same as Service logic but adapted for Case)
	milestones := []models.CaseMilestone{
		{FirmID: caseRecord.FirmID, CaseID: caseRecord.ID, Title: "Recepción de requerimiento", SortOrder: 1, Status: models.MilestoneStatusPending},
		{FirmID: caseRecord.FirmID, CaseID: caseRecord.ID, Title: "Análisis inicial", SortOrder: 2, Status: models.MilestoneStatusPending},
		{FirmID: caseRecord.FirmID, CaseID: caseRecord.ID, Title: "Desarrollo", SortOrder: 3, Status: models.MilestoneStatusPending},
		{FirmID: caseRecord.FirmID, CaseID: caseRecord.ID, Title: "Revisión", SortOrder: 4, Status: models.MilestoneStatusPending},
		{FirmID: caseRecord.FirmID, CaseID: caseRecord.ID, Title: "Entrega", SortOrder: 5, Status: models.MilestoneStatusPending},
	}

	// Create all milestones
	for i := range milestones {
		if err := db.Create(&milestones[i]).Error; err != nil {
			return err
		}
	}

	return nil
}
