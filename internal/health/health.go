package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Checker интерфейс для health check
type Checker interface {
	Check(ctx context.Context) error
	Name() string
}

// Health структура для health checks
type Health struct {
	checkers []Checker
}

// New создает новый Health checker
func New() *Health {
	return &Health{
		checkers: make([]Checker, 0),
	}
}

// AddChecker добавляет checker
func (h *Health) AddChecker(checker Checker) {
	h.checkers = append(h.checkers, checker)
}

// Check выполняет все health checks
func (h *Health) Check(ctx context.Context) map[string]interface{} {
	results := make(map[string]interface{})
	overall := "healthy"

	for _, checker := range h.checkers {
		start := time.Now()
		err := checker.Check(ctx)
		duration := time.Since(start)

		status := "healthy"
		if err != nil {
			status = "unhealthy"
			overall = "unhealthy"
		}

		results[checker.Name()] = map[string]interface{}{
			"status":   status,
			"duration": duration.String(),
			"error":    err,
		}
	}

	results["overall"] = overall
	return results
}

// Handler возвращает HTTP handler для health checks
func (h *Health) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		results := h.Check(ctx)

		status := http.StatusOK
		if results["overall"] == "unhealthy" {
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    results["overall"],
			"checks":    results,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// DatabaseChecker проверяет состояние базы данных
type DatabaseChecker struct {
	name      string
	checkFunc func(ctx context.Context) error
}

// NewDatabaseChecker создает новый DatabaseChecker
func NewDatabaseChecker(name string, checkFunc func(ctx context.Context) error) *DatabaseChecker {
	return &DatabaseChecker{
		name:      name,
		checkFunc: checkFunc,
	}
}

// Check выполняет проверку базы данных
func (c *DatabaseChecker) Check(ctx context.Context) error {
	return c.checkFunc(ctx)
}

// Name возвращает имя checker'а
func (c *DatabaseChecker) Name() string {
	return c.name
}

// KafkaChecker проверяет состояние Kafka
type KafkaChecker struct {
	name      string
	checkFunc func(ctx context.Context) error
}

// NewKafkaChecker создает новый KafkaChecker
func NewKafkaChecker(name string, checkFunc func(ctx context.Context) error) *KafkaChecker {
	return &KafkaChecker{
		name:      name,
		checkFunc: checkFunc,
	}
}

// Check выполняет проверку Kafka
func (c *KafkaChecker) Check(ctx context.Context) error {
	return c.checkFunc(ctx)
}

// Name возвращает имя checker'а
func (c *KafkaChecker) Name() string {
	return c.name
}

// CacheChecker проверяет состояние кеша
type CacheChecker struct {
	name      string
	checkFunc func(ctx context.Context) error
}

// NewCacheChecker создает новый CacheChecker
func NewCacheChecker(name string, checkFunc func(ctx context.Context) error) *CacheChecker {
	return &CacheChecker{
		name:      name,
		checkFunc: checkFunc,
	}
}

// Check выполняет проверку кеша
func (c *CacheChecker) Check(ctx context.Context) error {
	return c.checkFunc(ctx)
}

// Name возвращает имя checker'а
func (c *CacheChecker) Name() string {
	return c.name
}
