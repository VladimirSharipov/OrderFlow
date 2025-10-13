package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"wbtest/internal/interfaces"
	"wbtest/internal/model"
)

// Server HTTP сервер для заказов
type Server struct {
	Cache interfaces.OrderCache
	DB    interfaces.OrderRepository
}

// NewServer создает сервер
func NewServer(c interfaces.OrderCache, db interfaces.OrderRepository) *Server {
	return &Server{Cache: c, DB: db}
}

// ServeHTTP маршрутизирует запросы
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" {
		s.handleHealth(w, r)
		return
	}

	if r.URL.Path == "/order" && r.Method == "POST" {
		s.handleCreateOrder(w, r)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/order/") {
		s.handleGetOrder(w, r)
		return
	}

	serveStatic(w, r)
}

// handleHealth возвращает статус
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":     "ok",
		"service":    "order-service",
		"cache_size": s.Cache.GetStats().Size,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleCreateOrder создает заказ
func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var order model.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Добавляем заказ в кеш
	s.Cache.Set(&order)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	response := map[string]interface{}{
		"message":   "Order created successfully",
		"order_uid": order.OrderUID,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleGetOrder возвращает заказ по UID
func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	orderUID := strings.TrimPrefix(r.URL.Path, "/order/")
	if orderUID == "" {
		http.Error(w, "Order ID is required", http.StatusBadRequest)
		return
	}

	// Сначала пытаемся найти в кеше
	order, ok := s.Cache.Get(orderUID)
	if ok {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(order); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	// Если не найдено в кеше, пытаемся загрузить из БД
	if s.DB != nil {
		ctx := r.Context()
		dbOrder, err := s.DB.GetOrderByUID(ctx, orderUID)
		if err == nil && dbOrder != nil {
			// Загружаем в кеш для следующих запросов
			s.Cache.Set(dbOrder)

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(dbOrder); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
			return
		}
	}

	// Если не найдено ни в кеше, ни в БД
	http.Error(w, "Order not found", http.StatusNotFound)
}

// serveStatic отдает статику
func serveStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// Формируем путь безопасно После Clean убираем ведущий слэш
	// чтобы Join не сделал абсолютный путь на Windows
	cleanPath := filepath.Clean(path)
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	filePath := filepath.Join("web", cleanPath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, filePath)
}
