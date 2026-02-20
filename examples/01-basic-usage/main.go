package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"

	"github.com/ruudy-sib/rebound/pkg/rebound"
)

// Example 1: Basic usage - Simplest way to use Rebound
func main() {
	fmt.Println("=== Rebound Example 1: Basic Usage ===\n")

	// Create a logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Configure Rebound
	cfg := &rebound.Config{
		RedisAddr:    "localhost:6379",
		RedisPassword: "",
		RedisDB:      0,
		KafkaBrokers: []string{"localhost:9092"},
		PollInterval: 1 * time.Second,
		Logger:       logger,
	}

	// Create Rebound instance
	rb, err := rebound.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create rebound: %v", err)
	}
	defer rb.Close()

	// Start the worker
	ctx := context.Background()
	if err := rb.Start(ctx); err != nil {
		log.Fatalf("Failed to start rebound: %v", err)
	}

	fmt.Println("✓ Rebound started successfully\n")

	// Example 1: Kafka task
	fmt.Println("Creating Kafka task...")
	kafkaTask := &rebound.Task{
		ID:     fmt.Sprintf("kafka-example-%d", time.Now().Unix()),
		Source: "basic-example",
		Destination: rebound.Destination{
			Host:  "localhost",
			Port:  "9092",
			Topic: "example-orders",
		},
		DeadDestination: rebound.Destination{
			Host:  "localhost",
			Port:  "9092",
			Topic: "example-orders-dlq",
		},
		MaxRetries:      3,
		BaseDelay:       5,
		ClientID:        "basic-example",
		IsPriority:      false,
		MessageData:     `{"order_id": 12345, "customer": "john@example.com", "amount": 99.99}`,
		DestinationType: rebound.DestinationTypeKafka,
	}

	if err := rb.CreateTask(ctx, kafkaTask); err != nil {
		log.Fatalf("Failed to create kafka task: %v", err)
	}
	fmt.Printf("✓ Kafka task created: %s\n\n", kafkaTask.ID)

	// Example 2: HTTP webhook task
	fmt.Println("Creating HTTP webhook task...")
	httpTask := &rebound.Task{
		ID:     fmt.Sprintf("webhook-example-%d", time.Now().Unix()),
		Source: "basic-example",
		Destination: rebound.Destination{
			URL: "http://localhost:8090/webhook",
		},
		DeadDestination: rebound.Destination{
			URL: "http://localhost:8090/dlq",
		},
		MaxRetries:      5,
		BaseDelay:       10,
		ClientID:        "basic-example",
		IsPriority:      false,
		MessageData:     `{"event": "order.created", "order_id": 12345, "timestamp": "2026-02-16T10:30:00Z"}`,
		DestinationType: rebound.DestinationTypeHTTP,
	}

	if err := rb.CreateTask(ctx, httpTask); err != nil {
		log.Fatalf("Failed to create http task: %v", err)
	}
	fmt.Printf("✓ HTTP task created: %s\n\n", httpTask.ID)

	fmt.Println("Tasks created successfully!")
	fmt.Println("The worker will process these tasks in the background.")
	fmt.Println("\nNote: Make sure Redis, Kafka, and the webhook receiver are running.")
	fmt.Println("      Run 'docker-compose up -d' and 'go run examples/webhook-receiver.go'")

	// Keep running for a bit to let tasks process
	time.Sleep(5 * time.Second)
	fmt.Println("\nExiting...")
}
