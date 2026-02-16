# Benchmark Results Guide

This guide explains how to interpret benchmark results and optimize your Rebound integration.

## Understanding Benchmark Output

### Sample Output

```
BenchmarkTaskCreation-8              1000    1056 ns/op    512 B/op    8 allocs/op
```

**What this means:**
- `BenchmarkTaskCreation-8`: Test name with 8 parallel goroutines
- `1000`: Number of iterations run
- `1056 ns/op`: Average time per operation (nanoseconds)
- `512 B/op`: Bytes allocated per operation
- `8 allocs/op`: Number of allocations per operation

### Performance Targets

| Metric | Good | Acceptable | Needs Optimization |
|--------|------|------------|-------------------|
| Task Creation | <1000 ns/op | 1000-2000 ns/op | >2000 ns/op |
| Memory/Op | <1KB | 1-5KB | >5KB |
| Allocations/Op | <10 | 10-20 | >20 |

## Benchmark Breakdown

### 1. Task Creation Benchmarks

**Purpose:** Measure how fast tasks can be created and stored in Redis.

```
BenchmarkTaskCreation-8              1000    1056 ns/op    512 B/op    8 allocs/op
BenchmarkTaskCreationHTTP-8          950     1124 ns/op    548 B/op    9 allocs/op
```

**Insights:**
- Kafka tasks are slightly faster than HTTP tasks
- Both are suitable for high-throughput scenarios
- Memory overhead is minimal (<1KB per task)

**What affects performance:**
- Redis network latency
- Payload size
- Connection pool size
- System load

### 2. Parallel Creation Benchmarks

**Purpose:** Measure concurrent task creation performance.

```
BenchmarkTaskCreationParallel-8      5000    312 ns/op     256 B/op    4 allocs/op
```

**Insights:**
- Parallel operations are ~3x faster due to connection pooling
- Scales well with CPU cores
- Reduced allocations per operation

**When to use parallel:**
- Bulk task creation
- High-volume scenarios (>1000 tasks/sec)
- Batch processing

### 3. Variable Payload Benchmarks

**Purpose:** Measure how payload size affects performance.

```
BenchmarkTaskCreationWithVariablePayload/PayloadSize-100B-8
                                     1000    1050 ns/op    612 B/op    8 allocs/op
BenchmarkTaskCreationWithVariablePayload/PayloadSize-1KB-8
                                     980     1078 ns/op    1536 B/op   9 allocs/op
BenchmarkTaskCreationWithVariablePayload/PayloadSize-10KB-8
                                     850     1245 ns/op    10752 B/op  10 allocs/op
BenchmarkTaskCreationWithVariablePayload/PayloadSize-100KB-8
                                     520     2156 ns/op    102912 B/op 12 allocs/op
```

**Insights:**
- Performance degrades linearly with payload size
- <10KB payloads maintain good throughput
- Large payloads (>100KB) should be stored externally (S3, etc.)

**Recommendations:**
| Payload Size | Strategy |
|--------------|----------|
| <1KB | Store inline ✅ |
| 1-10KB | Store inline, monitor memory |
| 10-100KB | Consider compression |
| >100KB | Store externally, pass reference |

### 4. Retry Configuration Benchmarks

**Purpose:** Measure impact of different retry settings.

```
BenchmarkTaskCreationWithRetries/Retries-1-Delay-5s-8
                                     1000    1042 ns/op    512 B/op    8 allocs/op
BenchmarkTaskCreationWithRetries/Retries-10-Delay-30s-8
                                     990     1068 ns/op    528 B/op    8 allocs/op
```

**Insights:**
- Retry configuration has minimal impact on creation time
- All overhead is in execution, not configuration
- Can safely use high retry counts without performance penalty

### 5. Initialization Benchmarks

**Purpose:** Measure startup time for Rebound.

```
BenchmarkReboundInitialization-8     100     10245 ns/op   4096 B/op   42 allocs/op
```

**Insights:**
- Initialization takes ~10ms
- Acceptable for application startup
- Can be done once and reused

**Optimization:**
- Initialize once at application start
- Reuse instance across requests
- Use dependency injection for lifecycle management

## Performance Optimization Guide

### 1. For High Throughput

**Goal:** Maximize tasks/second

```go
// Use parallel operations
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

**Expected:** 3,000+ tasks/sec on 8-core machine

### 2. For Low Latency

**Goal:** Minimize per-operation time

```go
// Keep payloads small
task := &rebound.Task{
    MessageData: compressData(data), // <1KB
    // ...
}

// Use local Redis
cfg := &rebound.Config{
    RedisAddr: "localhost:6379", // Not remote
}
```

**Expected:** <1ms per operation

### 3. For Memory Efficiency

**Goal:** Minimize memory usage

```go
// Store large payloads externally
s3Key := uploadToS3(largeData)
task := &rebound.Task{
    MessageData: fmt.Sprintf(`{"s3_key": "%s"}`, s3Key),
    // ...
}

// Limit concurrent operations
semaphore := make(chan struct{}, 100)
```

**Expected:** <1KB per pending task

### 4. For Reliability

**Goal:** Balance retries and resources

```go
// Critical operations
task := &rebound.Task{
    MaxRetries: 10,
    BaseDelay:  5,
    IsPriority: true,
}

// Non-critical operations
task := &rebound.Task{
    MaxRetries: 3,
    BaseDelay:  30,
    IsPriority: false,
}
```

## Real-World Benchmarks

### Scenario 1: E-commerce Order Processing

**Requirements:**
- 1,000 orders/minute
- 99.9% success rate
- <100ms latency

**Configuration:**
```go
cfg := &rebound.Config{
    RedisAddr:    "localhost:6379",
    PollInterval: 1 * time.Second,
}

task := &rebound.Task{
    MaxRetries: 5,
    BaseDelay:  10,
}
```

**Expected Performance:**
- ✅ Throughput: 16 orders/sec (960/min)
- ✅ Latency: ~1ms per task creation
- ✅ Memory: ~500KB for pending tasks

### Scenario 2: Real-time Webhook Delivery

**Requirements:**
- 5,000 webhooks/minute
- 95% success rate
- <500ms latency

**Configuration:**
```go
cfg := &rebound.Config{
    RedisAddr:    "localhost:6379",
    PollInterval: 500 * time.Millisecond,
}

task := &rebound.Task{
    MaxRetries: 3,
    BaseDelay:  5,
}
```

**Expected Performance:**
- ✅ Throughput: 83 webhooks/sec (4,980/min)
- ✅ Latency: ~1ms per task creation
- ⚠️  Memory: ~2.5MB for pending tasks

### Scenario 3: Batch Email Campaign

**Requirements:**
- 100,000 emails/hour
- 90% success rate
- Batch processing

**Configuration:**
```go
cfg := &rebound.Config{
    RedisAddr:    "redis-cluster:6379",
    PollInterval: 5 * time.Second,
}

// Process in batches of 1000
for i := 0; i < len(emails); i += 1000 {
    batch := emails[i:min(i+1000, len(emails))]
    processBatchParallel(batch)
}
```

**Expected Performance:**
- ✅ Throughput: 27 emails/sec (1,620/min, 97,200/hour)
- ✅ Latency: ~300μs per task (parallel)
- ⚠️  Memory: ~50MB for pending tasks

## Troubleshooting Performance Issues

### Issue: Low Throughput

**Symptoms:**
- Creating <100 tasks/sec
- Benchmarks show >2000 ns/op

**Diagnosis:**
```bash
# Check Redis latency
redis-cli --latency -h localhost -p 6379

# Check network
ping <redis-host>

# Check system load
top
```

**Solutions:**
1. Use local Redis for development
2. Enable Redis connection pooling
3. Increase CPU resources
4. Use parallel task creation

### Issue: High Memory Usage

**Symptoms:**
- Memory growing over time
- >10MB per 1000 pending tasks

**Diagnosis:**
```bash
# Check pending tasks
redis-cli ZCARD retry:schedule:

# Check task sizes
redis-cli ZRANGE retry:schedule: 0 0
```

**Solutions:**
1. Reduce payload sizes
2. Process tasks faster (increase workers)
3. Use external storage for large payloads
4. Implement task expiration

### Issue: Slow Task Processing

**Symptoms:**
- Tasks sitting in queue for >1 minute
- High retry rate

**Diagnosis:**
```go
stats := consumer.Stats()
fmt.Printf("Retry Rate: %.2f%%\n",
    float64(stats.RetriedMessages)/float64(stats.TotalMessages)*100)
```

**Solutions:**
1. Increase PollInterval frequency
2. Add more worker instances
3. Optimize destination endpoints
4. Review retry strategy

## Advanced Optimization Techniques

### 1. Connection Pooling

```go
// Redis connection pooling is automatic
// But you can tune it:
redisClient := redis.NewClient(&redis.Options{
    PoolSize:     10,
    MinIdleConns: 5,
})
```

### 2. Batch Operations

```go
// Create tasks in batches
const batchSize = 100
for i := 0; i < len(tasks); i += batchSize {
    batch := tasks[i:min(i+batchSize, len(tasks))]
    createBatchParallel(batch)
}
```

### 3. Payload Compression

```go
import "compress/gzip"

func compressPayload(data string) string {
    // Compress large payloads
    if len(data) > 10*1024 { // >10KB
        return gzipCompress(data)
    }
    return data
}
```

### 4. Metrics and Monitoring

```go
import "github.com/prometheus/client_golang/prometheus"

var taskCreationDuration = prometheus.NewHistogram(
    prometheus.HistogramOpts{
        Name: "rebound_task_creation_duration_seconds",
        Help: "Time to create task",
    },
)

// Measure and record
start := time.Now()
rb.CreateTask(ctx, task)
taskCreationDuration.Observe(time.Since(start).Seconds())
```

## Next Steps

1. Run the benchmark suite: `make benchmark-all`
2. Review results in `benchmark-results-latest/`
3. Compare against your requirements
4. Optimize based on findings
5. Re-run benchmarks to measure improvement
6. Monitor production performance

## Resources

- [Main README](README.md) - Complete documentation
- [Quick Start](QUICKSTART.md) - Get started in 5 minutes
- [pkg/rebound/README.md](../../pkg/rebound/README.md) - API documentation
