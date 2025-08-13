# OrderFlow - Демонстрационный микросервис

Демонстрационный сервис для обработки заказов с использованием Kafka, PostgreSQL и кеширования в памяти.

## 🚀 Возможности

- **HTTP API** для получения заказов по ID
- **Веб-интерфейс** для поиска заказов
- **Kafka consumer** для получения сообщений о заказах
- **PostgreSQL** для хранения данных
- **Кеширование** в памяти для быстрого доступа
- **Восстановление кеша** из БД при перезапуске
- **Валидация данных** и обработка ошибок

## 🏗️ Архитектура

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Kafka    │───▶│  Go Service │───▶│ PostgreSQL │
│  Producer  │    │             │    │             │
└─────────────┘    └─────────────┘    └─────────────┘
                          │
                          ▼
                   ┌─────────────┐
                   │   Cache     │
                   │  (Memory)   │
                   └─────────────┘
                          │
                          ▼
                   ┌─────────────┐
                   │ HTTP API    │
                   │ + Web UI    │
                   └─────────────┘
```

## 📋 Требования

- Go 1.21+
- Docker & Docker Compose

## 🚀 Быстрый старт

### 1. Клонирование репозитория
```bash
git clone <repository-url>
cd project1
```

### 2. Запуск инфраструктуры
```bash
docker-compose up -d
```

### 3. Применение миграций БД
```bash
Get-Content migrations/001_init.sql | docker exec -i project1-postgres-1 psql -U orders_user -d orders_db
```

### 4. Запуск сервиса
```bash
go run cmd/service/main.go
```

Сервис будет доступен по адресу: **http://localhost:8081**

## 🔧 Конфигурация

### Порты
- **HTTP API**: 8081
- **PostgreSQL**: 5433
- **Kafka**: 9093
- **Zookeeper**: 2182

### Переменные окружения
```bash
DB_CONN=postgres://orders_user:orders_pass@localhost:5433/orders_db?sslmode=disable
```

## 📡 API Endpoints

### GET /health
Проверка состояния сервиса
```bash
curl http://localhost:8081/health
```

### GET /order/{order_uid}
Получение заказа по ID
```bash
curl http://localhost:8081/order/b563feb7b2b84b6test
```

### POST /order
Добавление нового заказа
```bash
curl -X POST http://localhost:8081/order \
  -H "Content-Type: application/json" \
  -d @order.json
```

### GET /
Веб-интерфейс для поиска заказов

## 🧪 Тестирование

### Быстрый тест
```bash
# проверяем что сервис работает
curl http://localhost:8081/health

# получаем заказ по ID
curl http://localhost:8081/order/b563feb7b2b84b6test

# добавляем новый заказ
curl -X POST http://localhost:8081/order \
  -H "Content-Type: application/json" \
  -d @order.json
```

### Веб интерфейс
1. Откройте браузер
2. Перейдите на http://localhost:8081/
3. Введите ID заказа: `b563feb7b2b84b6test`
4. Нажмите "Найти заказ"

### Отправка заказа через Kafka
```bash
go run scripts/send_order_kafka.go
```

## 📊 Структура данных

### Основная таблица (orders)
- `order_uid` - уникальный идентификатор заказа
- `track_number` - номер отслеживания
- `entry` - код входа
- `locale` - локаль
- `customer_id` - ID клиента
- `delivery_service` - сервис доставки
- `date_created` - дата создания

### Доставка (delivery)
- `name` - имя получателя
- `phone` - телефон
- `address` - адрес
- `email` - email

### Оплата (payment)
- `transaction` - ID транзакции
- `amount` - сумма
- `currency` - валюта
- `provider` - провайдер

### Товары (items)
- `name` - название товара
- `price` - цена
- `brand` - бренд
- `status` - статус

## 🔄 Поток данных

1. **Получение сообщения** из Kafka
2. **Парсинг JSON** и валидация
3. **Сохранение в БД** с использованием транзакций
4. **Обновление кеша** в памяти
5. **Подтверждение** сообщения Kafka

## 🛡️ Обработка ошибок

- **Валидация входных данных**
- **Транзакции БД** для атомарности
- **Логирование ошибок**
- **Graceful degradation** при сбоях

## 📈 Производительность

- **Кеширование** в памяти для быстрого доступа
- **Оптимизированные SQL запросы**
- **Connection pooling** для БД
- **Асинхронная обработка** Kafka сообщений

## 🧹 Очистка

### Остановка сервисов
```bash
docker-compose down
```

### Удаление данных
```bash
docker-compose down -v
```

## 📝 Логи

Логи сервиса выводятся в stdout и содержат:
- Подключение к БД и Kafka
- Обработку HTTP запросов
- Обработку Kafka сообщений
- Ошибки и предупреждения

## 🤝 Разработка

### Структура проекта
```
project1/
├── cmd/service/          # Основной исполняемый файл
├── internal/             # Внутренняя логика
│   ├── cache/           # Кеширование
│   ├── db/              # Работа с БД
│   ├── http/            # HTTP API
│   ├── kafka/           # Kafka consumer
│   └── model/           # Модели данных
├── migrations/           # Миграции БД
├── scripts/              # Скрипты для тестирования
├── web/                  # Веб-интерфейс
└── docker-compose.yml    # Docker конфигурация
```

### Добавление новых функций
1. Обновить модели в `internal/model/`
2. Добавить миграции БД
3. Реализовать логику в соответствующих пакетах
4. Добавить HTTP эндпоинты
5. Обновить тесты

## 📚 Дополнительные ресурсы

- [Go Documentation](https://golang.org/doc/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Kafka Documentation](https://kafka.apache.org/documentation/)
- [Docker Documentation](https://docs.docker.com/)

## 🆘 Поддержка

При возникновении проблем:
1. Проверьте логи сервиса
2. Убедитесь, что все Docker контейнеры запущены
3. Проверьте доступность портов
4. Проверьте подключение к БД и Kafka

## 📄 Лицензия

MIT License#   O r d e r F l o w 
 
 
