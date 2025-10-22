package config

import (
	"testing"
	"time"

	apperrors "wbtest/internal/errors"
	"wbtest/internal/logger"
)

func TestValidator_Validate(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Database: DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					User:            "test_user",
					Password:        "test_pass",
					Database:        "test_db",
					MaxOpenConns:    10,
					MaxIdleConns:    5,
					ConnMaxLifetime: 5 * time.Minute,
				},
				Kafka: KafkaConfig{
					Brokers:      []string{"localhost:9092"},
					Topic:        "test-topic",
					GroupID:      "test-group",
					BatchSize:    100,
					BatchTimeout: 100 * time.Millisecond,
				},
				Cache: CacheConfig{
					TTLMinutes:      60,
					MaxSize:         1000,
					CleanupInterval: 5 * time.Minute,
				},
				HTTP: HTTPConfig{
					Port:         8080,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				Retry: RetryConfig{
					MaxAttempts:  3,
					InitialDelay: 100 * time.Millisecond,
					MaxDelay:     5 * time.Second,
					Multiplier:   2.0,
				},
				DLQ: DLQConfig{
					Enabled:    true,
					Topic:      "dlq-topic",
					MaxRetries: 5,
				},
				Logger: logger.Config{
					Level:  "info",
					Format: "json",
				},
				Metrics: MetricsConfig{
					Enabled: true,
					Port:    9090,
					Path:    "/metrics",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid database config",
			config: &Config{
				Database: DatabaseConfig{
					Host:            "",
					Port:            0,
					User:            "",
					Password:        "",
					Database:        "",
					MaxOpenConns:    0,
					MaxIdleConns:    0,
					ConnMaxLifetime: 0,
				},
				Kafka: KafkaConfig{
					Brokers:      []string{"localhost:9092"},
					Topic:        "test-topic",
					GroupID:      "test-group",
					BatchSize:    100,
					BatchTimeout: 100 * time.Millisecond,
				},
				Cache: CacheConfig{
					TTLMinutes:      60,
					MaxSize:         1000,
					CleanupInterval: 5 * time.Minute,
				},
				HTTP: HTTPConfig{
					Port:         8080,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				Retry: RetryConfig{
					MaxAttempts:  3,
					InitialDelay: 100 * time.Millisecond,
					MaxDelay:     5 * time.Second,
					Multiplier:   2.0,
				},
				DLQ: DLQConfig{
					Enabled:    true,
					Topic:      "dlq-topic",
					MaxRetries: 5,
				},
				Logger: logger.Config{
					Level:  "info",
					Format: "json",
				},
				Metrics: MetricsConfig{
					Enabled: true,
					Port:    9090,
					Path:    "/metrics",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid kafka config",
			config: &Config{
				Database: DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					User:            "test_user",
					Password:        "test_pass",
					Database:        "test_db",
					MaxOpenConns:    10,
					MaxIdleConns:    5,
					ConnMaxLifetime: 5 * time.Minute,
				},
				Kafka: KafkaConfig{
					Brokers:      []string{},
					Topic:        "",
					GroupID:      "",
					BatchSize:    0,
					BatchTimeout: 0,
				},
				Cache: CacheConfig{
					TTLMinutes:      60,
					MaxSize:         1000,
					CleanupInterval: 5 * time.Minute,
				},
				HTTP: HTTPConfig{
					Port:         8080,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				Retry: RetryConfig{
					MaxAttempts:  3,
					InitialDelay: 100 * time.Millisecond,
					MaxDelay:     5 * time.Second,
					Multiplier:   2.0,
				},
				DLQ: DLQConfig{
					Enabled:    true,
					Topic:      "dlq-topic",
					MaxRetries: 5,
				},
				Logger: logger.Config{
					Level:  "info",
					Format: "json",
				},
				Metrics: MetricsConfig{
					Enabled: true,
					Port:    9090,
					Path:    "/metrics",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid logger config",
			config: &Config{
				Database: DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					User:            "test_user",
					Password:        "test_pass",
					Database:        "test_db",
					MaxOpenConns:    10,
					MaxIdleConns:    5,
					ConnMaxLifetime: 5 * time.Minute,
				},
				Kafka: KafkaConfig{
					Brokers:      []string{"localhost:9092"},
					Topic:        "test-topic",
					GroupID:      "test-group",
					BatchSize:    100,
					BatchTimeout: 100 * time.Millisecond,
				},
				Cache: CacheConfig{
					TTLMinutes:      60,
					MaxSize:         1000,
					CleanupInterval: 5 * time.Minute,
				},
				HTTP: HTTPConfig{
					Port:         8080,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				Retry: RetryConfig{
					MaxAttempts:  3,
					InitialDelay: 100 * time.Millisecond,
					MaxDelay:     5 * time.Second,
					Multiplier:   2.0,
				},
				DLQ: DLQConfig{
					Enabled:    true,
					Topic:      "dlq-topic",
					MaxRetries: 5,
				},
				Logger: logger.Config{
					Level:  "invalid_level",
					Format: "invalid_format",
				},
				Metrics: MetricsConfig{
					Enabled: true,
					Port:    9090,
					Path:    "/metrics",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Если ожидается ошибка, проверяем что это AppError
			if tt.wantErr && err != nil {
				if appErr, ok := err.(*apperrors.AppError); !ok {
					t.Errorf("Expected AppError, got %T", err)
				} else if appErr.Type != apperrors.ErrorTypeValidation {
					t.Errorf("Expected ErrorTypeValidation, got %s", appErr.Type)
				}
			}
		})
	}
}

func TestValidator_validateDatabase(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		config  DatabaseConfig
		wantErr bool
	}{
		{
			name: "valid database config",
			config: DatabaseConfig{
				Host:            "localhost",
				Port:            5432,
				User:            "test_user",
				Password:        "test_pass",
				Database:        "test_db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "empty host",
			config: DatabaseConfig{
				Host:            "",
				Port:            5432,
				User:            "test_user",
				Password:        "test_pass",
				Database:        "test_db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: DatabaseConfig{
				Host:            "localhost",
				Port:            0,
				User:            "test_user",
				Password:        "test_pass",
				Database:        "test_db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "max_idle_conns greater than max_open_conns",
			config: DatabaseConfig{
				Host:            "localhost",
				Port:            5432,
				User:            "test_user",
				Password:        "test_pass",
				Database:        "test_db",
				MaxOpenConns:    5,
				MaxIdleConns:    10,
				ConnMaxLifetime: 5 * time.Minute,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateDatabase(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDatabase() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_validateKafka(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		config  KafkaConfig
		wantErr bool
	}{
		{
			name: "valid kafka config",
			config: KafkaConfig{
				Brokers:      []string{"localhost:9092"},
				Topic:        "test-topic",
				GroupID:      "test-group",
				BatchSize:    100,
				BatchTimeout: 100 * time.Millisecond,
			},
			wantErr: false,
		},
		{
			name: "empty brokers",
			config: KafkaConfig{
				Brokers:      []string{},
				Topic:        "test-topic",
				GroupID:      "test-group",
				BatchSize:    100,
				BatchTimeout: 100 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "invalid broker format",
			config: KafkaConfig{
				Brokers:      []string{"invalid-broker"},
				Topic:        "test-topic",
				GroupID:      "test-group",
				BatchSize:    100,
				BatchTimeout: 100 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "empty topic",
			config: KafkaConfig{
				Brokers:      []string{"localhost:9092"},
				Topic:        "",
				GroupID:      "test-group",
				BatchSize:    100,
				BatchTimeout: 100 * time.Millisecond,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateKafka(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateKafka() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_validateMetrics(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		config  MetricsConfig
		wantErr bool
	}{
		{
			name: "valid metrics config",
			config: MetricsConfig{
				Enabled: true,
				Port:    9090,
				Path:    "/metrics",
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			config: MetricsConfig{
				Enabled: true,
				Port:    0,
				Path:    "/metrics",
			},
			wantErr: true,
		},
		{
			name: "empty path",
			config: MetricsConfig{
				Enabled: true,
				Port:    9090,
				Path:    "",
			},
			wantErr: true,
		},
		{
			name: "path without leading slash",
			config: MetricsConfig{
				Enabled: true,
				Port:    9090,
				Path:    "metrics",
			},
			wantErr: true,
		},
		{
			name: "path with invalid characters",
			config: MetricsConfig{
				Enabled: true,
				Port:    9090,
				Path:    "/metrics@#$",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateMetrics(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_validateHostPort(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{"valid address", "localhost:9092", false},
		{"valid IP", "127.0.0.1:5432", false},
		{"invalid format", "localhost", true},
		{"empty host", ":9092", true},
		{"invalid port", "localhost:abc", true},
		{"port too high", "localhost:65536", true},
		{"port zero", "localhost:0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateHostPort(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateHostPort() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
