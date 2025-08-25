package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"wbtest/internal/interfaces"
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
	if strings.HasPrefix(r.URL.Path, "/order/") {
		s.handleGetOrder(w, r)
		return
	}

	serveStatic(w, r)
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
