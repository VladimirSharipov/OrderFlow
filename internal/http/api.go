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

// Server простой HTTPсервер для работы с заказами
type Server struct {
	Cache interfaces.OrderCache
}

// NewServer создаёт новый сервер с переданным кешом
func NewServer(c interfaces.OrderCache) *Server {
	return &Server{Cache: c}
}

// ServeHTTP маршрутизирует запросы /order/{id} API остальное статика
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

// handleHealth отдаёт статус сервиса
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

// handleCreateOrder создаёт новый заказ
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
		"message": "Order created successfully",
		"order_uid": order.OrderUID,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleGetOrder отдаёт один заказ по его UID
func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	orderUID := strings.TrimPrefix(r.URL.Path, "/order/")
	if orderUID == "" {
		http.Error(w, "Order ID is required", http.StatusBadRequest)
		return
	}

	order, ok := s.Cache.Get(orderUID)
	if !ok {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// serveStatic отдаёт статические файлы из папки web
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
