package producerfactory

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"rebound/internal/domain/entity"
	"rebound/internal/port/secondary"
)

// Factory routes message production to the appropriate producer based on destination type.
type Factory struct {
	kafkaProducer secondary.MessageProducer
	httpProducer  secondary.MessageProducer
	logger        *zap.Logger
}

// NewFactory creates a producer factory with both Kafka and HTTP producers.
func NewFactory(
	kafkaProducer secondary.MessageProducer,
	httpProducer secondary.MessageProducer,
	logger *zap.Logger,
) secondary.MessageProducer {
	return &Factory{
		kafkaProducer: kafkaProducer,
		httpProducer:  httpProducer,
		logger:        logger.Named("producer-factory"),
	}
}

// Produce routes the message to the appropriate producer based on destination type.
func (f *Factory) Produce(ctx context.Context, destination entity.Destination, key, value []byte) error {
	// Route based on which fields are set
	if destination.URL != "" {
		f.logger.Debug("routing to http producer", zap.String("url", destination.URL))
		return f.httpProducer.Produce(ctx, destination, key, value)
	}

	if destination.Topic != "" {
		f.logger.Debug("routing to kafka producer", zap.String("topic", destination.Topic))
		return f.kafkaProducer.Produce(ctx, destination, key, value)
	}

	return fmt.Errorf("unable to determine destination type: neither URL nor Topic is set")
}

// Close closes all underlying producers.
func (f *Factory) Close() error {
	var errs []error

	if err := f.kafkaProducer.Close(); err != nil {
		errs = append(errs, fmt.Errorf("closing kafka producer: %w", err))
	}

	if err := f.httpProducer.Close(); err != nil {
		errs = append(errs, fmt.Errorf("closing http producer: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("closing producers: %v", errs)
	}

	return nil
}
