package handlers

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetUsers returns all users
func GetUsers(c echo.Context) error {
	var users []models.User

	if err := db.DB.Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch users",
		})
	}

	return c.JSON(http.StatusOK, users)
}

// GetUser returns a single user by ID
func GetUser(c echo.Context) error {
	id := c.Param("id")
	var user models.User

	if err := db.DB.First(&user, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	return c.JSON(http.StatusOK, user)
}

// CreateUser creates a new user
func CreateUser(c echo.Context) error {
	user := new(models.User)

	if err := c.Bind(user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if err := db.DB.Create(user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create user",
		})
	}

	// Send welcome email asynchronously (non-blocking)
	cfg := config.Load()
	if user.Email != "" {
		userName := user.Name
		if userName == "" {
			userName = user.Email
		}
		email := services.BuildWelcomeEmail(user.Email, userName)
		services.SendEmailAsync(cfg, email)
	}

	return c.JSON(http.StatusCreated, user)
}

// UpdateUser updates an existing user
func UpdateUser(c echo.Context) error {
	id := c.Param("id")
	var user models.User

	if err := db.DB.First(&user, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if err := db.DB.Save(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update user",
		})
	}

	return c.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user
func DeleteUser(c echo.Context) error {
	id := c.Param("id")

	if err := db.DB.Delete(&models.User{}, id).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete user",
		})
	}

	return c.JSON(http.StatusNoContent, nil)
}
