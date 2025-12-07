package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaClient struct {
	writer *kafka.Writer
}

func NewKafkaClient(brokers string) (*KafkaClient, error) {
	// brokers := os.Getenv("KAFKA_BROKERS")
	// if brokers == "" {
	brokers = "kafka:9092"
	// }
	fmt.Println("Connecting to Kafka brokers at:", brokers)
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
