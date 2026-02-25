package kafkaproducer

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	"go.uber.org/zap"

	"github.com/ruudy-sib/rebound/internal/domain/entity"
	"github.com/ruudy-sib/rebound/internal/port/secondary"
)

// DestinationProducer implements secondary.MessageProducer by creating Kafka
// writers on-demand per broker address derived from the task destination.
// Writers are cached by a key derived from address + SASL identity and reused across calls.
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
// If destination.KafkaWriter is set, it is used directly and Host/Port are ignored.
// Otherwise Host and Port are required.
func (p *DestinationProducer) Produce(ctx context.Context, destination entity.Destination, key, value []byte) error {
	writer, err := p.resolveWriter(destination)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Topic: destination.Topic,
		Key:   key,
		Value: value,
	}

	if err := writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("writing message to kafka topic %q: %w", destination.Topic, err)
	}

	p.logger.Debug("message produced",
		zap.String("topic", destination.Topic),
		zap.Int("value_size", len(value)),
	)

	return nil
}

// Close shuts down all cached writers.
// Writers injected via KafkaWriter are not closed here — the caller owns them.
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

// resolveWriter returns the writer to use for the given destination.
// Injected writers are used as-is; all others are created and cached.
func (p *DestinationProducer) resolveWriter(dest entity.Destination) (*kafka.Writer, error) {
	if dest.KafkaWriter != nil {
		w, ok := dest.KafkaWriter.(*kafka.Writer)
		if !ok {
			return nil, fmt.Errorf("KafkaWriter must be a *kafka.Writer, got %T", dest.KafkaWriter)
		}
		return w, nil
	}

	if dest.Host == "" || dest.Port == "" {
		return nil, fmt.Errorf("kafka destination requires host and port (or a KafkaWriter)")
	}

	return p.writerFor(dest), nil
}

// writerFor returns a cached writer for the destination, creating one if needed.
// The cache key incorporates the broker address and SASL identity so that
// different credentials for the same broker get distinct writers.
func (p *DestinationProducer) writerFor(dest entity.Destination) *kafka.Writer {
	addr := dest.Host + ":" + dest.Port
	cacheKey := addr
	if dest.SASLMechanism != "" {
		cacheKey = addr + "|" + dest.SASLMechanism + "|" + dest.SASLUsername
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if w, ok := p.writers[cacheKey]; ok {
		return w
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(addr),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 100 * time.Millisecond,
		RequiredAcks: kafka.RequireAll,
	}

	if dest.SASLMechanism != "" {
		dialer, err := dialerWithSASL(dest)
		if err != nil {
			p.logger.Warn("unsupported SASL mechanism; proceeding without SASL",
				zap.String("mechanism", dest.SASLMechanism),
				zap.Error(err),
			)
		} else {
			w.Transport = &kafka.Transport{
				SASL: dialer,
				TLS:  &tls.Config{MinVersion: tls.VersionTLS12},
			}
		}
	}

	p.writers[cacheKey] = w
	p.logger.Info("kafka writer created",
		zap.String("broker", addr),
		zap.String("sasl_mechanism", dest.SASLMechanism),
	)

	return w
}

// dialerWithSASL returns the sasl.Mechanism for the given destination.
func dialerWithSASL(dest entity.Destination) (sasl.Mechanism, error) {
	switch dest.SASLMechanism {
	case "PLAIN":
		return plain.Mechanism{
			Username: dest.SASLUsername,
			Password: dest.SASLPassword,
		}, nil
	case "SCRAM-SHA-256":
		return scram.Mechanism(scram.SHA256, dest.SASLUsername, dest.SASLPassword)
	case "SCRAM-SHA-512":
		return scram.Mechanism(scram.SHA512, dest.SASLUsername, dest.SASLPassword)
	default:
		return nil, fmt.Errorf("unsupported SASL mechanism %q (supported: PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)", dest.SASLMechanism)
	}
}
