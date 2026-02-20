package secondary

import (
	"context"
	"time"

	"github.com/ruudy-sib/rebound/internal/domain/entity"
)

// TaskScheduler defines the secondary port for scheduling and retrieving
// tasks from a time-based queue (e.g., Redis sorted set).
type TaskScheduler interface {
	// Schedule adds a task to the queue with the given delay from now.
	Schedule(ctx context.Context, task *entity.Task, delay time.Duration) error

	// FetchDue retrieves up to limit tasks whose scheduled time has passed.
	FetchDue(ctx context.Context, limit int) ([]*entity.Task, error)

	// Remove removes a task from the queue. The raw member is used for
	// exact match removal from the sorted set.
	Remove(ctx context.Context, rawMember string) error
}
