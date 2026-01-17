package main

import (
	"context"
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/handlers"
	"law_flow_app_go/middleware"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"law_flow_app_go/services/i18n"
	"law_flow_app_go/services/jobs"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize i18n
	if err := i18n.Load(); err != nil {
		log.Fatalf("Failed to load translations: %v", err)
	}

	// Initialize database with environment for logging config
	if err := db.Initialize(cfg.DBPath, cfg.Environment); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.AutoMigrate(&models.Firm{}, &models.User{}, &models.Session{}, &models.PasswordResetToken{}, &models.CaseRequest{}, &models.ChoiceCategory{}, &models.ChoiceOption{}, &models.CaseDomain{}, &models.CaseBranch{}, &models.CaseSubtype{}, &models.Case{}, &models.CaseDocument{}, &models.Availability{}, &models.BlockedDate{}, &models.AppointmentType{}, &models.Appointment{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Seed admin user from environment variables (for Railway deployment)
	if err := services.SeedAdminFromEnv(db.DB); err != nil {
		log.Printf("[WARNING] Failed to seed admin user: %v", err)
	}

	// Check sensitive configuration
	checkSensitiveConfig(cfg)

	// Create Echo instance
	e := echo.New()

	// Configure Debug mode (disable in production)
	e.Debug = cfg.Environment != "production"

	// Middleware
	if cfg.Environment == "production" {
		// JSON logging for production
		e.Use(echomiddleware.LoggerWithConfig(echomiddleware.LoggerConfig{
			Format: `{"time":"${time_rfc3339_nano}","id":"${id}","remote_ip":"${remote_ip}",` +
				`"host":"${host}","method":"${method}","uri":"${uri}","user_agent":"${user_agent}",` +
				`"status":${status},"error":"${error}","latency":${latency},"latency_human":"${latency_human}"` +
				`,"bytes_in":${bytes_in},"bytes_out":${bytes_out}}` + "\n",
		}))
		// Hide sensitive headers from logs
		e.Use(echomiddleware.Secure())
	} else {
		// Development logging (pretty print)
		e.Use(echomiddleware.Logger())
	}
	e.Use(echomiddleware.Recover())

	// Security Middleware
	// Rate Limiting (20 requests/sec per IP)
	e.Use(echomiddleware.RateLimiter(echomiddleware.NewRateLimiterMemoryStore(20)))

	// CORS Configuration
	corsConfig := echomiddleware.DefaultCORSConfig
	if cfg.Environment == "production" {
		corsConfig.AllowOrigins = cfg.AllowedOrigins
	}
	e.Use(echomiddleware.CORSWithConfig(corsConfig))

	// CSRF Protection
	e.Use(echomiddleware.CSRFWithConfig(echomiddleware.CSRFConfig{
		TokenLookup:    "header:X-CSRF-Token,form:_csrf",
		CookieName:     "_csrf",
		CookieSecure:   cfg.Environment == "production",
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
		CookiePath:     "/",
	}))

	// Locale Middleware
	e.Use(middleware.Locale(cfg))

	// Make config available to handlers
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("config", cfg)
			return next(c)
		}
	})

	// Static files
	e.Static("/static", "static")

	// Health check endpoint for load balancers
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

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

	// Public appointment booking routes (no authentication)
	e.GET("/firm/:slug/book", handlers.PublicBookingPageHandler)
	e.GET("/firm/:slug/book/lawyers", handlers.PublicGetLawyersHandler)
	e.GET("/firm/:slug/book/slots", handlers.PublicGetSlotsHandler)
	e.POST("/firm/:slug/book", handlers.PublicSubmitBookingHandler)
	e.GET("/appointment/:token", handlers.PublicAppointmentDetailHandler)
	e.POST("/appointment/:token/cancel", handlers.PublicCancelAppointmentHandler)

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

		// Profile settings (all authenticated users)
		protected.GET("/profile", handlers.ProfileSettingsPageHandler)
		protected.PUT("/api/profile", handlers.UpdateProfileHandler)
		protected.POST("/api/profile/password", handlers.ChangePasswordHandler)

		// User management page (all users can view)
		protected.GET("/users", handlers.UsersPageHandler)

		// User viewing routes (all roles, firm-scoped, with handler-level auth checks)
		protected.GET("/api/users", handlers.GetUsers)
		protected.GET("/api/users/list", handlers.GetUsersListHTMX)
		protected.GET("/api/users/:id", handlers.GetUser)
		protected.GET("/api/users/:id/edit", handlers.GetUserFormEdit)
		protected.PUT("/api/users/:id", handlers.UpdateUser)

		// Admin-only routes
		adminRoutes := protected.Group("")
		adminRoutes.Use(middleware.RequireRole("admin"))
		{
			adminRoutes.GET("/api/users/new", handlers.GetUserFormNew)
			adminRoutes.POST("/api/users", handlers.CreateUser)
			adminRoutes.GET("/api/users/:id/delete-confirm", handlers.GetUserDeleteConfirm)
			adminRoutes.DELETE("/api/users/:id", handlers.DeleteUser)

			// Firm settings (admin only)
			adminRoutes.GET("/firm/settings", handlers.FirmSettingsPageHandler)
			adminRoutes.PUT("/api/firm/settings", handlers.UpdateFirmHandler)
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

		// Case acceptance routes (admin and lawyer only)
		caseAcceptanceRoutes := protected.Group("/api/case-requests/:id/accept")
		caseAcceptanceRoutes.Use(middleware.RequireRole("admin", "lawyer"))
		{
			caseAcceptanceRoutes.GET("/start", handlers.StartCaseAcceptanceHandler)
			caseAcceptanceRoutes.POST("/client", handlers.ProcessClientStepHandler)
			caseAcceptanceRoutes.GET("/lawyers", handlers.GetLawyerListHandler)
			caseAcceptanceRoutes.POST("/lawyer", handlers.AssignLawyerStepHandler)
			caseAcceptanceRoutes.GET("/classification", handlers.GetClassificationOptionsHandler)
			caseAcceptanceRoutes.POST("/classification", handlers.SaveClassificationStepHandler)
			caseAcceptanceRoutes.POST("/finalize", handlers.FinalizeCaseCreationHandler)
			caseAcceptanceRoutes.DELETE("/cancel", handlers.CancelAcceptanceHandler)
		}

		// Case requests dashboard page
		protected.GET("/case-requests", handlers.CaseRequestsPageHandler)

		// Case viewer routes (admin and lawyer only)
		protected.GET("/cases", handlers.CasesPageHandler)
		protected.GET("/cases/:id", handlers.GetCaseDetailHandler)

		caseRoutes := protected.Group("/api/cases")
		caseRoutes.Use(middleware.RequireRole("admin", "lawyer"))
		{
			caseRoutes.GET("", handlers.GetCasesHandler)
			caseRoutes.GET("/:id/edit", handlers.GetCaseEditFormHandler)
			caseRoutes.PUT("/:id", handlers.UpdateCaseHandler)
			caseRoutes.GET("/:id/documents", handlers.GetCaseDocumentsHandler)
			caseRoutes.POST("/:id/documents/upload", handlers.UploadCaseDocumentHandler)
			caseRoutes.GET("/:id/documents/:docId/download", handlers.DownloadCaseDocumentHandler)
			caseRoutes.GET("/:id/documents/:docId/view", handlers.ViewCaseDocumentHandler)
			// Collaborator routes
			caseRoutes.POST("/:id/collaborators", handlers.AddCaseCollaboratorHandler)
			caseRoutes.DELETE("/:id/collaborators/:userId", handlers.RemoveCaseCollaboratorHandler)
			caseRoutes.GET("/:id/collaborators/available", handlers.GetAvailableCollaboratorsHandler)
			// Historical case routes
			caseRoutes.GET("/history/new", handlers.GetHistoricalCaseFormHandler)
			caseRoutes.POST("/history", handlers.CreateHistoricalCaseHandler)
			caseRoutes.GET("/history/branches", handlers.GetHistoricalCaseBranchesHandler)
			caseRoutes.GET("/history/subtypes", handlers.GetHistoricalCaseSubtypesHandler)
		}

		// Historical Cases page
		protected.GET("/historical-cases", handlers.HistoricalCasesPageHandler)

		// Lawyer filter route (admin only) - add to adminRoutes
		adminRoutes.GET("/api/lawyers", handlers.GetLawyersForFilterHandler)

		// Availability routes (lawyer and admin only)
		availabilityRoutes := protected.Group("")
		availabilityRoutes.Use(middleware.RequireRole("admin", "lawyer"))
		{
			availabilityRoutes.GET("/availability", handlers.AvailabilityPageHandler)
			availabilityRoutes.GET("/api/availability", handlers.GetAvailabilityHandler)
			availabilityRoutes.POST("/api/availability", handlers.CreateAvailabilityHandler)
			availabilityRoutes.POST("/api/availability/validate", handlers.CheckOverlapHandler)
			availabilityRoutes.PUT("/api/availability/:id", handlers.UpdateAvailabilityHandler)
			availabilityRoutes.DELETE("/api/availability/:id", handlers.DeleteAvailabilityHandler)
			availabilityRoutes.GET("/api/blocked-dates", handlers.GetBlockedDatesHandler)
			availabilityRoutes.POST("/api/blocked-dates", handlers.CreateBlockedDateHandler)
			availabilityRoutes.POST("/api/blocked-dates/validate", handlers.CheckBlockedDateOverlapHandler)
			availabilityRoutes.DELETE("/api/blocked-dates/:id", handlers.DeleteBlockedDateHandler)
		}

		// Buffer settings (admin only)
		adminRoutes.PUT("/api/firm/buffer-settings", handlers.UpdateBufferSettingsHandler)

		// Calendar View
		protected.GET("/calendar", handlers.CalendarPageHandler)
		protected.GET("/api/calendar/events", handlers.CalendarEventsHandler)

		// Appointments page (lawyer and admin only)
		protected.GET("/appointments", handlers.AppointmentsPageHandler)

		// Appointment routes (lawyer and admin only)
		appointmentRoutes := protected.Group("/api/appointments")
		appointmentRoutes.Use(middleware.RequireRole("admin", "lawyer"))
		{
			appointmentRoutes.GET("", handlers.GetAppointmentsHandler)
			appointmentRoutes.GET("/slots", handlers.GetAvailableSlotsHandler)
			appointmentRoutes.GET("/clients", handlers.GetClientsForAppointmentHandler)
			appointmentRoutes.GET("/lawyers", handlers.GetLawyersForAppointmentHandler)
			appointmentRoutes.GET("/types", handlers.GetActiveAppointmentTypesHandler)
			appointmentRoutes.POST("", handlers.CreateAppointmentHandler)
			appointmentRoutes.GET("/:id", handlers.GetAppointmentHandler)
			appointmentRoutes.PUT("/:id/status", handlers.UpdateAppointmentStatusHandler)
			appointmentRoutes.PUT("/:id/reschedule", handlers.RescheduleAppointmentHandler)
			appointmentRoutes.DELETE("/:id", handlers.CancelAppointmentHandler)
		}

		// Appointment Type management (admin only)
		appointmentTypeRoutes := adminRoutes.Group("/appointment-types")
		{
			appointmentTypeRoutes.GET("", handlers.GetAppointmentTypesHandler)
			appointmentTypeRoutes.POST("", handlers.CreateAppointmentTypeHandler)
			appointmentTypeRoutes.PUT("/:id", handlers.UpdateAppointmentTypeHandler)
			appointmentTypeRoutes.DELETE("/:id", handlers.DeleteAppointmentTypeHandler)
		}
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

	// Start background jobs
	go func() {
		// Run immediately on startup (for demo/testing)
		jobs.SendAppointmentReminders(cfg)

		// Then run every hour
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			jobs.SendAppointmentReminders(cfg)
		}
	}()

	// Graceful shutdown setup
	go func() {
		if err := e.Start(":" + cfg.ServerPort); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("Shutting down the server")
		}
	}()

	// Wait for interrupt signal (SIGINT or SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	log.Println("Server gracefully stopped")
}

// checkSensitiveConfig performs startup security checks
func checkSensitiveConfig(cfg *config.Config) {
	if cfg.Environment == "production" {
		if cfg.SMTPPassword == "" {
			log.Println("[WARNING] SMTP_PASSWORD is not set in production!")
		}
		if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*" {
			log.Println("[WARNING] ALLOWED_ORIGINS is set to '*' in production! This is insecure.")
		}
		if cfg.ServerPort == "8080" {
			log.Println("[INFO] Running on default port 8080 in production. Ensure this is intended.")
		}
		if cfg.SessionSecret == "" {
			log.Println("[WARNING] SESSION_SECRET is not set in production!")
		}
	}
}
