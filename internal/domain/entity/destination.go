package entity

// Destination represents a target endpoint where messages are delivered.
// For Kafka: use Host, Port, and Topic.
// For HTTP: use URL (e.g., "http://api.example.com/webhook").
type Destination struct {
	Host  string // Kafka broker host
	Port  string // Kafka broker port
	Topic string // Kafka topic name
	URL   string // HTTP endpoint URL (for HTTP destinations)
}

// Address returns the host:port combination for Kafka connections.
func (d Destination) Address() string {
	return d.Host + ":" + d.Port
}
