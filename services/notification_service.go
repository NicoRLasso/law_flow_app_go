package services

import (
	"law_flow_app_go/models"
	"time"

	"gorm.io/gorm"
)

type NotificationService struct {
	DB *gorm.DB
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{DB: db}
}

func (s *NotificationService) GetUnreadNotifications(firmID, userID string) ([]models.Notification, error) {
	var notifications []models.Notification
	err := s.DB.Where("firm_id = ? AND (user_id IS NULL OR user_id = ?) AND read_at IS NULL", firmID, userID).
		Order("created_at DESC").
		Limit(5).
		Find(&notifications).Error
	return notifications, err
}

func (s *NotificationService) MarkAsRead(notificationID, userID string, firmID string) error {
	now := time.Now()
	// Ensure the notification belongs to the firm and (optionally) the user
	return s.DB.Model(&models.Notification{}).
		Where("id = ? AND firm_id = ? AND (user_id IS NULL OR user_id = ?)", notificationID, firmID, userID).
		Update("read_at", now).Error
}

func (s *NotificationService) MarkAllAsRead(firmID, userID string) error {
	now := time.Now()
	return s.DB.Model(&models.Notification{}).
		Where("firm_id = ? AND (user_id IS NULL OR user_id = ?) AND read_at IS NULL", firmID, userID).
		Update("read_at", now).Error
}

func (s *NotificationService) GetNotificationCount(firmID, userID string) (int64, error) {
	var count int64
	err := s.DB.Model(&models.Notification{}).
		Where("firm_id = ? AND (user_id IS NULL OR user_id = ?) AND read_at IS NULL", firmID, userID).
		Count(&count).Error
	return count, err
}

func (s *NotificationService) CreateNotification(notification *models.Notification) error {
	return s.DB.Create(notification).Error
}
