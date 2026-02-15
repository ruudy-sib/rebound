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
│  ┌──────────────────┐              ┌──────────────────┐       │
│  │  Redis Adapter   │              │  Kafka Adapter   │       │
│  │  - Sorted Set    │              │  - Producer      │       │
│  │  - Health Check  │              │                  │       │
│  └──────────────────┘              └──────────────────┘       │
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
│       └── secondary/         # Output adapters
│           ├── kafkaproducer/ # Kafka producer
│           └── redisstore/    # Redis scheduler
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

Submit a task to be retried with exponential backoff.

**Endpoint:** `POST /tasks`

**Request Body:**
```json
{
  "payload": "{\"user_id\": 123, \"action\": \"send_email\"}",
  "destination": {
    "topic": "user-events",
    "partition": 0,
    "kafka_conn_string": "localhost:9092"
  },
  "dead_destination": {
    "topic": "user-events-dlq",
    "partition": 0,
    "kafka_conn_string": "localhost:9092"
  },
  "max_retries": 5,
  "retry_delay_seconds": 10
}
```

**Field Descriptions:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `payload` | string | Yes | JSON string of the message to retry |
| `destination.topic` | string | Yes | Kafka topic for retries |
| `destination.partition` | int | Yes | Kafka partition (typically 0) |
| `destination.kafka_conn_string` | string | Yes | Kafka broker address |
| `dead_destination.topic` | string | No | Dead letter queue topic |
| `dead_destination.partition` | int | No | Dead letter queue partition |
| `dead_destination.kafka_conn_string` | string | No | Dead letter queue broker |
| `max_retries` | int | Yes | Maximum retry attempts |
| `retry_delay_seconds` | int | Yes | Base delay between retries |

**Example with cURL:**

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "payload": "{\"user_id\": 123, \"order_id\": 456}",
    "destination": {
      "topic": "orders",
      "partition": 0,
      "kafka_conn_string": "localhost:9092"
    },
    "dead_destination": {
      "topic": "orders-dlq",
      "partition": 0,
      "kafka_conn_string": "localhost:9092"
    },
    "max_retries": 3,
    "retry_delay_seconds": 5
  }'
```

**Success Response (201 Created):**
```json
{
  "task_id": "01JQXYZ123ABC456DEF789GH",
  "message": "Task scheduled successfully"
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

### 1. Task Submission Flow

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

### 2. Retry Processing Flow

```
Worker                  Redis                   Domain Service           Kafka
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

### 3. Retry Logic

**Exponential Backoff Formula:**
```
nextDelay = baseDelay * (2 ^ attemptNumber)
```

**Example with baseDelay=10s:**
- Attempt 1: 10s delay
- Attempt 2: 20s delay
- Attempt 3: 40s delay
- Attempt 4: 80s delay
- Attempt 5: 160s delay

**After Max Retries Exceeded:**
- Task is sent to `dead_destination` topic
- Task is removed from Redis
- No further retries occur

### 4. Graceful Shutdown

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

## Roadmap

- [ ] Add Prometheus metrics
- [ ] Implement distributed tracing (OpenTelemetry)
- [ ] Add admin API for task inspection
- [ ] Support for priority queues
- [ ] Batch processing support
- [ ] Web UI for monitoring
- [ ] Support for custom backoff strategies
- [ ] Integration with APM tools

---

Built with ❤️ using Go and Hexagonal Architecture
