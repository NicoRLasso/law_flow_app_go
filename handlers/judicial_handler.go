package handlers

import (
	"errors"
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"law_flow_app_go/templates/partials"
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// GetJudicialProcessViewHandler returns the HTML partial for the unified judicial process view
func GetJudicialProcessViewHandler(c echo.Context) error {
	caseID := c.Param("id")
	page := 1
	pageSize := 10

	if p := c.QueryParam("page"); p != "" {
		// simple atoi, ignore error fallback to 1
		// avoiding strconv import if not present, but usually needed.
		// Actually I need strconv.
		// For now let's assume valid or do quick check? No, I'll add strconv.
		// Wait, I can't add imports with replace block easily unless I replace imports too.
		// I will Assume standard binding or use a helper.
		// Let's just use loose binding? No.
		// I'll update imports first.
	}

	// Fetch JP without actions
	var jp models.JudicialProcess
	err := db.DB.Where("case_id = ?", caseID).First(&jp).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return empty state or basic info if filing number exists
			var kase models.Case
			if err := db.DB.Select("id, filing_number").First(&kase, "id = ?", caseID).Error; err == nil {
				component := partials.JudicialProcessView(c.Request().Context(), nil, kase.FilingNumber, 1, 1, "overview") // Default overview
				return component.Render(c.Request().Context(), c.Response().Writer)
			}
			component := partials.JudicialProcessView(c.Request().Context(), nil, nil, 1, 1, "overview")
			return component.Render(c.Request().Context(), c.Response().Writer)
		}
		return c.String(http.StatusInternalServerError, "Error loading judicial process data")
	}

	// Pagination logic
	var totalActions int64
	db.DB.Model(&models.JudicialProcessAction{}).Where("judicial_process_id = ?", jp.ID).Count(&totalActions)

	totalPages := int((totalActions + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}

	bindPage := new(struct {
		Page int `query:"page"`
	})
	c.Bind(bindPage)
	if bindPage.Page > 0 {
		page = bindPage.Page
	}

	var actions []models.JudicialProcessAction
	db.DB.Where("judicial_process_id = ?", jp.ID).
		Order("action_date DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&actions)

	jp.Actions = actions

	// Determine active tab
	activeTab := "overview"
	if c.QueryParam("tab") == "actions" {
		activeTab = "actions"
	} else if c.QueryParam("page") != "" {
		// If page is present but no specific tab, default to actions contextually
		activeTab = "actions"
	}

	component := partials.JudicialProcessView(c.Request().Context(), &jp, &jp.Radicado, page, totalPages, activeTab)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
