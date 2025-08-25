package kafka

import (
	"context"
	"errors"
	"log"

	"github.com/segmentio/kafka-go"
)

// Consumer простой consumer для чтения сообщений из Kafka
type Consumer struct {
	Reader *kafka.Reader
}

// NewConsumer создаёт новый consumer
// brokers адреса брокеров topic топик groupID группа потребителей
func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,
	})
	return &Consumer{Reader: reader}
}

// Close закрывает reader
func (c *Consumer) Close() error {
	return c.Reader.Close()
}

// ReadMessages читает сообщения и вызывает handle для каждого
// Если handle не задан вернём ошибку
func (c *Consumer) ReadMessages(ctx context.Context, handle func([]byte)) error {
	if handle == nil {
		return errors.New("handle is nil")
	}
	for {
		m, err := c.Reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("Kafka read error: %v", err)
			continue
		}

		handle(m.Value)
	}
}
