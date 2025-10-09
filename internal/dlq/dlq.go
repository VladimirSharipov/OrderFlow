package dlq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"wbtest/internal/config"
	"wbtest/internal/interfaces"

	"github.com/segmentio/kafka-go"
)

type DLQMessage struct {
	OriginalMessage []byte    `json:"original_message"`
	Reason          string    `json:"reason"`
	Timestamp       time.Time `json:"timestamp"`
	RetryCount      int       `json:"retry_count"`
}

// MarshalJSON implements json.Marshaler interface
func (d *DLQMessage) MarshalJSON() ([]byte, error) {
	type Alias DLQMessage
	return json.Marshal((*Alias)(d))
}

// UnmarshalJSON implements json.Unmarshaler interface
func (d *DLQMessage) UnmarshalJSON(data []byte) error {
	type Alias DLQMessage
	return json.Unmarshal(data, (*Alias)(d))
}

type DLQService struct {
	config *config.DLQConfig
	writer *kafka.Writer
	reader *kafka.Reader
}

func NewDLQService(cfg *config.DLQConfig, brokers []string) interfaces.DLQService {
	if !cfg.Enabled {
		return &NoOpDLQService{}
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    cfg.Topic,
		Balancer: &kafka.LeastBytes{},
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    cfg.Topic,
		GroupID:  "dlq-processor",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	return &DLQService{
		config: cfg,
		writer: writer,
		reader: reader,
	}
}

func (d *DLQService) SendToDLQ(message []byte, reason string) error {
	dlqMessage := DLQMessage{
		OriginalMessage: message,
		Reason:          reason,
		Timestamp:       time.Now(),
		RetryCount:      0,
	}

	messageBytes, err := json.Marshal(dlqMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ message: %w", err)
	}

	err = d.writer.WriteMessages(context.Background(), kafka.Message{
		Value: messageBytes,
	})
	if err != nil {
		return fmt.Errorf("failed to send message to DLQ: %w", err)
	}

	log.Printf("Message sent to DLQ: reason=%s, size=%d bytes", reason, len(message))
	return nil
}

func (d *DLQService) ProcessDLQ() error {
	if !d.config.Enabled {
		return nil
	}

	log.Println("Starting DLQ processing...")

	for {
		message, err := d.reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading from DLQ: %v", err)
			continue
		}

		var dlqMessage DLQMessage
		if err := json.Unmarshal(message.Value, &dlqMessage); err != nil {
			log.Printf("Failed to unmarshal DLQ message: %v", err)
			continue
		}

		// Увеличиваем счетчик попыток
		dlqMessage.RetryCount++

		// Если превышено максимальное количество попыток, логируем и пропускаем
		if dlqMessage.RetryCount > d.config.MaxRetries {
			log.Printf("Message exceeded max retries (%d), dropping: %s",
				d.config.MaxRetries, dlqMessage.Reason)
			continue
		}

		// Попытка повторной обработки
		log.Printf("Retrying DLQ message (attempt %d/%d): %s",
			dlqMessage.RetryCount, d.config.MaxRetries, dlqMessage.Reason)

		// Здесь можно добавить логику повторной обработки сообщения
		// Например, отправить обратно в основной топик или обработать по-другому
		if err := d.retryMessage(&dlqMessage); err != nil {
			log.Printf("Failed to retry message: %v", err)
			// Можно отправить в другой DLQ или обработать по-другому
		}
	}
}

func (d *DLQService) retryMessage(dlqMessage *DLQMessage) error {
	// Здесь можно реализовать логику повторной обработки
	// Например, отправить обратно в основной топик
	log.Printf("Retrying message: %s", string(dlqMessage.OriginalMessage))
	return nil
}

func (d *DLQService) Close() error {
	if d.writer != nil {
		if err := d.writer.Close(); err != nil {
			log.Printf("Error closing DLQ writer: %v", err)
		}
	}
	if d.reader != nil {
		if err := d.reader.Close(); err != nil {
			log.Printf("Error closing DLQ reader: %v", err)
		}
	}
	return nil
}

// NoOpDLQService - заглушка для случая, когда DLQ отключен
type NoOpDLQService struct{}

func (n *NoOpDLQService) SendToDLQ(message []byte, reason string) error {
	log.Printf("DLQ disabled, message dropped: %s", reason)
	return nil
}

func (n *NoOpDLQService) ProcessDLQ() error {
	return nil
}

func (n *NoOpDLQService) Close() error {
	return nil
}
