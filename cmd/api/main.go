package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"trackflow/internal/auth"
	"trackflow/internal/client"
	"trackflow/internal/config"
	"trackflow/internal/event"
	"trackflow/internal/httpx"
	"trackflow/internal/link"
	"trackflow/internal/postgresx"
	"trackflow/internal/redirect"
	"trackflow/internal/redisx"
	"trackflow/internal/report"
	"trackflow/internal/stats"
	"trackflow/internal/user"
)

func main() {
	cfg := config.LoadAPI()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer startupCancel()

	db, err := postgresx.Open(startupCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	userRepo := user.NewRepository(db)
	linkRepo := link.NewRepository(db)
	statsRepo := stats.NewRepository(db)
	clientRepo := client.NewRepository(db)
	reportRepo := report.NewRepository(db)

	tokenManager := auth.NewTokenManager(cfg.JWTSecret, 24*time.Hour)
	authHandler := auth.NewHandler(userRepo, tokenManager)
	authMiddleware := auth.Middleware(tokenManager)
	eventOutbox := event.NewOutbox(db)
	eventPublisher := event.NewPublisher(cfg.RedisURL, &eventOutbox)
	redirectHandler := redirect.NewHandler(linkRepo, redirect.NewCache(cfg.RedisURL), eventPublisher, logger)
	redirectCache := redirect.NewCache(cfg.RedisURL)
	linkHandler := link.NewHandler(linkRepo, redirectCache)
	statsHandler := stats.NewHandler(statsRepo)
	clientHandler := client.NewHandler(clientRepo)

	pdfStoragePath := os.Getenv("PDF_STORAGE_PATH")
	if pdfStoragePath == "" {
		pdfStoragePath = "/data/reports"
	}
	reportHandler := report.NewHandler(reportRepo, eventPublisher, pdfStoragePath)

	router := chi.NewRouter()
	router.Get("/healthz", httpx.HealthHandler("api"))
	router.Get("/readyz", httpx.ReadinessHandler("api", map[string]func(context.Context) error{
		"postgres": db.Ping,
		"redis": func(ctx context.Context) error {
			return redisx.Ping(ctx, cfg.RedisURL)
		},
	}))
	router.Post("/auth/login", authHandler.Login)

	router.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/auth/change-password", authHandler.ChangePassword)
		r.Get("/api/me", authHandler.Me)
		r.Patch("/api/me/password", authHandler.ChangePassword)
		r.Get("/api/links", linkHandler.List)
		r.Post("/api/links", linkHandler.Create)
		r.Get("/api/links/{id}", linkHandler.Get)
		r.Get("/api/links/{id}/stats", statsHandler.Get)
		r.Delete("/api/links/{id}", linkHandler.Delete)
		r.Get("/api/clients", clientHandler.List)
		r.Post("/api/clients", clientHandler.Create)
		r.Get("/api/reports", reportHandler.List)
		r.Post("/api/reports", reportHandler.Create)
		r.Get("/api/reports/{id}", reportHandler.Get)
		r.Get("/api/reports/{id}/download", reportHandler.Download)
	})
	router.Get("/{short_code:[A-Za-z0-9_-]+}", redirectHandler.Redirect)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 3 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("api listening", "addr", cfg.HTTPAddr)
		errCh <- server.ListenAndServe()
	}()

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-shutdownCh:
		logger.Info("api shutting down", "signal", sig.String())
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api stopped", "error", err)
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("api shutdown failed", "error", err)
		os.Exit(1)
	}
}
