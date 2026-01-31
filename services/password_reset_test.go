package services

import (
	"fmt"
	"law_flow_app_go/models"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupResetTestDB() *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=busy_timeout(5000)", uuid.New().String())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.User{}, &models.PasswordResetToken{}, &models.Session{}, &models.AuditLog{})
	return db
}

func TestGenerateResetToken(t *testing.T) {
	email := "test@example.com"

	t.Run("Generate token for active user", func(t *testing.T) {
		db := setupResetTestDB()
		user := &models.User{ID: "u1", Email: email, IsActive: true}
		db.Create(user)

		token, err := GenerateResetToken(db, email)
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, user.ID, token.UserID)
		assert.NotEmpty(t, token.Token)
	})

	t.Run("Generate token for inactive user", func(t *testing.T) {
		db := setupResetTestDB()
		inactiveEmail := "inactive@example.com"
		user := &models.User{ID: "u2", Email: inactiveEmail}
		db.Create(user)
		db.Model(user).Update("is_active", false)
		token, err := GenerateResetToken(db, inactiveEmail)
		assert.NoError(t, err)
		assert.Nil(t, token)
	})

	t.Run("Generate token for non-existent email", func(t *testing.T) {
		db := setupResetTestDB()
		token, err := GenerateResetToken(db, "nope@example.com")
		assert.NoError(t, err)
		assert.Nil(t, token)
	})

	t.Run("Clears old tokens", func(t *testing.T) {
		db := setupResetTestDB()
		db.Create(&models.User{ID: "u3", Email: email, IsActive: true})
		// Use a transaction to avoid database locking issues
		tx := db.Begin()
		GenerateResetToken(tx, email)
		tx.Commit()
		tx = db.Begin()
		GenerateResetToken(tx, email)
		tx.Commit()
		var count int64
		db.Model(&models.PasswordResetToken{}).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

func TestValidateResetToken(t *testing.T) {
	user := &models.User{ID: "u1", Email: "test@example.com", IsActive: true}

	t.Run("Valid token", func(t *testing.T) {
		db := setupResetTestDB()
		db.Create(user)
		rt, _ := GenerateResetToken(db, user.Email)
		foundUser, err := ValidateResetToken(db, rt.Token)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, foundUser.ID)
	})

	t.Run("Expired token", func(t *testing.T) {
		db := setupResetTestDB()
		db.Create(user)
		rt := &models.PasswordResetToken{
			UserID:    user.ID,
			Token:     "expired-token",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		db.Create(rt)
		_, err := ValidateResetToken(db, "expired-token")
		assert.Error(t, err)
	})

	t.Run("Invalid token string", func(t *testing.T) {
		db := setupResetTestDB()
		_, err := ValidateResetToken(db, "non-existent")
		assert.Error(t, err)
	})
}

func TestResetPasswordFlow(t *testing.T) {
	db := setupResetTestDB()
	email := "test@example.com"
	user := &models.User{ID: "u1", Email: email, IsActive: true, Password: "OldPassword1"}
	db.Create(user)

	db.Create(&models.Session{ID: "s1", UserID: user.ID, Token: "t1", ExpiresAt: time.Now().Add(time.Hour)})
	db.Create(&models.Session{ID: "s2", UserID: user.ID, Token: "t2", ExpiresAt: time.Now().Add(time.Hour)})

	rt, _ := GenerateResetToken(db, email)

	newPass := "NewStrongPassword123!"
	err := ResetPassword(db, rt.Token, newPass)
	assert.NoError(t, err)

	var updatedUser models.User
	db.First(&updatedUser, "id = ?", user.ID)
	assert.True(t, VerifyPassword(updatedUser.Password, newPass))

	var tokenCount int64
	db.Model(&models.PasswordResetToken{}).Where("token = ?", rt.Token).Count(&tokenCount)
	assert.Equal(t, int64(0), tokenCount)

	var sessionCount int64
	db.Model(&models.Session{}).Where("user_id = ?", user.ID).Count(&sessionCount)
	assert.Equal(t, int64(0), sessionCount)
}

func TestCleanupExpiredTokens(t *testing.T) {
	db := setupResetTestDB()
	db.Create(&models.PasswordResetToken{UserID: "u1", Token: "e1", ExpiresAt: time.Now().Add(-1 * time.Hour)})
	db.Create(&models.PasswordResetToken{UserID: "u2", Token: "v1", ExpiresAt: time.Now().Add(1 * time.Hour)})

	err := CleanupExpiredTokens(db)
	assert.NoError(t, err)

	var count int64
	db.Model(&models.PasswordResetToken{}).Count(&count)
	assert.Equal(t, int64(1), count)
}
