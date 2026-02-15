package kafkaproducer

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"rebound/internal/config"
	"rebound/internal/domain/entity"
	"rebound/internal/port/secondary"
)

// Producer implements secondary.MessageProducer using segmentio/kafka-go.
// It maintains a single writer connection for all message deliveries.
type Producer struct {
	writer *kafka.Writer
	logger *zap.Logger
}

// NewProducer creates a Kafka producer from the application configuration.
func NewProducer(cfg *config.Config, logger *zap.Logger) secondary.MessageProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBrokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 100 * time.Millisecond,
		RequiredAcks: kafka.RequireAll,
	}

	logger.Info("kafka producer initialized",
		zap.Strings("brokers", cfg.KafkaBrokers),
	)

	return &Producer{
		writer: writer,
		logger: logger.Named("kafka-producer"),
	}
}

// Produce sends a message to the specified Kafka topic.
func (p *Producer) Produce(ctx context.Context, destination entity.Destination, key, value []byte) error {
	msg := kafka.Message{
		Topic: destination.Topic,
		Key:   key,
		Value: value,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("writing message to kafka topic %q: %w", destination.Topic, err)
	}

	p.logger.Debug("message produced",
		zap.String("topic", destination.Topic),
		zap.Int("value_size", len(value)),
	)

	return nil
}

// Close shuts down the Kafka writer and releases its resources.
func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
