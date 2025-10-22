package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"wbtest/internal/cache"
	"wbtest/internal/config"
	"wbtest/internal/db"
	"wbtest/internal/dlq"
	httpapi "wbtest/internal/http"
	"wbtest/internal/interfaces"
	"wbtest/internal/kafka"
	"wbtest/internal/migrations"
	"wbtest/internal/model"
	"wbtest/internal/retry"
	"wbtest/internal/validator"
)

// App представляет основное приложение
type App struct {
	Config       *config.Config
	DB           interfaces.OrderRepository
	Cache        interfaces.OrderCache
	Validator    interfaces.OrderValidator
	Consumer     interfaces.MessageConsumer
	RetryService interfaces.RetryService
	DLQService   interfaces.DLQService
	HTTPServer   *http.Server
}

// NewApp создает приложение с компонентами
func NewApp(cfg *config.Config) (*App, error) {
	app := &App{Config: cfg}

	// Инициализация БД
	if err := app.initDB(); err != nil {
		return nil, err
	}

	// Инициализация кеша
	if err := app.initCache(); err != nil {
		return nil, err
	}

	// Инициализация валидатора
	app.initValidator()

	// Инициализация retry сервиса
	app.initRetryService()

	// Инициализация DLQ сервиса
	if err := app.initDLQService(); err != nil {
		return nil, err
	}

	// Инициализация Kafka consumer
	if err := app.initKafkaConsumer(); err != nil {
		return nil, err
	}

	// Инициализация HTTP сервера
	app.initHTTPServer()

	return app, nil
}

// initDB подключается к БД
func (a *App) initDB() error {
	log.Println("Initializing database connection...")

	dbConn, err := db.New(a.Config.DatabaseURL())
	if err != nil {
		return err
	}

	a.DB = dbConn
	log.Println("Database connected successfully")
	return nil
}

// initCache создает кеш и загружает данные
func (a *App) initCache() error {
	log.Println("Initializing cache...")

	// Создаем кеш с настройками из конфигурации
	orderCache := cache.NewOrderCache(a.Config.Cache.MaxSize, time.Duration(a.Config.Cache.TTLMinutes)*time.Minute)
	a.Cache = orderCache

	// Пытаемся загрузить заказы из БД в кеш
	log.Println("Loading orders from database...")
	ctx, cancel := context.WithTimeout(context.Background(), a.Config.App.DatabaseLoadTimeout)
	defer cancel()

	orders, err := a.DB.LoadAllOrders(ctx)
	if err != nil {
		log.Printf("Warning: Failed to load orders from database: %v", err)
		log.Println("Starting with empty cache...")
		orders = []*model.Order{}
	}

	a.Cache.LoadAll(orders)
	log.Printf("Cache loaded: %d orders", len(orders))

	return nil
}

// initValidator создает валидатор
func (a *App) initValidator() {
	log.Println("Initializing validator...")
	a.Validator = validator.NewOrderValidator()
	log.Println("Validator initialized")
}

// initRetryService создает retry сервис
func (a *App) initRetryService() {
	log.Println("Initializing retry service...")
	a.RetryService = retry.NewRetryService(&a.Config.Retry)
	log.Println("Retry service initialized")
}

// initDLQService создает DLQ сервис
func (a *App) initDLQService() error {
	log.Println("Initializing DLQ service...")

	a.DLQService = dlq.NewDLQService(&a.Config.DLQ, a.Config.Kafka.Brokers)
	log.Println("DLQ service initialized")

	return nil
}

// initKafkaConsumer создает Kafka consumer
func (a *App) initKafkaConsumer() error {
	log.Printf("Initializing Kafka consumer: brokers=%v, topic=%s, groupID=%s",
		a.Config.Kafka.Brokers, a.Config.Kafka.Topic, a.Config.Kafka.GroupID)

	consumer := kafka.NewConsumer(a.Config.Kafka.Brokers, a.Config.Kafka.Topic, a.Config.Kafka.GroupID)
	a.Consumer = consumer

	log.Println("Kafka consumer initialized")
	return nil
}

// initHTTPServer создает HTTP сервер
func (a *App) initHTTPServer() {
	log.Println("Initializing HTTP server...")

	// Создаем API с кешем и БД
	api := httpapi.NewServer(a.Cache, a.DB)

	// Создаем HTTP сервер
	a.HTTPServer = &http.Server{
		Addr:         ":" + strconv.Itoa(a.Config.HTTP.Port),
		Handler:      api,
		ReadTimeout:  a.Config.HTTP.ReadTimeout,
		WriteTimeout: a.Config.HTTP.WriteTimeout,
		IdleTimeout:  a.Config.HTTP.IdleTimeout,
	}

	log.Printf("HTTP server configured on port %d", a.Config.HTTP.Port)
}

// Close закрывает ресурсы
func (a *App) Close() error {
	log.Println("Closing application resources...")

	// Закрываем кеш
	if cacheImpl, ok := a.Cache.(*cache.OrderCache); ok {
		cacheImpl.Stop()
	}

	// Закрываем БД
	if a.DB != nil {
		a.DB.Close()
	}

	// Закрываем Kafka consumer
	if a.Consumer != nil {
		if err := a.Consumer.Close(); err != nil {
			log.Printf("Error closing Kafka consumer: %v", err)
		}
	}

	// Закрываем DLQ service
	if a.DLQService != nil {
		if err := a.DLQService.Close(); err != nil {
			log.Printf("Error closing DLQ service: %v", err)
		}
	}

	log.Println("Application resources closed")
	return nil
}

// runMigrations запускает миграции базы данных
func (a *App) runMigrations() error {
	log.Println("Running database migrations...")

	// Создаем мигратор
	migrator := migrations.NewMigrator(a.DB.(*db.DB).DB, "schema_migrations")

	// Загружаем миграции
	if err := migrations.LoadMigrationsFromFiles(migrator, "migrations"); err != nil {
		return fmt.Errorf("failed to load migrations: %v", err)
	}

	// Запускаем миграции
	ctx := context.Background()
	if err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}
