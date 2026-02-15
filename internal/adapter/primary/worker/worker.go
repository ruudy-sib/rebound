package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"rebound/internal/port/primary"
)

// Worker polls for due tasks at regular intervals and processes them.
// It respects context cancellation for graceful shutdown.
type Worker struct {
	service      primary.TaskService
	pollInterval time.Duration
	logger       *zap.Logger
}

// NewWorker creates a Worker that processes tasks at the given interval.
func NewWorker(
	service primary.TaskService,
	pollInterval time.Duration,
	logger *zap.Logger,
) *Worker {
	return &Worker{
		service:      service,
		pollInterval: pollInterval,
		logger:       logger.Named("worker"),
	}
}

// Run starts the polling loop. It blocks until the context is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("worker started",
		zap.Duration("poll_interval", w.pollInterval),
	)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("worker shutting down")
			return ctx.Err()
		case <-ticker.C:
			if err := w.service.ProcessDueTasks(ctx); err != nil {
				// Log but do not return -- the worker should keep running.
				w.logger.Error("error processing due tasks", zap.Error(err))
			}
		}
	}
}
