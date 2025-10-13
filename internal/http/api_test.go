package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wbtest/internal/interfaces"
	"wbtest/internal/model"
)

// MockOrderCache мок кеша
type MockOrderCache struct {
	orders map[string]*model.Order
}

func NewMockOrderCache() *MockOrderCache {
	return &MockOrderCache{
		orders: make(map[string]*model.Order),
	}
}

func (m *MockOrderCache) Get(orderUID string) (*model.Order, bool) {
	if order, exists := m.orders[orderUID]; exists {
		return order, true
	}
	return nil, false
}

func (m *MockOrderCache) Set(order *model.Order) {
	m.orders[order.OrderUID] = order
}

func (m *MockOrderCache) LoadAll(orders []*model.Order) {
	for _, order := range orders {
		m.orders[order.OrderUID] = order
	}
}

func (m *MockOrderCache) Delete(orderUID string) {
	delete(m.orders, orderUID)
}

func (m *MockOrderCache) Size() int {
	return len(m.orders)
}

func (m *MockOrderCache) Clear() {
	m.orders = make(map[string]*model.Order)
}

func (m *MockOrderCache) GetStats() interfaces.CacheStats {
	return interfaces.CacheStats{
		Size: len(m.orders),
	}
}

func (m *MockOrderCache) Stop() {}

// MockOrderRepository мок БД
type MockOrderRepository struct {
	orders map[string]*model.Order
}

func NewMockOrderRepository() *MockOrderRepository {
	return &MockOrderRepository{
		orders: make(map[string]*model.Order),
	}
}

func (m *MockOrderRepository) LoadAllOrders(ctx context.Context) ([]*model.Order, error) {
	var orders []*model.Order
	for _, order := range m.orders {
		orders = append(orders, order)
	}
	return orders, nil
}

func (m *MockOrderRepository) SaveOrder(ctx context.Context, order *model.Order) error {
	m.orders[order.OrderUID] = order
	return nil
}

func (m *MockOrderRepository) GetOrderByUID(ctx context.Context, orderUID string) (*model.Order, error) {
	if order, exists := m.orders[orderUID]; exists {
		return order, nil
	}
	return nil, nil
}

func (m *MockOrderRepository) Close() {}

func TestServer_handleGetOrder_CacheHit(t *testing.T) {
	// Создаем моки
	cache := NewMockOrderCache()
	db := NewMockOrderRepository()

	// Создаем сервер
	server := NewServer(cache, db)

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
	}

	// Добавляем заказ в кеш
	cache.Set(testOrder)

	// Создаем запрос
	req, err := http.NewRequest("GET", "/order/test-order-123", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Создаем ResponseRecorder
	rr := httptest.NewRecorder()

	// Выполняем запрос
	server.ServeHTTP(rr, req)

	// Проверяем статус код
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}

	// Проверяем содержимое ответа
	var response model.Order
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.OrderUID != testOrder.OrderUID {
		t.Errorf("Expected OrderUID %s, got %s", testOrder.OrderUID, response.OrderUID)
	}
}

func TestServer_handleGetOrder_CacheMiss_DBFallback(t *testing.T) {
	// Создаем моки
	cache := NewMockOrderCache()
	db := NewMockOrderRepository()

	// Создаем сервер
	server := NewServer(cache, db)

	// Создаем тестовый заказ
	testOrder := &model.Order{
		OrderUID:    "test-order-456",
		TrackNumber: "TRACK456",
		Entry:       "WBIL",
		Delivery: model.Delivery{
			Name:  "Test User 2",
			Phone: "+1234567891",
			Email: "test2@example.com",
		},
		Payment: model.Payment{
			Transaction: "test-order-456",
			Currency:    "EUR",
			Amount:      2000,
		},
	}

	// Добавляем заказ в БД (но НЕ в кеш)
	db.SaveOrder(context.Background(), testOrder)

	// Создаем запрос
	req, err := http.NewRequest("GET", "/order/test-order-456", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Создаем ResponseRecorder
	rr := httptest.NewRecorder()

	// Выполняем запрос
	server.ServeHTTP(rr, req)

	// Проверяем статус код
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}

	// Проверяем содержимое ответа
	var response model.Order
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.OrderUID != testOrder.OrderUID {
		t.Errorf("Expected OrderUID %s, got %s", testOrder.OrderUID, response.OrderUID)
	}

	// Проверяем, что заказ теперь в кеше
	_, exists := cache.Get(testOrder.OrderUID)
	if !exists {
		t.Error("Expected order to be cached after DB fallback")
	}
}

func TestServer_handleGetOrder_NotFound(t *testing.T) {
	// Создаем моки
	cache := NewMockOrderCache()
	db := NewMockOrderRepository()

	// Создаем сервер
	server := NewServer(cache, db)

	// Создаем запрос для несуществующего заказа
	req, err := http.NewRequest("GET", "/order/nonexistent-order", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Создаем ResponseRecorder
	rr := httptest.NewRecorder()

	// Выполняем запрос
	server.ServeHTTP(rr, req)

	// Проверяем статус код
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, status)
	}
}

func TestServer_handleCreateOrder(t *testing.T) {
	// Создаем моки
	cache := NewMockOrderCache()
	db := NewMockOrderRepository()

	// Создаем сервер
	server := NewServer(cache, db)

	// Создаем тестовый заказ
	testOrder := &model.Order{
		OrderUID:    "test-order-789",
		TrackNumber: "TRACK789",
		Entry:       "WBIL",
		Delivery: model.Delivery{
			Name:  "Test User 3",
			Phone: "+1234567892",
			Email: "test3@example.com",
		},
		Payment: model.Payment{
			Transaction: "test-order-789",
			Currency:    "GBP",
			Amount:      3000,
		},
	}

	// Преобразуем заказ в JSON
	orderJSON, err := json.Marshal(testOrder)
	if err != nil {
		t.Fatalf("Failed to marshal test order: %v", err)
	}

	// Создаем запрос
	req, err := http.NewRequest("POST", "/order", bytes.NewBuffer(orderJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder
	rr := httptest.NewRecorder()

	// Выполняем запрос
	server.ServeHTTP(rr, req)

	// Проверяем статус код
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, status)
	}

	// Проверяем, что заказ добавлен в кеш
	_, exists := cache.Get(testOrder.OrderUID)
	if !exists {
		t.Error("Expected order to be cached after creation")
	}
}

func TestServer_handleHealth(t *testing.T) {
	// Создаем моки
	cache := NewMockOrderCache()
	db := NewMockOrderRepository()

	// Создаем сервер
	server := NewServer(cache, db)

	// Создаем запрос
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Создаем ResponseRecorder
	rr := httptest.NewRecorder()

	// Выполняем запрос
	server.ServeHTTP(rr, req)

	// Проверяем статус код
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}

	// Проверяем содержимое ответа
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", response["status"])
	}

	if response["service"] != "order-service" {
		t.Errorf("Expected service 'order-service', got %v", response["service"])
	}
}
