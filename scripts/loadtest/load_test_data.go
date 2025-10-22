package loadtest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"wbtest/internal/config"
	"wbtest/internal/db"
	"wbtest/internal/model"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/load_test_data.go <filename>")
		fmt.Println("Example: go run scripts/load_test_data.go test_data_3_orders.json")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Подключаемся к БД
	dbConn, err := db.New(cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Читаем файл с данными
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filename, err)
	}
	defer file.Close()

	// Декодируем JSON данные
	decoder := json.NewDecoder(file)

	successCount := 0
	errorCount := 0

	for decoder.More() {
		var order model.Order
		if err := decoder.Decode(&order); err != nil {
			log.Printf("Failed to decode order: %v", err)
			errorCount++
			continue
		}

		// Парсим дату создания
		if order.DateCreated.IsZero() {
			// Если дата не установлена, используем текущее время
			order.DateCreated = time.Now()
		}

		// Сохраняем заказ в БД
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := dbConn.SaveOrder(ctx, &order); err != nil {
			log.Printf("Failed to save order %s: %v", order.OrderUID, err)
			errorCount++
		} else {
			log.Printf("Successfully saved order: %s", order.OrderUID)
			successCount++
		}
		cancel()
	}

	log.Printf("Load completed: %d successful, %d errors", successCount, errorCount)
}
