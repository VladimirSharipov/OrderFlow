# Order Service

Демонстрационный микросервис для обработки заказов с использованием Kafka, PostgreSQL и кеширования.

## Возможности

- ✅ Получение заказов из Kafka
- ✅ Сохранение в PostgreSQL с транзакциями
- ✅ In-memory кеш с TTL и LRU эвикцией
- ✅ HTTP API для получения заказов
- ✅ Веб-интерфейс для поиска заказов
- ✅ Расширенная валидация входящих данных
- ✅ Graceful shutdown
- ✅ Конфигурация через переменные окружения
- ✅ Генератор тестовых данных с gofakeit
- ✅ Система миграций с поддержкой down миграций
- ✅ Мелкогранулярные блокировки в кеше
- ✅ Метрики кеша (hits, misses, hit rate)
- ✅ Покрытие тестами основных компонентов

## Архитектура

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│  Kafka  │───▶│   Go    │───▶│PostgreSQL│   │  Cache  │
│         │    │ Service │    │         │   │         │
└─────────┘    └─────────┘    └─────────┘   └─────────┘
                       │
                       ▼
                ┌─────────┐
                │   HTTP  │
                │   API   │
                └─────────┘
```

## Быстрый старт

### 1. Запуск инфраструктуры

```bash
docker-compose up -d
```

### 2. Применение миграций

```bash
# Up миграции
go run migrations/migrate.go 'postgres://orders_user:orders_pass@localhost:5432/orders_db?sslmode=disable' up

# Down миграции (если нужно)
go run migrations/migrate.go 'postgres://orders_user:orders_pass@localhost:5432/orders_db?sslmode=disable' down 1

# Статус миграций
go run migrations/migrate.go 'postgres://orders_user:orders_pass@localhost:5432/orders_db?sslmode=disable' status
```

### 3. Сборка и запуск

```bash
go build -o order-service.exe ./cmd/service
./order-service.exe
```

### 4. Генерация тестовых данных

```bash
# Генерируем 10 тестовых заказов с качественными данными
go run scripts/generate_test_data.go 10

# Генерируем 100 тестовых заказов
go run scripts/generate_test_data.go 100
```

## Конфигурация

Все настройки можно изменить через переменные окружения:

```bash
# База данных
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=orders_user
export DB_PASSWORD=orders_pass
export DB_NAME=orders_db
export DB_SSLMODE=disable
export DB_MAX_OPEN_CONNS=25
export DB_MAX_IDLE_CONNS=5
export DB_CONN_MAX_LIFETIME=5m

# Kafka
export KAFKA_BROKERS=localhost:9092
export KAFKA_TOPIC=orders
export KAFKA_GROUP_ID=order-service
export KAFKA_AUTO_OFFSET_RESET=earliest
export KAFKA_ENABLE_AUTO_COMMIT=true
export KAFKA_SESSION_TIMEOUT_MS=30000

# HTTP сервер
export HTTP_PORT=8082
export HTTP_READ_TIMEOUT=30s
export HTTP_WRITE_TIMEOUT=30s
export HTTP_IDLE_TIMEOUT=60s

# Кеш
export CACHE_MAX_SIZE=1000
export CACHE_TTL=24h
export CACHE_CLEANUP_INTERVAL=5m

# Приложение
export GRACEFUL_SHUTDOWN_TIMEOUT=30s
export LOG_LEVEL=info
export ENVIRONMENT=development
export DB_LOAD_TIMEOUT=10s
export SHUTDOWN_WAIT_TIMEOUT=5s

# Генератор тестовых данных
export GENERATOR_MAX_ORDERS=10000
export GENERATOR_MAX_ITEMS_PER_ORDER=5
export GENERATOR_MIN_PRICE=50
export GENERATOR_MAX_PRICE=5000
export GENERATOR_MAX_SALE=50

# Валидация
export VALIDATION_ORDER_UID_MIN_LENGTH=10
export VALIDATION_ORDER_UID_MAX_LENGTH=50
export VALIDATION_TRACK_NUMBER_MIN_LENGTH=5
export VALIDATION_TRACK_NUMBER_MAX_LENGTH=20
export VALIDATION_MAX_PAYMENT_AMOUNT=1000000
export VALIDATION_MAX_ITEMS_PER_ORDER=100
export VALIDATION_MAX_ITEM_PRICE=100000
```

## API

### Получить заказ по ID

```bash
curl http://localhost:8082/order/b563feb7b2b84b6test
```

### Веб-интерфейс

Откройте http://localhost:8082/ в браузере

## Тестирование

### Запуск тестов

```bash
# Все тесты
go test ./...

# Тесты кеша
go test ./internal/cache

# Тесты валидатора
go test ./internal/validator

# Тесты с покрытием
go test -cover ./...
```

### Генерация тестовых данных

```bash
# Создать 10 тестовых заказов с качественными данными
go run scripts/generate_test_data.go 10

# Создать 100 тестовых заказов
go run scripts/generate_test_data.go 100
```

### Отправка в Kafka

```bash
# Отправить тестовый заказ в Kafka
echo '{"order_uid":"test123","track_number":"TRACK123",...}' | \
  docker exec -i wbtestl0-kafka-1 kafka-console-producer --bootstrap-server localhost:9092 --topic orders
```

## Структура проекта

```
├── cmd/
│   └── service/
│       └── main.go              # Точка входа
├── internal/
│   ├── cache/                   # Кеш заказов с мелкогранулярными блокировками
│   │   ├── cache.go
│   │   └── cache_test.go
│   ├── config/                  # Конфигурация
│   ├── db/                      # Работа с БД
│   ├── http/                    # HTTP API
│   ├── interfaces/              # Интерфейсы
│   ├── kafka/                   # Kafka consumer
│   ├── model/                   # Модели данных
│   └── validator/               # Расширенная валидация
│       ├── validator.go
│       └── validator_test.go
├── migrations/                  # Система миграций
│   ├── migrate.go
│   ├── 001_init.sql
│   └── 002_down.sql
├── scripts/                     # Скрипты
│   └── generate_test_data.go    # Генератор с gofakeit
├── web/                         # Веб-интерфейс
└── docker-compose.yml           # Инфраструктура
```

## Особенности реализации

### Кеш
- TTL: настраивается через конфигурацию (по умолчанию 24 часа)
- Максимальный размер: настраивается через конфигурацию (по умолчанию 1000 заказов)
- LRU эвикция при переполнении
- Автоматическая очистка устаревших записей
- Мелкогранулярные блокировки для лучшей производительности
- Метрики: hits, misses, hit rate, evictions, expirations

### Валидация
- Проверка обязательных полей
- Валидация форматов (email, телефон, UID)
- Проверка бизнес-логики (цены, количества, даты)
- Валидация допустимых значений (валюты, провайдеры, локали)
- Проверка целостности данных (суммы, соответствие полей)

### Graceful Shutdown
- Обработка SIGINT/SIGTERM
- Корректное завершение HTTP сервера с таймаутами
- Остановка Kafka consumer
- Очистка ресурсов кеша
- Настраиваемый таймаут завершения

### Миграции
- Поддержка up и down миграций
- Транзакционное выполнение
- Отслеживание примененных миграций
- Команда status для проверки состояния

### Безопасность
- Транзакции при сохранении заказов
- Подтверждение сообщений Kafka
- Расширенная валидация входящих данных
- Обработка всех ошибок
- Логирование всех операций

## Мониторинг

### Логи
Сервис выводит подробные логи:
- Подключение к БД и Kafka
- Обработка сообщений
- Валидация заказов
- Ошибки и предупреждения
- Graceful shutdown

### Метрики кеша
- Размер кеша
- Количество попаданий/промахов
- Hit rate (процент попаданий)
- Количество эвикций и экспираций
- Время жизни записей

## Разработка

### Добавление новых полей
1. Обновите модель в `internal/model/`
2. Добавьте валидацию в `internal/validator/`
3. Обновите миграции
4. Добавьте тесты

### Тестирование
```bash
# Запуск всех тестов
go test ./...

# Тесты с покрытием
go test -cover ./...

# Тесты с детальным отчетом
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Линтинг
```bash
# Сортировка импортов
goimports -w .

# Проверка кода
golangci-lint run
```

### Генерация моков
```bash
# Установка mockgen
go install github.com/golang/mock/mockgen@latest

# Генерация моков для интерфейсов
mockgen -source=internal/interfaces/interfaces.go -destination=internal/mocks/mocks.go
```

## Исправленные проблемы

1. ✅ Убрана команда Get-Content из README
2. ✅ Импорты отсортированы через goimports
3. ✅ **Все конфигурационные данные вынесены в отдельный конфиг**:
   - Настройки базы данных (таймауты подключения, пулы соединений)
   - Настройки Kafka (таймауты сессий, автокоммит)
   - Настройки HTTP сервера (таймауты чтения/записи)
   - Настройки кеша (размер, TTL, интервал очистки)
   - Настройки приложения (graceful shutdown, логирование)
   - Настройки генератора тестовых данных (лимиты, диапазоны цен)
   - Настройки валидации (длины полей, максимальные значения)
4. ✅ Реализована инвалидация кеша с автоматической очисткой
5. ✅ Добавлены мелкогранулярные блокировки в кеше
6. ✅ Создана система миграций с поддержкой down миграций
7. ✅ Расширена валидация данных (форматы, бизнес-логика, целостность)
8. ✅ Улучшен генератор тестовых данных с использованием gofakeit
9. ✅ Добавлена обработка всех ошибок
10. ✅ Созданы интерфейсы и тесты для основных компонентов
11. ✅ Реализован полноценный graceful shutdown

## Конфигурируемые параметры

Все ранее захардкоженные значения теперь настраиваются через переменные окружения:

### Таймауты и лимиты
- `DB_LOAD_TIMEOUT` - таймаут загрузки данных из БД при старте (по умолчанию 10s)
- `SHUTDOWN_WAIT_TIMEOUT` - время ожидания завершения горутин (по умолчанию 5s)
- `GENERATOR_MAX_ORDERS` - максимальное количество генерируемых заказов (по умолчанию 10000)
- `VALIDATION_MAX_PAYMENT_AMOUNT` - максимальная сумма платежа (по умолчанию 1000000)

### Диапазоны генерации данных
- `GENERATOR_MIN_PRICE` / `GENERATOR_MAX_PRICE` - диапазон цен товаров (50-5000)
- `GENERATOR_MAX_ITEMS_PER_ORDER` - максимальное количество товаров в заказе (5)
- `GENERATOR_MAX_SALE` - максимальная скидка в процентах (50)

### Валидация
- `VALIDATION_ORDER_UID_MIN_LENGTH` / `VALIDATION_ORDER_UID_MAX_LENGTH` - длина UID заказа (10-50)
- `VALIDATION_TRACK_NUMBER_MIN_LENGTH` / `VALIDATION_TRACK_NUMBER_MAX_LENGTH` - длина трек-номера (5-20)
- `VALIDATION_MAX_ITEMS_PER_ORDER` - максимальное количество товаров в заказе (100)
- `VALIDATION_MAX_ITEM_PRICE` - максимальная цена товара (100000)