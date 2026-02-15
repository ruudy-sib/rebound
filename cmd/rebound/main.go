package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"rebound/internal/adapter/primary/worker"
	"rebound/internal/config"
	"rebound/internal/port/secondary"
)

const appName = "rebound"

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Root context with cancellation for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build the dependency injection container.
	c, err := buildContainer(ctx)
	if err != nil {
		return fmt.Errorf("building container: %w", err)
	}

	// Invoke the application, resolving all dependencies and starting services.
	return c.Invoke(func(
		router http.Handler,
		w *worker.Worker,
		cfg *config.Config,
		logger *zap.Logger,
		redisClient *goredis.Client,
		producer secondary.MessageProducer,
	) {
		defer func() {
			// Clean up resources on shutdown.
			if err := redisClient.Close(); err != nil {
				logger.Error("error closing redis", zap.Error(err))
			}
			if err := producer.Close(); err != nil {
				logger.Error("error closing kafka producer", zap.Error(err))
			}
			_ = logger.Sync()
		}()

		logger.Info("starting application",
			zap.String("app", appName),
			zap.String("version", version),
			zap.String("environment", cfg.Environment),
			zap.String("http_addr", cfg.HTTPAddr),
		)

		// Start the background worker.
		workerCtx, workerCancel := context.WithCancel(ctx)
		defer workerCancel()

		errCh := make(chan error, 2)
		go func() {
			errCh <- w.Run(workerCtx)
		}()

		// Start the HTTP server.
		server := &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
		}

		go func() {
			logger.Info("http server listening", zap.String("addr", cfg.HTTPAddr))
			if srvErr := server.ListenAndServe(); srvErr != nil && srvErr != http.ErrServerClosed {
				errCh <- fmt.Errorf("http server: %w", srvErr)
			}
		}()

		// Wait for shutdown signal.
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-quit:
			logger.Info("received shutdown signal", zap.String("signal", sig.String()))
		case srvErr := <-errCh:
			if srvErr != nil && srvErr != context.Canceled {
				logger.Error("service error", zap.Error(srvErr))
			}
		}

		// Graceful shutdown with timeout.
		logger.Info("shutting down gracefully")
		cancel()
		workerCancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("http server shutdown error", zap.Error(err))
		}

		logger.Info("shutdown complete")
	})
}
