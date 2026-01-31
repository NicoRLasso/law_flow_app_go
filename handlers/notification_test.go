package handlers

import (
	"law_flow_app_go/models"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNotificationsHandler(t *testing.T) {
	_, c, rec := setupEcho(http.MethodGet, "/api/notifications", nil)

	err := GetNotificationsHandler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMarkNotificationReadHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-notify", Name: "Notify Firm"}
	database.Create(firm)
	user := &models.User{ID: "user-notify", Name: "Notify User", Email: "notify@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(user)

	notification := &models.Notification{
		ID: "notif-1", FirmID: firm.ID, UserID: stringToPtr(user.ID), Title: "Test Notification", Message: "This is a test notification",
	}
	database.Create(notification)

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodPost, "/api/notifications/notif-1/read", nil)
		c.SetParamNames("id")
		c.SetParamValues("notif-1")
		c.Set("user", user)
		c.Set("firm", firm)

		err := MarkNotificationReadHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify notification was marked as read
		var updatedNotif models.Notification
		database.First(&updatedNotif, "id = ?", "notif-1")
		assert.True(t, updatedNotif.IsRead())
	})

	t.Run("Notification not found", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodPost, "/api/notifications/nonexistent/read", nil)
		c.SetParamNames("id")
		c.SetParamValues("nonexistent")
		c.Set("user", user)
		c.Set("firm", firm)

		err := MarkNotificationReadHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestMarkAllNotificationsReadHandler(t *testing.T) {
	database := setupTestDB(t)
	firm := &models.Firm{ID: "firm-notify-all", Name: "Notify All Firm"}
	database.Create(firm)
	user := &models.User{ID: "user-notify-all", Name: "Notify All User", Email: "notifyall@test.com", FirmID: stringToPtr(firm.ID), Role: "admin"}
	database.Create(user)

	// Create multiple unread notifications
	database.Create(&models.Notification{
		ID: "notif-1", FirmID: firm.ID, UserID: stringToPtr(user.ID), Title: "Test 1", Message: "Message 1",
	})
	database.Create(&models.Notification{
		ID: "notif-2", FirmID: firm.ID, UserID: stringToPtr(user.ID), Title: "Test 2", Message: "Message 2",
	})
	database.Create(&models.Notification{
		ID: "notif-3", FirmID: firm.ID, UserID: stringToPtr(user.ID), Title: "Test 3", Message: "Message 3",
	})

	t.Run("Success", func(t *testing.T) {
		_, c, rec := setupEcho(http.MethodPost, "/api/notifications/read-all", nil)
		c.Set("user", user)
		c.Set("firm", firm)

		err := MarkAllNotificationsReadHandler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify all notifications were marked as read
		var count int64
		database.Model(&models.Notification{}).Where("user_id = ? AND read_at IS NULL", stringToPtr(user.ID)).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}
