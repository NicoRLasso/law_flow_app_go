package services

import (
	"law_flow_app_go/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupNotificationTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.Notification{})
	return db
}

func TestNotificationService(t *testing.T) {
	db := setupNotificationTestDB()
	svc := NewNotificationService(db)

	firmID := "firm-1"
	userID := "user-1"

	t.Run("Create and Get Unread", func(t *testing.T) {
		err := svc.CreateNotification(&models.Notification{
			FirmID:  firmID,
			UserID:  &userID,
			Title:   "Test",
			Message: "Message",
		})
		assert.NoError(t, err)

		notifications, err := svc.GetUnreadNotifications(firmID, userID)
		assert.NoError(t, err)
		assert.Len(t, notifications, 1)
		assert.Equal(t, "Test", notifications[0].Title)

		count, _ := svc.GetNotificationCount(firmID, userID)
		assert.Equal(t, int64(1), count)
	})

	t.Run("Mark as Read", func(t *testing.T) {
		var n models.Notification
		db.First(&n)

		err := svc.MarkAsRead(n.ID, userID, firmID)
		assert.NoError(t, err)

		count, _ := svc.GetNotificationCount(firmID, userID)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Mark All as Read", func(t *testing.T) {
		svc.CreateNotification(&models.Notification{FirmID: firmID, UserID: &userID})
		svc.CreateNotification(&models.Notification{FirmID: firmID, UserID: &userID})

		count, _ := svc.GetNotificationCount(firmID, userID)
		assert.Equal(t, int64(2), count)

		err := svc.MarkAllAsRead(firmID, userID)
		assert.NoError(t, err)

		count, _ = svc.GetNotificationCount(firmID, userID)
		assert.Equal(t, int64(0), count)
	})
}
