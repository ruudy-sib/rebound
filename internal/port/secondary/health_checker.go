package secondary

import "context"

// HealthChecker defines the secondary port for checking the health
// of an external dependency.
type HealthChecker interface {
	// Name returns the name of the dependency being checked.
	Name() string

	// Check performs a health check and returns an error if unhealthy.
	Check(ctx context.Context) error
}
