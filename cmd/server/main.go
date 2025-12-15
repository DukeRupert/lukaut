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
	"github.com/DukeRupert/lukaut/internal/handler"
	"github.com/DukeRupert/lukaut/internal/middleware"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/service"
	_ "github.com/jackc/pgx/v5/stdlib"
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

	// Initialize template renderer
	renderer, err := handler.NewRenderer(handler.RendererConfig{
		TemplatesDir: "web/templates",
		Logger:       logger,
		IsDev:        cfg.Env == "development",
	})
	if err != nil {
		return fmt.Errorf("renderer initialization failed: %w", err)
	}
	logger.Info("Templates loaded", "count", len(renderer.ListTemplates()))

	// Initialize services
	userService := service.NewUserService(repo, logger)

	// Initialize middleware
	isSecure := cfg.Env != "development"
	authMw := middleware.NewAuthMiddleware(userService, logger, isSecure)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(userService, renderer, logger, isSecure)

	// ==========================================================================
	// Create router and register routes
	// ==========================================================================

	mux := http.NewServeMux()

	// Static files
	staticFS := http.FileServer(http.Dir("web/static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticFS))

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Public pages
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// Only handle exact root path
		if r.URL.Path != "/" {
			handler.NotFoundResponse(w, r, logger)
			return
		}
		renderer.RenderHTTP(w, "public/home", map[string]interface{}{
			"CurrentPath": r.URL.Path,
		})
	})

	// Auth routes (public - no auth required)
	authHandler.RegisterRoutes(mux)

	// Create middleware stacks for protected routes
	requireUser := middleware.Stack(authMw.WithUser, authMw.RequireUser)

	// Dashboard (requires authentication)
	mux.Handle("GET /dashboard", requireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetUser(r.Context())
		renderer.RenderHTTP(w, "dashboard", map[string]interface{}{
			"CurrentPath":       r.URL.Path,
			"User":              user,
			"Stats":             map[string]int{},
			"RecentInspections": nil,
		})
	})))

	// ==========================================================================
	// Start server
	// ==========================================================================

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
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
