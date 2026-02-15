package entity

import (
	"testing"
	"time"
)

func TestTask_IncrementAttempt(t *testing.T) {
	task := &Task{Attempt: 0}
	task.IncrementAttempt()
	if task.Attempt != 1 {
		t.Fatalf("expected attempt 1, got %d", task.Attempt)
	}
}

func TestTask_HasRetriesLeft(t *testing.T) {
	tests := []struct {
		name       string
		attempt    int
		maxRetries int
		want       bool
	}{
		{
			name:       "first attempt with retries available",
			attempt:    0,
			maxRetries: 3,
			want:       true,
		},
		{
			name:       "at max retries",
			attempt:    3,
			maxRetries: 3,
			want:       true,
		},
		{
			name:       "exceeded max retries",
			attempt:    4,
			maxRetries: 3,
			want:       false,
		},
		{
			name:       "no retries configured",
			attempt:    1,
			maxRetries: 0,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				Attempt:    tt.attempt,
				MaxRetries: tt.maxRetries,
			}
			got := task.HasRetriesLeft()
			if got != tt.want {
				t.Fatalf("HasRetriesLeft() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTask_NextRetryDelay(t *testing.T) {
	tests := []struct {
		name      string
		attempt   int
		baseDelay int
		want      time.Duration
	}{
		{
			name:      "first retry",
			attempt:   1,
			baseDelay: 2,
			want:      2 * time.Second,
		},
		{
			name:      "second retry",
			attempt:   2,
			baseDelay: 2,
			want:      4 * time.Second,
		},
		{
			name:      "third retry",
			attempt:   3,
			baseDelay: 2,
			want:      8 * time.Second,
		},
		{
			name:      "zero attempt uses exponent 0",
			attempt:   0,
			baseDelay: 1,
			want:      1 * time.Second,
		},
		{
			name:      "base delay of 1",
			attempt:   4,
			baseDelay: 1,
			want:      8 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				Attempt:   tt.attempt,
				BaseDelay: tt.baseDelay,
			}
			got := task.NextRetryDelay()
			if got != tt.want {
				t.Fatalf("NextRetryDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTask_ShouldSendToDeadDestination(t *testing.T) {
	tests := []struct {
		name       string
		attempt    int
		maxRetries int
		want       bool
	}{
		{
			name:       "retries available",
			attempt:    1,
			maxRetries: 3,
			want:       false,
		},
		{
			name:       "retries exhausted",
			attempt:    4,
			maxRetries: 3,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				Attempt:    tt.attempt,
				MaxRetries: tt.maxRetries,
			}
			got := task.ShouldSendToDeadDestination()
			if got != tt.want {
				t.Fatalf("ShouldSendToDeadDestination() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDestination_Address(t *testing.T) {
	d := Destination{Host: "localhost", Port: "9092"}
	want := "localhost:9092"
	if got := d.Address(); got != want {
		t.Fatalf("Address() = %q, want %q", got, want)
	}
}
