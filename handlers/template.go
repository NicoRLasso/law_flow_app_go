package handlers

import (
	"context"
	"net/http"

	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/pages"
	"law_flow_app_go/templates/partials"

	"github.com/labstack/echo/v4"
)

// TemplatesPageHandler renders the templates management page
func TemplatesPageHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)
	firm := c.Get("firm").(*models.Firm)
	csrfToken := middleware.GetCSRFToken(c)
	ctx := context.Background()

	var categories []models.TemplateCategory
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		Where("is_active = ?", true).
		Order("sort_order ASC, name ASC").
		Find(&categories).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error fetching categories")
	}

	return pages.TemplatesPage(ctx, "Document Templates | "+firm.Name, csrfToken, user, firm, categories).Render(c.Request().Context(), c.Response().Writer)
}

// GetTemplatesHandler returns the list of templates for the current firm
func GetTemplatesHandler(c echo.Context) error {
	categoryID := c.QueryParam("category_id")
	search := c.QueryParam("search")
	activeOnly := c.QueryParam("active") == "true"

	var templates []models.DocumentTemplate
	query := middleware.GetFirmScopedQuery(c, db.DB).
		Preload("Category").
		Preload("CreatedBy").
		Order("name ASC")

	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	if search != "" {
		likeSearch := "%" + search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", likeSearch, likeSearch)
	}

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&templates).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error fetching templates")
	}

	ctx := context.Background()
	return partials.TemplateTable(ctx, templates).Render(c.Request().Context(), c.Response().Writer)
}

// GetTemplateHandler returns a single template
func GetTemplateHandler(c echo.Context) error {
	id := c.Param("id")

	var template models.DocumentTemplate
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		Preload("Category").
		First(&template, "id = ?", id).Error; err != nil {
		return c.String(http.StatusNotFound, "Template not found")
	}

	return c.JSON(http.StatusOK, template)
}

// TemplateEditorPageHandler renders the template create/edit page
func TemplateEditorPageHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)
	firm := c.Get("firm").(*models.Firm)
	csrfToken := middleware.GetCSRFToken(c)
	id := c.Param("id")
	ctx := context.Background()

	// Fetch categories for dropdown
	var categories []models.TemplateCategory
	middleware.GetFirmScopedQuery(c, db.DB).Where("is_active = ?", true).Order("sort_order ASC, name ASC").Find(&categories)

	if id == "" || id == "new" {
		// New template page
		return pages.TemplateEditor(ctx, "New Template | "+firm.Name, csrfToken, user, firm, models.DocumentTemplate{}, categories, true).Render(c.Request().Context(), c.Response().Writer)
	}

	// Edit existing template
	var template models.DocumentTemplate
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&template, "id = ?", id).Error; err != nil {
		return c.Redirect(http.StatusFound, "/templates")
	}

	return pages.TemplateEditor(ctx, "Edit Template | "+firm.Name, csrfToken, user, firm, template, categories, false).Render(c.Request().Context(), c.Response().Writer)
}

// CreateTemplateHandler creates a new document template
func CreateTemplateHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)
	firmID := *user.FirmID

	name := c.FormValue("name")
	description := c.FormValue("description")
	categoryID := c.FormValue("category_id")
	content := c.FormValue("content")
	pageOrientation := c.FormValue("page_orientation")
	pageSize := c.FormValue("page_size")

	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	if content == "" {
		content = "<p></p>" // Default empty content
	}

	if pageOrientation == "" {
		pageOrientation = models.OrientationPortrait
	}
	if pageSize == "" {
		pageSize = models.PageSizeLetter
	}

	template := models.DocumentTemplate{
		FirmID:          firmID,
		Name:            name,
		Content:         content,
		PageOrientation: pageOrientation,
		PageSize:        pageSize,
		CreatedByID:     user.ID,
		IsActive:        true,
		Version:         1,
		MarginTop:       72,
		MarginBottom:    72,
		MarginLeft:      72,
		MarginRight:     72,
	}

	if description != "" {
		template.Description = &description
	}
	if categoryID != "" {
		template.CategoryID = &categoryID
	}

	if err := db.DB.Create(&template).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error creating template")
	}

	// Redirect to templates list
	c.Response().Header().Set("HX-Redirect", "/templates")
	return c.NoContent(http.StatusOK)
}

// UpdateTemplateHandler updates an existing template
func UpdateTemplateHandler(c echo.Context) error {
	id := c.Param("id")

	var template models.DocumentTemplate
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&template, "id = ?", id).Error; err != nil {
		return c.String(http.StatusNotFound, "Template not found")
	}

	name := c.FormValue("name")
	description := c.FormValue("description")
	categoryID := c.FormValue("category_id")
	content := c.FormValue("content")
	pageOrientation := c.FormValue("page_orientation")
	pageSize := c.FormValue("page_size")
	isActive := c.FormValue("is_active") == "true" || c.FormValue("is_active") == "on"

	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	// Increment version if content changed
	if template.Content != content {
		template.Version++
	}

	template.Name = name
	template.Content = content
	template.IsActive = isActive

	if description != "" {
		template.Description = &description
	} else {
		template.Description = nil
	}

	if categoryID != "" {
		template.CategoryID = &categoryID
	} else {
		template.CategoryID = nil
	}

	if pageOrientation != "" && models.IsValidOrientation(pageOrientation) {
		template.PageOrientation = pageOrientation
	}
	if pageSize != "" && models.IsValidPageSize(pageSize) {
		template.PageSize = pageSize
	}

	if err := db.DB.Save(&template).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error updating template")
	}

	// Redirect to templates list
	c.Response().Header().Set("HX-Redirect", "/templates")
	return c.NoContent(http.StatusOK)
}

// DeleteTemplateHandler soft-deletes a template
func DeleteTemplateHandler(c echo.Context) error {
	id := c.Param("id")

	var template models.DocumentTemplate
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&template, "id = ?", id).Error; err != nil {
		return c.String(http.StatusNotFound, "Template not found")
	}

	if err := db.DB.Delete(&template).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error deleting template")
	}

	return GetTemplatesHandler(c)
}

// GetTemplateVariablesHandler returns the variable dictionary for the editor
func GetTemplateVariablesHandler(c echo.Context) error {
	variables := services.GetVariableDictionary()
	return c.JSON(http.StatusOK, variables)
}

// --- Category Handlers ---

// GetCategoriesHandler returns the list of template categories as HTML
func GetCategoriesHandler(c echo.Context) error {
	var categories []models.TemplateCategory
	if err := middleware.GetFirmScopedQuery(c, db.DB).
		Order("sort_order ASC, name ASC").
		Find(&categories).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error fetching categories")
	}

	ctx := context.Background()
	return partials.CategoryList(ctx, categories).Render(c.Request().Context(), c.Response().Writer)
}

// CreateCategoryHandler creates a new template category
func CreateCategoryHandler(c echo.Context) error {
	user := c.Get("user").(*models.User)
	firmID := *user.FirmID

	name := c.FormValue("name")
	description := c.FormValue("description")

	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	category := models.TemplateCategory{
		FirmID:   firmID,
		Name:     name,
		IsActive: true,
	}

	if description != "" {
		category.Description = &description
	}

	if err := db.DB.Create(&category).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error creating category")
	}

	// Return updated list
	return GetCategoriesHandler(c)
}

// UpdateCategoryHandler updates a template category
func UpdateCategoryHandler(c echo.Context) error {
	id := c.Param("id")

	var category models.TemplateCategory
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&category, "id = ?", id).Error; err != nil {
		return c.String(http.StatusNotFound, "Category not found")
	}

	name := c.FormValue("name")
	description := c.FormValue("description")

	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	category.Name = name
	if description != "" {
		category.Description = &description
	} else {
		category.Description = nil
	}

	if err := db.DB.Save(&category).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error updating category")
	}

	// Return updated list
	return GetCategoriesHandler(c)
}

// DeleteCategoryHandler soft-deletes a template category
func DeleteCategoryHandler(c echo.Context) error {
	id := c.Param("id")

	var category models.TemplateCategory
	if err := middleware.GetFirmScopedQuery(c, db.DB).First(&category, "id = ?", id).Error; err != nil {
		return c.String(http.StatusNotFound, "Category not found")
	}

	if err := db.DB.Delete(&category).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Error deleting category")
	}

	// Return updated list
	return GetCategoriesHandler(c)
}
