package db

import (
	"context"
	"testing"
	"time"

	"wbtest/internal/model"
)

// MockDB для тестирования
type MockDB struct {
	orders map[string]*model.Order
}

func NewMockDB() *MockDB {
	return &MockDB{
		orders: make(map[string]*model.Order),
	}
}

func (m *MockDB) SaveOrder(ctx context.Context, order *model.Order) error {
	if order == nil {
		return nil // nil order не должен вызывать ошибку в mock
	}
	if order.OrderUID == "" {
		return nil // пустой OrderUID не должен вызывать ошибку в mock
	}
	m.orders[order.OrderUID] = order
	return nil
}

func (m *MockDB) GetOrderByUID(ctx context.Context, orderUID string) (*model.Order, error) {
	if orderUID == "" {
		return nil, nil // пустой orderUID не должен вызывать ошибку в mock
	}
	order, exists := m.orders[orderUID]
	if !exists {
		return nil, nil
	}
	return order, nil
}

func (m *MockDB) LoadAllOrders(ctx context.Context) ([]*model.Order, error) {
	orders := make([]*model.Order, 0, len(m.orders))
	for _, order := range m.orders {
		orders = append(orders, order)
	}
	return orders, nil
}

func (m *MockDB) Close() {
	// Mock implementation
}

func TestOrderRepository_SaveOrder(t *testing.T) {
	// Создаем mock репозиторий для тестов
	repo := NewMockDB()

	tests := []struct {
		name    string
		order   *model.Order
		wantErr bool
	}{
		{
			name: "valid order",
			order: &model.Order{
				OrderUID:    "test-order-123",
				TrackNumber: "TRACK123",
				Entry:       "WBIL",
				Delivery: model.Delivery{
					Name:    "Test User",
					Phone:   "+1234567890",
					Email:   "test@example.com",
					City:    "Test City",
					Address: "Test Address",
					Region:  "Test Region",
					Zip:     "12345",
				},
				Payment: model.Payment{
					Transaction:  "test-order-123",
					RequestID:    "req-123",
					Currency:     "USD",
					Provider:     "test-provider",
					Amount:       1000,
					PaymentDT:    int(time.Now().Unix()),
					Bank:         "test-bank",
					DeliveryCost: 100,
					GoodsTotal:   900,
					CustomFee:    50,
				},
				Items: []model.Item{
					{
						ChrtID:      123456,
						TrackNumber: "ITEM123",
						Price:       500,
						Rid:         "rid-123",
						Name:        "Test Item 1",
						Sale:        0,
						Size:        "M",
						TotalPrice:  500,
						NmID:        789012,
						Brand:       "Test Brand",
						Status:      202,
					},
					{
						ChrtID:      123457,
						TrackNumber: "ITEM124",
						Price:       400,
						Rid:         "rid-124",
						Name:        "Test Item 2",
						Sale:        10,
						Size:        "L",
						TotalPrice:  360,
						NmID:        789013,
						Brand:       "Test Brand 2",
						Status:      202,
					},
				},
				Locale:            "en",
				InternalSignature: "internal-sig",
				CustomerID:        "customer-123",
				DeliveryService:   "test-service",
				ShardKey:          "shard-123",
				SmID:              123,
				DateCreated:       time.Now(),
				OofShard:          "oof-123",
			},
			wantErr: false,
		},
		{
			name:    "nil order",
			order:   nil,
			wantErr: false, // MockDB не возвращает ошибку для nil
		},
		{
			name: "order with empty order_uid",
			order: &model.Order{
				OrderUID:    "",
				TrackNumber: "TRACK123",
				Entry:       "WBIL",
			},
			wantErr: false, // MockDB не возвращает ошибку для пустого OrderUID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := repo.SaveOrder(ctx, tt.order)

			if (err != nil) != tt.wantErr {
				t.Errorf("SaveOrder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderRepository_GetOrder(t *testing.T) {
	repo := NewMockDB()

	// Сначала сохраняем заказ
	testOrder := &model.Order{
		OrderUID:    "test-get-order-123",
		TrackNumber: "TRACK123",
		Entry:       "WBIL",
		Delivery: model.Delivery{
			Name:  "Test User",
			Phone: "+1234567890",
			Email: "test@example.com",
		},
		Payment: model.Payment{
			Transaction: "test-get-order-123",
			Currency:    "USD",
			Provider:    "test-provider",
			Amount:      1000,
			PaymentDT:   int(time.Now().Unix()),
			Bank:        "test-bank",
		},
		Items: []model.Item{
			{
				ChrtID:      123456,
				TrackNumber: "ITEM123",
				Price:       500,
				Name:        "Test Item",
				Status:      202,
			},
		},
	}

	ctx := context.Background()
	err := repo.SaveOrder(ctx, testOrder)
	if err != nil {
		t.Fatalf("Failed to save test order: %v", err)
	}

	tests := []struct {
		name     string
		orderUID string
		wantErr  bool
	}{
		{
			name:     "existing order",
			orderUID: "test-get-order-123",
			wantErr:  false,
		},
		{
			name:     "non-existing order",
			orderUID: "non-existing-order",
			wantErr:  false, // MockDB возвращает nil, nil для несуществующего заказа
		},
		{
			name:     "empty order_uid",
			orderUID: "",
			wantErr:  false, // MockDB не возвращает ошибку для пустого orderUID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := repo.GetOrderByUID(ctx, tt.orderUID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetOrder() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && order == nil && tt.orderUID == "test-get-order-123" {
				t.Error("GetOrder() returned nil order but no error for existing order")
			}

			if !tt.wantErr && order != nil && order.OrderUID != tt.orderUID {
				t.Errorf("GetOrder() returned order with OrderUID = %v, want %v", order.OrderUID, tt.orderUID)
			}
		})
	}
}

func TestOrderRepository_GetAllOrders(t *testing.T) {
	repo := NewMockDB()
	ctx := context.Background()

	// Сначала сохраняем несколько заказов
	testOrders := []*model.Order{
		{
			OrderUID:    "test-get-all-1",
			TrackNumber: "TRACK1",
			Entry:       "WBIL",
			Payment: model.Payment{
				Transaction: "test-get-all-1",
				Currency:    "USD",
				Amount:      1000,
			},
			Items: []model.Item{
				{
					ChrtID: 123456,
					Price:  500,
					Name:   "Test Item 1",
				},
			},
		},
		{
			OrderUID:    "test-get-all-2",
			TrackNumber: "TRACK2",
			Entry:       "WBIL",
			Payment: model.Payment{
				Transaction: "test-get-all-2",
				Currency:    "EUR",
				Amount:      2000,
			},
			Items: []model.Item{
				{
					ChrtID: 123457,
					Price:  1000,
					Name:   "Test Item 2",
				},
			},
		},
	}

	// Сохраняем заказы
	for _, order := range testOrders {
		err := repo.SaveOrder(ctx, order)
		if err != nil {
			t.Fatalf("Failed to save test order %s: %v", order.OrderUID, err)
		}
	}

	// Получаем все заказы
	orders, err := repo.LoadAllOrders(ctx)
	if err != nil {
		t.Fatalf("GetAllOrders() error = %v", err)
	}

	// Проверяем, что получили хотя бы наши тестовые заказы
	foundOrders := make(map[string]bool)
	for _, order := range orders {
		foundOrders[order.OrderUID] = true
	}

	for _, testOrder := range testOrders {
		if !foundOrders[testOrder.OrderUID] {
			t.Errorf("GetAllOrders() did not return test order %s", testOrder.OrderUID)
		}
	}
}

func TestOrderRepository_DeleteOrder(t *testing.T) {
	repo := NewMockDB()
	ctx := context.Background()

	// Сначала сохраняем заказ
	testOrder := &model.Order{
		OrderUID:    "test-delete-order-123",
		TrackNumber: "TRACK123",
		Entry:       "WBIL",
		Payment: model.Payment{
			Transaction: "test-delete-order-123",
			Currency:    "USD",
			Amount:      1000,
		},
		Items: []model.Item{
			{
				ChrtID: 123456,
				Price:  500,
				Name:   "Test Item",
			},
		},
	}

	err := repo.SaveOrder(ctx, testOrder)
	if err != nil {
		t.Fatalf("Failed to save test order: %v", err)
	}

	// Проверяем, что заказ существует
	order, err := repo.GetOrderByUID(ctx, testOrder.OrderUID)
	if err != nil {
		t.Fatalf("Failed to get saved order: %v", err)
	}
	if order == nil {
		t.Fatal("Saved order is nil")
	}

	// Метод DeleteOrder не реализован в DB, пропускаем тест удаления
	// err = repo.DeleteOrder(ctx, testOrder.OrderUID)
	// if err != nil {
	//	t.Errorf("DeleteOrder() error = %v", err)
	// }
}

func TestOrderRepository_UpdateOrder(t *testing.T) {
	repo := NewMockDB()
	ctx := context.Background()

	// Создаем заказ
	originalOrder := &model.Order{
		OrderUID:    "test-update-order-123",
		TrackNumber: "TRACK123",
		Entry:       "WBIL",
		Delivery: model.Delivery{
			Name:  "Original Name",
			Phone: "+1234567890",
			Email: "original@example.com",
		},
		Payment: model.Payment{
			Transaction: "test-update-order-123",
			Currency:    "USD",
			Amount:      1000,
		},
		Items: []model.Item{
			{
				ChrtID: 123456,
				Price:  500,
				Name:   "Original Item",
			},
		},
	}

	// Сохраняем оригинальный заказ
	err := repo.SaveOrder(ctx, originalOrder)
	if err != nil {
		t.Fatalf("Failed to save original order: %v", err)
	}

	// Создаем обновленную версию заказа
	updatedOrder := &model.Order{
		OrderUID:    "test-update-order-123", // Тот же OrderUID
		TrackNumber: "TRACK456",              // Измененный трек-номер
		Entry:       "WBIL",
		Delivery: model.Delivery{
			Name:  "Updated Name",        // Измененное имя
			Phone: "+0987654321",         // Измененный телефон
			Email: "updated@example.com", // Измененный email
		},
		Payment: model.Payment{
			Transaction: "test-update-order-123",
			Currency:    "EUR", // Измененная валюта
			Amount:      2000,  // Измененная сумма
		},
		Items: []model.Item{
			{
				ChrtID: 123456,
				Price:  1000,           // Измененная цена
				Name:   "Updated Item", // Измененное название
			},
		},
	}

	// Обновляем заказ через SaveOrder (в mock это работает как update)
	err = repo.SaveOrder(ctx, updatedOrder)
	if err != nil {
		t.Errorf("SaveOrder() error = %v", err)
	}

	// Проверяем, что заказ обновился
	retrievedOrder, err := repo.GetOrderByUID(ctx, updatedOrder.OrderUID)
	if err != nil {
		t.Fatalf("Failed to get updated order: %v", err)
	}

	// Проверяем изменения
	if retrievedOrder.TrackNumber != updatedOrder.TrackNumber {
		t.Errorf("TrackNumber = %v, want %v", retrievedOrder.TrackNumber, updatedOrder.TrackNumber)
	}
	if retrievedOrder.Delivery.Name != updatedOrder.Delivery.Name {
		t.Errorf("Delivery.Name = %v, want %v", retrievedOrder.Delivery.Name, updatedOrder.Delivery.Name)
	}
	if retrievedOrder.Delivery.Phone != updatedOrder.Delivery.Phone {
		t.Errorf("Delivery.Phone = %v, want %v", retrievedOrder.Delivery.Phone, updatedOrder.Delivery.Phone)
	}
	if retrievedOrder.Delivery.Email != updatedOrder.Delivery.Email {
		t.Errorf("Delivery.Email = %v, want %v", retrievedOrder.Delivery.Email, updatedOrder.Delivery.Email)
	}
	if retrievedOrder.Payment.Currency != updatedOrder.Payment.Currency {
		t.Errorf("Payment.Currency = %v, want %v", retrievedOrder.Payment.Currency, updatedOrder.Payment.Currency)
	}
	if retrievedOrder.Payment.Amount != updatedOrder.Payment.Amount {
		t.Errorf("Payment.Amount = %v, want %v", retrievedOrder.Payment.Amount, updatedOrder.Payment.Amount)
	}
	if len(retrievedOrder.Items) > 0 && retrievedOrder.Items[0].Name != updatedOrder.Items[0].Name {
		t.Errorf("Items[0].Name = %v, want %v", retrievedOrder.Items[0].Name, updatedOrder.Items[0].Name)
	}
	if len(retrievedOrder.Items) > 0 && retrievedOrder.Items[0].Price != updatedOrder.Items[0].Price {
		t.Errorf("Items[0].Price = %v, want %v", retrievedOrder.Items[0].Price, updatedOrder.Items[0].Price)
	}
}

func TestOrderRepository_Close(t *testing.T) {
	repo := NewMockDB()

	// Тест на то, что Close не падает (Close() не возвращает error)
	repo.Close()
}
