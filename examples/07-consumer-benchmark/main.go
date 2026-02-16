package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

var (
	kafkaBrokers = flag.String("kafka", "localhost:9092", "Kafka brokers (comma-separated)")
	redisAddr    = flag.String("redis", "localhost:6379", "Redis address")
	topic        = flag.String("topic", "consumer-benchmark", "Kafka topic to consume from")
	groupID      = flag.String("group", "benchmark-consumer-group", "Kafka consumer group ID")
	failureRate  = flag.Float64("failure-rate", 0.3, "Simulated failure rate (0.0-1.0)")
	duration     = flag.Duration("duration", 60*time.Second, "How long to run the consumer")
)

func main() {
	flag.Parse()

	fmt.Println("=== Rebound Consumer Benchmark ===\n")

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create consumer with simulated failure processor
	consumer, err := NewConsumer(&ConsumerConfig{
		KafkaBrokers:   []string{*kafkaBrokers},
		GroupID:        *groupID,
		Topic:          *topic,
		RedisAddr:      *redisAddr,
		MaxRetries:     3,
		BaseDelay:      5,
		ProcessingFunc: simulatedProcessor(*failureRate),
		Logger:         logger,
	})
	if err != nil {
		logger.Fatal("failed to create consumer", zap.Error(err))
	}
	defer consumer.Close()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer
	if err := consumer.Start(ctx); err != nil {
		logger.Fatal("failed to start consumer", zap.Error(err))
	}

	fmt.Printf("✓ Consumer started\n")
	fmt.Printf("  Topic: %s\n", *topic)
	fmt.Printf("  Group ID: %s\n", *groupID)
	fmt.Printf("  Failure Rate: %.1f%%\n", *failureRate*100)
	fmt.Printf("  Duration: %s\n\n", *duration)

	// Start producing test messages
	go produceTestMessages(ctx, *topic, *kafkaBrokers, logger)

	// Run for specified duration or until interrupted
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-time.After(*duration):
		fmt.Println("\n⏱️  Duration elapsed, shutting down...")
	case sig := <-sigChan:
		fmt.Printf("\n📡 Received signal %v, shutting down...\n", sig)
	}

	// Graceful shutdown
	cancel()
	time.Sleep(2 * time.Second) // Allow in-flight messages to complete

	// Print statistics
	stats := consumer.Stats()
	fmt.Println("\n=== Final Statistics ===")
	fmt.Println(stats.String())

	successRate := float64(stats.SuccessfulMessages) / float64(stats.TotalMessages) * 100
	retryRate := float64(stats.RetriedMessages) / float64(stats.TotalMessages) * 100

	fmt.Printf("\nSuccess Rate: %.2f%%\n", successRate)
	fmt.Printf("Retry Rate: %.2f%%\n", retryRate)
	fmt.Println("\n✓ Consumer benchmark completed")
}

// simulatedProcessor creates a message processor that fails randomly based on the failure rate.
func simulatedProcessor(failureRate float64) MessageProcessor {
	return func(ctx context.Context, message kafka.Message) error {
		// Simulate some processing time
		time.Sleep(time.Duration(10+rand.Intn(40)) * time.Millisecond)

		// Randomly fail based on failure rate
		if rand.Float64() < failureRate {
			return errors.New("simulated processing failure")
		}

		// Simulate successful processing
		var data map[string]interface{}
		if err := json.Unmarshal(message.Value, &data); err != nil {
			return fmt.Errorf("invalid message format: %w", err)
		}

		return nil
	}
}

// produceTestMessages produces test messages to Kafka for the consumer to process.
func produceTestMessages(ctx context.Context, topic, broker string, logger *zap.Logger) {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        true,
	}
	defer writer.Close()

	logger.Info("starting test message producer")

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	messageID := 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping test message producer")
			return
		case <-ticker.C:
			messageID++
			message := map[string]interface{}{
				"message_id": messageID,
				"timestamp":  time.Now().Format(time.RFC3339),
				"data":       fmt.Sprintf("test-data-%d", messageID),
				"priority":   rand.Intn(5),
			}

			data, _ := json.Marshal(message)

			err := writer.WriteMessages(ctx, kafka.Message{
				Key:   []byte(fmt.Sprintf("key-%d", messageID)),
				Value: data,
			})

			if err != nil {
				logger.Error("failed to write message", zap.Error(err))
			}
		}
	}
}
