package cache

import (
	"fmt"
	"testing"
	"time"

	"wbtest/internal/model"
)

func TestOrderCache_Get(t *testing.T) {
	cache := NewOrderCache(10, time.Hour)
	defer cache.(*OrderCache).Stop()

	// Тест получения несуществующего элемента
	order, exists := cache.Get("nonexistent")
	if exists {
		t.Error("Expected non-existent order to return false")
	}
	if order != nil {
		t.Error("Expected non-existent order to return nil")
	}

	// Тест получения существующего элемента
	testOrder := &model.Order{
		OrderUID:    "test123",
		TrackNumber: "TRACK123",
		Entry:       "WBIL",
	}
	cache.Set(testOrder)

	order, exists = cache.Get("test123")
	if !exists {
		t.Error("Expected existing order to return true")
	}
	if order == nil {
		t.Error("Expected existing order to return non-nil")
	}
	if order.OrderUID != "test123" {
		t.Errorf("Expected order UID 'test123', got '%s'", order.OrderUID)
	}
}

func TestOrderCache_Set(t *testing.T) {
	cache := NewOrderCache(10, time.Hour)
	defer cache.(*OrderCache).Stop()

	// Тест установки валидного заказа
	testOrder := &model.Order{
		OrderUID:    "test123",
		TrackNumber: "TRACK123",
		Entry:       "WBIL",
	}
	cache.Set(testOrder)

	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}

	// Тест установки nil заказа
	cache.Set(nil)
	if cache.Size() != 1 {
		t.Errorf("Expected cache size to remain 1 after setting nil, got %d", cache.Size())
	}

	// Тест установки заказа с пустым UID
	emptyOrder := &model.Order{
		OrderUID:    "",
		TrackNumber: "TRACK123",
	}
	cache.Set(emptyOrder)
	if cache.Size() != 1 {
		t.Errorf("Expected cache size to remain 1 after setting order with empty UID, got %d", cache.Size())
	}
}

func TestOrderCache_Delete(t *testing.T) {
	cache := NewOrderCache(10, time.Hour)
	defer cache.(*OrderCache).Stop()

	// Добавляем заказ
	testOrder := &model.Order{
		OrderUID:    "test123",
		TrackNumber: "TRACK123",
	}
	cache.Set(testOrder)

	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}

	// Удаляем заказ
	cache.Delete("test123")
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after deletion, got %d", cache.Size())
	}

	// Проверяем, что заказ действительно удален
	order, exists := cache.Get("test123")
	if exists {
		t.Error("Expected deleted order to not exist")
	}
	if order != nil {
		t.Error("Expected deleted order to return nil")
	}
}

func TestOrderCache_Clear(t *testing.T) {
	cache := NewOrderCache(10, time.Hour)
	defer cache.(*OrderCache).Stop()

	// Добавляем несколько заказов
	for i := 0; i < 3; i++ {
		order := &model.Order{
			OrderUID:    fmt.Sprintf("test%d", i),
			TrackNumber: fmt.Sprintf("TRACK%d", i),
		}
		cache.Set(order)
	}

	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}

	// Очищаем кеш
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}
}

func TestOrderCache_LoadAll(t *testing.T) {
	cache := NewOrderCache(10, time.Hour)
	defer cache.(*OrderCache).Stop()

	// Создаем тестовые заказы
	orders := []*model.Order{
		{OrderUID: "test1", TrackNumber: "TRACK1"},
		{OrderUID: "test2", TrackNumber: "TRACK2"},
		{OrderUID: "test3", TrackNumber: "TRACK3"},
	}

	// Загружаем все заказы
	cache.LoadAll(orders)

	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}

	// Проверяем, что все заказы доступны
	for _, order := range orders {
		retrieved, exists := cache.Get(order.OrderUID)
		if !exists {
			t.Errorf("Expected order %s to exist", order.OrderUID)
		}
		if retrieved.OrderUID != order.OrderUID {
			t.Errorf("Expected order UID %s, got %s", order.OrderUID, retrieved.OrderUID)
		}
	}
}

func TestOrderCache_Eviction(t *testing.T) {
	cache := NewOrderCache(2, time.Hour) // Максимум 2 элемента
	defer cache.(*OrderCache).Stop()

	// Добавляем 3 заказа (больше максимального размера)
	for i := 0; i < 3; i++ {
		order := &model.Order{
			OrderUID:    fmt.Sprintf("test%d", i),
			TrackNumber: fmt.Sprintf("TRACK%d", i),
		}
		cache.Set(order)
	}

	// Проверяем, что размер кеша не превышает максимум
	if cache.Size() > 2 {
		t.Errorf("Expected cache size <= 2, got %d", cache.Size())
	}

	// Проверяем, что самый старый элемент был удален
	_, exists := cache.Get("test0")
	if exists {
		t.Error("Expected oldest element to be evicted")
	}
}

func TestOrderCache_TTL(t *testing.T) {
	cache := NewOrderCache(10, time.Millisecond*100) // TTL 100ms
	defer cache.(*OrderCache).Stop()

	// Добавляем заказ
	testOrder := &model.Order{
		OrderUID:    "test123",
		TrackNumber: "TRACK123",
	}
	cache.Set(testOrder)

	// Проверяем, что заказ доступен сразу
	order, exists := cache.Get("test123")
	if !exists {
		t.Error("Expected order to exist immediately after setting")
	}
	if order == nil {
		t.Error("Expected order to be non-nil immediately after setting")
	}

	// Ждем, пока заказ истечет
	time.Sleep(time.Millisecond * 150)

	// Проверяем, что заказ больше не доступен
	order, exists = cache.Get("test123")
	if exists {
		t.Error("Expected order to be expired")
	}
	if order != nil {
		t.Error("Expected expired order to return nil")
	}
}

func TestOrderCache_GetStats(t *testing.T) {
	cache := NewOrderCache(10, time.Hour)
	defer cache.(*OrderCache).Stop()

	// Добавляем заказ
	testOrder := &model.Order{
		OrderUID:    "test123",
		TrackNumber: "TRACK123",
	}
	cache.Set(testOrder)

	// Получаем статистику
	stats := cache.GetStats()

	if stats.Size != 1 {
		t.Errorf("Expected stats size 1, got %d", stats.Size)
	}

	// Делаем несколько запросов для проверки hits/misses
	cache.Get("test123")     // hit
	cache.Get("nonexistent") // miss
	cache.Get("test123")     // hit

	stats = cache.GetStats()
	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
	if stats.HitRate < 66.0 || stats.HitRate > 67.0 {
		t.Errorf("Expected hit rate ~66.67%%, got %.2f%%", stats.HitRate)
	}
}
