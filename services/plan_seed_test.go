package services

import (
	"law_flow_app_go/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSeedTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(
		&models.Plan{},
		&models.PlanAddOn{},
		&models.Firm{},
		&models.FirmSubscription{},
		&models.FirmUsage{},
		&models.User{},
		&models.Case{},
		&models.CaseDocument{},
	)
	return db
}

func TestSeedDefaultPlans(t *testing.T) {
	db := setupSeedTestDB()

	// 1. Initial seed
	err := SeedDefaultPlans(db)
	assert.NoError(t, err)

	var count int64
	db.Model(&models.Plan{}).Count(&count)
	assert.Equal(t, int64(4), count) // Trial, Starter, Pro, Enterprise

	// 2. Test Idempotency
	err = SeedDefaultPlans(db)
	assert.NoError(t, err)
	db.Model(&models.Plan{}).Count(&count)
	assert.Equal(t, int64(4), count)

	// Verify specific plan
	var trialPlan models.Plan
	db.Where("tier = ?", models.PlanTierTrial).First(&trialPlan)
	assert.Equal(t, "Trial", trialPlan.Name)
	assert.True(t, trialPlan.IsDefault)
}

func TestSeedDefaultAddOns(t *testing.T) {
	db := setupSeedTestDB()

	// 1. Initial seed
	err := SeedDefaultAddOns(db)
	assert.NoError(t, err)

	var count int64
	db.Model(&models.PlanAddOn{}).Count(&count)
	assert.Equal(t, int64(4), count) // Users, Storage, Cases, Templates

	// 2. Test Idempotency
	err = SeedDefaultAddOns(db)
	assert.NoError(t, err)
	db.Model(&models.PlanAddOn{}).Count(&count)
	assert.Equal(t, int64(4), count)
}

func TestMigrateExistingFirmsToTrial(t *testing.T) {
	db := setupSeedTestDB()
	SeedDefaultPlans(db)

	firm1 := &models.Firm{ID: "f1", Name: "Firm 1"}
	firm2 := &models.Firm{ID: "f2", Name: "Firm 2"}
	db.Create(firm1)
	db.Create(firm2)

	// Pre-existing subscription for firm 2
	db.Create(&models.FirmSubscription{FirmID: firm2.ID, PlanID: "some-plan", Status: "active"})

	err := MigrateExistingFirmsToTrial(db)
	assert.NoError(t, err)

	// Firm 1 should have a trial now
	var sub1 models.FirmSubscription
	err = db.Where("firm_id = ?", firm1.ID).First(&sub1).Error
	assert.NoError(t, err)
	assert.Equal(t, models.SubscriptionStatusTrialing, sub1.Status)

	// Firm 1 should have usage initialized
	var usage1 models.FirmUsage
	err = db.Where("firm_id = ?", firm1.ID).First(&usage1).Error
	assert.NoError(t, err)
	assert.Equal(t, firm1.ID, usage1.FirmID)

	// Firm 2 should still have its original subscription (or at least count is 1)
	var count int64
	db.Model(&models.FirmSubscription{}).Where("firm_id = ?", firm2.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestInitializeSubscriptionSystem(t *testing.T) {
	db := setupSeedTestDB()

	err := InitializeSubscriptionSystem(db)
	assert.NoError(t, err)

	var planCount int64
	db.Model(&models.Plan{}).Count(&planCount)
	assert.Equal(t, int64(4), planCount)

	var addOnCount int64
	db.Model(&models.PlanAddOn{}).Count(&addOnCount)
	assert.Equal(t, int64(4), addOnCount)
}
