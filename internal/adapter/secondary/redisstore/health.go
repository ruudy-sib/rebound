package redisstore

import (
	"context"

	"github.com/redis/go-redis/v9"

	"github.com/ruudy-sib/rebound/internal/port/secondary"
)

// HealthCheck implements secondary.HealthChecker for Redis.
type HealthCheck struct {
	client redis.UniversalClient
}

// NewHealthCheck creates a Redis health checker.
func NewHealthCheck(client redis.UniversalClient) secondary.HealthChecker {
	return &HealthCheck{client: client}
}

// Name returns the name of this health check.
func (h *HealthCheck) Name() string {
	return "redis"
}

// Check pings Redis to verify connectivity.
func (h *HealthCheck) Check(ctx context.Context) error {
	return h.client.Ping(ctx).Err()
}
