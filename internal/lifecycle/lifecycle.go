package lifecycle

import (
	"context"
	"sync"
	"time"
)

// Service интерфейс для сервисов с жизненным циклом
type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Name() string
}

// Manager управляет жизненным циклом сервисов
type Manager struct {
	services []Service
	mu       sync.RWMutex
}

// New создает новый менеджер жизненного цикла
func New() *Manager {
	return &Manager{
		services: make([]Service, 0),
	}
}

// Register регистрирует сервис
func (m *Manager) Register(service Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services = append(m.services, service)
}

// Start запускает все зарегистрированные сервисы
func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	services := make([]Service, len(m.services))
	copy(services, m.services)
	m.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(services))

	// Запускаем все сервисы параллельно
	for _, service := range services {
		wg.Add(1)
		go func(svc Service) {
			defer wg.Done()
			if err := svc.Start(ctx); err != nil {
				errChan <- err
			}
		}(service)
	}

	// Ждем завершения запуска всех сервисов
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Проверяем ошибки
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// Stop останавливает все сервисы
func (m *Manager) Stop(ctx context.Context, timeout time.Duration) error {
	m.mu.RLock()
	services := make([]Service, len(m.services))
	copy(services, m.services)
	m.mu.RUnlock()

	// Создаем контекст с таймаутом
	stopCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var wg sync.WaitGroup
	errChan := make(chan error, len(services))

	// Останавливаем все сервисы параллельно
	for _, service := range services {
		wg.Add(1)
		go func(svc Service) {
			defer wg.Done()
			if err := svc.Stop(stopCtx); err != nil {
				errChan <- err
			}
		}(service)
	}

	// Ждем завершения остановки всех сервисов
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Собираем ошибки
	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	// Возвращаем первую ошибку если есть
	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}

// ServiceWrapper обертка для сервисов без интерфейса Service
type ServiceWrapper struct {
	name    string
	startFn func(ctx context.Context) error
	stopFn  func(ctx context.Context) error
}

// NewServiceWrapper создает обертку для сервиса
func NewServiceWrapper(name string, startFn, stopFn func(ctx context.Context) error) *ServiceWrapper {
	return &ServiceWrapper{
		name:    name,
		startFn: startFn,
		stopFn:  stopFn,
	}
}

// Start запускает сервис
func (w *ServiceWrapper) Start(ctx context.Context) error {
	if w.startFn != nil {
		return w.startFn(ctx)
	}
	return nil
}

// Stop останавливает сервис
func (w *ServiceWrapper) Stop(ctx context.Context) error {
	if w.stopFn != nil {
		return w.stopFn(ctx)
	}
	return nil
}

// Name возвращает имя сервиса
func (w *ServiceWrapper) Name() string {
	return w.name
}
