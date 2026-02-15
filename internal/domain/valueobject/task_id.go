package valueobject

import (
	"fmt"
	"strings"
)

// TaskID is an immutable value object representing a unique task identifier.
type TaskID struct {
	value string
}

// NewTaskID creates a validated TaskID from a string.
func NewTaskID(value string) (TaskID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return TaskID{}, fmt.Errorf("task ID must not be empty")
	}
	return TaskID{value: trimmed}, nil
}

// String returns the string representation of the TaskID.
func (t TaskID) String() string {
	return t.value
}

// Equals checks equality with another TaskID.
func (t TaskID) Equals(other TaskID) bool {
	return t.value == other.value
}
