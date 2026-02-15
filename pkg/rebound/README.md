# Rebound - Resilient Retry Orchestration

Rebound is a Go library for building resilient applications with intelligent retry mechanisms. It provides exponential backoff, dead letter queues, and support for multiple destination types (Kafka, HTTP webhooks).

## Features

- 🔄 **Exponential Backoff** - Intelligent retry scheduling with configurable delays
- 📊 **Multiple Destinations** - Kafka topics and HTTP webhooks
- ☠️ **Dead Letter Queue** - Automatic routing of exhausted retries
- 🏗️ **Hexagonal Architecture** - Clean separation of concerns
- 💉 **Dependency Injection** - First-class support for uber-go/dig
- 📝 **Structured Logging** - Built on uber-go/zap
- 🧪 **High Test Coverage** - 87%+ coverage on core logic

## Installation

```bash
# Add to your go.mod (when published):
# go get github.com/your-org/rebound

# For local development:
# Replace "github.com/your-org/rebound" with your actual module path
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/brevo/rebound/pkg/rebound"
)

func main() {
    // Configure Rebound
    cfg := &rebound.Config{
        RedisAddr:    "localhost:6379",
        KafkaBrokers: []string{"localhost:9092"},
        PollInterval: 1 * time.Second,
    }

    // Create instance
    rb, err := rebound.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer rb.Close()

    // Start worker
    ctx := context.Background()
    rb.Start(ctx)

    // Create a retry task
    task := &rebound.Task{
        ID:     "order-123",
        Source: "order-service",
        Destination: rebound.Destination{
            URL: "https://api.partner.com/webhook",
        },
        DeadDestination: rebound.Destination{
            URL: "https://api.partner.com/webhook-dlq",
        },
        MaxRetries:      3,
        BaseDelay:       5,
        ClientID:        "order-service",
        MessageData:     `{"order_id": 123}`,
        DestinationType: rebound.DestinationTypeHTTP,
    }

    if err := rb.CreateTask(ctx, task); err != nil {
        log.Fatal(err)
    }
}
```

## Integration with Brevo Apps

### Using Dependency Injection (Recommended)

For apps using `golang-libraries/di`:

```go
package main

import (
    "context"

    "github.com/brevo/golang-libraries/di"
    "github.com/brevo/rebound/pkg/rebound"
    "go.uber.org/zap"
)

func main() {
    container := di.GetContainer()

    // Provide configuration
    container.Provide(func(logger *zap.Logger) *rebound.Config {
        return &rebound.Config{
            RedisAddr:    "redis.production.svc.cluster.local:6379",
            KafkaBrokers: []string{"kafka-1:9092", "kafka-2:9092"},
            PollInterval: 1 * time.Second,
            Logger:       logger,
        }
    })

    // Register Rebound
    rebound.RegisterWithContainer(container)

    // Start in your app lifecycle
    container.Invoke(func(rb *rebound.Rebound) error {
        ctx := context.Background()
        return rb.Start(ctx)
    })
}
```

### Using in HTTP Handlers

```go
package handler

import (
    "net/http"

    "github.com/brevo/rebound/pkg/rebound"
    "go.uber.org/zap"
)

type OrderHandler struct {
    rebound *rebound.Rebound
    logger  *zap.Logger
}

func NewOrderHandler(rb *rebound.Rebound, logger *zap.Logger) *OrderHandler {
    return &OrderHandler{
        rebound: rb,
        logger:  logger,
    }
}

func (h *OrderHandler) ProcessOrder(w http.ResponseWriter, r *http.Request) {
    // ... process order logic ...

    // If processing fails, schedule a retry
    task := &rebound.Task{
        ID:     orderID,
        Source: "order-service",
        Destination: rebound.Destination{
            Host:  "kafka.prod",
            Port:  "9092",
            Topic: "order-processing",
        },
        DeadDestination: rebound.Destination{
            Host:  "kafka.prod",
            Port:  "9092",
            Topic: "order-processing-dlq",
        },
        MaxRetries:      5,
        BaseDelay:       10,
        ClientID:        "order-service",
        MessageData:     orderJSON,
        DestinationType: rebound.DestinationTypeKafka,
    }

    if err := h.rebound.CreateTask(r.Context(), task); err != nil {
        h.logger.Error("failed to schedule retry", zap.Error(err))
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusAccepted)
}
```

## Configuration

### Config Options

```go
type Config struct {
    // Redis configuration (required)
    RedisAddr     string
    RedisPassword string
    RedisDB       int

    // Kafka configuration (required if using Kafka destinations)
    KafkaBrokers []string

    // Worker configuration
    PollInterval time.Duration // Default: 1s

    // Logger (optional, defaults to production logger)
    Logger *zap.Logger
}
```

### Environment-Based Configuration

```go
func NewConfigFromEnv() *rebound.Config {
    return &rebound.Config{
        RedisAddr:    os.Getenv("REDIS_ADDR"),
        RedisPassword: os.Getenv("REDIS_PASSWORD"),
        RedisDB:      getEnvInt("REDIS_DB", 0),
        KafkaBrokers: strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
        PollInterval: getEnvDuration("POLL_INTERVAL", 1*time.Second),
    }
}
```

## Destination Types

### Kafka Destinations

```go
task := &rebound.Task{
    Destination: rebound.Destination{
        Host:  "kafka.prod",
        Port:  "9092",
        Topic: "events",
    },
    DestinationType: rebound.DestinationTypeKafka,
    // ...
}
```

**Requirements:**
- `Host` - Kafka broker hostname
- `Port` - Kafka broker port
- `Topic` - Kafka topic name

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
    // ...
}
```

**Requirements:**
- `URL` - HTTP endpoint URL

**Behavior:**
- HTTP POST request
- Headers: `Content-Type: application/json`, `X-Message-Key: {task_id}|{attempt}`
- Body: Raw message data
- Success: 2xx status codes
- Failure: Non-2xx triggers retry

## Retry Logic

### Exponential Backoff

```
delay = base_delay * (2 ^ (attempt - 1))
```

**Example with base_delay=10s:**
- Attempt 1: 10s
- Attempt 2: 20s
- Attempt 3: 40s
- Attempt 4: 80s
- Attempt 5: 160s

### Dead Letter Queue

After `max_retries` attempts, tasks are automatically routed to the `dead_destination`.

## Best Practices

### 1. Use Appropriate Retry Limits

```go
// For critical operations
MaxRetries: 10,
BaseDelay:  5,

// For non-critical operations
MaxRetries: 3,
BaseDelay:  10,
```

### 2. Always Configure Dead Letter Destinations

```go
task := &rebound.Task{
    DeadDestination: rebound.Destination{
        // Same type as primary destination
    },
    // ...
}
```

### 3. Use Meaningful Task IDs

```go
// Good: Unique and traceable
ID: fmt.Sprintf("order-%s-%d", orderID, time.Now().Unix())

// Bad: Not unique
ID: "task-1"
```

### 4. Include Context in Message Data

```go
MessageData: json.Marshal(map[string]interface{}{
    "original_request_id": requestID,
    "timestamp":           time.Now(),
    "user_id":             userID,
    "action":              "process_payment",
    "payload":             originalPayload,
})
```

### 5. Handle Shutdown Gracefully

```go
func main() {
    rb, _ := rebound.New(cfg)
    defer rb.Close()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    rb.Start(ctx)

    // Handle signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    cancel()                             // Stop worker
    time.Sleep(5 * time.Second)          // Wait for in-flight tasks
    rb.Close()                           // Release resources
}
```

## Monitoring

### Structured Logging

Rebound logs all operations with structured fields:

```json
{
  "level": "info",
  "msg": "task scheduled",
  "task_id": "order-123",
  "source": "order-service",
  "destination_type": "http"
}
```

### Metrics (Recommended)

Instrument your code with Prometheus metrics:

```go
var (
    tasksCreated = promauto.NewCounter(prometheus.CounterOpts{
        Name: "rebound_tasks_created_total",
    })

    tasksCompleted = promauto.NewCounter(prometheus.CounterOpts{
        Name: "rebound_tasks_completed_total",
    })
)

func (h *Handler) CreateRetry(ctx context.Context, task *rebound.Task) error {
    err := h.rebound.CreateTask(ctx, task)
    if err == nil {
        tasksCreated.Inc()
    }
    return err
}
```

## Testing

### Unit Tests

```go
func TestMyService(t *testing.T) {
    // Use a mock Rebound for unit tests
    mockRebound := &MockRebound{}
    service := NewMyService(mockRebound)

    // Test your business logic
    err := service.ProcessOrder(ctx, order)
    assert.NoError(t, err)
    assert.Equal(t, 1, mockRebound.CreateTaskCalls)
}
```

### Integration Tests

```go
func TestReboundIntegration(t *testing.T) {
    // Start Redis for tests
    redisContainer := testcontainers.RunRedis(t)
    defer redisContainer.Stop()

    cfg := &rebound.Config{
        RedisAddr: redisContainer.Addr(),
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

## Performance

### Throughput

- **Task Creation:** ~1000 tasks/sec (limited by Redis)
- **Task Processing:** ~500 tasks/sec (limited by destination)

### Resource Usage

- **Memory:** ~50MB baseline + 1KB per pending task
- **CPU:** <5% idle, <20% under load
- **Redis:** ~1KB per pending task

## Troubleshooting

### Tasks Not Being Processed

**Check worker is running:**
```go
rb.Start(ctx) // Make sure this was called
```

**Check Redis connectivity:**
```bash
redis-cli -h localhost -p 6379 PING
```

### High Retry Rate

**Increase base delay:**
```go
BaseDelay: 30, // 30 seconds instead of 5
```

**Review destination health:**
- Check HTTP endpoint availability
- Verify Kafka broker connectivity

### Memory Growth

**Limit pending tasks:**
- Process tasks faster
- Increase worker poll frequency
- Scale horizontally

## Migration Guide

### From Standalone Service to Library

If you're currently running Rebound as a standalone service:

**Before:**
```bash
docker run rebound:latest
curl -X POST http://rebound:8080/tasks -d '{...}'
```

**After:**
```go
// Embed in your app
rb, _ := rebound.New(cfg)
rb.Start(ctx)
rb.CreateTask(ctx, task)
```

**Benefits:**
- ✅ No extra network hop
- ✅ No separate service to deploy
- ✅ Shares Redis/Kafka connections
- ✅ Better observability

## License

[Add your license here]

## Support

- **Issues:** https://github.com/brevo/rebound/issues
- **Documentation:** https://pkg.go.dev/github.com/brevo/rebound
- **Slack:** #rebound-support
