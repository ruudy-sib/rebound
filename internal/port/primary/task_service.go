package primary

import (
	"context"

	"kafkaretry/internal/domain/entity"
)

// TaskService defines the primary port for task operations
// exposed to driving adapters (HTTP handlers, CLI, etc.).
type TaskService interface {
	// CreateTask validates and schedules a new task for immediate processing.
	CreateTask(ctx context.Context, task *entity.Task) error

	// ProcessDueTasks fetches and processes all tasks whose scheduled time has passed.
	ProcessDueTasks(ctx context.Context) error
}
