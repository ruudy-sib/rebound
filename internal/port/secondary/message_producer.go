package secondary

import (
	"context"

	"github.com/ruudy-sib/rebound/internal/domain/entity"
)

// MessageProducer defines the secondary port for delivering messages
// to an external destination (e.g., Kafka, SQS).
type MessageProducer interface {
	// Produce sends a message to the specified destination.
	Produce(ctx context.Context, destination entity.Destination, key, value []byte) error

	// Close releases any resources held by the producer.
	Close() error
}
