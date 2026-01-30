package handlers

import (
	"law_flow_app_go/services"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// BuildFilingNumberHandler builds a filing number from the provided input
// POST /api/tools/filing-number/build
func BuildFilingNumberHandler(c echo.Context) error {
	courtOfficeCode := c.FormValue("court_office_code")
	yearStr := c.FormValue("year")
	processCode := c.FormValue("process_code")
	resourceCode := c.FormValue("resource_code")

	// Parse year
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		year = time.Now().Year()
	}

	input := services.FilingNumberInput{
		CourtOfficeCode: courtOfficeCode,
		Year:            year,
		ProcessCode:     processCode,
		ResourceCode:    resourceCode,
	}

	// Validate input
	errors := services.ValidateFilingNumberInput(input)
	if len(errors) > 0 {
		if c.Request().Header.Get("HX-Request") == "true" {
			html := `<div class="alert alert-error rounded-sm"><ul class="list-disc list-inside">`
			for _, e := range errors {
				html += `<li>` + e + `</li>`
			}
			html += `</ul></div>`
			return c.HTML(http.StatusBadRequest, html)
		}
		return echo.NewHTTPError(http.StatusBadRequest, errors)
	}

	// Build filing number
	filingNumber := services.BuildFilingNumber(input)

	if c.Request().Header.Get("HX-Request") == "true" {
		html := `
		<div class="flex flex-col gap-4">
			<div class="bg-base-200 p-6 rounded-sm border border-base-300">
				<label class="label pt-0 pb-2">
					<span class="label-text text-xs font-bold uppercase tracking-wider opacity-60">Generated Filing Number</span>
				</label>
				<div class="flex items-center gap-4">
					<code id="filing-number-result" class="text-2xl font-mono font-bold text-primary flex-1">` + filingNumber + `</code>
					<button
						type="button"
						class="btn btn-outline btn-sm gap-2"
						x-data
						@click="navigator.clipboard.writeText('` + filingNumber + `'); $el.classList.add('btn-success'); $el.innerHTML = '<i data-lucide=\'check\'></i> Copied'; setTimeout(() => { $el.classList.remove('btn-success'); $el.innerHTML = '<i data-lucide=\'copy\'></i> Copy'; lucide.createIcons(); }, 2000); lucide.createIcons();"
					>
						<i data-lucide="copy"></i>
						Copy
					</button>
				</div>
			</div>
			<div class="text-xs text-base-content/60">
				<p><strong>Format:</strong> {court_office_code}{year}{process(5 digits)}{resource(2 digits)}</p>
			</div>
		</div>
		`
		return c.HTML(http.StatusOK, html)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"filing_number": filingNumber,
	})
}

// ParseFilingNumberHandler parses a filing number into its components
// POST /api/tools/filing-number/parse
func ParseFilingNumberHandler(c echo.Context) error {
	filingNumber := c.FormValue("filing_number")

	if filingNumber == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Filing number is required")
	}

	components, err := services.ParseFilingNumber(filingNumber)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="alert alert-error rounded-sm">`+err.Error()+`</div>`)
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		html := `
		<div class="bg-base-200 p-4 rounded-sm">
			<table class="table table-sm">
				<tbody>
					<tr><td class="font-semibold">Department</td><td class="font-mono">` + components.DepartmentCode + `</td></tr>
					<tr><td class="font-semibold">City</td><td class="font-mono">` + components.CityCode + `</td></tr>
					<tr><td class="font-semibold">Entity</td><td class="font-mono">` + components.EntityCode + `</td></tr>
					<tr><td class="font-semibold">Specialty</td><td class="font-mono">` + components.SpecialtyCode + `</td></tr>
					<tr><td class="font-semibold">Court Office</td><td class="font-mono">` + components.CourtOfficeCode + `</td></tr>
					<tr><td class="font-semibold">Year</td><td class="font-mono">` + components.Year + `</td></tr>
					<tr><td class="font-semibold">Process</td><td class="font-mono">` + components.ProcessCode + `</td></tr>
					<tr><td class="font-semibold">Resource</td><td class="font-mono">` + components.ResourceCode + `</td></tr>
				</tbody>
			</table>
		</div>
		`
		return c.HTML(http.StatusOK, html)
	}

	return c.JSON(http.StatusOK, components)
}
