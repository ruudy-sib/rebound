package entity

import (
	"math"
	"time"
)

// DestinationType defines the type of message destination.
type DestinationType string

const (
	DestinationTypeKafka DestinationType = "kafka"
)

// Task represents a retryable unit of work that delivers a message
// to a destination with exponential backoff on failure.
type Task struct {
	ID              string
	Attempt         int
	Source          string
	Destination     Destination
	DeadDestination Destination
	MaxRetries      int
	BaseDelay       int
	ClientID        string
	IsPriority      bool
	MessageData     string
	DestinationType DestinationType
}

// IncrementAttempt advances the attempt counter by one.
func (t *Task) IncrementAttempt() {
	t.Attempt++
}

// HasRetriesLeft reports whether the task can be retried.
func (t *Task) HasRetriesLeft() bool {
	return t.Attempt <= t.MaxRetries
}

// NextRetryDelay calculates the exponential backoff delay for the current attempt.
// Formula: baseDelay * 2^(attempt-1)
func (t *Task) NextRetryDelay() time.Duration {
	exponent := float64(t.Attempt - 1)
	if exponent < 0 {
		exponent = 0
	}
	multiplier := math.Pow(2, exponent)
	return time.Duration(float64(t.BaseDelay)*multiplier) * time.Second
}

// ShouldSendToDeadDestination reports whether the task has exhausted
// all retries and should be routed to its dead-letter destination.
func (t *Task) ShouldSendToDeadDestination() bool {
	return !t.HasRetriesLeft()
}
