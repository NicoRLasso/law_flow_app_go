package handlers

import "law_flow_app_go/models"

const (
	baseURL        = "https://lexlegalcloud.org"
	defaultOGImage = "https://lexlegalcloud.org/static/images/og-image.png"
)

// SEO configurations for public pages
var pageSEO = map[string]*models.SEO{
	"landing": {
		Title:       "LexLegal Cloud - Modern Legal Practice Management",
		Description: "Streamline your legal practice with LexLegal Cloud. Manage cases, clients, documents, and appointments in one secure, cloud-based platform designed for modern law firms.",
		Keywords:    "legal practice management, law firm software, case management, legal cloud software, attorney software",
		Canonical:   baseURL + "/",
		OGImage:     defaultOGImage,
		OGType:      "website",
		TwitterCard: "summary_large_image",
		Locale:      "es",
		AltLocales:  []string{"en"},
	},
	"about": {
		Title:       "About Us | LexLegal Cloud",
		Description: "Learn about LexLegal Cloud's mission to modernize legal practice management. We help law firms work smarter with intuitive, secure cloud-based solutions.",
		Keywords:    "about LexLegal Cloud, legal tech company, law firm solutions",
		Canonical:   baseURL + "/about",
		OGImage:     defaultOGImage,
		OGType:      "website",
		TwitterCard: "summary_large_image",
		Locale:      "es",
		AltLocales:  []string{"en"},
	},
	"contact": {
		Title:       "Contact Us | LexLegal Cloud",
		Description: "Get in touch with LexLegal Cloud. We're here to help you transform your legal practice with our cloud-based management solutions.",
		Keywords:    "contact LexLegal Cloud, legal software support, law firm software inquiry",
		Canonical:   baseURL + "/contact",
		OGImage:     defaultOGImage,
		OGType:      "website",
		TwitterCard: "summary_large_image",
		Locale:      "es",
		AltLocales:  []string{"en"},
	},
	"security": {
		Title:       "Security | LexLegal Cloud",
		Description: "Your data security is our priority. Learn about LexLegal Cloud's enterprise-grade security measures, encryption, and compliance certifications.",
		Keywords:    "legal data security, law firm security, cloud security, data encryption, GDPR compliance",
		Canonical:   baseURL + "/security",
		OGImage:     defaultOGImage,
		OGType:      "website",
		TwitterCard: "summary_large_image",
		Locale:      "es",
		AltLocales:  []string{"en"},
	},
	"privacy": {
		Title:       "Privacy Policy | LexLegal Cloud",
		Description: "Read LexLegal Cloud's privacy policy. Learn how we collect, use, and protect your personal information.",
		Keywords:    "privacy policy, data protection, personal information",
		Canonical:   baseURL + "/privacy",
		OGImage:     defaultOGImage,
		OGType:      "website",
		TwitterCard: "summary",
		Locale:      "es",
		AltLocales:  []string{"en"},
	},
	"terms": {
		Title:       "Terms of Service | LexLegal Cloud",
		Description: "Review LexLegal Cloud's terms of service. Understand the terms and conditions governing your use of our legal practice management platform.",
		Keywords:    "terms of service, legal terms, user agreement",
		Canonical:   baseURL + "/terms",
		OGImage:     defaultOGImage,
		OGType:      "website",
		TwitterCard: "summary",
		Locale:      "es",
		AltLocales:  []string{"en"},
	},
	"cookies": {
		Title:       "Cookie Policy | LexLegal Cloud",
		Description: "Learn about how LexLegal Cloud uses cookies to improve your experience on our platform.",
		Keywords:    "cookie policy, cookies, website cookies",
		Canonical:   baseURL + "/cookies",
		OGImage:     defaultOGImage,
		OGType:      "website",
		TwitterCard: "summary",
		Locale:      "es",
		AltLocales:  []string{"en"},
	},
	"compliance": {
		Title:       "Compliance | LexLegal Cloud",
		Description: "Discover LexLegal Cloud's compliance certifications and regulatory adherence. We meet the highest standards for legal industry requirements.",
		Keywords:    "compliance, legal compliance, regulatory compliance, GDPR, data protection",
		Canonical:   baseURL + "/compliance",
		OGImage:     defaultOGImage,
		OGType:      "website",
		TwitterCard: "summary_large_image",
		Locale:      "es",
		AltLocales:  []string{"en"},
	},
}

// GetSEO returns the SEO configuration for a page
func GetSEO(page string) *models.SEO {
	if seo, ok := pageSEO[page]; ok {
		// Return a copy to avoid mutations
		copy := *seo
		return &copy
	}
	return nil
}
