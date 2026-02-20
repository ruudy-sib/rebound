package kafkaproducer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/ruudy-sib/rebound/internal/domain/entity"
	"github.com/ruudy-sib/rebound/internal/port/secondary"
)

// DestinationProducer implements secondary.MessageProducer by creating Kafka
// writers on-demand per broker address derived from the task destination.
// Writers are cached by "host:port" and reused across calls.
// This is used when no global broker list is configured (package embedding mode).
type DestinationProducer struct {
	writers map[string]*kafka.Writer
	mu      sync.Mutex
	logger  *zap.Logger
}

// NewDestinationProducer creates a Kafka producer that connects per destination.
func NewDestinationProducer(logger *zap.Logger) secondary.MessageProducer {
	return &DestinationProducer{
		writers: make(map[string]*kafka.Writer),
		logger:  logger.Named("kafka-destination-producer"),
	}
}

// Produce sends a message to the broker and topic specified in destination.
func (p *DestinationProducer) Produce(ctx context.Context, destination entity.Destination, key, value []byte) error {
	if destination.Host == "" || destination.Port == "" {
		return fmt.Errorf("kafka destination requires host and port")
	}

	addr := destination.Host + ":" + destination.Port
	writer := p.writerFor(addr)

	msg := kafka.Message{
		Topic: destination.Topic,
		Key:   key,
		Value: value,
	}

	if err := writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("writing message to kafka topic %q at %q: %w", destination.Topic, addr, err)
	}

	p.logger.Debug("message produced",
		zap.String("broker", addr),
		zap.String("topic", destination.Topic),
		zap.Int("value_size", len(value)),
	)

	return nil
}

// Close shuts down all cached writers.
func (p *DestinationProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for addr, w := range p.writers {
		if err := w.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing writer for %s: %w", addr, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing kafka writers: %v", errs)
	}

	return nil
}

func (p *DestinationProducer) writerFor(addr string) *kafka.Writer {
	p.mu.Lock()
	defer p.mu.Unlock()

	if w, ok := p.writers[addr]; ok {
		return w
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(addr),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 100 * time.Millisecond,
		RequiredAcks: kafka.RequireAll,
	}
	p.writers[addr] = w

	p.logger.Info("kafka writer created", zap.String("broker", addr))

	return w
}
