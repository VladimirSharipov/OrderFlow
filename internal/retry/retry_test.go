package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"wbtest/internal/config"
)

func TestRetryService_ExecuteWithRetry(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.RetryConfig
		operation      func() error
		expectedErrors int
		wantErr        bool
	}{
		{
			name: "success on first attempt",
			config: &config.RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 1 * time.Millisecond,
				MaxDelay:     10 * time.Millisecond,
				Multiplier:   2.0,
			},
			operation: func() error {
				return nil
			},
			expectedErrors: 1,
			wantErr:        false,
		},
		{
			name: "failure after max attempts",
			config: &config.RetryConfig{
				MaxAttempts:  2,
				InitialDelay: 1 * time.Millisecond,
				MaxDelay:     10 * time.Millisecond,
				Multiplier:   2.0,
			},
			operation: func() error {
				return errors.New("persistent error")
			},
			expectedErrors: 2,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewRetryService(tt.config)

			attemptCount := 0
			operation := func() error {
				attemptCount++
				return tt.operation()
			}

			err := service.ExecuteWithRetry(operation)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteWithRetry() error = %v, wantErr %v", err, tt.wantErr)
			}

			if attemptCount != tt.expectedErrors {
				t.Errorf("Expected %d attempts, got %d", tt.expectedErrors, attemptCount)
			}
		})
	}
}

func TestRetryService_ExecuteWithRetryContext(t *testing.T) {
	config := &config.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
	}

	service := NewRetryService(config)

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		operation := func() error {
			return errors.New("operation error")
		}

		// Cast to concrete type to access ExecuteWithRetryContext
		retryService := service.(*RetryService)
		err := retryService.ExecuteWithRetryContext(ctx, operation)
		if err == nil {
			t.Error("Expected context cancellation error")
		}
	})

	t.Run("success with context", func(t *testing.T) {
		ctx := context.Background()

		operation := func() error {
			return nil
		}

		// Cast to concrete type to access ExecuteWithRetryContext
		retryService := service.(*RetryService)
		err := retryService.ExecuteWithRetryContext(ctx, operation)
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
	})
}

func TestRetryService_calculateDelay(t *testing.T) {
	config := &config.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	service := NewRetryService(config)

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 10 * time.Millisecond},  // 10ms * 2^0 = 10ms
		{2, 20 * time.Millisecond},  // 10ms * 2^1 = 20ms
		{3, 40 * time.Millisecond},  // 10ms * 2^2 = 40ms
		{4, 80 * time.Millisecond},  // 10ms * 2^3 = 80ms
		{5, 100 * time.Millisecond}, // Capped at MaxDelay
	}

	for _, tt := range tests {
		t.Run("attempt "+string(rune(tt.attempt)), func(t *testing.T) {
			// Cast to concrete type to access calculateDelay
			retryService := service.(*RetryService)
			delay := retryService.calculateDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("calculateDelay(%d) = %v, want %v", tt.attempt, delay, tt.expected)
			}
		})
	}
}
