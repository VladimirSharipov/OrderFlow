package main

import (
	"testing"
	"time"

	"wbtest/internal/config"
)

func TestNewApp(t *testing.T) {
	// Создаем тестовую конфигурацию
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test_user",
			Password: "test_pass",
			Database: "test_db",
			SSLMode:  "disable",
		},
		Cache: config.CacheConfig{
			MaxSize:         100,
			TTLMinutes:      60,
			CleanupInterval: 5 * time.Minute,
		},
		HTTP: config.HTTPConfig{
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		App: config.AppConfig{
			DatabaseLoadTimeout: 10 * time.Second,
		},
		Retry: config.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
		},
		DLQ: config.DLQConfig{
			Enabled:    true,
			Topic:      "test-dlq",
			MaxRetries: 3,
		},
		Kafka: config.KafkaConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "test-topic",
			GroupID: "test-group",
		},
	}

	// Тестируем создание приложения с неверными настройками БД
	// (приложение создается, но БД недоступна при загрузке кеша)
	app, err := NewApp(cfg)
	if err != nil {
		t.Errorf("Unexpected error when creating app: %v", err)
	}
	if app != nil {
		// Закрываем приложение
		app.Close()
	}
}

func TestApp_Close(t *testing.T) {
	// Создаем пустое приложение для тестирования Close
	app := &App{}

	// Тестируем закрытие пустого приложения
	err := app.Close()
	if err != nil {
		t.Errorf("Expected no error when closing empty app, got: %v", err)
	}
}
