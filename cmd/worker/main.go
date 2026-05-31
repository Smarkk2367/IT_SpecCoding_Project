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
	"github.com/redis/go-redis/v9"

	"trackflow/internal/click"
	"trackflow/internal/config"
	"trackflow/internal/event"
	"trackflow/internal/httpx"
	"trackflow/internal/postgresx"
	"trackflow/internal/redisx"
	"trackflow/internal/report"
	"trackflow/internal/worker"
)

func main() {
	cfg := config.LoadWorker()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer startupCancel()

	db, err := postgresx.Open(startupCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	redisOptions, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Error("redis url parse failed", "error", err)
		os.Exit(1)
	}
	redisClient := redis.NewClient(redisOptions)
	defer redisClient.Close()

	clickRepo := click.NewRepository(db)
	clickConsumer := worker.NewClickConsumer(redisClient, clickRepo, cfg.ConsumerGroup, cfg.ConsumerName, logger)

	reportRepo := report.NewRepository(db)
	eventOutbox := event.NewOutbox(db)
	eventPublisher := event.NewPublisher(cfg.RedisURL, &eventOutbox)
	reportConsumer := worker.NewReportConsumer(redisClient, reportRepo, eventPublisher, cfg.ConsumerGroup, cfg.ConsumerName, cfg.PDFStoragePath, logger)

	router := chi.NewRouter()
	router.Get("/healthz", httpx.HealthHandler("worker"))
	router.Get("/readyz", httpx.ReadinessHandler("worker", map[string]func(context.Context) error{
		"postgres": db.Ping,
		"redis": func(ctx context.Context) error {
			return redisx.Ping(ctx, cfg.RedisURL)
		},
	}))

	server := &http.Server{
		Addr:              cfg.HealthAddr,
		Handler:           router,
		ReadHeaderTimeout: 3 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("worker health server listening", "addr", cfg.HealthAddr)
		errCh <- server.ListenAndServe()
	}()

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	go func() {
		logger.Info("click consumer started", "stream", "clicks", "group", cfg.ConsumerGroup, "consumer", cfg.ConsumerName)
		err := clickConsumer.Run(workerCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()

	go func() {
		logger.Info("report consumer started", "stream", "reports", "group", cfg.ConsumerGroup, "consumer", cfg.ConsumerName)
		err := reportConsumer.Run(workerCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-shutdownCh:
		logger.Info("worker shutting down", "signal", sig.String())
		workerCancel()
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("worker stopped", "error", err)
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("worker shutdown failed", "error", err)
		os.Exit(1)
	}
}
