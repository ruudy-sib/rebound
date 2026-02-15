# Rebound Examples

Comprehensive examples showing how to use Rebound in real-world scenarios.

## Prerequisites

```bash
# Start infrastructure
docker-compose up -d

# Start webhook receiver (for HTTP examples)
go run examples/webhook-receiver.go
```

## Examples Overview

### 01. Basic Usage
**File:** `01-basic-usage/main.go`

The simplest way to use Rebound. Shows basic configuration and task creation for both Kafka and HTTP destinations.

**What you'll learn:**
- Creating a Rebound instance
- Starting the worker
- Creating Kafka tasks
- Creating HTTP webhook tasks

**Run it:**
```bash
go run examples/01-basic-usage/main.go
```

**Output:**
```
✓ Rebound started successfully
✓ Kafka task created: kafka-example-1234567890
✓ HTTP task created: webhook-example-1234567890
```

---

### 02. Email Service Integration
**File:** `02-email-service/main.go`

Real-world email sending service with automatic retry on failures. Demonstrates bulk email sending with simulated failures.

**What you'll learn:**
- Building a service around Rebound
- Handling failures gracefully
- Bulk operations with retry
- Serializing complex data structures

**Run it:**
```bash
go run examples/02-email-service/main.go
```

**Output:**
```
📊 Results:
   ✓ Sent successfully: 4
   ⚠ Failed (retrying): 1

💡 Failed emails are being retried with exponential backoff
```

**Use case:** Send transactional emails with automatic retry on SMTP failures.

---

### 03. Webhook Delivery Service
**File:** `03-webhook-delivery/main.go`

Customer webhook delivery with different retry strategies based on event importance.

**What you'll learn:**
- Prioritizing critical events
- Broadcasting to multiple webhooks
- Dynamic retry policies
- Webhook security (headers, secrets)

**Run it:**
```bash
go run examples/03-webhook-delivery/main.go
```

**Output:**
```
📡 Broadcasting event: contact.created
📡 Broadcasting event: payment.succeeded (priority)

💡 Retry Strategy:
   - Regular events: 10 retries, 30s base delay
   - Critical events: 15 retries, 10s base delay
```

**Use case:** SaaS platform delivering webhooks to customer endpoints.

---

### 04. Dependency Injection Integration
**File:** `04-di-integration/main.go`

Complete application structure using uber-go/dig for dependency injection. Production-ready pattern for Brevo services.

**What you'll learn:**
- DI container setup
- Lifecycle management
- Graceful shutdown
- HTTP server integration
- Signal handling

**Run it:**
```bash
go run examples/04-di-integration/main.go
```

**Test it:**
```bash
curl -X POST http://localhost:8080/orders
```

**Use case:** Production service architecture for Brevo applications.

---

### 05. Payment Processing with Smart Retry
**File:** `05-payment-processing/main.go`

Payment service with intelligent retry strategies based on failure type.

**What you'll learn:**
- Failure type classification
- Adaptive retry strategies
- Permanent vs transient failures
- Rate limiting awareness

**Run it:**
```bash
go run examples/05-payment-processing/main.go
```

**Output:**
```
┌─────────────────────────┬──────────┬───────────┐
│ Failure Type            │ Retries  │ Base Delay│
├─────────────────────────┼──────────┼───────────┤
│ Network Error           │    10    │    5s     │
│ Rate Limited            │     5    │   60s     │
│ Insufficient Funds      │     3    │  300s     │
│ Invalid Card            │     0    │    -      │
└─────────────────────────┴──────────┴───────────┘
```

**Use case:** Payment gateway integration with smart failure handling.

---

### 06. Multi-Tenant Service
**File:** `06-multi-tenant/main.go`

Multi-tenant SaaS platform with different retry policies per tenant plan.

**What you'll learn:**
- Tenant-specific configuration
- Plan-based resource allocation
- Fair retry distribution
- Isolated failure domains

**Run it:**
```bash
go run examples/06-multi-tenant/main.go
```

**Output:**
```
┌──────────────┬──────────────────┬────────────┬──────────┐
│ Tenant ID    │ Name             │ Plan       │ Retries  │
├──────────────┼──────────────────┼────────────┼──────────┤
│ tenant-001   │ Acme Corp        │ enterprise │    15    │
│ tenant-002   │ TechStart Inc    │ pro        │    10    │
│ tenant-004   │ Startup XYZ      │ free       │     3    │
└──────────────┴──────────────────┴────────────┴──────────┘
```

**Use case:** SaaS platform with tiered service levels.

---

## Quick Comparison

| Example | Complexity | Best For | Key Feature |
|---------|-----------|----------|-------------|
| 01 - Basic | ⭐ | Learning Rebound | Simple setup |
| 02 - Email | ⭐⭐ | Transactional ops | Bulk operations |
| 03 - Webhooks | ⭐⭐ | Event delivery | Priority handling |
| 04 - DI | ⭐⭐⭐ | Production apps | Full lifecycle |
| 05 - Payments | ⭐⭐⭐ | Complex logic | Smart retry |
| 06 - Multi-tenant | ⭐⭐⭐ | SaaS platforms | Tenant isolation |

---

## Running Multiple Examples

Run examples simultaneously to see concurrent workload handling:

```bash
# Terminal 1: Webhook receiver
go run examples/webhook-receiver.go

# Terminal 2: Email service
go run examples/02-email-service/main.go

# Terminal 3: Webhook delivery
go run examples/03-webhook-delivery/main.go

# Terminal 4: Monitor
watch -n 1 'docker exec redis redis-cli ZCARD retry:schedule:'
```

---

## Common Patterns

### Pattern 1: Service Wrapper
```go
type MyService struct {
    rebound *rebound.Rebound
    logger  *zap.Logger
}

func (s *MyService) DoWork(ctx context.Context) error {
    if err := s.tryOperation(); err != nil {
        return s.scheduleRetry(ctx, data)
    }
    return nil
}
```

### Pattern 2: Adaptive Retry
```go
func (s *Service) getRetryStrategy(failure FailureType) RetryStrategy {
    switch failure {
    case TransientError:
        return RetryStrategy{MaxRetries: 10, BaseDelay: 5}
    case PermanentError:
        return RetryStrategy{MaxRetries: 0}
    }
}
```

### Pattern 3: Tenant Policies
```go
func (s *Service) GetPolicy(tenant *Tenant) RetryPolicy {
    switch tenant.Plan {
    case "enterprise":
        return RetryPolicy{MaxRetries: 15, IsPriority: true}
    }
}
```

---

## Monitoring

```bash
# Count pending tasks
docker exec redis redis-cli ZCARD retry:schedule:

# Watch processing
docker-compose logs -f rebound | grep "task"

# View task details
docker exec redis redis-cli ZRANGE retry:schedule: 0 -1 WITHSCORES
```

---

## Troubleshooting

**Redis connection failed:**
```bash
docker-compose restart redis
```

**Tasks not processing:**
```bash
docker-compose logs rebound | grep "Worker started"
```

**Webhook receiver down:**
```bash
go run examples/webhook-receiver.go
```

---

## Next Steps

1. Start with `01-basic-usage`
2. Pick the example closest to your use case
3. Modify and experiment
4. Use `04-di-integration` as production template
5. Read `INTEGRATION.md` for Brevo patterns

---

## Learn More

- **Package API:** `pkg/rebound/README.md`
- **Integration Guide:** `INTEGRATION.md`
- **Main README:** `README.md`
