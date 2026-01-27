package services

import (
	"law_flow_app_go/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupFTSTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	// Normal migrations
	db.AutoMigrate(&models.Case{}, &models.User{}, &models.CaseLog{}, &models.CaseParty{}, &models.CaseDocument{}, &models.Firm{})
	return db
}

func TestInitializeFTS5(t *testing.T) {
	db := setupFTSTestDB()

	// 1. Initialize
	err := InitializeFTS5(db)
	assert.NoError(t, err)

	// 2. Verify tables exist
	var tableName string
	db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name='cases_fts'").Scan(&tableName)
	assert.Equal(t, "cases_fts", tableName)

	db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name='cases_fts_mapping'").Scan(&tableName)
	assert.Equal(t, "cases_fts_mapping", tableName)

	// 3. Verify triggers exist
	var count int
	db.Raw("SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name LIKE 'cases_fts%'").Scan(&count)
	assert.True(t, count >= 3)
}

func TestFTSTriggersAndSync(t *testing.T) {
	db := setupFTSTestDB()
	InitializeFTS5(db)

	firmID := "firm-fts"
	caseID := "case-fts"
	clientID := "client-001"

	db.Create(&models.Firm{ID: firmID, Name: "FTS Firm"})
	db.Create(&models.User{ID: clientID, Name: "Test Client", Email: "client@test.com"})

	// 1. Test Insert Trigger
	c := &models.Case{
		ID:          caseID,
		FirmID:      firmID,
		ClientID:    clientID,
		CaseNumber:  "FTS-001",
		CaseType:    "Legal",
		Title:       stringToPtr("Searchable Title"),
		Description: "Unique Description Word",
	}
	err := db.Create(c).Error
	assert.NoError(t, err)

	// Verify it reached FTS
	var ftsEntry struct {
		CaseTitle string `gorm:"column:case_title"`
	}
	db.Raw("SELECT case_title FROM cases_fts WHERE case_id = ?", caseID).Scan(&ftsEntry)
	assert.Equal(t, "Searchable Title", ftsEntry.CaseTitle)

	// 2. Test Case Log Trigger
	logEntry := &models.CaseLog{
		CaseID:  caseID,
		FirmID:  firmID,
		Title:   "Log Title",
		Content: "Log content for searching",
	}
	db.Create(logEntry)

	var logContent string
	db.Raw("SELECT log_content FROM cases_fts WHERE case_id = ?", caseID).Scan(&logContent)
	assert.Contains(t, logContent, "Log content for searching")

	// 3. Test Update Trigger
	db.Model(c).Update("title", stringToPtr("Updated Title"))
	db.Raw("SELECT case_title FROM cases_fts WHERE case_id = ?", caseID).Scan(&ftsEntry)
	assert.Equal(t, "Updated Title", ftsEntry.CaseTitle)

	// 4. Test Delete Trigger
	db.Delete(c)
	var count int64
	db.Table("cases_fts").Where("case_id = ?", caseID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestRebuildFTSIndex(t *testing.T) {
	db := setupFTSTestDB()
	InitializeFTS5(db)

	firmID := "firm-rebuild"
	clientID := "client-002"
	db.Create(&models.Firm{ID: firmID, Name: "Rebuild Firm"})
	db.Create(&models.User{ID: clientID, Name: "Test Client", Email: "client2@test.com"})

	// Insert items directly into cases table
	db.Create(&models.Case{ID: "c1", FirmID: firmID, ClientID: clientID, CaseNumber: "NUM1", CaseType: "X", Title: stringToPtr("Case 1"), Description: "D1"})
	db.Create(&models.Case{ID: "c2", FirmID: firmID, ClientID: clientID, CaseNumber: "NUM2", CaseType: "X", Title: stringToPtr("Case 2"), Description: "D2"})

	// Manually clear FTS
	db.Exec("DELETE FROM cases_fts")
	db.Exec("DELETE FROM cases_fts_mapping")

	// Rebuild
	err := RebuildFTSIndex(db)
	assert.NoError(t, err)

	var count int64
	db.Table("cases_fts").Count(&count)
	assert.Equal(t, int64(2), count)
}
