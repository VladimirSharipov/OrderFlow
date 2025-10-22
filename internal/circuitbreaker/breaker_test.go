package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	config := DefaultConfig()
	cb := New(config)

	// Выполняем успешную функцию
	result, err := cb.Execute(context.Background(), func() (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != "success" {
		t.Errorf("Expected 'success', got %v", result)
	}

	// Проверяем что состояние остается закрытым
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state CLOSED, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	config := Config{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
		MaxRequests:      2,
	}
	cb := New(config)

	// Выполняем несколько неудачных функций
	for i := 0; i < config.FailureThreshold; i++ {
		_, err := cb.Execute(context.Background(), func() (interface{}, error) {
			return nil, errors.New("test error")
		})

		if err == nil {
			t.Errorf("Expected error on attempt %d", i+1)
		}
	}

	// Проверяем что circuit breaker открылся
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state OPEN, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_Execute_OpenState(t *testing.T) {
	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
		MaxRequests:      1,
	}
	cb := New(config)

	// Открываем circuit breaker
	cb.Execute(context.Background(), func() (interface{}, error) {
		return nil, errors.New("test error")
	})

	// Проверяем что запросы блокируются
	_, err := cb.Execute(context.Background(), func() (interface{}, error) {
		return "should not execute", nil
	})

	if err == nil {
		t.Error("Expected error when circuit breaker is open")
	}

	if cbErr, ok := err.(*CircuitBreakerError); !ok {
		t.Errorf("Expected CircuitBreakerError, got %T", err)
	} else if cbErr.State != StateOpen {
		t.Errorf("Expected OPEN state, got %s", cbErr.State)
	}
}

func TestCircuitBreaker_Execute_HalfOpenState(t *testing.T) {
	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      3,
	}
	cb := New(config)

	// Открываем circuit breaker
	cb.Execute(context.Background(), func() (interface{}, error) {
		return nil, errors.New("test error")
	})

	// Ждем timeout
	time.Sleep(100 * time.Millisecond)

	// Проверяем что состояние изменилось на полуоткрытое при попытке выполнения
	// (состояние меняется только при вызове CanExecute)
	allowed := cb.CanExecute()
	if !allowed {
		t.Error("Expected to be allowed to execute after timeout")
	}

	if cb.GetState() != StateHalfOpen {
		t.Errorf("Expected state HALF_OPEN, got %s", cb.GetState())
	}

	// Выполняем успешные запросы
	for i := 0; i < config.SuccessThreshold; i++ {
		result, err := cb.Execute(context.Background(), func() (interface{}, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("Expected no error on attempt %d, got %v", i+1, err)
		}

		if result != "success" {
			t.Errorf("Expected 'success', got %v", result)
		}
	}

	// Проверяем что circuit breaker закрылся
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state CLOSED, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_Execute_HalfOpenFailure(t *testing.T) {
	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      2,
	}
	cb := New(config)

	// Открываем circuit breaker
	cb.Execute(context.Background(), func() (interface{}, error) {
		return nil, errors.New("test error")
	})

	// Ждем timeout
	time.Sleep(100 * time.Millisecond)

	// Выполняем неудачный запрос в полуоткрытом состоянии
	_, err := cb.Execute(context.Background(), func() (interface{}, error) {
		return nil, errors.New("test error")
	})

	if err == nil {
		t.Error("Expected error")
	}

	// Проверяем что circuit breaker снова открылся
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state OPEN, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_Execute_MaxRequests(t *testing.T) {
	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 3, // Увеличиваем чтобы circuit breaker не закрылся сразу
		Timeout:          50 * time.Millisecond,
		MaxRequests:      2,
	}
	cb := New(config)

	// Открываем circuit breaker
	cb.Execute(context.Background(), func() (interface{}, error) {
		return nil, errors.New("test error")
	})

	// Ждем timeout
	time.Sleep(100 * time.Millisecond)

	// Выполняем максимальное количество запросов
	for i := 0; i < config.MaxRequests; i++ {
		_, err := cb.Execute(context.Background(), func() (interface{}, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("Expected no error on attempt %d, got %v", i+1, err)
		}
	}

	// Следующий запрос должен быть заблокирован
	_, err := cb.Execute(context.Background(), func() (interface{}, error) {
		return "should not execute", nil
	})

	if err == nil {
		t.Error("Expected error when max requests exceeded")
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	config := Config{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
		MaxRequests:      2,
	}
	cb := New(config)

	// Выполняем несколько операций
	cb.Execute(context.Background(), func() (interface{}, error) {
		return nil, errors.New("error")
	})

	cb.Execute(context.Background(), func() (interface{}, error) {
		return "success", nil
	})

	stats := cb.GetStats()

	if stats.State != StateClosed {
		t.Errorf("Expected state CLOSED, got %s", stats.State)
	}

	if stats.FailureCount != 0 {
		t.Errorf("Expected 0 failures (reset by success), got %d", stats.FailureCount)
	}
}

func TestCircuitBreaker_StateChangeCallback(t *testing.T) {
	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      1,
	}

	var stateChanges []StateChange
	cb := New(config).WithStateChangeCallback(func(from, to State) {
		stateChanges = append(stateChanges, StateChange{From: from, To: to})
	})

	// Открываем circuit breaker
	cb.Execute(context.Background(), func() (interface{}, error) {
		return nil, errors.New("test error")
	})

	// Ждем timeout
	time.Sleep(100 * time.Millisecond)

	// Выполняем успешный запрос
	cb.Execute(context.Background(), func() (interface{}, error) {
		return "success", nil
	})

	// Проверяем что callbacks были вызваны
	if len(stateChanges) < 2 {
		t.Errorf("Expected at least 2 state changes, got %d", len(stateChanges))
	}

	// Проверяем переходы состояний
	expectedTransitions := []StateChange{
		{From: StateClosed, To: StateOpen},
		{From: StateOpen, To: StateHalfOpen},
		{From: StateHalfOpen, To: StateClosed},
	}

	for i, expected := range expectedTransitions {
		if i >= len(stateChanges) {
			break
		}

		actual := stateChanges[i]
		if actual.From != expected.From || actual.To != expected.To {
			t.Errorf("State change %d: expected %v -> %v, got %v -> %v",
				i, expected.From, expected.To, actual.From, actual.To)
		}
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
		MaxRequests:      1,
	}
	cb := New(config)

	// Открываем circuit breaker
	cb.Execute(context.Background(), func() (interface{}, error) {
		return nil, errors.New("test error")
	})

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state OPEN, got %s", cb.GetState())
	}

	// Сбрасываем
	cb.Reset()

	// Проверяем что состояние сброшено
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state CLOSED after reset, got %s", cb.GetState())
	}

	stats := cb.GetStats()
	if stats.FailureCount != 0 {
		t.Errorf("Expected 0 failures after reset, got %d", stats.FailureCount)
	}
}

func TestIsCircuitBreakerOpen(t *testing.T) {
	config := Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
		MaxRequests:      1,
	}
	cb := New(config)

	// Открываем circuit breaker
	cb.Execute(context.Background(), func() (interface{}, error) {
		return nil, errors.New("test error")
	})

	// Проверяем что запросы блокируются
	_, err := cb.Execute(context.Background(), func() (interface{}, error) {
		return "should not execute", nil
	})

	if !IsCircuitBreakerOpen(err) {
		t.Error("Expected circuit breaker to be open")
	}

	// Проверяем обычную ошибку
	if IsCircuitBreakerOpen(errors.New("regular error")) {
		t.Error("Expected regular error not to be circuit breaker open")
	}
}

// StateChange представляет изменение состояния
type StateChange struct {
	From State
	To   State
}
