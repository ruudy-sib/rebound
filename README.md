# Kafka Retry Service

A production-ready Kafka retry service built with Go that provides intelligent retry mechanisms for failed message processing with exponential backoff and dead letter queue support.

## Overview

The Kafka Retry Service acts as a centralized retry orchestration system for distributed applications. When a message processing fails in your application, you can send it to this service which will:

1. **Schedule intelligent retries** with exponential backoff
2. **Track retry attempts** and prevent infinite loops
3. **Route exhausted messages** to dead letter queues
4. **Provide visibility** into retry status via health checks

### Key Features

- 🔄 **Exponential Backoff** - Intelligent retry scheduling with configurable delays
- 📊 **Retry Tracking** - Monitors attempts and prevents infinite retries
- ☠️ **Dead Letter Queue** - Automatic routing of exhausted retries
- 🏥 **Health Checks** - Redis connectivity monitoring
- 🎯 **HTTP API** - Simple REST interface for task submission
- 🏗️ **Hexagonal Architecture** - Clean separation of concerns
- 📝 **Structured Logging** - Zap-based contextual logging
- 🧪 **High Test Coverage** - 94.5% domain, 84.6% handler coverage
- 🔌 **Graceful Shutdown** - SIGTERM/SIGINT handling

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
    │  - MessageProducer (Kafka)                        │
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

### Project Structure

```
kafkaretry-poc/
├── cmd/kafka-retry/           # Application entry point
│   ├── main.go                # Main with graceful shutdown
│   ├── container.go           # Dependency injection container
│   └── logger.go              # Zap logger configuration
├── internal/
│   ├── config/                # Configuration management
│   │   ├── config.go          # Environment-based config
│   │   └── config_test.go
│   ├── domain/                # Business logic (zero infrastructure deps)
│   │   ├── constants.go       # Business constants
│   │   ├── errors.go          # Domain errors
│   │   ├── entity/
│   │   │   ├── destination.go # Kafka destination entity
│   │   │   ├── task.go        # Task entity with behavior
│   │   │   └── task_test.go
│   │   ├── service/
│   │   │   ├── task_service.go      # Core business logic
│   │   │   ├── task_service_test.go # 94.5% coverage
│   │   │   └── mocks_test.go
│   │   └── valueobject/
│   │       ├── task_id.go     # Immutable TaskID
│   │       └── task_id_test.go
│   ├── port/                  # Interface contracts
│   │   ├── primary/           # What domain exposes
│   │   │   └── task_service.go
│   │   └── secondary/         # What domain needs
│   │       ├── task_scheduler.go
│   │       ├── message_producer.go
│   │       └── health_checker.go
│   └── adapter/               # External integrations
│       ├── primary/           # Input adapters
│       │   ├── http/          # REST API handlers
│       │   └── worker/        # Redis polling worker
│       └── secondary/           # Output adapters
│           ├── kafkaproducer/   # Kafka producer
│           ├── httpproducer/    # HTTP webhook producer
│           ├── producerfactory/ # Routes to correct producer
│           └── redisstore/      # Redis scheduler
├── openapi.yaml               # API specification
├── docker-compose.yml         # Infrastructure setup
├── Dockerfile
├── go.mod
└── go.sum
```

## Prerequisites

- **Go 1.23.1+**
- **Docker & Docker Compose** (for running dependencies)
- **Make** (optional, for convenience commands)

## Installation

### 1. Clone the Repository

```bash
git clone <repository-url>
cd kafkaretry-poc
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Start Infrastructure (Redis + Kafka)

```bash
docker-compose up -d
```

This starts:
- **Redis** on `localhost:6379`
- **Kafka** on `localhost:9092`
- **Zookeeper** on `localhost:2181`

### 4. Verify Infrastructure

```bash
# Check Redis
docker exec -it kafkaretry-redis-1 redis-cli ping
# Expected: PONG

# Check Kafka
docker exec -it kafkaretry-kafka-1 kafka-topics --list --bootstrap-server localhost:9092
```

## Configuration

The application is configured via environment variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `HTTP_PORT` | HTTP server port | `8080` | No |
| `REDIS_ADDR` | Redis address | `localhost:6379` | No |
| `REDIS_PASSWORD` | Redis password | _(empty)_ | No |
| `REDIS_DB` | Redis database number | `0` | No |
| `KAFKA_BROKERS` | Comma-separated Kafka brokers | `localhost:9092` | No |
| `POLL_INTERVAL` | Worker poll interval | `1s` | No |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | `info` | No |
| `ENVIRONMENT` | Environment (dev/prod) | `dev` | No |

### Example Configuration

```bash
# Development
export HTTP_PORT=8080
export REDIS_ADDR=localhost:6379
export KAFKA_BROKERS=localhost:9092
export LOG_LEVEL=debug
export ENVIRONMENT=dev

# Production
export HTTP_PORT=8080
export REDIS_ADDR=redis.production.svc.cluster.local:6379
export REDIS_PASSWORD=your-secure-password
export KAFKA_BROKERS=kafka-1.prod:9092,kafka-2.prod:9092,kafka-3.prod:9092
export LOG_LEVEL=info
export ENVIRONMENT=prod
```

## Running the Application

### Option 1: Run with Go

```bash
# With default configuration
go run cmd/kafka-retry/main.go

# With custom configuration
HTTP_PORT=9090 REDIS_ADDR=localhost:6379 go run cmd/kafka-retry/main.go
```

### Option 2: Build and Run Binary

```bash
# Build
go build -o bin/kafka-retry cmd/kafka-retry/main.go

# Run
./bin/kafka-retry
```

### Option 3: Run with Docker

```bash
# Build Docker image
docker build -t kafka-retry:latest .

# Run container
docker run -d \
  --name kafka-retry \
  -p 8080:8080 \
  -e REDIS_ADDR=host.docker.internal:6379 \
  -e KAFKA_BROKERS=host.docker.internal:9092 \
  kafka-retry:latest
```

### Option 4: Run Full Stack with Docker Compose

```bash
# Start everything (Redis, Kafka, Application)
docker-compose up --build

# View logs
docker-compose logs -f kafka-retry

# Stop everything
docker-compose down
```

## API Usage

### Health Check

```bash
# Check application health
curl http://localhost:8080/health
```

**Response (Healthy):**
```json
{
  "status": "healthy"
}
```

**Response (Unhealthy - Redis down):**
```json
{
  "status": "unhealthy",
  "error": "redis connection failed"
}
```

### Create Retry Task

Submit a task to be retried with exponential backoff. Supports both **Kafka** and **HTTP** destinations.

**Endpoint:** `POST /tasks`

#### Kafka Destination Example

**Request Body:**
```json
{
  "id": "task-12345",
  "source": "order-service",
  "destination": {
    "host": "localhost",
    "port": "9092",
    "topic": "user-events"
  },
  "dead_destination": {
    "host": "localhost",
    "port": "9092",
    "topic": "user-events-dlq"
  },
  "max_retries": 5,
  "base_delay": 10,
  "client_id": "client-001",
  "is_priority": false,
  "message_data": "{\"user_id\": 123, \"action\": \"send_email\"}",
  "destination_type": "kafka"
}
```

#### HTTP Destination Example

**Request Body:**
```json
{
  "id": "webhook-task-789",
  "source": "notification-service",
  "destination": {
    "url": "https://api.partner.com/webhooks/events"
  },
  "dead_destination": {
    "url": "https://api.partner.com/webhooks/dlq"
  },
  "max_retries": 5,
  "base_delay": 10,
  "client_id": "partner-001",
  "is_priority": false,
  "message_data": "{\"event\": \"user.created\", \"user_id\": 789}",
  "destination_type": "http"
}
```

**Field Descriptions:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier for the task |
| `source` | string | Yes | Source system or application name |
| `destination.host` | string | Kafka only | Kafka broker hostname or IP |
| `destination.port` | string | Kafka only | Kafka broker port |
| `destination.topic` | string | Kafka only | Kafka topic for retries |
| `destination.url` | string | HTTP only | HTTP endpoint URL for webhook delivery |
| `dead_destination.host` | string | Kafka only | Dead letter queue broker host |
| `dead_destination.port` | string | Kafka only | Dead letter queue broker port |
| `dead_destination.topic` | string | Kafka only | Dead letter queue topic |
| `dead_destination.url` | string | HTTP only | HTTP endpoint URL for dead letter delivery |
| `max_retries` | int | Yes | Maximum retry attempts (0 or higher) |
| `base_delay` | int | Yes | Base delay in seconds for exponential backoff |
| `client_id` | string | Yes | Client identifier for tracking |
| `is_priority` | boolean | No | Whether this is a priority task (default: false) |
| `message_data` | string | Yes | The actual message/payload to be retried |
| `destination_type` | string | Yes | Destination type: `"kafka"` or `"http"` |

**Example with cURL (Kafka):**

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "order-task-456",
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
    "client_id": "web-client-123",
    "is_priority": true,
    "message_data": "{\"user_id\": 123, \"order_id\": 456, \"action\": \"process_payment\"}",
    "destination_type": "kafka"
  }'
```

**Example with cURL (HTTP):**

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "webhook-task-789",
    "source": "notification-service",
    "destination": {
      "url": "https://api.partner.com/webhooks/events"
    },
    "dead_destination": {
      "url": "https://api.partner.com/webhooks/dlq"
    },
    "max_retries": 3,
    "base_delay": 5,
    "client_id": "partner-api",
    "is_priority": false,
    "message_data": "{\"event\": \"user.created\", \"user_id\": 789}",
    "destination_type": "http"
  }'
```

**Success Response (201 Created):**
```json
{
  "message": "Task order-task-456 scheduled successfully"
}
```

**Error Response (400 Bad Request):**
```json
{
  "error": "payload is required"
}
```

**Error Response (500 Internal Server Error):**
```json
{
  "error": "failed to schedule task"
}
```

## How It Works

### 1. HTTP Webhook Delivery

When using HTTP destinations, the service delivers messages via HTTP POST requests:

**Request Headers:**
```
POST /webhooks/events HTTP/1.1
Host: api.partner.com
Content-Type: application/json
X-Message-Key: task-123|2
User-Agent: kafka-retry-service/1.0
```

**Request Body:**
```json
{
  "event": "user.created",
  "user_id": 789
}
```

**Success Response (2xx):**
- Any HTTP status code 200-299 is considered successful
- Task is marked as complete
- No further retries occur

**Failure Response (non-2xx):**
- HTTP status codes outside 200-299 trigger a retry
- Task is rescheduled with exponential backoff
- After max retries, sent to dead letter URL

**Connection Settings:**
- Timeout: 30 seconds
- Max idle connections: 100
- Max idle connections per host: 10
- Idle connection timeout: 90 seconds

### 2. Task Submission Flow

```
Client                  HTTP Handler              Domain Service           Redis
  │                          │                          │                    │
  │  POST /tasks             │                          │                    │
  ├─────────────────────────>│                          │                    │
  │                          │  CreateTask()            │                    │
  │                          ├─────────────────────────>│                    │
  │                          │                          │  Schedule(taskID, │
  │                          │                          │  nextRetry)        │
  │                          │                          ├───────────────────>│
  │                          │                          │                    │
  │                          │  task_id                 │                    │
  │                          │<─────────────────────────┤                    │
  │  201 {task_id}           │                          │                    │
  │<─────────────────────────┤                          │                    │
```

### 3. Retry Processing Flow

```
Worker                  Redis                   Domain Service           Kafka/HTTP
  │                       │                          │                       │
  │  Poll every 1s        │                          │                       │
  ├──────────────────────>│                          │                       │
  │  tasks due now        │                          │                       │
  │<──────────────────────┤                          │                       │
  │                       │                          │                       │
  │  ProcessTask()        │                          │                       │
  ├──────────────────────────────────────────────────>│                       │
  │                       │                          │  Send to destination  │
  │                       │                          ├──────────────────────>│
  │                       │                          │                       │
  │                       │  Schedule next retry     │                       │
  │                       │<─────────────────────────┤                       │
```

### 4. Retry Logic

**Exponential Backoff Formula:**
```
nextDelay = base_delay * (2 ^ (attempt - 1))
```

**Example with base_delay=10s:**
- Attempt 1: 10s delay (10 × 2^0 = 10)
- Attempt 2: 20s delay (10 × 2^1 = 20)
- Attempt 3: 40s delay (10 × 2^2 = 40)
- Attempt 4: 80s delay (10 × 2^3 = 80)
- Attempt 5: 160s delay (10 × 2^4 = 160)

**Priority Tasks:**
- Tasks with `is_priority: true` are currently treated the same as regular tasks
- Future enhancement: Priority queue implementation for faster processing

**Destination Types:**

**Kafka (`"kafka"`)**
- Messages sent to Kafka topics via Kafka producer
- Requires: `host`, `port`, `topic`
- Uses exponential backoff for retries
- Failed messages routed to dead letter topic

**HTTP (`"http"`)**
- Messages sent via HTTP POST to webhook endpoints
- Requires: `url`
- Message sent as JSON in request body
- Message key sent as `X-Message-Key` header
- HTTP status 2xx = success, others = failure
- Failed webhooks retried with exponential backoff
- Exhausted retries sent to dead letter URL

**After Max Retries Exceeded:**
- Task is sent to `dead_destination` topic
- Task is removed from Redis
- No further retries occur

### 5. Graceful Shutdown

```
SIGTERM/SIGINT received
         │
         ├─> Cancel context
         │
         ├─> Stop accepting new HTTP requests
         │
         ├─> Worker stops polling
         │
         ├─> Wait for in-flight tasks (max 10s)
         │
         ├─> Close Redis connection
         │
         ├─> Close Kafka producer
         │
         └─> Exit
```

## Testing

### Run All Tests

```bash
go test ./...
```

### Run Tests with Coverage

```bash
go test -cover ./...
```

### Generate Coverage Report

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out
```

### Run Specific Package Tests

```bash
# Domain service tests
go test ./internal/domain/service -v

# HTTP handler tests
go test ./internal/adapter/primary/http -v

# Worker tests
go test ./internal/adapter/primary/worker -v
```

### Run Tests with Race Detector

```bash
go test -race ./...
```

### Test Coverage by Package

| Package | Coverage |
|---------|----------|
| `domain/entity` | 100% |
| `domain/service` | 94.5% |
| `domain/valueobject` | 100% |
| `adapter/primary/http` | 84.6% |
| `adapter/primary/worker` | 100% |
| `config` | 100% |

## Development

### Code Quality

```bash
# Run linter
golangci-lint run

# Format code
go fmt ./...

# Vet code
go vet ./...

# Run all quality checks
go fmt ./... && go vet ./... && golangci-lint run && go test ./...
```

### Adding New Features

When adding features, follow the hexagonal architecture pattern:

1. **Define Domain Entity/Value Object** (`internal/domain/entity/`)
2. **Add Business Logic** (`internal/domain/service/`)
3. **Define Port Interface** (`internal/port/primary/` or `internal/port/secondary/`)
4. **Implement Adapter** (`internal/adapter/primary/` or `internal/adapter/secondary/`)
5. **Wire in DI Container** (`cmd/kafka-retry/container.go`)
6. **Add Tests** (co-located `*_test.go` files)

### Dependency Injection

All dependencies are wired in `cmd/kafka-retry/container.go`:

```go
// Register in this order:
// 1. Config
// 2. Infrastructure (Logger, Redis, Kafka)
// 3. Secondary Adapters
// 4. Domain Services
// 5. Primary Adapters
```

## Monitoring & Observability

### Structured Logging

All logs are JSON-formatted in production:

```json
{
  "level": "info",
  "ts": "2026-02-16T10:30:45.123Z",
  "caller": "service/task_service.go:45",
  "msg": "Task processing started",
  "task_id": "01JQXYZ123ABC456",
  "attempt": 2,
  "max_retries": 5
}
```

### Health Monitoring

```bash
# Continuous health check
watch -n 5 curl -s http://localhost:8080/health
```

### Metrics (Future Enhancement)

Consider adding Prometheus metrics:
- `kafka_retry_tasks_created_total`
- `kafka_retry_tasks_processed_total`
- `kafka_retry_tasks_failed_total`
- `kafka_retry_tasks_dead_lettered_total`
- `kafka_retry_processing_duration_seconds`

## Troubleshooting

### Application Won't Start

**Issue:** `failed to create redis client: connection refused`

**Solution:**
```bash
# Verify Redis is running
docker ps | grep redis

# Restart Redis
docker-compose restart redis
```

**Issue:** `failed to create kafka producer: connection refused`

**Solution:**
```bash
# Verify Kafka is running
docker ps | grep kafka

# Restart Kafka
docker-compose restart kafka
```

### Tasks Not Being Processed

**Issue:** Tasks created but never sent to Kafka

**Solution:**
```bash
# Check worker is running
docker-compose logs kafka-retry | grep "Worker started"

# Check Redis for pending tasks
docker exec -it kafkaretry-redis-1 redis-cli
> ZRANGE kafkaretry:tasks 0 -1 WITHSCORES
```

### High Memory Usage

**Issue:** Memory grows over time

**Solution:**
- Check for goroutine leaks: `go tool pprof http://localhost:8080/debug/pprof/goroutine`
- Verify graceful shutdown is working
- Monitor Redis connection pool

### Redis Connection Pool Exhausted

**Issue:** `connection pool timeout`

**Solution:**
```go
// Increase pool size in internal/adapter/secondary/redisstore/client.go
PoolSize: 100,
```

## Production Considerations

### Scaling

**Horizontal Scaling:**
- Run multiple instances behind a load balancer
- Each instance independently polls Redis
- Redis ZRANGE ensures no duplicate processing

**Vertical Scaling:**
- Increase worker goroutines (modify `internal/adapter/primary/worker/worker.go`)
- Increase Redis connection pool size
- Increase Kafka producer batch size

### Security

**Redis:**
```bash
# Enable authentication
export REDIS_PASSWORD=your-secure-password-here
```

**Kafka:**
```bash
# Use SASL/SSL
export KAFKA_BROKERS=kafka.prod:9093
# Add SASL config in internal/adapter/secondary/kafkaproducer/producer.go
```

### High Availability

**Redis:**
- Use Redis Sentinel or Redis Cluster
- Configure `REDIS_ADDR` with sentinel addresses

**Kafka:**
- Use multiple brokers: `kafka-1:9092,kafka-2:9092,kafka-3:9092`
- Configure producer acknowledgment: `acks=all`

### Monitoring

**Health Checks:**
```yaml
# Kubernetes liveness probe
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30
```

**Log Aggregation:**
- Ship JSON logs to ELK, Splunk, or Datadog
- Filter by `task_id` for distributed tracing

## Performance

### Benchmarks

```bash
# Run benchmarks
go test -bench=. ./internal/domain/service -benchmem
```

### Expected Throughput

- **Task Creation:** ~1000 tasks/sec (HTTP bottleneck)
- **Task Processing:** ~500 tasks/sec (Kafka bottleneck)
- **Redis Operations:** ~10,000 ops/sec

### Optimization Tips

1. **Batch Redis Reads:** Read multiple tasks per poll
2. **Kafka Compression:** Enable Snappy/LZ4 compression
3. **Connection Pooling:** Tune Redis pool size
4. **Worker Count:** Increase concurrent workers

## Contributing

### Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `golangci-lint` for linting
- Maintain test coverage >80%
- Document public APIs with GoDoc comments

### Pull Request Process

1. Create feature branch: `git checkout -b feature/my-feature`
2. Write tests for new code
3. Ensure all tests pass: `go test ./...`
4. Run linter: `golangci-lint run`
5. Submit PR with clear description

## License

[Add your license here]

## Support

For issues and questions:
- Open an issue on GitHub
- Contact: [your-email@example.com]

## Examples

The `examples/` directory contains ready-to-use scripts and tools:

- **`create-tasks.sh`** - Create sample tasks with Kafka and HTTP destinations
- **`webhook-receiver.go`** - Local HTTP server for testing webhooks
- **`examples/README.md`** - Detailed testing guide

See [examples/README.md](examples/README.md) for complete testing instructions.

## Testing HTTP Webhooks Locally

You can test HTTP webhook delivery using a local HTTP server:

**1. Start a simple webhook receiver:**

```bash
# Using Python
python3 -m http.server 8090

# Or using Node.js
npx http-server -p 8090
```

**2. Use a more sophisticated webhook testing tool:**

```bash
# Install webhook.site CLI or use ngrok
ngrok http 8090
```

**3. Create a task with HTTP destination:**

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-webhook-1",
    "source": "test",
    "destination": {
      "url": "http://localhost:8090/webhook"
    },
    "dead_destination": {
      "url": "http://localhost:8090/dlq"
    },
    "max_retries": 3,
    "base_delay": 2,
    "client_id": "test-client",
    "is_priority": false,
    "message_data": "{\"test\": \"data\"}",
    "destination_type": "http"
  }'
```

**4. Check the webhook receiver logs** to see the incoming POST request.

## Roadmap

- [x] Support for HTTP webhook destinations
- [ ] Add Prometheus metrics
- [ ] Implement distributed tracing (OpenTelemetry)
- [ ] Add admin API for task inspection
- [ ] Support for priority queues
- [ ] Batch processing support
- [ ] Web UI for monitoring
- [ ] Support for custom backoff strategies
- [ ] Integration with APM tools
- [ ] Support for additional destination types (SQS, PubSub, RabbitMQ)

---

Built with ❤️ using Go and Hexagonal Architecture
