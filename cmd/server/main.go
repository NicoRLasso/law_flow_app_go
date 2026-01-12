package main

import (
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/handlers"
	"law_flow_app_go/models"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Static files
	e.Static("/static", "static")

	// Routes
	// Web routes
	e.GET("/", handlers.LandingHandler)  // Landing page is now the home page
	e.GET("/demo", handlers.HomeHandler) // Old home page available at /demo for reference

	// HTMX routes
	htmx := e.Group("/htmx")
	{
		htmx.GET("/users", handlers.GetUsersHTMX)
	}

	// API routes
	api := e.Group("/api")
	{
		// User endpoints
		api.GET("/users", handlers.GetUsers)
		api.GET("/users/:id", handlers.GetUser)
		api.POST("/users", handlers.CreateUser)
		api.PUT("/users/:id", handlers.UpdateUser)
		api.DELETE("/users/:id", handlers.DeleteUser)
	}

	// Start server
	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := e.Start(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
