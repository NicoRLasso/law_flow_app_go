package models

// SEO contains metadata for search engine optimization and social sharing
type SEO struct {
	Title       string // Page title
	Description string // Meta description (150-160 chars recommended)
	Keywords    string // Meta keywords (comma-separated)
	Canonical   string // Canonical URL
	OGTitle     string // Open Graph title (defaults to Title if empty)
	OGDesc      string // Open Graph description (defaults to Description if empty)
	OGImage     string // Open Graph image URL
	OGType      string // Open Graph type (website, article, etc.)
	TwitterCard string // Twitter card type (summary, summary_large_image)
	NoIndex     bool   // If true, adds noindex directive
	Locale      string // Current locale (e.g., "en", "es")
	AltLocales  []string // Alternative locales for hreflang
}

// DefaultSEO returns SEO with sensible defaults
func DefaultSEO(title, description string) *SEO {
	return &SEO{
		Title:       title,
		Description: description,
		OGType:      "website",
		TwitterCard: "summary_large_image",
		Locale:      "en",
		AltLocales:  []string{"es"},
	}
}

// WithCanonical sets the canonical URL
func (s *SEO) WithCanonical(url string) *SEO {
	s.Canonical = url
	return s
}

// WithOGImage sets the Open Graph image
func (s *SEO) WithOGImage(imageURL string) *SEO {
	s.OGImage = imageURL
	return s
}

// WithKeywords sets meta keywords
func (s *SEO) WithKeywords(keywords string) *SEO {
	s.Keywords = keywords
	return s
}

// WithLocale sets the current locale and alternative locales
func (s *SEO) WithLocale(locale string, altLocales ...string) *SEO {
	s.Locale = locale
	s.AltLocales = altLocales
	return s
}

// WithNoIndex sets the noindex directive
func (s *SEO) WithNoIndex() *SEO {
	s.NoIndex = true
	return s
}

// GetOGTitle returns OGTitle or falls back to Title
func (s *SEO) GetOGTitle() string {
	if s.OGTitle != "" {
		return s.OGTitle
	}
	return s.Title
}

// GetOGDesc returns OGDesc or falls back to Description
func (s *SEO) GetOGDesc() string {
	if s.OGDesc != "" {
		return s.OGDesc
	}
	return s.Description
}
