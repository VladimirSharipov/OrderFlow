package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wbtest/internal/config"
)

func main() {
	log.Println("Starting order service...")

	// Загружаем конфигурацию
	cfg := config.Load()
	log.Printf("Configuration loaded: DB=%s:%d, Kafka=%v, HTTP=%d",
		cfg.Database.Host, cfg.Database.Port, cfg.Kafka.Brokers, cfg.HTTP.Port)

	// Создаем приложение
	app, err := NewApp(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Printf("Error closing application: %v", err)
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
			log.Printf("Kafka consumer stopped with error: %v", err)
		}
	}()

	// Запускаем обработчик DLQ
	go func() {
		if err := app.DLQService.ProcessDLQ(); err != nil {
			log.Printf("DLQ processor error: %v", err)
		}
	}()

	// Запускаем HTTP сервер
	go func() {
		log.Printf("Starting HTTP server on :%d", cfg.HTTP.Port)
		if err := app.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	log.Println("Order service started successfully")
	log.Println("Waiting for shutdown signal...")

	// Ждем сигнала завершения
	<-sigChan
	log.Println("Received shutdown signal, starting graceful shutdown...")

	// Graceful shutdown с таймаутом из конфигурации
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.App.GracefulShutdownTimeout)
	defer shutdownCancel()

	// Останавливаем HTTP сервер
	if err := app.HTTPServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Println("HTTP server stopped gracefully")
	}

	// Отменяем контекст для остановки Kafka consumer
	cancel()

	// Ждем завершения всех горутин
	select {
	case <-shutdownCtx.Done():
		log.Println("Graceful shutdown timeout exceeded")
	case <-time.After(cfg.App.ShutdownWaitTimeout):
		log.Println("Graceful shutdown completed")
	}

	log.Println("Order service stopped")
}
