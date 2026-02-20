package rebound

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/ruudy-sib/rebound/internal/adapter/primary/worker"
	"github.com/ruudy-sib/rebound/internal/adapter/secondary/httpproducer"
	"github.com/ruudy-sib/rebound/internal/adapter/secondary/kafkaproducer"
	"github.com/ruudy-sib/rebound/internal/adapter/secondary/producerfactory"
	"github.com/ruudy-sib/rebound/internal/adapter/secondary/redisstore"
	"github.com/ruudy-sib/rebound/internal/config"
	"github.com/ruudy-sib/rebound/internal/domain/entity"
	"github.com/ruudy-sib/rebound/internal/domain/service"
	"github.com/ruudy-sib/rebound/internal/port/primary"
	"github.com/ruudy-sib/rebound/internal/port/secondary"
)

// Rebound is the main entry point for the retry service.
// It can be embedded in other Go applications to provide retry functionality.
type Rebound struct {
	taskService primary.TaskService
	worker      *worker.Worker
	producer    secondary.MessageProducer
	redisClient goredis.UniversalClient
	logger      *zap.Logger
	config      *Config
}

// Config holds configuration for Rebound.
type Config struct {
	// Redis mode: "standalone" (default), "sentinel", "cluster"
	RedisMode string

	// Standalone Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Sentinel Redis (RedisMode = "sentinel")
	RedisMasterName    string
	RedisSentinelAddrs []string

	// Cluster Redis (RedisMode = "cluster")
	RedisClusterAddrs []string

	// Worker configuration
	PollInterval time.Duration

	// Logger (if nil, a default logger will be created)
	Logger *zap.Logger
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		RedisAddr:    "localhost:6379",
		RedisPassword: "",
		RedisDB:      0,
		PollInterval: 1 * time.Second,
	}
}

// New creates a new Rebound instance with the given configuration.
func New(cfg *Config) (*Rebound, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Create logger if not provided
	logger := cfg.Logger
	if logger == nil {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, fmt.Errorf("creating logger: %w", err)
		}
	}

	// Convert to internal config format
	internalCfg := &config.Config{
		RedisMode:          cfg.RedisMode,
		RedisAddr:          cfg.RedisAddr,
		RedisPassword:      cfg.RedisPassword,
		RedisDB:            cfg.RedisDB,
		RedisMasterName:    cfg.RedisMasterName,
		RedisSentinelAddrs: cfg.RedisSentinelAddrs,
		RedisClusterAddrs:  cfg.RedisClusterAddrs,
		PollInterval:       cfg.PollInterval,
	}

	// Create Redis client
	redisClient, err := redisstore.NewClient(context.Background(), internalCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("creating redis client: %w", err)
	}

	// Create scheduler
	scheduler := redisstore.NewScheduler(redisClient, logger)

	// Create producers — Kafka connections are established per destination at delivery time.
	kafkaProd := kafkaproducer.NewDestinationProducer(logger)
	httpProd := httpproducer.NewProducer(internalCfg, logger)
	producer := producerfactory.NewFactory(kafkaProd, httpProd, logger)

	// Create domain service
	taskService := service.NewTaskService(scheduler, producer, logger)

	// Create worker
	wrk := worker.NewWorker(taskService, cfg.PollInterval, logger)

	return &Rebound{
		taskService: taskService,
		worker:      wrk,
		producer:    producer,
		redisClient: redisClient,
		logger:      logger,
		config:      cfg,
	}, nil
}

// Start begins the retry worker in the background.
// It returns immediately and the worker runs in a separate goroutine.
func (r *Rebound) Start(ctx context.Context) error {
	r.logger.Info("starting rebound retry service")
	go r.worker.Run(ctx)
	return nil
}

// CreateTask schedules a new task for retry with exponential backoff.
func (r *Rebound) CreateTask(ctx context.Context, task *Task) error {
	domainTask := task.toDomain()
	return r.taskService.CreateTask(ctx, domainTask)
}

// Close gracefully shuts down the Rebound service and releases resources.
func (r *Rebound) Close() error {
	r.logger.Info("shutting down rebound retry service")

	var errs []error

	if err := r.producer.Close(); err != nil {
		errs = append(errs, fmt.Errorf("closing producer: %w", err))
	}

	if err := r.redisClient.Close(); err != nil {
		errs = append(errs, fmt.Errorf("closing redis client: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	return nil
}

// Task represents a retryable task to be scheduled.
type Task struct {
	// ID is a unique identifier for the task
	ID string

	// Source identifies the originating system or application
	Source string

	// Destination is where the message should be delivered
	Destination Destination

	// DeadDestination is where the message goes after max retries
	DeadDestination Destination

	// MaxRetries is the maximum number of retry attempts (0-100)
	MaxRetries int

	// BaseDelay is the base delay in seconds for exponential backoff (1-3600)
	BaseDelay int

	// ClientID identifies the client making the request
	ClientID string

	// IsPriority indicates if this is a priority task
	IsPriority bool

	// MessageData is the actual payload to be delivered
	MessageData string

	// DestinationType is either "kafka" or "http"
	DestinationType DestinationType
}

// DestinationType specifies how the message should be delivered.
type DestinationType string

const (
	// DestinationTypeKafka sends messages to Kafka topics
	DestinationTypeKafka DestinationType = "kafka"

	// DestinationTypeHTTP sends messages via HTTP POST
	DestinationTypeHTTP DestinationType = "http"
)

// Destination specifies where a message should be delivered.
// For Kafka: use Host, Port, and Topic.
// For HTTP: use URL.
type Destination struct {
	// Kafka fields
	Host  string
	Port  string
	Topic string

	// HTTP field
	URL string
}

// toDomain converts a public Task to an internal domain entity.
func (t *Task) toDomain() *entity.Task {
	return &entity.Task{
		ID:     t.ID,
		Source: t.Source,
		Destination: entity.Destination{
			Host:  t.Destination.Host,
			Port:  t.Destination.Port,
			Topic: t.Destination.Topic,
			URL:   t.Destination.URL,
		},
		DeadDestination: entity.Destination{
			Host:  t.DeadDestination.Host,
			Port:  t.DeadDestination.Port,
			Topic: t.DeadDestination.Topic,
			URL:   t.DeadDestination.URL,
		},
		MaxRetries:      t.MaxRetries,
		BaseDelay:       t.BaseDelay,
		ClientID:        t.ClientID,
		IsPriority:      t.IsPriority,
		MessageData:     t.MessageData,
		DestinationType: entity.DestinationType(t.DestinationType),
	}
}
