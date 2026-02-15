package main

import (
	"context"
	"net/http"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/dig"
	"go.uber.org/zap"

	httphandler "rebound/internal/adapter/primary/http"
	"rebound/internal/adapter/primary/worker"
	"rebound/internal/adapter/secondary/httpproducer"
	"rebound/internal/adapter/secondary/kafkaproducer"
	"rebound/internal/adapter/secondary/producerfactory"
	"rebound/internal/adapter/secondary/redisstore"
	"rebound/internal/config"
	"rebound/internal/domain/service"
	"rebound/internal/port/primary"
	"rebound/internal/port/secondary"
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
	if err := c.Provide(func(cfg *config.Config, logger *zap.Logger) secondary.MessageProducer {
		return kafkaproducer.NewProducer(cfg, logger)
	}, dig.Name("kafka")); err != nil {
		return nil, err
	}

	// HTTP producer
	if err := c.Provide(func(cfg *config.Config, logger *zap.Logger) secondary.MessageProducer {
		return httpproducer.NewProducer(cfg, logger)
	}, dig.Name("http")); err != nil {
		return nil, err
	}

	// Producer factory (implements secondary.MessageProducer)
	// Routes messages to the appropriate producer based on destination type
	type producerParams struct {
		dig.In
		KafkaProd secondary.MessageProducer `name:"kafka"`
		HTTPProd  secondary.MessageProducer `name:"http"`
		Logger    *zap.Logger
	}

	if err := c.Provide(func(params producerParams) secondary.MessageProducer {
		return producerfactory.NewFactory(params.KafkaProd, params.HTTPProd, params.Logger)
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
