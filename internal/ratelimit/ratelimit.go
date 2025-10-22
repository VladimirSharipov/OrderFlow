package ratelimit

import (
	"context"
	"sync"
	"time"
)

// RateLimiter интерфейс для ограничения скорости запросов
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	Wait(ctx context.Context, key string) error
	Reset(key string)
	Stats(key string) *Stats
}

// Stats статистика rate limiter
type Stats struct {
	Allowed   int64
	Denied    int64
	ResetTime time.Time
}

// Config конфигурация rate limiter
type Config struct {
	Requests int           // Количество запросов
	Window   time.Duration // Временное окно
	Burst    int           // Размер burst (дополнительные запросы)
}

// TokenBucket реализация rate limiter на основе token bucket
type TokenBucket struct {
	config      Config
	buckets     map[string]*bucket
	mutex       sync.RWMutex
	cleanupTick time.Duration
}

// bucket представляет один bucket для ключа
type bucket struct {
	tokens     float64
	lastRefill time.Time
	allowed    int64
	denied     int64
	burst      float64
}

// NewTokenBucket создает новый token bucket rate limiter
func NewTokenBucket(config Config) *TokenBucket {
	if config.Burst <= 0 {
		config.Burst = config.Requests
	}

	return &TokenBucket{
		config:      config,
		buckets:     make(map[string]*bucket),
		cleanupTick: 5 * time.Minute,
	}
}

// Allow проверяет можно ли выполнить запрос
func (tb *TokenBucket) Allow(ctx context.Context, key string) (bool, error) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	b := tb.getOrCreateBucket(key)
	now := time.Now()

	// Пополняем токены
	tb.refill(b, now)

	// Проверяем есть ли токены
	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		b.allowed++
		return true, nil
	}

	b.denied++
	return false, nil
}

// Wait ждет пока можно будет выполнить запрос
func (tb *TokenBucket) Wait(ctx context.Context, key string) error {
	for {
		allowed, err := tb.Allow(ctx, key)
		if err != nil {
			return err
		}

		if allowed {
			return nil
		}

		// Ждем немного перед следующей попыткой
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 10):
			// Продолжаем
		}
	}
}

// Reset сбрасывает bucket для ключа
func (tb *TokenBucket) Reset(key string) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	if b, exists := tb.buckets[key]; exists {
		b.tokens = float64(tb.config.Requests)
		b.lastRefill = time.Now()
		b.allowed = 0
		b.denied = 0
	}
}

// Stats возвращает статистику для ключа
func (tb *TokenBucket) Stats(key string) *Stats {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()

	if b, exists := tb.buckets[key]; exists {
		return &Stats{
			Allowed:   b.allowed,
			Denied:    b.denied,
			ResetTime: b.lastRefill.Add(tb.config.Window),
		}
	}

	return &Stats{}
}

// getOrCreateBucket получает или создает bucket для ключа
func (tb *TokenBucket) getOrCreateBucket(key string) *bucket {
	if b, exists := tb.buckets[key]; exists {
		return b
	}

	b := &bucket{
		tokens:     float64(tb.config.Requests),
		lastRefill: time.Now(),
		burst:      float64(tb.config.Burst),
	}

	tb.buckets[key] = b
	return b
}

// refill пополняет токены в bucket
func (tb *TokenBucket) refill(b *bucket, now time.Time) {
	elapsed := now.Sub(b.lastRefill)
	if elapsed <= 0 {
		return
	}

	// Вычисляем сколько токенов добавить
	tokensToAdd := float64(elapsed) / float64(tb.config.Window) * float64(tb.config.Requests)

	// Добавляем токены, но не больше burst размера
	b.tokens += tokensToAdd
	if b.tokens > b.burst {
		b.tokens = b.burst
	}

	b.lastRefill = now
}

// StartCleanup запускает периодическую очистку неиспользуемых buckets
func (tb *TokenBucket) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(tb.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tb.cleanup()
		}
	}
}

// cleanup удаляет старые неиспользуемые buckets
func (tb *TokenBucket) cleanup() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-tb.config.Window * 2) // Удаляем buckets старше 2 окон

	for key, bucket := range tb.buckets {
		if bucket.lastRefill.Before(cutoff) && bucket.allowed == 0 && bucket.denied == 0 {
			delete(tb.buckets, key)
		}
	}
}

// FixedWindow реализация rate limiter на основе fixed window
type FixedWindow struct {
	config  Config
	windows map[string]*window
	mutex   sync.RWMutex
}

// window представляет одно временное окно
type window struct {
	count     int64
	startTime time.Time
	allowed   int64
	denied    int64
}

// NewFixedWindow создает новый fixed window rate limiter
func NewFixedWindow(config Config) *FixedWindow {
	return &FixedWindow{
		config:  config,
		windows: make(map[string]*window),
	}
}

// Allow проверяет можно ли выполнить запрос
func (fw *FixedWindow) Allow(ctx context.Context, key string) (bool, error) {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()

	w := fw.getOrCreateWindow(key)
	now := time.Now()

	// Проверяем нужно ли сбросить окно
	if now.Sub(w.startTime) >= fw.config.Window {
		w.count = 0
		w.startTime = now.Truncate(fw.config.Window)
	}

	// Проверяем лимит
	if w.count < int64(fw.config.Requests) {
		w.count++
		w.allowed++
		return true, nil
	}

	w.denied++
	return false, nil
}

// Wait ждет пока можно будет выполнить запрос
func (fw *FixedWindow) Wait(ctx context.Context, key string) error {
	for {
		allowed, err := fw.Allow(ctx, key)
		if err != nil {
			return err
		}

		if allowed {
			return nil
		}

		// Вычисляем когда следующее окно начнется
		now := time.Now()
		nextWindow := now.Truncate(fw.config.Window).Add(fw.config.Window)
		waitTime := nextWindow.Sub(now)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Следующее окно началось
		}
	}
}

// Reset сбрасывает окно для ключа
func (fw *FixedWindow) Reset(key string) {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()

	if w, exists := fw.windows[key]; exists {
		w.count = 0
		w.startTime = time.Now().Truncate(fw.config.Window)
		w.allowed = 0
		w.denied = 0
	}
}

// Stats возвращает статистику для ключа
func (fw *FixedWindow) Stats(key string) *Stats {
	fw.mutex.RLock()
	defer fw.mutex.RUnlock()

	if w, exists := fw.windows[key]; exists {
		return &Stats{
			Allowed:   w.allowed,
			Denied:    w.denied,
			ResetTime: w.startTime.Add(fw.config.Window),
		}
	}

	return &Stats{}
}

// getOrCreateWindow получает или создает окно для ключа
func (fw *FixedWindow) getOrCreateWindow(key string) *window {
	if w, exists := fw.windows[key]; exists {
		return w
	}

	now := time.Now()
	w := &window{
		startTime: now.Truncate(fw.config.Window),
	}

	fw.windows[key] = w
	return w
}

// NewRateLimiter создает новый rate limiter
func NewRateLimiter(config Config, algorithm string) RateLimiter {
	switch algorithm {
	case "token-bucket":
		return NewTokenBucket(config)
	case "fixed-window":
		return NewFixedWindow(config)
	default:
		return NewTokenBucket(config) // По умолчанию token bucket
	}
}
