package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNew(t *testing.T) {
	m := New()
	if m == nil {
		t.Error("Metrics is nil")
	}

	// Проверяем что все метрики созданы
	if m.HTTPRequestsTotal == nil {
		t.Error("HTTPRequestsTotal is nil")
	}
	if m.HTTPRequestDuration == nil {
		t.Error("HTTPRequestDuration is nil")
	}
	if m.KafkaMessagesConsumed == nil {
		t.Error("KafkaMessagesConsumed is nil")
	}
	if m.OrdersProcessed == nil {
		t.Error("OrdersProcessed is nil")
	}
	if m.RetryAttempts == nil {
		t.Error("RetryAttempts is nil")
	}
	if m.DLQMessagesSent == nil {
		t.Error("DLQMessagesSent is nil")
	}
	if m.DatabaseConnections == nil {
		t.Error("DatabaseConnections is nil")
	}
}

func TestHTTPMiddleware(t *testing.T) {
	// Создаем новый registry для теста чтобы избежать конфликтов
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg

	m := New()

	// Создаем тестовый handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Обертываем в middleware
	wrappedHandler := m.HTTPMiddleware(handler)

	// Создаем тестовый запрос
	req := httptest.NewRequest("GET", "/test", nil)
	req.ContentLength = 100

	// Создаем ResponseRecorder
	rr := httptest.NewRecorder()

	// Выполняем запрос
	wrappedHandler.ServeHTTP(rr, req)

	// Проверяем результат
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", rr.Body.String())
	}
}

func TestHandler(t *testing.T) {
	// Создаем новый registry для теста чтобы избежать конфликтов
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg

	m := New()
	handler := m.Handler()

	if handler == nil {
		t.Error("Handler is nil")
	}

	// Создаем тестовый запрос
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	// Выполняем запрос
	handler.ServeHTTP(rr, req)

	// Проверяем что ответ содержит метрики
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	if body == "" {
		t.Error("Response body is empty")
	}

	// Проверяем что в ответе есть Prometheus метрики
	if !contains(body, "# HELP") {
		t.Error("Response doesn't contain Prometheus help text")
	}
}

func TestResponseWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: 200}

	// Тестируем WriteHeader
	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rw.statusCode)
	}

	// Тестируем Write
	data := []byte("test data")
	n, err := rw.Write(data)
	if err != nil {
		t.Errorf("Write error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected written %d bytes, got %d", len(data), n)
	}

	if rw.size != len(data) {
		t.Errorf("Expected size %d, got %d", len(data), rw.size)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
