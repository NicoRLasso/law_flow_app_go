package services

import (
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Reusing setupTestDB pattern (locally scoped to avoid conflicts if parallel)
func setupAuthTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.Session{}, &models.User{}, &models.Firm{})
	return db
}

func TestPasswordHashing(t *testing.T) {
	password := "SecretPass123!"

	// Test HashPassword
	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Test VerifyPassword
	assert.True(t, VerifyPassword(hash, password))
	assert.False(t, VerifyPassword(hash, "WrongPass"))
}

func TestSessionLifecycle(t *testing.T) {
	db := setupAuthTestDB()
	userID := "user-123"
	firmID := "firm-456"
	ip := "127.0.0.1"
	ua := "TestAgent"

	// 1. Create Session
	session, err := CreateSession(db, userID, firmID, ip, ua)
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.NotEmpty(t, session.Token)
	assert.Equal(t, userID, session.UserID)
	assert.Equal(t, firmID, *session.FirmID)
	assert.WithinDuration(t, time.Now().Add(DefaultSessionDuration), session.ExpiresAt, 10*time.Second)

	// 2. Validate Session (Valid)
	validSession, err := ValidateSession(db, session.Token)
	assert.NoError(t, err)
	assert.NotNil(t, validSession)
	assert.Equal(t, session.ID, validSession.ID)

	// 3. Validate Session (Invalid Token)
	invalidSession, err := ValidateSession(db, "invalid-token")
	assert.Error(t, err)
	assert.Nil(t, invalidSession)
	assert.Contains(t, err.Error(), "session not found")

	// 4. Delete Session
	err = DeleteSession(db, session.Token)
	assert.NoError(t, err)

	// 5. Validate Deleted Session
	deletedSession, err := ValidateSession(db, session.Token)
	assert.Error(t, err) // Should be not found
	assert.Nil(t, deletedSession)
}

func TestSessionExpiry(t *testing.T) {
	db := setupAuthTestDB()
	userID := "user-exp"

	// Create a manually expired session
	token := "expired-token"
	expiredSession := models.Session{
		ID:        "sess-expired",
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}
	db.Create(&expiredSession)

	// Validate Expired Session
	// Should return error and delete the session
	sess, err := ValidateSession(db, token)
	assert.Error(t, err)
	assert.Equal(t, "session expired", err.Error())
	assert.Nil(t, sess)

	// Verify deletion
	var count int64
	db.Model(&models.Session{}).Where("token = ?", token).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestCleanupExpiredSessions(t *testing.T) {
	db := setupAuthTestDB()

	// Seed mixed sessions
	db.Create(&models.Session{
		ID:        "sess-valid",
		Token:     "valid",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})
	db.Create(&models.Session{
		ID:        "sess-expired-1",
		Token:     "exp1",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})
	db.Create(&models.Session{
		ID:        "sess-expired-2",
		Token:     "exp2",
		ExpiresAt: time.Now().Add(-2 * time.Hour),
	})

	err := CleanupExpiredSessions(db)
	assert.NoError(t, err)

	// Verify count
	var count int64
	db.Model(&models.Session{}).Count(&count)
	assert.Equal(t, int64(1), count)

	var remaining models.Session
	db.First(&remaining)
	assert.Equal(t, "sess-valid", remaining.ID)
}

func TestDeleteAllUserSessions(t *testing.T) {
	db := setupAuthTestDB()
	targetUser := "target-user"
	otherUser := "other-user"

	db.Create(&models.Session{ID: "s1", UserID: targetUser, Token: "t1"})
	db.Create(&models.Session{ID: "s2", UserID: targetUser, Token: "t2"})
	db.Create(&models.Session{ID: "s3", UserID: otherUser, Token: "t3"})

	err := DeleteAllUserSessions(db, targetUser)
	assert.NoError(t, err)

	var count int64
	db.Model(&models.Session{}).Where("user_id = ?", targetUser).Count(&count)
	assert.Equal(t, int64(0), count)

	db.Model(&models.Session{}).Where("user_id = ?", otherUser).Count(&count)
	assert.Equal(t, int64(1), count)
}
