package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"rebound/internal/domain/entity"
)

// mockTaskService implements primary.TaskService for worker tests.
type mockTaskService struct {
	processFunc  func(ctx context.Context) error
	processCalls atomic.Int32
}

func (m *mockTaskService) CreateTask(_ context.Context, _ *entity.Task) error {
	return nil
}

func (m *mockTaskService) ProcessDueTasks(ctx context.Context) error {
	m.processCalls.Add(1)
	if m.processFunc != nil {
		return m.processFunc(ctx)
	}
	return nil
}

func TestWorker_Run(t *testing.T) {
	tests := []struct {
		name             string
		pollInterval     time.Duration
		runDuration      time.Duration
		processErr       error
		wantMinCalls     int32
		wantContextError bool
	}{
		{
			name:             "processes tasks at poll interval",
			pollInterval:     50 * time.Millisecond,
			runDuration:      200 * time.Millisecond,
			wantMinCalls:     2,
			wantContextError: true,
		},
		{
			name:         "continues on process error",
			pollInterval: 50 * time.Millisecond,
			runDuration:  200 * time.Millisecond,
			processErr:   errors.New("redis timeout"),
			wantMinCalls: 2,
			wantContextError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTaskService{}
			if tt.processErr != nil {
				svc.processFunc = func(_ context.Context) error {
					return tt.processErr
				}
			}

			w := NewWorker(svc, tt.pollInterval, zap.NewNop())

			ctx, cancel := context.WithTimeout(context.Background(), tt.runDuration)
			defer cancel()

			err := w.Run(ctx)

			if tt.wantContextError {
				if err == nil {
					t.Fatal("expected context error, got nil")
				}
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expected DeadlineExceeded, got %v", err)
				}
			}

			calls := svc.processCalls.Load()
			if calls < tt.wantMinCalls {
				t.Fatalf("expected at least %d process calls, got %d", tt.wantMinCalls, calls)
			}
		})
	}
}

func TestWorker_Run_respectsCancellation(t *testing.T) {
	svc := &mockTaskService{}
	w := NewWorker(svc, 1*time.Hour, zap.NewNop()) // Very long interval

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- w.Run(ctx)
	}()

	// Cancel immediately
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop within 2 seconds after cancellation")
	}
}
