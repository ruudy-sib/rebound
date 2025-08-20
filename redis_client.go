package main

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() *redis.Client {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	return redis.NewClient(&redis.Options{
		Addr: redisHost + ":6379",
	})
}

// Example: Use context.Background() for simple usage
func PingRedis(client *redis.Client) error {
	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	return err
}

// ...existing code...

// GetAllKeys returns all keys matching the given pattern (use "*" for all keys)
func GetAllKeys(client *redis.Client, pattern string) ([]string, error) {
	ctx := context.Background()
	keys, err := client.Keys(ctx, pattern).Result()
	return keys, err
}
