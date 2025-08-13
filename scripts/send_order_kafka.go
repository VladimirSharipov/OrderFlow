package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// тестовый заказ для отправки
var testOrder = map[string]interface{}{
	"order_uid": "b563feb7b2b84b6test",
	"track_number": "WBILMTESTTRACK",
	"entry": "WBIL",
	"delivery": map[string]interface{}{
		"name": "Test Testov",
		"phone": "+9720000000",
		"zip": "2639809",
		"city": "Kiryat Mozkin",
		"address": "Ploshad Mira 15",
		"region": "Kraiot",
		"email": "test@gmail.com",
	},
	"payment": map[string]interface{}{
		"transaction": "b563feb7b2b84b6test",
		"request_id": "",
		"currency": "USD",
		"provider": "wbpay",
		"amount": 1817,
		"payment_dt": 1637907727,
		"bank": "alpha",
		"delivery_cost": 1500,
		"goods_total": 317,
		"custom_fee": 0,
	},
	"items": []map[string]interface{}{
		{
			"chrt_id": 9934930,
			"track_number": "WBILMTESTTRACK",
			"price": 453,
			"rid": "ab4219087a764ae0btest",
			"name": "Mascaras",
			"sale": 30,
			"size": "0",
			"total_price": 317,
			"nm_id": 2389212,
			"brand": "Vivienne Sabo",
			"status": 202,
		},
	},
	"locale": "en",
	"internal_signature": "",
	"customer_id": "test",
	"delivery_service": "meest",
	"shardkey": "9",
	"sm_id": 99,
	"date_created": "2021-11-26T06:22:19Z",
	"oof_shard": "1",
}

func main() {
	// настраиваем флаги командной строки
	brokers := flag.String("brokers", "localhost:9093", "адреса Kafka брокеров через запятую")
	topic := flag.String("topic", "orders", "название Kafka топика")
	flag.Parse()

	// создаем Kafka writer
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{*brokers},
		Topic:   *topic,
	})
	defer writer.Close()

	// превращаем заказ в JSON
	orderJSON, err := json.Marshal(testOrder)
	if err != nil {
		log.Fatalf("ошибка при создании JSON: %v", err)
	}

	// отправляем сообщение в Kafka
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte("b563feb7b2b84b6test"), // используем order_uid как ключ
		Value: orderJSON,
	})

	if err != nil {
		log.Fatalf("ошибка при отправке сообщения: %v", err)
	}

	fmt.Println("✅ Заказ успешно отправлен в Kafka")
	fmt.Printf("📤 Топик: %s\n", *topic)
	fmt.Printf("🔑 Ключ: %s\n", "b563feb7b2b84b6test")
	fmt.Printf("📦 Размер данных: %d байт\n", len(orderJSON))
} 