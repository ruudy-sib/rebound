package service

import (
	"context"
	"time"

	"rebound/internal/domain/entity"
)

// mockScheduler implements secondary.TaskScheduler for testing.
type mockScheduler struct {
	scheduleFunc func(ctx context.Context, task *entity.Task, delay time.Duration) error
	fetchDueFunc func(ctx context.Context, limit int) ([]*entity.Task, error)
	removeFunc   func(ctx context.Context, rawMember string) error

	scheduledTasks []scheduledCall
}

type scheduledCall struct {
	Task  *entity.Task
	Delay time.Duration
}

func (m *mockScheduler) Schedule(ctx context.Context, task *entity.Task, delay time.Duration) error {
	m.scheduledTasks = append(m.scheduledTasks, scheduledCall{Task: task, Delay: delay})
	if m.scheduleFunc != nil {
		return m.scheduleFunc(ctx, task, delay)
	}
	return nil
}

func (m *mockScheduler) FetchDue(ctx context.Context, limit int) ([]*entity.Task, error) {
	if m.fetchDueFunc != nil {
		return m.fetchDueFunc(ctx, limit)
	}
	return nil, nil
}

func (m *mockScheduler) Remove(ctx context.Context, rawMember string) error {
	if m.removeFunc != nil {
		return m.removeFunc(ctx, rawMember)
	}
	return nil
}

// mockProducer implements secondary.MessageProducer for testing.
type mockProducer struct {
	produceFunc func(ctx context.Context, destination entity.Destination, key, value []byte) error
	closeFunc   func() error

	produceCalls []produceCall
}

type produceCall struct {
	Destination entity.Destination
	Key         []byte
	Value       []byte
	Err         error
}

func (m *mockProducer) Produce(ctx context.Context, destination entity.Destination, key, value []byte) error {
	var err error
	if m.produceFunc != nil {
		err = m.produceFunc(ctx, destination, key, value)
	}
	m.produceCalls = append(m.produceCalls, produceCall{
		Destination: destination,
		Key:         key,
		Value:       value,
		Err:         err,
	})
	return err
}

func (m *mockProducer) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// successfulProduceCalls returns only the calls that did not return an error.
func (m *mockProducer) successfulProduceCalls() []produceCall {
	var result []produceCall
	for _, c := range m.produceCalls {
		if c.Err == nil {
			result = append(result, c)
		}
	}
	return result
}

// testTask returns a standard test task fixture.
func testTask() *entity.Task {
	return &entity.Task{
		ID:     "task-1",
		Source: "test-app",
		Destination: entity.Destination{
			Host:  "localhost",
			Port:  "9092",
			Topic: "my-topic",
		},
		DeadDestination: entity.Destination{
			Host:  "localhost",
			Port:  "9092",
			Topic: "dead-topic",
		},
		MaxRetries:      3,
		BaseDelay:       2,
		ClientID:        "client-1",
		IsPriority:      false,
		MessageData:     "test message data",
		DestinationType: entity.DestinationTypeKafka,
	}
}
