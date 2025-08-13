package cache

import (
	"sync"
	"wbtest/internal/model"
)

// OrderCache это кеш заказов в памяти
// используем map для быстрого поиска по ID заказа
type OrderCache struct {
	orders map[string]*model.Order
	mutex  sync.RWMutex
}

// NewOrderCache создает новый кеш заказов
func NewOrderCache() *OrderCache {
	return &OrderCache{
		orders: make(map[string]*model.Order),
	}
}

// Set добавляет заказ в кеш
func (c *OrderCache) Set(order *model.Order) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.orders[order.OrderUID] = order
}

// Get получает заказ из кеша по ID
func (c *OrderCache) Get(orderUID string) (*model.Order, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	order, exists := c.orders[orderUID]
	return order, exists
}

// LoadAll загружает все заказы в кеш
// используется при старте сервиса чтобы восстановить кеш из базы
func (c *OrderCache) LoadAll(orders []*model.Order) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, order := range orders {
		c.orders[order.OrderUID] = order
	}
}

// GetAll возвращает все заказы из кеша
func (c *OrderCache) GetAll() []*model.Order {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	orders := make([]*model.Order, 0, len(c.orders))
	for _, order := range c.orders {
		orders = append(orders, order)
	}
	return orders
}

// Clear очищает весь кеш
func (c *OrderCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.orders = make(map[string]*model.Order)
}

// Size возвращает количество заказов в кеше
func (c *OrderCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.orders)
}
