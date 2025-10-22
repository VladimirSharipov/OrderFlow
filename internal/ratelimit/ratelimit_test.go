package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	config := Config{
		Requests: 10,
		Window:   time.Minute,
		Burst:    15,
	}

	limiter := NewTokenBucket(config)

	tests := []struct {
		name     string
		key      string
		requests int
		wantErr  bool
	}{
		{
			name:     "single request",
			key:      "test-key",
			requests: 1,
			wantErr:  false,
		},
		{
			name:     "multiple requests within limit",
			key:      "test-key-2",
			requests: 5,
			wantErr:  false,
		},
		{
			name:     "requests exceeding limit",
			key:      "test-key-3",
			requests: 20,
			wantErr:  false, // Не должно быть ошибки, но некоторые запросы должны быть отклонены
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowedCount := 0

			for i := 0; i < tt.requests; i++ {
				allowed, err := limiter.Allow(context.Background(), tt.key)
				if err != nil {
					if !tt.wantErr {
						t.Errorf("Allow() error = %v, wantErr %v", err, tt.wantErr)
					}
					return
				}

				if allowed {
					allowedCount++
				}
			}

			// Проверяем что не все запросы были разрешены если превышен лимит
			if tt.requests > config.Requests && allowedCount == tt.requests {
				t.Errorf("All requests were allowed, but limit should be %d", config.Requests)
			}
		})
	}
}

func TestTokenBucket_Stats(t *testing.T) {
	config := Config{
		Requests: 5,
		Window:   time.Minute,
		Burst:    10,
	}

	limiter := NewTokenBucket(config)
	key := "stats-test"

	// Выполняем несколько запросов
	for i := 0; i < 3; i++ {
		limiter.Allow(context.Background(), key)
	}

	stats := limiter.Stats(key)
	if stats.Allowed != 3 {
		t.Errorf("Expected 3 allowed requests, got %d", stats.Allowed)
	}

	if stats.ResetTime.IsZero() {
		t.Error("ResetTime should be set")
	}
}

func TestTokenBucket_Reset(t *testing.T) {
	config := Config{
		Requests: 2,
		Window:   time.Minute,
		Burst:    5,
	}

	limiter := NewTokenBucket(config)
	key := "reset-test"

	// Исчерпываем лимит
	for i := 0; i < 3; i++ {
		limiter.Allow(context.Background(), key)
	}

	stats := limiter.Stats(key)
	if stats.Allowed != 2 {
		t.Errorf("Expected 2 allowed requests before reset, got %d", stats.Allowed)
	}

	// Сбрасываем
	limiter.Reset(key)

	// Проверяем что можно снова делать запросы
	allowed, err := limiter.Allow(context.Background(), key)
	if err != nil {
		t.Errorf("Allow() after reset error = %v", err)
	}

	if !allowed {
		t.Error("Request should be allowed after reset")
	}
}

func TestFixedWindow_Allow(t *testing.T) {
	config := Config{
		Requests: 5,
		Window:   time.Minute,
	}

	limiter := NewFixedWindow(config)

	tests := []struct {
		name     string
		key      string
		requests int
		wantErr  bool
	}{
		{
			name:     "single request",
			key:      "fw-test-key",
			requests: 1,
			wantErr:  false,
		},
		{
			name:     "multiple requests within limit",
			key:      "fw-test-key-2",
			requests: 3,
			wantErr:  false,
		},
		{
			name:     "requests exceeding limit",
			key:      "fw-test-key-3",
			requests: 10,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowedCount := 0

			for i := 0; i < tt.requests; i++ {
				allowed, err := limiter.Allow(context.Background(), tt.key)
				if err != nil {
					if !tt.wantErr {
						t.Errorf("Allow() error = %v, wantErr %v", err, tt.wantErr)
					}
					return
				}

				if allowed {
					allowedCount++
				}
			}

			// Проверяем что не все запросы были разрешены если превышен лимит
			if tt.requests > config.Requests && allowedCount == tt.requests {
				t.Errorf("All requests were allowed, but limit should be %d", config.Requests)
			}
		})
	}
}

func TestFixedWindow_Stats(t *testing.T) {
	config := Config{
		Requests: 3,
		Window:   time.Minute,
	}

	limiter := NewFixedWindow(config)
	key := "fw-stats-test"

	// Выполняем несколько запросов
	for i := 0; i < 2; i++ {
		limiter.Allow(context.Background(), key)
	}

	stats := limiter.Stats(key)
	if stats.Allowed != 2 {
		t.Errorf("Expected 2 allowed requests, got %d", stats.Allowed)
	}

	if stats.ResetTime.IsZero() {
		t.Error("ResetTime should be set")
	}
}

func TestNewRateLimiter(t *testing.T) {
	config := Config{
		Requests: 10,
		Window:   time.Minute,
		Burst:    15,
	}

	tests := []struct {
		name      string
		algorithm string
		wantType  string
	}{
		{
			name:      "token bucket",
			algorithm: "token-bucket",
			wantType:  "*ratelimit.TokenBucket",
		},
		{
			name:      "fixed window",
			algorithm: "fixed-window",
			wantType:  "*ratelimit.FixedWindow",
		},
		{
			name:      "default algorithm",
			algorithm: "unknown",
			wantType:  "*ratelimit.TokenBucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewRateLimiter(config, tt.algorithm)

			actualType := fmt.Sprintf("%T", limiter)
			if actualType != tt.wantType {
				t.Errorf("NewRateLimiter() = %v, want %v", actualType, tt.wantType)
			}
		})
	}
}
