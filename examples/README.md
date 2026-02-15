# Examples

This directory contains example scripts and tools for testing the Kafka Retry Service.

## Files

- **`create-tasks.sh`** - Shell script demonstrating task creation with Kafka and HTTP destinations
- **`webhook-receiver.go`** - Simple HTTP server for testing webhook deliveries

## Quick Start

### 1. Start the Infrastructure and Application

```bash
# From the project root
docker-compose up -d
go run cmd/kafka-retry/main.go
```

### 2. Start the Webhook Receiver (for HTTP testing)

In a separate terminal:

```bash
# From the examples directory
cd examples
go run webhook-receiver.go
```

The webhook receiver will start on `http://localhost:8090` with the following endpoints:

- **`POST /webhook`** - Always returns 200 OK (simulates successful webhook)
- **`POST /dlq`** - Dead letter queue endpoint
- **`POST /fail`** - Always returns 500 (simulates failed webhook, triggers retries)

### 3. Create Test Tasks

In another terminal:

```bash
# From the examples directory
cd examples
./create-tasks.sh
```

This script creates three example tasks:
1. **Kafka task** - Sends message to Kafka topic
2. **HTTP task** - Sends webhook to HTTP endpoint
3. **Priority Kafka task** - High-priority Kafka message

## Manual Testing

### Test HTTP Webhook Success

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "http-success-test",
    "source": "manual-test",
    "destination": {
      "url": "http://localhost:8090/webhook"
    },
    "dead_destination": {
      "url": "http://localhost:8090/dlq"
    },
    "max_retries": 3,
    "base_delay": 2,
    "client_id": "test",
    "is_priority": false,
    "message_data": "{\"test\": \"success\"}",
    "destination_type": "http"
  }'
```

**Expected behavior:**
- Task created successfully
- Worker picks up task after ~1 second
- HTTP POST sent to `http://localhost:8090/webhook`
- Webhook receiver logs the request
- Task completes successfully (no retries)

### Test HTTP Webhook Failure (with Retries)

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "http-failure-test",
    "source": "manual-test",
    "destination": {
      "url": "http://localhost:8090/fail"
    },
    "dead_destination": {
      "url": "http://localhost:8090/dlq"
    },
    "max_retries": 3,
    "base_delay": 2,
    "client_id": "test",
    "is_priority": false,
    "message_data": "{\"test\": \"failure\"}",
    "destination_type": "http"
  }'
```

**Expected behavior:**
- Task created successfully
- Attempt 1: Fails (500 error), retry scheduled in 2s
- Attempt 2: Fails (500 error), retry scheduled in 4s
- Attempt 3: Fails (500 error), retry scheduled in 8s
- Attempt 4: Fails (500 error), max retries exceeded
- Task sent to dead letter queue (`http://localhost:8090/dlq`)

### Test Kafka Message

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "kafka-test",
    "source": "manual-test",
    "destination": {
      "host": "localhost",
      "port": "9092",
      "topic": "test-topic"
    },
    "dead_destination": {
      "host": "localhost",
      "port": "9092",
      "topic": "test-topic-dlq"
    },
    "max_retries": 3,
    "base_delay": 2,
    "client_id": "test",
    "is_priority": false,
    "message_data": "{\"test\": \"kafka\"}",
    "destination_type": "kafka"
  }'
```

**Expected behavior:**
- Task created successfully
- Worker picks up task
- Message sent to Kafka topic `test-topic`
- Task completes successfully

## Monitoring

### View Application Logs

```bash
docker-compose logs -f kafka-retry
```

### Check Redis Queue

```bash
# See all pending tasks
docker exec -it kafkaretry-redis-1 redis-cli ZRANGE kafkaretry:tasks 0 -1 WITHSCORES

# Count pending tasks
docker exec -it kafkaretry-redis-1 redis-cli ZCARD kafkaretry:tasks

# Clear all tasks (for testing)
docker exec -it kafkaretry-redis-1 redis-cli DEL kafkaretry:tasks
```

### Consume Kafka Messages

```bash
# Create the test topic
docker exec -it kafkaretry-kafka-1 kafka-topics \
  --create \
  --topic test-topic \
  --bootstrap-server localhost:9092 \
  --partitions 1 \
  --replication-factor 1

# Consume messages
docker exec -it kafkaretry-kafka-1 kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic test-topic \
  --from-beginning
```

### Health Check

```bash
curl http://localhost:8080/health
```

## Testing External Webhooks

If you want to test with a real external webhook service:

### Option 1: webhook.site

1. Go to https://webhook.site
2. Copy your unique URL
3. Create a task with that URL as the destination

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "id": "external-webhook-test",
    "source": "test",
    "destination": {
      "url": "https://webhook.site/YOUR-UNIQUE-ID"
    },
    "dead_destination": {
      "url": "https://webhook.site/YOUR-UNIQUE-ID-dlq"
    },
    "max_retries": 3,
    "base_delay": 2,
    "client_id": "test",
    "is_priority": false,
    "message_data": "{\"test\": \"external\"}",
    "destination_type": "http"
  }'
```

4. View the webhook delivery on webhook.site

### Option 2: ngrok (for local webhook receiver)

1. Start the webhook receiver: `go run webhook-receiver.go`
2. In another terminal: `ngrok http 8090`
3. Copy the ngrok URL (e.g., `https://abc123.ngrok.io`)
4. Create a task with the ngrok URL

## Troubleshooting

### Webhook receiver not receiving requests

- Check if the webhook receiver is running: `curl http://localhost:8090/webhook`
- Verify the application can reach the webhook: check logs for connection errors
- Ensure the URL in the task matches the webhook receiver endpoint

### Kafka messages not appearing

- Verify Kafka is running: `docker ps | grep kafka`
- Check if topic exists: `docker exec -it kafkaretry-kafka-1 kafka-topics --list --bootstrap-server localhost:9092`
- Create topic if missing (see "Consume Kafka Messages" above)

### Tasks stuck in Redis

- Check worker is running: `docker-compose logs kafka-retry | grep "Worker started"`
- Verify Redis connection: `curl http://localhost:8080/health`
- Check task scores (timestamps): `docker exec -it kafkaretry-redis-1 redis-cli ZRANGE kafkaretry:tasks 0 -1 WITHSCORES`

## Cleanup

```bash
# Stop webhook receiver: Ctrl+C
# Stop application: Ctrl+C
# Stop infrastructure: docker-compose down
# Remove all data: docker-compose down -v
```
