package config

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	apperrors "wbtest/internal/errors"
	"wbtest/internal/logger"
)

// Validator валидирует конфигурацию приложения
type Validator struct{}

// NewValidator создает новый валидатор конфигурации
func NewValidator() *Validator {
	return &Validator{}
}

// Validate проверяет корректность всей конфигурации
func (v *Validator) Validate(cfg *Config) error {
	var errors []string

	// Валидация основных секций
	if err := v.validateDatabase(&cfg.Database); err != nil {
		errors = append(errors, fmt.Sprintf("Database: %v", err))
	}

	if err := v.validateKafka(&cfg.Kafka); err != nil {
		errors = append(errors, fmt.Sprintf("Kafka: %v", err))
	}

	if err := v.validateCache(&cfg.Cache); err != nil {
		errors = append(errors, fmt.Sprintf("Cache: %v", err))
	}

	if err := v.validateHTTP(&cfg.HTTP); err != nil {
		errors = append(errors, fmt.Sprintf("HTTP: %v", err))
	}

	if err := v.validateRetry(&cfg.Retry); err != nil {
		errors = append(errors, fmt.Sprintf("Retry: %v", err))
	}

	if err := v.validateDLQ(&cfg.DLQ); err != nil {
		errors = append(errors, fmt.Sprintf("DLQ: %v", err))
	}

	if err := v.validateLogger(&cfg.Logger); err != nil {
		errors = append(errors, fmt.Sprintf("Logger: %v", err))
	}

	if err := v.validateMetrics(&cfg.Metrics); err != nil {
		errors = append(errors, fmt.Sprintf("Metrics: %v", err))
	}

	if len(errors) > 0 {
		return apperrors.NewWithCode(
			apperrors.ErrorTypeValidation,
			fmt.Sprintf("Configuration validation failed: %s", strings.Join(errors, "; ")),
			"CONFIG_VALIDATION_FAILED",
		)
	}

	return nil
}

// validateDatabase валидирует конфигурацию базы данных
func (v *Validator) validateDatabase(cfg *DatabaseConfig) error {
	var errors []string

	if cfg.Host == "" {
		errors = append(errors, "host is required")
	}

	if cfg.Port <= 0 || cfg.Port > 65535 {
		errors = append(errors, "port must be between 1 and 65535")
	}

	if cfg.User == "" {
		errors = append(errors, "user is required")
	}

	if cfg.Password == "" {
		errors = append(errors, "password is required")
	}

	if cfg.Database == "" {
		errors = append(errors, "database name is required")
	}

	if cfg.MaxOpenConns <= 0 {
		errors = append(errors, "max_open_conns must be greater than 0")
	}

	if cfg.MaxIdleConns <= 0 {
		errors = append(errors, "max_idle_conns must be greater than 0")
	}

	if cfg.ConnMaxLifetime <= 0 {
		errors = append(errors, "conn_max_lifetime must be greater than 0")
	}

	// Проверяем что max_idle_conns не больше max_open_conns
	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		errors = append(errors, "max_idle_conns cannot be greater than max_open_conns")
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

// validateKafka валидирует конфигурацию Kafka
func (v *Validator) validateKafka(cfg *KafkaConfig) error {
	var errors []string

	if len(cfg.Brokers) == 0 {
		errors = append(errors, "at least one broker is required")
	}

	// Проверяем формат каждого брокера
	for i, broker := range cfg.Brokers {
		if err := v.validateHostPort(broker); err != nil {
			errors = append(errors, fmt.Sprintf("broker %d (%s): %v", i, broker, err))
		}
	}

	if cfg.Topic == "" {
		errors = append(errors, "topic is required")
	}

	if cfg.GroupID == "" {
		errors = append(errors, "group_id is required")
	}

	if cfg.BatchSize <= 0 {
		errors = append(errors, "batch_size must be greater than 0")
	}

	if cfg.BatchTimeout <= 0 {
		errors = append(errors, "batch_timeout must be greater than 0")
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

// validateCache валидирует конфигурацию кеша
func (v *Validator) validateCache(cfg *CacheConfig) error {
	var errors []string

	if cfg.TTLMinutes <= 0 {
		errors = append(errors, "ttl_minutes must be greater than 0")
	}

	if cfg.MaxSize <= 0 {
		errors = append(errors, "max_size must be greater than 0")
	}

	if cfg.CleanupInterval <= 0 {
		errors = append(errors, "cleanup_interval must be greater than 0")
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

// validateHTTP валидирует конфигурацию HTTP сервера
func (v *Validator) validateHTTP(cfg *HTTPConfig) error {
	var errors []string

	if cfg.Port <= 0 || cfg.Port > 65535 {
		errors = append(errors, "port must be between 1 and 65535")
	}

	if cfg.ReadTimeout <= 0 {
		errors = append(errors, "read_timeout must be greater than 0")
	}

	if cfg.WriteTimeout <= 0 {
		errors = append(errors, "write_timeout must be greater than 0")
	}

	if cfg.IdleTimeout <= 0 {
		errors = append(errors, "idle_timeout must be greater than 0")
	}

	// Проверяем что read_timeout и write_timeout разумные
	if cfg.ReadTimeout > 5*time.Minute {
		errors = append(errors, "read_timeout should not exceed 5 minutes")
	}

	if cfg.WriteTimeout > 5*time.Minute {
		errors = append(errors, "write_timeout should not exceed 5 minutes")
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

// validateRetry валидирует конфигурацию retry механизма
func (v *Validator) validateRetry(cfg *RetryConfig) error {
	var errors []string

	if cfg.MaxAttempts <= 0 {
		errors = append(errors, "max_attempts must be greater than 0")
	}

	if cfg.MaxAttempts > 10 {
		errors = append(errors, "max_attempts should not exceed 10")
	}

	if cfg.InitialDelay <= 0 {
		errors = append(errors, "initial_delay must be greater than 0")
	}

	if cfg.MaxDelay <= 0 {
		errors = append(errors, "max_delay must be greater than 0")
	}

	if cfg.Multiplier <= 0 {
		errors = append(errors, "multiplier must be greater than 0")
	}

	if cfg.InitialDelay > cfg.MaxDelay {
		errors = append(errors, "initial_delay cannot be greater than max_delay")
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

// validateDLQ валидирует конфигурацию DLQ
func (v *Validator) validateDLQ(cfg *DLQConfig) error {
	var errors []string

	if cfg.Enabled && cfg.Topic == "" {
		errors = append(errors, "topic is required when DLQ is enabled")
	}

	if cfg.MaxRetries <= 0 {
		errors = append(errors, "max_retries must be greater than 0")
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

// validateLogger валидирует конфигурацию логгера
func (v *Validator) validateLogger(cfg *logger.Config) error {
	var errors []string

	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "warning": true, "error": true, "fatal": true, "panic": true,
	}

	if !validLevels[strings.ToLower(cfg.Level)] {
		errors = append(errors, fmt.Sprintf("invalid log level '%s', valid levels: debug, info, warn, error, fatal, panic", cfg.Level))
	}

	validFormats := map[string]bool{
		"json": true, "text": true,
	}

	if !validFormats[strings.ToLower(cfg.Format)] {
		errors = append(errors, fmt.Sprintf("invalid log format '%s', valid formats: json, text", cfg.Format))
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

// validateMetrics валидирует конфигурацию метрик
func (v *Validator) validateMetrics(cfg *MetricsConfig) error {
	var errors []string

	if cfg.Port <= 0 || cfg.Port > 65535 {
		errors = append(errors, "port must be between 1 and 65535")
	}

	if cfg.Path == "" {
		errors = append(errors, "path is required")
	}

	// Проверяем что path начинается с /
	if !strings.HasPrefix(cfg.Path, "/") {
		errors = append(errors, "path must start with '/'")
	}

	// Проверяем что path не содержит недопустимых символов
	pathRegex := regexp.MustCompile(`^/[a-zA-Z0-9/_-]*$`)
	if !pathRegex.MatchString(cfg.Path) {
		errors = append(errors, "path contains invalid characters")
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

// validateHostPort валидирует формат host:port
func (v *Validator) validateHostPort(addr string) error {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid host:port format: %v", err)
	}

	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %v", err)
	}

	if port <= 0 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

// validateURL валидирует URL
func (v *Validator) validateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	_, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	return nil
}
