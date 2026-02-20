package config

import (
	"os"
	"strings"
	"time"
)

// Config holds all application configuration values.
type Config struct {
	// HTTP server
	HTTPAddr string

	// Redis
	RedisMode          string // "standalone" (default), "sentinel", "cluster"
	RedisAddr          string // standalone: host:port
	RedisPassword      string
	RedisDB            int
	RedisMasterName    string   // sentinel: master name
	RedisSentinelAddrs []string // sentinel: sentinel node addresses
	RedisClusterAddrs  []string // cluster: cluster node addresses

	// Kafka
	KafkaBrokers []string

	// Worker
	PollInterval time.Duration
	BatchSize    int

	// Application
	Environment string
	LogLevel    string
}

// New creates a Config populated from environment variables with sensible defaults.
func New() *Config {
	cfg := &Config{
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
		RedisMode:     getEnv("REDIS_MODE", "standalone"),
		RedisAddr:     getEnv("REDIS_HOST", "localhost") + ":" + getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
		KafkaBrokers:  strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		PollInterval:  1 * time.Second,
		BatchSize:     10,
		Environment:   getEnv("ENVIRONMENT", "local"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
	}

	if v := getEnv("REDIS_MASTER_NAME", ""); v != "" {
		cfg.RedisMasterName = v
	}
	if v := getEnv("REDIS_SENTINEL_ADDRS", ""); v != "" {
		cfg.RedisSentinelAddrs = strings.Split(v, ",")
	}
	if v := getEnv("REDIS_CLUSTER_ADDRS", ""); v != "" {
		cfg.RedisClusterAddrs = strings.Split(v, ",")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
