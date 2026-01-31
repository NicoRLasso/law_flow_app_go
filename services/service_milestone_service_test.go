package services

import (
	"law_flow_app_go/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupServiceMilestoneTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.ServiceMilestone{}, &models.User{}, &models.ServiceDocument{}, &models.LegalService{}, &models.ChoiceOption{})
	return db
}

func TestServiceMilestoneService(t *testing.T) {
	db := setupServiceMilestoneTestDB()
	serviceID := "svc-1"
	userID := "user-1"
	firmID := "firm-1"

	db.Create(&models.LegalService{ID: serviceID, FirmID: firmID, ServiceNumber: "S1", Title: "S1", Objective: "O1"})

	t.Run("Create Default Milestones (Generic)", func(t *testing.T) {
		err := CreateDefaultMilestones(db, &models.LegalService{ID: serviceID, FirmID: firmID})
		assert.NoError(t, err)

		milestones, err := GetMilestonesByService(db, serviceID)
		assert.NoError(t, err)
		assert.Len(t, milestones, 5)
	})

	t.Run("Create Default Milestones (All Types)", func(t *testing.T) {
		types := []string{
			"COMPANY_FORMATION",
			"VISA_PROCESSING",
			"TRADEMARK_REGISTRATION",
			"NOTARIAL_PROCESS",
			"CONTRACT_REVIEW",
			"DOCUMENT_CREATION",
			"LEGAL_CONCEPT",
			"TAX_OPINION",
			"REAL_ESTATE",
			"UNKNOWN",
		}

		for _, code := range types {
			t.Run(code, func(t *testing.T) {
				svcID := "svc-" + code
				db.Create(&models.LegalService{ID: svcID, FirmID: firmID, ServiceNumber: "S-" + code, Title: code, Objective: code})

				err := CreateDefaultMilestones(db, &models.LegalService{
					ID:          svcID,
					FirmID:      firmID,
					ServiceType: &models.ChoiceOption{Code: code},
				})
				assert.NoError(t, err)

				milestones, _ := GetMilestonesByService(db, svcID)
				assert.NotEmpty(t, milestones)
			})
		}
	})

	t.Run("Create Default Milestones (via ID)", func(t *testing.T) {
		opt := models.ChoiceOption{Code: "VISA_PROCESSING", Label: "Visa"}
		db.Create(&opt)

		svcID := "svc-id-only"
		db.Create(&models.LegalService{ID: svcID, FirmID: firmID, ServiceNumber: "S-ID", Title: "ID", Objective: "ID", ServiceTypeID: &opt.ID})

		err := CreateDefaultMilestones(db, &models.LegalService{
			ID:            svcID,
			FirmID:        firmID,
			ServiceTypeID: &opt.ID,
		})
		assert.NoError(t, err)

		milestones, _ := GetMilestonesByService(db, svcID)
		assert.Len(t, milestones, 6)
		assert.Equal(t, "Recolección de documentos", milestones[0].Title)
	})

	t.Run("Lifecycle and Progress", func(t *testing.T) {
		milestones, _ := GetMilestonesByService(db, serviceID)
		m1 := milestones[0]

		err := StartMilestone(db, m1.ID)
		assert.NoError(t, err)

		err = CompleteMilestone(db, m1.ID, userID)
		assert.NoError(t, err)

		retrieved, _ := GetMilestoneByID(db, m1.ID)
		assert.Equal(t, models.MilestoneStatusCompleted, retrieved.Status)

		// Test GetMilestoneByID Not Found
		_, err = GetMilestoneByID(db, "non-existent")
		assert.ErrorIs(t, err, ErrMilestoneNotFound)

		next, _ := GetNextPendingMilestone(db, serviceID)
		assert.Equal(t, milestones[1].ID, next.ID)

		allDone, _ := AreAllMilestonesComplete(db, serviceID)
		assert.False(t, allDone)

		err = SkipMilestone(db, milestones[1].ID, userID)
		assert.NoError(t, err)

		progress, _ := GetMilestoneProgress(db, serviceID)
		assert.Equal(t, 40, progress.Percent) // 2 out of 5

		// Reset Milestone
		err = ResetMilestone(db, m1.ID)
		assert.NoError(t, err)
		retrieved, _ = GetMilestoneByID(db, m1.ID)
		assert.Equal(t, models.MilestoneStatusPending, retrieved.Status)
		assert.Nil(t, retrieved.CompletedAt)
	})

	t.Run("Reorder and Link", func(t *testing.T) {
		milestones, _ := GetMilestonesByService(db, serviceID)
		ids := []string{milestones[1].ID, milestones[0].ID} // Reversed order

		err := ReorderMilestones(db, serviceID, ids)
		assert.NoError(t, err)

		reordered, _ := GetMilestonesByService(db, serviceID)
		assert.Equal(t, milestones[1].ID, reordered[0].ID)
		assert.Equal(t, 1, reordered[0].SortOrder)

		// Link Document
		docID := "doc-1"
		err = LinkDocumentToMilestone(db, milestones[0].ID, docID)
		assert.NoError(t, err)
		retrieved, _ := GetMilestoneByID(db, milestones[0].ID)
		assert.Equal(t, docID, *retrieved.OutputDocumentID)
	})

	t.Run("GetNextPendingMilestone Empty", func(t *testing.T) {
		svcID := "svc-empty"
		db.Create(&models.LegalService{ID: svcID, FirmID: firmID, ServiceNumber: "S-EMPTY", Title: "Empty", Objective: "O-EMPTY"})

		next, err := GetNextPendingMilestone(db, svcID)
		assert.NoError(t, err)
		assert.Nil(t, next)

		done, _ := AreAllMilestonesComplete(db, svcID)
		assert.True(t, done)
	})
}
