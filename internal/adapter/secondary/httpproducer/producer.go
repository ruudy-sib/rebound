package httpproducer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"kafkaretry/internal/config"
	"kafkaretry/internal/domain/entity"
	"kafkaretry/internal/port/secondary"
)

// Producer implements secondary.MessageProducer using HTTP POST requests.
// It sends the message payload to the destination URL via HTTP.
type Producer struct {
	client *http.Client
	logger *zap.Logger
}

// NewProducer creates an HTTP producer with configurable timeout.
func NewProducer(cfg *config.Config, logger *zap.Logger) secondary.MessageProducer {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	logger.Info("http producer initialized",
		zap.Duration("timeout", client.Timeout),
	)

	return &Producer{
		client: client,
		logger: logger.Named("http-producer"),
	}
}

// Produce sends a message via HTTP POST to the destination URL.
// The key is sent as X-Message-Key header, and value is sent as the request body.
func (p *Producer) Produce(ctx context.Context, destination entity.Destination, key, value []byte) error {
	if destination.URL == "" {
		return fmt.Errorf("destination URL is required for HTTP delivery")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, destination.URL, bytes.NewReader(value))
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Message-Key", string(key))
	req.Header.Set("User-Agent", "kafka-retry-service/1.0")

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("executing http request to %q: %w", destination.URL, err)
	}
	defer resp.Body.Close()

	// Read response body for logging
	body, _ := io.ReadAll(resp.Body)

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http request failed with status %d: %s", resp.StatusCode, string(body))
	}

	p.logger.Debug("message produced via http",
		zap.String("url", destination.URL),
		zap.Int("status_code", resp.StatusCode),
		zap.Int("value_size", len(value)),
	)

	return nil
}

// Close releases any resources held by the producer.
func (p *Producer) Close() error {
	if p.client != nil {
		p.client.CloseIdleConnections()
	}
	return nil
}
