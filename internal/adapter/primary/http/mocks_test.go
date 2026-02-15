package http

import (
	"context"

	"kafkaretry/internal/domain/entity"
	"kafkaretry/internal/port/secondary"
)

// mockTaskService implements primary.TaskService for testing.
type mockTaskService struct {
	createErr      error
	processErr     error
	createCalled   int
	processCalled  int
}

func (m *mockTaskService) CreateTask(_ context.Context, _ *entity.Task) error {
	m.createCalled++
	return m.createErr
}

func (m *mockTaskService) ProcessDueTasks(_ context.Context) error {
	m.processCalled++
	return m.processErr
}

// mockHealthCheck is a test double for health checks.
type mockHealthCheck struct {
	name string
	err  error
}

// healthCheckerAdapter wraps mockHealthCheck to satisfy secondary.HealthChecker.
type healthCheckerAdapter struct {
	check mockHealthCheck
}

func (a healthCheckerAdapter) Name() string {
	return a.check.name
}

func (a healthCheckerAdapter) Check(_ context.Context) error {
	return a.check.err
}

// Compile-time interface assertion
var _ secondary.HealthChecker = healthCheckerAdapter{}

// toHealthCheckers converts a slice of adapters to a slice of the interface.
func toHealthCheckers(adapters []healthCheckerAdapter) []secondary.HealthChecker {
	if len(adapters) == 0 {
		return nil
	}
	result := make([]secondary.HealthChecker, len(adapters))
	for i, a := range adapters {
		result[i] = a
	}
	return result
}

