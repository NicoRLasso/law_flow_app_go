package handlers

import (
	"encoding/json"
	"law_flow_app_go/models"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetUsers(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-u1", Name: "User Firm"}
	database.Create(firm)

	admin := &models.User{ID: "admin-u1", Name: "Admin", Email: "admin1@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/users", nil)
		c.Set("user", admin)
		c.Set("firm", firm)

		err := GetUsers(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var users []models.User
		err = json.Unmarshal(rec.Body.Bytes(), &users)
		assert.NoError(t, err)
		assert.Len(t, users, 1)
	})
}

func TestGetUser(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-u2", Name: "User Firm 2"}
	database.Create(firm)

	admin := &models.User{ID: "admin-u2", Name: "Admin 2", Email: "admin2@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	otherUser := &models.User{ID: "user-u2", Name: "Other", Email: "other2@test.com", FirmID: stringToPtr(firm.ID), Role: "staff"}
	database.Create(otherUser)

	t.Run("Success Admin", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/api/users/user-u2", nil)
		c.SetParamNames("id")
		c.SetParamValues("user-u2")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := GetUser(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Forbidden", func(t *testing.T) {
		_, c, _ := setupEcho(http.MethodGet, "/api/users/admin-u2", nil)
		c.SetParamNames("id")
		c.SetParamValues("admin-u2")
		c.Set("user", otherUser)
		c.Set("firm", firm)

		err := GetUser(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusForbidden, err.(*echo.HTTPError).Code)
	})
}

func TestCreateUser(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-u3", Name: "User Firm 3"}
	database.Create(firm)

	plan := &models.Plan{ID: "plan-pro", Name: "Pro", MaxUsers: 5}
	database.Create(plan)
	database.Create(&models.FirmSubscription{FirmID: firm.ID, PlanID: plan.ID, Status: "active"})
	database.Create(&models.FirmUsage{FirmID: firm.ID, CurrentUsers: 1})

	admin := &models.User{ID: "admin-u3", Name: "Admin 3", Email: "admin3@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	t.Run("Valid creation", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "New User")
		f.Add("email", "new@example.com")
		f.Add("password", "SecurePassword123!")
		f.Add("role", "lawyer")
		f.Add("is_active", "true")

		_, c, rec := setupEcho(http.MethodPost, "/api/users", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := CreateUser(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})
}

func TestUpdateUser(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-u4", Name: "User Firm 4"}
	database.Create(firm)

	admin := &models.User{ID: "admin-u4", Name: "Admin 4", Email: "admin4@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	user := &models.User{ID: "user-u4", Name: "Original Name", Email: "user4@test.com", FirmID: stringToPtr(firm.ID), Role: "staff"}
	database.Create(user)

	t.Run("Success", func(t *testing.T) {
		f := url.Values{}
		f.Add("name", "Updated Name")
		f.Add("is_active", "true")

		_, c, rec := setupEcho(http.MethodPut, "/api/users/user-u4", strings.NewReader(f.Encode()))
		c.Request().Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.SetParamNames("id")
		c.SetParamValues("user-u4")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := UpdateUser(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		database.First(user, "id = ?", "user-u4")
		assert.Equal(t, "Updated Name", user.Name)
	})
}

func TestDeleteUser(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-u5", Name: "User Firm 5"}
	database.Create(firm)

	admin := &models.User{ID: "admin-u5", Name: "Admin 5", Email: "admin5@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	user := &models.User{ID: "user-u5", Name: "To Delete", Email: "user5@test.com", FirmID: stringToPtr(firm.ID), Role: "staff"}
	database.Create(user)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodDelete, "/api/users/user-u5", nil)
		c.SetParamNames("id")
		c.SetParamValues("user-u5")
		c.Set("user", admin)
		c.Set("firm", firm)

		err := DeleteUser(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

func TestUserUIHandlers(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-ui", Name: "UI Firm"}
	database.Create(firm)

	admin := &models.User{ID: "admin-ui", Name: "Admin UI", Email: "adminui@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(admin)

	t.Run("UsersPageHandler", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/users", nil)
		c.Set("user", admin)
		c.Set("firm", firm)
		err := UsersPageHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("GetUserFormNew", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/users/new", nil)
		err := GetUserFormNew(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("GetUserFormEdit", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/users/admin-ui/edit", nil)
		c.SetParamNames("id")
		c.SetParamValues("admin-ui")
		c.Set("user", admin)
		c.Set("firm", firm)
		err := GetUserFormEdit(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("GetUserDeleteConfirm", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/users/admin-ui/delete", nil)
		c.SetParamNames("id")
		c.SetParamValues("admin-ui")
		c.Set("user", admin)
		c.Set("firm", firm)
		err := GetUserDeleteConfirm(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("GetUsersListHTMX", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodGet, "/users/list", nil)
		c.Set("user", admin)
		c.Set("firm", firm)
		err := GetUsersListHTMX(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "tbody")
	})
}
