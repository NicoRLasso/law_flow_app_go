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
	cfg := config.Load()
	if err := i18n.Load(); err != nil {
		log.Fatalf("Failed to load translations: %v", err)
	}
	if err := db.InitializeWithConfig(db.DatabaseConfig{
		DBPath:           cfg.DBPath,
		Environment:      cfg.Environment,
		TursoDatabaseURL: cfg.TursoDatabaseURL,
		TursoAuthToken:   cfg.TursoAuthToken,
	}); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	if err := db.AutoMigrate(&models.Firm{}, &models.User{}, &models.Session{}, &models.PasswordResetToken{}, &models.ChoiceCategory{}, &models.ChoiceOption{}, &models.CaseDomain{}, &models.CaseBranch{}, &models.CaseSubtype{}, &models.Case{}, &models.CaseParty{}, &models.CaseDocument{}, &models.CaseLog{}, &models.Availability{}, &models.BlockedDate{}, &models.AppointmentType{}, &models.Appointment{}, &models.AuditLog{}, &models.TemplateCategory{}, &models.DocumentTemplate{}, &models.GeneratedDocument{}, &models.SupportTicket{}, &models.JudicialProcess{}, &models.JudicialProcessAction{}, &models.Plan{}, &models.FirmSubscription{}, &models.FirmUsage{}, &models.PlanAddOn{}, &models.FirmAddOn{}, &models.LegalService{}, &models.ServiceMilestone{}, &models.ServiceDocument{}, &models.ServiceExpense{}, &models.ServiceActivity{}, &models.Notification{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	if err := services.InitializeFTS5(db.DB); err != nil {
		log.Printf("[WARNING] Failed to initialize FTS5: %v", err)
	}
	if err := services.MigrateFTSData(db.DB); err != nil {
		log.Printf("[WARNING] Failed to migrate FTS5 data: %v", err)
	}
	handlers.InitSearchService()
	if err := services.SeedSuperadminFromEnv(db.DB); err != nil {
		log.Printf("[WARNING] Failed to seed superadmin user: %v", err)
	}
	if err := services.InitializeSubscriptionSystem(db.DB); err != nil {
		log.Printf("[WARNING] Failed to initialize subscription system: %v", err)
	}
	services.InitializeStorage(cfg)
	middleware.InitAssetVersions()
	checkSensitiveConfig(cfg)
	e := echo.New()
	e.Debug = cfg.Environment != "production"
	e.HideBanner = true
	e.Use(echomiddleware.BodyLimit("2M"))
	e.Use(echomiddleware.Gzip())
	if cfg.Environment == "production" {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger)
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
		e.Use(middleware.CSPNonce(cfg.Environment != "production"))
		e.Use(echomiddleware.SecureWithConfig(echomiddleware.SecureConfig{
			XSSProtection:         "1; mode=block",
			ContentTypeNosniff:    "nosniff",
			XFrameOptions:         "SAMEORIGIN",
			HSTSMaxAge:            31536000,
			HSTSExcludeSubdomains: true,
		}))
	} else {
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
	e.Use(echomiddleware.RateLimiter(echomiddleware.NewRateLimiterMemoryStore(20)))
	corsConfig := echomiddleware.DefaultCORSConfig
	if cfg.Environment == "production" {
		corsConfig.AllowOrigins = cfg.AllowedOrigins
	}
	e.Use(echomiddleware.CORSWithConfig(corsConfig))
	e.Use(echomiddleware.CSRFWithConfig(echomiddleware.CSRFConfig{
		TokenLookup:    "header:X-CSRF-Token,form:_csrf",
		CookieName:     "_csrf",
		CookieSecure:   cfg.Environment == "production",
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
		CookiePath:     "/",
	}))
	e.Use(middleware.Locale(cfg))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("config", cfg)
			return next(c)
		}
	})
	staticGroup := e.Group("/static")
	staticGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if cfg.Environment == "production" {
				c.Response().Header().Set("Cache-Control", "public, max-age=2592000")
			} else {
				c.Response().Header().Set("Cache-Control", "public, max-age=3600")
			}
			return next(c)
		}
	})
	staticGroup.Static("/", "static")
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

	seoCacheMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if cfg.Environment == "production" {
				c.Response().Header().Set("Cache-Control", "public, max-age=86400")
			}
			return next(c)
		}
	}
	e.File("/robots.txt", "static/robots.txt", seoCacheMiddleware)
	e.GET("/sitemap.xml", handlers.GetSitemapHandler)

	e.GET("/", handlers.LandingHandler)
	e.GET("/login", handlers.LoginHandler)
	e.POST("/login", handlers.LoginPostHandler, middleware.LoginRateLimiter.Middleware())
	e.GET("/forgot-password", handlers.ForgotPasswordHandler)
	e.POST("/forgot-password", handlers.ForgotPasswordPostHandler, middleware.PasswordResetRateLimiter.Middleware())
	e.GET("/reset-password", handlers.ResetPasswordHandler)
	e.POST("/reset-password", handlers.ResetPasswordPostHandler, middleware.PasswordResetRateLimiter.Middleware())
	e.GET("/about", handlers.WebsiteAboutHandler)
	e.GET("/contact", handlers.WebsiteContactHandler)
	e.GET("/security", handlers.WebsiteSecurityHandler)
	e.GET("/privacy", handlers.WebsitePrivacyHandler)
	e.GET("/terms", handlers.WebsiteTermsHandler)
	e.GET("/cookies", handlers.WebsiteCookiesHandler)
	e.GET("/compliance", handlers.WebsiteComplianceHandler)
	e.POST("/api/website/contact", handlers.WebsiteContactSubmitHandler, middleware.PublicFormRateLimiter.Middleware())

	firmSetup := e.Group("/firm")
	firmSetup.Use(middleware.RequireAuth())
	{
		firmSetup.GET("/setup", handlers.FirmSetupHandler)
		firmSetup.POST("/setup", handlers.FirmSetupPostHandler)
	}
	e.POST("/logout", middleware.RequireAuth()(handlers.LogoutHandler))
	superadminRoutes := e.Group("/superadmin")
	superadminRoutes.Use(middleware.RequireAuth())
	superadminRoutes.Use(middleware.RequireSuperadmin())
	{
		superadminRoutes.GET("", func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, "/superadmin/dashboard")
		})
		superadminRoutes.GET("/dashboard", handlers.SuperadminDashboardHandler)
		superadminRoutes.GET("/users", handlers.SuperadminUsersPageHandler)
		superadminRoutes.GET("/users/list", handlers.SuperadminGetUsersListHTMX)
		superadminRoutes.GET("/users/new", handlers.SuperadminGetUserFormNew)
		superadminRoutes.POST("/users", handlers.SuperadminCreateUserHandler)
		superadminRoutes.GET("/users/:id/edit", handlers.SuperadminGetUserFormEdit)
		superadminRoutes.PUT("/users/:id", handlers.SuperadminUpdateUser)
		superadminRoutes.PATCH("/users/:id/toggle-active", handlers.SuperadminToggleUserActive)
		superadminRoutes.GET("/users/:id/delete-confirm", handlers.SuperadminGetUserDeleteConfirm)
		superadminRoutes.DELETE("/users/:id", handlers.SuperadminDeleteUser)
		superadminRoutes.GET("/firms", handlers.SuperadminFirmsPageHandler)
		superadminRoutes.GET("/firms/list", handlers.SuperadminGetFirmsListHTMX)
		superadminRoutes.GET("/firms/new", handlers.SuperadminGetFirmFormNew)
		superadminRoutes.POST("/firms", handlers.SuperadminCreateFirmHandler)
		superadminRoutes.GET("/firms/:id/edit", handlers.SuperadminGetFirmFormEdit)
		superadminRoutes.PUT("/firms/:id", handlers.SuperadminUpdateFirm)
		superadminRoutes.PATCH("/firms/:id/toggle-active", handlers.SuperadminToggleFirmActive)
		superadminRoutes.GET("/firms/:id/delete-confirm", handlers.SuperadminGetFirmDeleteConfirm)
		superadminRoutes.GET("/support", handlers.SuperadminSupportPageHandler)
		superadminRoutes.GET("/support/:id", handlers.SuperadminSupportDetailHandler)
		superadminRoutes.POST("/support/:id/status", handlers.SuperadminUpdateTicketStatusHandler)
		superadminRoutes.POST("/support/:id/reply", handlers.SuperadminReplyTicketHandler)
		superadminRoutes.POST("/support/:id/take", handlers.SuperadminTakeTicketHandler)
		superadminRoutes.GET("/plans", handlers.SuperadminPlansPageHandler)
		superadminRoutes.GET("/addons", handlers.SuperadminAddOnsPageHandler)
		superadminRoutes.PUT("/firms/:id/subscription", handlers.SuperadminUpdateFirmSubscriptionHandler)
		superadminRoutes.GET("/firms/:id/subscription", handlers.SuperadminGetFirmSubscriptionForm)
		superadminRoutes.PATCH("/addons/:id/toggle-active", handlers.SuperadminToggleAddOnActiveHandler)
	}
	protected := e.Group("")
	protected.Use(middleware.RequireAuth())
	protected.Use(middleware.RequireFirm())
	protected.Use(middleware.AuditContext())
	{
		protected.GET("/dashboard", handlers.DashboardHandler)
		protected.GET("/api/notifications", handlers.GetNotificationsHandler)
		protected.PATCH("/api/notifications/:id/read", handlers.MarkNotificationReadHandler)
		protected.PATCH("/api/notifications/read-all", handlers.MarkAllNotificationsReadHandler)
		protected.GET("/api/me", handlers.GetCurrentUserHandler)
		protected.GET("/profile", handlers.ProfileSettingsPageHandler)
		protected.PUT("/api/profile", handlers.UpdateProfileHandler)
		protected.POST("/api/profile/password", handlers.ChangePasswordHandler)
		protected.GET("/support", handlers.SupportPageHandler)
		protected.GET("/api/support/tickets", handlers.GetSupportTicketsHandler)
		protected.POST("/api/support/contact", handlers.SubmitSupportRequestHandler)
		protected.GET("/users", handlers.UsersPageHandler)
		protected.GET("/users", handlers.UsersPageHandler)
		protected.GET("/api/users", handlers.GetUsers)
		protected.GET("/api/users/list", handlers.GetUsersListHTMX)
		protected.GET("/api/users/:id", handlers.GetUser)
		protected.GET("/api/users/:id/edit", handlers.GetUserFormEdit)
		protected.PUT("/api/users/:id", handlers.UpdateUser)
		adminRoutes := protected.Group("")
		adminRoutes.Use(middleware.RequireRole("admin"))
		{
			adminRoutes.GET("/api/users/new", handlers.GetUserFormNew)
			adminRoutes.POST("/api/users", handlers.CreateUser)
			adminRoutes.GET("/api/users/:id/delete-confirm", handlers.GetUserDeleteConfirm)
			adminRoutes.DELETE("/api/users/:id", handlers.DeleteUser)
			adminRoutes.GET("/firm/settings", handlers.FirmSettingsPageHandler)
			adminRoutes.PUT("/api/firm/settings", handlers.UpdateFirmHandler)
			adminRoutes.POST("/api/firm/logo", handlers.UploadFirmLogoHandler)
			adminRoutes.DELETE("/api/firm/logo", handlers.DeleteFirmLogoHandler)
			adminRoutes.GET("/api/firm/settings/billing", handlers.FirmBillingTabHandler)
			adminRoutes.POST("/api/addons/purchase", handlers.PurchaseAddOnHandler)
			adminRoutes.DELETE("/api/addons/:id", handlers.CancelAddOnHandler)
			adminRoutes.GET("/audit-logs", handlers.AuditLogsPageHandler)
			adminRoutes.GET("/api/audit-logs", handlers.GetAuditLogsHandler)
			adminRoutes.GET("/api/audit-logs/:type/:id", handlers.GetResourceHistoryHandler)
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

		templateRoutes := protected.Group("/templates")
		templateRoutes.Use(middleware.RequireRole("admin", "lawyer"))
		{
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
			templateApiRoutes.GET("/categories", handlers.GetCategoriesHandler)
			templateApiRoutes.POST("/categories", handlers.CreateCategoryHandler)
			templateApiRoutes.PUT("/categories/:id", handlers.UpdateCategoryHandler)
			templateApiRoutes.DELETE("/categories/:id", handlers.DeleteCategoryHandler)
		}

		protected.GET("/api/subtypes/branches", handlers.GetBranchesForDomainHandler)
		protected.GET("/api/subtypes/options", handlers.GetSubtypeOptionsHandler)

		protected.GET("/cases", handlers.CasesPageHandler)
		protected.GET("/cases/:id", handlers.GetCaseDetailHandler)

		searchRoutes := protected.Group("/api")
		searchRoutes.Use(middleware.RequireRole("admin", "lawyer", "staff"))
		{
			searchRoutes.GET("/search", handlers.SearchCasesHandler)
		}
		clientCaseRoutes := protected.Group("/api/cases")
		clientCaseRoutes.Use(middleware.RequireRole("admin", "lawyer", "client"))
		{
			clientCaseRoutes.GET("", handlers.GetCasesHandler)
			clientCaseRoutes.GET("/:id/documents", handlers.GetCaseDocumentsHandler)
			clientCaseRoutes.POST("/:id/documents/upload", handlers.UploadCaseDocumentHandler)
			clientCaseRoutes.GET("/:id/documents/:docId/download", handlers.DownloadCaseDocumentHandler)
			clientCaseRoutes.GET("/:id/documents/:docId/view", handlers.ViewCaseDocumentHandler)
			clientCaseRoutes.GET("/:id/judicial-view", handlers.GetJudicialProcessViewHandler)
		}
		caseRoutes := protected.Group("/api/cases")
		caseRoutes.Use(middleware.RequireRole("admin", "lawyer"))
		{
			caseRoutes.GET("/new", handlers.CreateCaseModalHandler)
			caseRoutes.POST("", handlers.CreateCaseHandler)
			caseRoutes.GET("/:id/edit", handlers.GetCaseEditFormHandler)
			caseRoutes.PUT("/:id", handlers.UpdateCaseHandler)
			caseRoutes.PATCH("/:id/documents/:docId/visibility", handlers.ToggleDocumentVisibilityHandler)
			caseRoutes.DELETE("/:id/documents/:docId", handlers.DeleteCaseDocumentHandler)
			caseRoutes.POST("/:id/collaborators", handlers.AddCaseCollaboratorHandler)
			caseRoutes.DELETE("/:id/collaborators/:userId", handlers.RemoveCaseCollaboratorHandler)
			caseRoutes.GET("/:id/collaborators/available", handlers.GetAvailableCollaboratorsHandler)
			caseRoutes.GET("/import/modal", handlers.ImportCasesModalHandler)
			caseRoutes.GET("/import/template", handlers.GetImportTemplateHandler)
			caseRoutes.POST("/import", handlers.ImportCasesHandler)
			caseRoutes.GET("/:id/party/modal", handlers.GetCasePartyModalHandler)
			caseRoutes.POST("/:id/party", handlers.AddCasePartyHandler)
			caseRoutes.PUT("/:id/party", handlers.UpdateCasePartyHandler)
			caseRoutes.DELETE("/:id/party", handlers.DeleteCasePartyHandler)
			caseRoutes.GET("/:id/logs", handlers.GetCaseLogsHandler)
			caseRoutes.GET("/:id/logs/new", handlers.GetCaseLogFormHandler)
			caseRoutes.POST("/:id/logs", handlers.CreateCaseLogHandler)
			caseRoutes.GET("/:id/logs/:logId", handlers.GetCaseLogHandler)
			caseRoutes.GET("/:id/logs/:logId/view", handlers.GetCaseLogViewHandler)
			caseRoutes.PUT("/:id/logs/:logId", handlers.UpdateCaseLogHandler)
			caseRoutes.DELETE("/:id/logs/:logId", handlers.DeleteCaseLogHandler)
			caseRoutes.GET("/:id/generate", handlers.GetGenerateDocumentTabHandler)
			caseRoutes.GET("/:id/generate/preview", handlers.PreviewTemplateHandler)
			caseRoutes.POST("/:id/generate", handlers.GenerateDocumentHandler)
			caseRoutes.GET("/:id/generated", handlers.GetGeneratedDocumentsHandler)
			caseRoutes.GET("/:id/generated/:docId/download", handlers.DownloadGeneratedDocumentHandler)
			caseRoutes.GET("/:id/templates/modal", handlers.GetTemplateSelectorModalHandler)
			caseRoutes.GET("/history/new", handlers.GetHistoricalCaseFormHandler)
			caseRoutes.POST("/history", handlers.CreateHistoricalCaseHandler)
			caseRoutes.GET("/history/branches", handlers.GetHistoricalCaseBranchesHandler)
			caseRoutes.GET("/history/subtypes", handlers.GetHistoricalCaseSubtypesHandler)
			caseRoutes.GET("/history/subtypes", handlers.GetHistoricalCaseSubtypesHandler)
		}

		// Legal Services Routes
		protected.GET("/services", handlers.ServicesPageHandler)
		protected.GET("/services/:id", handlers.GetServiceDetailHandler)

		// Services Routes (Shared: Admin, Lawyer, Client)
		serviceShared := protected.Group("/api/services")
		serviceShared.Use(middleware.RequireRole("admin", "lawyer", "client"))
		{
			serviceShared.GET("", handlers.GetServicesHandler)
			serviceShared.GET("/:id", handlers.GetServiceHandler)
			serviceShared.GET("/:id/milestones", handlers.GetServiceMilestonesHandler)
			serviceShared.GET("/:id/timeline", handlers.GetServiceTimelineHandler)
			serviceShared.GET("/:id/documents", handlers.GetServiceDocumentsHandler)
			serviceShared.POST("/:id/documents/upload", handlers.UploadServiceDocumentHandler)
			serviceShared.GET("/:id/documents/:did/download", handlers.DownloadServiceDocumentHandler)
			serviceShared.GET("/:id/documents/:did/view", handlers.ViewServiceDocumentHandler)
		}

		// Services Routes (Admin/Lawyer Only)
		serviceAdmin := protected.Group("/api/services")
		serviceAdmin.Use(middleware.RequireRole("admin", "lawyer"))
		{
			// Service CRUD
			serviceAdmin.GET("/new", handlers.CreateServiceModalHandler)
			serviceAdmin.POST("", handlers.CreateServiceHandler)
			serviceAdmin.GET("/:id/edit", handlers.GetUpdateServiceModalHandler)
			serviceAdmin.PUT("/:id", handlers.UpdateServiceHandler)
			serviceAdmin.PATCH("/:id/status", handlers.UpdateServiceStatusHandler)
			serviceAdmin.GET("/:id/delete-confirm", handlers.DeleteServiceConfirmHandler)
			serviceAdmin.DELETE("/:id", handlers.DeleteServiceHandler)

			// Milestones Write
			serviceAdmin.POST("/:id/milestones", handlers.CreateMilestoneHandler)
			serviceAdmin.PUT("/:id/milestones/:mid", handlers.UpdateMilestoneHandler)
			serviceAdmin.PATCH("/:id/milestones/:mid/complete", handlers.CompleteMilestoneHandler)
			serviceAdmin.DELETE("/:id/milestones/:mid", handlers.DeleteMilestoneHandler)
			serviceAdmin.POST("/:id/milestones/reorder", handlers.ReorderMilestonesHandler)

			// Documents Write
			serviceAdmin.PATCH("/:id/documents/:did/visibility", handlers.ToggleServiceDocumentVisibilityHandler)
			serviceAdmin.DELETE("/:id/documents/:did", handlers.DeleteServiceDocumentHandler)

			// Expenses
			serviceAdmin.GET("/:id/expenses", handlers.GetServiceExpensesHandler)
			serviceAdmin.POST("/:id/expenses", handlers.CreateServiceExpenseHandler)
			serviceAdmin.GET("/:id/expenses/:eid/edit-modal", handlers.GetServiceExpenseEditModalHandler)
			serviceAdmin.PUT("/:id/expenses/:eid", handlers.UpdateServiceExpenseHandler)
			serviceAdmin.PATCH("/:id/expenses/:eid/approve", handlers.ApproveServiceExpenseHandler)
			serviceAdmin.DELETE("/:id/expenses/:eid", handlers.DeleteServiceExpenseHandler)

			// Activities
			serviceAdmin.GET("/:id/activities", handlers.GetServiceActivitiesHandler)
			serviceAdmin.GET("/:id/activities/new", handlers.GetServiceActivityForm)
			serviceAdmin.GET("/:id/activities/:aid/edit-modal", handlers.GetServiceActivityEditModalHandler)
			serviceAdmin.POST("/:id/activities", handlers.CreateServiceActivityHandler)
			serviceAdmin.PUT("/:id/activities/:aid", handlers.UpdateServiceActivityHandler)
			serviceAdmin.DELETE("/:id/activities/:aid", handlers.DeleteServiceActivityHandler)

			// Document Generation
			serviceAdmin.GET("/:id/templates/modal", handlers.GetServiceTemplateModalHandler)
			serviceAdmin.GET("/:id/generate/preview", handlers.PreviewServiceTemplateHandler)
			serviceAdmin.POST("/:id/generate", handlers.GenerateServiceDocumentHandler)
		}

		protected.GET("/historical-cases", handlers.HistoricalCasesPageHandler)
		adminRoutes.GET("/api/lawyers", handlers.GetLawyersForFilterHandler)
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
		adminRoutes.PUT("/api/firm/buffer-settings", handlers.UpdateBufferSettingsHandler)
		protected.GET("/calendar", handlers.CalendarPageHandler)
		protected.GET("/api/calendar/events", handlers.CalendarEventsHandler)
		protected.GET("/appointments", handlers.AppointmentsPageHandler)
		appointmentRoutes := protected.Group("/api/appointments")
		appointmentRoutes.Use(middleware.RequireRole("admin", "lawyer"))
		{
			appointmentRoutes.GET("", handlers.GetAppointmentsHandler)
			appointmentRoutes.GET("/slots", handlers.GetAvailableSlotsHandler)
			appointmentRoutes.GET("/clients", handlers.GetClientsForAppointmentHandler)
			appointmentRoutes.GET("/cases", handlers.GetCasesForAppointmentHandler)
			appointmentRoutes.GET("/lawyers", handlers.GetLawyersForAppointmentHandler)
			appointmentRoutes.GET("/types", handlers.GetActiveAppointmentTypesHandler)
			appointmentRoutes.POST("", handlers.CreateAppointmentHandler)
			appointmentRoutes.GET("/:id", handlers.GetAppointmentHandler)
			appointmentRoutes.PUT("/:id/status", handlers.UpdateAppointmentStatusHandler)
			appointmentRoutes.PUT("/:id/reschedule", handlers.RescheduleAppointmentHandler)
			appointmentRoutes.DELETE("/:id", handlers.CancelAppointmentHandler)
		}

		appointmentTypeRoutes := adminRoutes.Group("/appointment-types")
		{
			appointmentTypeRoutes.GET("", handlers.GetAppointmentTypesHandler)
			appointmentTypeRoutes.POST("", handlers.CreateAppointmentTypeHandler)
			appointmentTypeRoutes.PUT("/:id", handlers.UpdateAppointmentTypeHandler)
			appointmentTypeRoutes.DELETE("/:id", handlers.DeleteAppointmentTypeHandler)
		}
	}

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			if err := services.CleanupExpiredSessions(db.DB); err != nil {
				log.Printf("Error cleaning up expired sessions: %v", err)
			}

			if err := services.CleanupExpiredTokens(db.DB); err != nil {
				log.Printf("Error cleaning up expired tokens: %v", err)
			}

			if err := services.ExpireAddOns(db.DB); err != nil {
				log.Printf("Error expiring add-ons: %v", err)
			}
		}
	}()

	jobs.StartScheduler(db.DB)

	go func() {
		if err := e.Start(":" + cfg.ServerPort); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("Shutting down the server")
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	log.Println("Server gracefully stopped")
}

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
