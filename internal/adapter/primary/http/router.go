package http

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/ruudy-sib/rebound/internal/port/primary"
	"github.com/ruudy-sib/rebound/internal/port/secondary"
)

// NewRouter creates an HTTP mux with all application routes registered.
func NewRouter(
	taskService primary.TaskService,
	healthChecks []secondary.HealthChecker,
	logger *zap.Logger,
) http.Handler {
	mux := http.NewServeMux()

	// Task endpoints
	createHandler := NewCreateTaskHandler(taskService, logger)
	mux.Handle("/tasks", createHandler)

	// Health check endpoint
	healthHandler := NewHealthHandler(healthChecks)
	mux.Handle("/health", healthHandler)

	return mux
}
