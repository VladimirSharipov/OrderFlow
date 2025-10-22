package circuitbreaker

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

// HTTPMiddleware HTTP middleware для circuit breaker
type HTTPMiddleware struct {
	breaker *CircuitBreaker
	logger  *logrus.Logger
	onOpen  OnOpenFunc
}

// OnOpenFunc функция вызываемая при открытии circuit breaker
type OnOpenFunc func(w http.ResponseWriter, r *http.Request)

// NewHTTPMiddleware создает новый HTTP middleware для circuit breaker
func NewHTTPMiddleware(config Config, logger *logrus.Logger) *HTTPMiddleware {
	breaker := New(config)

	return &HTTPMiddleware{
		breaker: breaker,
		logger:  logger,
		onOpen:  DefaultOnOpen,
	}
}

// WithOnOpen устанавливает функцию для обработки открытого состояния
func (m *HTTPMiddleware) WithOnOpen(onOpen OnOpenFunc) *HTTPMiddleware {
	m.onOpen = onOpen
	return m
}

// WithStateChangeCallback устанавливает callback для изменения состояния
func (m *HTTPMiddleware) WithStateChangeCallback(callback func(from, to State)) *HTTPMiddleware {
	m.breaker.WithStateChangeCallback(callback)
	return m
}

// Handler возвращает HTTP handler с circuit breaker
func (m *HTTPMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Создаем обертку для HTTP handler
		handler := func() (interface{}, error) {
			// Создаем ResponseWriter для перехвата статуса ответа
			rr := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rr, r)

			// Считаем ошибкой статусы >= 500
			if rr.statusCode >= http.StatusInternalServerError {
				return nil, &HTTPError{
					StatusCode: rr.statusCode,
					Message:    http.StatusText(rr.statusCode),
				}
			}

			return nil, nil
		}

		// Выполняем с circuit breaker
		_, err := m.breaker.Execute(r.Context(), handler)

		if err != nil {
			// Проверяем является ли это ошибкой circuit breaker
			if cbErr, ok := err.(*CircuitBreakerError); ok && cbErr.State == StateOpen {
				m.logger.WithFields(logrus.Fields{
					"method":      r.Method,
					"path":        r.URL.Path,
					"remote_addr": r.RemoteAddr,
					"state":       cbErr.State.String(),
				}).Warn("Circuit breaker is open")

				m.onOpen(w, r)
				return
			}

			// Логируем другие ошибки
			m.logger.WithError(err).WithFields(logrus.Fields{
				"method":      r.Method,
				"path":        r.URL.Path,
				"remote_addr": r.RemoteAddr,
			}).Error("Handler execution failed")

			// Возвращаем ошибку сервера
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})
}

// GetStats возвращает статистику circuit breaker
func (m *HTTPMiddleware) GetStats() Stats {
	return m.breaker.GetStats()
}

// Reset сбрасывает circuit breaker
func (m *HTTPMiddleware) Reset() {
	m.breaker.Reset()
}

// responseRecorder перехватывает статус код ответа
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// HTTPError ошибка HTTP запроса
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return e.Message
}

// DefaultOnOpen обработчик по умолчанию для открытого circuit breaker
func DefaultOnOpen(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte(`{"error": "Service temporarily unavailable", "message": "Circuit breaker is open"}`))
}

// CustomOnOpen создает кастомный обработчик
func CustomOnOpen(message string, statusCode int) OnOpenFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(`{"error": "Service unavailable", "message": "` + message + `"}`))
	}
}
