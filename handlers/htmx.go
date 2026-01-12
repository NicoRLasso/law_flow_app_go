package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"law_flow_app_go/templates/partials"

	"github.com/labstack/echo/v4"
)

// GetUsersHTMX returns the user list as an HTMX partial
func GetUsersHTMX(c echo.Context) error {
	var users []models.User

	if err := db.DB.Find(&users).Error; err != nil {
		// Return error partial
		return c.HTML(500, `<div class="bg-red-50 border-l-4 border-red-400 p-4 rounded">
			<p class="text-red-700">Error loading users. Please try again.</p>
		</div>`)
	}

	// Return partial template
	component := partials.UserList(users)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
