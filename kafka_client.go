package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaClient struct {
	writer *kafka.Writer
}

func NewKafkaClient() (*KafkaClient, error) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      strings.Split(brokers, ","),
		BatchTimeout: time.Millisecond * 100,
	})

	return &KafkaClient{
		writer: writer,
	}, nil
}

func (k *KafkaClient) PushMessage(ctx context.Context, topic string, key, value []byte) error {
	return k.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	})
}

func (k *KafkaClient) Close() error {
	return k.writer.Close()
}
