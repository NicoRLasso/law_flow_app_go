package main

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/handlers"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"log"
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	if err := db.Initialize(cfg.DBPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.AutoMigrate(&models.Firm{}, &models.User{}, &models.Session{}, &models.PasswordResetToken{}, &models.CaseRequest{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(echomiddleware.RequestLogger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORS())

	// Make config available to handlers
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("config", cfg)
			return next(c)
		}
	})

	// Static files
	e.Static("/static", "static")

	// Public routes (no authentication required)
	e.GET("/", handlers.LandingHandler)
	e.GET("/login", handlers.LoginHandler)
	e.POST("/login", handlers.LoginPostHandler)
	e.GET("/forgot-password", handlers.ForgotPasswordHandler)
	e.POST("/forgot-password", handlers.ForgotPasswordPostHandler)
	e.GET("/reset-password", handlers.ResetPasswordHandler)
	e.POST("/reset-password", handlers.ResetPasswordPostHandler)

	// Public case request routes (no authentication)
	e.GET("/firm/:slug/request", handlers.PublicCaseRequestHandler)
	e.POST("/firm/:slug/request", handlers.PublicCaseRequestPostHandler)
	e.GET("/firm/:slug/request/success", handlers.PublicCaseRequestSuccessHandler)

	// Firm setup routes (authenticated but no firm required)
	firmSetup := e.Group("/firm")
	firmSetup.Use(middleware.RequireAuth())
	{
		firmSetup.GET("/setup", handlers.FirmSetupHandler)
		firmSetup.POST("/setup", handlers.FirmSetupPostHandler)
	}

	// Protected routes (authentication + firm required)
	protected := e.Group("")
	protected.Use(middleware.RequireAuth())
	protected.Use(middleware.RequireFirm()) // Ensure user has a firm
	{
		// All users with a firm can access dashboard and their own profile
		protected.GET("/dashboard", handlers.DashboardHandler)
		protected.POST("/logout", handlers.LogoutHandler)
		protected.GET("/api/me", handlers.GetCurrentUserHandler)

		// HTMX routes (all roles, firm-scoped)
		protected.GET("/htmx/users", handlers.GetUsersHTMX)

		// User viewing routes (all roles, firm-scoped, with handler-level auth checks)
		protected.GET("/api/users", handlers.GetUsers)
		protected.GET("/api/users/:id", handlers.GetUser)
		protected.PUT("/api/users/:id", handlers.UpdateUser)

		// Admin-only routes
		adminRoutes := protected.Group("")
		adminRoutes.Use(middleware.RequireRole("admin"))
		{
			adminRoutes.POST("/api/users", handlers.CreateUser)
			adminRoutes.DELETE("/api/users/:id", handlers.DeleteUser)
		}

		// Case request routes (admin and lawyer only)
		caseRequestRoutes := protected.Group("/api/case-requests")
		caseRequestRoutes.Use(middleware.RequireRole("admin", "lawyer"))
		{
			caseRequestRoutes.GET("", handlers.GetCaseRequestsHandler)
			caseRequestRoutes.GET("/:id", handlers.GetCaseRequestHandler)
			caseRequestRoutes.GET("/:id/detail", handlers.GetCaseRequestDetailHandler)
			caseRequestRoutes.GET("/:id/file", handlers.DownloadCaseRequestFileHandler)
			caseRequestRoutes.PUT("/:id/status", handlers.UpdateCaseRequestStatusHandler)
			caseRequestRoutes.DELETE("/:id", handlers.DeleteCaseRequestHandler)
		}

		// Case requests dashboard page
		protected.GET("/case-requests", handlers.CaseRequestsPageHandler)
	}

	// Development-only routes
	if cfg.Environment == "development" {
		devRoutes := e.Group("/dev")
		devRoutes.Use(middleware.RequireAuth())
		devRoutes.Use(middleware.RequireRole("admin"))
		{
			devRoutes.GET("/email/test", handlers.TestEmailHandler)
		}
	}

	// Start background cleanup jobs (runs every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			// Clean up expired sessions
			if err := services.CleanupExpiredSessions(db.DB); err != nil {
				log.Printf("Error cleaning up expired sessions: %v", err)
			}

			// Clean up expired password reset tokens
			if err := services.CleanupExpiredTokens(db.DB); err != nil {
				log.Printf("Error cleaning up expired tokens: %v", err)
			}
		}
	}()

	// Start server
	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := e.Start(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
