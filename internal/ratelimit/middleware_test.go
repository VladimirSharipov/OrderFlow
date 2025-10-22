package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestMiddleware_Handler(t *testing.T) {
	logger := logrus.New()
	config := MiddlewareConfig{
		Requests:  5,
		Window:    time.Minute,
		Burst:     10,
		Algorithm: "token-bucket",
	}

	middleware := NewMiddleware(config, logger)

	// Создаем тестовый handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrappedHandler := middleware.Handler(handler)

	tests := []struct {
		name           string
		requests       int
		expectedStatus int
		expectHeaders  bool
	}{
		{
			name:           "single request",
			requests:       1,
			expectedStatus: http.StatusOK,
			expectHeaders:  true,
		},
		{
			name:           "multiple requests within limit",
			requests:       3,
			expectedStatus: http.StatusOK,
			expectHeaders:  true,
		},
		{
			name:           "requests exceeding limit",
			requests:       10,
			expectedStatus: http.StatusTooManyRequests,
			expectHeaders:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сбрасываем лимитер для каждого теста
			middleware.limiter.Reset("test-ip")

			lastStatus := 0

			for i := 0; i < tt.requests; i++ {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.168.1.1:8080"

				rr := httptest.NewRecorder()
				wrappedHandler.ServeHTTP(rr, req)

				lastStatus = rr.Code
			}

			if lastStatus != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, lastStatus)
			}
		})
	}
}

func TestMiddleware_WithCustomKeyFunc(t *testing.T) {
	logger := logrus.New()
	config := MiddlewareConfig{
		Requests:  2,
		Window:    time.Minute,
		Algorithm: "token-bucket",
	}

	middleware := NewMiddleware(config, logger)
	middleware.WithKeyFunc(func(r *http.Request) string {
		return "custom-key"
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Handler(handler)

	// Делаем 3 запроса с одинаковым ключом
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:8080"

		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)

		// Первые 2 должны пройти, третий должен быть отклонен
		expectedStatus := http.StatusOK
		if i == 2 {
			expectedStatus = http.StatusTooManyRequests
		}

		if rr.Code != expectedStatus {
			t.Errorf("Request %d: expected status %d, got %d", i+1, expectedStatus, rr.Code)
		}
	}
}

func TestMiddleware_WithCustomOnLimit(t *testing.T) {
	logger := logrus.New()
	config := MiddlewareConfig{
		Requests:  1,
		Window:    time.Minute,
		Algorithm: "token-bucket",
	}

	middleware := NewMiddleware(config, logger)
	middleware.WithOnLimit(func(w http.ResponseWriter, r *http.Request, key string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"custom": "error message"}`))
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Handler(handler)

	// Делаем 2 запроса - второй должен быть отклонен
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:8080"

		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)

		if i == 1 {
			// Проверяем кастомный ответ
			if rr.Code != http.StatusTooManyRequests {
				t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, rr.Code)
			}

			expectedBody := `{"custom": "error message"}`
			if rr.Body.String() != expectedBody {
				t.Errorf("Expected body %s, got %s", expectedBody, rr.Body.String())
			}
		}
	}
}

func TestKeyFuncs(t *testing.T) {
	tests := []struct {
		name    string
		keyFunc KeyFunc
		req     *http.Request
		want    string
	}{
		{
			name:    "DefaultKeyFunc",
			keyFunc: DefaultKeyFunc,
			req: &http.Request{
				RemoteAddr: "192.168.1.1:8080",
			},
			want: "192.168.1.1",
		},
		{
			name:    "IPKeyFunc",
			keyFunc: IPKeyFunc,
			req: &http.Request{
				RemoteAddr: "10.0.0.1:8080",
			},
			want: "10.0.0.1",
		},
		{
			name:    "UserKeyFunc without user",
			keyFunc: UserKeyFunc,
			req: &http.Request{
				RemoteAddr: "192.168.1.1:8080",
			},
			want: "ip:192.168.1.1",
		},
		{
			name:    "UserKeyFunc with user",
			keyFunc: UserKeyFunc,
			req: func() *http.Request {
				req := &http.Request{
					RemoteAddr: "192.168.1.1:8080",
					Header:     make(http.Header),
				}
				req.Header.Set("X-User-ID", "user123")
				return req
			}(),
			want: "user:user123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.keyFunc(tt.req)
			if got != tt.want {
				t.Errorf("KeyFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomOnLimitFunctions(t *testing.T) {
	tests := []struct {
		name            string
		onLimit         OnLimitFunc
		wantCode        int
		wantBody        string
		wantHeader      string
		wantHeaderValue string
	}{
		{
			name:     "CustomOnLimit",
			onLimit:  CustomOnLimit("Custom rate limit message"),
			wantCode: http.StatusTooManyRequests,
			wantBody: `{"error": "Rate limit exceeded", "message": "Custom rate limit message"}`,
		},
		{
			name:            "CustomOnLimitWithRetry",
			onLimit:         CustomOnLimitWithRetry("Retry message", 30*time.Second),
			wantCode:        http.StatusTooManyRequests,
			wantBody:        `{"error": "Rate limit exceeded", "message": "Retry message", "retry_after": 30}`,
			wantHeader:      "Retry-After",
			wantHeaderValue: "30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()

			tt.onLimit(rr, req, "test-key")

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
