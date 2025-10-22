package config

import (
	"os"
	"testing"
	"time"

	"wbtest/internal/logger"
)

func TestLoad(t *testing.T) {
	// Сохраняем текущие переменные окружения
	originalEnv := make(map[string]string)
	envVars := []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"KAFKA_BROKERS", "KAFKA_TOPIC", "KAFKA_GROUP_ID",
		"HTTP_PORT", "CACHE_MAX_SIZE", "CACHE_TTL_MINUTES",
		"RETRY_MAX_ATTEMPTS", "RETRY_INITIAL_DELAY", "RETRY_MAX_DELAY", "RETRY_MULTIPLIER",
		"DLQ_ENABLED", "DLQ_TOPIC", "DLQ_MAX_RETRIES",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
	}

	// Очищаем переменные окружения
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	defer func() {
		// Восстанавливаем оригинальные переменные окружения
		for _, envVar := range envVars {
			if value, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, value)
			} else {
				os.Unsetenv(envVar)
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected *Config
	}{
		{
			name:    "default values",
			envVars: map[string]string{},
			expected: &Config{
				Database: DatabaseConfig{
					Host:     "127.0.0.1",
					Port:     5432,
					User:     "orders_user",
					Password: "orders_pass",
					Database: "orders_db",
				},
				Kafka: KafkaConfig{
					Brokers: []string{"localhost:9092"},
					Topic:   "orders",
					GroupID: "order-service",
				},
				HTTP: HTTPConfig{
					Port: 8082,
				},
				Cache: CacheConfig{
					MaxSize:    1000,
					TTLMinutes: 60, // default value
				},
				Retry: RetryConfig{
					MaxAttempts:  3,
					InitialDelay: 1 * time.Second,
					MaxDelay:     30 * time.Second,
					Multiplier:   2.0,
				},
				DLQ: DLQConfig{
					Enabled:    true,
					Topic:      "orders-dlq",
					MaxRetries: 3,
				},
				Logger: logger.Config{
					Level:  "info",
					Format: "json",
				},
			},
		},
		{
			name: "custom values",
			envVars: map[string]string{
				"DB_HOST":             "custom-host",
				"DB_PORT":             "5433",
				"DB_USER":             "custom-user",
				"DB_PASSWORD":         "custom-password",
				"DB_NAME":             "custom-db",
				"KAFKA_BROKERS":       "kafka1:9092,kafka2:9092",
				"KAFKA_TOPIC":         "custom-orders",
				"KAFKA_GROUP_ID":      "custom-group",
				"HTTP_PORT":           "9090",
				"CACHE_MAX_SIZE":      "2000",
				"CACHE_TTL_MINUTES":   "120",
				"RETRY_MAX_ATTEMPTS":  "5",
				"RETRY_INITIAL_DELAY": "2s",
				"RETRY_MAX_DELAY":     "60s",
				"RETRY_MULTIPLIER":    "1.5",
				"DLQ_ENABLED":         "false",
				"DLQ_TOPIC":           "custom-dlq",
				"DLQ_MAX_RETRIES":     "5",
			},
			expected: &Config{
				Database: DatabaseConfig{
					Host:     "custom-host",
					Port:     5433,
					User:     "custom-user",
					Password: "custom-password",
					Database: "custom-db",
				},
				Kafka: KafkaConfig{
					Brokers: []string{"kafka1:9092", "kafka2:9092"},
					Topic:   "custom-orders",
					GroupID: "custom-group",
				},
				HTTP: HTTPConfig{
					Port: 9090,
				},
				Cache: CacheConfig{
					MaxSize:    2000,
					TTLMinutes: 120, // 2 hours in minutes
				},
				Retry: RetryConfig{
					MaxAttempts:  5,
					InitialDelay: 2 * time.Second,
					MaxDelay:     60 * time.Second,
					Multiplier:   1.5,
				},
				DLQ: DLQConfig{
					Enabled:    false,
					Topic:      "custom-dlq",
					MaxRetries: 5,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем переменные окружения для теста
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Загружаем конфигурацию
			config, err := Load()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Проверяем значения
			if config.Database.Host != tt.expected.Database.Host {
				t.Errorf("Database.Host = %v, want %v", config.Database.Host, tt.expected.Database.Host)
			}
			if config.Database.Port != tt.expected.Database.Port {
				t.Errorf("Database.Port = %v, want %v", config.Database.Port, tt.expected.Database.Port)
			}
			if config.Database.User != tt.expected.Database.User {
				t.Errorf("Database.User = %v, want %v", config.Database.User, tt.expected.Database.User)
			}
			if config.Database.Password != tt.expected.Database.Password {
				t.Errorf("Database.Password = %v, want %v", config.Database.Password, tt.expected.Database.Password)
			}
			if config.Database.Database != tt.expected.Database.Database {
				t.Errorf("Database.Database = %v, want %v", config.Database.Database, tt.expected.Database.Database)
			}

			// Проверяем Kafka конфигурацию
			if len(config.Kafka.Brokers) != len(tt.expected.Kafka.Brokers) {
				t.Errorf("Kafka.Brokers length = %v, want %v", len(config.Kafka.Brokers), len(tt.expected.Kafka.Brokers))
			}
			for i, broker := range config.Kafka.Brokers {
				if broker != tt.expected.Kafka.Brokers[i] {
					t.Errorf("Kafka.Brokers[%d] = %v, want %v", i, broker, tt.expected.Kafka.Brokers[i])
				}
			}
			if config.Kafka.Topic != tt.expected.Kafka.Topic {
				t.Errorf("Kafka.Topic = %v, want %v", config.Kafka.Topic, tt.expected.Kafka.Topic)
			}
			if config.Kafka.GroupID != tt.expected.Kafka.GroupID {
				t.Errorf("Kafka.GroupID = %v, want %v", config.Kafka.GroupID, tt.expected.Kafka.GroupID)
			}

			// Проверяем HTTP конфигурацию
			if config.HTTP.Port != tt.expected.HTTP.Port {
				t.Errorf("HTTP.Port = %v, want %v", config.HTTP.Port, tt.expected.HTTP.Port)
			}

			// Проверяем Cache конфигурацию
			if config.Cache.MaxSize != tt.expected.Cache.MaxSize {
				t.Errorf("Cache.MaxSize = %v, want %v", config.Cache.MaxSize, tt.expected.Cache.MaxSize)
			}
			if config.Cache.TTLMinutes != tt.expected.Cache.TTLMinutes {
				t.Errorf("Cache.TTLMinutes = %v, want %v", config.Cache.TTLMinutes, tt.expected.Cache.TTLMinutes)
			}

			// Проверяем Retry конфигурацию
			if config.Retry.MaxAttempts != tt.expected.Retry.MaxAttempts {
				t.Errorf("Retry.MaxAttempts = %v, want %v", config.Retry.MaxAttempts, tt.expected.Retry.MaxAttempts)
			}
			if config.Retry.InitialDelay != tt.expected.Retry.InitialDelay {
				t.Errorf("Retry.InitialDelay = %v, want %v", config.Retry.InitialDelay, tt.expected.Retry.InitialDelay)
			}
			if config.Retry.MaxDelay != tt.expected.Retry.MaxDelay {
				t.Errorf("Retry.MaxDelay = %v, want %v", config.Retry.MaxDelay, tt.expected.Retry.MaxDelay)
			}
			if config.Retry.Multiplier != tt.expected.Retry.Multiplier {
				t.Errorf("Retry.Multiplier = %v, want %v", config.Retry.Multiplier, tt.expected.Retry.Multiplier)
			}

			// Проверяем DLQ конфигурацию
			if config.DLQ.Enabled != tt.expected.DLQ.Enabled {
				t.Errorf("DLQ.Enabled = %v, want %v", config.DLQ.Enabled, tt.expected.DLQ.Enabled)
			}
			if config.DLQ.Topic != tt.expected.DLQ.Topic {
				t.Errorf("DLQ.Topic = %v, want %v", config.DLQ.Topic, tt.expected.DLQ.Topic)
			}
			if config.DLQ.MaxRetries != tt.expected.DLQ.MaxRetries {
				t.Errorf("DLQ.MaxRetries = %v, want %v", config.DLQ.MaxRetries, tt.expected.DLQ.MaxRetries)
			}

			// Очищаем переменные окружения после теста
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "valid integer",
			key:          "TEST_INT",
			defaultValue: 42,
			envValue:     "100",
			expected:     100,
		},
		{
			name:         "invalid integer",
			key:          "TEST_INVALID_INT",
			defaultValue: 42,
			envValue:     "not-a-number",
			expected:     42,
		},
		{
			name:         "empty value",
			key:          "TEST_EMPTY_INT",
			defaultValue: 42,
			envValue:     "",
			expected:     42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}
			defer os.Unsetenv(tt.key)

			result := getEnvAsInt(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvAsInt(%s, %d) = %d, want %d", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetEnvAsDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		expected     time.Duration
	}{
		{
			name:         "valid duration",
			key:          "TEST_DURATION",
			defaultValue: time.Second,
			envValue:     "5s",
			expected:     5 * time.Second,
		},
		{
			name:         "invalid duration",
			key:          "TEST_INVALID_DURATION",
			defaultValue: time.Second,
			envValue:     "not-a-duration",
			expected:     time.Second,
		},
		{
			name:         "empty value",
			key:          "TEST_EMPTY_DURATION",
			defaultValue: time.Second,
			envValue:     "",
			expected:     time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}
			defer os.Unsetenv(tt.key)

			result := getEnvAsDuration(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvAsDuration(%s, %v) = %v, want %v", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetEnvAsFloat(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue float64
		envValue     string
		expected     float64
	}{
		{
			name:         "valid float",
			key:          "TEST_FLOAT",
			defaultValue: 2.0,
			envValue:     "3.14",
			expected:     3.14,
		},
		{
			name:         "invalid float",
			key:          "TEST_INVALID_FLOAT",
			defaultValue: 2.0,
			envValue:     "not-a-float",
			expected:     2.0,
		},
		{
			name:         "empty value",
			key:          "TEST_EMPTY_FLOAT",
			defaultValue: 2.0,
			envValue:     "",
			expected:     2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}
			defer os.Unsetenv(tt.key)

			result := getEnvAsFloat(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvAsFloat(%s, %f) = %f, want %f", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetEnvAsBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		expected     bool
	}{
		{
			name:         "true value",
			key:          "TEST_BOOL_TRUE",
			defaultValue: false,
			envValue:     "true",
			expected:     true,
		},
		{
			name:         "false value",
			key:          "TEST_BOOL_FALSE",
			defaultValue: true,
			envValue:     "false",
			expected:     false,
		},
		{
			name:         "invalid value",
			key:          "TEST_BOOL_INVALID",
			defaultValue: false,
			envValue:     "maybe",
			expected:     false,
		},
		{
			name:         "empty value",
			key:          "TEST_BOOL_EMPTY",
			defaultValue: false,
			envValue:     "",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}
			defer os.Unsetenv(tt.key)

			result := getEnvAsBool(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvAsBool(%s, %t) = %t, want %t", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}
