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
	RedisAddr     string
	RedisPassword string
	RedisDB       int

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
	return &Config{
		HTTPAddr:     getEnv("HTTP_ADDR", ":8080"),
		RedisAddr:    getEnv("REDIS_HOST", "localhost") + ":" + getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:      0,
		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		PollInterval: 1 * time.Second,
		BatchSize:    10,
		Environment:  getEnv("ENVIRONMENT", "local"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
