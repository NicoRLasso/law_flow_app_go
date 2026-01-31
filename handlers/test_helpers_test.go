package handlers

import (
	"io"
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Use unique shared memory name to isolate tests while allowing shared cache for async tasks
	dbName := "mem_" + uuid.New().String()
	testDB, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared&_busy_timeout=5000"), &gorm.Config{})
	assert.NoError(t, err)

	err = testDB.Exec("PRAGMA journal_mode=WAL;").Error
	assert.NoError(t, err)

	// Initialize Storage for tests if not already set
	if services.Storage == nil {
		services.Storage = services.NewLocalStorage("tmp/test_uploads")
	}

	err = testDB.AutoMigrate(
		&models.Firm{},
		&models.Country{},
		&models.User{},
		&models.Session{},
		&models.AuditLog{},
		&models.Case{},
		&models.Appointment{},
		&models.Notification{},
		&models.CaseSubtype{},
		&models.CaseParty{},
		&models.CaseDomain{},
		&models.CaseBranch{},
		&models.CaseDocument{},
		&models.LegalService{},
		&models.ServiceMilestone{},
		&models.ChoiceCategory{},
		&models.ChoiceOption{},
		&models.ServiceDocument{},
		&models.ServiceExpense{},
		&models.Plan{},
		&models.FirmSubscription{},
		&models.FirmUsage{},
		&models.FirmAddOn{},
		&models.PlanAddOn{},
		&models.CaseMilestone{},
		&models.Availability{},
		&models.BlockedDate{},
	)
	assert.NoError(t, err)

	// Set global DB
	db.DB = testDB

	// Initialize security monitor
	services.InitSecurityMonitor()

	return testDB
}

func setupEcho(method, path string, body io.Reader) (*echo.Echo, echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, body)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Add config to context
	c.Set("config", &config.Config{
		Environment: "test",
	})

	return e, c, rec
}

func stringToPtr(s string) *string {
	return &s
}
