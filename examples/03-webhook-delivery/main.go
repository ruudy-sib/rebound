package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"

	"rebound/pkg/rebound"
)

// Example 3: Webhook Delivery Service
// Shows how to use Rebound to reliably deliver webhooks to customer endpoints

type WebhookService struct {
	rebound *rebound.Rebound
	logger  *zap.Logger
}

type WebhookEvent struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

type CustomerWebhook struct {
	CustomerID string
	URL        string
	Secret     string
}

func NewWebhookService(rb *rebound.Rebound, logger *zap.Logger) *WebhookService {
	return &WebhookService{
		rebound: rb,
		logger:  logger.Named("webhook-service"),
	}
}

// DeliverWebhook attempts to deliver a webhook to a customer endpoint
func (s *WebhookService) DeliverWebhook(ctx context.Context, webhook *CustomerWebhook, event *WebhookEvent) error {
	s.logger.Info("delivering webhook",
		zap.String("customer_id", webhook.CustomerID),
		zap.String("event_type", event.EventType),
		zap.String("event_id", event.EventID),
	)

	// Serialize event
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	// Try immediate delivery
	if err := s.sendWebhook(ctx, webhook.URL, webhook.Secret, eventJSON); err != nil {
		s.logger.Warn("immediate delivery failed, scheduling retry",
			zap.String("customer_id", webhook.CustomerID),
			zap.Error(err),
		)
		return s.scheduleRetry(ctx, webhook, event, eventJSON)
	}

	s.logger.Info("webhook delivered successfully",
		zap.String("customer_id", webhook.CustomerID),
		zap.String("event_id", event.EventID),
	)
	return nil
}

func (s *WebhookService) sendWebhook(ctx context.Context, url, secret string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", secret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *WebhookService) scheduleRetry(ctx context.Context, webhook *CustomerWebhook, event *WebhookEvent, eventJSON []byte) error {
	// Determine retry strategy based on event importance
	maxRetries := 10
	baseDelay := 30
	isPriority := false

	// Critical events get more aggressive retry
	if event.EventType == "payment.succeeded" || event.EventType == "subscription.cancelled" {
		maxRetries = 15
		baseDelay = 10
		isPriority = true
	}

	task := &rebound.Task{
		ID:     fmt.Sprintf("webhook-%s-%s-%d", webhook.CustomerID, event.EventID, time.Now().Unix()),
		Source: "webhook-service",
		Destination: rebound.Destination{
			URL: webhook.URL,
		},
		DeadDestination: rebound.Destination{
			// Send failed webhooks to internal audit endpoint
			URL: "http://internal-api.brevo.com/webhooks/failed",
		},
		MaxRetries:      maxRetries,
		BaseDelay:       baseDelay,
		ClientID:        webhook.CustomerID,
		IsPriority:      isPriority,
		MessageData:     string(eventJSON),
		DestinationType: rebound.DestinationTypeHTTP,
	}

	if err := s.rebound.CreateTask(ctx, task); err != nil {
		return fmt.Errorf("creating retry task: %w", err)
	}

	s.logger.Info("webhook retry scheduled",
		zap.String("customer_id", webhook.CustomerID),
		zap.String("event_id", event.EventID),
		zap.String("task_id", task.ID),
		zap.Int("max_retries", maxRetries),
		zap.Bool("is_priority", isPriority),
	)

	return nil
}

// BroadcastEvent sends an event to all registered customer webhooks
func (s *WebhookService) BroadcastEvent(ctx context.Context, event *WebhookEvent, webhooks []*CustomerWebhook) {
	s.logger.Info("broadcasting event",
		zap.String("event_type", event.EventType),
		zap.Int("webhook_count", len(webhooks)),
	)

	for _, webhook := range webhooks {
		if err := s.DeliverWebhook(ctx, webhook, event); err != nil {
			s.logger.Error("failed to deliver webhook",
				zap.String("customer_id", webhook.CustomerID),
				zap.Error(err),
			)
		}
	}
}

func main() {
	fmt.Println("=== Rebound Example 3: Webhook Delivery Service ===\n")

	// Setup logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Configure Rebound
	cfg := &rebound.Config{
		RedisAddr:    "localhost:6379",
		KafkaBrokers: []string{"localhost:9092"},
		PollInterval: 1 * time.Second,
		Logger:       logger,
	}

	// Create Rebound
	rb, err := rebound.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create rebound: %v", err)
	}
	defer rb.Close()

	// Start worker
	ctx := context.Background()
	if err := rb.Start(ctx); err != nil {
		log.Fatalf("Failed to start rebound: %v", err)
	}

	fmt.Println("✓ Webhook delivery service with Rebound started\n")

	// Create webhook service
	webhookService := NewWebhookService(rb, logger)

	// Register customer webhooks
	customerWebhooks := []*CustomerWebhook{
		{
			CustomerID: "customer-001",
			URL:        "http://localhost:8090/webhook",
			Secret:     "secret-001",
		},
		{
			CustomerID: "customer-002",
			URL:        "http://localhost:8090/fail", // Will fail, trigger retries
			Secret:     "secret-002",
		},
		{
			CustomerID: "customer-003",
			URL:        "http://localhost:8090/webhook",
			Secret:     "secret-003",
		},
	}

	fmt.Printf("Registered %d customer webhooks\n\n", len(customerWebhooks))

	// Simulate different event types
	events := []*WebhookEvent{
		{
			EventID:   "evt-001",
			EventType: "contact.created",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"contact_id": "contact-12345",
				"email":      "john@example.com",
				"name":       "John Doe",
			},
		},
		{
			EventID:   "evt-002",
			EventType: "payment.succeeded",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"payment_id": "pay-67890",
				"amount":     9999,
				"currency":   "USD",
			},
		},
		{
			EventID:   "evt-003",
			EventType: "email.delivered",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"email_id": "email-111",
				"to":       "customer@example.com",
				"status":   "delivered",
			},
		},
	}

	// Broadcast each event to all customer webhooks
	for _, event := range events {
		fmt.Printf("📡 Broadcasting event: %s\n", event.EventType)
		webhookService.BroadcastEvent(ctx, event, customerWebhooks)
		time.Sleep(1 * time.Second)
		fmt.Println()
	}

	fmt.Println("\n📊 Summary:")
	fmt.Printf("   - Events broadcasted: %d\n", len(events))
	fmt.Printf("   - Total webhook deliveries: %d\n", len(events)*len(customerWebhooks))
	fmt.Printf("   - Failed webhooks are being retried automatically\n")

	fmt.Println("\n💡 Retry Strategy:")
	fmt.Println("   - Regular events: 10 retries, 30s base delay")
	fmt.Println("   - Critical events (payment, subscription): 15 retries, 10s base delay")
	fmt.Println("   - Exponential backoff: delay doubles each attempt")
	fmt.Println("   - Failed deliveries after max retries go to audit endpoint")

	fmt.Println("\nNote: Start webhook receiver with 'go run examples/webhook-receiver.go'")

	// Keep running
	fmt.Println("\nService running... Press Ctrl+C to exit")
	time.Sleep(30 * time.Second)
}
