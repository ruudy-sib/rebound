package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.uber.org/zap"

	"github.com/ruudy-sib/rebound/pkg/rebound"
)

// Example 2: Email Service Integration
// Shows how to use Rebound to retry failed email deliveries

type EmailService struct {
	rebound *rebound.Rebound
	logger  *zap.Logger
}

type Email struct {
	ID        string    `json:"id"`
	To        string    `json:"to"`
	From      string    `json:"from"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
}

func NewEmailService(rb *rebound.Rebound, logger *zap.Logger) *EmailService {
	return &EmailService{
		rebound: rb,
		logger:  logger.Named("email-service"),
	}
}

// SendEmail attempts to send an email, schedules retry on failure
func (s *EmailService) SendEmail(ctx context.Context, email *Email) error {
	s.logger.Info("attempting to send email",
		zap.String("email_id", email.ID),
		zap.String("to", email.To),
	)

	// Simulate email sending (80% success rate)
	if rand.Float64() < 0.8 {
		s.logger.Info("email sent successfully", zap.String("email_id", email.ID))
		return nil
	}

	// Email sending failed, schedule retry
	s.logger.Warn("email sending failed, scheduling retry", zap.String("email_id", email.ID))
	return s.scheduleRetry(ctx, email)
}

// scheduleRetry creates a retry task for failed email delivery
func (s *EmailService) scheduleRetry(ctx context.Context, email *Email) error {
	// Serialize email to JSON
	emailJSON, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("marshaling email: %w", err)
	}

	// Create retry task with HTTP webhook
	// In production, this would call your email delivery API
	task := &rebound.Task{
		ID:     fmt.Sprintf("email-%s-%d", email.ID, time.Now().Unix()),
		Source: "email-service",
		Destination: rebound.Destination{
			URL: "http://localhost:8090/webhook", // Email delivery API
		},
		DeadDestination: rebound.Destination{
			URL: "http://localhost:8090/dlq", // Failed emails endpoint
		},
		MaxRetries:      5,  // Retry up to 5 times
		BaseDelay:       10, // Start with 10s, exponential backoff
		ClientID:        "email-service",
		IsPriority:      false,
		MessageData:     string(emailJSON),
		DestinationType: rebound.DestinationTypeHTTP,
	}

	if err := s.rebound.CreateTask(ctx, task); err != nil {
		return fmt.Errorf("creating retry task: %w", err)
	}

	s.logger.Info("retry scheduled",
		zap.String("email_id", email.ID),
		zap.String("task_id", task.ID),
		zap.Int("max_retries", task.MaxRetries),
	)

	return nil
}

// SendBulkEmails sends multiple emails with retry support
func (s *EmailService) SendBulkEmails(ctx context.Context, emails []*Email) (sent, failed int) {
	for _, email := range emails {
		if err := s.SendEmail(ctx, email); err != nil {
			s.logger.Error("failed to send email", zap.String("email_id", email.ID), zap.Error(err))
			failed++
		} else {
			sent++
		}
	}
	return sent, failed
}

func main() {
	fmt.Println("=== Rebound Example 2: Email Service Integration ===\n")

	// Setup logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Configure Rebound
	cfg := &rebound.Config{
		RedisAddr:    "localhost:6379",
		PollInterval: 1 * time.Second,
		Logger:       logger,
	}

	// Create Rebound instance
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

	fmt.Println("✓ Email service with Rebound started\n")

	// Create email service
	emailService := NewEmailService(rb, logger)

	// Prepare test emails
	emails := []*Email{
		{
			ID:        "email-001",
			To:        "customer1@example.com",
			From:      "noreply@brevo.com",
			Subject:   "Welcome to Brevo!",
			Body:      "Thank you for signing up...",
			Timestamp: time.Now(),
		},
		{
			ID:        "email-002",
			To:        "customer2@example.com",
			From:      "noreply@brevo.com",
			Subject:   "Your order confirmation",
			Body:      "Your order #12345 has been confirmed...",
			Timestamp: time.Now(),
		},
		{
			ID:        "email-003",
			To:        "customer3@example.com",
			From:      "noreply@brevo.com",
			Subject:   "Password reset request",
			Body:      "Click here to reset your password...",
			Timestamp: time.Now(),
		},
		{
			ID:        "email-004",
			To:        "customer4@example.com",
			From:      "noreply@brevo.com",
			Subject:   "Monthly newsletter",
			Body:      "Check out our latest updates...",
			Timestamp: time.Now(),
		},
		{
			ID:        "email-005",
			To:        "customer5@example.com",
			From:      "noreply@brevo.com",
			Subject:   "Account verification",
			Body:      "Please verify your email address...",
			Timestamp: time.Now(),
		},
	}

	// Send bulk emails
	fmt.Printf("Sending %d emails...\n\n", len(emails))
	sent, failed := emailService.SendBulkEmails(ctx, emails)

	fmt.Printf("\n📊 Results:\n")
	fmt.Printf("   ✓ Sent successfully: %d\n", sent)
	fmt.Printf("   ⚠ Failed (retrying): %d\n", failed)
	fmt.Printf("\n💡 Failed emails are being retried with exponential backoff:\n")
	fmt.Printf("   - Attempt 1: 10s delay\n")
	fmt.Printf("   - Attempt 2: 20s delay\n")
	fmt.Printf("   - Attempt 3: 40s delay\n")
	fmt.Printf("   - Attempt 4: 80s delay\n")
	fmt.Printf("   - Attempt 5: 160s delay\n")
	fmt.Printf("   - After 5 attempts: Sent to dead letter queue\n")

	// Keep running to let retries process
	fmt.Println("\nKeeping service running to process retries...")
	fmt.Println("Press Ctrl+C to exit")
	time.Sleep(30 * time.Second)
}
