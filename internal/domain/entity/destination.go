package entity

// Destination represents a target endpoint where messages are delivered.
// For Kafka: use Host, Port, and Topic (or KafkaWriter to inject a pre-built writer).
// For HTTP: use URL.
type Destination struct {
	Host  string // Kafka broker host
	Port  string // Kafka broker port
	Topic string // Kafka topic name
	URL   string // HTTP endpoint URL (for HTTP destinations)

	// SASL authentication fields (optional).
	// SASLMechanism is one of "PLAIN", "SCRAM-SHA-256", or "SCRAM-SHA-512".
	SASLMechanism string
	SASLUsername  string
	SASLPassword  string

	// KafkaWriter can hold a pre-built *kafka.Writer (package-embedding mode only).
	// When set, Host/Port are ignored and this writer is used directly.
	KafkaWriter any
}

// Address returns the host:port combination for connection.
func (d Destination) Address() string {
	return d.Host + ":" + d.Port
}
