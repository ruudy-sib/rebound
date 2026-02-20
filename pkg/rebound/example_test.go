package rebound_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/dig"
	"go.uber.org/zap"

	"github.com/ruudy-sib/rebound/pkg/rebound"
)

// Example_basic demonstrates basic usage of Rebound as a library.
func Example_basic() {
	// Create configuration
	cfg := &rebound.Config{
		RedisAddr:    "localhost:6379",
		KafkaBrokers: []string{"localhost:9092"},
		PollInterval: 1 * time.Second,
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

	// Create a task for Kafka
	kafkaTask := &rebound.Task{
		ID:     "order-123",
		Source: "order-service",
		Destination: rebound.Destination{
			Host:  "localhost",
			Port:  "9092",
			Topic: "orders",
		},
		DeadDestination: rebound.Destination{
			Host:  "localhost",
			Port:  "9092",
			Topic: "orders-dlq",
		},
		MaxRetries:      3,
		BaseDelay:       5,
		ClientID:        "order-service",
		MessageData:     `{"order_id": 123, "action": "process"}`,
		DestinationType: rebound.DestinationTypeKafka,
	}

	if err := rb.CreateTask(ctx, kafkaTask); err != nil {
		log.Fatalf("Failed to create kafka task: %v", err)
	}

	// Create a task for HTTP webhook
	httpTask := &rebound.Task{
		ID:     "webhook-456",
		Source: "notification-service",
		Destination: rebound.Destination{
			URL: "https://api.partner.com/webhooks/events",
		},
		DeadDestination: rebound.Destination{
			URL: "https://api.partner.com/webhooks/dlq",
		},
		MaxRetries:      5,
		BaseDelay:       10,
		ClientID:        "notification-service",
		MessageData:     `{"event": "user.created", "user_id": 456}`,
		DestinationType: rebound.DestinationTypeHTTP,
	}

	if err := rb.CreateTask(ctx, httpTask); err != nil {
		log.Fatalf("Failed to create http task: %v", err)
	}

	fmt.Println("Tasks created successfully")
}

// Example_dependencyInjection demonstrates using Rebound with uber-go/dig.
func Example_dependencyInjection() {
	// Create a DI container
	container := dig.New()

	// Provide logger
	container.Provide(func() *zap.Logger {
		logger, _ := zap.NewProduction()
		return logger
	})

	// Provide Rebound config
	container.Provide(func() *rebound.Config {
		return &rebound.Config{
			RedisAddr:    "localhost:6379",
			KafkaBrokers: []string{"localhost:9092"},
			PollInterval: 1 * time.Second,
		}
	})

	// Register Rebound
	if err := rebound.RegisterWithContainer(container); err != nil {
		log.Fatalf("Failed to register rebound: %v", err)
	}

	// Start Rebound
	container.Invoke(func(rb *rebound.Rebound) {
		ctx := context.Background()
		if err := rb.Start(ctx); err != nil {
			log.Fatalf("Failed to start rebound: %v", err)
		}

		// Use rebound in your application
		task := &rebound.Task{
			ID:     "task-1",
			Source: "my-service",
			Destination: rebound.Destination{
				URL: "https://api.example.com/webhook",
			},
			MaxRetries:      3,
			BaseDelay:       5,
			ClientID:        "my-service",
			MessageData:     `{"hello": "world"}`,
			DestinationType: rebound.DestinationTypeHTTP,
		}

		if err := rb.CreateTask(ctx, task); err != nil {
			log.Fatalf("Failed to create task: %v", err)
		}

		fmt.Println("Task created via DI")
	})
}

// Example_gracefulShutdown demonstrates proper shutdown handling.
func Example_gracefulShutdown() {
	cfg := rebound.DefaultConfig()
	rb, err := rebound.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create rebound: %v", err)
	}

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the worker
	if err := rb.Start(ctx); err != nil {
		log.Fatalf("Failed to start rebound: %v", err)
	}

	// ... do work ...

	// Graceful shutdown
	cancel()                     // Stop the worker
	time.Sleep(100 * time.Millisecond) // Wait for in-flight tasks
	rb.Close()                   // Release resources

	fmt.Println("Shutdown complete")
}
