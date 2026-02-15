package domain

import "errors"

var (
	// ErrTaskNotFound indicates the requested task does not exist.
	ErrTaskNotFound = errors.New("task not found")

	// ErrInvalidTask indicates the task failed validation.
	ErrInvalidTask = errors.New("invalid task")

	// ErrScheduleFailed indicates a failure when scheduling a task for retry.
	ErrScheduleFailed = errors.New("failed to schedule task")

	// ErrDeliveryFailed indicates the message could not be delivered to the destination.
	ErrDeliveryFailed = errors.New("delivery failed")

	// ErrMaxRetriesExceeded indicates the task exhausted all retry attempts.
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)
