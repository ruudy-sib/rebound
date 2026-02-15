package entity

// Destination represents a target endpoint where messages are delivered.
type Destination struct {
	Host  string
	Port  string
	Topic string
}

// Address returns the host:port combination for connection.
func (d Destination) Address() string {
	return d.Host + ":" + d.Port
}
