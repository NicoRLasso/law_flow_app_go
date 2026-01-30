package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"

	"github.com/labstack/echo/v4"
)

// ToolsPageHandler renders the tools page
func ToolsPageHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	firm := middleware.GetCurrentFirm(c)
	csrfToken := middleware.GetCSRFToken(c)

	// Fetch Colombia's departments for the radicado builder
	departments, err := services.GetDepartmentsByCountryCode(db.DB, "COL")
	if err != nil {
		departments = []models.Department{} // Empty list on error
	}

	// Fetch Clients and Lawyers for Report Generator
	var clients []models.User
	var lawyers []models.User

	// Fetch clients (assuming role 'client' or just filtered by current firm)
	// For simplicity, fetching all users in firm for now, or filter by role if possible
	// Using a refined query:
	if err := db.DB.Where("firm_id = ? AND role = 'client'", firm.ID).Find(&clients).Error; err != nil {
		clients = []models.User{}
	}

	// Fetch lawyers (assuming role 'lawyer' or 'admin' or 'superadmin')
	if err := db.DB.Where("firm_id = ? AND role IN ('lawyer', 'admin', 'superadmin')", firm.ID).Find(&lawyers).Error; err != nil {
		lawyers = []models.User{}
	}

	component := pages.Tools(c.Request().Context(), "Tools | LexLegal Cloud", csrfToken, currentUser, firm, departments, clients, lawyers)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
