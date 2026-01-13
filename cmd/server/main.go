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
	if err := db.AutoMigrate(&models.Firm{}, &models.User{}, &models.Session{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(echomiddleware.RequestLogger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORS())

	// Static files
	e.Static("/static", "static")

	// Public routes (no authentication required)
	e.GET("/", handlers.LandingHandler)
	e.GET("/login", handlers.LoginHandler)
	e.POST("/login", handlers.LoginPostHandler)

	// Firm setup routes (authenticated but no firm required)
	firmSetup := e.Group("/firm")
	firmSetup.Use(middleware.RequireAuth())
	{
		firmSetup.GET("/setup", handlers.FirmSetupHandler)
		firmSetup.POST("/setup", handlers.FirmSetupPostHandler)
	}

	// Protected routes (authentication required)
	protected := e.Group("")
	protected.Use(middleware.RequireAuth())
	{
		protected.GET("/dashboard", handlers.DashboardHandler)
		protected.POST("/logout", handlers.LogoutHandler)
		protected.GET("/api/me", handlers.GetCurrentUserHandler)

		// HTMX routes
		protected.GET("/htmx/users", handlers.GetUsersHTMX)

		// API routes
		protected.GET("/api/users", handlers.GetUsers)
		protected.GET("/api/users/:id", handlers.GetUser)
		protected.POST("/api/users", handlers.CreateUser)
		protected.PUT("/api/users/:id", handlers.UpdateUser)
		protected.DELETE("/api/users/:id", handlers.DeleteUser)
	}

	// Start background session cleanup (runs every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			if err := services.CleanupExpiredSessions(db.DB); err != nil {
				log.Printf("Error cleaning up expired sessions: %v", err)
			}
		}
	}()

	// Start server
	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := e.Start(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
