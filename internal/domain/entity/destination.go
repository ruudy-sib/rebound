package entity

// Destination represents a target endpoint where messages are delivered.
// For Kafka: use Host, Port, and Topic.
// For HTTP: use URL.
type Destination struct {
	Host  string // Kafka broker host
	Port  string // Kafka broker port
	Topic string // Kafka topic name
	URL   string // HTTP endpoint URL (for HTTP destinations)
}

// Address returns the host:port combination for connection.
func (d Destination) Address() string {
	return d.Host + ":" + d.Port
}
