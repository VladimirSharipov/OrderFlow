package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"wbtest/internal/interfaces"
	"wbtest/internal/model"
)

// MockDB мок БД
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

// MockCache мок кеша
type MockCache struct {
	orders map[string]*model.Order
}

func NewMockCache() *MockCache {
	return &MockCache{
		orders: make(map[string]*model.Order),
	}
}

func (m *MockCache) Get(orderUID string) (*model.Order, bool) {
	if order, exists := m.orders[orderUID]; exists {
		return order, true
	}
	return nil, false
}

func (m *MockCache) Set(order *model.Order) {
	m.orders[order.OrderUID] = order
}

func (m *MockCache) LoadAll(orders []*model.Order) {
	for _, order := range orders {
		m.orders[order.OrderUID] = order
	}
}

func (m *MockCache) Delete(orderUID string) {
	delete(m.orders, orderUID)
}

func (m *MockCache) Size() int {
	return len(m.orders)
}

func (m *MockCache) Clear() {
	m.orders = make(map[string]*model.Order)
}

func (m *MockCache) GetStats() interfaces.CacheStats {
	return interfaces.CacheStats{
		Size: len(m.orders),
	}
}

func (m *MockCache) Stop() {}

// MockValidator мок валидатора
type MockValidator struct{}

func (m *MockValidator) Validate(order *model.Order) error {
	if order == nil {
		return errors.New("order is nil")
	}
	if order.OrderUID == "" {
		return errors.New("order UID is empty")
	}
	return nil
}

// MockRetryService мок retry
type MockRetryService struct{}

func (m *MockRetryService) ExecuteWithRetry(operation func() error) error {
	return operation()
}

// MockDLQService мок DLQ
type MockDLQService struct{}

func (m *MockDLQService) SendToDLQ(message []byte, reason string) error {
	return nil
}

func (m *MockDLQService) ProcessDLQ() error {
	return nil
}

func (m *MockDLQService) Close() error {
	return nil
}

func TestMessageHandler_HandleMessage(t *testing.T) {
	// Создаем моки
	mockDB := NewMockDB()
	mockCache := NewMockCache()
	mockValidator := &MockValidator{}
	mockRetryService := &MockRetryService{}
	mockDLQService := &MockDLQService{}

	// Создаем тестовое приложение
	app := &App{
		DB:           mockDB,
		Cache:        mockCache,
		Validator:    mockValidator,
		RetryService: mockRetryService,
		DLQService:   mockDLQService,
	}

	// Создаем обработчик сообщений
	handler := NewMessageHandler(app)

	// Создаем тестовый заказ
	testOrder := &model.Order{
		OrderUID:    "test-order-123",
		TrackNumber: "TRACK123",
		Entry:       "WBIL",
		Delivery: model.Delivery{
			Name:  "Test User",
			Phone: "+1234567890",
			Email: "test@example.com",
		},
		Payment: model.Payment{
			Transaction: "test-order-123",
			Currency:    "USD",
			Amount:      1000,
		},
		Items: []model.Item{
			{
				ChrtID:      123456,
				TrackNumber: "ITEM123",
				Price:       1000,
				Name:        "Test Item",
			},
		},
	}

	// Преобразуем заказ в JSON
	orderJSON, err := json.Marshal(testOrder)
	if err != nil {
		t.Fatalf("Failed to marshal test order: %v", err)
	}

	// Тестируем обработку сообщения
	ctx := context.Background()
	err = handler.HandleMessage(ctx, orderJSON)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Проверяем, что заказ сохранился в БД
	savedOrder, err := mockDB.GetOrderByUID(ctx, testOrder.OrderUID)
	if err != nil {
		t.Errorf("Failed to get saved order: %v", err)
	}
	if savedOrder == nil {
		t.Error("Expected order to be saved in DB")
	}

	// Проверяем, что заказ сохранился в кеше
	_, exists := mockCache.Get(testOrder.OrderUID)
	if !exists {
		t.Error("Expected order to be saved in cache")
	}
}

func TestMessageHandler_HandleMessage_InvalidJSON(t *testing.T) {
	// Создаем моки
	mockDB := NewMockDB()
	mockCache := NewMockCache()
	mockValidator := &MockValidator{}
	mockRetryService := &MockRetryService{}
	mockDLQService := &MockDLQService{}

	// Создаем тестовое приложение
	app := &App{
		DB:           mockDB,
		Cache:        mockCache,
		Validator:    mockValidator,
		RetryService: mockRetryService,
		DLQService:   mockDLQService,
	}

	// Создаем обработчик сообщений
	handler := NewMessageHandler(app)

	// Тестируем обработку невалидного JSON
	ctx := context.Background()
	invalidJSON := []byte(`{"invalid": json}`)

	err := handler.HandleMessage(ctx, invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestMessageHandler_HandleMessage_InvalidOrder(t *testing.T) {
	// Создаем моки
	mockDB := NewMockDB()
	mockCache := NewMockCache()
	mockValidator := &MockValidator{}
	mockRetryService := &MockRetryService{}
	mockDLQService := &MockDLQService{}

	// Создаем тестовое приложение
	app := &App{
		DB:           mockDB,
		Cache:        mockCache,
		Validator:    mockValidator,
		RetryService: mockRetryService,
		DLQService:   mockDLQService,
	}

	// Создаем обработчик сообщений
	handler := NewMessageHandler(app)

	// Создаем невалидный заказ (без OrderUID)
	invalidOrder := &model.Order{
		TrackNumber: "TRACK123",
		Entry:       "WBIL",
	}

	// Преобразуем заказ в JSON
	orderJSON, err := json.Marshal(invalidOrder)
	if err != nil {
		t.Fatalf("Failed to marshal invalid order: %v", err)
	}

	// Тестируем обработку невалидного заказа
	ctx := context.Background()
	err = handler.HandleMessage(ctx, orderJSON)
	if err == nil {
		t.Error("Expected error for invalid order")
	}
}
