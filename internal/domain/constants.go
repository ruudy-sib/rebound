package domain

import "time"

const (
	// RedisRetryKey is the sorted set key used for scheduling tasks.
	RedisRetryKey = "retry:schedule:"

	// DefaultPollInterval is the interval between worker polling cycles.
	DefaultPollInterval = 1 * time.Second

	// DefaultBatchSize is the maximum number of tasks fetched per poll cycle.
	DefaultBatchSize = 10

	// MaxBaseDelay caps the base delay to prevent excessively long waits.
	MaxBaseDelay = 3600

	// MinBaseDelay ensures a minimum delay between retries.
	MinBaseDelay = 1

	// MaxRetryLimit caps the maximum number of retries allowed.
	MaxRetryLimit = 100
)
