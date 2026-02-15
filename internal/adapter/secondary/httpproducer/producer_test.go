package httpproducer

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"kafkaretry/internal/config"
	"kafkaretry/internal/domain/entity"
)

func TestProducer_Produce_Success(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		expectedKey := "task-123|1"
		if r.Header.Get("X-Message-Key") != expectedKey {
			t.Errorf("expected X-Message-Key: %s, got %s", expectedKey, r.Header.Get("X-Message-Key"))
		}

		// Verify body
		body, _ := io.ReadAll(r.Body)
		expectedBody := `{"test":"data"}`
		if string(body) != expectedBody {
			t.Errorf("expected body: %s, got %s", expectedBody, string(body))
		}

		// Return success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create producer
	cfg := &config.Config{}
	logger := zap.NewNop()
	producer := NewProducer(cfg, logger)
	defer producer.Close()

	// Test produce
	ctx := context.Background()
	destination := entity.Destination{
		URL: server.URL,
	}
	key := []byte("task-123|1")
	value := []byte(`{"test":"data"}`)

	err := producer.Produce(ctx, destination, key, value)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProducer_Produce_HTTPError(t *testing.T) {
	// Create a test HTTP server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer server.Close()

	// Create producer
	cfg := &config.Config{}
	logger := zap.NewNop()
	producer := NewProducer(cfg, logger)
	defer producer.Close()

	// Test produce
	ctx := context.Background()
	destination := entity.Destination{
		URL: server.URL,
	}
	key := []byte("task-123|1")
	value := []byte(`{"test":"data"}`)

	err := producer.Produce(ctx, destination, key, value)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedErr := "http request failed with status 500"
	if err.Error()[:len(expectedErr)] != expectedErr {
		t.Errorf("expected error to start with %q, got %q", expectedErr, err.Error())
	}
}

func TestProducer_Produce_MissingURL(t *testing.T) {
	// Create producer
	cfg := &config.Config{}
	logger := zap.NewNop()
	producer := NewProducer(cfg, logger)
	defer producer.Close()

	// Test produce with empty URL
	ctx := context.Background()
	destination := entity.Destination{
		URL: "",
	}
	key := []byte("task-123|1")
	value := []byte(`{"test":"data"}`)

	err := producer.Produce(ctx, destination, key, value)
	if err == nil {
		t.Fatal("expected error for missing URL, got nil")
	}

	expectedErr := "destination URL is required for HTTP delivery"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestProducer_Close(t *testing.T) {
	cfg := &config.Config{}
	logger := zap.NewNop()
	producer := NewProducer(cfg, logger)

	err := producer.Close()
	if err != nil {
		t.Fatalf("unexpected error closing producer: %v", err)
	}
}
