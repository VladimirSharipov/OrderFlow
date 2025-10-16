package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wbtest/internal/config"
	"wbtest/internal/logger"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Инициализируем логгер
	log := logger.New(cfg.Logger)
	log.Info("Starting order service...")

	log.WithFields(map[string]interface{}{
		"db_host":   cfg.Database.Host,
		"db_port":   cfg.Database.Port,
		"kafka":     cfg.Kafka.Brokers,
		"http_port": cfg.HTTP.Port,
	}).Info("Configuration loaded")

	// Создаем приложение
	app, err := NewApp(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize application")
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.WithError(err).Error("Error closing application")
		}
	}()

	// Создаем контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Канал для сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Создаем обработчик сообщений
	messageHandler := NewMessageHandler(app)

	// Запускаем обработчик сообщений Kafka
	go func() {
		if err := messageHandler.StartKafkaConsumer(ctx); err != nil && ctx.Err() == nil {
			log.WithError(err).Error("Kafka consumer stopped with error")
		}
	}()

	// Запускаем обработчик DLQ
	go func() {
		if err := app.DLQService.ProcessDLQ(); err != nil {
			log.WithError(err).Error("DLQ processor error")
		}
	}()

	// Запускаем HTTP сервер
	go func() {
		log.WithField("port", cfg.HTTP.Port).Info("Starting HTTP server")
		if err := app.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("HTTP server error")
		}
	}()

	log.Info("Order service started successfully")
	log.Info("Waiting for shutdown signal...")

	// Ждем сигнала завершения
	<-sigChan
	log.Info("Received shutdown signal, starting graceful shutdown...")

	// Graceful shutdown с таймаутом из конфигурации
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.App.GracefulShutdownTimeout)
	defer shutdownCancel()

	// Останавливаем HTTP сервер
	if err := app.HTTPServer.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("HTTP server shutdown error")
	} else {
		log.Info("HTTP server stopped gracefully")
	}

	// Отменяем контекст для остановки Kafka consumer
	cancel()

	// Ждем завершения всех горутин
	select {
	case <-shutdownCtx.Done():
		log.Warn("Graceful shutdown timeout exceeded")
	case <-time.After(cfg.App.ShutdownWaitTimeout):
		log.Info("Graceful shutdown completed")
	}

	log.Info("Order service stopped")
}
