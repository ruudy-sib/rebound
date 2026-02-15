# Rebound Quick Start Guide

Get started with Rebound in under 5 minutes!

## Choose Your Path

### Path A: Go Service (Embedded Package) ⭐

Best for: **New Go microservices at Brevo**

**Step 1:** Install dependencies
```bash
# Infrastructure
docker-compose up -d redis kafka

# Your go.mod
go get rebound/pkg/rebound
```

**Step 2:** Add to your service
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

    // Use it!
    rb.CreateTask(ctx, &rebound.Task{
        ID: "order-123",
        Source: "order-service",
        Destination: rebound.Destination{
            URL: "https://webhook.example.com",
        },
        MaxRetries: 5,
        BaseDelay: 10,
        ClientID: "order-service",
        MessageData: `{"order_id": 123}`,
        DestinationType: rebound.DestinationTypeHTTP,
    })
}
```

**Step 3:** Run your service
```bash
go run main.go
```

**Done!** ✅ Rebound is now embedded in your service.

**Next:** See [INTEGRATION.md](INTEGRATION.md) for production patterns.

---

### Path B: Non-Go Service (Standalone HTTP API)

Best for: **Node.js, Python, Ruby, or legacy services**

**Step 1:** Start Rebound service
```bash
# Start infrastructure + Rebound
docker-compose up -d

# Or run directly
go run cmd/rebound/main.go
```

**Step 2:** Create tasks via HTTP
```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "order-123",
    "source": "order-service",
    "destination": {
      "url": "https://webhook.example.com"
    },
    "dead_destination": {
      "url": "https://webhook.example.com/dlq"
    },
    "max_retries": 5,
    "base_delay": 10,
    "client_id": "order-service",
    "message_data": "{\"order_id\": 123}",
    "destination_type": "http"
  }'
```

**From Python:**
```python
import requests

requests.post('http://localhost:8080/tasks', json={
    'id': 'order-123',
    'source': 'order-service',
    'destination': {'url': 'https://webhook.example.com'},
    'max_retries': 5,
    'base_delay': 10,
    'client_id': 'order-service',
    'message_data': '{"order_id": 123}',
    'destination_type': 'http'
})
```

**From Node.js:**
```javascript
const axios = require('axios');

await axios.post('http://localhost:8080/tasks', {
  id: 'order-123',
  source: 'order-service',
  destination: { url: 'https://webhook.example.com' },
  max_retries: 5,
  base_delay: 10,
  client_id: 'order-service',
  message_data: JSON.stringify({ order_id: 123 }),
  destination_type: 'http'
});
```

**Done!** ✅ Your service can now use Rebound via HTTP.

**Next:** See [README.md](README.md) for full API documentation.

---

## Test It Out

### Start Webhook Receiver
```bash
# Terminal 1
go run examples/webhook-receiver.go
```

### Run Example
```bash
# Terminal 2
go run examples/01-basic-usage/main.go
```

You should see webhooks being delivered!

---

## Common Scenarios

### Scenario 1: Retry Failed Kafka Messages

```go
rb.CreateTask(ctx, &rebound.Task{
    ID: "kafka-msg-123",
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
    MaxRetries: 5,
    BaseDelay: 10,
    MessageData: `{"order_id": 123}`,
    DestinationType: rebound.DestinationTypeKafka,
})
```

### Scenario 2: Retry Failed HTTP Webhooks

```go
rb.CreateTask(ctx, &rebound.Task{
    ID: "webhook-456",
    Destination: rebound.Destination{
        URL: "https://partner-api.com/webhook",
    },
    DeadDestination: rebound.Destination{
        URL: "https://internal-api.com/webhooks/failed",
    },
    MaxRetries: 10,
    BaseDelay: 5,
    MessageData: `{"event": "order.created", "data": {...}}`,
    DestinationType: rebound.DestinationTypeHTTP,
})
```

### Scenario 3: Priority Tasks

```go
rb.CreateTask(ctx, &rebound.Task{
    ID: "urgent-789",
    IsPriority: true,  // Process with higher priority
    MaxRetries: 15,    // More retries
    BaseDelay: 5,      // Faster retry
    // ...
})
```

---

## Configuration

### Environment Variables

```bash
# Redis
export REDIS_ADDR=localhost:6379
export REDIS_PASSWORD=your-password
export REDIS_DB=0

# Kafka (optional, only if using Kafka destinations)
export KAFKA_BROKERS=localhost:9092

# Worker
export POLL_INTERVAL=1s

# Logging
export LOG_LEVEL=info
export ENVIRONMENT=production
```

### In Code (Embedded)

```go
cfg := &rebound.Config{
    RedisAddr:     os.Getenv("REDIS_ADDR"),
    RedisPassword: os.Getenv("REDIS_PASSWORD"),
    RedisDB:       0,
    KafkaBrokers:  strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
    PollInterval:  1 * time.Second,
    Logger:        logger,
}

rb, _ := rebound.New(cfg)
```

---

## Monitoring

### Health Check
```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "checks": {
    "redis": "connected"
  }
}
```

### Check Pending Tasks
```bash
# Count
docker exec kafkaretry-redis-1 redis-cli ZCARD kafkaretry:tasks

# List
docker exec kafkaretry-redis-1 redis-cli ZRANGE kafkaretry:tasks 0 -1
```

### View Logs
```bash
# Standalone service
docker-compose logs -f rebound

# Embedded (your app logs)
# Rebound uses your logger, so check your app logs
```

---

## Next Steps

### For Go Services (Embedded)
1. ✅ [INTEGRATION.md](INTEGRATION.md) - Production patterns
2. ✅ [examples/](examples/) - Real-world examples
3. ✅ [pkg/rebound/README.md](pkg/rebound/README.md) - API documentation

### For Non-Go Services (HTTP API)
1. ✅ [README.md](README.md) - Full API documentation
2. ✅ [openapi.yaml](openapi.yaml) - OpenAPI specification
3. ✅ [DEPLOYMENT.md](DEPLOYMENT.md) - Deployment options

### Understanding Deployment Options
1. ✅ [DEPLOYMENT.md](DEPLOYMENT.md) - Detailed comparison

---

## Troubleshooting

### "Connection refused" to Redis
```bash
# Check Redis is running
docker ps | grep redis

# Start it
docker-compose up -d redis
```

### Tasks not processing
```bash
# Check worker started (embedded)
# Look for "rebound worker started" in your app logs

# Check worker started (standalone)
docker-compose logs rebound | grep "Worker started"
```

### High memory usage
```bash
# Check pending task count
docker exec kafkaretry-redis-1 redis-cli ZCARD kafkaretry:tasks

# If too many, increase worker frequency or add more workers
```

---

## Performance Tips

### Embedded Package
- ✅ Use single Rebound instance per service (singleton)
- ✅ Reuse across handlers/workers
- ✅ Share Redis/Kafka connections with your app

### Standalone Service
- ✅ Run multiple replicas (3-5)
- ✅ Use load balancer
- ✅ Enable connection pooling in clients

### Both
- ✅ Tune Redis connection pool
- ✅ Monitor task queue depth
- ✅ Set appropriate `max_retries` per use case

---

## Example Gallery

Run these examples to see Rebound in action:

```bash
# Basic usage
go run examples/01-basic-usage/main.go

# Email service
go run examples/02-email-service/main.go

# Webhook delivery
go run examples/03-webhook-delivery/main.go

# DI integration
go run examples/04-di-integration/main.go

# Payment processing
go run examples/05-payment-processing/main.go

# Multi-tenant
go run examples/06-multi-tenant/main.go
```

See [examples/EXAMPLES.md](examples/EXAMPLES.md) for details.

---

## Questions?

- **How do I decide between embedded and standalone?** → [DEPLOYMENT.md](DEPLOYMENT.md)
- **How do I integrate with my Go service?** → [INTEGRATION.md](INTEGRATION.md)
- **What are best practices for Brevo?** → [INTEGRATION.md](INTEGRATION.md#best-practices-for-brevo)
- **How do I test Rebound?** → [examples/](examples/)
- **What's the API?** → [openapi.yaml](openapi.yaml) or [README.md](README.md)

---

**Ready to go?** Pick your path above and start building resilient systems! 🚀
