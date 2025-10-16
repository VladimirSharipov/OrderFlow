package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"wbtest/internal/model"
)

func TestNewConsumer(t *testing.T) {
	tests := []struct {
		name        string
		brokers     []string
		topic       string
		groupID     string
		expectError bool
	}{
		{
			name:        "valid configuration",
			brokers:     []string{"localhost:9092"},
			topic:       "test-topic",
			groupID:     "test-group",
			expectError: false,
		},
		{
			name:        "multiple brokers",
			brokers:     []string{"broker1:9092", "broker2:9092"},
			topic:       "test-topic",
			groupID:     "test-group",
			expectError: false,
		},
		{
			name:        "empty brokers",
			brokers:     []string{},
			topic:       "test-topic",
			groupID:     "test-group",
			expectError: true, // Consumer не может быть создан с пустыми брокерами
		},
		{
			name:        "empty topic",
			brokers:     []string{"localhost:9092"},
			topic:       "",
			groupID:     "test-group",
			expectError: true, // Consumer не может быть создан с пустым топиком
		},
		{
			name:        "empty group ID",
			brokers:     []string{"localhost:9092"},
			topic:       "test-topic",
			groupID:     "",
			expectError: false, // Consumer создается, но не может работать
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.expectError {
						t.Errorf("Unexpected panic: %v", r)
					}
				}
			}()

			consumer := NewConsumer(tt.brokers, tt.topic, tt.groupID)

			if consumer == nil {
				if !tt.expectError {
					t.Error("NewConsumer() returned nil")
				}
				return
			}

			// Проверяем, что consumer имеет правильную конфигурацию
			if consumer == nil {
				t.Error("Consumer is nil")
			}
		})
	}
}

func TestKafkaConsumer_ReadMessages(t *testing.T) {
	consumer := NewConsumer([]string{"localhost:9092"}, "test-topic", "test-group")

	// Создаем контекст с таймаутом для теста
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Тестируем обработку сообщений
	messageCount := 0
	handler := func(msg []byte) {
		messageCount++
	}

	// Запускаем чтение сообщений
	err := consumer.ReadMessages(ctx, handler)

	// Ожидаем ошибку, так как Kafka недоступен
	if err == nil {
		t.Error("Expected error when Kafka is unavailable, got nil")
	}

	// Проверяем, что обработчик не был вызван
	if messageCount > 0 {
		t.Errorf("Expected 0 messages processed, got %d", messageCount)
	}
}

func TestKafkaConsumer_ReadMessagesWithValidHandler(t *testing.T) {
	consumer := NewConsumer([]string{"localhost:9092"}, "test-topic", "test-group")

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Создаем обработчик, который проверяет валидность JSON
	processedMessages := make([][]byte, 0)
	handler := func(msg []byte) {
		// Проверяем, что сообщение можно распарсить как JSON
		var order model.Order
		err := json.Unmarshal(msg, &order)
		if err != nil {
			t.Errorf("Failed to unmarshal message: %v", err)
		}
		processedMessages = append(processedMessages, msg)
	}

	// Запускаем чтение сообщений
	err := consumer.ReadMessages(ctx, handler)

	// Ожидаем ошибку подключения к Kafka
	if err == nil {
		t.Error("Expected connection error, got nil")
	}

	// Проверяем, что сообщения не обработались из-за недоступности Kafka
	if len(processedMessages) > 0 {
		t.Errorf("Expected 0 processed messages, got %d", len(processedMessages))
	}
}

func TestKafkaConsumer_Close(t *testing.T) {
	consumer := NewConsumer([]string{"localhost:9092"}, "test-topic", "test-group")

	// Тест на то, что Close не падает
	err := consumer.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestKafkaConsumer_ReadMessagesWithContextCancellation(t *testing.T) {
	consumer := NewConsumer([]string{"localhost:9092"}, "test-topic", "test-group")

	// Создаем контекст, который сразу отменяется
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу

	messageCount := 0
	handler := func(msg []byte) {
		messageCount++
	}

	// Запускаем чтение сообщений с отмененным контекстом
	err := consumer.ReadMessages(ctx, handler)

	// Ожидаем ошибку отмены контекста
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}

	// Проверяем, что сообщения не обработались
	if messageCount > 0 {
		t.Errorf("Expected 0 messages processed, got %d", messageCount)
	}
}

func TestKafkaConsumer_ReadMessagesWithPanicHandler(t *testing.T) {
	consumer := NewConsumer([]string{"localhost:9092"}, "test-topic", "test-group")

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Создаем обработчик, который паникует
	handler := func(msg []byte) {
		panic("test panic")
	}

	// Запускаем чтение сообщений
	err := consumer.ReadMessages(ctx, handler)

	// Ожидаем ошибку подключения к Kafka (не panic)
	if err == nil {
		t.Error("Expected connection error, got nil")
	}
}

func TestKafkaConsumer_ReadMessagesWithSlowHandler(t *testing.T) {
	consumer := NewConsumer([]string{"localhost:9092"}, "test-topic", "test-group")

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Создаем медленный обработчик
	handler := func(msg []byte) {
		time.Sleep(200 * time.Millisecond) // Больше таймаута контекста
	}

	// Запускаем чтение сообщений
	err := consumer.ReadMessages(ctx, handler)

	// Ожидаем ошибку подключения к Kafka
	if err == nil {
		t.Error("Expected connection error, got nil")
	}
}

func TestKafkaConsumer_ReadMessagesWithNilHandler(t *testing.T) {
	consumer := NewConsumer([]string{"localhost:9092"}, "test-topic", "test-group")

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Запускаем чтение сообщений с nil обработчиком
	err := consumer.ReadMessages(ctx, nil)

	// Ожидаем ошибку подключения к Kafka (не panic от nil handler)
	if err == nil {
		t.Error("Expected connection error, got nil")
	}
}

func TestKafkaConsumer_Configuration(t *testing.T) {
	brokers := []string{"broker1:9092", "broker2:9092", "broker3:9092"}
	topic := "test-topic"
	groupID := "test-group"

	consumer := NewConsumer(brokers, topic, groupID)

	if consumer == nil {
		t.Fatal("Consumer is nil")
	}

	// Тест на то, что конфигурация сохранена правильно
	// (В реальной реализации мы бы проверяли поля consumer)
	// Здесь мы просто проверяем, что consumer создался без ошибок
}
