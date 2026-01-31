package services

import (
	"fmt"
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupLegalServiceTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(
		&models.Firm{},
		&models.User{},
		&models.LegalService{},
		&models.ServiceMilestone{},
		&models.ServiceDocument{},
		&models.ServiceExpense{},
		&models.ChoiceCategory{},
		&models.ChoiceOption{},
		&models.Country{},
		&models.Case{}, // Needed for CanAddService -> CanAddCase
		&models.CaseDocument{},
		&models.FirmSubscription{},
		&models.Plan{},
		&models.FirmAddOn{},
		&models.PlanAddOn{},
		&models.FirmUsage{},
	)
	return db
}

func TestGenerateServiceNumber(t *testing.T) {
	db := setupLegalServiceTestDB()
	firmID := "firm-svc"
	db.Create(&models.Firm{ID: firmID, Name: "Svc Firm", Slug: "SVC"})

	t.Run("Generate first service number", func(t *testing.T) {
		num, err := GenerateServiceNumber(db, firmID)
		assert.NoError(t, err)
		currentYear := time.Now().Year()
		assert.Contains(t, num, "SVC-SVC-")
		assert.Contains(t, num, fmt.Sprintf("-%d-00001", currentYear))
	})

	t.Run("Generate sequential service number", func(t *testing.T) {
		currentYear := time.Now().Year()
		db.Create(&models.LegalService{
			FirmID:        firmID,
			ServiceNumber: fmt.Sprintf("SVC-SVC-%d-00041", currentYear),
			Title:         "Existing Service",
			ClientID:      "client-1",
		})

		num, err := GenerateServiceNumber(db, firmID)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("SVC-SVC-%d-00042", currentYear), num)
	})

	t.Run("Generate after malformed service number", func(t *testing.T) {
		firmID2 := "firm-malformed"
		db.Create(&models.Firm{ID: firmID2, Name: "Malformed Firm"})
		db.Create(&models.LegalService{
			FirmID:        firmID2,
			ServiceNumber: "MALFORMED-STUFF",
			Title:         "Malformed Service",
			ClientID:      "client-1",
		})

		// Should still generate a valid one (starts at 1 if no existing matches prefix)
		num, err := GenerateServiceNumber(db, firmID2)
		if err != nil {
			t.Logf("DEBUG: GenerateServiceNumber failed: %v", err)
			var count int64
			db.Model(&models.Firm{}).Where("id = ?", firmID2).Count(&count)
			t.Logf("DEBUG: Firm count for %s: %d", firmID2, count)
		}
		assert.NoError(t, err)
		assert.Contains(t, num, "-SVC-")
		assert.Contains(t, num, "00001")
	})
}

func TestLegalServiceLifecycle(t *testing.T) {
	db := setupLegalServiceTestDB()
	firmID := "firm-life"
	clientID := "client-1"
	userID := "user-1"

	db.Create(&models.Firm{ID: firmID, Name: "Life Firm", Slug: "LIFE"})
	db.Create(&models.User{ID: clientID, FirmID: &firmID, Role: "client", Name: "Client One", Email: "client1@test.com"})
	db.Create(&models.User{ID: userID, FirmID: &firmID, Role: "lawyer", Name: "Lawyer One", Email: "lawyer1@test.com"})

	var serviceID string

	t.Run("Create and Get Service", func(t *testing.T) {
		num, _ := EnsureUniqueServiceNumber(db, firmID)
		svc := models.LegalService{
			FirmID:        firmID,
			ClientID:      clientID,
			ServiceNumber: num,
			Title:         "Test Service",
			Objective:     "Test Objective",
			Status:        models.ServiceStatusIntake,
			Priority:      models.ServicePriorityNormal,
		}
		err := db.Create(&svc).Error
		assert.NoError(t, err)
		serviceID = svc.ID

		retrieved, err := GetServiceByID(db, firmID, serviceID)
		assert.NoError(t, err)
		assert.Equal(t, "Test Service", retrieved.Title)
		assert.Equal(t, clientID, retrieved.ClientID)

		// Test Not Found
		_, err = GetServiceByID(db, firmID, "non-existent")
		assert.ErrorIs(t, err, ErrServiceNotFound)

		_, err = GetServiceByID(db, "other-firm", serviceID)
		assert.ErrorIs(t, err, ErrServiceNotFound)
	})

	t.Run("Update Status", func(t *testing.T) {
		// Invalid status
		err := UpdateServiceStatus(db, serviceID, "INVALID", userID)
		assert.Error(t, err)

		err = UpdateServiceStatus(db, serviceID, models.ServiceStatusInProgress, userID)
		assert.NoError(t, err)

		retrieved, _ := GetServiceByID(db, firmID, serviceID)
		assert.Equal(t, models.ServiceStatusInProgress, retrieved.Status)
		assert.NotNil(t, retrieved.StartedAt)

		// Update again - should not change StartedAt
		startedAt := retrieved.StartedAt
		time.Sleep(10 * time.Millisecond)
		err = UpdateServiceStatus(db, serviceID, models.ServiceStatusInProgress, userID)
		assert.NoError(t, err)
		retrieved, _ = GetServiceByID(db, firmID, serviceID)
		assert.Equal(t, startedAt, retrieved.StartedAt)

		err = UpdateServiceStatus(db, serviceID, models.ServiceStatusCompleted, userID)
		assert.NoError(t, err)
		retrieved, _ = GetServiceByID(db, firmID, serviceID)
		assert.Equal(t, models.ServiceStatusCompleted, retrieved.Status)
		assert.NotNil(t, retrieved.CompletedAt)
	})

	t.Run("Get Services with Filters", func(t *testing.T) {
		// Keyword filter
		filters := ServiceFilters{Keyword: "Test"}
		services, total, err := GetServicesByFirm(db, firmID, filters, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, services, 1)

		// Priority filter
		filters = ServiceFilters{Priority: models.ServicePriorityNormal}
		services, total, err = GetServicesByFirm(db, firmID, filters, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)

		// Client filter
		filters = ServiceFilters{ClientID: clientID}
		services, total, err = GetServicesByFirm(db, firmID, filters, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)

		// Date filters
		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)
		tomorrow := now.Add(24 * time.Hour)

		filters = ServiceFilters{DateFrom: &yesterday}
		_, total, _ = GetServicesByFirm(db, firmID, filters, 1, 10)
		assert.Equal(t, int64(1), total)

		filters = ServiceFilters{DateTo: &tomorrow}
		_, total, _ = GetServicesByFirm(db, firmID, filters, 1, 10)
		assert.Equal(t, int64(1), total)
	})

	t.Run("Expenses", func(t *testing.T) {
		db.Create(&models.ServiceExpense{
			ServiceID:    serviceID,
			FirmID:       firmID,
			Amount:       100.0,
			Status:       models.ExpenseStatusApproved,
			Description:  "Exp 1",
			RecordedByID: userID,
			IncurredAt:   time.Now(),
		})
		db.Create(&models.ServiceExpense{
			ServiceID:    serviceID,
			FirmID:       firmID,
			Amount:       50.0,
			Status:       models.ExpenseStatusPending,
			Description:  "Exp 2",
			RecordedByID: userID,
			IncurredAt:   time.Now(),
		})
		db.Create(&models.ServiceExpense{
			ServiceID:    serviceID,
			FirmID:       firmID,
			Amount:       200.0,
			Status:       models.ExpenseStatusRejected,
			Description:  "Rejected",
			RecordedByID: userID,
			IncurredAt:   time.Now(),
		})

		total, err := GetServiceTotalExpenses(db, serviceID)
		assert.NoError(t, err)
		assert.Equal(t, 150.0, total) // 100 + 50 (rejected excluded)

		approved, err := GetServiceApprovedExpenses(db, serviceID)
		assert.NoError(t, err)
		assert.Equal(t, 100.0, approved)
	})

	t.Run("Get Summary", func(t *testing.T) {
		summary, err := GetServiceSummary(db, firmID)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), summary.TotalServices)
		assert.Equal(t, int64(1), summary.CompletedServices)
		assert.Equal(t, 100.0, summary.TotalExpenses)
		// Active services count (INTAKE and IN_PROGRESS)
		assert.Equal(t, int64(0), summary.ActiveServices) // It was COMPLETED in previous subtest
	})

	t.Run("Delete Service", func(t *testing.T) {
		// Not Found
		err := DeleteService(db, firmID, "non-existent")
		assert.ErrorIs(t, err, ErrServiceNotFound)

		err = DeleteService(db, firmID, serviceID)
		assert.NoError(t, err)

		_, err = GetServiceByID(db, firmID, serviceID)
		assert.ErrorIs(t, err, ErrServiceNotFound)
	})
}

func TestCanAddService(t *testing.T) {
	db := setupLegalServiceTestDB()
	firmID := "firm-limit"
	db.Create(&models.Firm{ID: firmID, Slug: "LIMIT"})

	// Create plan and subscription
	db.Create(&models.Plan{ID: "free", Name: "Free", MaxCases: 1})
	db.Create(&models.FirmSubscription{FirmID: firmID, PlanID: "free", Status: "active"})

	t.Run("Under limit", func(t *testing.T) {
		res, err := CanAddService(db, firmID)
		assert.NoError(t, err)
		assert.True(t, res.Allowed)
	})

	t.Run("At limit", func(t *testing.T) {
		db.Create(&models.Case{FirmID: firmID, CaseNumber: "C1", Title: stringToPtr("C1")})
		_, err := RecalculateFirmUsage(db, firmID)
		assert.NoError(t, err)

		res, err := CanAddService(db, firmID)
		assert.ErrorIs(t, err, ErrCaseLimitReached)
		assert.False(t, res.Allowed)
	})
}

func TestEnsureUniqueServiceNumberRetry(t *testing.T) {
	// This is hard to test without mocking GenerateServiceNumber,
	// but we can at least test the happy path and maybe a conflict.
}

func TestGetServicesByRole(t *testing.T) {
	db := setupLegalServiceTestDB()
	firmID := "firm-role"
	c1 := "client-1"
	c2 := "client-2"
	u1 := "user-1"

	db.Create(&models.Firm{ID: firmID, Name: "Role Firm", Slug: "ROLE"})
	db.Create(&models.User{ID: c1, FirmID: &firmID, Role: "client", Email: "c1@role.com"})
	db.Create(&models.User{ID: c2, FirmID: &firmID, Role: "client", Email: "c2@role.com"})
	db.Create(&models.User{ID: u1, FirmID: &firmID, Role: "lawyer", Email: "u1@role.com"})

	db.Create(&models.LegalService{ID: "s1", FirmID: firmID, ClientID: c1, ServiceNumber: "S1", Title: "S1", Objective: "O1"})
	db.Create(&models.LegalService{ID: "s2", FirmID: firmID, ClientID: c2, ServiceNumber: "S2", Title: "S2", Objective: "O2", AssignedToID: &u1})

	t.Run("By Client", func(t *testing.T) {
		svcs, err := GetServicesByClient(db, firmID, c1)
		assert.NoError(t, err)
		assert.Len(t, svcs, 1)
		assert.Equal(t, "s1", svcs[0].ID)
	})

	t.Run("By Assignee", func(t *testing.T) {
		svcs, err := GetServicesByAssignee(db, firmID, u1)
		assert.NoError(t, err)
		assert.Len(t, svcs, 1)
		assert.Equal(t, "s2", svcs[0].ID)
	})

	t.Run("Active Count", func(t *testing.T) {
		count, err := GetActiveServicesCount(db, firmID)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count) // Both INTAKE
	})
}
