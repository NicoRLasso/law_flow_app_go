package handlers

import (
	"fmt"
	"law_flow_app_go/db"
	"law_flow_app_go/middleware"
	"law_flow_app_go/services"
	"law_flow_app_go/services/i18n"
	"law_flow_app_go/templates/partials"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetImportTemplateHandler generates and serves the Excel template
func GetImportTemplateHandler(c echo.Context) error {
	// Set locale from context (ensures correct language for headers/instructions)
	ctx := c.Request().Context()
	currentFirm := middleware.GetCurrentFirm(c)

	buf, err := services.GenerateExcelTemplate(ctx, db.DB, currentFirm.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate template")
	}

	// Serve file
	lang := i18n.GetLocale(ctx)
	filename := fmt.Sprintf("case_import_template_%s.xlsx", lang)

	c.Response().Header().Set("Content-Disposition", "attachment; filename="+filename)
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	return c.Blob(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// ImportCasesModalHandler renders the import modal
func ImportCasesModalHandler(c echo.Context) error {
	component := partials.CaseImportModal(c.Request().Context())
	return component.Render(c.Request().Context(), c.Response().Writer)
}

// ImportCasesHandler handles the Excel file upload and processing
func ImportCasesHandler(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	currentFirm := middleware.GetCurrentFirm(c)
	ctx := c.Request().Context()

	// Parse file
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No file uploaded"})
	}

	// Validate file extension
	// (Simple check, real validation is done by excelize opening it)

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to open file"})
	}
	defer src.Close()

	// Process Import
	result, err := services.BulkCreateFromExcel(ctx, db.DB, currentFirm.ID, currentUser.ID, src)

	// Prepare Header Triggers
	// "reload-cases" updates the table
	// "close-import-modal" closes the modal (only if full success)

	if err != nil {
		errorMsg := i18n.T(ctx, "cases.import.failed_msg")
		return c.HTML(http.StatusOK, fmt.Sprintf(`<div class="p-4 bg-red-100 text-red-700 rounded-lg">
			<h4 class="font-bold">Error</h4>
			<p>%s: %v</p>
		</div>`, errorMsg, err))
	}

	// Full Success: Close Modal, Refresh Table
	if result.FailedCount == 0 && result.SuccessCount > 0 {
		c.Response().Header().Set("HX-Trigger", `{"reload-cases": true, "close-import-modal": true}`)
		return c.NoContent(http.StatusOK)
	}

	// Partial Success or Failure: Show Summary, Refresh Table (if any success)
	if result.SuccessCount > 0 {
		c.Response().Header().Set("HX-Trigger", `{"reload-cases": true}`)
	}

	successMsg := i18n.T(ctx, "cases.import.success_msg")

	summaryHtml := fmt.Sprintf(`
		<div class="space-y-4">
			<div class="p-4 bg-green-50/10 border border-green-500/20 rounded-lg">
				<h4 class="font-bold text-green-400">%s</h4>
				<div class="grid grid-cols-3 gap-4 mt-2 text-sm">
					<div>
						<span class="block text-gray-400">%s</span>
						<span class="text-xl font-bold text-white">%d</span>
					</div>
					<div>
						<span class="block text-gray-400">%s</span>
						<span class="text-xl font-bold text-green-400">%d</span>
					</div>
					<div>
						<span class="block text-gray-400">%s</span>
						<span class="text-xl font-bold text-red-400">%d</span>
					</div>
				</div>
			</div>
	`,
		successMsg,
		i18n.T(ctx, "cases.import.summary_total"), result.TotalProcessed,
		i18n.T(ctx, "cases.import.summary_success"), result.SuccessCount,
		i18n.T(ctx, "cases.import.summary_failed"), result.FailedCount,
	)

	if len(result.Errors) > 0 {
		summaryHtml += `<div class="p-4 bg-red-500/10 border border-red-500/20 rounded-lg max-h-60 overflow-y-auto">
			<h5 class="font-bold text-red-400 mb-2">Errors Details</h5>
			<ul class="list-disc list-inside text-sm text-red-300 space-y-1">`
		for _, errMsg := range result.Errors {
			summaryHtml += fmt.Sprintf("<li>%s</li>", errMsg)
		}
		summaryHtml += `</ul></div>`
	}

	summaryHtml += `
		<div class="flex justify-end pt-4">
			<button onclick="document.getElementById('import-cases-modal').remove()" class="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-white text-sm transition-colors">
				` + i18n.T(ctx, "common.close") + `
			</button>
		</div>
	</div>`

	return c.HTML(http.StatusOK, summaryHtml)
}
