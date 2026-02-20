package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"

	"github.com/ruudy-sib/rebound/pkg/rebound"
)

// Example 5: Payment Processing with Smart Retry
// Shows how to use different retry strategies based on failure type

type PaymentService struct {
	rebound *rebound.Rebound
	logger  *zap.Logger
}

type Payment struct {
	ID          string  `json:"id"`
	CustomerID  string  `json:"customer_id"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Method      string  `json:"method"`
	Description string  `json:"description"`
}

type PaymentFailure string

const (
	FailureInsufficientFunds PaymentFailure = "insufficient_funds"
	FailureNetworkError      PaymentFailure = "network_error"
	FailureInvalidCard       PaymentFailure = "invalid_card"
	FailureGatewayTimeout    PaymentFailure = "gateway_timeout"
	FailureRateLimited       PaymentFailure = "rate_limited"
)

func NewPaymentService(rb *rebound.Rebound, logger *zap.Logger) *PaymentService {
	return &PaymentService{
		rebound: rb,
		logger:  logger.Named("payment-service"),
	}
}

// ProcessPayment attempts to process a payment
func (s *PaymentService) ProcessPayment(ctx context.Context, payment *Payment) error {
	s.logger.Info("processing payment",
		zap.String("payment_id", payment.ID),
		zap.Float64("amount", payment.Amount),
	)

	// In real implementation, call payment gateway here
	// For demo, we'll simulate failures
	return s.handlePaymentFailure(ctx, payment, FailureNetworkError)
}

// handlePaymentFailure determines retry strategy based on failure type
func (s *PaymentService) handlePaymentFailure(ctx context.Context, payment *Payment, failure PaymentFailure) error {
	s.logger.Warn("payment failed",
		zap.String("payment_id", payment.ID),
		zap.String("failure_type", string(failure)),
	)

	strategy := s.getRetryStrategy(failure)
	return s.scheduleRetry(ctx, payment, strategy)
}

type RetryStrategy struct {
	MaxRetries int
	BaseDelay  int
	IsPriority bool
	Reason     string
}

// getRetryStrategy returns appropriate retry strategy for each failure type
func (s *PaymentService) getRetryStrategy(failure PaymentFailure) RetryStrategy {
	switch failure {
	case FailureNetworkError, FailureGatewayTimeout:
		// Network issues: aggressive retry
		return RetryStrategy{
			MaxRetries: 10,
			BaseDelay:  5,
			IsPriority: true,
			Reason:     "Network/gateway issues are usually temporary",
		}

	case FailureRateLimited:
		// Rate limiting: slower retry to avoid hammering
		return RetryStrategy{
			MaxRetries: 5,
			BaseDelay:  60,
			IsPriority: false,
			Reason:     "Respect rate limits with longer delays",
		}

	case FailureInsufficientFunds:
		// Insufficient funds: slow retry (user needs to add funds)
		return RetryStrategy{
			MaxRetries: 3,
			BaseDelay:  300, // 5 minutes
			IsPriority: false,
			Reason:     "User needs time to add funds",
		}

	case FailureInvalidCard:
		// Invalid card: no retry (permanent failure)
		return RetryStrategy{
			MaxRetries: 0,
			BaseDelay:  0,
			IsPriority: false,
			Reason:     "Permanent failure, no retry needed",
		}

	default:
		// Unknown error: moderate retry
		return RetryStrategy{
			MaxRetries: 5,
			BaseDelay:  10,
			IsPriority: false,
			Reason:     "Unknown error, moderate retry",
		}
	}
}

func (s *PaymentService) scheduleRetry(ctx context.Context, payment *Payment, strategy RetryStrategy) error {
	// Don't retry permanent failures
	if strategy.MaxRetries == 0 {
		s.logger.Info("permanent failure, not scheduling retry",
			zap.String("payment_id", payment.ID),
			zap.String("reason", strategy.Reason),
		)
		return s.sendToFailureQueue(ctx, payment)
	}

	// Serialize payment
	paymentJSON, err := json.Marshal(payment)
	if err != nil {
		return fmt.Errorf("marshaling payment: %w", err)
	}

	// Create retry task
	task := &rebound.Task{
		ID:     fmt.Sprintf("payment-%s-%d", payment.ID, time.Now().Unix()),
		Source: "payment-service",
		Destination: rebound.Destination{
			URL: "http://payment-gateway.brevo.com/process",
		},
		DeadDestination: rebound.Destination{
			URL: "http://internal-api.brevo.com/payments/failed",
		},
		MaxRetries:      strategy.MaxRetries,
		BaseDelay:       strategy.BaseDelay,
		ClientID:        payment.CustomerID,
		IsPriority:      strategy.IsPriority,
		MessageData:     string(paymentJSON),
		DestinationType: rebound.DestinationTypeHTTP,
	}

	if err := s.rebound.CreateTask(ctx, task); err != nil {
		return fmt.Errorf("creating retry task: %w", err)
	}

	s.logger.Info("payment retry scheduled",
		zap.String("payment_id", payment.ID),
		zap.String("task_id", task.ID),
		zap.Int("max_retries", strategy.MaxRetries),
		zap.Int("base_delay", strategy.BaseDelay),
		zap.Bool("is_priority", strategy.IsPriority),
		zap.String("reason", strategy.Reason),
	)

	return nil
}

func (s *PaymentService) sendToFailureQueue(ctx context.Context, payment *Payment) error {
	paymentJSON, _ := json.Marshal(payment)

	task := &rebound.Task{
		ID:     fmt.Sprintf("payment-failed-%s", payment.ID),
		Source: "payment-service",
		Destination: rebound.Destination{
			URL: "http://internal-api.brevo.com/payments/permanent-failures",
		},
		MaxRetries:      1, // Just deliver once
		BaseDelay:       0,
		ClientID:        payment.CustomerID,
		MessageData:     string(paymentJSON),
		DestinationType: rebound.DestinationTypeHTTP,
	}

	return s.rebound.CreateTask(ctx, task)
}

func main() {
	fmt.Println("=== Rebound Example 5: Payment Processing with Smart Retry ===\n")

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

	fmt.Println("✓ Payment service with smart retry started\n")

	// Create payment service
	paymentService := NewPaymentService(rb, logger)

	// Test different failure scenarios
	fmt.Println("💳 Testing different payment failure scenarios:\n")

	scenarios := []struct {
		payment *Payment
		failure PaymentFailure
	}{
		{
			payment: &Payment{
				ID:          "pay-001",
				CustomerID:  "cust-123",
				Amount:      99.99,
				Currency:    "USD",
				Method:      "card",
				Description: "Subscription payment",
			},
			failure: FailureNetworkError,
		},
		{
			payment: &Payment{
				ID:          "pay-002",
				CustomerID:  "cust-456",
				Amount:      49.99,
				Currency:    "EUR",
				Method:      "card",
				Description: "One-time purchase",
			},
			failure: FailureRateLimited,
		},
		{
			payment: &Payment{
				ID:          "pay-003",
				CustomerID:  "cust-789",
				Amount:      199.99,
				Currency:    "USD",
				Method:      "card",
				Description: "Annual subscription",
			},
			failure: FailureInsufficientFunds,
		},
		{
			payment: &Payment{
				ID:          "pay-004",
				CustomerID:  "cust-101",
				Amount:      29.99,
				Currency:    "GBP",
				Method:      "card",
				Description: "Monthly payment",
			},
			failure: FailureInvalidCard,
		},
	}

	for i, scenario := range scenarios {
		fmt.Printf("%d. Payment: %s (%.2f %s)\n", i+1, scenario.payment.ID, scenario.payment.Amount, scenario.payment.Currency)
		fmt.Printf("   Failure Type: %s\n", scenario.failure)

		strategy := paymentService.getRetryStrategy(scenario.failure)
		fmt.Printf("   Retry Strategy:\n")
		fmt.Printf("     - Max Retries: %d\n", strategy.MaxRetries)
		fmt.Printf("     - Base Delay: %ds\n", strategy.BaseDelay)
		fmt.Printf("     - Priority: %v\n", strategy.IsPriority)
		fmt.Printf("     - Reason: %s\n", strategy.Reason)

		paymentService.handlePaymentFailure(ctx, scenario.payment, scenario.failure)
		fmt.Println()
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\n📊 Retry Strategy Summary:")
	fmt.Println("\n┌─────────────────────────┬──────────┬───────────┬──────────┬─────────────────────────┐")
	fmt.Println("│ Failure Type            │ Retries  │ Base Delay│ Priority │ Reason                  │")
	fmt.Println("├─────────────────────────┼──────────┼───────────┼──────────┼─────────────────────────┤")
	fmt.Println("│ Network Error           │    10    │    5s     │   Yes    │ Usually temporary       │")
	fmt.Println("│ Gateway Timeout         │    10    │    5s     │   Yes    │ Usually temporary       │")
	fmt.Println("│ Rate Limited            │     5    │   60s     │   No     │ Respect rate limits     │")
	fmt.Println("│ Insufficient Funds      │     3    │  300s     │   No     │ User needs time         │")
	fmt.Println("│ Invalid Card            │     0    │    -      │   No     │ Permanent failure       │")
	fmt.Println("└─────────────────────────┴──────────┴───────────┴──────────┴─────────────────────────┘")

	fmt.Println("\n💡 Key Insights:")
	fmt.Println("   • Transient failures (network) → aggressive retry")
	fmt.Println("   • Rate limiting → respect backoff, slower retry")
	fmt.Println("   • User action needed (funds) → very slow retry")
	fmt.Println("   • Permanent failures (invalid card) → no retry, immediate failure queue")

	fmt.Println("\nService running... Press Ctrl+C to exit")
	time.Sleep(30 * time.Second)
}
