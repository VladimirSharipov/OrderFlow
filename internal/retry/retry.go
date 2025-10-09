package retry

import (
	"context"
	"fmt"
	"math"
	"time"

	"wbtest/internal/config"
	"wbtest/internal/interfaces"
)

type RetryService struct {
	config *config.RetryConfig
}

func NewRetryService(cfg *config.RetryConfig) interfaces.RetryService {
	return &RetryService{
		config: cfg,
	}
}

func (r *RetryService) ExecuteWithRetry(operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		if err := operation(); err != nil {
			lastErr = err

			// Если это последняя попытка, возвращаем ошибку
			if attempt == r.config.MaxAttempts {
				return fmt.Errorf("operation failed after %d attempts, last error: %w", r.config.MaxAttempts, lastErr)
			}

			// Вычисляем задержку с экспоненциальным backoff
			delay := r.calculateDelay(attempt)

			// Ждем перед следующей попыткой
			time.Sleep(delay)
			continue
		}

		// Операция успешна
		return nil
	}

	return lastErr
}

func (r *RetryService) calculateDelay(attempt int) time.Duration {
	// Экспоненциальный backoff: delay = initialDelay * (multiplier ^ (attempt - 1))
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.Multiplier, float64(attempt-1))

	// Ограничиваем максимальной задержкой
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	return time.Duration(delay)
}

// ExecuteWithRetryContext выполняет операцию с retry и контекстом
func (r *RetryService) ExecuteWithRetryContext(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		// Проверяем контекст перед каждой попыткой
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := operation(); err != nil {
			lastErr = err

			// Если это последняя попытка, возвращаем ошибку
			if attempt == r.config.MaxAttempts {
				return fmt.Errorf("operation failed after %d attempts, last error: %w", r.config.MaxAttempts, lastErr)
			}

			// Вычисляем задержку
			delay := r.calculateDelay(attempt)

			// Ждем с возможностью отмены через контекст
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Операция успешна
		return nil
	}

	return lastErr
}
