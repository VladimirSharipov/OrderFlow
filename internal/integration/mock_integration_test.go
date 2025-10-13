package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"wbtest/internal/cache"
	"wbtest/internal/config"
	"wbtest/internal/model"
	"wbtest/internal/retry"
	"wbtest/internal/validator"
)

// MockDB простой мок
type MockDB struct {
	orders map[string]*model.Order
}

func NewMockDB() *MockDB {
	return &MockDB{
		orders: make(map[string]*model.Order),
	}
}

func (m *MockDB) LoadAllOrders(ctx context.Context) ([]*model.Order, error) {
	var orders []*model.Order
	for _, order := range m.orders {
		orders = append(orders, order)
	}
	return orders, nil
}

func (m *MockDB) SaveOrder(ctx context.Context, order *model.Order) error {
	m.orders[order.OrderUID] = order
	return nil
}

func (m *MockDB) GetOrderByUID(ctx context.Context, orderUID string) (*model.Order, error) {
	if order, exists := m.orders[orderUID]; exists {
		return order, nil
	}
	return nil, nil
}

func (m *MockDB) Close() {}

// TestOrderServiceMockIntegration тестирует цикл
func TestOrderServiceMockIntegration(t *testing.T) {
	// Создаем мок БД
	mockDB := NewMockDB()

	// Создаем кеш
	orderCache := cache.NewOrderCache(100, time.Hour)
	defer orderCache.(*cache.OrderCache).Stop()

	// Создаем валидатор
	validator := validator.NewOrderValidator()

	// Создаем retry сервис
	retryService := retry.NewRetryService(&config.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
	})

	// Создаем тестовый заказ
	testOrder := &model.Order{
		OrderUID:    "integration-test-123",
		TrackNumber: "TRACK123",
		Entry:       "WBIL",
		Delivery: model.Delivery{
			Name:    "Integration Test User",
			Phone:   "+1234567890",
			Zip:     "12345",
			City:    "Test City",
			Address: "123 Test Street",
			Region:  "Test Region",
			Email:   "integration@test.com",
		},
		Payment: model.Payment{
			Transaction:  "integration-test-123",
			RequestID:    "",
			Currency:     "USD",
			Provider:     "test-provider",
			Amount:       1500,
			PaymentDT:    1637907727,
			Bank:         "test-bank",
			DeliveryCost: 100,
			GoodsTotal:   1400,
			CustomFee:    0,
		},
		Items: []model.Item{
			{
				ChrtID:      654321,
				TrackNumber: "ITEM123",
				Price:       1400,
				Rid:         "test-rid-123",
				Name:        "Integration Test Item",
				Sale:        0,
				Size:        "0",
				TotalPrice:  1400,
				NmID:        654321,
				Brand:       "Test Brand",
				Status:      202,
			},
		},
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "integration-customer",
		DeliveryService:   "integration-service",
		ShardKey:          "0",
		SmID:              1,
		DateCreated:       time.Now(),
		OofShard:          "0",
	}

	// Тест 1: Валидация заказа
	t.Run("validate_order", func(t *testing.T) {
		err := validator.Validate(testOrder)
		if err != nil {
			t.Errorf("Order validation failed: %v", err)
		}
	})

	// Тест 2: Сохранение в БД
	t.Run("save_to_database", func(t *testing.T) {
		err := mockDB.SaveOrder(context.Background(), testOrder)
		if err != nil {
			t.Errorf("Failed to save order to database: %v", err)
		}

		// Проверяем, что заказ сохранился
		savedOrder, err := mockDB.GetOrderByUID(context.Background(), testOrder.OrderUID)
		if err != nil {
			t.Errorf("Failed to get saved order: %v", err)
		}
		if savedOrder == nil {
			t.Error("Saved order not found")
		}
		if savedOrder.OrderUID != testOrder.OrderUID {
			t.Errorf("Expected OrderUID %s, got %s", testOrder.OrderUID, savedOrder.OrderUID)
		}
	})

	// Тест 3: Загрузка из БД в кеш
	t.Run("load_orders_from_database", func(t *testing.T) {
		orders, err := mockDB.LoadAllOrders(context.Background())
		if err != nil {
			t.Errorf("Failed to load orders from database: %v", err)
		}

		if len(orders) == 0 {
			t.Error("No orders loaded from database")
		}

		// Загружаем заказы в кеш
		orderCache.LoadAll(orders)

		// Проверяем, что заказ в кеше
		cachedOrder, exists := orderCache.Get(testOrder.OrderUID)
		if !exists {
			t.Error("Test order not found in cache")
		}
		if cachedOrder.OrderUID != testOrder.OrderUID {
			t.Errorf("Expected OrderUID %s, got %s", testOrder.OrderUID, cachedOrder.OrderUID)
		}
	})

	// Тест 4: Retry механизм
	t.Run("retry_mechanism", func(t *testing.T) {
		attempts := 0
		testOperation := func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary error")
			}
			return nil
		}

		err := retryService.ExecuteWithRetry(testOperation)
		if err != nil {
			t.Errorf("Retry mechanism failed: %v", err)
		}

		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})

	// Тест 5: JSON сериализация/десериализация
	t.Run("json_serialization", func(t *testing.T) {
		// Сериализуем заказ в JSON
		orderJSON, err := json.Marshal(testOrder)
		if err != nil {
			t.Errorf("Failed to marshal order: %v", err)
		}

		// Десериализуем обратно
		var deserializedOrder model.Order
		err = json.Unmarshal(orderJSON, &deserializedOrder)
		if err != nil {
			t.Errorf("Failed to unmarshal order: %v", err)
		}

		// Проверяем, что данные совпадают
		if deserializedOrder.OrderUID != testOrder.OrderUID {
			t.Errorf("Expected OrderUID %s, got %s", testOrder.OrderUID, deserializedOrder.OrderUID)
		}
		if deserializedOrder.Payment.Amount != testOrder.Payment.Amount {
			t.Errorf("Expected Amount %d, got %d", testOrder.Payment.Amount, deserializedOrder.Payment.Amount)
		}
	})
}
