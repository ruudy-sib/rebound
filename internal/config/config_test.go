package config

import (
	"os"
	"testing"
	"time"
)

func TestNew_defaults(t *testing.T) {
	// Clear environment to test defaults
	envKeys := []string{"HTTP_ADDR", "REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "KAFKA_BROKERS", "ENVIRONMENT", "LOG_LEVEL"}
	for _, key := range envKeys {
		os.Unsetenv(key)
	}

	cfg := New()

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"HTTPAddr", cfg.HTTPAddr, ":8080"},
		{"RedisAddr", cfg.RedisAddr, "localhost:6379"},
		{"RedisPassword", cfg.RedisPassword, ""},
		{"RedisDB", cfg.RedisDB, 0},
		{"Environment", cfg.Environment, "local"},
		{"LogLevel", cfg.LogLevel, "info"},
		{"PollInterval", cfg.PollInterval, 1 * time.Second},
		{"BatchSize", cfg.BatchSize, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %v, want %v", tt.got, tt.want)
			}
		})
	}

	// Check KafkaBrokers slice
	if len(cfg.KafkaBrokers) != 1 || cfg.KafkaBrokers[0] != "localhost:9092" {
		t.Fatalf("expected [localhost:9092], got %v", cfg.KafkaBrokers)
	}
}

func TestNew_fromEnvironment(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("REDIS_HOST", "redis-host")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_PASSWORD", "secret")
	t.Setenv("KAFKA_BROKERS", "broker1:9092,broker2:9092")
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("LOG_LEVEL", "debug")

	cfg := New()

	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("expected :9090, got %s", cfg.HTTPAddr)
	}
	if cfg.RedisAddr != "redis-host:6380" {
		t.Fatalf("expected redis-host:6380, got %s", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "secret" {
		t.Fatalf("expected secret, got %s", cfg.RedisPassword)
	}
	if cfg.Environment != "production" {
		t.Fatalf("expected production, got %s", cfg.Environment)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected debug, got %s", cfg.LogLevel)
	}
	if len(cfg.KafkaBrokers) != 2 {
		t.Fatalf("expected 2 brokers, got %d", len(cfg.KafkaBrokers))
	}
	if cfg.KafkaBrokers[0] != "broker1:9092" || cfg.KafkaBrokers[1] != "broker2:9092" {
		t.Fatalf("unexpected brokers: %v", cfg.KafkaBrokers)
	}
}
