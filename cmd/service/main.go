package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"wbtest/internal/cache"
	"wbtest/internal/config"
	"wbtest/internal/db"
	httpapi "wbtest/internal/http"
	"wbtest/internal/interfaces"
	"wbtest/internal/kafka"
	"wbtest/internal/model"
	"wbtest/internal/validator"
)

func main() {
	log.Println("Starting order service...")

	// Загружаем конфигурацию
	cfg := config.Load()
	log.Printf("Configuration loaded: DB=%s:%d, Kafka=%v, HTTP=%d",
		cfg.Database.Host, cfg.Database.Port, cfg.Kafka.Brokers, cfg.HTTP.Port)

	// Подключение к БД
	dbConn, err := db.New(cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		dbConn.Close()
	}()
	log.Println("Connected to DB")

	// Загружаем заказы из БД в кеш
	ctx, cancel := context.WithTimeout(context.Background(), cfg.App.DatabaseLoadTimeout)
	defer cancel()

	log.Println("Loading orders from DB...")
	orders, err := dbConn.LoadAllOrders(ctx)
	if err != nil {
		log.Fatalf("Failed to load orders from database: %v", err)
	}

	// Создаем кеш с настройками из конфигурации
	orderCache := cache.NewOrderCache(cfg.Cache.MaxSize, cfg.Cache.TTL)
	defer func() {
		if cacheImpl, ok := orderCache.(*cache.OrderCache); ok {
			cacheImpl.Stop()
		}
	}()

	orderCache.LoadAll(orders)
	log.Printf("Cache loaded: %d orders", len(orders))

	// Инициализируем валидатор
	orderValidator := validator.NewOrderValidator()

	// Настройка Kafka
	log.Printf("Connecting to Kafka: brokers=%v, topic=%s, groupID=%s",
		cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID)

	consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID)
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Printf("Error closing Kafka consumer: %v", err)
		}
	}()
	log.Println("Kafka consumer initialized")

	// Создаем контекст для graceful shutdown
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Канал для сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем обработчик сообщений
	go func() {
		log.Println("Starting Kafka consumer...")
		err := consumer.ReadMessages(ctx, func(msg []byte) {
			log.Printf("[KAFKA] Received message: %s", string(msg))

			var order model.Order
			if err := json.Unmarshal(msg, &order); err != nil {
				log.Printf("[KAFKA] Failed to parse JSON: %v", err)
				return
			}

			// Валидируем заказ
			if err := orderValidator.Validate(&order); err != nil {
				log.Printf("[KAFKA] Order validation failed: %v", err)
				return
			}

			log.Printf("[KAFKA] Parsed and validated order: %s", order.OrderUID)

			// Сохраняем в БД с обработкой ошибок
			if err := dbConn.SaveOrder(context.Background(), &order); err != nil {
				log.Printf("[KAFKA] Failed to save order %s: %v", order.OrderUID, err)
				return
			}

			// Обновляем кеш
			orderCache.Set(&order)
			log.Printf("[KAFKA] Order %s saved and cached", order.OrderUID)
		})

		if err != nil && ctx.Err() == nil {
			log.Printf("Kafka consumer stopped with error: %v", err)
		}
	}()

	// Запускаем HTTP сервер
	api := httpapi.NewServer(orderCache)
	server := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.HTTP.Port),
		Handler:      api,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	go func() {
		log.Printf("Starting HTTP server on :%d", cfg.HTTP.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
	if err := server.Shutdown(shutdownCtx); err != nil {
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

// Функция для логирования статистики кеша
func logCacheStats(cache interfaces.OrderCache) {
	stats := cache.GetStats()
	log.Printf("Cache stats: size=%d, hits=%d, misses=%d, hit_rate=%.2f%%, evictions=%d, expirations=%d",
		stats.Size, stats.Hits, stats.Misses, stats.HitRate, stats.Evictions, stats.Expirations)
}
