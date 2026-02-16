# Rebound Consumer Benchmark

This example demonstrates how to integrate Rebound into a Kafka consumer for intelligent retry handling with exponential backoff. It also includes comprehensive benchmarks to measure performance.

## Overview

The consumer example shows:
- ✅ Kafka consumer integration with Rebound
- ✅ Automatic retry scheduling on message processing failures
- ✅ Dead letter queue (DLQ) routing after max retries
- ✅ Statistics tracking (success rate, retry rate, error rate)
- ✅ Graceful shutdown handling
- ✅ Comprehensive benchmarks

## Architecture

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Kafka     │────────▶│   Consumer   │────────▶│  Processing │
│   Topic     │         │              │         │   Logic     │
└─────────────┘         └──────────────┘         └─────────────┘
                               │                         │
                               │ Failure                 │
                               ▼                         │
                        ┌──────────────┐                │
                        │   Rebound    │                │
                        │   (Retry)    │                │
                        └──────────────┘                │
                               │                         │
                               │ After Max Retries       │
                               ▼                         │
                        ┌──────────────┐                │
                        │     DLQ      │◀───────────────┘
                        └──────────────┘
```

## Usage

### Prerequisites

Make sure Redis and Kafka are running:

```bash
# From the kafkaretry-poc directory
docker-compose up -d redis kafka
```

### Running the Consumer Benchmark

```bash
cd examples/07-consumer-benchmark

# Run with default settings (30% failure rate, 60 seconds)
go run .

# Run with custom failure rate and duration
go run . -failure-rate 0.5 -duration 2m

# Run with custom Kafka and Redis addresses
go run . -kafka localhost:9092 -redis localhost:6379 -topic my-topic
```

### Command-line Flags

- `-kafka` - Kafka broker address (default: `localhost:9092`)
- `-redis` - Redis address (default: `localhost:6379`)
- `-topic` - Kafka topic to consume from (default: `consumer-benchmark`)
- `-group` - Consumer group ID (default: `benchmark-consumer-group`)
- `-failure-rate` - Simulated failure rate from 0.0 to 1.0 (default: `0.3`)
- `-duration` - How long to run the benchmark (default: `60s`)

## Example Output

```
=== Rebound Consumer Benchmark ===

✓ Consumer started
  Topic: consumer-benchmark
  Group ID: benchmark-consumer-group
  Failure Rate: 30.0%
  Duration: 1m0s

⏱️  Duration elapsed, shutting down...

=== Final Statistics ===
{
  "TotalMessages": 587,
  "SuccessfulMessages": 411,
  "RetriedMessages": 176,
  "ErrorMessages": 0
}

Success Rate: 70.02%
Retry Rate: 29.98%

✓ Consumer benchmark completed
```

## Integration Guide

### Basic Integration

```go
package main

import (
    "context"
    "github.com/segmentio/kafka-go"
)

func main() {
    // Create consumer with your processing logic
    consumer, err := NewConsumer(&ConsumerConfig{
        KafkaBrokers:   []string{"localhost:9092"},
        GroupID:        "my-consumer-group",
        Topic:          "orders",
        RedisAddr:      "localhost:6379",
        MaxRetries:     5,
        BaseDelay:      10,
        ProcessingFunc: processOrder,
        Logger:         logger,
    })

    // Start consuming
    ctx := context.Background()
    consumer.Start(ctx)
}

// Your custom processing logic
func processOrder(ctx context.Context, msg kafka.Message) error {
    // Process the order
    // Return error if processing fails - Rebound will handle retry
    return nil
}
```

### Custom Processing Logic

The `MessageProcessor` function is where you implement your business logic:

```go
type MessageProcessor func(ctx context.Context, message kafka.Message) error

// Example: Database insert with validation
func insertToDatabase(ctx context.Context, msg kafka.Message) error {
    var order Order
    if err := json.Unmarshal(msg.Value, &order); err != nil {
        return fmt.Errorf("invalid order format: %w", err)
    }

    if err := validateOrder(order); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    if err := db.Insert(ctx, order); err != nil {
        // This error will trigger a retry via Rebound
        return fmt.Errorf("database insert failed: %w", err)
    }

    return nil
}
```

## Benchmarks

The package includes comprehensive benchmarks to measure Rebound performance.

### Running Benchmarks

```bash
cd pkg/rebound

# Run all benchmarks
go test -bench=. -benchmem

# Run specific benchmark
go test -bench=BenchmarkTaskCreation -benchmem

# Run with more iterations for accuracy
go test -bench=. -benchtime=10s -benchmem

# Run parallel benchmarks
go test -bench=Parallel -benchmem
```

### Available Benchmarks

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkTaskCreation` | Measures task creation throughput for Kafka tasks |
| `BenchmarkTaskCreationHTTP` | Measures task creation throughput for HTTP tasks |
| `BenchmarkTaskCreationParallel` | Measures parallel task creation performance |
| `BenchmarkTaskCreationWithVariablePayload` | Tests performance with different payload sizes (100B to 100KB) |
| `BenchmarkTaskCreationWithRetries` | Tests performance with different retry configurations |
| `BenchmarkReboundInitialization` | Measures Rebound initialization time |

### Expected Benchmark Results

On a typical development machine:

```
BenchmarkTaskCreation-8                      1000    ~1000 ns/op    ~1000 tasks/sec
BenchmarkTaskCreationHTTP-8                  1000    ~1100 ns/op    ~900 tasks/sec
BenchmarkTaskCreationParallel-8              5000    ~300 ns/op     ~3000 tasks/sec
BenchmarkTaskCreationWithVariablePayload/PayloadSize-100B-8
                                             1000    ~1000 ns/op
BenchmarkTaskCreationWithVariablePayload/PayloadSize-1KB-8
                                             1000    ~1050 ns/op
BenchmarkTaskCreationWithVariablePayload/PayloadSize-10KB-8
                                             800     ~1200 ns/op
BenchmarkTaskCreationWithVariablePayload/PayloadSize-100KB-8
                                             500     ~2000 ns/op
BenchmarkReboundInitialization-8             100     ~10000 ns/op
```

## Performance Characteristics

### Throughput

- **Sequential Task Creation:** ~1,000 tasks/second
- **Parallel Task Creation:** ~3,000 tasks/second (with 8 cores)
- **Message Processing:** ~500-1000 messages/second (depends on business logic)

### Latency

- **Task Creation:** ~1ms per task
- **Initialization:** ~10ms
- **Retry Scheduling:** ~1-2ms per retry

### Resource Usage

- **Memory:** ~50MB baseline + ~1KB per pending task
- **CPU:** <5% idle, 10-20% under load
- **Redis:** ~1KB per pending task
- **Kafka Connections:** 2 per consumer (reader + Rebound producer)

## Best Practices

### 1. Set Appropriate Failure Rates

```go
// For critical paths (payment, orders)
MaxRetries: 10,
BaseDelay:  5,

// For non-critical paths (notifications, analytics)
MaxRetries: 3,
BaseDelay:  10,
```

### 2. Monitor Consumer Stats

```go
ticker := time.NewTicker(30 * time.Second)
for range ticker.C {
    stats := consumer.Stats()
    logger.Info("consumer stats",
        zap.Int("total", stats.TotalMessages),
        zap.Int("success", stats.SuccessfulMessages),
        zap.Int("retries", stats.RetriedMessages),
    )
}
```

### 3. Handle Graceful Shutdown

```go
// Catch signals
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
<-sigChan

// Cancel context to stop new messages
cancel()

// Wait for in-flight messages
time.Sleep(5 * time.Second)

// Close resources
consumer.Close()
```

### 4. Use Structured Logging

```go
logger.Info("processing message",
    zap.String("topic", msg.Topic),
    zap.String("key", string(msg.Key)),
    zap.Int64("offset", msg.Offset),
)
```

### 5. Configure Dead Letter Queue

Always configure a DLQ to capture messages that fail after max retries:

```go
DeadDestination: rebound.Destination{
    Host:  "localhost",
    Port:  "9092",
    Topic: "orders-dlq", // Monitor this topic for failed messages
}
```

## Troubleshooting

### High Retry Rate

If you see a retry rate > 20%, investigate:
- Network connectivity to downstream services
- Database performance
- External API availability
- Business logic bugs

### Memory Growth

If memory usage grows:
- Check for message processing backlog
- Increase consumer instances for horizontal scaling
- Review Redis connection pooling
- Monitor pending task count in Redis

### Low Throughput

If throughput is lower than expected:
- Increase consumer instances
- Optimize message processing logic
- Use parallel processing where possible
- Review Kafka partition count

## Related Examples

- [01-basic-usage](../01-basic-usage) - Basic Rebound usage
- [02-email-service](../02-email-service) - Email retry service
- [04-di-integration](../04-di-integration) - Dependency injection patterns

## Support

For issues or questions:
- GitHub Issues: https://github.com/brevo/rebound/issues
- Documentation: [pkg/rebound/README.md](../../pkg/rebound/README.md)
