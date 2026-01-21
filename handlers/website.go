package handlers

import (
	"law_flow_app_go/middleware"
	"law_flow_app_go/templates/pages/company"
	"law_flow_app_go/templates/pages/legal"
	"law_flow_app_go/templates/pages/product"

	"github.com/labstack/echo/v4"
)

func WebsiteAboutHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("about")
	component := company.About(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteContactHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("contact")
	component := company.Contact(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteSecurityHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("security")
	component := product.Security(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsitePrivacyHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("privacy")
	component := legal.Privacy(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteTermsHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("terms")
	component := legal.Terms(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteCookiesHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("cookies")
	component := legal.Cookies(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteComplianceHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	seo := GetSEO("compliance")
	component := legal.Compliance(c.Request().Context(), seo.Title, csrfToken, seo)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
