package main

import (
	"context"
	"log"
	"os"
	"time"

	"encoding/json"

	"net/http"
	"wbtest/internal/cache"
	"wbtest/internal/db"
	httpapi "wbtest/internal/http"
	"wbtest/internal/kafka"
	"wbtest/internal/model"
)

func main() {
	log.Println("Запускаем сервис заказов...")

	// подключаемся к базе данных
	connStr := os.Getenv("DB_CONN")
	if connStr == "" {
		connStr = "postgres://orders_user:orders_pass@localhost:5433/orders_db?sslmode=disable"
	}
	log.Printf("Подключаемся к базе данных: %s", connStr)

	dbConn, err := db.New(connStr)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer dbConn.Close()
	log.Println("Подключились к базе данных")

	// загружаем все заказы из базы в кеш при старте
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Загружаем заказы из базы данных...")
	orders, err := dbConn.LoadAllOrders(ctx)
	if err != nil {
		log.Fatalf("Не удалось загрузить заказы из базы: %v", err)
	}

	orderCache := cache.NewOrderCache()
	orderCache.LoadAll(orders)
	log.Printf("Кеш загружен: %d заказов", len(orders))

	// настраиваем подключение к Kafka
	kafkaBrokers := []string{"localhost:9093"}
	topic := "orders"
	groupID := "order-service"

	log.Printf("Подключаемся к Kafka: брокеры=%v, топик=%s, группа=%s", kafkaBrokers, topic, groupID)

	consumer := kafka.NewConsumer(kafkaBrokers, topic, groupID)
	defer consumer.Close()
	log.Println("Kafka consumer создан")

	// запускаем обработчик сообщений из Kafka в отдельной горутине
	go func() {
		log.Println("Запускаем Kafka consumer...")
		err := consumer.ReadMessages(context.Background(), func(msg []byte) {
			log.Printf("[KAFKA] Получили сообщение: %s", string(msg))

			// парсим JSON в структуру заказа
			var order model.Order
			if err := json.Unmarshal(msg, &order); err != nil {
				log.Printf("[KAFKA] Не удалось распарсить JSON: %v", err)
				return
			}

			// проверяем что у заказа есть ID
			if order.OrderUID == "" {
				log.Printf("[KAFKA] Пропускаем заказ без order_uid")
				return
			}

			log.Printf("[KAFKA] Распарсили заказ: %+v", order)

			// сохраняем в базу данных
			if err := dbConn.SaveOrder(context.Background(), &order); err != nil {
				log.Printf("[KAFKA] Не удалось сохранить заказ %s: %v", order.OrderUID, err)
				return
			}

			// обновляем кеш
			orderCache.Set(&order)
			log.Printf("[KAFKA] Заказ %s сохранен и добавлен в кеш", order.OrderUID)
		})

		if err != nil {
			log.Printf("Kafka consumer остановился: %v", err)
		}
	}()

	// запускаем HTTP сервер
	api := httpapi.NewServer(orderCache, dbConn)
	log.Println("Запускаем HTTP сервер на порту 8081")

	go func() {
		log.Println("HTTP API сервер запущен")
		if err := http.ListenAndServe(":8081", api); err != nil {
			log.Fatalf("Ошибка HTTP сервера: %v", err)
		}
	}()

	log.Println("Сервис заказов успешно запущен")
	log.Println("Ожидаем запросы...")

	// держим программу запущенной
	select {}
}
