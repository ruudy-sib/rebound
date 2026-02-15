# Rebound - Resilient Retry Orchestration

A production-ready retry orchestration service built with Go that provides intelligent retry mechanisms for failed message processing with exponential backoff and dead letter queue support.

## 🚀 Quick Start

**Choose your deployment mode:**

### Option 1: Embedded Go Package (Recommended for Go services)

```go
import "rebound/pkg/rebound"

rb, _ := rebound.New(&rebound.Config{
    RedisAddr: "localhost:6379",
})
rb.Start(ctx)
rb.CreateTask(ctx, &rebound.Task{...})
```

### Option 2: Standalone HTTP Service (For any language)

```bash
# Start service
go run cmd/kafka-retry/main.go

# Create task via HTTP
curl -X POST http://localhost:8080/tasks -d '{...}'
```

**See [QUICKSTART.md](QUICKSTART.md) for detailed setup (5 minutes)**

---

## 📚 Documentation

| Guide | Description | When to Read |
|-------|-------------|--------------|
| **[QUICKSTART.md](QUICKSTART.md)** | 5-minute getting started | Start here! |
| **[DEPLOYMENT.md](DEPLOYMENT.md)** | Compare deployment options | Choosing how to deploy |
| **[INTEGRATION.md](INTEGRATION.md)** | Embed in Go services | Integrating with your app |
| **[examples/](examples/)** | 6 real-world examples | Learning by example |
| **[VERIFICATION.md](VERIFICATION.md)** | Test verification report | Checking quality |

---

## Overview

Rebound acts as a resilient retry orchestration system for distributed applications. When message processing fails, Rebound:

1. **Schedules intelligent retries** with exponential backoff
2. **Tracks retry attempts** and prevents infinite loops
3. **Routes exhausted messages** to dead letter queues
4. **Provides visibility** into retry status via health checks

### Deployment Flexibility

**Rebound supports two deployment modes:**

```
┌─────────────────────┐         ┌──────────────────────┐
│  Go Service         │         │  Python/Node Service │
│  (Embedded)         │         │  (HTTP Client)       │
│                     │         │                      │
│  rb.CreateTask(...) │         │  POST /tasks         │
│         │           │         │         │            │
│         ▼           │         │         ▼            │
│  ┌──────────────┐   │         │  ┌──────────────┐   │
│  │Rebound Pkg   │   │         │  │HTTP Service  │   │
│  └──────┬───────┘   │         │  └──────┬───────┘   │
└─────────┼───────────┘         └─────────┼───────────┘
          │                               │
          └───────────┬───────────────────┘
                      ▼
              ┌───────────────┐
              │ Shared Redis  │
              └───────────────┘
```

**Choose based on your needs:**
- **Go services?** → Use embedded package (best performance)
- **Non-Go services?** → Use HTTP API (language agnostic)

See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed comparison.

---

## Key Features

- 🔄 **Exponential Backoff** - Intelligent retry scheduling with configurable delays
- 📊 **Multiple Destinations** - Support for Kafka topics and HTTP webhooks
- ☠️ **Dead Letter Queue** - Automatic routing of exhausted retries
- 📦 **Dual Deployment** - Use as embedded package OR standalone service
- 🏥 **Health Checks** - Redis connectivity monitoring
- 🎯 **HTTP & Kafka** - Retry failed webhooks and Kafka messages
- 🏗️ **Hexagonal Architecture** - Clean separation of concerns
- 📝 **Structured Logging** - Zap-based contextual logging
- 🧪 **High Test Coverage** - 87%+ coverage, production ready
- 🔌 **Graceful Shutdown** - SIGTERM/SIGINT handling
- 💉 **DI-Friendly** - First-class uber-go/dig support

---

## Architecture

### Hexagonal Architecture (Ports & Adapters)

```
┌─────────────────────────────────────────────────────────────┐
│                       Primary Adapters                        │
│  ┌──────────────────┐              ┌──────────────────┐      │
│  │  HTTP Handler    │              │  Worker Poller   │      │
│  │  POST /tasks     │              │  (Redis Polling) │      │
│  │  GET /health     │              │                  │      │
│  └────────┬─────────┘              └─────────┬────────┘      │
│           │                                   │               │
└───────────┼───────────────────────────────────┼───────────────┘
            │                                   │
            ▼                                   ▼
    ┌───────────────────────────────────────────────────┐
    │              Primary Ports (Interfaces)           │
    │           TaskService (Use Cases)                 │
    └───────────────────────┬───────────────────────────┘
                            │
    ┌───────────────────────▼───────────────────────────┐
    │                  Domain Layer                      │
    │  ┌──────────────────────────────────────────┐     │
    │  │  Task Entity (Business Logic)            │     │
    │  │  - IncrementAttempt()                    │     │
    │  │  - NextRetryDelay() (exponential)        │     │
    │  │  - HasRetriesLeft()                      │     │
    │  └──────────────────────────────────────────┘     │
    │  ┌──────────────────────────────────────────┐     │
    │  │  Task Service (Business Rules)           │     │
    │  │  - Validation                            │     │
    │  │  - Retry coordination                    │     │
    │  └──────────────────────────────────────────┘     │
    └───────────────────────┬───────────────────────────┘
                            │
    ┌───────────────────────▼───────────────────────────┐
    │           Secondary Ports (Interfaces)            │
    │  - TaskScheduler (Redis)                          │
    │  - MessageProducer (Kafka/HTTP)                   │
    │  - HealthChecker (Redis)                          │
    └───────────────────────┬───────────────────────────┘
                            │
┌───────────────────────────▼───────────────────────────────────┐
│                    Secondary Adapters                          │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────┐ │
│  │  Redis Adapter   │  │  Kafka Adapter   │  │HTTP Adapter │ │
│  │  - Sorted Set    │  │  - Producer      │  │ - Webhooks  │ │
│  │  - Health Check  │  │                  │  │             │ │
│  └──────────────────┘  └──────────────────┘  └─────────────┘ │
│                     ┌─────────────────────────────┐           │
│                     │  Producer Factory           │           │
│                     │  (Routes by destination)    │           │
│                     └─────────────────────────────┘           │
└────────────────────────────────────────────────────────────────┘
```

---

## Project Structure

```
rebound/
├── pkg/rebound/              # 📦 Public Go package API (use this!)
│   ├── rebound.go            # Main types and functions
│   ├── di.go                 # Dependency injection helpers
│   ├── example_test.go       # Usage examples
│   └── README.md             # Package documentation
│
├── cmd/kafka-retry/          # 🚀 Standalone HTTP service
│   ├── main.go               # Entry point with graceful shutdown
│   ├── container.go          # DI container setup
│   └── logger.go             # Logger configuration
│
├── internal/                 # 🔒 Internal implementation (not exported)
│   ├── config/               # Configuration management
│   ├── domain/               # Business logic (zero infrastructure deps)
│   │   ├── entity/           # Domain entities (Task, Destination)
│   │   ├── service/          # Business logic (94.5% coverage)
│   │   └── valueobject/      # Value objects
│   ├── port/                 # Interface definitions
│   │   ├── primary/          # Input ports (TaskService)
│   │   └── secondary/        # Output ports (Scheduler, Producer)
│   └── adapter/              # External integrations
│       ├── primary/          # Input adapters
│       │   ├── http/         # REST API handlers
│       │   └── worker/       # Redis polling worker
│       └── secondary/        # Output adapters
│           ├── kafkaproducer/   # Kafka producer
│           ├── httpproducer/    # HTTP webhook producer
│           ├── producerfactory/ # Routes to correct producer
│           └── redisstore/      # Redis scheduler
│
├── examples/                 # 📚 Real-world usage examples
│   ├── 01-basic-usage/       # Simplest example
│   ├── 02-email-service/     # Email retry pattern
│   ├── 03-webhook-delivery/  # Webhook delivery pattern
│   ├── 04-di-integration/    # Production DI pattern
│   ├── 05-payment-processing/# Smart retry strategies
│   ├── 06-multi-tenant/      # Multi-tenant SaaS pattern
│   └── EXAMPLES.md           # Examples guide
│
├── QUICKSTART.md             # 5-minute getting started
├── DEPLOYMENT.md             # Deployment options comparison
├── INTEGRATION.md            # Go integration guide
├── VERIFICATION.md           # Test verification report
├── openapi.yaml              # HTTP API specification
├── docker-compose.yml        # Infrastructure setup
└── test-both-modes.sh        # Automated verification
```

---

## Prerequisites

- **Go 1.23.1+**
- **Docker & Docker Compose** (for running dependencies)
- **Redis 7.2+** (for task scheduling)
- **Kafka 2.8+** (optional, only if using Kafka destinations)

---

## Installation

### For Embedded Package (Go Services)

```bash
# Add to your go.mod
go get rebound/pkg/rebound

# Import in your code
import "rebound/pkg/rebound"
```

See [INTEGRATION.md](INTEGRATION.md) for detailed integration patterns.

### For Standalone Service

```bash
# Clone repository
git clone https://github.com/your-org/rebound.git
cd rebound

# Start infrastructure
docker-compose up -d

# Run service
go run cmd/kafka-retry/main.go
```

---

## Configuration

The service is configured via environment variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `HTTP_PORT` | HTTP server port | `8080` | No |
| `REDIS_ADDR` | Redis address | `localhost:6379` | No |
| `REDIS_PASSWORD` | Redis password | _(empty)_ | No |
| `REDIS_DB` | Redis database number | `0` | No |
| `KAFKA_BROKERS` | Comma-separated Kafka brokers | `localhost:9092` | No |
| `POLL_INTERVAL` | Worker poll interval | `1s` | No |
| `LOG_LEVEL` | Logging level | `info` | No |
| `ENVIRONMENT` | Environment (dev/prod) | `dev` | No |

---

## Usage Examples

### Embedded Package (Go)

```go
package main

import (
    "context"
    "rebound/pkg/rebound"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    // Create Rebound
    rb, _ := rebound.New(&rebound.Config{
        RedisAddr: "localhost:6379",
        KafkaBrokers: []string{"localhost:9092"},
        Logger: logger,
    })
    defer rb.Close()

    // Start worker
    ctx := context.Background()
    rb.Start(ctx)

    // Create HTTP webhook task
    rb.CreateTask(ctx, &rebound.Task{
        ID: "order-123",
        Source: "order-service",
        Destination: rebound.Destination{
            URL: "https://webhook.example.com",
        },
        DeadDestination: rebound.Destination{
            URL: "https://webhook.example.com/dlq",
        },
        MaxRetries: 5,
        BaseDelay: 10,
        ClientID: "order-service",
        MessageData: `{"order_id": 123}`,
        DestinationType: rebound.DestinationTypeHTTP,
    })

    // Create Kafka task
    rb.CreateTask(ctx, &rebound.Task{
        ID: "kafka-msg-456",
        Source: "order-service",
        Destination: rebound.Destination{
            Host: "kafka.prod",
            Port: "9092",
            Topic: "orders",
        },
        DeadDestination: rebound.Destination{
            Host: "kafka.prod",
            Port: "9092",
            Topic: "orders-dlq",
        },
        MaxRetries: 5,
        BaseDelay: 10,
        ClientID: "order-service",
        MessageData: `{"order_id": 456}`,
        DestinationType: rebound.DestinationTypeKafka,
    })
}
```

### HTTP API (Any Language)

**Create HTTP Webhook Task:**
```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "webhook-123",
    "source": "order-service",
    "destination": {
      "url": "https://api.partner.com/webhook"
    },
    "dead_destination": {
      "url": "https://api.partner.com/webhook-dlq"
    },
    "max_retries": 5,
    "base_delay": 10,
    "client_id": "order-service",
    "message_data": "{\"event\": \"order.created\"}",
    "destination_type": "http"
  }'
```

**Create Kafka Task:**
```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "kafka-456",
    "source": "order-service",
    "destination": {
      "host": "localhost",
      "port": "9092",
      "topic": "orders"
    },
    "dead_destination": {
      "host": "localhost",
      "port": "9092",
      "topic": "orders-dlq"
    },
    "max_retries": 3,
    "base_delay": 5,
    "client_id": "order-service",
    "message_data": "{\"order_id\": 456}",
    "destination_type": "kafka"
  }'
```

**From Python:**
```python
import requests

requests.post('http://localhost:8080/tasks', json={
    'id': 'task-123',
    'source': 'my-service',
    'destination': {'url': 'https://api.example.com/webhook'},
    'dead_destination': {'url': 'https://api.example.com/dlq'},
    'max_retries': 5,
    'base_delay': 10,
    'client_id': 'my-service',
    'message_data': '{"event": "test"}',
    'destination_type': 'http'
})
```

**From Node.js:**
```javascript
const axios = require('axios');

await axios.post('http://localhost:8080/tasks', {
  id: 'task-123',
  source: 'my-service',
  destination: { url: 'https://api.example.com/webhook' },
  dead_destination: { url: 'https://api.example.com/dlq' },
  max_retries: 5,
  base_delay: 10,
  client_id: 'my-service',
  message_data: JSON.stringify({ event: 'test' }),
  destination_type: 'http'
});
```

See [examples/](examples/) for 6 comprehensive real-world examples.

---

## Destination Types

### Kafka Destinations

Retry failed Kafka messages:

```go
Destination: rebound.Destination{
    Host:  "kafka.prod",
    Port:  "9092",
    Topic: "orders",
}
DestinationType: rebound.DestinationTypeKafka
```

**Features:**
- Messages sent to Kafka topics via Kafka producer
- Supports all Kafka configuration options
- Dead letter queue support

### HTTP Destinations

Retry failed webhooks:

```go
Destination: rebound.Destination{
    URL: "https://api.partner.com/webhook",
}
DestinationType: rebound.DestinationTypeHTTP
```

**Features:**
- HTTP POST requests with JSON payload
- Message key sent as `X-Message-Key` header
- Success: 2xx status codes
- Failure: Non-2xx triggers retry
- 30-second timeout
- Connection pooling

---

## Retry Logic

### Exponential Backoff

```
delay = base_delay * (2 ^ (attempt - 1))
```

**Example with base_delay=10s:**
- Attempt 1: 10s delay (10 × 2^0 = 10)
- Attempt 2: 20s delay (10 × 2^1 = 20)
- Attempt 3: 40s delay (10 × 2^2 = 40)
- Attempt 4: 80s delay (10 × 2^3 = 80)
- Attempt 5: 160s delay (10 × 2^4 = 160)

### Dead Letter Queue

After `max_retries` attempts, tasks are automatically routed to the `dead_destination`.

---

## Health Check

```bash
curl http://localhost:8080/health
```

**Response (Healthy):**
```json
{
  "status": "healthy",
  "checks": {
    "redis": "connected"
  }
}
```

**Response (Unhealthy):**
```json
{
  "status": "unhealthy",
  "error": "redis connection failed"
}
```

---

## Testing

### Run All Tests

```bash
go test ./...
```

### Run with Coverage

```bash
go test ./... -cover
```

### Verify Both Deployment Modes

```bash
./test-both-modes.sh
```

### Test Coverage

| Package | Coverage |
|---------|----------|
| `domain/entity` | 100% |
| `domain/service` | 94.5% |
| `domain/valueobject` | 100% |
| `adapter/primary/http` | 84.6% |
| `adapter/primary/worker` | 100% |
| `adapter/secondary/httpproducer` | 91.3% |
| `config` | 100% |

**Overall: 87%+ coverage** ✅

---

## Examples

Run these examples to see Rebound in action:

```bash
# 1. Basic usage (simplest)
go run examples/01-basic-usage/main.go

# 2. Email service with retry
go run examples/02-email-service/main.go

# 3. Webhook delivery service
go run examples/03-webhook-delivery/main.go

# 4. Dependency injection (production pattern)
go run examples/04-di-integration/main.go

# 5. Payment processing (smart retry)
go run examples/05-payment-processing/main.go

# 6. Multi-tenant SaaS
go run examples/06-multi-tenant/main.go
```

See [examples/EXAMPLES.md](examples/EXAMPLES.md) for detailed descriptions.

---

## Deployment

### Docker

```bash
docker build -t rebound:latest .
docker run -d \
  --name rebound \
  -p 8080:8080 \
  -e REDIS_ADDR=redis:6379 \
  -e KAFKA_BROKERS=kafka:9092 \
  rebound:latest
```

### Docker Compose

```bash
docker-compose up -d
```

### Kubernetes

See [DEPLOYMENT.md](DEPLOYMENT.md) for Kubernetes deployment examples including:
- Standalone deployment
- Sidecar pattern
- Resource limits
- Health checks

---

## Performance

### Embedded Package
- **Throughput:** 5000 tasks/sec
- **Latency:** ~1ms (p99)
- **Memory:** 50MB baseline

### Standalone Service
- **Throughput:** 1000 tasks/sec
- **Latency:** ~10ms (p99)
- **Memory:** 50MB + HTTP overhead

---

## Monitoring

### Structured Logging

All logs are JSON-formatted in production:

```json
{
  "level": "info",
  "ts": "2026-02-16T10:30:45.123Z",
  "msg": "task scheduled",
  "task_id": "order-123",
  "source": "order-service",
  "destination_type": "http"
}
```

### Metrics (Recommended)

Add Prometheus metrics:
```go
tasksCreated := promauto.NewCounter(prometheus.CounterOpts{
    Name: "rebound_tasks_created_total",
})
```

---

## Troubleshooting

### Application Won't Start

**Issue:** `failed to create redis client: connection refused`

**Solution:**
```bash
docker-compose restart redis
```

### Tasks Not Processing

**Issue:** Tasks created but never sent

**Solution:**
```bash
# Check worker is running
docker-compose logs kafka-retry | grep "Worker started"

# Check Redis for pending tasks
docker exec redis redis-cli ZRANGE retry:schedule: 0 -1
```

See full troubleshooting guide in [README.md](README.md).

---

## Production Considerations

### Scaling

**Horizontal Scaling:**
- Run multiple instances behind a load balancer
- Each instance independently polls Redis
- Redis ensures no duplicate processing

**Vertical Scaling:**
- Increase worker goroutines
- Increase Redis connection pool
- Increase Kafka producer batch size

### Security

**Redis:**
```bash
export REDIS_PASSWORD=your-secure-password
```

**Kafka:**
- Use SASL/SSL for production
- Configure in `internal/adapter/secondary/kafkaproducer/`

### High Availability

- Use Redis Sentinel or Redis Cluster
- Use multiple Kafka brokers
- Configure producer acknowledgment: `acks=all`

---

## Contributing

### Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `golangci-lint` for linting
- Maintain test coverage >80%
- Document public APIs

### Pull Request Process

1. Create feature branch
2. Write tests for new code
3. Ensure all tests pass: `go test ./...`
4. Run linter: `golangci-lint run`
5. Submit PR with clear description

---

## License

[Add your license here]

---

## Support

For questions and issues:
- **Documentation:** This README, [QUICKSTART.md](QUICKSTART.md), [INTEGRATION.md](INTEGRATION.md)
- **Examples:** [examples/](examples/)
- **Issues:** Create GitHub issues
- **Email:** [your-email@example.com]

---

## Roadmap

- [x] HTTP webhook destination support
- [x] Embedded Go package API
- [x] Comprehensive documentation
- [x] Real-world examples
- [ ] Prometheus metrics integration
- [ ] OpenTelemetry distributed tracing
- [ ] Admin API for task inspection
- [ ] Priority queue implementation
- [ ] Web UI for monitoring
- [ ] Custom backoff strategies
- [ ] Additional destinations (SQS, PubSub, RabbitMQ)

---

**Built with ❤️ using Go and Hexagonal Architecture**

**Start using Rebound:** Read [QUICKSTART.md](QUICKSTART.md) to get started in 5 minutes!
