package ratelimit

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Middleware HTTP middleware для rate limiting
type Middleware struct {
	limiter RateLimiter
	logger  *logrus.Logger
	keyFunc KeyFunc
	onLimit OnLimitFunc
}

// KeyFunc функция для извлечения ключа из запроса
type KeyFunc func(r *http.Request) string

// OnLimitFunc функция вызываемая при превышении лимита
type OnLimitFunc func(w http.ResponseWriter, r *http.Request, key string)

// Config конфигурация middleware
type MiddlewareConfig struct {
	Requests  int
	Window    time.Duration
	Burst     int
	Algorithm string
}

// NewMiddleware создает новый middleware для rate limiting
func NewMiddleware(config MiddlewareConfig, logger *logrus.Logger) *Middleware {
	limiterConfig := Config{
		Requests: config.Requests,
		Window:   config.Window,
		Burst:    config.Burst,
	}

	limiter := NewRateLimiter(limiterConfig, config.Algorithm)

	return &Middleware{
		limiter: limiter,
		logger:  logger,
		keyFunc: DefaultKeyFunc,
		onLimit: DefaultOnLimit,
	}
}

// WithKeyFunc устанавливает функцию для извлечения ключа
func (m *Middleware) WithKeyFunc(keyFunc KeyFunc) *Middleware {
	m.keyFunc = keyFunc
	return m
}

// WithOnLimit устанавливает функцию для обработки превышения лимита
func (m *Middleware) WithOnLimit(onLimit OnLimitFunc) *Middleware {
	m.onLimit = onLimit
	return m
}

// Handler возвращает HTTP handler с rate limiting
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := m.keyFunc(r)

		// Проверяем лимит
		allowed, err := m.limiter.Allow(r.Context(), key)
		if err != nil {
			m.logger.WithError(err).Error("Rate limiter error")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !allowed {
			// Превышен лимит
			m.logger.WithFields(logrus.Fields{
				"key":         key,
				"method":      r.Method,
				"path":        r.URL.Path,
				"remote_addr": r.RemoteAddr,
			}).Warn("Rate limit exceeded")

			m.onLimit(w, r, key)
			return
		}

		// Добавляем заголовки с информацией о лимитах
		stats := m.limiter.Stats(key)
		if tb, ok := m.limiter.(*TokenBucket); ok {
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(tb.config.Requests))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(tb.config.Requests-int(stats.Allowed)))
			w.Header().Set("X-RateLimit-Reset", stats.ResetTime.Format(time.RFC3339))
		}

		next.ServeHTTP(w, r)
	})
}

// DefaultKeyFunc извлекает ключ по IP адресу
func DefaultKeyFunc(r *http.Request) string {
	// Пытаемся получить реальный IP
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = strings.Split(r.RemoteAddr, ":")[0]
	}

	return ip
}

// DefaultOnLimit обработчик по умолчанию для превышения лимита
func DefaultOnLimit(w http.ResponseWriter, r *http.Request, key string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(`{"error": "Rate limit exceeded", "message": "Too many requests"}`))
}

// IPKeyFunc извлекает ключ по IP адресу
func IPKeyFunc(r *http.Request) string {
	return DefaultKeyFunc(r)
}

// UserKeyFunc извлекает ключ по пользователю (если есть аутентификация)
func UserKeyFunc(r *http.Request) string {
	// В реальном приложении здесь можно извлекать user ID из JWT или сессии
	userID := r.Header.Get("X-User-ID")
	if userID != "" {
		return "user:" + userID
	}

	// Fallback на IP если пользователь не определен
	return "ip:" + DefaultKeyFunc(r)
}

// PathKeyFunc извлекает ключ по пути и IP
func PathKeyFunc(r *http.Request) string {
	ip := DefaultKeyFunc(r)
	path := r.URL.Path

	// Группируем похожие пути
	if strings.HasPrefix(path, "/api/orders/") {
		path = "/api/orders/*"
	} else if strings.HasPrefix(path, "/api/") {
		path = "/api/*"
	}

	return ip + ":" + path
}

// CustomOnLimit создает кастомный обработчик с кастомным сообщением
func CustomOnLimit(message string) OnLimitFunc {
	return func(w http.ResponseWriter, r *http.Request, key string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "Rate limit exceeded", "message": "` + message + `"}`))
	}
}

// CustomOnLimitWithRetry создает обработчик с информацией о retry-after
func CustomOnLimitWithRetry(message string, retryAfter time.Duration) OnLimitFunc {
	return func(w http.ResponseWriter, r *http.Request, key string) {
		retrySeconds := int(retryAfter.Seconds())
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "Rate limit exceeded", "message": "` + message + `", "retry_after": ` + strconv.Itoa(retrySeconds) + `}`))
	}
}
