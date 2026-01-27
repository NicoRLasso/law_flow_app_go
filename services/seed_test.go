package services

import (
	"law_flow_app_go/models"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSuperadminSeedTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.User{})
	return db
}

func TestSeedSuperadminFromEnv(t *testing.T) {
	t.Run("Creates superadmin when env vars are set", func(t *testing.T) {
		db := setupSuperadminSeedTestDB()
		os.Setenv("SUPERADMIN_EMAIL", "admin@test.com")
		os.Setenv("SUPERADMIN_PASSWORD", "StrongPass123!")
		os.Setenv("SUPERADMIN_NAME", "Test Admin")
		defer os.Unsetenv("SUPERADMIN_EMAIL")
		defer os.Unsetenv("SUPERADMIN_PASSWORD")
		defer os.Unsetenv("SUPERADMIN_NAME")

		err := SeedSuperadminFromEnv(db)
		assert.NoError(t, err)

		var user models.User
		err = db.Where("email = ?", "admin@test.com").First(&user).Error
		assert.NoError(t, err)
		assert.Equal(t, "superadmin", user.Role)
		assert.Equal(t, "Test Admin", user.Name)
	})

	t.Run("Skips if superadmin already exists", func(t *testing.T) {
		db := setupSeedTestDB()
		db.Create(&models.User{Email: "existing@test.com", Role: "superadmin"})

		os.Setenv("SUPERADMIN_EMAIL", "new@test.com")
		os.Setenv("SUPERADMIN_PASSWORD", "Pass123!")
		defer os.Unsetenv("SUPERADMIN_EMAIL")
		defer os.Unsetenv("SUPERADMIN_PASSWORD")

		err := SeedSuperadminFromEnv(db)
		assert.NoError(t, err)

		var count int64
		db.Model(&models.User{}).Where("email = ?", "new@test.com").Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Skips if email is taken by non-superadmin", func(t *testing.T) {
		db := setupSeedTestDB()
		db.Create(&models.User{Email: "taken@test.com", Role: "admin"})

		os.Setenv("SUPERADMIN_EMAIL", "taken@test.com")
		os.Setenv("SUPERADMIN_PASSWORD", "Pass123!")
		defer os.Unsetenv("SUPERADMIN_EMAIL")
		defer os.Unsetenv("SUPERADMIN_PASSWORD")

		err := SeedSuperadminFromEnv(db)
		assert.NoError(t, err)

		var user models.User
		db.Where("email = ?", "taken@test.com").First(&user)
		assert.Equal(t, "admin", user.Role)
	})
}
