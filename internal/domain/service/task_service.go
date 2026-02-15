package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"kafkaretry/internal/domain"
	"kafkaretry/internal/domain/entity"
	"kafkaretry/internal/port/secondary"
)

// TaskService orchestrates task scheduling, processing, and retry logic.
type TaskService struct {
	scheduler secondary.TaskScheduler
	producer  secondary.MessageProducer
	logger    *zap.Logger
}

// NewTaskService creates a TaskService with its dependencies injected.
func NewTaskService(
	scheduler secondary.TaskScheduler,
	producer secondary.MessageProducer,
	logger *zap.Logger,
) *TaskService {
	return &TaskService{
		scheduler: scheduler,
		producer:  producer,
		logger:    logger.Named("task-service"),
	}
}

// CreateTask validates and schedules a new task for immediate processing.
func (s *TaskService) CreateTask(ctx context.Context, task *entity.Task) error {
	if err := s.validateTask(task); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidTask, err)
	}

	task.Attempt = 0

	if err := s.scheduler.Schedule(ctx, task, 0); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrScheduleFailed, err)
	}

	s.logger.Info("task scheduled",
		zap.String("task_id", task.ID),
		zap.String("source", task.Source),
		zap.String("destination_type", string(task.DestinationType)),
	)

	return nil
}

// ProcessDueTasks fetches due tasks and processes each one.
// Failed tasks are rescheduled with exponential backoff.
// Tasks that exceed max retries are sent to the dead-letter destination.
func (s *TaskService) ProcessDueTasks(ctx context.Context) error {
	tasks, err := s.scheduler.FetchDue(ctx, domain.DefaultBatchSize)
	if err != nil {
		return fmt.Errorf("fetching due tasks: %w", err)
	}

	for _, task := range tasks {
		s.processTask(ctx, task)
	}

	return nil
}

func (s *TaskService) processTask(ctx context.Context, task *entity.Task) {
	logger := s.logger.With(
		zap.String("task_id", task.ID),
		zap.Int("attempt", task.Attempt),
	)

	logger.Info("processing task")

	if err := s.deliver(ctx, task); err != nil {
		logger.Warn("delivery failed", zap.Error(err))
		s.handleFailure(ctx, task, logger)
		return
	}

	logger.Info("task completed successfully")
}

func (s *TaskService) deliver(ctx context.Context, task *entity.Task) error {
	switch task.DestinationType {
	case entity.DestinationTypeKafka:
		key := []byte(fmt.Sprintf("%s|%d", task.ID, task.Attempt))
		value := []byte(task.MessageData)
		return s.producer.Produce(ctx, task.Destination, key, value)
	default:
		return fmt.Errorf("%w: unsupported destination type %q", domain.ErrDeliveryFailed, task.DestinationType)
	}
}

func (s *TaskService) handleFailure(ctx context.Context, task *entity.Task, logger *zap.Logger) {
	task.IncrementAttempt()

	if task.ShouldSendToDeadDestination() {
		logger.Error("max retries exceeded, sending to dead-letter destination",
			zap.Int("max_retries", task.MaxRetries),
			zap.Int("attempts", task.Attempt),
		)
		s.sendToDeadLetter(ctx, task, logger)
		return
	}

	delay := task.NextRetryDelay()
	logger.Info("scheduling retry",
		zap.Duration("delay", delay),
		zap.Int("next_attempt", task.Attempt),
	)

	if err := s.scheduler.Schedule(ctx, task, delay); err != nil {
		logger.Error("failed to reschedule task", zap.Error(err))
	}
}

func (s *TaskService) sendToDeadLetter(ctx context.Context, task *entity.Task, logger *zap.Logger) {
	if task.DeadDestination.Topic == "" {
		logger.Warn("no dead-letter destination configured, dropping task")
		return
	}

	key := []byte(fmt.Sprintf("%s|dead|%d", task.ID, task.Attempt))
	value := []byte(task.MessageData)

	if err := s.producer.Produce(ctx, task.DeadDestination, key, value); err != nil {
		logger.Error("failed to send to dead-letter destination", zap.Error(err))
	}
}

func (s *TaskService) validateTask(task *entity.Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if task.Source == "" {
		return fmt.Errorf("source is required")
	}
	if task.Destination.Topic == "" {
		return fmt.Errorf("destination topic is required")
	}
	if task.DestinationType == "" {
		return fmt.Errorf("destination type is required")
	}
	if task.MaxRetries < 0 || task.MaxRetries > domain.MaxRetryLimit {
		return fmt.Errorf("max_retries must be between 0 and %d", domain.MaxRetryLimit)
	}
	if task.BaseDelay < domain.MinBaseDelay || task.BaseDelay > domain.MaxBaseDelay {
		return fmt.Errorf("base_delay must be between %d and %d", domain.MinBaseDelay, domain.MaxBaseDelay)
	}
	return nil
}
