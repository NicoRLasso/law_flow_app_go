package handlers

import (
	"encoding/xml"
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type SitemapURL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float32 `xml:"priority,omitempty"`
}

type SitemapURLSet struct {
	XMLName string       `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []SitemapURL `xml:"url"`
}

// GetSitemapHandler generates a dynamic XML sitemap
func GetSitemapHandler(c echo.Context) error {
	cfg := c.Get("config").(*config.Config)
	baseURL := cfg.AppURL

	// Static pages
	urls := []SitemapURL{
		{Loc: baseURL + "/", ChangeFreq: "weekly", Priority: 1.0},
		{Loc: baseURL + "/about", ChangeFreq: "monthly", Priority: 0.8},
		{Loc: baseURL + "/contact", ChangeFreq: "monthly", Priority: 0.8},
		{Loc: baseURL + "/security", ChangeFreq: "monthly", Priority: 0.7},
		{Loc: baseURL + "/privacy", ChangeFreq: "yearly", Priority: 0.5},
		{Loc: baseURL + "/terms", ChangeFreq: "yearly", Priority: 0.5},
		{Loc: baseURL + "/cookies", ChangeFreq: "yearly", Priority: 0.5},
		{Loc: baseURL + "/compliance", ChangeFreq: "yearly", Priority: 0.6},
	}

	// Dynamic pages: Active Firms
	var firms []models.Firm
	// Fetch active firms
	if err := db.DB.Where("is_active = ?", true).Find(&firms).Error; err != nil {
		c.Logger().Error("Failed to fetch firms for sitemap", err)
		// Continue with static pages if DB fails, or consider returning error
	}

	for _, firm := range firms {
		// Only include firms with a valid slug
		if firm.Slug != "" {
			// Public Case Request Page
			urls = append(urls, SitemapURL{
				Loc:        baseURL + "/firm/" + firm.Slug + "/request",
				ChangeFreq: "daily",
				Priority:   0.9,
				LastMod:    firm.UpdatedAt.Format(time.RFC3339),
			})

			// Public Booking Page
			urls = append(urls, SitemapURL{
				Loc:        baseURL + "/firm/" + firm.Slug + "/book",
				ChangeFreq: "daily",
				Priority:   0.9,
				LastMod:    firm.UpdatedAt.Format(time.RFC3339),
			})
		}
	}

	urlSet := SitemapURLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationXML)
	c.Response().WriteHeader(http.StatusOK)
	if _, err := c.Response().Write([]byte(xml.Header)); err != nil {
		return err
	}

	encoder := xml.NewEncoder(c.Response().Writer)
	encoder.Indent("", "  ")
	return encoder.Encode(urlSet)
}
