package editor

import "law_flow_app_go/models"

func GetPaperStyle(t models.DocumentTemplate) string {
	width := "816px" // Letter default (8.5" * 96dpi)

	if t.PageSize == "A4" {
		width = "794px" // 210mm at 96dpi
	}
	// Legal uses same width as Letter (8.5" = 816px)

	return "width: " + width + "; max-width: 100%;"
}

func GetPageHeight(t models.DocumentTemplate) string {
	if t.PageSize == "A4" {
		return "1123" // 297mm at 96dpi
	} else if t.PageSize == "legal" {
		return "1344" // 14" at 96dpi
	}
	return "1056" // Letter: 11" at 96dpi
}
