package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DukeRupert/lukaut/internal"
	"github.com/DukeRupert/lukaut/internal/ai"
	"github.com/DukeRupert/lukaut/internal/ai/anthropic"
	"github.com/DukeRupert/lukaut/internal/ai/mock"
	"github.com/DukeRupert/lukaut/internal/email"
	"github.com/DukeRupert/lukaut/internal/handler"
	"github.com/DukeRupert/lukaut/internal/invite"
	"github.com/DukeRupert/lukaut/internal/jobs"
	"github.com/DukeRupert/lukaut/internal/metrics"
	"github.com/DukeRupert/lukaut/internal/middleware"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/DukeRupert/lukaut/internal/storage"
	publicpages "github.com/DukeRupert/lukaut/internal/templ/pages/public"
	"github.com/DukeRupert/lukaut/internal/worker"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func run() error {
	ctx := context.Background()

	// Load configuration
	cfg, err := internal.NewConfig()
	if err != nil {
		return fmt.Errorf("config initialization failed: %w", err)
	}

	// Configure logger
	logger := internal.NewLogger(os.Stdout, cfg.Env, cfg.LogLevel)

	// Initialize database connection
	db, err := sql.Open("pgx", cfg.DatabaseUrl)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Run migrations
	if err := internal.RunMigrations(db); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	logger.Info("Database ready")

	// Initialize repository
	repo := repository.New(db)

	// Initialize storage service
	var storageService storage.Storage
	if cfg.StorageProvider == storage.ProviderR2 {
		storageService, err = storage.NewR2Storage(storage.R2Config{
			AccountID:       cfg.R2AccountID,
			AccessKeyID:     cfg.R2AccessKeyID,
			SecretAccessKey: cfg.R2SecretAccessKey,
			BucketName:      cfg.R2BucketName,
			PublicURL:       cfg.R2PublicURL,
			Region:          "auto",
		}, logger)
		if err != nil {
			return fmt.Errorf("R2 storage initialization failed: %w", err)
		}
	} else {
		storageService, err = storage.NewLocalStorage(storage.LocalConfig{
			BasePath: cfg.LocalStoragePath,
			BaseURL:  cfg.LocalStorageURL,
		}, logger)
		if err != nil {
			return fmt.Errorf("local storage initialization failed: %w", err)
		}
	}
	logger.Info("Storage service initialized", "provider", cfg.StorageProvider)

	// Initialize services
	userService := service.NewUserService(repo, logger)
	inspectionService := service.NewInspectionService(repo, logger)
	violationService := service.NewViolationService(repo, logger)
	clientService := service.NewClientService(repo, logger)

	// Initialize thumbnail processor
	thumbnailProcessor := service.NewImagingProcessor()

	// Initialize image service
	imageService := service.NewImageService(repo, storageService, thumbnailProcessor, logger)

	// Initialize email service
	emailService, err := email.NewSMTPEmailService(
		email.SMTPConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
			FromName: cfg.SMTPFromName,
		},
		cfg.BaseURL,
		"web/templates/email",
		logger,
	)
	if err != nil {
		return fmt.Errorf("email service initialization failed: %w", err)
	}
	logger.Info("Email service initialized", "host", cfg.SMTPHost, "port", cfg.SMTPPort)

	// Initialize AI provider
	var aiProvider ai.AIProvider
	if cfg.AIProvider == "anthropic" {
		aiProvider, err = anthropic.New(
			anthropic.Config{
				APIKey: cfg.AnthropicAPIKey,
				Model:  cfg.AnthropicModel,
				ProviderConfig: ai.ProviderConfig{
					MaxRetries:     cfg.AIMaxRetries,
					RetryBaseDelay: cfg.AIRetryBaseDelay,
					RequestTimeout: cfg.AIRequestTimeout,
				},
			},
			repo,
			logger,
		)
		if err != nil {
			return fmt.Errorf("anthropic provider initialization failed: %w", err)
		}
	} else {
		aiProvider = mock.New(logger)
	}
	logger.Info("AI provider initialized", "provider", cfg.AIProvider)

	// Initialize background worker
	var jobWorker *worker.Worker
	if cfg.WorkerEnabled {
		workerConfig := worker.Config{
			Concurrency:       cfg.WorkerConcurrency,
			PollInterval:      cfg.WorkerPollInterval,
			JobTimeout:        cfg.WorkerJobTimeout,
			ShutdownTimeout:   30 * time.Second,
			StaleJobThreshold: 10 * time.Minute,
		}

		jobWorker, err = worker.New(db, repo, workerConfig, logger)
		if err != nil {
			return fmt.Errorf("worker initialization failed: %w", err)
		}

		// Register job handlers
		jobWorker.Register(jobs.NewAnalyzeInspectionHandler(repo, aiProvider, storageService, inspectionService, logger))
		jobWorker.Register(jobs.NewGenerateReportHandler(repo, storageService, emailService, logger, cfg.BaseURL))

		// Start the worker
		jobWorker.Start(ctx)
		logger.Info("Background worker started", "concurrency", workerConfig.Concurrency)
	}

	// Initialize invite code validator for MVP testing
	inviteValidator := invite.New(cfg.InviteCodesEnabled, cfg.ValidInviteCodes)
	if inviteValidator.IsEnabled() {
		logger.Info("invite code validation enabled", "valid_codes_count", len(cfg.ValidInviteCodes))
	} else {
		logger.Info("invite code validation disabled - open registration")
	}

	// Initialize middleware
	isSecure := cfg.Env != "development"
	authMw := middleware.NewAuthMiddleware(userService, logger, isSecure)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(userService, emailService, inviteValidator, logger, isSecure)
	dashboardHandler := handler.NewDashboardHandler(repo, logger)
	inspectionHandler := handler.NewInspectionHandler(inspectionService, imageService, violationService, clientService, repo, logger)
	imageHandler := handler.NewImageHandler(imageService, inspectionService, repo, logger)
	violationHandler := handler.NewViolationHandler(violationService, inspectionService, imageService, logger)
	regulationHandler := handler.NewRegulationHandler(repo, logger)
	settingsHandler := handler.NewSettingsHandler(userService, logger)
	clientHandler := handler.NewClientHandler(clientService, logger)
	reportHandler := handler.NewReportHandler(repo, storageService, logger)

	// ==========================================================================
	// Create router and register routes
	// ==========================================================================

	mux := http.NewServeMux()

	// Static files
	staticFS := http.FileServer(http.Dir("web/static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticFS))

	// File storage (local development only)
	if cfg.StorageProvider == storage.ProviderLocal {
		filesFS := http.FileServer(http.Dir(cfg.LocalStoragePath))
		mux.Handle("GET /files/", http.StripPrefix("/files/", filesFS))
		logger.Info("Local file server enabled", "path", cfg.LocalStoragePath)
	}

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Prometheus metrics endpoint
	mux.Handle("GET /metrics", promhttp.Handler())

	// Public pages - using templ
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// Only handle exact root path
		if r.URL.Path != "/" {
			handler.NotFoundResponse(w, r, logger)
			return
		}
		component := publicpages.HomePage(publicpages.HomePageData{
			CurrentPath: r.URL.Path,
		})
		if err := component.Render(r.Context(), w); err != nil {
			logger.Error("failed to render home page", "error", err)
		}
	})

	// Auth routes (public - no auth required)
	// Using templ-based routes with CSRF protection
	authHandler.RegisterTemplRoutes(mux)

	// Create middleware stacks for protected routes
	requireUser := middleware.Stack(authMw.WithUser, authMw.RequireUser)

	// Dashboard (requires authentication) - using templ
	mux.Handle("GET /dashboard", requireUser(http.HandlerFunc(dashboardHandler.ShowTempl)))

	// Inspection routes (requires authentication) - using templ
	inspectionHandler.RegisterTemplRoutes(mux, requireUser)

	// Image routes (requires authentication)
	imageHandler.RegisterRoutes(mux, requireUser)

	// Violation routes (requires authentication)
	violationHandler.RegisterRoutes(mux, requireUser)

	// Regulation routes (requires authentication) - using templ
	regulationHandler.RegisterTemplRoutes(mux, requireUser)

	// Settings routes (requires authentication) - using templ
	settingsHandler.RegisterTemplRoutes(mux, requireUser)

	// Client routes (requires authentication) - using templ
	clientHandler.RegisterTemplRoutes(mux, requireUser)

	// Report routes (requires authentication)
	reportHandler.RegisterRoutes(mux, requireUser)

	// ==========================================================================
	// Start server
	// ==========================================================================

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: metrics.Middleware(mux),
	}

	// Channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		logger.Info("Server started", "address", server.Addr, "env", cfg.Env)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "error", err)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	logger.Info("Shutdown signal received, initiating graceful shutdown...")

	// Stop background worker first (if running)
	if jobWorker != nil {
		logger.Info("Stopping background worker...")
		jobWorker.Stop()
	}

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", "error", err)
	}

	logger.Info("Graceful shutdown complete")
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
