# Rebound Reference Guide

## Advanced Configuration

### Multi-Environment Setup

```go
func NewReboundConfig(env string) *rebound.Config {
    cfg := &rebound.Config{
        PollInterval: 1 * time.Second,
    }

    switch env {
    case "production":
        cfg.RedisMode          = "sentinel"
        cfg.RedisMasterName    = "mymaster"
        cfg.RedisSentinelAddrs = []string{
            "sentinel-1.prod:26379",
            "sentinel-2.prod:26379",
            "sentinel-3.prod:26379",
        }
    case "staging":
        cfg.RedisAddr = "redis.staging:6379"
    default:
        cfg.RedisAddr = "localhost:6379"
    }

    return cfg
}
```

### Configuration from Environment Variables

```go
import "os"

func ConfigFromEnv() *rebound.Config {
    cfg := &rebound.Config{
        RedisMode:     getEnv("REDIS_MODE", "standalone"),
        RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
        RedisPassword: os.Getenv("REDIS_PASSWORD"),
        RedisDB:       getEnvInt("REDIS_DB", 0),
        PollInterval:  getEnvDuration("POLL_INTERVAL", 1*time.Second),
    }
    if v := os.Getenv("REDIS_MASTER_NAME"); v != "" {
        cfg.RedisMasterName = v
    }
    if v := os.Getenv("REDIS_SENTINEL_ADDRS"); v != "" {
        cfg.RedisSentinelAddrs = strings.Split(v, ",")
    }
    if v := os.Getenv("REDIS_CLUSTER_ADDRS"); v != "" {
        cfg.RedisClusterAddrs = strings.Split(v, ",")
    }
    return cfg
}

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}

func getEnvInt(key string, fallback int) int {
    if value := os.Getenv(key); value != "" {
        if i, err := strconv.Atoi(value); err == nil {
            return i
        }
    }
    return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if d, err := time.ParseDuration(value); err == nil {
            return d
        }
    }
    return fallback
}
```

## Advanced Retry Strategies

### Adaptive Retry Based on Error Type

```go
type RetryStrategy struct {
    MaxRetries int
    BaseDelay  int
}

func getRetryStrategy(err error) RetryStrategy {
    switch {
    case isNetworkError(err):
        return RetryStrategy{MaxRetries: 10, BaseDelay: 5}
    case isRateLimitError(err):
        return RetryStrategy{MaxRetries: 5, BaseDelay: 60}
    case isTemporaryError(err):
        return RetryStrategy{MaxRetries: 7, BaseDelay: 10}
    case isPermanentError(err):
        return RetryStrategy{MaxRetries: 0, BaseDelay: 0} // Don't retry
    default:
        return RetryStrategy{MaxRetries: 3, BaseDelay: 30}
    }
}

func scheduleRetryWithStrategy(ctx context.Context, rb *rebound.Rebound, err error, data []byte) error {
    strategy := getRetryStrategy(err)

    if strategy.MaxRetries == 0 {
        // Don't retry permanent errors - send directly to DLQ
        return sendToDLQ(ctx, data)
    }

    task := &rebound.Task{
        ID:          generateID(),
        MaxRetries:  strategy.MaxRetries,
        BaseDelay:   strategy.BaseDelay,
        MessageData: string(data),
        // ... other fields
    }

    return rb.CreateTask(ctx, task)
}
```

### Priority-Based Retry

```go
func createTaskWithPriority(ctx context.Context, rb *rebound.Rebound, priority string, data []byte) error {
    var maxRetries, baseDelay int

    switch priority {
    case "critical":
        maxRetries = 15
        baseDelay = 5  // Retry quickly
    case "high":
        maxRetries = 10
        baseDelay = 10
    case "normal":
        maxRetries = 5
        baseDelay = 30
    case "low":
        maxRetries = 3
        baseDelay = 60
    }

    task := &rebound.Task{
        ID:         generateID(),
        MaxRetries: maxRetries,
        BaseDelay:  baseDelay,
        IsPriority: priority == "critical" || priority == "high",
        MessageData: string(data),
        // ... other fields
    }

    return rb.CreateTask(ctx, task)
}
```

## Multi-Tenant Configuration

```go
type TenantConfig struct {
    TenantID   string
    Plan       string
    MaxRetries int
    BaseDelay  int
}

var tenantConfigs = map[string]TenantConfig{
    "tenant-enterprise": {Plan: "enterprise", MaxRetries: 15, BaseDelay: 5},
    "tenant-pro":        {Plan: "pro", MaxRetries: 10, BaseDelay: 10},
    "tenant-free":       {Plan: "free", MaxRetries: 3, BaseDelay: 30},
}

func createTaskForTenant(ctx context.Context, rb *rebound.Rebound, tenantID string, data []byte) error {
    cfg, exists := tenantConfigs[tenantID]
    if !exists {
        cfg = TenantConfig{MaxRetries: 3, BaseDelay: 30} // Default
    }

    task := &rebound.Task{
        ID:          fmt.Sprintf("%s-%s", tenantID, generateID()),
        Source:      tenantID,
        MaxRetries:  cfg.MaxRetries,
        BaseDelay:   cfg.BaseDelay,
        MessageData: string(data),
        // ... other fields
    }

    return rb.CreateTask(ctx, task)
}
```

## Batch Operations

### Parallel Task Creation

```go
func createTasksBatch(ctx context.Context, rb *rebound.Rebound, tasks []*rebound.Task) error {
    errChan := make(chan error, len(tasks))
    var wg sync.WaitGroup

    for _, task := range tasks {
        wg.Add(1)
        go func(t *rebound.Task) {
            defer wg.Done()
            if err := rb.CreateTask(ctx, t); err != nil {
                errChan <- err
            }
        }(task)
    }

    wg.Wait()
    close(errChan)

    // Collect errors
    var errs []error
    for err := range errChan {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("batch creation failed: %d errors", len(errs))
    }

    return nil
}
```

### Rate-Limited Batch Creation

```go
func createTasksRateLimited(ctx context.Context, rb *rebound.Rebound, tasks []*rebound.Task, rateLimit int) error {
    semaphore := make(chan struct{}, rateLimit)
    errChan := make(chan error, len(tasks))
    var wg sync.WaitGroup

    for _, task := range tasks {
        wg.Add(1)
        semaphore <- struct{}{} // Acquire

        go func(t *rebound.Task) {
            defer func() {
                <-semaphore // Release
                wg.Done()
            }()

            if err := rb.CreateTask(ctx, t); err != nil {
                errChan <- err
            }
        }(task)
    }

    wg.Wait()
    close(errChan)

    var errs []error
    for err := range errChan {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("rate-limited batch failed: %d errors", len(errs))
    }

    return nil
}
```

## Monitoring and Observability

### Custom Metrics Wrapper

```go
type InstrumentedRebound struct {
    rb              *rebound.Rebound
    tasksCreated    prometheus.Counter
    tasksSucceeded  prometheus.Counter
    tasksFailed     prometheus.Counter
    creationLatency prometheus.Histogram
}

func (ir *InstrumentedRebound) CreateTask(ctx context.Context, task *rebound.Task) error {
    start := time.Now()
    defer func() {
        ir.creationLatency.Observe(time.Since(start).Seconds())
    }()

    err := ir.rb.CreateTask(ctx, task)

    ir.tasksCreated.Inc()
    if err != nil {
        ir.tasksFailed.Inc()
        return err
    }

    ir.tasksSucceeded.Inc()
    return nil
}
```

### Distributed Tracing

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func createTaskWithTracing(ctx context.Context, rb *rebound.Rebound, task *rebound.Task) error {
    tracer := otel.Tracer("rebound")
    ctx, span := tracer.Start(ctx, "rebound.createTask",
        trace.WithAttributes(
            attribute.String("task.id", task.ID),
            attribute.String("task.source", task.Source),
            attribute.String("task.destination_type", string(task.DestinationType)),
            attribute.Int("task.max_retries", task.MaxRetries),
        ),
    )
    defer span.End()

    err := rb.CreateTask(ctx, task)
    if err != nil {
        span.RecordError(err)
    }

    return err
}
```

## Production Deployment Patterns

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: order-service
        image: order-service:latest
        env:
        - name: REDIS_MODE
          value: "standalone"  # or "sentinel" or "cluster"
        - name: REDIS_ADDR
          value: "redis.default.svc.cluster.local:6379"
        # Sentinel HA:
        # - name: REDIS_MASTER_NAME
        #   value: "mymaster"
        # - name: REDIS_SENTINEL_ADDRS
        #   value: "sentinel-1:26379,sentinel-2:26379"
        # Kafka (optional):
        # - name: KAFKA_BROKERS
        #   value: "kafka-1:9092,kafka-2:9092,kafka-3:9092"
        - name: POLL_INTERVAL
          value: "1s"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### Health Checks

```go
func (s *Service) HealthCheck(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    // Check Redis connectivity
    if err := s.redisClient.Ping(ctx).Err(); err != nil {
        http.Error(w, "redis unhealthy", http.StatusServiceUnavailable)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
        "rebound": "operational",
    })
}
```

## Testing Patterns

### Table-Driven Tests

```go
func TestRetryStrategies(t *testing.T) {
    tests := []struct {
        name          string
        errorType     error
        wantRetries   int
        wantBaseDelay int
    }{
        {"network error", networkError, 10, 5},
        {"rate limit", rateLimitError, 5, 60},
        {"permanent error", permanentError, 0, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            strategy := getRetryStrategy(tt.errorType)
            assert.Equal(t, tt.wantRetries, strategy.MaxRetries)
            assert.Equal(t, tt.wantBaseDelay, strategy.BaseDelay)
        })
    }
}
```

### Integration Test with Testcontainers

```go
import "github.com/testcontainers/testcontainers-go"

func TestReboundWithRedis(t *testing.T) {
    ctx := context.Background()

    // Start Redis container
    redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "redis:7.2-alpine",
            ExposedPorts: []string{"6379/tcp"},
        },
        Started: true,
    })
    require.NoError(t, err)
    defer redisC.Terminate(ctx)

    host, _ := redisC.Host(ctx)
    port, _ := redisC.MappedPort(ctx, "6379")

    // Test with real Redis
    cfg := &rebound.Config{
        RedisAddr: fmt.Sprintf("%s:%s", host, port.Port()),
    }

    rb, err := rebound.New(cfg)
    require.NoError(t, err)
    defer rb.Close()

    // Run tests...
}
```

## Performance Optimization

### Connection Pooling

Redis connections are pooled by default via go-redis. For high-throughput scenarios:

```go
import goredis "github.com/redis/go-redis/v9"

// Tune connection pool (if needed)
redisClient := goredis.NewClient(&goredis.Options{
    Addr:         cfg.RedisAddr,
    PoolSize:     20,  // Default: 10 * runtime.NumCPU()
    MinIdleConns: 10,  // Keep connections warm
    MaxRetries:   3,
})
```

### Payload Compression

For large payloads:

```go
import "compress/gzip"

func compressPayload(data []byte) ([]byte, error) {
    var buf bytes.Buffer
    gw := gzip.NewWriter(&buf)

    if _, err := gw.Write(data); err != nil {
        return nil, err
    }

    if err := gw.Close(); err != nil {
        return nil, err
    }

    return buf.Bytes(), nil
}

// Use in task creation
compressed, _ := compressPayload(largeData)
task.MessageData = base64.StdEncoding.EncodeToString(compressed)
```

## Troubleshooting Guide

### Debug Logging

Enable debug logging:

```go
logger, _ := zap.NewDevelopment() // Development logger with debug level

cfg := &rebound.Config{
    Logger: logger,
    // ...
}
```

### Redis Inspection

```bash
# Count pending tasks
redis-cli ZCARD retry:schedule:

# View scheduled tasks
redis-cli ZRANGE retry:schedule: 0 10 WITHSCORES

# Check task details
redis-cli HGETALL retry:task:{task_id}
```

### Common Issues

1. **Tasks not processing**: Ensure `rb.Start(ctx)` was called
2. **High memory usage**: Check pending task count, reduce payload sizes
3. **Slow task creation**: Use parallel creation, optimize Redis latency
4. **Missing tasks**: Check logs for creation errors, verify Redis connectivity

## Migration Checklist

Migrating from manual retry logic to Rebound:

- [ ] Identify all manual retry loops in codebase
- [ ] Design retry strategies (critical, high, normal, low)
- [ ] Configure Redis connection
- [ ] Set up Kafka brokers (optional: only if using Kafka destinations)
- [ ] Choose Redis mode: standalone / sentinel / cluster
- [ ] Implement Rebound initialization
- [ ] Replace manual retry with `CreateTask` calls
- [ ] Configure dead letter destinations
- [ ] Add monitoring and logging
- [ ] Write integration tests
- [ ] Performance test with expected load
- [ ] Deploy to staging
- [ ] Monitor DLQ for unexpected failures
- [ ] Deploy to production
- [ ] Set up alerts for DLQ size
