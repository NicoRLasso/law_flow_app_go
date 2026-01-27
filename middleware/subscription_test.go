package middleware

import (
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSubscriptionTestDB(t *testing.T) *gorm.DB {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	err = testDB.AutoMigrate(
		&models.Firm{},
		&models.User{},
		&models.Session{},
		&models.FirmSubscription{},
		&models.Plan{},
		&models.FirmUsage{},
		&models.FirmAddOn{},
		&models.PlanAddOn{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	db.DB = testDB
	return testDB
}

func TestRequireActiveSubscription(t *testing.T) {
	testDB := setupSubscriptionTestDB(t)
	e := echo.New()

	plan := models.Plan{ID: uuid.New().String(), Name: "Basic", Tier: models.PlanTierStarter}
	testDB.Create(&plan)

	t.Run("ActiveSubscription", func(t *testing.T) {
		firm := models.Firm{ID: uuid.New().String(), Name: "Active Firm"}
		testDB.Create(&firm)

		sub := models.FirmSubscription{
			ID:     uuid.New().String(),
			FirmID: firm.ID,
			PlanID: plan.ID,
			Status: models.SubscriptionStatusActive,
		}
		testDB.Create(&sub)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(ContextKeyFirm, &firm)

		handler := RequireActiveSubscription()(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, sub.ID, GetSubscription(c).ID)
	})

	t.Run("ExpiredSubscription", func(t *testing.T) {
		firm := models.Firm{ID: uuid.New().String(), Name: "Expired Firm"}
		testDB.Create(&firm)

		sub := models.FirmSubscription{
			ID:     uuid.New().String(),
			FirmID: firm.ID,
			PlanID: plan.ID,
			Status: models.SubscriptionStatusExpired,
		}
		testDB.Create(&sub)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(ContextKeyFirm, &firm)

		handler := RequireActiveSubscription()(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "/firm/settings")
	})

	t.Run("TrialExpired", func(t *testing.T) {
		firm := models.Firm{ID: uuid.New().String(), Name: "Trial Firm"}
		testDB.Create(&firm)

		past := time.Now().Add(-24 * time.Hour)
		sub := models.FirmSubscription{
			ID:          uuid.New().String(),
			FirmID:      firm.ID,
			PlanID:      plan.ID,
			Status:      models.SubscriptionStatusTrialing,
			TrialEndsAt: &past,
		}
		testDB.Create(&sub)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(ContextKeyFirm, &firm)

		handler := RequireActiveSubscription()(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)

		// Verify DB was updated
		testDB.First(&sub, "id = ?", sub.ID)
		assert.Equal(t, models.SubscriptionStatusExpired, sub.Status)
	})
}

func TestRequireTemplatesAccess(t *testing.T) {
	testDB := setupSubscriptionTestDB(t)
	e := echo.New()

	t.Run("HasAccess", func(t *testing.T) {
		plan := models.Plan{ID: uuid.New().String(), Name: "Pro", Tier: models.PlanTierProfessional, TemplatesEnabled: true}
		testDB.Create(&plan)
		firm := models.Firm{ID: uuid.New().String(), Name: "Pro Firm"}
		testDB.Create(&firm)
		sub := models.FirmSubscription{
			ID:     uuid.New().String(),
			FirmID: firm.ID,
			PlanID: plan.ID,
			Status: models.SubscriptionStatusActive,
		}
		testDB.Create(&sub)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(ContextKeyFirm, &firm)

		handler := RequireTemplatesAccess()(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("NoAccess", func(t *testing.T) {
		plan := models.Plan{ID: uuid.New().String(), Name: "Free", Tier: models.PlanTierStarter, TemplatesEnabled: false}
		testDB.Create(&plan)
		// Force false because of GORM default:true
		testDB.Model(&plan).Update("templates_enabled", false)

		firm := models.Firm{ID: uuid.New().String(), Name: "Free Firm"}
		testDB.Create(&firm)
		sub := models.FirmSubscription{
			ID:     uuid.New().String(),
			FirmID: firm.ID,
			PlanID: plan.ID,
			Status: models.SubscriptionStatusActive,
		}
		testDB.Create(&sub)
		testDB.First(&sub, "id = ?", sub.ID) // Reload to get plan associations if needed? Actually CanAccessTemplates preloads it.

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(ContextKeyFirm, &firm)

		handler := RequireTemplatesAccess()(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "upgrade=templates")
	})
}
