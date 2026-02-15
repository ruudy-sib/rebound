#!/bin/bash
# Example script demonstrating task creation with Kafka and HTTP destinations

set -e

API_URL="${API_URL:-http://localhost:8080}"

echo "🚀 Creating tasks with different destination types..."
echo

# Example 1: Kafka destination
echo "📤 Creating Kafka task..."
curl -X POST "$API_URL/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "kafka-task-1",
    "source": "example-service",
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
    "client_id": "example-client",
    "is_priority": false,
    "message_data": "{\"order_id\": 123, \"action\": \"process\"}",
    "destination_type": "kafka"
  }'
echo
echo

# Example 2: HTTP destination
echo "🌐 Creating HTTP task..."
curl -X POST "$API_URL/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "http-task-1",
    "source": "example-service",
    "destination": {
      "url": "https://webhook.site/unique-id"
    },
    "dead_destination": {
      "url": "https://webhook.site/unique-id-dlq"
    },
    "max_retries": 3,
    "base_delay": 5,
    "client_id": "example-client",
    "is_priority": false,
    "message_data": "{\"event\": \"user.created\", \"user_id\": 456}",
    "destination_type": "http"
  }'
echo
echo

# Example 3: Priority task (Kafka)
echo "⚡ Creating priority Kafka task..."
curl -X POST "$API_URL/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "priority-task-1",
    "source": "example-service",
    "destination": {
      "host": "localhost",
      "port": "9092",
      "topic": "urgent-orders"
    },
    "dead_destination": {
      "host": "localhost",
      "port": "9092",
      "topic": "urgent-orders-dlq"
    },
    "max_retries": 5,
    "base_delay": 2,
    "client_id": "example-client",
    "is_priority": true,
    "message_data": "{\"order_id\": 789, \"action\": \"urgent_process\"}",
    "destination_type": "kafka"
  }'
echo
echo

echo "✅ All tasks created successfully!"
echo
echo "💡 Tips:"
echo "  - Check task processing: docker-compose logs -f kafka-retry"
echo "  - Monitor Kafka messages: docker exec -it kafkaretry-kafka-1 kafka-console-consumer --bootstrap-server localhost:9092 --topic orders --from-beginning"
echo "  - Check Redis queue: docker exec -it kafkaretry-redis-1 redis-cli ZRANGE kafkaretry:tasks 0 -1 WITHSCORES"
echo "  - Health check: curl http://localhost:8080/health"
