package http

import "github.com/ruudy-sib/rebound/internal/domain/entity"

// CreateTaskRequest matches the OpenAPI Task schema.
type CreateTaskRequest struct {
	ID              string         `json:"id"`
	Source          string         `json:"source"`
	Destination     DestinationDTO `json:"destination"`
	DeadDestination DestinationDTO `json:"dead_destination"`
	MaxRetries      int            `json:"max_retries"`
	BaseDelay       int            `json:"base_delay"`
	ClientID        string         `json:"client_id"`
	IsPriority      bool           `json:"is_priority"`
	MessageData     string         `json:"message_data"`
	DestinationType string         `json:"destination_type"`
}

// DestinationDTO matches the OpenAPI Destination schema.
type DestinationDTO struct {
	Host  string `json:"host"`
	Port  string `json:"port"`
	Topic string `json:"topic"`
	URL   string `json:"url"`

	// SASL authentication (Kafka only). Mechanism is one of "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512".
	SASLMechanism string `json:"sasl_mechanism,omitempty"`
	SASLUsername  string `json:"sasl_username,omitempty"`
	SASLPassword  string `json:"sasl_password,omitempty"`
}

// CreateTaskResponse is returned on successful task creation.
type CreateTaskResponse struct {
	Message string `json:"message"`
}

// ErrorResponse is the standard error payload.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// HealthResponse is returned by the health check endpoint.
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// toEntity converts a CreateTaskRequest DTO to a domain entity.
func (r *CreateTaskRequest) toEntity() *entity.Task {
	return &entity.Task{
		ID:     r.ID,
		Source: r.Source,
		Destination: entity.Destination{
			Host:          r.Destination.Host,
			Port:          r.Destination.Port,
			Topic:         r.Destination.Topic,
			URL:           r.Destination.URL,
			SASLMechanism: r.Destination.SASLMechanism,
			SASLUsername:  r.Destination.SASLUsername,
			SASLPassword:  r.Destination.SASLPassword,
		},
		DeadDestination: entity.Destination{
			Host:          r.DeadDestination.Host,
			Port:          r.DeadDestination.Port,
			Topic:         r.DeadDestination.Topic,
			URL:           r.DeadDestination.URL,
			SASLMechanism: r.DeadDestination.SASLMechanism,
			SASLUsername:  r.DeadDestination.SASLUsername,
			SASLPassword:  r.DeadDestination.SASLPassword,
		},
		MaxRetries:      r.MaxRetries,
		BaseDelay:       r.BaseDelay,
		ClientID:        r.ClientID,
		IsPriority:      r.IsPriority,
		MessageData:     r.MessageData,
		DestinationType: entity.DestinationType(r.DestinationType),
	}
}
