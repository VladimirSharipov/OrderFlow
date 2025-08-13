package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

// Consumer это Kafka consumer для чтения сообщений
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer создает новый Kafka consumer
func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	return &Consumer{
		reader: reader,
	}
}

// Close закрывает consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// ReadMessages читает сообщения из Kafka
// для каждого сообщения вызывает функцию handler
func (c *Consumer) ReadMessages(ctx context.Context, handler func([]byte)) error {
	for {
		// читаем сообщение из Kafka
		message, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Ошибка чтения сообщения из Kafka: %v", err)
			return err
		}

		// вызываем обработчик для сообщения
		handler(message.Value)

		// логируем что сообщение обработано
		log.Printf("Сообщение обработано: топик=%s, партиция=%d, смещение=%d",
			message.Topic, message.Partition, message.Offset)
	}
}
