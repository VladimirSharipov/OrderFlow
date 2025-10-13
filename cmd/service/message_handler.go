package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"wbtest/internal/model"
)

// MessageHandler обрабатывает Kafka сообщения
type MessageHandler struct {
	app *App
}

// NewMessageHandler создает обработчик
func NewMessageHandler(app *App) *MessageHandler {
	return &MessageHandler{app: app}
}

// HandleMessage обрабатывает сообщение
func (h *MessageHandler) HandleMessage(ctx context.Context, msg []byte) error {
	log.Printf("[KAFKA] Received message: %s", string(msg))

	// Обрабатываем сообщение с retry логикой
	processMessage := func() error {
		var order model.Order
		if err := json.Unmarshal(msg, &order); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}

		// Валидируем заказ
		if err := h.app.Validator.Validate(&order); err != nil {
			return fmt.Errorf("order validation failed: %w", err)
		}

		log.Printf("[KAFKA] Parsed and validated order: %s", order.OrderUID)

		// Сохраняем в БД
		if err := h.app.DB.SaveOrder(ctx, &order); err != nil {
			return fmt.Errorf("failed to save order %s: %w", order.OrderUID, err)
		}

		// Обновляем кеш
		h.app.Cache.Set(&order)
		log.Printf("[KAFKA] Order %s saved and cached", order.OrderUID)
		return nil
	}

	// Выполняем обработку с retry
	if err := h.app.RetryService.ExecuteWithRetry(processMessage); err != nil {
		log.Printf("[KAFKA] Failed to process message after retries: %v", err)

		// Отправляем в DLQ
		if dlqErr := h.app.DLQService.SendToDLQ(msg, err.Error()); dlqErr != nil {
			log.Printf("[KAFKA] Failed to send message to DLQ: %v", dlqErr)
		}
		return err
	}

	return nil
}

// StartKafkaConsumer запускает consumer
func (h *MessageHandler) StartKafkaConsumer(ctx context.Context) error {
	log.Println("Starting Kafka consumer...")

	return h.app.Consumer.ReadMessages(ctx, func(msg []byte) {
		if err := h.HandleMessage(ctx, msg); err != nil {
			log.Printf("[KAFKA] Error handling message: %v", err)
		}
	})
}
