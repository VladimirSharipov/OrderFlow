package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"wbtest/internal/config"
	"wbtest/internal/model"

	"github.com/brianvoe/gofakeit/v6"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/generate_test_data.go <count>")
		fmt.Println("Example: go run scripts/generate_test_data.go 10")
		os.Exit(1)
	}

	// Загружаем конфигурацию
	cfg := config.Load()

	count, err := strconv.Atoi(os.Args[1])
	if err != nil || count <= 0 {
		log.Fatalf("Invalid count: must be a positive integer")
	}

	if count > cfg.Generator.MaxOrdersCount {
		log.Fatalf("Count too large: maximum %d orders allowed", cfg.Generator.MaxOrdersCount)
	}

	// Инициализируем faker
	gofakeit.Seed(time.Now().UnixNano())

	orders := generateOrders(count, cfg)

	// Сохраняем в файл
	filename := fmt.Sprintf("test_data_%d_orders.json", count)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	for _, order := range orders {
		if err := encoder.Encode(order); err != nil {
			log.Printf("Failed to encode order %s: %v", order.OrderUID, err)
		}
	}

	log.Printf("Generated %d orders in %s", count, filename)
}

func generateOrders(count int, cfg *config.Config) []*model.Order {
	orders := make([]*model.Order, count)

	for i := 0; i < count; i++ {
		orders[i] = generateOrder(i, cfg)
	}

	return orders
}

func generateOrder(index int, cfg *config.Config) *model.Order {
	orderUID := gofakeit.UUID()

	// Генерируем количество товаров из конфигурации
	itemsCount := gofakeit.IntRange(1, cfg.Generator.MaxItemsPerOrder)
	items := generateItems(itemsCount, cfg)

	// Вычисляем общую стоимость товаров
	totalItemsPrice := 0
	for _, item := range items {
		totalItemsPrice += item.TotalPrice
	}

	// Генерируем стоимость доставки
	deliveryCost := gofakeit.IntRange(100, 2000)
	totalAmount := totalItemsPrice + deliveryCost

	// Генерируем дату создания (не старше 30 дней)
	dateCreated := gofakeit.DateRange(time.Now().AddDate(0, 0, -30), time.Now())

	return &model.Order{
		OrderUID:          orderUID,
		TrackNumber:       generateTrackNumber(),
		Entry:             gofakeit.RandomString([]string{"WBIL", "WBILMT", "WBILM", "WBILT"}),
		Delivery:          generateDelivery(),
		Payment:           generatePayment(orderUID, totalAmount, deliveryCost, totalItemsPrice),
		Items:             items,
		Locale:            gofakeit.RandomString([]string{"en", "ru", "es", "fr", "de"}),
		InternalSignature: "",
		CustomerID:        gofakeit.Username(),
		DeliveryService:   gofakeit.RandomString([]string{"meest", "cdek", "dhl", "fedex", "ups"}),
		ShardKey:          fmt.Sprintf("%d", gofakeit.IntRange(0, 9)),
		SmID:              gofakeit.IntRange(1, 100),
		DateCreated:       dateCreated,
		OofShard:          fmt.Sprintf("%d", gofakeit.IntRange(0, 4)),
	}
}

func generateDelivery() model.Delivery {
	return model.Delivery{
		Name:    gofakeit.Name(),
		Phone:   gofakeit.Phone(),
		Zip:     gofakeit.Zip(),
		City:    gofakeit.City(),
		Address: gofakeit.Address().Address,
		Region:  gofakeit.State(),
		Email:   gofakeit.Email(),
	}
}

func generatePayment(orderUID string, totalAmount, deliveryCost, goodsTotal int) model.Payment {
	currencies := []string{"USD", "EUR", "RUB", "GBP"}
	providers := []string{"wbpay", "stripe", "paypal", "square"}
	banks := []string{"alpha", "beta", "gamma", "delta"}

	// Генерируем дату платежа (не в будущем)
	paymentTime := gofakeit.DateRange(time.Now().AddDate(0, 0, -7), time.Now())

	return model.Payment{
		Transaction:  orderUID,
		RequestID:    "",
		Currency:     gofakeit.RandomString(currencies),
		Provider:     gofakeit.RandomString(providers),
		Amount:       totalAmount,
		PaymentDT:    int(paymentTime.Unix()),
		Bank:         gofakeit.RandomString(banks),
		DeliveryCost: deliveryCost,
		GoodsTotal:   goodsTotal,
		CustomFee:    0,
	}
}

func generateItems(count int, cfg *config.Config) []model.Item {
	items := make([]model.Item, count)

	names := []string{
		"Laptop", "Smartphone", "Tablet", "Headphones", "Keyboard", "Mouse",
		"Monitor", "Speaker", "Camera", "Watch", "Book", "Game Console",
		"Fitness Tracker", "Bluetooth Earbuds", "Power Bank", "USB Cable",
		"Wireless Charger", "Gaming Mouse", "Mechanical Keyboard", "Webcam",
	}

	brands := []string{
		"Apple", "Samsung", "Sony", "Bose", "Logitech", "Dell", "HP", "Lenovo",
		"Microsoft", "Google", "Asus", "Acer", "Razer", "SteelSeries", "JBL",
		"Canon", "Nikon", "Garmin", "Fitbit", "Anker",
	}

	for i := 0; i < count; i++ {
		price := gofakeit.IntRange(cfg.Generator.MinPrice, cfg.Generator.MaxPrice)
		sale := gofakeit.IntRange(0, cfg.Generator.MaxSale)
		totalPrice := price * (100 - sale) / 100

		items[i] = model.Item{
			ChrtID:      gofakeit.IntRange(1000000, 9999999),
			TrackNumber: generateItemTrackNumber(),
			Price:       price,
			Rid:         gofakeit.UUID(),
			Name:        gofakeit.RandomString(names),
			Sale:        sale,
			Size:        fmt.Sprintf("%d", gofakeit.IntRange(0, 5)),
			TotalPrice:  totalPrice,
			NmID:        gofakeit.IntRange(1000000, 9999999),
			Brand:       gofakeit.RandomString(brands),
			Status:      gofakeit.IntRange(200, 299),
		}
	}

	return items
}

func generateTrackNumber() string {
	// Генерируем трек-номер в формате: TRACK + 8 символов
	return "TRACK" + gofakeit.Regex(`[A-Z0-9]{8}`)
}

func generateItemTrackNumber() string {
	// Генерируем трек-номер товара в формате: ITEM + 6 символов
	return "ITEM" + gofakeit.Regex(`[A-Z0-9]{6}`)
}
