package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/ruudy-sib/rebound/internal/domain"
	"github.com/ruudy-sib/rebound/internal/domain/entity"
)

func TestTaskService_CreateTask(t *testing.T) {
	tests := []struct {
		name          string
		task          *entity.Task
		scheduleErr   error
		wantErr       error
		wantScheduled bool
	}{
		{
			name:          "valid task is scheduled",
			task:          testTask(),
			scheduleErr:   nil,
			wantErr:       nil,
			wantScheduled: true,
		},
		{
			name: "missing ID returns validation error",
			task: func() *entity.Task {
				t := testTask()
				t.ID = ""
				return t
			}(),
			wantErr:       domain.ErrInvalidTask,
			wantScheduled: false,
		},
		{
			name: "missing source returns validation error",
			task: func() *entity.Task {
				t := testTask()
				t.Source = ""
				return t
			}(),
			wantErr:       domain.ErrInvalidTask,
			wantScheduled: false,
		},
		{
			name: "missing destination topic returns validation error",
			task: func() *entity.Task {
				t := testTask()
				t.Destination.Topic = ""
				return t
			}(),
			wantErr:       domain.ErrInvalidTask,
			wantScheduled: false,
		},
		{
			name: "missing destination type returns validation error",
			task: func() *entity.Task {
				t := testTask()
				t.DestinationType = ""
				return t
			}(),
			wantErr:       domain.ErrInvalidTask,
			wantScheduled: false,
		},
		{
			name: "base delay below minimum returns validation error",
			task: func() *entity.Task {
				t := testTask()
				t.BaseDelay = 0
				return t
			}(),
			wantErr:       domain.ErrInvalidTask,
			wantScheduled: false,
		},
		{
			name: "base delay above maximum returns validation error",
			task: func() *entity.Task {
				t := testTask()
				t.BaseDelay = 9999
				return t
			}(),
			wantErr:       domain.ErrInvalidTask,
			wantScheduled: false,
		},
		{
			name: "negative max retries returns validation error",
			task: func() *entity.Task {
				t := testTask()
				t.MaxRetries = -1
				return t
			}(),
			wantErr:       domain.ErrInvalidTask,
			wantScheduled: false,
		},
		{
			name:          "valid http task is scheduled",
			task:          testHTTPTask(),
			wantErr:       nil,
			wantScheduled: true,
		},
		{
			name: "http task missing url returns validation error",
			task: func() *entity.Task {
				t := testHTTPTask()
				t.Destination.URL = ""
				return t
			}(),
			wantErr:       domain.ErrInvalidTask,
			wantScheduled: false,
		},
		{
			name: "kafka task missing topic returns validation error",
			task: func() *entity.Task {
				t := testTask()
				t.Destination.Topic = ""
				return t
			}(),
			wantErr:       domain.ErrInvalidTask,
			wantScheduled: false,
		},
		{
			name:          "scheduler error is wrapped",
			task:          testTask(),
			scheduleErr:   errors.New("redis connection refused"),
			wantErr:       domain.ErrScheduleFailed,
			wantScheduled: false,
		},
		{
			name: "attempt is reset to 0",
			task: func() *entity.Task {
				t := testTask()
				t.Attempt = 5
				return t
			}(),
			wantErr:       nil,
			wantScheduled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := &mockScheduler{}
			if tt.scheduleErr != nil {
				scheduler.scheduleFunc = func(_ context.Context, _ *entity.Task, _ time.Duration) error {
					return tt.scheduleErr
				}
			}
			producer := &mockProducer{}
			logger := zap.NewNop()

			svc := NewTaskService(scheduler, producer, logger)
			err := svc.CreateTask(context.Background(), tt.task)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error wrapping %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantScheduled && len(scheduler.scheduledTasks) != 1 {
				t.Fatalf("expected 1 scheduled task, got %d", len(scheduler.scheduledTasks))
			}

			if tt.wantScheduled && tt.task.Attempt != 0 {
				t.Fatalf("expected attempt reset to 0, got %d", tt.task.Attempt)
			}

			if tt.wantScheduled {
				wantDelay := time.Duration(tt.task.BaseDelay) * time.Second
				gotDelay := scheduler.scheduledTasks[0].Delay
				if gotDelay != wantDelay {
					t.Fatalf("expected initial delay %v, got %v", wantDelay, gotDelay)
				}
			}
		})
	}
}

func TestTaskService_ProcessDueTasks(t *testing.T) {
	tests := []struct {
		name                string
		fetchedTasks        []*entity.Task
		fetchErr            error
		produceErr          error
		wantErr             bool
		wantSuccessProduced int
		wantTotalProduced   int
	}{
		{
			name:                "no due tasks does nothing",
			fetchedTasks:        nil,
			wantErr:             false,
			wantSuccessProduced: 0,
			wantTotalProduced:   0,
		},
		{
			name: "successful delivery",
			fetchedTasks: []*entity.Task{
				testTask(),
			},
			wantErr:             false,
			wantSuccessProduced: 1,
			wantTotalProduced:   1,
		},
		{
			name:     "fetch error is returned",
			fetchErr: errors.New("redis timeout"),
			wantErr:  true,
		},
		{
			name: "delivery failure triggers retry scheduling",
			fetchedTasks: []*entity.Task{
				testTask(),
			},
			produceErr:          errors.New("kafka unavailable"),
			wantErr:             false,
			wantSuccessProduced: 0,
			wantTotalProduced:   1, // 1 failed attempt
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := &mockScheduler{
				fetchDueFunc: func(_ context.Context, _ int) ([]*entity.Task, error) {
					return tt.fetchedTasks, tt.fetchErr
				},
			}
			producer := &mockProducer{}
			if tt.produceErr != nil {
				producer.produceFunc = func(_ context.Context, _ entity.Destination, _, _ []byte) error {
					return tt.produceErr
				}
			}
			logger := zap.NewNop()

			svc := NewTaskService(scheduler, producer, logger)
			err := svc.ProcessDueTasks(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			successCount := len(producer.successfulProduceCalls())
			if successCount != tt.wantSuccessProduced {
				t.Fatalf("expected %d successful produces, got %d", tt.wantSuccessProduced, successCount)
			}

			if len(producer.produceCalls) != tt.wantTotalProduced {
				t.Fatalf("expected %d total produce calls, got %d", tt.wantTotalProduced, len(producer.produceCalls))
			}
		})
	}
}

func TestTaskService_ProcessDueTasks_retryScheduling(t *testing.T) {
	task := testTask()
	task.Attempt = 1

	scheduler := &mockScheduler{
		fetchDueFunc: func(_ context.Context, _ int) ([]*entity.Task, error) {
			return []*entity.Task{task}, nil
		},
	}
	producer := &mockProducer{
		produceFunc: func(_ context.Context, _ entity.Destination, _, _ []byte) error {
			return errors.New("kafka down")
		},
	}
	logger := zap.NewNop()

	svc := NewTaskService(scheduler, producer, logger)
	err := svc.ProcessDueTasks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Task should have been rescheduled (attempt incremented to 2)
	if len(scheduler.scheduledTasks) != 1 {
		t.Fatalf("expected 1 scheduled retry, got %d", len(scheduler.scheduledTasks))
	}

	rescheduled := scheduler.scheduledTasks[0]
	if rescheduled.Task.Attempt != 2 {
		t.Fatalf("expected attempt 2, got %d", rescheduled.Task.Attempt)
	}
	if rescheduled.Delay <= 0 {
		t.Fatalf("expected positive delay, got %v", rescheduled.Delay)
	}
}

func TestTaskService_ProcessDueTasks_deadLetterOnExhaustedRetries(t *testing.T) {
	task := testTask()
	task.Attempt = 3
	task.MaxRetries = 3

	scheduler := &mockScheduler{
		fetchDueFunc: func(_ context.Context, _ int) ([]*entity.Task, error) {
			return []*entity.Task{task}, nil
		},
	}

	// First call to produce (delivery) fails, second call (dead letter) succeeds
	callCount := 0
	producer := &mockProducer{
		produceFunc: func(_ context.Context, _ entity.Destination, _, _ []byte) error {
			callCount++
			if callCount == 1 {
				return errors.New("kafka down")
			}
			return nil
		},
	}
	logger := zap.NewNop()

	svc := NewTaskService(scheduler, producer, logger)
	err := svc.ProcessDueTasks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have attempted delivery (failed) + dead letter (succeeded) = 2 calls
	if len(producer.produceCalls) != 2 {
		t.Fatalf("expected 2 produce calls, got %d", len(producer.produceCalls))
	}

	// First call: delivery attempt (failed)
	if producer.produceCalls[0].Destination.Topic != "my-topic" {
		t.Fatalf("expected first call to my-topic, got %s", producer.produceCalls[0].Destination.Topic)
	}
	if producer.produceCalls[0].Err == nil {
		t.Fatal("expected first call to have error")
	}

	// Second call: dead letter (succeeded)
	if producer.produceCalls[1].Destination.Topic != "dead-topic" {
		t.Fatalf("expected second call to dead-topic, got %s", producer.produceCalls[1].Destination.Topic)
	}
	if producer.produceCalls[1].Err != nil {
		t.Fatalf("expected second call to succeed, got error: %v", producer.produceCalls[1].Err)
	}

	// Should NOT have been rescheduled
	if len(scheduler.scheduledTasks) != 0 {
		t.Fatalf("expected 0 scheduled retries, got %d", len(scheduler.scheduledTasks))
	}
}

func TestTaskService_ProcessDueTasks_noDeadLetterWhenNotConfigured(t *testing.T) {
	task := testTask()
	task.Attempt = 3
	task.MaxRetries = 3
	task.DeadDestination = entity.Destination{} // No dead letter topic

	scheduler := &mockScheduler{
		fetchDueFunc: func(_ context.Context, _ int) ([]*entity.Task, error) {
			return []*entity.Task{task}, nil
		},
	}
	producer := &mockProducer{
		produceFunc: func(_ context.Context, _ entity.Destination, _, _ []byte) error {
			return errors.New("kafka down")
		},
	}
	logger := zap.NewNop()

	svc := NewTaskService(scheduler, producer, logger)
	err := svc.ProcessDueTasks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only 1 produce call (failed delivery), no dead letter attempt
	if len(producer.produceCalls) != 1 {
		t.Fatalf("expected 1 produce call, got %d", len(producer.produceCalls))
	}
}

func TestTaskService_ProcessDueTasks_httpDelivery(t *testing.T) {
	tests := []struct {
		name                string
		task                *entity.Task
		produceErr          error
		wantSuccessProduced int
		wantTotalProduced   int
		wantScheduled       int
		wantDestinationURL  string
	}{
		{
			name:                "http task delivered successfully",
			task:                testHTTPTask(),
			wantSuccessProduced: 1,
			wantTotalProduced:   1,
			wantScheduled:       0,
			wantDestinationURL:  "http://localhost:8090/webhook",
		},
		{
			name: "http task delivery failure triggers retry scheduling",
			task: testHTTPTask(),
			produceErr:          errors.New("connection refused"),
			wantSuccessProduced: 0,
			wantTotalProduced:   1,
			wantScheduled:       1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := &mockScheduler{
				fetchDueFunc: func(_ context.Context, _ int) ([]*entity.Task, error) {
					return []*entity.Task{tt.task}, nil
				},
			}
			producer := &mockProducer{}
			if tt.produceErr != nil {
				producer.produceFunc = func(_ context.Context, _ entity.Destination, _, _ []byte) error {
					return tt.produceErr
				}
			}

			svc := NewTaskService(scheduler, producer, zap.NewNop())
			if err := svc.ProcessDueTasks(context.Background()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(producer.successfulProduceCalls()) != tt.wantSuccessProduced {
				t.Fatalf("expected %d successful produces, got %d", tt.wantSuccessProduced, len(producer.successfulProduceCalls()))
			}
			if len(producer.produceCalls) != tt.wantTotalProduced {
				t.Fatalf("expected %d total produce calls, got %d", tt.wantTotalProduced, len(producer.produceCalls))
			}
			if len(scheduler.scheduledTasks) != tt.wantScheduled {
				t.Fatalf("expected %d scheduled retries, got %d", tt.wantScheduled, len(scheduler.scheduledTasks))
			}
			if tt.wantDestinationURL != "" && producer.produceCalls[0].Destination.URL != tt.wantDestinationURL {
				t.Fatalf("expected destination URL %q, got %q", tt.wantDestinationURL, producer.produceCalls[0].Destination.URL)
			}
		})
	}
}

func TestTaskService_ProcessDueTasks_httpDeadLetter(t *testing.T) {
	task := testHTTPTask()
	task.Attempt = 3
	task.MaxRetries = 3

	callCount := 0
	producer := &mockProducer{
		produceFunc: func(_ context.Context, _ entity.Destination, _, _ []byte) error {
			callCount++
			if callCount == 1 {
				return errors.New("endpoint down")
			}
			return nil
		},
	}
	scheduler := &mockScheduler{
		fetchDueFunc: func(_ context.Context, _ int) ([]*entity.Task, error) {
			return []*entity.Task{task}, nil
		},
	}

	svc := NewTaskService(scheduler, producer, zap.NewNop())
	if err := svc.ProcessDueTasks(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// delivery attempt (failed) + dead letter (succeeded) = 2 calls
	if len(producer.produceCalls) != 2 {
		t.Fatalf("expected 2 produce calls, got %d", len(producer.produceCalls))
	}
	if producer.produceCalls[0].Destination.URL != "http://localhost:8090/webhook" {
		t.Fatalf("expected first call to webhook URL, got %q", producer.produceCalls[0].Destination.URL)
	}
	if producer.produceCalls[1].Destination.URL != "http://localhost:8090/dead" {
		t.Fatalf("expected second call to dead-letter URL, got %q", producer.produceCalls[1].Destination.URL)
	}
	if len(scheduler.scheduledTasks) != 0 {
		t.Fatalf("expected 0 scheduled retries, got %d", len(scheduler.scheduledTasks))
	}
}

func TestTaskService_ProcessDueTasks_httpNoDeadLetterWhenNotConfigured(t *testing.T) {
	task := testHTTPTask()
	task.Attempt = 3
	task.MaxRetries = 3
	task.DeadDestination = entity.Destination{} // no URL, no topic

	scheduler := &mockScheduler{
		fetchDueFunc: func(_ context.Context, _ int) ([]*entity.Task, error) {
			return []*entity.Task{task}, nil
		},
	}
	producer := &mockProducer{
		produceFunc: func(_ context.Context, _ entity.Destination, _, _ []byte) error {
			return errors.New("endpoint down")
		},
	}

	svc := NewTaskService(scheduler, producer, zap.NewNop())
	if err := svc.ProcessDueTasks(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only 1 produce call (failed delivery), no dead letter attempt
	if len(producer.produceCalls) != 1 {
		t.Fatalf("expected 1 produce call, got %d", len(producer.produceCalls))
	}
}
