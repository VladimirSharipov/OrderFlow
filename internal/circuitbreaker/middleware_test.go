package circuitbreaker

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestHTTPMiddleware_Handler_Success(t *testing.T) {
	logger := logrus.New()
	config := DefaultConfig()
	middleware := NewHTTPMiddleware(config, logger)

	// Создаем тестовый handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrappedHandler := middleware.Handler(handler)

	// Выполняем запрос
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", rr.Body.String())
	}
}

func TestHTTPMiddleware_Handler_ServerError(t *testing.T) {
	logger := logrus.New()
	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
		MaxRequests:      1,
	}
	middleware := NewHTTPMiddleware(config, logger)

	// Создаем тестовый handler который возвращает ошибку сервера
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})

	wrappedHandler := middleware.Handler(handler)

	// Выполняем запрос
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	// Первый запрос должен пройти, но вернуть ошибку сервера
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	// Следующий запрос должен быть заблокирован circuit breaker
	rr2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr2, req)

	if rr2.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, rr2.Code)
	}
}

func TestHTTPMiddleware_Handler_CustomOnOpen(t *testing.T) {
	logger := logrus.New()
	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
		MaxRequests:      1,
	}
	middleware := NewHTTPMiddleware(config, logger)
	middleware.WithOnOpen(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "Custom error message"}`))
	})

	// Создаем тестовый handler который возвращает ошибку сервера
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	wrappedHandler := middleware.Handler(handler)

	// Выполняем запрос для открытия circuit breaker
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Следующий запрос должен использовать кастомный обработчик
	rr2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr2, req)

	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, rr2.Code)
	}

	expectedBody := `{"error": "Custom error message"}`
	if rr2.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, rr2.Body.String())
	}
}

func TestHTTPMiddleware_GetStats(t *testing.T) {
	logger := logrus.New()
	config := DefaultConfig()
	middleware := NewHTTPMiddleware(config, logger)

	// Проверяем начальную статистику
	stats := middleware.GetStats()
	if stats.State != StateClosed {
		t.Errorf("Expected initial state CLOSED, got %s", stats.State)
	}

	if stats.FailureCount != 0 {
		t.Errorf("Expected initial failure count 0, got %d", stats.FailureCount)
	}
}

func TestHTTPMiddleware_Reset(t *testing.T) {
	logger := logrus.New()
	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
		MaxRequests:      1,
	}
	middleware := NewHTTPMiddleware(config, logger)

	// Создаем тестовый handler который возвращает ошибку сервера
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	wrappedHandler := middleware.Handler(handler)

	// Открываем circuit breaker
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Проверяем что circuit breaker открылся
	stats := middleware.GetStats()
	if stats.State != StateOpen {
		t.Errorf("Expected state OPEN, got %s", stats.State)
	}

	// Сбрасываем
	middleware.Reset()

	// Проверяем что состояние сброшено
	stats = middleware.GetStats()
	if stats.State != StateClosed {
		t.Errorf("Expected state CLOSED after reset, got %s", stats.State)
	}
}

func TestHTTPMiddleware_StateChangeCallback(t *testing.T) {
	logger := logrus.New()
	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      1,
	}

	var stateChanges []StateChange
	middleware := NewHTTPMiddleware(config, logger)
	middleware.WithStateChangeCallback(func(from, to State) {
		stateChanges = append(stateChanges, StateChange{From: from, To: to})
	})

	// Создаем тестовый handler который возвращает ошибку сервера
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	wrappedHandler := middleware.Handler(handler)

	// Открываем circuit breaker
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Ждем timeout
	time.Sleep(100 * time.Millisecond)

	// Выполняем успешный запрос
	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedSuccessHandler := middleware.Handler(successHandler)
	wrappedSuccessHandler.ServeHTTP(rr, req)

	// Проверяем что callbacks были вызваны
	if len(stateChanges) < 2 {
		t.Errorf("Expected at least 2 state changes, got %d", len(stateChanges))
	}
}

func TestCustomOnOpenFunctions(t *testing.T) {
	tests := []struct {
		name            string
		onOpen          OnOpenFunc
		wantCode        int
		wantBody        string
		wantHeader      string
		wantHeaderValue string
	}{
		{
			name:     "DefaultOnOpen",
			onOpen:   DefaultOnOpen,
			wantCode: http.StatusServiceUnavailable,
			wantBody: `{"error": "Service temporarily unavailable", "message": "Circuit breaker is open"}`,
		},
		{
			name:     "CustomOnOpen",
			onOpen:   CustomOnOpen("Custom error", http.StatusTooManyRequests),
			wantCode: http.StatusTooManyRequests,
			wantBody: `{"error": "Service unavailable", "message": "Custom error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()

			tt.onOpen(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, rr.Code)
			}

			if rr.Body.String() != tt.wantBody {
				t.Errorf("Expected body %s, got %s", tt.wantBody, rr.Body.String())
			}

			if tt.wantHeader != "" {
				gotHeader := rr.Header().Get(tt.wantHeader)
				if gotHeader != tt.wantHeaderValue {
					t.Errorf("Expected header %s: %s, got %s", tt.wantHeader, tt.wantHeaderValue, gotHeader)
				}
			}
		})
	}
}

func TestResponseRecorder(t *testing.T) {
	rr := httptest.NewRecorder()

	recorder := &responseRecorder{
		ResponseWriter: rr,
		statusCode:     http.StatusOK,
	}

	// Проверяем начальный статус
	if recorder.statusCode != http.StatusOK {
		t.Errorf("Expected initial status %d, got %d", http.StatusOK, recorder.statusCode)
	}

	// Устанавливаем новый статус
	recorder.WriteHeader(http.StatusNotFound)

	if recorder.statusCode != http.StatusNotFound {
		t.Errorf("Expected status %d after WriteHeader, got %d", http.StatusNotFound, recorder.statusCode)
	}

	// Проверяем что статус передался в оригинальный ResponseWriter
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected underlying ResponseWriter status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHTTPError(t *testing.T) {
	err := &HTTPError{
		StatusCode: http.StatusInternalServerError,
		Message:    "Internal Server Error",
	}

	expectedError := "Internal Server Error"
	if err.Error() != expectedError {
		t.Errorf("Expected error message %s, got %s", expectedError, err.Error())
	}
}
