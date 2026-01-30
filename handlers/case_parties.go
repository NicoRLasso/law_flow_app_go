package handlers

import (
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"law_flow_app_go/templates/partials"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// GetCasePartyModalHandler returns the modal for adding/editing an opposing party
func GetCasePartyModalHandler(c echo.Context) error {
	caseID := c.Param("id")
	firm := middleware.GetCurrentFirm(c)

	// Fetch case with firm scoping
	var caseRecord models.Case
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.Preload("OpposingParty").Preload("OpposingParty.DocumentType").First(&caseRecord, "id = ?", caseID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Fetch document types for the form
	documentTypes, _ := services.GetChoiceOptions(db.DB, firm.ID, "document_type")

	// Determine the party type for the opposing party (opposite of client's role)
	partyType := models.ClientRoleDemandado // Default if client is demandante
	if caseRecord.ClientRole != nil && *caseRecord.ClientRole == models.ClientRoleDemandado {
		partyType = models.ClientRoleDemandante
	}

	// Render the modal
	component := partials.CasePartyModal(c.Request().Context(), caseRecord, documentTypes, partyType)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// AddCasePartyHandler adds an opposing party to a case
func AddCasePartyHandler(c echo.Context) error {
	caseID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	// Only admins can add opposing parties
	if currentUser.Role != "admin" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusForbidden, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Solo los administradores pueden gestionar las partes</div>`)
		}
		return echo.NewHTTPError(http.StatusForbidden, "Only admins can manage parties")
	}

	// Fetch case with firm scoping
	var caseRecord models.Case
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.Preload("OpposingParty").First(&caseRecord, "id = ?", caseID).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusNotFound, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Caso no encontrado</div>`)
		}
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Check if case already has an opposing party
	if caseRecord.OpposingParty != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-yellow-500/20 text-yellow-400 rounded-lg">Este caso ya tiene una contraparte</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Case already has an opposing party")
	}

	// Parse form data
	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">El nombre es requerido</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Name is required")
	}

	email := strings.TrimSpace(c.FormValue("email"))
	phone := strings.TrimSpace(c.FormValue("phone"))
	documentTypeID := strings.TrimSpace(c.FormValue("document_type_id"))
	documentNumber := strings.TrimSpace(c.FormValue("document_number"))

	// Determine party type (opposite of client's role)
	partyType := models.ClientRoleDemandado
	if caseRecord.ClientRole != nil && *caseRecord.ClientRole == models.ClientRoleDemandado {
		partyType = models.ClientRoleDemandante
	}

	// Create the opposing party
	party := models.CaseParty{
		CaseID:    caseID,
		PartyType: partyType,
		Name:      name,
	}

	if email != "" {
		emailLower := strings.ToLower(email)
		party.Email = &emailLower
	}
	if phone != "" {
		party.Phone = &phone
	}
	if documentTypeID != "" {
		party.DocumentTypeID = &documentTypeID
	}
	if documentNumber != "" {
		party.DocumentNumber = &documentNumber
	}

	if err := db.DB.Create(&party).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Error al agregar la contraparte</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to add opposing party")
	}

	// Return success and trigger page reload
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, `
			<div class="p-4 bg-green-500/20 text-green-400 rounded-lg mb-4" x-init="setTimeout(() => window.location.reload(), 1000)">
				Contraparte agregada exitosamente
			</div>
		`)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Opposing party added successfully",
	})
}

// UpdateCasePartyHandler updates an existing opposing party
func UpdateCasePartyHandler(c echo.Context) error {
	caseID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	// Only admins can update opposing parties
	if currentUser.Role != "admin" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusForbidden, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Solo los administradores pueden gestionar las partes</div>`)
		}
		return echo.NewHTTPError(http.StatusForbidden, "Only admins can manage parties")
	}

	// Fetch case with firm scoping
	var caseRecord models.Case
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.Preload("OpposingParty").First(&caseRecord, "id = ?", caseID).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusNotFound, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Caso no encontrado</div>`)
		}
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Check if case has an opposing party to update
	if caseRecord.OpposingParty == nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-yellow-500/20 text-yellow-400 rounded-lg">Este caso no tiene una contraparte para actualizar</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Case does not have an opposing party")
	}

	// Parse form data
	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">El nombre es requerido</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Name is required")
	}

	email := strings.TrimSpace(c.FormValue("email"))
	phone := strings.TrimSpace(c.FormValue("phone"))
	documentTypeID := strings.TrimSpace(c.FormValue("document_type_id"))
	documentNumber := strings.TrimSpace(c.FormValue("document_number"))

	// Update the opposing party
	party := caseRecord.OpposingParty
	party.Name = name

	if email != "" {
		emailLower := strings.ToLower(email)
		party.Email = &emailLower
	} else {
		party.Email = nil
	}
	if phone != "" {
		party.Phone = &phone
	} else {
		party.Phone = nil
	}
	if documentTypeID != "" {
		party.DocumentTypeID = &documentTypeID
	} else {
		party.DocumentTypeID = nil
	}
	if documentNumber != "" {
		party.DocumentNumber = &documentNumber
	} else {
		party.DocumentNumber = nil
	}

	if err := db.DB.Save(party).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Error al actualizar la contraparte</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update opposing party")
	}

	// Return success and trigger page reload
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, `
			<div class="p-4 bg-green-500/20 text-green-400 rounded-lg mb-4" x-init="setTimeout(() => window.location.reload(), 1000)">
				Contraparte actualizada exitosamente
			</div>
		`)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Opposing party updated successfully",
	})
}

// DeleteCasePartyHandler removes the opposing party from a case
func DeleteCasePartyHandler(c echo.Context) error {
	caseID := c.Param("id")
	currentUser := middleware.GetCurrentUser(c)

	// Only admins can delete opposing parties
	if currentUser.Role != "admin" {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusForbidden, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Solo los administradores pueden gestionar las partes</div>`)
		}
		return echo.NewHTTPError(http.StatusForbidden, "Only admins can manage parties")
	}

	// Fetch case with firm scoping
	var caseRecord models.Case
	query := middleware.GetFirmScopedQuery(c, db.DB)
	if err := query.Preload("OpposingParty").First(&caseRecord, "id = ?", caseID).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusNotFound, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Caso no encontrado</div>`)
		}
		return echo.NewHTTPError(http.StatusNotFound, "Case not found")
	}

	// Check if case has an opposing party to delete
	if caseRecord.OpposingParty == nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="p-4 bg-yellow-500/20 text-yellow-400 rounded-lg">Este caso no tiene una contraparte para eliminar</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Case does not have an opposing party")
	}

	// Delete the opposing party
	if err := db.DB.Delete(caseRecord.OpposingParty).Error; err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="p-4 bg-red-500/20 text-red-400 rounded-lg">Error al eliminar la contraparte</div>`)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete opposing party")
	}

	// Return success and trigger page reload
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, `
			<div class="p-4 bg-green-500/20 text-green-400 rounded-lg mb-4" x-init="setTimeout(() => window.location.reload(), 1000)">
				Contraparte eliminada exitosamente
			</div>
		`)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Opposing party deleted successfully",
	})
}
