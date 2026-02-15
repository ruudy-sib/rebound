package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"rebound/internal/domain"
)

func TestCreateTaskHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           interface{}
		createErr      error
		wantStatusCode int
		wantMessage    string
	}{
		{
			name:   "successful task creation",
			method: http.MethodPost,
			body: CreateTaskRequest{
				ID:     "task-1",
				Source: "test-app",
				Destination: DestinationDTO{
					Host:  "localhost",
					Port:  "9092",
					Topic: "my-topic",
				},
				DeadDestination: DestinationDTO{
					Host:  "localhost",
					Port:  "9092",
					Topic: "dead-topic",
				},
				MaxRetries:      3,
				BaseDelay:       2,
				ClientID:        "client-1",
				MessageData:     "test data",
				DestinationType: "kafka",
			},
			wantStatusCode: http.StatusCreated,
			wantMessage:    "Task task-1 scheduled successfully",
		},
		{
			name:           "method not allowed",
			method:         http.MethodGet,
			body:           nil,
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid JSON body",
			method:         http.MethodPost,
			body:           "not json",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:   "validation error",
			method: http.MethodPost,
			body: CreateTaskRequest{
				ID: "", // Missing required field
			},
			createErr:      domain.ErrInvalidTask,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:   "internal server error",
			method: http.MethodPost,
			body: CreateTaskRequest{
				ID:     "task-2",
				Source: "test-app",
				Destination: DestinationDTO{
					Host: "localhost", Port: "9092", Topic: "my-topic",
				},
				MaxRetries:      3,
				BaseDelay:       2,
				DestinationType: "kafka",
			},
			createErr:      errors.New("unexpected error"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockTaskService{
				createErr: tt.createErr,
			}
			handler := NewCreateTaskHandler(mockSvc, zap.NewNop())

			var bodyBytes []byte
			if tt.body != nil {
				switch v := tt.body.(type) {
				case string:
					bodyBytes = []byte(v)
				default:
					bodyBytes, _ = json.Marshal(v)
				}
			}

			req := httptest.NewRequest(tt.method, "/tasks", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Fatalf("expected status %d, got %d (body: %s)", tt.wantStatusCode, rec.Code, rec.Body.String())
			}

			if tt.wantMessage != "" {
				var resp CreateTaskResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Message != tt.wantMessage {
					t.Fatalf("expected message %q, got %q", tt.wantMessage, resp.Message)
				}
			}
		})
	}
}

func TestHealthHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		checks         []mockHealthCheck
		wantStatusCode int
		wantStatus     string
	}{
		{
			name:           "healthy with no checks",
			checks:         nil,
			wantStatusCode: http.StatusOK,
			wantStatus:     "healthy",
		},
		{
			name: "healthy with passing checks",
			checks: []mockHealthCheck{
				{name: "redis", err: nil},
			},
			wantStatusCode: http.StatusOK,
			wantStatus:     "healthy",
		},
		{
			name: "unhealthy with failing check",
			checks: []mockHealthCheck{
				{name: "redis", err: errors.New("connection refused")},
			},
			wantStatusCode: http.StatusServiceUnavailable,
			wantStatus:     "unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var checks []healthCheckerAdapter
			for _, c := range tt.checks {
				checks = append(checks, healthCheckerAdapter{check: c})
			}

			handler := NewHealthHandler(toHealthCheckers(checks))

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Fatalf("expected status %d, got %d", tt.wantStatusCode, rec.Code)
			}

			var resp HealthResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if resp.Status != tt.wantStatus {
				t.Fatalf("expected status %q, got %q", tt.wantStatus, resp.Status)
			}
		})
	}
}
