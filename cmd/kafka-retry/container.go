package main

import (
	"context"
	"net/http"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/dig"
	"go.uber.org/zap"

	httphandler "kafkaretry/internal/adapter/primary/http"
	"kafkaretry/internal/adapter/primary/worker"
	"kafkaretry/internal/adapter/secondary/httpproducer"
	"kafkaretry/internal/adapter/secondary/kafkaproducer"
	"kafkaretry/internal/adapter/secondary/producerfactory"
	"kafkaretry/internal/adapter/secondary/redisstore"
	"kafkaretry/internal/config"
	"kafkaretry/internal/domain/service"
	"kafkaretry/internal/port/primary"
	"kafkaretry/internal/port/secondary"
)

func buildContainer(ctx context.Context) (*dig.Container, error) {
	c := dig.New()

	// --- Configuration ---
	if err := c.Provide(config.New); err != nil {
		return nil, err
	}

	// --- Logger ---
	if err := c.Provide(newLogger); err != nil {
		return nil, err
	}

	// --- Secondary Adapters (infrastructure) ---

	// Redis client
	if err := c.Provide(func(cfg *config.Config, logger *zap.Logger) (*goredis.Client, error) {
		return redisstore.NewClient(ctx, cfg, logger)
	}); err != nil {
		return nil, err
	}

	// Task scheduler (implements secondary.TaskScheduler)
	if err := c.Provide(func(client *goredis.Client, logger *zap.Logger) secondary.TaskScheduler {
		return redisstore.NewScheduler(client, logger)
	}); err != nil {
		return nil, err
	}

	// Redis health check (implements secondary.HealthChecker)
	if err := c.Provide(func(client *goredis.Client) secondary.HealthChecker {
		return redisstore.NewHealthCheck(client)
	}); err != nil {
		return nil, err
	}

	// Collect all health checks
	if err := c.Provide(func(redisCheck secondary.HealthChecker) []secondary.HealthChecker {
		return []secondary.HealthChecker{redisCheck}
	}); err != nil {
		return nil, err
	}

	// Kafka producer
	if err := c.Provide(func(cfg *config.Config, logger *zap.Logger) *kafkaproducer.Producer {
		return kafkaproducer.NewProducer(cfg, logger).(*kafkaproducer.Producer)
	}); err != nil {
		return nil, err
	}

	// HTTP producer
	if err := c.Provide(func(cfg *config.Config, logger *zap.Logger) *httpproducer.Producer {
		return httpproducer.NewProducer(cfg, logger).(*httpproducer.Producer)
	}); err != nil {
		return nil, err
	}

	// Producer factory (implements secondary.MessageProducer)
	// Routes messages to the appropriate producer based on destination type
	if err := c.Provide(func(
		kafkaProd *kafkaproducer.Producer,
		httpProd *httpproducer.Producer,
		logger *zap.Logger,
	) secondary.MessageProducer {
		return producerfactory.NewFactory(kafkaProd, httpProd, logger)
	}); err != nil {
		return nil, err
	}

	// --- Domain Services ---

	if err := c.Provide(service.NewTaskService); err != nil {
		return nil, err
	}

	// Bind concrete TaskService to the primary port interface
	if err := c.Provide(func(s *service.TaskService) primary.TaskService {
		return s
	}); err != nil {
		return nil, err
	}

	// --- Primary Adapters ---

	// HTTP router
	if err := c.Provide(func(taskSvc primary.TaskService, checks []secondary.HealthChecker, logger *zap.Logger) http.Handler {
		return httphandler.NewRouter(taskSvc, checks, logger)
	}); err != nil {
		return nil, err
	}

	// Worker
	if err := c.Provide(func(taskSvc primary.TaskService, cfg *config.Config, logger *zap.Logger) *worker.Worker {
		return worker.NewWorker(taskSvc, cfg.PollInterval, logger)
	}); err != nil {
		return nil, err
	}

	return c, nil
}
