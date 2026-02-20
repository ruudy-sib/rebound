package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/dig"
	"go.uber.org/zap"

	"github.com/ruudy-sib/rebound/pkg/rebound"
)

// Example 4: Dependency Injection Integration
// Shows how to integrate Rebound into a service using uber-go/dig (Brevo pattern)

// Config holds application configuration
type Config struct {
	HTTPPort     string
	RedisAddr    string
	KafkaBrokers []string
}

// OrderService handles order processing with retry support
type OrderService struct {
	rebound *rebound.Rebound
	logger  *zap.Logger
}

type Order struct {
	ID       string
	Customer string
	Amount   float64
}

func NewOrderService(rb *rebound.Rebound, logger *zap.Logger) *OrderService {
	return &OrderService{
		rebound: rb,
		logger:  logger.Named("order-service"),
	}
}

func (s *OrderService) ProcessOrder(ctx context.Context, order *Order) error {
	s.logger.Info("processing order", zap.String("order_id", order.ID))

	// Simulate order processing
	// If processing fails, schedule retry
	task := &rebound.Task{
		ID:     fmt.Sprintf("order-%s-%d", order.ID, time.Now().Unix()),
		Source: "order-service",
		Destination: rebound.Destination{
			Host:  "localhost",
			Port:  "9092",
			Topic: "order-processing",
		},
		DeadDestination: rebound.Destination{
			Host:  "localhost",
			Port:  "9092",
			Topic: "order-processing-dlq",
		},
		MaxRetries:      5,
		BaseDelay:       10,
		ClientID:        "order-service",
		MessageData:     fmt.Sprintf(`{"order_id":"%s","customer":"%s","amount":%.2f}`, order.ID, order.Customer, order.Amount),
		DestinationType: rebound.DestinationTypeKafka,
	}

	return s.rebound.CreateTask(ctx, task)
}

// HTTPHandler handles HTTP requests
type HTTPHandler struct {
	orderService *OrderService
	logger       *zap.Logger
}

func NewHTTPHandler(orderService *OrderService, logger *zap.Logger) *HTTPHandler {
	return &HTTPHandler{
		orderService: orderService,
		logger:       logger.Named("http-handler"),
	}
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse order from request (simplified)
	order := &Order{
		ID:       fmt.Sprintf("order-%d", time.Now().Unix()),
		Customer: "customer@example.com",
		Amount:   99.99,
	}

	if err := h.orderService.ProcessOrder(r.Context(), order); err != nil {
		h.logger.Error("failed to process order", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, `{"order_id":"%s","status":"processing"}`, order.ID)
}

// HTTPServer wraps http.Server with lifecycle management
type HTTPServer struct {
	server  *http.Server
	handler *HTTPHandler
	logger  *zap.Logger
}

func NewHTTPServer(handler *HTTPHandler, cfg *Config, logger *zap.Logger) *HTTPServer {
	mux := http.NewServeMux()
	mux.Handle("/orders", handler)

	return &HTTPServer{
		server: &http.Server{
			Addr:    ":" + cfg.HTTPPort,
			Handler: mux,
		},
		handler: handler,
		logger:  logger.Named("http-server"),
	}
}

func (s *HTTPServer) Start() error {
	s.logger.Info("starting http server", zap.String("addr", s.server.Addr))
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	s.logger.Info("stopping http server")
	return s.server.Shutdown(ctx)
}

// Application lifecycle
type Application struct {
	server  *HTTPServer
	rebound *rebound.Rebound
	logger  *zap.Logger
}

func NewApplication(server *HTTPServer, rb *rebound.Rebound, logger *zap.Logger) *Application {
	return &Application{
		server:  server,
		rebound: rb,
		logger:  logger,
	}
}

func (app *Application) Run(ctx context.Context) error {
	// Start Rebound worker
	if err := app.rebound.Start(ctx); err != nil {
		return fmt.Errorf("starting rebound: %w", err)
	}
	app.logger.Info("rebound worker started")

	// Start HTTP server in background
	go func() {
		if err := app.server.Start(); err != nil {
			app.logger.Error("http server error", zap.Error(err))
		}
	}()

	app.logger.Info("application started successfully")

	// Wait for shutdown signal
	<-ctx.Done()

	app.logger.Info("shutting down application")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.server.Stop(shutdownCtx); err != nil {
		app.logger.Error("error stopping http server", zap.Error(err))
	}

	if err := app.rebound.Close(); err != nil {
		app.logger.Error("error closing rebound", zap.Error(err))
	}

	app.logger.Info("application stopped")
	return nil
}

// buildContainer sets up dependency injection
func buildContainer() (*dig.Container, error) {
	container := dig.New()

	// Provide configuration
	container.Provide(func() *Config {
		return &Config{
			HTTPPort:     getEnv("HTTP_PORT", "8080"),
			RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
			KafkaBrokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
		}
	})

	// Provide logger
	container.Provide(func() (*zap.Logger, error) {
		if getEnv("ENVIRONMENT", "dev") == "production" {
			return zap.NewProduction()
		}
		return zap.NewDevelopment()
	})

	// Provide Rebound configuration
	container.Provide(func(cfg *Config, logger *zap.Logger) *rebound.Config {
		return &rebound.Config{
			RedisAddr:    cfg.RedisAddr,
			KafkaBrokers: cfg.KafkaBrokers,
			PollInterval: 1 * time.Second,
			Logger:       logger,
		}
	})

	// Register Rebound (using helper function)
	if err := rebound.RegisterWithContainer(container); err != nil {
		return nil, fmt.Errorf("registering rebound: %w", err)
	}

	// Provide domain services
	container.Provide(NewOrderService)

	// Provide HTTP components
	container.Provide(NewHTTPHandler)
	container.Provide(NewHTTPServer)

	// Provide application
	container.Provide(NewApplication)

	return container, nil
}

func main() {
	fmt.Println("=== Rebound Example 4: Dependency Injection Integration ===\n")

	// Build DI container
	container, err := buildContainer()
	if err != nil {
		log.Fatalf("Failed to build container: %v", err)
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal...")
		cancel()
	}()

	// Run application
	err = container.Invoke(func(app *Application) error {
		return app.Run(ctx)
	})

	if err != nil {
		log.Fatalf("Application error: %v", err)
	}

	fmt.Println("\n✓ Application shut down successfully")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
