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
	"log/slog"
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
	if err := db.InitializeWithConfig(db.DatabaseConfig{
		DBPath:           cfg.DBPath,
		Environment:      cfg.Environment,
		TursoDatabaseURL: cfg.TursoDatabaseURL,
		TursoAuthToken:   cfg.TursoAuthToken,
	}); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.AutoMigrate(&models.Firm{}, &models.User{}, &models.Session{}, &models.PasswordResetToken{}, &models.CaseRequest{}, &models.ChoiceCategory{}, &models.ChoiceOption{}, &models.CaseDomain{}, &models.CaseBranch{}, &models.CaseSubtype{}, &models.Case{}, &models.CaseParty{}, &models.CaseDocument{}, &models.CaseLog{}, &models.Availability{}, &models.BlockedDate{}, &models.AppointmentType{}, &models.Appointment{}, &models.AuditLog{}, &models.TemplateCategory{}, &models.DocumentTemplate{}, &models.GeneratedDocument{}, &models.SupportTicket{}, &models.JudicialProcess{}, &models.JudicialProcessAction{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize FTS5 search index
	if err := services.InitializeFTS5(db.DB); err != nil {
		log.Printf("[WARNING] Failed to initialize FTS5: %v", err)
	}

	// Migrate existing data to FTS5 index if needed
	if err := services.MigrateFTSData(db.DB); err != nil {
		log.Printf("[WARNING] Failed to migrate FTS5 data: %v", err)
	}

	// Initialize search service
	handlers.InitSearchService()

	// Seed superadmin user from environment variables
	if err := services.SeedSuperadminFromEnv(db.DB); err != nil {
		log.Printf("[WARNING] Failed to seed superadmin user: %v", err)
	}

	// Initialize storage (R2 or local filesystem)
	services.InitializeStorage(cfg)

	// Check sensitive configuration
	checkSensitiveConfig(cfg)

	// Create Echo instance
	e := echo.New()

	// Configure Debug mode (disable in production)
	e.Debug = cfg.Environment != "production"
	e.HideBanner = true

	// Middleware
	// Body Limit (2MB) - prevent large payloads
	e.Use(echomiddleware.BodyLimit("2M"))
	// Gzip Compression
	e.Use(echomiddleware.Gzip())

	if cfg.Environment == "production" {
		// Initialize Slog with JSON handler for production
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger)

		// Request Logger (Structured Logging)
		e.Use(echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
			LogStatus:        true,
			LogURI:           true,
			LogError:         true,
			HandleError:      true,
			LogRequestID:     true,
			LogRemoteIP:      true,
			LogHost:          true,
			LogMethod:        true,
			LogUserAgent:     true,
			LogLatency:       true,
			LogContentLength: true,
			LogResponseSize:  true,
			LogValuesFunc: func(c echo.Context, v echomiddleware.RequestLoggerValues) error {
				// Log using global slog
				attrs := []any{
					slog.String("id", v.RequestID),
					slog.String("remote_ip", v.RemoteIP),
					slog.String("host", v.Host),
					slog.String("method", v.Method),
					slog.String("uri", v.URI),
					slog.String("user_agent", v.UserAgent),
					slog.Int("status", v.Status),
					slog.Duration("latency", v.Latency),
					slog.String("latency_human", v.Latency.String()),
					slog.String("bytes_in", v.ContentLength),
					slog.Int64("bytes_out", v.ResponseSize),
				}
				if v.Error != nil {
					attrs = append(attrs, slog.String("error", v.Error.Error()))
				}
				slog.Info("request", attrs...)
				return nil
			},
		}))

		// CSP Nonce Generation (Must be before Secure middleware if Secure sets CSP, but here we set it manually)
		e.Use(middleware.CSPNonce())

		// Security Headers (CSP handled by middleware.CSPNonce)
		e.Use(echomiddleware.SecureWithConfig(echomiddleware.SecureConfig{
			XSSProtection:         "1; mode=block",
			ContentTypeNosniff:    "nosniff",
			XFrameOptions:         "SAMEORIGIN",
			HSTSMaxAge:            31536000,
			HSTSExcludeSubdomains: true,
		}))
	} else {
		// Development logging (Structured Text)
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		slog.SetDefault(logger)

		e.Use(echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
			LogStatus:   true,
			LogURI:      true,
			LogError:    true,
			HandleError: true,
			LogMethod:   true,
			LogLatency:  true,
			LogValuesFunc: func(c echo.Context, v echomiddleware.RequestLoggerValues) error {
				// Simple text logging for dev
				args := []any{
					slog.String("method", v.Method),
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("latency", v.Latency.String()),
				}
				if v.Error != nil {
					args = append(args, slog.String("error", v.Error.Error()))
					slog.Error("request", args...)
				} else {
					slog.Info("request", args...)
				}
				return nil
			},
		}))
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
	// Use a group to apply middleware specifically for static files
	staticGroup := e.Group("/static")
	staticGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Set Cache-Control headers before calling the next handler
			if cfg.Environment == "production" {
				c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				// 1 hour for development to help with testing but not be too sticky
				c.Response().Header().Set("Cache-Control", "public, max-age=3600")
			}
			return next(c)
		}
	})
	staticGroup.Static("/", "static")

	// Health check endpoint for load balancers
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

	// SEO Files with caching
	seoCacheMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if cfg.Environment == "production" {
				// SEO files can change, so we use a shorter cache (1 day)
				c.Response().Header().Set("Cache-Control", "public, max-age=86400")
			}
			return next(c)
		}
	}
	e.File("/robots.txt", "static/robots.txt", seoCacheMiddleware)
	// Dynamic sitemap
	e.GET("/sitemap.xml", handlers.GetSitemapHandler)

	// Public routes (no authentication required)
	e.GET("/", handlers.LandingHandler)
	e.GET("/login", handlers.LoginHandler)
	e.POST("/login", handlers.LoginPostHandler)
	e.GET("/forgot-password", handlers.ForgotPasswordHandler)
	e.POST("/forgot-password", handlers.ForgotPasswordPostHandler)
	e.GET("/reset-password", handlers.ResetPasswordHandler)
	e.POST("/reset-password", handlers.ResetPasswordPostHandler)

	// Website Static Pages (Footer)
	e.GET("/about", handlers.WebsiteAboutHandler)
	e.GET("/contact", handlers.WebsiteContactHandler)
	e.GET("/security", handlers.WebsiteSecurityHandler)
	e.GET("/privacy", handlers.WebsitePrivacyHandler)
	e.GET("/terms", handlers.WebsiteTermsHandler)
	e.GET("/cookies", handlers.WebsiteCookiesHandler)
	e.GET("/compliance", handlers.WebsiteComplianceHandler)
	e.POST("/api/website/contact", handlers.WebsiteContactSubmitHandler)

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

	// Logout route (authenticated, no firm required)
	e.POST("/logout", middleware.RequireAuth()(handlers.LogoutHandler))

	// Superadmin routes (authenticated, superadmin role, no firm required)
	superadminRoutes := e.Group("/superadmin")
	superadminRoutes.Use(middleware.RequireAuth())
	superadminRoutes.Use(middleware.RequireSuperadmin())
	{
		// Dashboard
		superadminRoutes.GET("", func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, "/superadmin/dashboard")
		})
		superadminRoutes.GET("/dashboard", handlers.SuperadminDashboardHandler)

		// User Management
		superadminRoutes.GET("/users", handlers.SuperadminUsersPageHandler)
		superadminRoutes.GET("/users/list", handlers.SuperadminGetUsersListHTMX)
		superadminRoutes.GET("/users/new", handlers.SuperadminGetUserFormNew)
		superadminRoutes.POST("/users", handlers.SuperadminCreateUserHandler)
		superadminRoutes.GET("/users/:id/edit", handlers.SuperadminGetUserFormEdit)
		superadminRoutes.PUT("/users/:id", handlers.SuperadminUpdateUser)
		superadminRoutes.PATCH("/users/:id/toggle-active", handlers.SuperadminToggleUserActive)
		superadminRoutes.GET("/users/:id/delete-confirm", handlers.SuperadminGetUserDeleteConfirm)
		superadminRoutes.DELETE("/users/:id", handlers.SuperadminDeleteUser)

		// Firm Management
		superadminRoutes.GET("/firms", handlers.SuperadminFirmsPageHandler)
		superadminRoutes.GET("/firms/list", handlers.SuperadminGetFirmsListHTMX)
		superadminRoutes.GET("/firms/new", handlers.SuperadminGetFirmFormNew)
		superadminRoutes.POST("/firms", handlers.SuperadminCreateFirmHandler)
		superadminRoutes.GET("/firms/:id/edit", handlers.SuperadminGetFirmFormEdit)
		superadminRoutes.PUT("/firms/:id", handlers.SuperadminUpdateFirm)
		superadminRoutes.PATCH("/firms/:id/toggle-active", handlers.SuperadminToggleFirmActive)
		superadminRoutes.GET("/firms/:id/delete-confirm", handlers.SuperadminGetFirmDeleteConfirm)
		// Support Ticket Management
		superadminRoutes.GET("/support", handlers.SuperadminSupportPageHandler)
		superadminRoutes.GET("/support/:id", handlers.SuperadminSupportDetailHandler)
		superadminRoutes.POST("/support/:id/status", handlers.SuperadminUpdateTicketStatusHandler)
		superadminRoutes.POST("/support/:id/reply", handlers.SuperadminReplyTicketHandler)
		superadminRoutes.POST("/support/:id/take", handlers.SuperadminTakeTicketHandler)
	}

	// Protected routes (authentication + firm required)
	protected := e.Group("")
	protected.Use(middleware.RequireAuth())
	protected.Use(middleware.RequireFirm()) // Ensure user has a firm
	protected.Use(middleware.AuditContext())
	{
		// All users with a firm can access dashboard and their own profile
		protected.GET("/dashboard", handlers.DashboardHandler)
		protected.GET("/api/me", handlers.GetCurrentUserHandler)

		// Profile settings (all authenticated users)
		protected.GET("/profile", handlers.ProfileSettingsPageHandler)
		protected.PUT("/api/profile", handlers.UpdateProfileHandler)
		protected.POST("/api/profile/password", handlers.ChangePasswordHandler)

		// Support Page
		protected.GET("/support", handlers.SupportPageHandler)
		protected.GET("/api/support/tickets", handlers.GetSupportTicketsHandler)
		protected.POST("/api/support/contact", handlers.SubmitSupportRequestHandler)

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
			adminRoutes.POST("/api/firm/logo", handlers.UploadFirmLogoHandler)
			adminRoutes.DELETE("/api/firm/logo", handlers.DeleteFirmLogoHandler)

			// Audit Logs (admin only)
			adminRoutes.GET("/audit-logs", handlers.AuditLogsPageHandler)
			adminRoutes.GET("/api/audit-logs", handlers.GetAuditLogsHandler)
			adminRoutes.GET("/api/audit-logs/:type/:id", handlers.GetResourceHistoryHandler)

			// Classification Subtypes (admin only) - Management
			adminRoutes.GET("/api/subtypes", handlers.GetSubtypesTabHandler)
			adminRoutes.GET("/api/subtypes/list", handlers.GetSubtypesForBranchHandler)
			adminRoutes.GET("/api/subtypes/checkboxes", handlers.GetSubtypeCheckboxesHandler)
			adminRoutes.GET("/api/subtypes/new", handlers.GetSubtypeFormHandler)
			adminRoutes.GET("/api/subtypes/:id/view", handlers.GetSubtypeViewHandler)
			adminRoutes.GET("/api/subtypes/:id/edit", handlers.GetSubtypeFormHandler)
			adminRoutes.POST("/api/subtypes", handlers.CreateSubtypeHandler)
			adminRoutes.PUT("/api/subtypes/:id", handlers.UpdateSubtypeHandler)
			adminRoutes.PATCH("/api/subtypes/:id/toggle-active", handlers.ToggleSubtypeActiveHandler)
			adminRoutes.DELETE("/api/subtypes/:id", handlers.DeleteSubtypeHandler)
		}

		// Document Template Management (Admin + Lawyer)
		templateRoutes := protected.Group("/templates")
		templateRoutes.Use(middleware.RequireRole("admin", "lawyer"))
		{
			// Page routes
			templateRoutes.GET("", handlers.TemplatesPageHandler)
			templateRoutes.GET("/new", handlers.TemplateEditorPageHandler)
			templateRoutes.GET("/:id/edit", handlers.TemplateEditorPageHandler)
		}

		templateApiRoutes := protected.Group("/api/templates")
		templateApiRoutes.Use(middleware.RequireRole("admin", "lawyer"))
		{
			templateApiRoutes.GET("", handlers.GetTemplatesHandler)
			templateApiRoutes.POST("", handlers.CreateTemplateHandler)
			templateApiRoutes.PUT("/:id", handlers.UpdateTemplateHandler)
			templateApiRoutes.DELETE("/:id", handlers.DeleteTemplateHandler)
			templateApiRoutes.GET("/:id/metadata", handlers.GetTemplateMetadataHandler)
			templateApiRoutes.GET("/:id/metadata/modal", handlers.GetTemplateMetadataModalHandler)
			templateApiRoutes.GET("/:id/clone/modal", handlers.GetCloneTemplateModalHandler)
			templateApiRoutes.POST("/:id/clone", handlers.CloneTemplateHandler)
			templateApiRoutes.GET("/variables", handlers.GetTemplateVariablesHandler)

			// Template Categories
			templateApiRoutes.GET("/categories", handlers.GetCategoriesHandler)
			templateApiRoutes.POST("/categories", handlers.CreateCategoryHandler)
			templateApiRoutes.PUT("/categories/:id", handlers.UpdateCategoryHandler)
			templateApiRoutes.DELETE("/categories/:id", handlers.DeleteCategoryHandler)
		}

		// Classification Options (accessible to all authenticated users with firm)
		protected.GET("/api/subtypes/branches", handlers.GetBranchesForDomainHandler)
		protected.GET("/api/subtypes/options", handlers.GetSubtypeOptionsHandler)

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

		// Case Routes Configuration

		// Search API (Admin, Lawyer, Staff - not clients)
		searchRoutes := protected.Group("/api")
		searchRoutes.Use(middleware.RequireRole("admin", "lawyer", "staff"))
		{
			searchRoutes.GET("/search", handlers.SearchCasesHandler)
		}

		// 1. Client Accessible Case Routes (Admin, Lawyer, Client) - STRICTLY for viewing list and documents
		clientCaseRoutes := protected.Group("/api/cases")
		clientCaseRoutes.Use(middleware.RequireRole("admin", "lawyer", "client"))
		{
			clientCaseRoutes.GET("", handlers.GetCasesHandler)

			// Document routes (Audit logs in handlers ensure strict permission checks)
			clientCaseRoutes.GET("/:id/documents", handlers.GetCaseDocumentsHandler)
			clientCaseRoutes.POST("/:id/documents/upload", handlers.UploadCaseDocumentHandler)
			clientCaseRoutes.GET("/:id/documents/:docId/download", handlers.DownloadCaseDocumentHandler)
			clientCaseRoutes.GET("/:id/documents/:docId/view", handlers.ViewCaseDocumentHandler)

			// Judicial Process View
			clientCaseRoutes.GET("/:id/judicial-view", handlers.GetJudicialProcessViewHandler)
		}

		// Client Case Request Routes (Authenticated Clients)
		clientRequestRoutes := protected.Group("/api/client")
		clientRequestRoutes.Use(middleware.RequireRole("client"))
		{
			clientRequestRoutes.GET("/case-request", handlers.ClientCaseRequestHandler)
			clientRequestRoutes.POST("/case-request", handlers.ClientSubmitCaseRequestHandler)
			clientRequestRoutes.GET("/requests", handlers.ClientRequestsPageHandler)
		}

		// 2. Restricted Case Routes (Admin, Lawyer ONLY) - Management and editing
		caseRoutes := protected.Group("/api/cases")
		caseRoutes.Use(middleware.RequireRole("admin", "lawyer"))
		{
			// Note: GET "" and document viewing routes are in clientCaseRoutes above

			caseRoutes.GET("/new", handlers.CreateCaseModalHandler)
			caseRoutes.POST("", handlers.CreateCaseHandler)
			caseRoutes.GET("/:id/edit", handlers.GetCaseEditFormHandler)
			caseRoutes.PUT("/:id", handlers.UpdateCaseHandler)
			caseRoutes.PATCH("/:id/documents/:docId/visibility", handlers.ToggleDocumentVisibilityHandler)
			caseRoutes.DELETE("/:id/documents/:docId", handlers.DeleteCaseDocumentHandler)
			// Collaborator routes
			caseRoutes.POST("/:id/collaborators", handlers.AddCaseCollaboratorHandler)
			caseRoutes.DELETE("/:id/collaborators/:userId", handlers.RemoveCaseCollaboratorHandler)
			caseRoutes.GET("/:id/collaborators/available", handlers.GetAvailableCollaboratorsHandler)
			// Import routes
			caseRoutes.GET("/import/modal", handlers.ImportCasesModalHandler)
			caseRoutes.GET("/import/template", handlers.GetImportTemplateHandler)
			caseRoutes.POST("/import", handlers.ImportCasesHandler)
			// Opposing Party routes
			caseRoutes.GET("/:id/party/modal", handlers.GetCasePartyModalHandler)
			caseRoutes.POST("/:id/party", handlers.AddCasePartyHandler)
			caseRoutes.PUT("/:id/party", handlers.UpdateCasePartyHandler)
			caseRoutes.DELETE("/:id/party", handlers.DeleteCasePartyHandler)
			// Case Log routes
			caseRoutes.GET("/:id/logs", handlers.GetCaseLogsHandler)
			caseRoutes.GET("/:id/logs/new", handlers.GetCaseLogFormHandler)
			caseRoutes.POST("/:id/logs", handlers.CreateCaseLogHandler)
			caseRoutes.GET("/:id/logs/:logId", handlers.GetCaseLogHandler)
			caseRoutes.GET("/:id/logs/:logId/view", handlers.GetCaseLogViewHandler)
			caseRoutes.PUT("/:id/logs/:logId", handlers.UpdateCaseLogHandler)
			caseRoutes.DELETE("/:id/logs/:logId", handlers.DeleteCaseLogHandler)
			// Document Generation routes
			caseRoutes.GET("/:id/generate", handlers.GetGenerateDocumentTabHandler)
			caseRoutes.GET("/:id/generate/preview", handlers.PreviewTemplateHandler)
			caseRoutes.POST("/:id/generate", handlers.GenerateDocumentHandler)
			caseRoutes.GET("/:id/generated", handlers.GetGeneratedDocumentsHandler)
			caseRoutes.GET("/:id/generated/:docId/download", handlers.DownloadGeneratedDocumentHandler)
			caseRoutes.GET("/:id/templates/modal", handlers.GetTemplateSelectorModalHandler)
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
			appointmentRoutes.GET("/cases", handlers.GetCasesForAppointmentHandler) // New route
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
		// Run appointment reminders immediately on startup (for demo/testing)
		jobs.SendAppointmentReminders(cfg)

		// Start Judicial Update Job
		jobs.StartJudicialUpdateJob()

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
		if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*" {
			log.Println("[WARNING] ALLOWED_ORIGINS is set to '*' in production! This is insecure.")
		}
		if cfg.ServerPort == "8080" {
			log.Println("[INFO] Running on default port 8080 in production. Ensure this is intended.")
		}
		if cfg.SessionSecret == "" {
			log.Fatal("[CRITICAL] SESSION_SECRET is not set in production! App cannot start securely.")
		}
	}
}
