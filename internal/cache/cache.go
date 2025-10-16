package cache

import (
	"sync"
	"time"
	"wbtest/internal/interfaces"
	"wbtest/internal/model"
)

type cacheEntry struct {
	order      *model.Order
	createdAt  time.Time
	lastAccess time.Time
	mu         sync.RWMutex // мелкогранулярная блокировка для каждого элемента
}

type OrderCache struct {
	orders          map[string]*cacheEntry
	mu              sync.RWMutex // блокировка для map
	maxSize         int
	ttl             time.Duration
	cleanupInterval time.Duration
	stopCleanup     chan struct{}

	// Метрики
	stats struct {
		mu          sync.RWMutex
		hits        int64
		misses      int64
		evictions   int64
		expirations int64
	}
}

func NewOrderCache(maxSize int, ttl time.Duration) interfaces.OrderCache {
	cache := &OrderCache{
		orders:          make(map[string]*cacheEntry),
		maxSize:         maxSize,
		ttl:             ttl,
		cleanupInterval: time.Minute * 5,
		stopCleanup:     make(chan struct{}),
	}

	go cache.startCleanup()
	return cache
}

func (c *OrderCache) Get(orderUID string) (*model.Order, bool) {
	// Сначала проверяем существование записи
	c.mu.RLock()
	entry, exists := c.orders[orderUID]
	c.mu.RUnlock()

	if !exists {
		c.incMisses()
		return nil, false
	}

	// Проверяем TTL с мелкогранулярной блокировкой
	entry.mu.RLock()
	if time.Since(entry.createdAt) > c.ttl {
		entry.mu.RUnlock()
		c.Delete(orderUID)
		c.incExpirations()
		c.incMisses()
		return nil, false
	}

	// Обновляем время последнего доступа
	entry.lastAccess = time.Now()
	order := entry.order
	entry.mu.RUnlock()

	c.incHits()
	return order, true
}

func (c *OrderCache) Set(order *model.Order) {
	if order == nil || order.OrderUID == "" {
		return
	}

	now := time.Now()
	newEntry := &cacheEntry{
		order:      order,
		createdAt:  now,
		lastAccess: now,
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Если кеш полный, удаляем самый старый элемент
	if len(c.orders) >= c.maxSize {
		c.evictOldest()
	}

	c.orders[order.OrderUID] = newEntry
}

func (c *OrderCache) LoadAll(orders []*model.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Очищаем кеш перед загрузкой
	c.orders = make(map[string]*cacheEntry)

	now := time.Now()
	for _, order := range orders {
		if order != nil && order.OrderUID != "" {
			c.orders[order.OrderUID] = &cacheEntry{
				order:      order,
				createdAt:  now,
				lastAccess: now,
			}
		}
	}
}

func (c *OrderCache) Delete(orderUID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.orders, orderUID)
}

func (c *OrderCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.orders)
}

func (c *OrderCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.orders = make(map[string]*cacheEntry)
}

func (c *OrderCache) GetStats() interfaces.CacheStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	total := c.stats.hits + c.stats.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.stats.hits) / float64(total) * 100
	}

	return interfaces.CacheStats{
		Size:        c.Size(),
		Hits:        c.stats.hits,
		Misses:      c.stats.misses,
		HitRate:     hitRate,
		Evictions:   c.stats.evictions,
		Expirations: c.stats.expirations,
	}
}

func (c *OrderCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.orders {
		entry.mu.RLock()
		createdAt := entry.createdAt
		entry.mu.RUnlock()

		if oldestKey == "" || createdAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = createdAt
		}
	}

	if oldestKey != "" {
		delete(c.orders, oldestKey)
		c.incEvictions()
	}
}

func (c *OrderCache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

func (c *OrderCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	// Собираем ключи устаревших записей
	for key, entry := range c.orders {
		entry.mu.RLock()
		createdAt := entry.createdAt
		entry.mu.RUnlock()

		if now.Sub(createdAt) > c.ttl {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// Удаляем устаревшие записи
	for _, key := range expiredKeys {
		delete(c.orders, key)
		c.incExpirations()
	}
}

func (c *OrderCache) Stop() {
	close(c.stopCleanup)
}

// Методы для метрик
func (c *OrderCache) incHits() {
	c.stats.mu.Lock()
	c.stats.hits++
	c.stats.mu.Unlock()
}

func (c *OrderCache) incMisses() {
	c.stats.mu.Lock()
	c.stats.misses++
	c.stats.mu.Unlock()
}

func (c *OrderCache) incEvictions() {
	c.stats.mu.Lock()
	c.stats.evictions++
	c.stats.mu.Unlock()
}

func (c *OrderCache) incExpirations() {
	c.stats.mu.Lock()
	c.stats.expirations++
	c.stats.mu.Unlock()
}
