package services

import (
	"context"
	"fmt"
	"os"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// getChromePath returns the Chrome executable path from environment variable
func getChromePath() string {
	return os.Getenv("CHROME_PATH")
}

// PDFOptions contains options for PDF generation
type PDFOptions struct {
	PageOrientation string // portrait, landscape
	PageSize        string // letter, legal, A4
	MarginTop       int    // points (72 = 1 inch)
	MarginBottom    int
	MarginLeft      int
	MarginRight     int
}

// DefaultPDFOptions returns default options for legal documents
func DefaultPDFOptions() PDFOptions {
	return PDFOptions{
		PageOrientation: "portrait",
		PageSize:        "letter",
		MarginTop:       72,
		MarginBottom:    72,
		MarginLeft:      72,
		MarginRight:     72,
	}
}

// GeneratePDF renders HTML content to PDF using headless Chrome
func GeneratePDF(htmlContent string, options PDFOptions) ([]byte, error) {
	// Configure Chrome executable path from environment or default
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.DisableGPU,
	)

	// Check for custom Chrome path (for headless-shell in Docker)
	if chromePath := getChromePath(); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	// Create a new browser context
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set up page dimensions based on options
	var paperWidth, paperHeight float64
	switch options.PageSize {
	case "legal":
		paperWidth = 8.5
		paperHeight = 14.0
	case "A4":
		paperWidth = 8.27
		paperHeight = 11.69
	default: // letter
		paperWidth = 8.5
		paperHeight = 11.0
	}

	// Swap dimensions for landscape
	if options.PageOrientation == "landscape" {
		paperWidth, paperHeight = paperHeight, paperWidth
	}

	// Convert points to inches for margins
	marginTop := float64(options.MarginTop) / 72.0
	marginBottom := float64(options.MarginBottom) / 72.0
	marginLeft := float64(options.MarginLeft) / 72.0
	marginRight := float64(options.MarginRight) / 72.0

	var pdfBuf []byte

	// Run the Chrome actions
	err := chromedp.Run(ctx,
		// Navigate to a blank page first
		chromedp.Navigate("about:blank"),
		// Set the HTML content
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, htmlContent).Do(ctx)
		}),
		// Wait for content to render
		chromedp.Sleep(100),
		// Generate PDF
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPaperWidth(paperWidth).
				WithPaperHeight(paperHeight).
				WithMarginTop(marginTop).
				WithMarginBottom(marginBottom).
				WithMarginLeft(marginLeft).
				WithMarginRight(marginRight).
				WithPrintBackground(true).
				WithDisplayHeaderFooter(false).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuf = buf
			return nil
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return pdfBuf, nil
}

// GeneratePDFFromTemplate is a convenience function that wraps HTML and generates PDF
func GeneratePDFFromTemplate(renderedHTML string, options PDFOptions) ([]byte, error) {
	// Wrap the rendered content with legal document styles
	fullHTML := WrapHTMLForPDF(renderedHTML)
	return GeneratePDF(fullHTML, options)
}
