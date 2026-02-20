package redisstore

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/ruudy-sib/rebound/internal/config"
)

// NewClient creates a Redis client from the application configuration
// and verifies the connection with a ping.
//
// Supported modes (config.RedisMode):
//   - "standalone" (default): single Redis instance via RedisAddr
//   - "sentinel": high-availability via RedisSentinelAddrs + RedisMasterName
//   - "cluster": Redis Cluster via RedisClusterAddrs
func NewClient(ctx context.Context, cfg *config.Config, logger *zap.Logger) (redis.UniversalClient, error) {
	var client redis.UniversalClient

	switch cfg.RedisMode {
	case "sentinel":
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.RedisMasterName,
			SentinelAddrs: cfg.RedisSentinelAddrs,
			Password:      cfg.RedisPassword,
			DB:            cfg.RedisDB,
		})
		logger.Info("connecting to redis via sentinel",
			zap.String("master", cfg.RedisMasterName),
			zap.Strings("sentinels", cfg.RedisSentinelAddrs),
		)

	case "cluster":
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.RedisClusterAddrs,
			Password: cfg.RedisPassword,
		})
		logger.Info("connecting to redis cluster",
			zap.Strings("addrs", cfg.RedisClusterAddrs),
		)

	default: // "standalone"
		client = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})
		logger.Info("connecting to redis standalone",
			zap.String("addr", cfg.RedisAddr),
		)
	}

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return client, nil
}
