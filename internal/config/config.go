package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Database   DatabaseConfig
	Kafka      KafkaConfig
	HTTP       HTTPConfig
	Cache      CacheConfig
	App        AppConfig
	Generator  GeneratorConfig
	Validation ValidationConfig
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type KafkaConfig struct {
	Brokers          []string
	Topic            string
	GroupID          string
	AutoOffsetReset  string
	EnableAutoCommit bool
	SessionTimeoutMs int
}

type HTTPConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type CacheConfig struct {
	MaxSize         int
	TTL             time.Duration
	CleanupInterval time.Duration
}

type AppConfig struct {
	GracefulShutdownTimeout time.Duration
	LogLevel                string
	Environment             string
	DatabaseLoadTimeout     time.Duration
	ShutdownWaitTimeout     time.Duration
}

type GeneratorConfig struct {
	MaxOrdersCount   int
	MaxItemsPerOrder int
	MinPrice         int
	MaxPrice         int
	MaxSale          int
}

type ValidationConfig struct {
	OrderUIDMinLength    int
	OrderUIDMaxLength    int
	TrackNumberMinLength int
	TrackNumberMaxLength int
	MaxPaymentAmount     int
	MaxItemsPerOrder     int
	MaxItemPrice         int
}

func Load() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "127.0.0.1"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "orders_user"),
			Password:        getEnv("DB_PASSWORD", "orders_pass"),
			DBName:          getEnv("DB_NAME", "orders_db"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Kafka: KafkaConfig{
			Brokers:          strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
			Topic:            getEnv("KAFKA_TOPIC", "orders"),
			GroupID:          getEnv("KAFKA_GROUP_ID", "order-service"),
			AutoOffsetReset:  getEnv("KAFKA_AUTO_OFFSET_RESET", "earliest"),
			EnableAutoCommit: getEnvAsBool("KAFKA_ENABLE_AUTO_COMMIT", true),
			SessionTimeoutMs: getEnvAsInt("KAFKA_SESSION_TIMEOUT_MS", 30000),
		},
		HTTP: HTTPConfig{
			Port:         getEnvAsInt("HTTP_PORT", 8082),
			ReadTimeout:  getEnvAsDuration("HTTP_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvAsDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvAsDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		},
		Cache: CacheConfig{
			MaxSize:         getEnvAsInt("CACHE_MAX_SIZE", 1000),
			TTL:             getEnvAsDuration("CACHE_TTL", 24*time.Hour),
			CleanupInterval: getEnvAsDuration("CACHE_CLEANUP_INTERVAL", 5*time.Minute),
		},
		App: AppConfig{
			GracefulShutdownTimeout: getEnvAsDuration("GRACEFUL_SHUTDOWN_TIMEOUT", 30*time.Second),
			LogLevel:                getEnv("LOG_LEVEL", "info"),
			Environment:             getEnv("ENVIRONMENT", "development"),
			DatabaseLoadTimeout:     getEnvAsDuration("DB_LOAD_TIMEOUT", 10*time.Second),
			ShutdownWaitTimeout:     getEnvAsDuration("SHUTDOWN_WAIT_TIMEOUT", 5*time.Second),
		},
		Generator: GeneratorConfig{
			MaxOrdersCount:   getEnvAsInt("GENERATOR_MAX_ORDERS", 10000),
			MaxItemsPerOrder: getEnvAsInt("GENERATOR_MAX_ITEMS_PER_ORDER", 5),
			MinPrice:         getEnvAsInt("GENERATOR_MIN_PRICE", 50),
			MaxPrice:         getEnvAsInt("GENERATOR_MAX_PRICE", 5000),
			MaxSale:          getEnvAsInt("GENERATOR_MAX_SALE", 50),
		},
		Validation: ValidationConfig{
			OrderUIDMinLength:    getEnvAsInt("VALIDATION_ORDER_UID_MIN_LENGTH", 10),
			OrderUIDMaxLength:    getEnvAsInt("VALIDATION_ORDER_UID_MAX_LENGTH", 50),
			TrackNumberMinLength: getEnvAsInt("VALIDATION_TRACK_NUMBER_MIN_LENGTH", 5),
			TrackNumberMaxLength: getEnvAsInt("VALIDATION_TRACK_NUMBER_MAX_LENGTH", 20),
			MaxPaymentAmount:     getEnvAsInt("VALIDATION_MAX_PAYMENT_AMOUNT", 1000000),
			MaxItemsPerOrder:     getEnvAsInt("VALIDATION_MAX_ITEMS_PER_ORDER", 100),
			MaxItemPrice:         getEnvAsInt("VALIDATION_MAX_ITEM_PRICE", 100000),
		},
	}
}

func (c *Config) DatabaseURL() string {
	return "postgres://" + c.Database.User + ":" + c.Database.Password + "@" +
		c.Database.Host + ":" + strconv.Itoa(c.Database.Port) + "/" +
		c.Database.DBName + "?sslmode=" + c.Database.SSLMode
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
