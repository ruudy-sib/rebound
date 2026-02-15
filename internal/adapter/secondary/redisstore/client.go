package redisstore

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"kafkaretry/internal/config"
)

// NewClient creates a Redis client from the application configuration
// and verifies the connection with a ping.
func NewClient(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	logger.Info("connected to redis", zap.String("addr", cfg.RedisAddr))
	return client, nil
}
