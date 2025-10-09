//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"wbtest/internal/cache"
	"wbtest/internal/config"
	"wbtest/internal/db"
	"wbtest/internal/dlq"
	"wbtest/internal/model"
	"wbtest/internal/retry"
	"wbtest/internal/validator"
)

// TestOrderServiceIntegration тестирует полный цикл обработки заказа
func TestOrderServiceIntegration(t *testing.T) {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Подключаемся к БД
	dbConn, err := db.New(cfg.DatabaseURL())
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Создаем кеш
	orderCache := cache.NewOrderCache(cfg.Cache.MaxSize, cfg.Cache.TTL)
	defer func() {
		if cacheImpl, ok := orderCache.(*cache.OrderCache); ok {
			cacheImpl.Stop()
		}
	}()

	// Создаем валидатор
	orderValidator := validator.NewOrderValidator()

	// Создаем retry сервис
	retryService := retry.NewRetryService(&cfg.Retry)

	// Создаем DLQ сервис
	dlqService := dlq.NewDLQService(&cfg.DLQ, cfg.Kafka.Brokers)
	defer dlqService.Close()

	// Создаем тестовый заказ
	testOrder := &model.Order{
		OrderUID:    "integration-test-order-123",
		TrackNumber: "WBILMTESTTRACK",
		Entry:       "WBIL",
		Delivery: model.Delivery{
			Name:    "Integration Test User",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Test City",
			Address: "Test Address 123",
			Region:  "Test Region",
			Email:   "test@example.com",
		},
		Payment: model.Payment{
			Transaction:  "integration-test-transaction",
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1000,
			PaymentDT:    int(time.Now().Unix()),
			Bank:         "alpha",
			DeliveryCost: 100,
			GoodsTotal:   900,
			CustomFee:    0,
		},
		Items: []model.Item{
			{
				ChrtID:      123456,
				TrackNumber: "WBILMTESTTRACK",
				Price:       900,
				Rid:         "integration-test-rid",
				Name:        "Test Product",
				Sale:        0,
				Size:        "M",
				TotalPrice:  900,
				NmID:        123456,
				Brand:       "Test Brand",
				Status:      202,
			},
		},
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "integration-test-customer",
		DeliveryService:   "meest",
		ShardKey:          "9",
		SmID:              99,
		DateCreated:       time.Now(),
		OofShard:          "1",
	}

	t.Run("validate_order", func(t *testing.T) {
		err := orderValidator.Validate(testOrder)
		if err != nil {
			t.Errorf("Order validation failed: %v", err)
		}
	})

	t.Run("save_to_database", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := dbConn.SaveOrder(ctx, testOrder)
		if err != nil {
			t.Errorf("Failed to save order to database: %v", err)
		}
	})

	t.Run("cache_operations", func(t *testing.T) {
		// Добавляем заказ в кеш
		orderCache.Set(testOrder)

		// Проверяем, что заказ есть в кеше
		cachedOrder, found := orderCache.Get(testOrder.OrderUID)
		if !found {
			t.Error("Order not found in cache")
		}

		if cachedOrder.OrderUID != testOrder.OrderUID {
			t.Errorf("Cached order UID mismatch: got %s, want %s",
				cachedOrder.OrderUID, testOrder.OrderUID)
		}

		// Проверяем статистику кеша
		stats := orderCache.GetStats()
		if stats.Size == 0 {
			t.Error("Cache size should be greater than 0")
		}
	})

	t.Run("retry_mechanism", func(t *testing.T) {
		attemptCount := 0
		operation := func() error {
			attemptCount++
			if attemptCount < 3 {
				return context.DeadlineExceeded
			}
			return nil
		}

		err := retryService.ExecuteWithRetry(operation)
		if err != nil {
			t.Errorf("Retry mechanism failed: %v", err)
		}

		if attemptCount != 3 {
			t.Errorf("Expected 3 attempts, got %d", attemptCount)
		}
	})

	t.Run("dlq_service", func(t *testing.T) {
		testMessage := []byte("test message for DLQ")
		reason := "integration test"

		err := dlqService.SendToDLQ(testMessage, reason)
		if err != nil {
			t.Errorf("Failed to send message to DLQ: %v", err)
		}
	})

	t.Run("load_orders_from_database", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		orders, err := dbConn.LoadAllOrders(ctx)
		if err != nil {
			t.Errorf("Failed to load orders from database: %v", err)
		}

		// Проверяем, что наш тестовый заказ есть в списке
		found := false
		for _, order := range orders {
			if order.OrderUID == testOrder.OrderUID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Test order not found in loaded orders")
		}
	})
}

// TestOrderProcessingWithRetry тестирует обработку заказа с retry логикой
func TestOrderProcessingWithRetry(t *testing.T) {
	cfg := config.Load()
	retryService := retry.NewRetryService(&cfg.Retry)

	t.Run("successful_operation", func(t *testing.T) {
		operation := func() error {
			return nil
		}

		err := retryService.ExecuteWithRetry(operation)
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
	})

	t.Run("failed_operation_with_retry", func(t *testing.T) {
		attemptCount := 0
		operation := func() error {
			attemptCount++
			if attemptCount < 2 {
				return context.DeadlineExceeded
			}
			return nil
		}

		err := retryService.ExecuteWithRetry(operation)
		if err != nil {
			t.Errorf("Expected success after retry, got error: %v", err)
		}

		if attemptCount != 2 {
			t.Errorf("Expected 2 attempts, got %d", attemptCount)
		}
	})

	t.Run("failed_operation_after_max_retries", func(t *testing.T) {
		operation := func() error {
			return context.DeadlineExceeded
		}

		err := retryService.ExecuteWithRetry(operation)
		if err == nil {
			t.Error("Expected error after max retries")
		}
	})
}

// TestCacheIntegration тестирует интеграцию кеша
func TestCacheIntegration(t *testing.T) {
	cfg := config.Load()
	orderCache := cache.NewOrderCache(cfg.Cache.MaxSize, cfg.Cache.TTL)
	defer func() {
		if cacheImpl, ok := orderCache.(*cache.OrderCache); ok {
			cacheImpl.Stop()
		}
	}()

	// Создаем несколько тестовых заказов
	orders := []*model.Order{
		{
			OrderUID:    "cache-test-1",
			TrackNumber: "TRACK001",
			Entry:       "WBIL",
			DateCreated: time.Now(),
		},
		{
			OrderUID:    "cache-test-2",
			TrackNumber: "TRACK002",
			Entry:       "WBIL",
			DateCreated: time.Now(),
		},
	}

	// Загружаем заказы в кеш
	orderCache.LoadAll(orders)

	// Проверяем размер кеша
	if orderCache.Size() != 2 {
		t.Errorf("Expected cache size 2, got %d", orderCache.Size())
	}

	// Проверяем получение заказов
	for _, order := range orders {
		cachedOrder, found := orderCache.Get(order.OrderUID)
		if !found {
			t.Errorf("Order %s not found in cache", order.OrderUID)
		}
		if cachedOrder.OrderUID != order.OrderUID {
			t.Errorf("Order UID mismatch: got %s, want %s",
				cachedOrder.OrderUID, order.OrderUID)
		}
	}

	// Проверяем статистику
	stats := orderCache.GetStats()
	if stats.Size != 2 {
		t.Errorf("Expected stats size 2, got %d", stats.Size)
	}
}
