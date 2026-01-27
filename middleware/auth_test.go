package middleware

import (
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	err = testDB.AutoMigrate(&models.User{}, &models.Firm{}, &models.Session{})
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	// Set the global DB variable used by middleware
	db.DB = testDB
	return testDB
}

func TestRequireAuth(t *testing.T) {
	testDB := setupTestDB(t)
	e := echo.New()

	// Create a test firm and user
	firm := models.Firm{ID: uuid.New().String(), Name: "Test Firm"}
	testDB.Create(&firm)

	user := models.User{
		ID:       uuid.New().String(),
		Name:     "Test User",
		Email:    "test@example.com",
		FirmID:   &firm.ID,
		IsActive: true,
		Role:     "admin",
	}
	testDB.Create(&user)

	// Create a valid session
	session, _ := services.CreateSession(testDB, user.ID, firm.ID, "127.0.0.1", "test-agent")

	t.Run("ValidSession", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: session.Token})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := RequireAuth()(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, user.ID, GetCurrentUser(c).ID)
	})

	t.Run("NoCookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := RequireAuth()(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})

	t.Run("InvalidToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "invalid-token"})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := RequireAuth()(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
	})

	t.Run("InactiveUser", func(t *testing.T) {
		inactiveUser := models.User{
			ID:       uuid.New().String(),
			Name:     "Inactive User",
			Email:    "inactive@example.com",
			IsActive: false,
		}
		testDB.Create(&inactiveUser)
		// Force IsActive to false because GORM default:true might override zero values during creation
		testDB.Model(&inactiveUser).Update("is_active", false)

		session, _ := services.CreateSession(testDB, inactiveUser.ID, "", "", "")

		req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: session.Token})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := RequireAuth()(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
	})
}

func TestRequireRole(t *testing.T) {
	e := echo.New()

	t.Run("HasRole", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(ContextKeyUser, &models.User{Role: "admin"})

		handler := RequireRole("admin", "lawyer")(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("MissingRole", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(ContextKeyUser, &models.User{Role: "staff"})

		handler := RequireRole("admin")(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusForbidden, he.Code)
	})
}

func TestRequireSuperadmin(t *testing.T) {
	e := echo.New()

	t.Run("IsSuperadmin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(ContextKeyUser, &models.User{Role: "superadmin"})

		handler := RequireSuperadmin()(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)
	})

	t.Run("NotSuperadmin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(ContextKeyUser, &models.User{Role: "admin"})

		handler := RequireSuperadmin()(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusForbidden, err.(*echo.HTTPError).Code)
	})
}

func TestRequireFirm(t *testing.T) {
	e := echo.New()

	t.Run("HasFirm", func(t *testing.T) {
		firmID := "firm-123"
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(ContextKeyUser, &models.User{FirmID: &firmID, Role: "admin"})

		handler := RequireFirm()(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)
	})

	t.Run("NoFirm", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(ContextKeyUser, &models.User{ID: "user-123", Role: "admin"})

		handler := RequireFirm()(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/firm/setup", rec.Header().Get("Location"))
	})
}

func TestAuthHelpers(t *testing.T) {
	e := echo.New()

	t.Run("GetCurrentUser", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		user := &models.User{ID: "123"}
		c.Set(ContextKeyUser, user)
		assert.Equal(t, user, GetCurrentUser(c))

		c = e.NewContext(req, rec)
		assert.Nil(t, GetCurrentUser(c))
	})

	t.Run("GetCurrentFirm", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		firm := &models.Firm{ID: "456"}
		c.Set(ContextKeyFirm, firm)
		assert.Equal(t, firm, GetCurrentFirm(c))

		c = e.NewContext(req, rec)
		assert.Nil(t, GetCurrentFirm(c))
	})
}
