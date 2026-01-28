package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to open file"})
	}
	defer src.Close()

	// Read file to buffer for multiple reads (Analysis + Import)
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, src); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to buffer file"})
	}
	fileBytes := buf.Bytes()

	// 1. Analyze File
	totalRows, err := services.AnalyzeExcelFile(bytes.NewReader(fileBytes))
	if err != nil {
		return c.HTML(http.StatusOK, fmt.Sprintf(`<div class="p-4 bg-red-100 text-red-700 rounded-lg">
			<h4 class="font-bold">Error</h4>
			<p>%v</p>
		</div>`, err))
	}

	// 2. Check Limits
	limitCheck, err := services.CanAddCase(db.DB, currentFirm.ID)
	if err != nil {
		// If error checking limit, fail safe or block? Block is safer.
		return c.HTML(http.StatusOK, fmt.Sprintf(`<div class="p-4 bg-red-100 text-red-700 rounded-lg">
			<h4 class="font-bold">System Error</h4>
			<p>Failed to verify subscription limits: %v</p>
		</div>`, err))
	}

	allowedCount := totalRows
	skippedCount := 0
	limitArg := -1

	// limitCheck.Limit == -1 means Unlimited
	if limitCheck.Limit != -1 {
		remaining := limitCheck.Limit - limitCheck.CurrentUsage
		if remaining < 0 {
			remaining = 0
		}

		if int64(totalRows) > remaining {
			allowedCount = int(remaining)
			skippedCount = totalRows - int(remaining)
			limitArg = int(remaining) // Passed to service to stop import after this many
		}
	}

	// 3. Estimate Time (Heuristic: 0.5s per case)
	estimatedSeconds := float64(allowedCount) * 0.5
	estimatedTimeMsg := "Less than a minute"
	if estimatedSeconds > 60 {
		mins := int(estimatedSeconds / 60)
		estimatedTimeMsg = fmt.Sprintf("~%d minutes", mins)
	}

	// 4. Async Execution
	// Use a background context as the request context will be cancelled
	bgCtx := context.Background()

	go func() {
		// New buffer reader for the goroutine
		reader := bytes.NewReader(fileBytes)
		_, err := services.BulkCreateFromExcel(bgCtx, db.DB, currentFirm.ID, currentUser.ID, reader, limitArg)

		// TODO: Notification system integration would go here
		// For now, we rely on the user refreshing or checking the case list later.
		if err != nil {
			fmt.Printf("Async import failed: %v\n", err)
		}
	}()

	// 5. Immediate Feedback
	msgStarted := i18n.T(ctx, "cases.import.started_msg")
	if msgStarted == "cases.import.started_msg" {
		msgStarted = "Import started in background."
	}

	summaryHtml := fmt.Sprintf(`
		<div class="space-y-4">
			<div class="p-4 bg-blue-50/10 border border-blue-500/20 rounded-lg">
				<h4 class="font-bold text-blue-400">%s</h4>
				<p class="text-sm text-gray-400 mt-1">
					Processing <strong>%d</strong> cases. Estimated time: <strong>%s</strong>.
				</p>
				
				<div class="grid grid-cols-3 gap-4 mt-4 text-sm">
					<div>
						<span class="block text-gray-400">Total Found</span>
						<span class="text-xl font-bold text-white">%d</span>
					</div>
					<div>
						<span class="block text-gray-400">To Import</span>
						<span class="text-xl font-bold text-blue-400">%d</span>
					</div>
					<div>
						<span class="block text-gray-400">Skipped (Over Limit)</span>
						<span class="text-xl font-bold text-orange-400">%d</span>
					</div>
				</div>
			</div>
	`, msgStarted, totalRows, estimatedTimeMsg, totalRows, allowedCount, skippedCount)

	if skippedCount > 0 {
		summaryHtml += fmt.Sprintf(`
			<div class="p-4 bg-orange-500/10 border border-orange-500/20 rounded-lg">
				<h4 class="font-bold text-orange-400">Subscription Limit Reached</h4>
				<p class="text-sm text-orange-200 mt-1">
					Your plan only allows for <strong>%d</strong> more cases. The remaining <strong>%d</strong> cases in this file will be skipped.
					<br><a href="/settings/billing" class="underline font-bold" target="_blank">Upgrade your plan</a> to increase your limit.
				</p>
			</div>
		`, allowedCount, skippedCount)
	}

	summaryHtml += `
		<div class="flex justify-between items-center pt-4">
			<span class="text-xs text-gray-500">You can close this window. The import will continue in the background.</span>
			<button onclick="document.getElementById('import-cases-modal').remove()" class="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-white text-sm transition-colors">
				` + i18n.T(ctx, "common.close") + `
			</button>
		</div>
	</div>`

	// Trigger table reload immediately so users see rows appearing
	c.Response().Header().Set("HX-Trigger", `{"reload-cases": true}`)

	return c.HTML(http.StatusOK, summaryHtml)
}
