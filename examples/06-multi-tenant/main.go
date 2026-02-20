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

// Example 6: Multi-Tenant Service with Tenant-Specific Retry Policies
// Shows how different tenants can have different retry configurations

type TenantService struct {
	rebound *rebound.Rebound
	logger  *zap.Logger
}

type Tenant struct {
	ID          string
	Name        string
	Plan        string // "free", "pro", "enterprise"
	RetryPolicy RetryPolicy
}

type RetryPolicy struct {
	MaxRetries int
	BaseDelay  int
	IsPriority bool
}

type NotificationEvent struct {
	TenantID  string                 `json:"tenant_id"`
	EventType string                 `json:"event_type"`
	Payload   map[string]interface{} `json:"payload"`
}

func NewTenantService(rb *rebound.Rebound, logger *zap.Logger) *TenantService {
	return &TenantService{
		rebound: rb,
		logger:  logger.Named("tenant-service"),
	}
}

// GetTenantRetryPolicy returns retry policy based on tenant plan
func (s *TenantService) GetTenantRetryPolicy(tenant *Tenant) RetryPolicy {
	switch tenant.Plan {
	case "enterprise":
		// Enterprise: aggressive retry, high priority
		return RetryPolicy{
			MaxRetries: 15,
			BaseDelay:  5,
			IsPriority: true,
		}

	case "pro":
		// Pro: moderate retry, normal priority
		return RetryPolicy{
			MaxRetries: 10,
			BaseDelay:  10,
			IsPriority: false,
		}

	case "free":
		// Free: limited retry, low priority
		return RetryPolicy{
			MaxRetries: 3,
			BaseDelay:  30,
			IsPriority: false,
		}

	default:
		return RetryPolicy{
			MaxRetries: 5,
			BaseDelay:  15,
			IsPriority: false,
		}
	}
}

// SendNotification sends a notification to tenant's webhook with tenant-specific retry
func (s *TenantService) SendNotification(ctx context.Context, tenant *Tenant, event *NotificationEvent) error {
	s.logger.Info("sending notification",
		zap.String("tenant_id", tenant.ID),
		zap.String("tenant_plan", tenant.Plan),
		zap.String("event_type", event.EventType),
	)

	// Get tenant-specific retry policy
	policy := s.GetTenantRetryPolicy(tenant)

	// Serialize event
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	// Create retry task with tenant-specific policy
	task := &rebound.Task{
		ID:     fmt.Sprintf("notification-%s-%d", tenant.ID, time.Now().Unix()),
		Source: "tenant-service",
		Destination: rebound.Destination{
			URL: fmt.Sprintf("http://tenant-webhooks.brevo.com/%s/notifications", tenant.ID),
		},
		DeadDestination: rebound.Destination{
			URL: fmt.Sprintf("http://internal-api.brevo.com/tenants/%s/failed-notifications", tenant.ID),
		},
		MaxRetries:      policy.MaxRetries,
		BaseDelay:       policy.BaseDelay,
		ClientID:        tenant.ID,
		IsPriority:      policy.IsPriority,
		MessageData:     string(eventJSON),
		DestinationType: rebound.DestinationTypeHTTP,
	}

	if err := s.rebound.CreateTask(ctx, task); err != nil {
		return fmt.Errorf("creating retry task: %w", err)
	}

	s.logger.Info("notification scheduled",
		zap.String("tenant_id", tenant.ID),
		zap.String("task_id", task.ID),
		zap.Int("max_retries", policy.MaxRetries),
		zap.Int("base_delay", policy.BaseDelay),
		zap.Bool("is_priority", policy.IsPriority),
	)

	return nil
}

// BroadcastToAllTenants sends the same event to all tenants
func (s *TenantService) BroadcastToAllTenants(ctx context.Context, tenants []*Tenant, eventType string, payload map[string]interface{}) {
	s.logger.Info("broadcasting to all tenants",
		zap.String("event_type", eventType),
		zap.Int("tenant_count", len(tenants)),
	)

	for _, tenant := range tenants {
		event := &NotificationEvent{
			TenantID:  tenant.ID,
			EventType: eventType,
			Payload:   payload,
		}

		if err := s.SendNotification(ctx, tenant, event); err != nil {
			s.logger.Error("failed to send notification",
				zap.String("tenant_id", tenant.ID),
				zap.Error(err),
			)
		}
	}
}

// CalculateRetrySchedule shows when retries will occur for a tenant
func CalculateRetrySchedule(policy RetryPolicy) []time.Duration {
	var schedule []time.Duration
	for i := 0; i < policy.MaxRetries; i++ {
		delay := time.Duration(policy.BaseDelay) * time.Second * time.Duration(1<<i)
		schedule = append(schedule, delay)
	}
	return schedule
}

func main() {
	fmt.Println("=== Rebound Example 6: Multi-Tenant Service ===\n")

	// Setup logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Configure Rebound
	cfg := &rebound.Config{
		RedisAddr:    "localhost:6379",
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

	fmt.Println("✓ Multi-tenant service with Rebound started\n")

	// Create tenant service
	tenantService := NewTenantService(rb, logger)

	// Define tenants with different plans
	tenants := []*Tenant{
		{
			ID:   "tenant-001",
			Name: "Acme Corp",
			Plan: "enterprise",
		},
		{
			ID:   "tenant-002",
			Name: "TechStart Inc",
			Plan: "pro",
		},
		{
			ID:   "tenant-003",
			Name: "SmallBiz LLC",
			Plan: "pro",
		},
		{
			ID:   "tenant-004",
			Name: "Startup XYZ",
			Plan: "free",
		},
		{
			ID:   "tenant-005",
			Name: "Individual User",
			Plan: "free",
		},
	}

	fmt.Println("📊 Tenant Retry Policies:\n")
	fmt.Println("┌──────────────┬──────────────────┬────────────┬──────────┬───────────┬──────────┐")
	fmt.Println("│ Tenant ID    │ Name             │ Plan       │ Retries  │ Base Delay│ Priority │")
	fmt.Println("├──────────────┼──────────────────┼────────────┼──────────┼───────────┼──────────┤")

	for _, tenant := range tenants {
		policy := tenantService.GetTenantRetryPolicy(tenant)
		priority := "No"
		if policy.IsPriority {
			priority = "Yes"
		}
		fmt.Printf("│ %-12s │ %-16s │ %-10s │    %2d    │    %3ds   │   %-3s    │\n",
			tenant.ID, tenant.Name, tenant.Plan, policy.MaxRetries, policy.BaseDelay, priority)
	}
	fmt.Println("└──────────────┴──────────────────┴────────────┴──────────┴───────────┴──────────┘")

	// Show retry schedules for each plan
	fmt.Println("\n⏱️  Retry Schedules by Plan:\n")

	for _, plan := range []string{"enterprise", "pro", "free"} {
		policy := tenantService.GetTenantRetryPolicy(&Tenant{Plan: plan})
		schedule := CalculateRetrySchedule(policy)

		fmt.Printf("%s Plan:\n", plan)
		for i, delay := range schedule {
			fmt.Printf("  Attempt %d: %v delay\n", i+1, delay)
		}
		fmt.Println()
	}

	// Broadcast event to all tenants
	fmt.Println("📡 Broadcasting system update to all tenants...\n")

	event := map[string]interface{}{
		"update_type": "feature_release",
		"version":     "2.0.0",
		"features": []string{
			"New dashboard",
			"Advanced analytics",
			"API v2",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	tenantService.BroadcastToAllTenants(ctx, tenants, "system.update", event)

	fmt.Println("\n✅ Notifications scheduled for all tenants")

	fmt.Println("\n💡 Key Insights:")
	fmt.Println("   • Enterprise tenants get 15 retries with 5s base delay (priority)")
	fmt.Println("   • Pro tenants get 10 retries with 10s base delay")
	fmt.Println("   • Free tenants get 3 retries with 30s base delay")
	fmt.Println("   • All retries use exponential backoff")
	fmt.Println("   • Each tenant has isolated retry tracking")

	fmt.Println("\n📈 Benefits of Tenant-Specific Policies:")
	fmt.Println("   ✓ Fair resource allocation by plan tier")
	fmt.Println("   ✓ Premium customers get better reliability")
	fmt.Println("   ✓ Prevent free tier from overwhelming the system")
	fmt.Println("   ✓ Isolated failure domains per tenant")

	fmt.Println("\nService running... Press Ctrl+C to exit")
	time.Sleep(30 * time.Second)
}
