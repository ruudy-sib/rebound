# Integrating Rebound into Brevo Go Applications

This guide shows how to embed Rebound retry functionality directly into your Brevo Go services.

## Overview

Rebound can be used in three ways:

1. **As a Go package** (Recommended for Brevo) - Embed directly in your service
2. **As a sidecar service** - Run alongside your service in the same pod
3. **As a standalone service** - Centralized retry service (not recommended)

## Architecture Decision

**For Brevo services, we recommend embedding Rebound as a Go package because:**

✅ **No extra network hop** - Direct function calls instead of HTTP
✅ **Shared connections** - Reuses your existing Redis/Kafka connections
✅ **Better observability** - Unified logging, tracing, and metrics
✅ **Simpler deployment** - No additional service to manage
✅ **Type safety** - Compile-time guarantees

## Installation

### Option 1: Local Module (Development)

```bash
# In your go.mod, add a replace directive
replace rebound => ../path/to/rebound

# Then import
import "github.com/ruudy-sib/rebound/pkg/rebound"
```

### Option 2: Internal Go Module (Production)

```bash
# Publish to internal registry
# go get github.com/brevo-internal/rebound

# Or use private GitHub repo
# go get github.com/brevo/rebound
```

## Integration Patterns

### Pattern 1: Embed in Existing DI Container

For services already using `golang-libraries/di`:

```go
// cmd/myservice/container.go
package main

import (
    "github.com/brevo/golang-libraries/di"
    "github.com/ruudy-sib/rebound/pkg/rebound"
    "go.uber.org/zap"
)

func buildContainer() (*dig.Container, error) {
    container := di.GetContainer()

    // Provide rebound configuration
    container.Provide(func(logger *zap.Logger) *rebound.Config {
        return &rebound.Config{
            RedisMode:          os.Getenv("REDIS_MODE"),
            RedisAddr:          os.Getenv("REDIS_ADDR"),
            RedisMasterName:    os.Getenv("REDIS_MASTER_NAME"),
            RedisSentinelAddrs: splitEnv("REDIS_SENTINEL_ADDRS"),
            RedisClusterAddrs:  splitEnv("REDIS_CLUSTER_ADDRS"),
            // KafkaBrokers: splitEnv("KAFKA_BROKERS"), // optional: only for Kafka destinations
            PollInterval: 1 * time.Second,
            Logger:       logger,
        }
    })

    // Register rebound
    if err := rebound.RegisterWithContainer(container); err != nil {
        return nil, err
    }

    return container, nil
}
```

```go
// cmd/myservice/main.go
package main

import (
    "context"
    "github.com/brevo/golang-libraries/mainutils"
    "github.com/ruudy-sib/rebound/pkg/rebound"
)

func main() {
    mainutils.Run(func(ctx context.Context) error {
        container, err := buildContainer()
        if err != nil {
            return err
        }

        // Start rebound worker
        err = container.Invoke(func(rb *rebound.Rebound) error {
            return rb.Start(ctx)
        })
        if err != nil {
            return err
        }

        // Start your HTTP server, workers, etc.
        return startServer(ctx, container)
    })
}
```

### Pattern 2: Use in HTTP Handlers

```go
// internal/adapter/primary/http/handler_order_create.go
package http

import (
    "net/http"
    "github.com/brevo/golang-libraries/httpjson"
    "github.com/ruudy-sib/rebound/pkg/rebound"
    "go.uber.org/zap"
)

type CreateOrderHandler struct {
    rebound *rebound.Rebound
    logger  *zap.Logger
}

func NewCreateOrderHandler(rb *rebound.Rebound, logger *zap.Logger) *CreateOrderHandler {
    return &CreateOrderHandler{
        rebound: rb,
        logger:  logger.Named("create-order"),
    }
}

func (h *CreateOrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    var req CreateOrderRequest
    if err := httpjson.DecodeRequestBody(ctx, r, &req); err != nil {
        httpjson.WriteResponse(ctx, w, http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
        return
    }

    // Process order...
    order, err := h.processOrder(ctx, req)
    if err != nil {
        // If processing fails, schedule a retry
        h.scheduleRetry(ctx, order)

        httpjson.WriteResponse(ctx, w, http.StatusAccepted, CreateOrderResponse{
            Status: "retry_scheduled",
        })
        return
    }

    httpjson.WriteResponse(ctx, w, http.StatusCreated, CreateOrderResponse{
        OrderID: order.ID,
        Status:  "created",
    })
}

func (h *CreateOrderHandler) scheduleRetry(ctx context.Context, order *Order) {
    task := &rebound.Task{
        ID:     fmt.Sprintf("order-%s-%d", order.ID, time.Now().Unix()),
        Source: "order-service",
        Destination: rebound.Destination{
            Host:  "kafka.production.svc.cluster.local",
            Port:  "9092",
            Topic: "order-processing",
        },
        DeadDestination: rebound.Destination{
            Host:  "kafka.production.svc.cluster.local",
            Port:  "9092",
            Topic: "order-processing-dlq",
        },
        MaxRetries:      5,
        BaseDelay:       10,
        ClientID:        "order-service",
        MessageData:     order.ToJSON(),
        DestinationType: rebound.DestinationTypeKafka,
    }

    if err := h.rebound.CreateTask(ctx, task); err != nil {
        h.logger.Error("failed to schedule retry",
            zap.String("order_id", order.ID),
            zap.Error(err),
        )
    }
}
```

### Pattern 3: Use in Kafka Consumers

```go
// internal/adapter/primary/kafka/consumer_order.go
package kafka

import (
    "context"
    "github.com/ruudy-sib/rebound/pkg/rebound"
    "go.uber.org/zap"
)

type OrderConsumer struct {
    rebound *rebound.Rebound
    logger  *zap.Logger
}

func (c *OrderConsumer) Consume(ctx context.Context, message []byte) error {
    var order Order
    if err := json.Unmarshal(message, &order); err != nil {
        return err
    }

    // Try to process
    if err := c.processOrder(ctx, &order); err != nil {
        c.logger.Warn("order processing failed, scheduling retry",
            zap.String("order_id", order.ID),
            zap.Error(err),
        )

        // Schedule HTTP webhook retry
        task := &rebound.Task{
            ID:     fmt.Sprintf("webhook-%s", order.ID),
            Source: "order-consumer",
            Destination: rebound.Destination{
                URL: "https://partner-api.com/orders",
            },
            DeadDestination: rebound.Destination{
                URL: "https://partner-api.com/orders-dlq",
            },
            MaxRetries:      3,
            BaseDelay:       5,
            ClientID:        "order-consumer",
            MessageData:     string(message),
            DestinationType: rebound.DestinationTypeHTTP,
        }

        return c.rebound.CreateTask(ctx, task)
    }

    return nil
}
```

### Pattern 4: Use in Domain Services

```go
// internal/domain/service/order_service.go
package service

import (
    "context"
    "github.com/ruudy-sib/rebound/pkg/rebound"
    "go.uber.org/zap"
)

type OrderService struct {
    rebound *rebound.Rebound
    logger  *zap.Logger
}

func NewOrderService(rb *rebound.Rebound, logger *zap.Logger) *OrderService {
    return &OrderService{
        rebound: rb,
        logger:  logger,
    }
}

func (s *OrderService) CreateOrder(ctx context.Context, order *Order) error {
    // Validate and save order
    if err := s.saveOrder(ctx, order); err != nil {
        return err
    }

    // Send notification (with retry on failure)
    if err := s.sendNotification(ctx, order); err != nil {
        s.logger.Warn("notification failed, scheduling retry", zap.Error(err))
        s.scheduleNotificationRetry(ctx, order)
    }

    return nil
}

func (s *OrderService) scheduleNotificationRetry(ctx context.Context, order *Order) {
    task := &rebound.Task{
        ID:     fmt.Sprintf("notification-%s", order.ID),
        Source: "order-service",
        Destination: rebound.Destination{
            URL: "https://notification-api.brevo.com/send",
        },
        MaxRetries:      3,
        BaseDelay:       10,
        ClientID:        "order-service",
        MessageData:     order.NotificationPayload(),
        DestinationType: rebound.DestinationTypeHTTP,
    }

    if err := s.rebound.CreateTask(ctx, task); err != nil {
        s.logger.Error("failed to schedule notification retry", zap.Error(err))
    }
}
```

## Configuration

### Environment Variables

```bash
# .env or Kubernetes ConfigMap

# Redis (choose one mode)
REDIS_MODE=standalone                                    # default
REDIS_ADDR=redis.production.svc.cluster.local:6379
REDIS_PASSWORD=your-secure-password
REDIS_DB=0

# Sentinel HA (set REDIS_MODE=sentinel)
# REDIS_MASTER_NAME=mymaster
# REDIS_SENTINEL_ADDRS=sentinel-1:26379,sentinel-2:26379,sentinel-3:26379

# Cluster (set REDIS_MODE=cluster)
# REDIS_CLUSTER_ADDRS=node-1:7000,node-2:7001,node-3:7002

# Kafka (optional: only set if using Kafka destinations)
# KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092

REBOUND_POLL_INTERVAL=1s
```

### Configuration Provider

```go
// internal/config/rebound.go
package config

import (
    "os"
    "strings"
    "time"
    "github.com/ruudy-sib/rebound/pkg/rebound"
    "go.uber.org/zap"
)

func ProvideReboundConfig(logger *zap.Logger) *rebound.Config {
    return &rebound.Config{
        RedisMode:          getEnv("REDIS_MODE", "standalone"),
        RedisAddr:          os.Getenv("REDIS_ADDR"),
        RedisPassword:      os.Getenv("REDIS_PASSWORD"),
        RedisDB:            getEnvInt("REDIS_DB", 0),
        RedisMasterName:    os.Getenv("REDIS_MASTER_NAME"),
        RedisSentinelAddrs: splitCommaSep(os.Getenv("REDIS_SENTINEL_ADDRS")),
        RedisClusterAddrs:  splitCommaSep(os.Getenv("REDIS_CLUSTER_ADDRS")),
        // KafkaBrokers: splitCommaSep(os.Getenv("KAFKA_BROKERS")), // optional
        PollInterval: getEnvDuration("REBOUND_POLL_INTERVAL", 1*time.Second),
        Logger:       logger,
    }
}
```

## Observability

### Structured Logging

Rebound uses zap for structured logging. All logs include context:

```json
{
  "level": "info",
  "ts": "2026-02-16T10:30:45.123Z",
  "msg": "task scheduled",
  "task_id": "order-123",
  "source": "order-service",
  "destination_type": "kafka",
  "max_retries": 5
}
```

### Metrics (Recommended)

Add Prometheus metrics to track Rebound usage:

```go
// internal/metrics/rebound.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    ReboundTasksCreated = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rebound_tasks_created_total",
            Help: "Total number of retry tasks created",
        },
        []string{"source", "destination_type"},
    )

    ReboundTasksCompleted = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rebound_tasks_completed_total",
            Help: "Total number of retry tasks completed",
        },
        []string{"source", "destination_type", "status"},
    )
)
```

### Tracing

Rebound respects context cancellation and can be traced:

```go
import "go.opentelemetry.io/otel"

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, span := otel.Tracer("myservice").Start(r.Context(), "schedule-retry")
    defer span.End()

    task := &rebound.Task{...}
    err := h.rebound.CreateTask(ctx, task)
    // ...
}
```

## Testing

### Unit Tests with Mock

```go
// internal/service/order_service_test.go
package service

import (
    "testing"
    "github.com/ruudy-sib/rebound/pkg/rebound"
)

type MockRebound struct {
    CreateTaskFunc func(ctx context.Context, task *rebound.Task) error
    CallCount      int
}

func (m *MockRebound) CreateTask(ctx context.Context, task *rebound.Task) error {
    m.CallCount++
    if m.CreateTaskFunc != nil {
        return m.CreateTaskFunc(ctx, task)
    }
    return nil
}

func TestOrderService_CreateOrder(t *testing.T) {
    mockRebound := &MockRebound{}
    svc := NewOrderService(mockRebound, zap.NewNop())

    err := svc.CreateOrder(context.Background(), &Order{ID: "123"})
    assert.NoError(t, err)
    assert.Equal(t, 1, mockRebound.CallCount)
}
```

### Integration Tests

```go
// test/integration/rebound_test.go
package integration

import (
    "testing"
    "github.com/ruudy-sib/rebound/pkg/rebound"
)

func TestReboundIntegration(t *testing.T) {
    // Start test Redis
    redisContainer := startTestRedis(t)
    defer redisContainer.Stop()

    cfg := &rebound.Config{
        RedisAddr: redisContainer.Addr(),
    }

    rb, err := rebound.New(cfg)
    require.NoError(t, err)
    defer rb.Close()

    ctx := context.Background()
    rb.Start(ctx)

    // Create task
    task := &rebound.Task{
        ID:              "test-1",
        Source:          "test",
        Destination:     rebound.Destination{URL: "http://localhost:8090/webhook"},
        MaxRetries:      3,
        BaseDelay:       1,
        ClientID:        "test",
        MessageData:     `{"test":"data"}`,
        DestinationType: rebound.DestinationTypeHTTP,
    }

    err = rb.CreateTask(ctx, task)
    assert.NoError(t, err)
}
```

## Deployment

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
spec:
  template:
    spec:
      containers:
      - name: order-service
        image: order-service:latest
        env:
        - name: REDIS_ADDR
          value: "redis.production.svc.cluster.local:6379"
        - name: REDIS_MODE
          value: "standalone"
        # Uncomment for Kafka destinations:
        # - name: KAFKA_BROKERS
        #   value: "kafka-1:9092,kafka-2:9092"
        - name: REBOUND_POLL_INTERVAL
          value: "1s"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### Resource Requirements

**Per service instance:**
- Memory: 50MB baseline + 1KB per pending task
- CPU: <5% idle, <20% under load
- Redis: 1KB per pending task

## Best Practices for Brevo

1. **Shared Redis Instance**
   - Use the same Redis cluster as your service's cache
   - Use a different DB number (e.g., DB 1 for rebound, DB 0 for cache)

2. **Shared Kafka Connections**
   - Rebound creates a single Kafka producer per service instance
   - Reuses the same broker connections as your application

3. **Namespace Tasks by Service**
   - Use service name as task ID prefix: `{service}-{resource}-{id}`
   - Example: `order-service-payment-12345`

4. **Configure Appropriate Retries**
   - Critical operations: 10 retries, 5s base delay
   - Non-critical: 3 retries, 10s base delay

5. **Monitor Dead Letter Queues**
   - Set up alerts for DLQ messages
   - Review and replay failed tasks

6. **Graceful Shutdown**
   - Always call `rb.Close()` in shutdown hooks
   - Wait 5-10s for in-flight tasks to complete

## Migration from Standalone Service

If you're currently using Rebound as a standalone HTTP service:

**Before (HTTP API):**
```go
resp, err := http.Post("http://rebound:8080/tasks", "application/json", body)
```

**After (Library):**
```go
err := rb.CreateTask(ctx, task)
```

**Benefits:**
- 10x faster (no network round trip)
- Type-safe
- Better error handling
- Unified observability

## FAQ

**Q: Can I use Rebound in multiple services?**
A: Yes! Each service embeds its own Rebound instance. They all share the same Redis cluster.

**Q: Does each service need its own Redis?**
A: No. All services can share a Redis cluster. Use different DB numbers or key prefixes to separate data.

**Q: What happens if Redis goes down?**
A: Task creation will fail, but your service continues running. Implement fallback logic (e.g., log and alert).

**Q: Can I horizontally scale services using Rebound?**
A: Yes! Multiple instances will coordinate via Redis. Each task is processed exactly once.

**Q: What's the performance impact?**
A: Minimal. Task creation is ~1ms. Worker polls Redis every 1s. CPU and memory usage are negligible.

**Q: Can I use custom backoff strategies?**
A: Not yet. Currently only exponential backoff is supported. Custom strategies are on the roadmap.

## Support

- **Documentation:** This file
- **Examples:** `pkg/rebound/example_test.go`
- **Issues:** Create GitHub issues
- **Slack:** #rebound-support (when available)
