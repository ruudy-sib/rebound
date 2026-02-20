---
name: implement-rebound-retry
description: "Implement Rebound retry orchestration for intelligent failure handling with exponential backoff, dead letter queues, and support for Kafka/HTTP destinations. Use when adding retry logic, handling transient failures, implementing circuit breakers, or building resilient services. Activates on: retry, rebound, exponential backoff, dead letter queue, dlq, failure handling, resilience."
---

# Implement Rebound Retry Orchestration

Build resilient applications with intelligent retry mechanisms using Rebound - a Go library for retry orchestration with exponential backoff and dead letter queues.

**Repository**: [ruudy-sib/rebound](https://github.com/ruudy-sib/rebound) | **Package**: `github.com/ruudy-sib/rebound/pkg/rebound` | **Version**: `v1.2.3` | **References**: [pkg/rebound/README.md](https://github.com/ruudy-sib/rebound/blob/v1.2.3/pkg/rebound/README.md)

## Overview

Rebound provides:
- 🔄 **Exponential Backoff** - Intelligent retry scheduling with configurable delays
- 📊 **Multiple Destinations** - Kafka topics and HTTP webhooks
- ☠️ **Dead Letter Queue** - Automatic routing of exhausted retries
- 🏗️ **Hexagonal Architecture** - Clean separation of concerns
- 💉 **Dependency Injection** - First-class support for uber-go/dig
- 📝 **Structured Logging** - Built on uber-go/zap

## Installation

```bash
go get github.com/ruudy-sib/rebound/pkg/rebound@v1.2.3
```

## Quick Start: Basic Integration

**Location**: Your service initialization code

```go
package main

import (
    "context"
    "time"

    "github.com/ruudy-sib/rebound/pkg/rebound"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    // Configure Rebound
    cfg := &rebound.Config{
        RedisAddr:    "localhost:6379",
        PollInterval: 1 * time.Second,
        Logger:       logger,
        // RedisMode: "sentinel", // or "cluster" — default is "standalone"
    }

    // Create instance
    rb, err := rebound.New(cfg)
    if err != nil {
        logger.Fatal("failed to create rebound", zap.Error(err))
    }
    defer rb.Close()

    // Start worker
    ctx := context.Background()
    if err := rb.Start(ctx); err != nil {
        logger.Fatal("failed to start rebound", zap.Error(err))
    }

    // Schedule a retry task
    task := &rebound.Task{
        ID:     "order-123",
        Source: "order-service",
        Destination: rebound.Destination{
            Host:  "kafka.prod",
            Port:  "9092",
            Topic: "orders",
        },
        DeadDestination: rebound.Destination{
            Host:  "kafka.prod",
            Port:  "9092",
            Topic: "orders-dlq",
        },
        MaxRetries:      5,
        BaseDelay:       10, // seconds
        ClientID:        "order-service",
        MessageData:     `{"order_id": 123}`,
        DestinationType: rebound.DestinationTypeKafka,
    }

    if err := rb.CreateTask(ctx, task); err != nil {
        logger.Error("failed to create task", zap.Error(err))
    }
}
```

## Integration Patterns

### Pattern 1: HTTP Handler Integration

Use Rebound to retry failed HTTP operations:

```go
package handler

import (
    "net/http"
    "github.com/ruudy-sib/rebound/pkg/rebound"
    "go.uber.org/zap"
)

type OrderHandler struct {
    rebound *rebound.Rebound
    logger  *zap.Logger
}

func (h *OrderHandler) ProcessOrder(w http.ResponseWriter, r *http.Request) {
    // Try processing
    if err := h.processOrderLogic(r); err != nil {
        // Schedule retry on failure
        task := &rebound.Task{
            ID:     fmt.Sprintf("order-%s", orderID),
            Source: "order-service",
            Destination: rebound.Destination{
                URL: "https://payment-gateway.com/process",
            },
            DeadDestination: rebound.Destination{
                URL: "https://payment-gateway.com/dlq",
            },
            MaxRetries:      3,
            BaseDelay:       5,
            ClientID:        "order-service",
            MessageData:     orderJSON,
            DestinationType: rebound.DestinationTypeHTTP,
        }

        if err := h.rebound.CreateTask(r.Context(), task); err != nil {
            h.logger.Error("failed to schedule retry", zap.Error(err))
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusAccepted)
        return
    }

    w.WriteHeader(http.StatusOK)
}
```

### Pattern 2: Kafka Consumer Integration

Retry failed Kafka messages:

```go
package consumer

import (
    "context"
    "github.com/segmentio/kafka-go"
    "github.com/ruudy-sib/rebound/pkg/rebound"
)

type Consumer struct {
    reader  *kafka.Reader
    rebound *rebound.Rebound
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) {
    if err := c.handleMessage(msg); err != nil {
        // Schedule retry with Rebound
        task := &rebound.Task{
            ID:     fmt.Sprintf("%s-%d-%d", msg.Topic, msg.Partition, msg.Offset),
            Source: "kafka-consumer",
            Destination: rebound.Destination{
                Host:  "localhost",
                Port:  "9092",
                Topic: msg.Topic,
            },
            DeadDestination: rebound.Destination{
                Host:  "localhost",
                Port:  "9092",
                Topic: msg.Topic + "-dlq",
            },
            MaxRetries:      5,
            BaseDelay:       10,
            ClientID:        "consumer-service",
            MessageData:     string(msg.Value),
            DestinationType: rebound.DestinationTypeKafka,
        }

        c.rebound.CreateTask(ctx, task)
    }
}
```

### Pattern 3: Dependency Injection (Recommended)

Production-ready pattern with uber-go/dig:

```go
package main

import (
    "github.com/DTSL/golang-libraries/di"
    "github.com/ruudy-sib/rebound/pkg/rebound"
    "go.uber.org/zap"
)

func main() {
    container := di.GetContainer()

    // Provide configuration
    container.Provide(func(logger *zap.Logger) *rebound.Config {
        return &rebound.Config{
            RedisAddr:    "redis.production.svc.cluster.local:6379",
            PollInterval: 1 * time.Second,
            Logger:       logger,
            // RedisMode: "sentinel", // uncomment for HA
        }
    })

    // Register Rebound
    rebound.RegisterWithContainer(container)

    // Use in your handlers
    container.Invoke(func(rb *rebound.Rebound, logger *zap.Logger) error {
        ctx := context.Background()
        return rb.Start(ctx)
    })
}
```

## Destination Types

## Redis Configuration

Rebound supports three Redis deployment modes via the `RedisMode` config field:

| Mode | Config Fields | Use Case |
|------|--------------|----------|
| `"standalone"` (default) | `RedisAddr` | Single instance, dev/simple prod |
| `"sentinel"` | `RedisMasterName`, `RedisSentinelAddrs` | High availability with automatic failover |
| `"cluster"` | `RedisClusterAddrs` | Horizontal scaling across multiple nodes |

```go
// Standalone (default)
cfg := &rebound.Config{RedisAddr: "localhost:6379"}

// Sentinel
cfg := &rebound.Config{
    RedisMode:          "sentinel",
    RedisMasterName:    "mymaster",
    RedisSentinelAddrs: []string{"sentinel-1:26379", "sentinel-2:26379"},
}

// Cluster
cfg := &rebound.Config{
    RedisMode:         "cluster",
    RedisClusterAddrs: []string{"node-1:7000", "node-2:7001", "node-3:7002"},
}
```

### Kafka Destinations

```go
task := &rebound.Task{
    Destination: rebound.Destination{
        Host:  "kafka.prod",
        Port:  "9092",
        Topic: "events",
    },
    DestinationType: rebound.DestinationTypeKafka,
}
```

**Behavior:**
- Messages sent via Kafka producer
- Key format: `{task_id}|{attempt}`
- Value: Raw message data

### HTTP Destinations

```go
task := &rebound.Task{
    Destination: rebound.Destination{
        URL: "https://api.partner.com/webhook",
    },
    DestinationType: rebound.DestinationTypeHTTP,
}
```

**Behavior:**
- HTTP POST request
- Headers: `Content-Type: application/json`, `X-Message-Key: {task_id}|{attempt}`
- Success: 2xx status codes
- Failure: Non-2xx triggers retry

## Retry Strategy Configuration

### Exponential Backoff Formula

```
delay = base_delay * (2 ^ (attempt - 1))
```

**Example with base_delay=10s:**
- Attempt 1: 10s
- Attempt 2: 20s
- Attempt 3: 40s
- Attempt 4: 80s
- Attempt 5: 160s

### Retry Strategy by Use Case

```go
// Critical operations (payments, orders)
MaxRetries: 10,
BaseDelay:  5,  // Retry quickly

// Rate-limited APIs
MaxRetries: 5,
BaseDelay:  60, // Wait longer between retries

// Non-critical operations (notifications)
MaxRetries: 3,
BaseDelay:  30,

// Transient network errors
MaxRetries: 7,
BaseDelay:  10,
```

## Performance Characteristics

Based on benchmarks (Apple M4 Pro):

| Metric | Performance |
|--------|-------------|
| **Sequential Task Creation** | 4,700 tasks/sec |
| **Parallel Task Creation** | 27,600 tasks/sec |
| **Task Creation Latency** | 214 μs |
| **Memory per Task** | 1.4 KB |
| **Retry Config Overhead** | ~0% (negligible) |

**Recommendations:**
- Use parallel task creation for high throughput (27K+ tasks/sec)
- Keep payloads under 10KB for optimal latency (<300μs)
- Use high retry counts without worry - no performance penalty
- Initialize once at startup (only 1.2ms overhead)

## Best Practices

| Do | Don't |
|----|-------|
| Always configure dead letter destination | Ignore exhausted retries |
| Use meaningful task IDs (traceable) | Use generic IDs like "task-1" |
| Include context in message data | Store minimal information |
| Register with DI container for lifecycle | Create new instances per request |
| Handle graceful shutdown | Force close connections |
| Use appropriate retry counts per use case | Use same config for all |
| Monitor dead letter queues | Forget about failed tasks |
| Keep payloads under 10KB | Store large data inline |

## Error Handling

```go
// Good: Check errors and handle appropriately
if err := rb.CreateTask(ctx, task); err != nil {
    logger.Error("failed to schedule retry",
        zap.Error(err),
        zap.String("task_id", task.ID),
    )
    // Decide: fail fast or continue without retry?
    return err
}

// Bad: Ignore errors
rb.CreateTask(ctx, task) // ❌ Error ignored
```

## HTTP API (Alternative to Go SDK)

Rebound exposes a REST API for creating tasks from any language or service.

### POST /tasks

Create a retry task via HTTP. Use `url` in `destination` for HTTP-type destinations (**added in v1.2.3**):

```bash
# Kafka destination
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "order-123",
    "source": "order-service",
    "destination": {
      "host": "kafka.prod",
      "port": "9092",
      "topic": "orders"
    },
    "dead_destination": {
      "host": "kafka.prod",
      "port": "9092",
      "topic": "orders-dlq"
    },
    "max_retries": 5,
    "base_delay": 10,
    "client_id": "order-service",
    "message_data": "{\"order_id\": 123}",
    "destination_type": "kafka"
  }'

# HTTP destination (url field required)
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "webhook-456",
    "source": "payment-service",
    "destination": {
      "url": "https://api.partner.com/webhook"
    },
    "dead_destination": {
      "url": "https://api.partner.com/dlq"
    },
    "max_retries": 3,
    "base_delay": 5,
    "client_id": "payment-service",
    "message_data": "{\"payment_id\": 456}",
    "destination_type": "http"
  }'
```

### GET /health

```bash
curl http://localhost:8080/health
# {"status":"ok","checks":{"redis":"ok"}}
```

**`DestinationDTO` schema** (both `destination` and `dead_destination`):

| Field | Type | Used by |
|-------|------|---------|
| `host` | string | Kafka |
| `port` | string | Kafka |
| `topic` | string | Kafka |
| `url` | string | HTTP *(added v1.2.3)* |

## Monitoring

### Structured Logging

Rebound logs all operations with rich fields for retry tracing (**enhanced in v1.2.3**):

```json
{
  "level": "info",
  "msg": "task saved to redis",
  "task_id": "order-123",
  "destination_type": "http",
  "destination_url": "https://api.partner.com/webhook",
  "destination_topic": "",
  "attempt": 1,
  "delay": "10s",
  "score": 1700000010
}
```

```json
{
  "level": "info",
  "msg": "scheduling retry",
  "destination_type": "http",
  "destination_url": "https://api.partner.com/webhook",
  "destination_topic": "",
  "delay": "20s",
  "next_attempt": 2
}
```

### Recommended Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    tasksCreated = promauto.NewCounter(prometheus.CounterOpts{
        Name: "rebound_tasks_created_total",
        Help: "Total retry tasks created",
    })

    tasksCompleted = promauto.NewCounter(prometheus.CounterOpts{
        Name: "rebound_tasks_completed_total",
        Help: "Total retry tasks completed",
    })

    tasksDead = promauto.NewCounter(prometheus.CounterOpts{
        Name: "rebound_tasks_dead_total",
        Help: "Total tasks sent to dead letter queue",
    })
)
```

## Testing

### Unit Tests with Mock

```go
func TestMyService(t *testing.T) {
    mockRebound := &MockRebound{}
    service := NewMyService(mockRebound)

    err := service.ProcessOrder(ctx, order)
    assert.NoError(t, err)
    assert.Equal(t, 1, mockRebound.CreateTaskCalls)
}
```

### Integration Tests

```go
func TestReboundIntegration(t *testing.T) {
    // Use test Redis
    cfg := &rebound.Config{
        RedisAddr: "localhost:6379",
        RedisDB:   15, // Test database
    }

    rb, err := rebound.New(cfg)
    require.NoError(t, err)
    defer rb.Close()

    // Test task creation
    task := &rebound.Task{...}
    err = rb.CreateTask(context.Background(), task)
    assert.NoError(t, err)
}
```

## Troubleshooting

### Tasks Not Processing

**Check:**
1. Worker is started: `rb.Start(ctx)`
2. Redis connectivity: `redis-cli -h localhost PING`
3. Poll interval not too high

**Fix:**
```go
// Ensure worker is started
rb.Start(ctx)

// Verify Redis connection
if err := redisClient.Ping(ctx).Err(); err != nil {
    logger.Fatal("redis unavailable")
}
```

### High Retry Rate

**Check:**
1. Destination availability
2. Network connectivity
3. Base delay configuration

**Fix:**
```go
// Increase base delay for unstable destinations
BaseDelay: 30, // Wait 30s instead of 5s
```

## Examples

See comprehensive examples in `examples/`:
- [01-basic-usage](https://github.com/ruudy-sib/rebound/tree/v1.2.3/examples/01-basic-usage) - Simple setup
- [02-email-service](https://github.com/ruudy-sib/rebound/tree/v1.2.3/examples/02-email-service) - Email retry
- [04-di-integration](https://github.com/ruudy-sib/rebound/tree/v1.2.3/examples/04-di-integration) - Production DI pattern
- [07-consumer-benchmark](https://github.com/ruudy-sib/rebound/tree/v1.2.3/examples/07-consumer-benchmark) - Kafka consumer + benchmarks

## Checklist

- [ ] Rebound configuration loaded from environment
- [ ] Worker started with `rb.Start(ctx)`
- [ ] Task IDs are unique and traceable
- [ ] Dead letter destination configured for all tasks
- [ ] Retry strategy appropriate for use case (critical vs non-critical)
- [ ] Graceful shutdown implemented (context cancellation + Close)
- [ ] Error handling for CreateTask failures
- [ ] Monitoring/logging configured
- [ ] Integration tests written
- [ ] Payloads kept under 10KB (or stored externally)
- [ ] Code compiles: `go build ./...`
- [ ] Benchmarks run successfully: `go test -bench=. github.com/ruudy-sib/rebound/pkg/rebound`

## Performance Tuning

### For High Throughput

```go
// Use parallel task creation
var wg sync.WaitGroup
for _, task := range tasks {
    wg.Add(1)
    go func(t *rebound.Task) {
        defer wg.Done()
        rb.CreateTask(ctx, t)
    }(task)
}
wg.Wait()
```

### For Low Latency

```go
// Keep payloads small
task.MessageData = compressData(data) // <1KB

// Use local Redis
cfg.RedisAddr = "localhost:6379" // Not remote
```

### For Memory Efficiency

```go
// Store large payloads externally
s3Key := uploadToS3(largeData)
task.MessageData = fmt.Sprintf(`{"s3_key": "%s"}`, s3Key)
```

## Migration from Manual Retry

**Before (manual retry):**
```go
func processWithRetry(data []byte) error {
    for attempt := 0; attempt < 3; attempt++ {
        if err := process(data); err == nil {
            return nil
        }
        time.Sleep(time.Duration(attempt+1) * 5 * time.Second)
    }
    return errors.New("max retries exceeded")
}
```

**After (Rebound):**
```go
func processWithRebound(ctx context.Context, rb *rebound.Rebound, data []byte) error {
    if err := process(data); err != nil {
        // Rebound handles retries asynchronously
        return rb.CreateTask(ctx, &rebound.Task{
            ID:              generateID(),
            Destination:     /* your destination */,
            DeadDestination: /* your DLQ */,
            MaxRetries:      3,
            BaseDelay:       5,
            MessageData:     string(data),
        })
    }
    return nil
}
```

**Benefits:**
- ✅ Asynchronous - doesn't block caller
- ✅ Persistent - survives restarts
- ✅ Exponential backoff - intelligent spacing
- ✅ Dead letter queue - handles exhausted retries
- ✅ Observable - structured logging
- ✅ Testable - easy to mock

## Additional Resources

- **Main README**: [pkg/rebound/README.md](https://github.com/ruudy-sib/rebound/blob/v1.2.3/pkg/rebound/README.md)
- **Integration Guide**: [INTEGRATION.md](https://github.com/ruudy-sib/rebound/blob/v1.2.3/INTEGRATION.md)
- **Examples**: [examples/](https://github.com/ruudy-sib/rebound/tree/v1.2.3/examples/)
- **Benchmarks**: [examples/07-consumer-benchmark/](https://github.com/ruudy-sib/rebound/tree/v1.2.3/examples/07-consumer-benchmark/)
