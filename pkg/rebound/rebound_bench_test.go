package rebound_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"

	"rebound/pkg/rebound"
)

// BenchmarkTaskCreation benchmarks the task creation throughput.
func BenchmarkTaskCreation(b *testing.B) {
	logger, _ := zap.NewProduction()
	cfg := &rebound.Config{
		RedisAddr:    "localhost:6379",
		KafkaBrokers: []string{"localhost:9092"},
		PollInterval: 1 * time.Second,
		Logger:       logger,
	}

	rb, err := rebound.New(cfg)
	if err != nil {
		b.Fatalf("Failed to create rebound: %v", err)
	}
	defer rb.Close()

	ctx := context.Background()
	if err := rb.Start(ctx); err != nil {
		b.Fatalf("Failed to start rebound: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		task := &rebound.Task{
			ID:     fmt.Sprintf("bench-task-%d", i),
			Source: "benchmark",
			Destination: rebound.Destination{
				Host:  "localhost",
				Port:  "9092",
				Topic: "bench-topic",
			},
			DeadDestination: rebound.Destination{
				Host:  "localhost",
				Port:  "9092",
				Topic: "bench-topic-dlq",
			},
			MaxRetries:      3,
			BaseDelay:       5,
			ClientID:        "benchmark",
			MessageData:     `{"benchmark": true}`,
			DestinationType: rebound.DestinationTypeKafka,
		}

		if err := rb.CreateTask(ctx, task); err != nil {
			b.Fatalf("Failed to create task: %v", err)
		}
	}
}

// BenchmarkTaskCreationHTTP benchmarks HTTP task creation.
func BenchmarkTaskCreationHTTP(b *testing.B) {
	logger, _ := zap.NewProduction()
	cfg := &rebound.Config{
		RedisAddr:    "localhost:6379",
		KafkaBrokers: []string{"localhost:9092"},
		PollInterval: 1 * time.Second,
		Logger:       logger,
	}

	rb, err := rebound.New(cfg)
	if err != nil {
		b.Fatalf("Failed to create rebound: %v", err)
	}
	defer rb.Close()

	ctx := context.Background()
	if err := rb.Start(ctx); err != nil {
		b.Fatalf("Failed to start rebound: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		task := &rebound.Task{
			ID:     fmt.Sprintf("bench-http-%d", i),
			Source: "benchmark",
			Destination: rebound.Destination{
				URL: "http://localhost:8090/webhook",
			},
			DeadDestination: rebound.Destination{
				URL: "http://localhost:8090/dlq",
			},
			MaxRetries:      3,
			BaseDelay:       5,
			ClientID:        "benchmark",
			MessageData:     `{"benchmark": true, "type": "http"}`,
			DestinationType: rebound.DestinationTypeHTTP,
		}

		if err := rb.CreateTask(ctx, task); err != nil {
			b.Fatalf("Failed to create task: %v", err)
		}
	}
}

// BenchmarkTaskCreationParallel benchmarks parallel task creation.
func BenchmarkTaskCreationParallel(b *testing.B) {
	logger, _ := zap.NewProduction()
	cfg := &rebound.Config{
		RedisAddr:    "localhost:6379",
		KafkaBrokers: []string{"localhost:9092"},
		PollInterval: 1 * time.Second,
		Logger:       logger,
	}

	rb, err := rebound.New(cfg)
	if err != nil {
		b.Fatalf("Failed to create rebound: %v", err)
	}
	defer rb.Close()

	ctx := context.Background()
	if err := rb.Start(ctx); err != nil {
		b.Fatalf("Failed to start rebound: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			task := &rebound.Task{
				ID:     fmt.Sprintf("bench-parallel-%d-%d", b.N, i),
				Source: "benchmark-parallel",
				Destination: rebound.Destination{
					Host:  "localhost",
					Port:  "9092",
					Topic: "bench-parallel",
				},
				DeadDestination: rebound.Destination{
					Host:  "localhost",
					Port:  "9092",
					Topic: "bench-parallel-dlq",
				},
				MaxRetries:      3,
				BaseDelay:       5,
				ClientID:        "benchmark",
				MessageData:     `{"parallel": true}`,
				DestinationType: rebound.DestinationTypeKafka,
			}

			if err := rb.CreateTask(ctx, task); err != nil {
				b.Fatalf("Failed to create task: %v", err)
			}
			i++
		}
	})
}

// BenchmarkTaskCreationWithVariablePayload benchmarks task creation with different payload sizes.
func BenchmarkTaskCreationWithVariablePayload(b *testing.B) {
	payloadSizes := []int{100, 1024, 10240, 102400} // 100B, 1KB, 10KB, 100KB

	for _, size := range payloadSizes {
		b.Run(fmt.Sprintf("PayloadSize-%dB", size), func(b *testing.B) {
			logger, _ := zap.NewProduction()
			cfg := &rebound.Config{
				RedisAddr:    "localhost:6379",
				KafkaBrokers: []string{"localhost:9092"},
				PollInterval: 1 * time.Second,
				Logger:       logger,
			}

			rb, err := rebound.New(cfg)
			if err != nil {
				b.Fatalf("Failed to create rebound: %v", err)
			}
			defer rb.Close()

			ctx := context.Background()
			if err := rb.Start(ctx); err != nil {
				b.Fatalf("Failed to start rebound: %v", err)
			}

			// Create a payload of the specified size
			payload := make([]byte, size)
			for i := range payload {
				payload[i] = byte('a' + (i % 26))
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				task := &rebound.Task{
					ID:     fmt.Sprintf("bench-payload-%d-%d", size, i),
					Source: "benchmark",
					Destination: rebound.Destination{
						Host:  "localhost",
						Port:  "9092",
						Topic: "bench-payload",
					},
					DeadDestination: rebound.Destination{
						Host:  "localhost",
						Port:  "9092",
						Topic: "bench-payload-dlq",
					},
					MaxRetries:      3,
					BaseDelay:       5,
					ClientID:        "benchmark",
					MessageData:     string(payload),
					DestinationType: rebound.DestinationTypeKafka,
				}

				if err := rb.CreateTask(ctx, task); err != nil {
					b.Fatalf("Failed to create task: %v", err)
				}
			}
		})
	}
}

// BenchmarkTaskCreationWithRetries benchmarks task creation with different retry configurations.
func BenchmarkTaskCreationWithRetries(b *testing.B) {
	retryConfigs := []struct {
		maxRetries int
		baseDelay  int
	}{
		{1, 5},
		{3, 5},
		{5, 10},
		{10, 30},
	}

	for _, rc := range retryConfigs {
		b.Run(fmt.Sprintf("Retries-%d-Delay-%ds", rc.maxRetries, rc.baseDelay), func(b *testing.B) {
			logger, _ := zap.NewProduction()
			cfg := &rebound.Config{
				RedisAddr:    "localhost:6379",
				KafkaBrokers: []string{"localhost:9092"},
				PollInterval: 1 * time.Second,
				Logger:       logger,
			}

			rb, err := rebound.New(cfg)
			if err != nil {
				b.Fatalf("Failed to create rebound: %v", err)
			}
			defer rb.Close()

			ctx := context.Background()
			if err := rb.Start(ctx); err != nil {
				b.Fatalf("Failed to start rebound: %v", err)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				task := &rebound.Task{
					ID:     fmt.Sprintf("bench-retry-%d-%d-%d", rc.maxRetries, rc.baseDelay, i),
					Source: "benchmark",
					Destination: rebound.Destination{
						Host:  "localhost",
						Port:  "9092",
						Topic: "bench-retry",
					},
					DeadDestination: rebound.Destination{
						Host:  "localhost",
						Port:  "9092",
						Topic: "bench-retry-dlq",
					},
					MaxRetries:      rc.maxRetries,
					BaseDelay:       rc.baseDelay,
					ClientID:        "benchmark",
					MessageData:     `{"retry_test": true}`,
					DestinationType: rebound.DestinationTypeKafka,
				}

				if err := rb.CreateTask(ctx, task); err != nil {
					b.Fatalf("Failed to create task: %v", err)
				}
			}
		})
	}
}

// BenchmarkReboundInitialization benchmarks the initialization time of Rebound.
func BenchmarkReboundInitialization(b *testing.B) {
	logger, _ := zap.NewProduction()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cfg := &rebound.Config{
			RedisAddr:    "localhost:6379",
			KafkaBrokers: []string{"localhost:9092"},
			PollInterval: 1 * time.Second,
			Logger:       logger,
		}

		rb, err := rebound.New(cfg)
		if err != nil {
			b.Fatalf("Failed to create rebound: %v", err)
		}
		rb.Close()
	}
}
