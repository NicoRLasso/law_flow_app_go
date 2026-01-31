package services

import (
	"law_flow_app_go/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCaseMilestoneTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.CaseMilestone{}, &models.User{}, &models.CaseDocument{}, &models.Case{})
	return db
}

func TestCaseMilestoneService(t *testing.T) {
	db := setupCaseMilestoneTestDB()
	caseID := "case-1"
	userID := "user-1"
	firmID := "firm-1"

	db.Create(&models.Case{ID: caseID, FirmID: firmID, CaseNumber: "C1"})

	t.Run("Create Default Milestones", func(t *testing.T) {
		err := CreateDefaultCaseMilestones(db, &models.Case{ID: caseID, FirmID: firmID})
		assert.NoError(t, err)

		milestones, err := GetMilestonesByCase(db, caseID)
		assert.NoError(t, err)
		assert.Len(t, milestones, 5)
	})

	t.Run("Get and Update Milestone", func(t *testing.T) {
		milestones, _ := GetMilestonesByCase(db, caseID)
		m1 := milestones[0]

		err := CompleteCaseMilestone(db, m1.ID, userID)
		assert.NoError(t, err)

		retrieved, _ := GetCaseMilestoneByID(db, m1.ID)
		assert.Equal(t, models.MilestoneStatusCompleted, retrieved.Status)

		progress, err := GetCaseMilestoneProgress(db, caseID)
		assert.NoError(t, err)
		assert.Equal(t, 20, progress.Percent) // 1 out of 5

		err = ResetCaseMilestone(db, m1.ID)
		assert.NoError(t, err)
		retrieved, _ = GetCaseMilestoneByID(db, m1.ID)
		assert.Equal(t, models.MilestoneStatusPending, retrieved.Status)
	})

	t.Run("Reorder", func(t *testing.T) {
		milestones, _ := GetMilestonesByCase(db, caseID)
		ids := []string{milestones[1].ID, milestones[0].ID} // Swap first two
		err := ReorderCaseMilestones(db, caseID, ids)
		assert.NoError(t, err)

		newOrder, _ := GetMilestonesByCase(db, caseID)
		assert.Equal(t, ids[0], newOrder[0].ID)
	})

	t.Run("Link Document", func(t *testing.T) {
		milestones, _ := GetMilestonesByCase(db, caseID)
		docID := "doc-1"
		err := LinkDocumentToCaseMilestone(db, milestones[0].ID, docID)
		assert.NoError(t, err)

		retrieved, _ := GetCaseMilestoneByID(db, milestones[0].ID)
		assert.Equal(t, docID, *retrieved.OutputDocumentID)
	})
}
