package http

import (
	"net/http"

	"rebound/internal/port/secondary"
)

// HealthHandler handles GET /health requests.
type HealthHandler struct {
	checks []secondary.HealthChecker
}

// NewHealthHandler creates a health check handler with the given checkers.
func NewHealthHandler(checks []secondary.HealthChecker) *HealthHandler {
	return &HealthHandler{checks: checks}
}

// ServeHTTP performs all health checks and reports the aggregate status.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	checks := make(map[string]string)

	for _, check := range h.checks {
		if err := check.Check(r.Context()); err != nil {
			status = http.StatusServiceUnavailable
			checks[check.Name()] = err.Error()
		} else {
			checks[check.Name()] = "ok"
		}
	}

	statusText := "healthy"
	if status != http.StatusOK {
		statusText = "unhealthy"
	}

	respondJSON(w, status, HealthResponse{
		Status: statusText,
		Checks: checks,
	})
}
