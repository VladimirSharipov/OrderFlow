package interfaces

import (
	"context"
	"wbtest/internal/model"
)

// CacheStats статистика кеша
type CacheStats struct {
	Size        int
	Hits        int64
	Misses      int64
	HitRate     float64
	Evictions   int64
	Expirations int64
}

// OrderRepository интерфейс для работы с заказами в БД
type OrderRepository interface {
	LoadAllOrders(ctx context.Context) ([]*model.Order, error)
	SaveOrder(ctx context.Context, order *model.Order) error
	GetOrderByUID(ctx context.Context, orderUID string) (*model.Order, error)
	Close()
}

// OrderCache интерфейс для кеша заказов
type OrderCache interface {
	Get(orderUID string) (*model.Order, bool)
	Set(order *model.Order)
	LoadAll(orders []*model.Order)
	Delete(orderUID string)
	Size() int
	Clear()
	GetStats() CacheStats
	Stop()
}

// MessageConsumer интерфейс для Kafka consumer
type MessageConsumer interface {
	ReadMessages(ctx context.Context, handle func([]byte)) error
	Close() error
}

// OrderValidator интерфейс для валидации заказов
type OrderValidator interface {
	Validate(order *model.Order) error
}

// RetryService интерфейс для retry логики
type RetryService interface {
	ExecuteWithRetry(operation func() error) error
}

// DLQService интерфейс для Dead Letter Queue
type DLQService interface {
	SendToDLQ(message []byte, reason string) error
	ProcessDLQ() error
	Close() error
}
