package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// State состояние circuit breaker
type State int

const (
	StateClosed   State = iota // Закрыт - запросы проходят
	StateOpen                  // Открыт - запросы блокируются
	StateHalfOpen              // Полуоткрыт - тестовые запросы проходят
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config конфигурация circuit breaker
type Config struct {
	// FailureThreshold количество ошибок для открытия circuit breaker
	FailureThreshold int
	// SuccessThreshold количество успешных запросов для закрытия circuit breaker
	SuccessThreshold int
	// Timeout время ожидания в открытом состоянии
	Timeout time.Duration
	// MaxRequests максимальное количество запросов в полуоткрытом состоянии
	MaxRequests int
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          60 * time.Second,
		MaxRequests:      3,
	}
}

// CircuitBreaker реализация circuit breaker паттерна
type CircuitBreaker struct {
	config        Config
	state         State
	failureCount  int
	successCount  int
	requestCount  int
	lastFailTime  time.Time
	nextAttempt   time.Time
	mutex         sync.RWMutex
	onStateChange func(from, to State)
}

// New создает новый circuit breaker
func New(config Config) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 3
	}
	if config.Timeout <= 0 {
		config.Timeout = 60 * time.Second
	}
	if config.MaxRequests <= 0 {
		config.MaxRequests = 3
	}

	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// WithStateChangeCallback устанавливает callback для изменения состояния
func (cb *CircuitBreaker) WithStateChangeCallback(callback func(from, to State)) *CircuitBreaker {
	cb.onStateChange = callback
	return cb
}

// Execute выполняет функцию с защитой circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	// Проверяем и обновляем состояние circuit breaker
	allowed, stateChanged := cb.checkAndUpdateState()
	if !allowed {
		return nil, &CircuitBreakerError{
			State: cb.getState(),
			Err:   fmt.Errorf("circuit breaker is %s", cb.getState()),
		}
	}

	// Выполняем функцию
	result, err := fn()

	// Обновляем состояние на основе результата
	cb.recordResult(err == nil, stateChanged)

	return result, err
}

// ExecuteAsync выполняет функцию асинхронно
func (cb *CircuitBreaker) ExecuteAsync(ctx context.Context, fn func() (interface{}, error)) <-chan Result {
	resultChan := make(chan Result, 1)

	go func() {
		defer close(resultChan)

		result, err := cb.Execute(ctx, fn)
		resultChan <- Result{
			Data: result,
			Err:  err,
		}
	}()

	return resultChan
}

// GetState возвращает текущее состояние
func (cb *CircuitBreaker) GetState() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// CanExecute проверяет можно ли выполнить запрос (и обновляет состояние если нужно)
func (cb *CircuitBreaker) CanExecute() bool {
	allowed, _ := cb.checkAndUpdateState()
	return allowed
}

// GetStats возвращает статистику circuit breaker
func (cb *CircuitBreaker) GetStats() Stats {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return Stats{
		State:        cb.state,
		FailureCount: cb.failureCount,
		SuccessCount: cb.successCount,
		RequestCount: cb.requestCount,
		LastFailTime: cb.lastFailTime,
		NextAttempt:  cb.nextAttempt,
	}
}

// Reset сбрасывает состояние circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.requestCount = 0
	cb.lastFailTime = time.Time{}
	cb.nextAttempt = time.Time{}

	if cb.onStateChange != nil {
		cb.onStateChange(oldState, cb.state)
	}
}

// checkAndUpdateState проверяет можно ли выполнить запрос и обновляет состояние если нужно
func (cb *CircuitBreaker) checkAndUpdateState() (bool, bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	stateChanged := false

	switch cb.state {
	case StateClosed:
		return true, false
	case StateOpen:
		// Проверяем истек ли timeout
		if now.After(cb.nextAttempt) {
			oldState := cb.state
			cb.state = StateHalfOpen
			cb.requestCount = 0
			cb.successCount = 0
			stateChanged = true

			if cb.onStateChange != nil {
				cb.onStateChange(oldState, cb.state)
			}
			return true, stateChanged
		}
		return false, false
	case StateHalfOpen:
		// Проверяем не превышен ли лимит запросов
		return cb.requestCount < cb.config.MaxRequests, false
	default:
		return false, false
	}
}

// canExecute проверяет можно ли выполнить запрос (для обратной совместимости)
func (cb *CircuitBreaker) canExecute() bool {
	allowed, _ := cb.checkAndUpdateState()
	return allowed
}

// recordResult записывает результат выполнения
func (cb *CircuitBreaker) recordResult(success bool, stateChanged bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case StateClosed:
		if success {
			cb.failureCount = 0 // Сбрасываем счетчик ошибок при успехе
		} else {
			cb.failureCount++
			cb.lastFailTime = time.Now()

			// Проверяем нужно ли открыть circuit breaker
			if cb.failureCount >= cb.config.FailureThreshold {
				oldState := cb.state
				cb.state = StateOpen
				cb.nextAttempt = time.Now().Add(cb.config.Timeout)

				if cb.onStateChange != nil {
					cb.onStateChange(oldState, cb.state)
				}
			}
		}
	case StateHalfOpen:
		cb.requestCount++ // Увеличиваем счетчик запросов

		if success {
			cb.successCount++

			// Проверяем нужно ли закрыть circuit breaker
			if cb.successCount >= cb.config.SuccessThreshold {
				oldState := cb.state
				cb.state = StateClosed
				cb.failureCount = 0
				cb.successCount = 0
				cb.requestCount = 0

				if cb.onStateChange != nil {
					cb.onStateChange(oldState, cb.state)
				}
			}
		} else {
			// При ошибке в полуоткрытом состоянии снова открываем
			oldState := cb.state
			cb.state = StateOpen
			cb.nextAttempt = time.Now().Add(cb.config.Timeout)
			cb.failureCount++
			cb.lastFailTime = time.Now()

			if cb.onStateChange != nil {
				cb.onStateChange(oldState, cb.state)
			}
		}
	}
}

// getState возвращает состояние (без блокировки)
func (cb *CircuitBreaker) getState() State {
	return cb.state
}

// Stats статистика circuit breaker
type Stats struct {
	State        State
	FailureCount int
	SuccessCount int
	RequestCount int
	LastFailTime time.Time
	NextAttempt  time.Time
}

// Result результат асинхронного выполнения
type Result struct {
	Data interface{}
	Err  error
}

// CircuitBreakerError ошибка circuit breaker
type CircuitBreakerError struct {
	State State
	Err   error
}

func (e *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker %s: %v", e.State, e.Err)
}

func (e *CircuitBreakerError) Unwrap() error {
	return e.Err
}

// IsCircuitBreakerOpen проверяет является ли ошибка результатом открытого circuit breaker
func IsCircuitBreakerOpen(err error) bool {
	if cbErr, ok := err.(*CircuitBreakerError); ok {
		return cbErr.State == StateOpen
	}
	return false
}
