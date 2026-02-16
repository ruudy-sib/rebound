# Quick Start Guide

Get up and running with the Rebound consumer benchmark in under 5 minutes.

## Prerequisites

- Go 1.23 or higher
- Docker and Docker Compose
- Make (optional, but recommended)

## 1. Start Services

```bash
# Start Redis and Kafka
make docker-up

# Or manually:
cd ../.. && docker-compose up -d redis kafka
```

## 2. Run the Consumer Benchmark

```bash
# Using Make (recommended)
make run

# Or directly:
go run .
```

## 3. View Results

The consumer will run for 60 seconds by default and display statistics:

```
=== Final Statistics ===
{
  "TotalMessages": 587,
  "SuccessfulMessages": 411,
  "RetriedMessages": 176,
  "ErrorMessages": 0
}

Success Rate: 70.02%
Retry Rate: 29.98%
```

## 4. Run Benchmarks

```bash
# Quick benchmarks
make benchmark

# Comprehensive benchmark suite
make benchmark-all

# View results
cat benchmark-results-latest/summary.md
```

## Common Scenarios

### Test High Failure Rate

```bash
make run-high-failure
# Or: go run . -failure-rate 0.5
```

### Run Extended Test

```bash
make run-long
# Or: go run . -duration 5m
```

### Test Parallel Performance

```bash
make benchmark-parallel
```

## What's Happening?

1. **Producer** → Generates test messages every 100ms
2. **Consumer** → Processes messages with simulated failures
3. **Rebound** → Automatically retries failed messages with exponential backoff
4. **Statistics** → Tracks success/failure/retry rates

## Troubleshooting

### Redis Connection Failed

```bash
# Check if Redis is running
redis-cli -h localhost -p 6379 PING

# Should return: PONG
```

### Kafka Connection Failed

```bash
# Check if Kafka is running
docker ps | grep kafka

# Check logs
docker logs kafkaretry-poc-kafka-1
```

### No Messages Being Processed

```bash
# Check consumer logs in the console
# Verify the topic exists:
docker exec -it kafkaretry-poc-kafka-1 \
  kafka-topics --list --bootstrap-server localhost:9092
```

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Explore [consumer.go](consumer.go) to understand the implementation
- Check out other [examples](../) for different use cases
- Review benchmark results and optimize for your use case

## Quick Commands Reference

```bash
make help              # Show all available commands
make docker-up         # Start services
make docker-down       # Stop services
make run              # Run consumer with defaults
make benchmark        # Run benchmarks
make benchmark-all    # Run comprehensive benchmarks
make clean           # Clean up generated files
```

## Expected Performance

On a typical development machine:

- **Throughput:** 500-1000 messages/second
- **Task Creation:** ~1ms per task
- **Memory Usage:** ~50MB baseline
- **Success Rate:** ~70% (with 30% failure rate)

## Support

For questions or issues:
- Check [README.md](README.md) for detailed documentation
- Review [../../pkg/rebound/README.md](../../pkg/rebound/README.md) for library docs
- Open an issue on GitHub
