package services

import (
	"law_flow_app_go/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCaseClassificationTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.CaseDomain{}, &models.CaseBranch{}, &models.CaseSubtype{}, &models.Firm{})
	return db
}

func TestGetCaseClassifications(t *testing.T) {
	db := setupCaseClassificationTestDB()
	firmID := "firm-1"

	// Seed some data
	domain := models.CaseDomain{FirmID: firmID, Name: "Domain 1", Code: "D1", IsActive: true, Order: 1}
	db.Create(&domain)

	branch := models.CaseBranch{FirmID: firmID, DomainID: domain.ID, Name: "Branch 1", Code: "B1", IsActive: true, Order: 1}
	db.Create(&branch)

	subtype := models.CaseSubtype{FirmID: firmID, BranchID: branch.ID, Name: "Subtype 1", Code: "S1", IsActive: true, Order: 1}
	db.Create(&subtype)

	// Test GetCaseDomains
	domains, err := GetCaseDomains(db, firmID)
	assert.NoError(t, err)
	assert.Len(t, domains, 1)
	assert.Equal(t, domain.ID, domains[0].ID)

	// Test GetCaseBranches
	branches, err := GetCaseBranches(db, firmID, domain.ID)
	assert.NoError(t, err)
	assert.Len(t, branches, 1)
	assert.Equal(t, branch.ID, branches[0].ID)

	// Test GetCaseSubtypes
	subtypes, err := GetCaseSubtypes(db, firmID, branch.ID)
	assert.NoError(t, err)
	assert.Len(t, subtypes, 1)
	assert.Equal(t, subtype.ID, subtypes[0].ID)
}

func TestValidateCaseClassification(t *testing.T) {
	db := setupCaseClassificationTestDB()
	firmID := "firm-1"

	// Seed Hierarchy
	domain := models.CaseDomain{FirmID: firmID, Name: "Domain", Code: "D", IsActive: true}
	db.Create(&domain)
	branch := models.CaseBranch{FirmID: firmID, DomainID: domain.ID, Name: "Branch", Code: "B", IsActive: true}
	db.Create(&branch)
	subtype := models.CaseSubtype{FirmID: firmID, BranchID: branch.ID, Name: "Subtype", Code: "S", IsActive: true}
	db.Create(&subtype)

	// Valid Cases
	assert.True(t, ValidateCaseClassification(db, firmID, nil, nil, nil), "Empty valid")
	assert.True(t, ValidateCaseClassification(db, firmID, &domain.ID, nil, nil), "Only domain valid")
	assert.True(t, ValidateCaseClassification(db, firmID, &domain.ID, &branch.ID, nil), "Domain+Branch valid")
	assert.True(t, ValidateCaseClassification(db, firmID, &domain.ID, &branch.ID, &subtype.ID), "Full hierarchy valid")

	// Invalid Cases
	wrongID := "wrong-id"
	assert.False(t, ValidateCaseClassification(db, firmID, &wrongID, nil, nil), "Wrong domain invalid")
	assert.False(t, ValidateCaseClassification(db, firmID, &domain.ID, &wrongID, nil), "Wrong branch invalid")
	assert.False(t, ValidateCaseClassification(db, firmID, &domain.ID, nil, &subtype.ID), "Subtype without branch invalid")
	assert.False(t, ValidateCaseClassification(db, firmID, nil, &branch.ID, &subtype.ID), "Subtype without domain invalid")
}

func TestSeedCaseClassifications_Colombia(t *testing.T) {
	db := setupCaseClassificationTestDB()
	firmID := "firm-col"

	err := SeedCaseClassifications(db, firmID, "Colombia")
	assert.NoError(t, err)

	// Verify Domains
	var domainCount int64
	db.Model(&models.CaseDomain{}).Where("firm_id = ?", firmID).Count(&domainCount)
	assert.Equal(t, int64(4), domainCount)

	// Verify Branches
	var branchCount int64
	db.Model(&models.CaseBranch{}).Where("firm_id = ?", firmID).Count(&branchCount)
	assert.Equal(t, int64(14), branchCount)

	// Verify Subtypes (based on total count in seed function)
	var subtypeCount int64
	db.Model(&models.CaseSubtype{}).Where("firm_id = ?", firmID).Count(&subtypeCount)
	assert.Greater(t, subtypeCount, int64(30))
}
