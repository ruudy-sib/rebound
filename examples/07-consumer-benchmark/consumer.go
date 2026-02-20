package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/ruudy-sib/rebound/pkg/rebound"
)

// Consumer represents a Kafka consumer that uses Rebound for retry logic.
type Consumer struct {
	reader        *kafka.Reader
	rebound       *rebound.Rebound
	logger        *zap.Logger
	messageCount  int
	errorCount    int
	retryCount    int
	successCount  int
	processingFunc MessageProcessor
}

// MessageProcessor is a function that processes a Kafka message.
// It returns an error if processing fails.
type MessageProcessor func(ctx context.Context, message kafka.Message) error

// ConsumerConfig holds configuration for the consumer.
type ConsumerConfig struct {
	KafkaBrokers    []string
	GroupID         string
	Topic           string
	RetryTopic      string
	DLQTopic        string
	RedisAddr       string
	MaxRetries      int
	BaseDelay       int
	ProcessingFunc  MessageProcessor
	Logger          *zap.Logger
}

// NewConsumer creates a new Kafka consumer with Rebound integration.
func NewConsumer(cfg *ConsumerConfig) (*Consumer, error) {
	// Create Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.KafkaBrokers,
		GroupID:        cfg.GroupID,
		Topic:          cfg.Topic,
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	// Create Rebound instance for retry logic
	reboundCfg := &rebound.Config{
		RedisAddr:    cfg.RedisAddr,
		PollInterval: 1 * time.Second,
		Logger:       cfg.Logger,
	}

	rb, err := rebound.New(reboundCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create rebound: %w", err)
	}

	return &Consumer{
		reader:         reader,
		rebound:        rb,
		logger:         cfg.Logger,
		processingFunc: cfg.ProcessingFunc,
	}, nil
}

// Start begins consuming messages from Kafka.
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("starting consumer")

	// Start rebound worker
	if err := c.rebound.Start(ctx); err != nil {
		return fmt.Errorf("failed to start rebound: %w", err)
	}

	// Start consuming messages
	go c.consumeLoop(ctx)

	return nil
}

// consumeLoop is the main message consumption loop.
func (c *Consumer) consumeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer loop stopped")
			return
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				c.logger.Error("failed to read message", zap.Error(err))
				c.errorCount++
				continue
			}

			c.messageCount++
			c.processMessage(ctx, msg)
		}
	}
}

// processMessage processes a single Kafka message with retry logic.
func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) {
	c.logger.Debug("processing message",
		zap.String("topic", msg.Topic),
		zap.Int("partition", msg.Partition),
		zap.Int64("offset", msg.Offset),
	)

	// Try to process the message
	err := c.processingFunc(ctx, msg)
	if err != nil {
		c.logger.Warn("message processing failed, scheduling retry",
			zap.Error(err),
			zap.String("message_key", string(msg.Key)),
		)

		// Schedule retry using Rebound
		if retryErr := c.scheduleRetry(ctx, msg); retryErr != nil {
			c.logger.Error("failed to schedule retry",
				zap.Error(retryErr),
				zap.String("message_key", string(msg.Key)),
			)
			c.errorCount++
			return
		}

		c.retryCount++
		return
	}

	c.successCount++
	c.logger.Debug("message processed successfully",
		zap.String("message_key", string(msg.Key)),
	)
}

// scheduleRetry creates a Rebound task for the failed message.
func (c *Consumer) scheduleRetry(ctx context.Context, msg kafka.Message) error {
	// Extract metadata from headers
	var maxRetries = 3
	var baseDelay = 5

	// Create retry task
	task := &rebound.Task{
		ID:     fmt.Sprintf("%s-%d-%d-%d", msg.Topic, msg.Partition, msg.Offset, time.Now().UnixNano()),
		Source: "kafka-consumer",
		Destination: rebound.Destination{
			Host:  "localhost",
			Port:  "9092",
			Topic: msg.Topic, // Retry to the same topic
		},
		DeadDestination: rebound.Destination{
			Host:  "localhost",
			Port:  "9092",
			Topic: msg.Topic + "-dlq",
		},
		MaxRetries:      maxRetries,
		BaseDelay:       baseDelay,
		ClientID:        "kafka-consumer",
		MessageData:     string(msg.Value),
		DestinationType: rebound.DestinationTypeKafka,
	}

	return c.rebound.CreateTask(ctx, task)
}

// Close gracefully shuts down the consumer.
func (c *Consumer) Close() error {
	c.logger.Info("closing consumer",
		zap.Int("total_messages", c.messageCount),
		zap.Int("successful", c.successCount),
		zap.Int("retries", c.retryCount),
		zap.Int("errors", c.errorCount),
	)

	var errs []error

	if err := c.reader.Close(); err != nil {
		errs = append(errs, fmt.Errorf("closing kafka reader: %w", err))
	}

	if err := c.rebound.Close(); err != nil {
		errs = append(errs, fmt.Errorf("closing rebound: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}

	return nil
}

// Stats returns consumer statistics.
func (c *Consumer) Stats() ConsumerStats {
	return ConsumerStats{
		TotalMessages:      c.messageCount,
		SuccessfulMessages: c.successCount,
		RetriedMessages:    c.retryCount,
		ErrorMessages:      c.errorCount,
	}
}

// ConsumerStats holds consumer statistics.
type ConsumerStats struct {
	TotalMessages      int
	SuccessfulMessages int
	RetriedMessages    int
	ErrorMessages      int
}

// String returns a formatted string representation of the stats.
func (s ConsumerStats) String() string {
	data, _ := json.MarshalIndent(s, "", "  ")
	return string(data)
}
