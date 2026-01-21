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
	component := company.About(c.Request().Context(), "About Us | LexLegal Cloud", csrfToken)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteContactHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	component := company.Contact(c.Request().Context(), "Contact Us | LexLegal Cloud", csrfToken)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteSecurityHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	component := product.Security(c.Request().Context(), "Security | LexLegal Cloud", csrfToken)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsitePrivacyHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	component := legal.Privacy(c.Request().Context(), "Privacy Policy | LexLegal Cloud", csrfToken)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteTermsHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	component := legal.Terms(c.Request().Context(), "Terms of Service | LexLegal Cloud", csrfToken)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteCookiesHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	component := legal.Cookies(c.Request().Context(), "Cookie Policy | LexLegal Cloud", csrfToken)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func WebsiteComplianceHandler(c echo.Context) error {
	csrfToken := middleware.GetCSRFToken(c)
	component := legal.Compliance(c.Request().Context(), "Compliance | LexLegal Cloud", csrfToken)
	return component.Render(c.Request().Context(), c.Response().Writer)
}
