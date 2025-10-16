package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	h := New()
	if h == nil {
		t.Error("Health is nil")
	}
	if h.checkers == nil {
		t.Error("checkers slice is nil")
	}
	if len(h.checkers) != 0 {
		t.Error("checkers slice should be empty")
	}
}

func TestAddChecker(t *testing.T) {
	h := New()
	checker := &mockChecker{name: "test", shouldFail: false}

	h.AddChecker(checker)

	if len(h.checkers) != 1 {
		t.Errorf("Expected 1 checker, got %d", len(h.checkers))
	}
	if h.checkers[0] != checker {
		t.Error("Added checker is not the same")
	}
}

func TestCheck(t *testing.T) {
	h := New()

	// Добавляем checker'ы
	h.AddChecker(&mockChecker{name: "healthy", shouldFail: false})
	h.AddChecker(&mockChecker{name: "unhealthy", shouldFail: true})

	ctx := context.Background()
	results := h.Check(ctx)

	// Проверяем результаты
	if results["overall"] != "unhealthy" {
		t.Errorf("Expected overall status 'unhealthy', got %v", results["overall"])
	}

	// Проверяем healthy checker
	healthyCheck, ok := results["healthy"].(map[string]interface{})
	if !ok {
		t.Error("Healthy check result is not a map")
	}
	if healthyCheck["status"] != "healthy" {
		t.Errorf("Expected healthy status 'healthy', got %v", healthyCheck["status"])
	}

	// Проверяем unhealthy checker
	unhealthyCheck, ok := results["unhealthy"].(map[string]interface{})
	if !ok {
		t.Error("Unhealthy check result is not a map")
	}
	if unhealthyCheck["status"] != "unhealthy" {
		t.Errorf("Expected unhealthy status 'unhealthy', got %v", unhealthyCheck["status"])
	}
}

func TestHandler(t *testing.T) {
	h := New()
	h.AddChecker(&mockChecker{name: "test", shouldFail: false})

	handler := h.Handler()
	if handler == nil {
		t.Error("Handler is nil")
	}

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Проверяем Content-Type
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", rr.Header().Get("Content-Type"))
	}
}

func TestHandlerUnhealthy(t *testing.T) {
	h := New()
	h.AddChecker(&mockChecker{name: "test", shouldFail: true})

	handler := h.Handler()
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
}

func TestDatabaseChecker(t *testing.T) {
	checker := NewDatabaseChecker("test-db", func(ctx context.Context) error {
		return nil
	})

	if checker.Name() != "test-db" {
		t.Errorf("Expected name 'test-db', got %s", checker.Name())
	}

	ctx := context.Background()
	err := checker.Check(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestKafkaChecker(t *testing.T) {
	checker := NewKafkaChecker("test-kafka", func(ctx context.Context) error {
		return errors.New("connection failed")
	})

	if checker.Name() != "test-kafka" {
		t.Errorf("Expected name 'test-kafka', got %s", checker.Name())
	}

	ctx := context.Background()
	err := checker.Check(ctx)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestCacheChecker(t *testing.T) {
	checker := NewCacheChecker("test-cache", func(ctx context.Context) error {
		return nil
	})

	if checker.Name() != "test-cache" {
		t.Errorf("Expected name 'test-cache', got %s", checker.Name())
	}

	ctx := context.Background()
	err := checker.Check(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// mockChecker для тестирования
type mockChecker struct {
	name       string
	shouldFail bool
}

func (m *mockChecker) Check(ctx context.Context) error {
	time.Sleep(10 * time.Millisecond) // Имитируем работу
	if m.shouldFail {
		return errors.New("mock error")
	}
	return nil
}

func (m *mockChecker) Name() string {
	return m.name
}
