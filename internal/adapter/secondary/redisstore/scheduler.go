package redisstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"kafkaretry/internal/domain"
	"kafkaretry/internal/domain/entity"
	"kafkaretry/internal/port/secondary"
)

// taskDTO is the Redis-specific representation of a task.
// It translates between domain entities and JSON stored in Redis.
type taskDTO struct {
	ID              string `json:"id"`
	Attempt         int    `json:"attempt"`
	Source          string `json:"source"`
	Destination     destDTO `json:"destination"`
	DeadDestination destDTO `json:"dead_destination"`
	MaxRetries      int    `json:"max_retries"`
	BaseDelay       int    `json:"base_delay"`
	ClientID        string `json:"client_id"`
	IsPriority      bool   `json:"is_priority"`
	MessageData     string `json:"message_data"`
	DestinationType string `json:"destination_type"`
}

type destDTO struct {
	Host  string `json:"host"`
	Port  string `json:"port"`
	Topic string `json:"topic"`
}

func toDTO(task *entity.Task) taskDTO {
	return taskDTO{
		ID:      task.ID,
		Attempt: task.Attempt,
		Source:  task.Source,
		Destination: destDTO{
			Host:  task.Destination.Host,
			Port:  task.Destination.Port,
			Topic: task.Destination.Topic,
		},
		DeadDestination: destDTO{
			Host:  task.DeadDestination.Host,
			Port:  task.DeadDestination.Port,
			Topic: task.DeadDestination.Topic,
		},
		MaxRetries:      task.MaxRetries,
		BaseDelay:       task.BaseDelay,
		ClientID:        task.ClientID,
		IsPriority:      task.IsPriority,
		MessageData:     task.MessageData,
		DestinationType: string(task.DestinationType),
	}
}

func toEntity(dto taskDTO) *entity.Task {
	return &entity.Task{
		ID:      dto.ID,
		Attempt: dto.Attempt,
		Source:  dto.Source,
		Destination: entity.Destination{
			Host:  dto.Destination.Host,
			Port:  dto.Destination.Port,
			Topic: dto.Destination.Topic,
		},
		DeadDestination: entity.Destination{
			Host:  dto.DeadDestination.Host,
			Port:  dto.DeadDestination.Port,
			Topic: dto.DeadDestination.Topic,
		},
		MaxRetries:      dto.MaxRetries,
		BaseDelay:       dto.BaseDelay,
		ClientID:        dto.ClientID,
		IsPriority:      dto.IsPriority,
		MessageData:     dto.MessageData,
		DestinationType: entity.DestinationType(dto.DestinationType),
	}
}

// Scheduler implements secondary.TaskScheduler using a Redis sorted set.
// Tasks are scored by their scheduled execution time (Unix timestamp).
type Scheduler struct {
	client *redis.Client
	key    string
	logger *zap.Logger
}

// NewScheduler creates a Redis-backed task scheduler.
func NewScheduler(client *redis.Client, logger *zap.Logger) secondary.TaskScheduler {
	return &Scheduler{
		client: client,
		key:    domain.RedisRetryKey,
		logger: logger.Named("redis-scheduler"),
	}
}

// Schedule adds a task to the sorted set with score = now + delay.
func (s *Scheduler) Schedule(ctx context.Context, task *entity.Task, delay time.Duration) error {
	dto := toDTO(task)
	data, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("marshaling task: %w", err)
	}

	score := float64(time.Now().Add(delay).Unix())
	if err := s.client.ZAdd(ctx, s.key, redis.Z{
		Score:  score,
		Member: data,
	}).Err(); err != nil {
		return fmt.Errorf("scheduling task in redis: %w", err)
	}

	return nil
}

// FetchDue retrieves tasks whose score (scheduled time) is <= now,
// removes them from the sorted set atomically, and returns them.
func (s *Scheduler) FetchDue(ctx context.Context, limit int) ([]*entity.Task, error) {
	now := fmt.Sprintf("%f", float64(time.Now().Unix()))

	results, err := s.client.ZRangeByScoreWithScores(ctx, s.key, &redis.ZRangeBy{
		Min:    "0",
		Max:    now,
		Offset: 0,
		Count:  int64(limit),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("fetching due tasks from redis: %w", err)
	}

	tasks := make([]*entity.Task, 0, len(results))
	for _, z := range results {
		member, ok := z.Member.(string)
		if !ok {
			s.logger.Warn("unexpected member type in sorted set")
			continue
		}

		// Remove the task from the queue before processing.
		if err := s.client.ZRem(ctx, s.key, z.Member).Err(); err != nil {
			s.logger.Error("failed to remove task from queue",
				zap.Error(err),
				zap.String("member", member),
			)
			continue
		}

		var dto taskDTO
		if err := json.Unmarshal([]byte(member), &dto); err != nil {
			s.logger.Warn("invalid task data in redis",
				zap.Error(err),
				zap.String("raw", member),
			)
			continue
		}

		tasks = append(tasks, toEntity(dto))
	}

	return tasks, nil
}

// Remove deletes a specific member from the sorted set.
func (s *Scheduler) Remove(ctx context.Context, rawMember string) error {
	return s.client.ZRem(ctx, s.key, rawMember).Err()
}
