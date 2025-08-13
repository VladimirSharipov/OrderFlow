package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"wbtest/internal/cache"
	"wbtest/internal/db"
	"wbtest/internal/model"
)

// Server это наш HTTP сервер для API заказов
type Server struct {
	Cache *cache.OrderCache
	DB    *db.DB
}

// NewServer создает новый HTTP сервер
func NewServer(c *cache.OrderCache, database *db.DB) *Server {
	return &Server{Cache: c, DB: database}
}

// ServeHTTP обрабатывает все HTTP запросы
// просто проверяем путь и направляем к нужному обработчику
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// если путь начинается с /order/ то это запрос на получение заказа
	if strings.HasPrefix(r.URL.Path, "/order/") {
		s.handleGetOrder(w, r)
		return
	}

	// если путь /health то проверяем состояние сервиса
	if r.URL.Path == "/health" {
		s.handleHealth(w, r)
		return
	}

	// если POST запрос на /order то добавляем новый заказ
	if r.URL.Path == "/order" && r.Method == "POST" {
		s.handleAddOrder(w, r)
		return
	}

	// все остальное это статические файлы веб интерфейса
	serveStatic(w, r)
}

// handleGetOrder получает заказ по ID из кеша
func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	// извлекаем ID заказа из URL
	orderUID := strings.TrimPrefix(r.URL.Path, "/order/")
	if orderUID == "" {
		http.Error(w, "Нужно указать ID заказа", http.StatusBadRequest)
		return
	}

	// ищем заказ в кеше
	order, ok := s.Cache.Get(orderUID)
	if !ok {
		http.Error(w, "Заказ не найден", http.StatusNotFound)
		return
	}

	// возвращаем заказ в формате JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError)
		return
	}
}

// handleHealth проверяет что сервис работает
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":    "ok",
		"service":   "order-service",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)
}

// handleAddOrder добавляет новый заказ через HTTP API
func (s *Server) handleAddOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// читаем JSON из тела запроса
	var order model.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Неправильный JSON", http.StatusBadRequest)
		return
	}

	// проверяем что у заказа есть ID
	if order.OrderUID == "" {
		http.Error(w, "order_uid обязателен", http.StatusBadRequest)
		return
	}

	// сохраняем в базу данных
	ctx := context.Background()
	if err := s.DB.SaveOrder(ctx, &order); err != nil {
		http.Error(w, "Не удалось сохранить заказ", http.StatusInternalServerError)
		return
	}

	// добавляем в кеш
	s.Cache.Set(&order)

	// возвращаем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	response := map[string]interface{}{
		"status":    "created",
		"order_uid": order.OrderUID,
		"message":   "Заказ успешно добавлен",
	}
	json.NewEncoder(w).Encode(response)
}

// serveStatic отдает статические файлы веб интерфейса
func serveStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// безопасно формируем путь к файлу
	rel := strings.TrimPrefix(path, "/")
	filePath := filepath.Join("web", filepath.Clean(rel))

	// проверяем что файл существует
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// отдаем файл
	http.ServeFile(w, r, filePath)
}
