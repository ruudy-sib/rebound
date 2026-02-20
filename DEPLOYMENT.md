# Rebound Deployment Options

Rebound can be deployed in three ways, each with different trade-offs:

## Option 1: Embedded Package (Recommended for Go Services) ⭐

**Best for:** Go microservices at Brevo

### Architecture

```
┌─────────────────────────────────────┐
│     Your Go Application             │
│                                     │
│  ┌───────────────────────────────┐  │
│  │  Your Business Logic          │  │
│  └───────────┬───────────────────┘  │
│              │                      │
│              ▼                      │
│  ┌───────────────────────────────┐  │
│  │  Rebound Package              │  │
│  │  (pkg/rebound)                │  │
│  └───────────┬───────────────────┘  │
│              │                      │
└──────────────┼──────────────────────┘
               │
       ┌───────┴────────┐
       ▼                ▼
   [Redis]          [Kafka]
```

### Pros
- ✅ **Zero network overhead** - Direct function calls
- ✅ **Shared connections** - Reuses your Redis/Kafka clients
- ✅ **Type safety** - Compile-time guarantees
- ✅ **Better observability** - Unified logging, tracing, metrics
- ✅ **Simpler deployment** - One service instead of two
- ✅ **Lower latency** - No HTTP serialization/deserialization

### Cons
- ❌ **Go only** - Can't be used by non-Go services
- ❌ **Tightly coupled** - Rebound version tied to service deployment

### When to Use
- ✅ New Go microservices
- ✅ Services with high retry volume (>1000 tasks/sec)
- ✅ Services that need low latency
- ✅ Services already using Redis/Kafka

### Usage

```go
import "github.com/ruudy-sib/rebound/pkg/rebound"

// In your main.go
rb, err := rebound.New(&rebound.Config{
    RedisAddr: "redis:6379",
    // KafkaBrokers: []string{"kafka:9092"}, // optional: only for Kafka destinations
})
rb.Start(ctx)

// In your handlers/services
rb.CreateTask(ctx, &rebound.Task{
    ID: "order-123",
    Destination: rebound.Destination{
        URL: "https://webhook.example.com",
    },
    MaxRetries: 5,
    BaseDelay: 10,
    // ...
})
```

See: `INTEGRATION.md` for detailed patterns

---

## Option 2: Standalone HTTP Service

**Best for:** Non-Go services, legacy systems, simple use cases

### Architecture

```
┌─────────────────────┐         ┌──────────────────────┐
│  Your Application   │         │  Rebound Service     │
│  (Any Language)     │         │  (Standalone)        │
│                     │         │                      │
│  ┌──────────────┐   │  HTTP   │  ┌───────────────┐  │
│  │Business Logic├───┼────────►│  │ HTTP Handler  │  │
│  └──────────────┘   │         │  └───────┬───────┘  │
└─────────────────────┘         │          │          │
                                │          ▼          │
                                │  ┌───────────────┐  │
                                │  │Rebound Package│  │
                                │  └───────┬───────┘  │
                                └──────────┼──────────┘
                                           │
                                   ┌───────┴────────┐
                                   ▼                ▼
                               [Redis]          [Kafka]
```

### Pros
- ✅ **Language agnostic** - Works with any language
- ✅ **Decoupled** - Independent deployment and scaling
- ✅ **Centralized** - Single retry service for all apps
- ✅ **Simple client** - Just HTTP POST

### Cons
- ❌ **Network overhead** - HTTP round trip per task
- ❌ **Extra hop** - Additional latency
- ❌ **Extra service** - More infrastructure to manage
- ❌ **Serialization cost** - JSON encoding/decoding

### When to Use
- ✅ Non-Go services (Node.js, Python, Ruby, etc.)
- ✅ Legacy applications
- ✅ Low retry volume (<100 tasks/sec)
- ✅ Services that don't use Redis/Kafka
- ✅ Quick prototyping

### Deployment

**Docker:**
```bash
docker build -t rebound:latest .
docker run -d \
  --name rebound \
  -p 8080:8080 \
  -e REDIS_ADDR=redis:6379 \
  rebound:latest
  # Add -e KAFKA_BROKERS=kafka:9092 only if using Kafka destinations
```

**Docker Compose:**
```yaml
version: '3.8'
services:
  rebound:
    build: .
    ports:
      - "8080:8080"
    environment:
      - REDIS_ADDR=redis:6379
      # - KAFKA_BROKERS=kafka:9092  # optional: only for Kafka destinations
    depends_on:
      - redis
      - kafka
```

**Kubernetes:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rebound
spec:
  replicas: 3
  selector:
    matchLabels:
      app: rebound
  template:
    metadata:
      labels:
        app: rebound
    spec:
      containers:
      - name: rebound
        image: rebound:latest
        ports:
        - containerPort: 8080
        env:
        - name: REDIS_ADDR
          value: "redis.prod.svc.cluster.local:6379"
        - name: REDIS_MODE
          value: "standalone"  # or "sentinel" or "cluster"
        # - name: KAFKA_BROKERS  # optional: only if using Kafka destinations
        #   value: "kafka-1:9092,kafka-2:9092"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### Usage (HTTP API)

**Create Task:**
```bash
curl -X POST http://rebound:8080/tasks \
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

response = requests.post('http://rebound:8080/tasks', json={
    'id': 'order-123',
    'source': 'order-service',
    'destination': {'url': 'https://webhook.example.com'},
    'dead_destination': {'url': 'https://webhook.example.com/dlq'},
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

await axios.post('http://rebound:8080/tasks', {
  id: 'order-123',
  source: 'order-service',
  destination: { url: 'https://webhook.example.com' },
  dead_destination: { url: 'https://webhook.example.com/dlq' },
  max_retries: 5,
  base_delay: 10,
  client_id: 'order-service',
  message_data: JSON.stringify({ order_id: 123 }),
  destination_type: 'http'
});
```

---

## Option 3: Sidecar Pattern (Kubernetes)

**Best for:** Services that want local performance but HTTP interface

### Architecture

```
┌─────────────────────────────────────────────────┐
│                   Pod                           │
│                                                 │
│  ┌──────────────────┐    ┌──────────────────┐  │
│  │  Your Service    │    │  Rebound Sidecar │  │
│  │  (Any Language)  │    │                  │  │
│  │                  │HTTP│  ┌────────────┐  │  │
│  │  ┌────────────┐  ├───►│  │HTTP Handler│  │  │
│  │  │   Logic    │  │    │  └─────┬──────┘  │  │
│  │  └────────────┘  │    │        │         │  │
│  └──────────────────┘    │  ┌─────▼──────┐  │  │
│                          │  │Rebound Pkg │  │  │
│                          │  └─────┬──────┘  │  │
│                          └────────┼─────────┘  │
└───────────────────────────────────┼────────────┘
                                    │
                            ┌───────┴────────┐
                            ▼                ▼
                        [Redis]          [Kafka]
```

### Pros
- ✅ **Local network** - Very low latency (localhost)
- ✅ **Language agnostic** - HTTP interface
- ✅ **Isolated resources** - Per-pod resource limits
- ✅ **Kubernetes native** - Easy deployment

### Cons
- ❌ **Resource overhead** - Rebound in every pod
- ❌ **More containers** - Increased pod size
- ❌ **Duplicate workers** - One worker per pod

### When to Use
- ✅ Kubernetes environment
- ✅ Non-Go services needing low latency
- ✅ Services with strict network policies
- ✅ Multi-language microservices

### Deployment

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myservice
spec:
  containers:
  # Main application
  - name: app
    image: myservice:latest
    ports:
    - containerPort: 3000
    env:
    - name: REBOUND_URL
      value: "http://localhost:8080"

  # Rebound sidecar
  - name: rebound
    image: rebound:latest
    ports:
    - containerPort: 8080
    env:
    - name: REDIS_ADDR
      value: "redis.prod.svc.cluster.local:6379"
    - name: REDIS_MODE
      value: "standalone"  # or "sentinel" or "cluster"
    # - name: KAFKA_BROKERS  # optional: only if using Kafka destinations
    #   value: "kafka:9092"
    resources:
      requests:
        memory: "128Mi"
        cpu: "100m"
      limits:
        memory: "256Mi"
        cpu: "200m"
```

### Usage (Sidecar)

```javascript
// From your application
await axios.post('http://localhost:8080/tasks', {
  // same as standalone
});
```

---

## Comparison Matrix

| Feature | Embedded | Standalone | Sidecar |
|---------|----------|------------|---------|
| **Language** | Go only | Any | Any |
| **Latency** | ~1ms | ~10ms | ~2ms |
| **Network** | None | External | Localhost |
| **Deployment** | In-app | Separate | Per-pod |
| **Scaling** | With app | Independent | With app |
| **Resources** | Shared | Centralized | Per-pod |
| **Complexity** | Low | Medium | Medium |
| **Best for** | Go services | Multi-lang | Kubernetes |

---

## Decision Tree

```
Do you use Go?
├─ Yes → Is performance critical?
│         ├─ Yes → Use Embedded Package ⭐
│         └─ No → Use Embedded Package (still recommended)
│
└─ No → Are you on Kubernetes?
          ├─ Yes → Is latency critical?
          │         ├─ Yes → Use Sidecar
          │         └─ No → Use Standalone Service
          │
          └─ No → Use Standalone Service
```

---

## Migration Path

### From Standalone to Embedded

**Before:**
```javascript
// Node.js service calling HTTP API
await axios.post('http://rebound:8080/tasks', taskData);
```

**Rewrite in Go:**
```go
// Embedded package
rb.CreateTask(ctx, task)
```

**Benefits:** 10x faster, lower infrastructure cost

### From Embedded to Standalone

Sometimes you need to extract Rebound for legacy services:

**Before:**
```go
// Embedded
rb.CreateTask(ctx, task)
```

**Wrapper service:**
```go
// New HTTP wrapper
http.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
    var req CreateTaskRequest
    json.NewDecoder(r.Body).Decode(&req)
    rb.CreateTask(r.Context(), req.toTask())
})
```

---

## Performance Comparison

### Embedded Package
- **Throughput:** 5000 tasks/sec
- **Latency:** 1-2ms (p99)
- **Memory:** 50MB baseline

### Standalone Service
- **Throughput:** 1000 tasks/sec
- **Latency:** 10-20ms (p99)
- **Memory:** 50MB + HTTP overhead

### Sidecar
- **Throughput:** 3000 tasks/sec
- **Latency:** 2-5ms (p99)
- **Memory:** 128MB per pod

---

## Cost Comparison (AWS)

**Embedded Package:**
- No additional compute (uses existing service resources)
- Shared Redis/Kafka connections
- **Cost:** $0/month (marginal increase in service resources)

**Standalone Service (3 replicas):**
- 3 × t3.small instances
- Load balancer
- **Cost:** ~$75/month

**Sidecar (100 pods):**
- 100 × 128MB containers
- **Cost:** ~$50/month (in aggregate pod costs)

---

## Recommendations by Service Type

### New Go Microservices
→ **Embedded Package** ⭐
- Best performance
- Lowest operational overhead
- Type safety

### Legacy Services (Node.js, Python, Ruby)
→ **Standalone Service**
- No rewrite needed
- HTTP interface familiar
- Quick integration

### Kubernetes-Native Services
→ **Sidecar Pattern**
- Low latency
- Language agnostic
- Kubernetes native

### High-Volume Services (>10k tasks/sec)
→ **Embedded Package** ⭐
- Only option that scales to this level
- Minimal latency overhead

---

## Running Both Modes Simultaneously

You can run Rebound as both embedded and standalone:

```
┌─────────────────────┐
│ Go Service A        │
│ (Embedded)          │
└──────────┬──────────┘
           │
           ▼
       [Redis] ◄────────┐
           ▲            │
           │            │
┌──────────┴──────────┐ │
│ Rebound Service     │ │
│ (Standalone)        │ │
└──────────┬──────────┘ │
           ▲            │
           │            │
┌──────────┴──────────┐ │
│ Python Service B    │ │
│ (HTTP Client)       ├─┘
└─────────────────────┘
```

**Key:** All services share the same Redis, so tasks are processed regardless of how they were created.

---

## Next Steps

1. **For Go services:** Read `INTEGRATION.md` for embedded patterns
2. **For other languages:** See standalone HTTP API in `README.md`
3. **For Kubernetes:** Use sidecar helm chart (coming soon)

## Support

- **Embedded Package:** `pkg/rebound/README.md`
- **Standalone API:** `openapi.yaml`
- **Integration:** `INTEGRATION.md`
