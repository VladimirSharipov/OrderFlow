package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics структура для метрик
type Metrics struct {
	// HTTP метрики
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// Kafka метрики
	KafkaMessagesConsumed *prometheus.CounterVec
	KafkaMessagesFailed   *prometheus.CounterVec
	KafkaConsumerLag      *prometheus.GaugeVec

	// Order метрики
	OrdersProcessed *prometheus.CounterVec
	OrdersFailed    *prometheus.CounterVec
	OrdersInCache   *prometheus.GaugeVec
	OrdersInDB      *prometheus.GaugeVec

	// Retry метрики
	RetryAttempts *prometheus.CounterVec
	RetryFailures *prometheus.CounterVec

	// DLQ метрики
	DLQMessagesSent      *prometheus.CounterVec
	DLQMessagesProcessed *prometheus.CounterVec

	// Database метрики
	DatabaseConnections   *prometheus.GaugeVec
	DatabaseQueryDuration *prometheus.HistogramVec
}

// New создает новые метрики
func New() *Metrics {
	return &Metrics{
		// HTTP метрики
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		HTTPRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),

		// Kafka метрики
		KafkaMessagesConsumed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafka_messages_consumed_total",
				Help: "Total number of Kafka messages consumed",
			},
			[]string{"topic", "group_id"},
		),
		KafkaMessagesFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafka_messages_failed_total",
				Help: "Total number of Kafka messages failed",
			},
			[]string{"topic", "group_id", "error_type"},
		),
		KafkaConsumerLag: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_consumer_lag",
				Help: "Kafka consumer lag",
			},
			[]string{"topic", "group_id"},
		),

		// Order метрики
		OrdersProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "orders_processed_total",
				Help: "Total number of orders processed",
			},
			[]string{"status"},
		),
		OrdersFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "orders_failed_total",
				Help: "Total number of orders failed",
			},
			[]string{"error_type"},
		),
		OrdersInCache: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "orders_in_cache",
				Help: "Number of orders in cache",
			},
			[]string{},
		),
		OrdersInDB: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "orders_in_database",
				Help: "Number of orders in database",
			},
			[]string{},
		),

		// Retry метрики
		RetryAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "retry_attempts_total",
				Help: "Total number of retry attempts",
			},
			[]string{"operation", "attempt"},
		),
		RetryFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "retry_failures_total",
				Help: "Total number of retry failures",
			},
			[]string{"operation"},
		),

		// DLQ метрики
		DLQMessagesSent: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dlq_messages_sent_total",
				Help: "Total number of messages sent to DLQ",
			},
			[]string{"topic", "reason"},
		),
		DLQMessagesProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dlq_messages_processed_total",
				Help: "Total number of DLQ messages processed",
			},
			[]string{"topic"},
		),

		// Database метрики
		DatabaseConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "database_connections",
				Help: "Number of database connections",
			},
			[]string{"state"},
		),
		DatabaseQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "database_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
	}
}

// HTTPMiddleware создает middleware для HTTP метрик
func (m *Metrics) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем ResponseWriter для отслеживания статуса и размера
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		status := http.StatusText(wrapped.statusCode)

		// Обновляем метрики
		m.HTTPRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		m.HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		m.HTTPRequestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
		m.HTTPResponseSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(wrapped.size))
	})
}

// Handler возвращает HTTP handler для Prometheus метрик
func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

// responseWriter обертка для http.ResponseWriter
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}
