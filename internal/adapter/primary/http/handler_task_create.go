package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"rebound/internal/domain"
	"rebound/internal/port/primary"
)

// CreateTaskHandler handles POST /tasks requests.
type CreateTaskHandler struct {
	service primary.TaskService
	logger  *zap.Logger
}

// NewCreateTaskHandler creates a handler for task creation.
func NewCreateTaskHandler(service primary.TaskService, logger *zap.Logger) *CreateTaskHandler {
	return &CreateTaskHandler{
		service: service,
		logger:  logger.Named("create-task-handler"),
	}
}

// ServeHTTP processes the create task request.
func (h *CreateTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error: "method not allowed",
			Code:  "METHOD_NOT_ALLOWED",
		})
		return
	}

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "invalid request body",
			Code:  "INVALID_BODY",
		})
		return
	}

	task := req.toEntity()
	if err := h.service.CreateTask(r.Context(), task); err != nil {
		if errors.Is(err, domain.ErrInvalidTask) {
			respondJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: err.Error(),
				Code:  "VALIDATION_ERROR",
			})
			return
		}
		h.logger.Error("failed to create task", zap.Error(err))
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error: "internal server error",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	respondJSON(w, http.StatusCreated, CreateTaskResponse{
		Message: fmt.Sprintf("Task %s scheduled successfully", task.ID),
	})
}
