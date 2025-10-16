package errors

import (
	"fmt"
	"net/http"
)

// ErrorType тип ошибки
type ErrorType string

const (
	// ErrorTypeValidation ошибка валидации
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeDatabase ошибка базы данных
	ErrorTypeDatabase ErrorType = "database"
	// ErrorTypeKafka ошибка Kafka
	ErrorTypeKafka ErrorType = "kafka"
	// ErrorTypeCache ошибка кеша
	ErrorTypeCache ErrorType = "cache"
	// ErrorTypeHTTP ошибка HTTP
	ErrorTypeHTTP ErrorType = "http"
	// ErrorTypeRetry ошибка retry
	ErrorTypeRetry ErrorType = "retry"
	// ErrorTypeDLQ ошибка DLQ
	ErrorTypeDLQ ErrorType = "dlq"
	// ErrorTypeInternal внутренняя ошибка
	ErrorTypeInternal ErrorType = "internal"
	// ErrorTypeNotFound ресурс не найден
	ErrorTypeNotFound ErrorType = "not_found"
	// ErrorTypeTimeout таймаут
	ErrorTypeTimeout ErrorType = "timeout"
)

// AppError структура для приложения ошибок
type AppError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	Code       string    `json:"code,omitempty"`
	Details    string    `json:"details,omitempty"`
	HTTPStatus int       `json:"-"`
	Cause      error     `json:"-"`
}

// Error реализует интерфейс error
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap возвращает причину ошибки
func (e *AppError) Unwrap() error {
	return e.Cause
}

// New создает новую ошибку
func New(errorType ErrorType, message string) *AppError {
	return &AppError{
		Type:       errorType,
		Message:    message,
		HTTPStatus: getDefaultHTTPStatus(errorType),
	}
}

// NewWithCode создает новую ошибку с кодом
func NewWithCode(errorType ErrorType, message, code string) *AppError {
	return &AppError{
		Type:       errorType,
		Message:    message,
		Code:       code,
		HTTPStatus: getDefaultHTTPStatus(errorType),
	}
}

// Wrap оборачивает существующую ошибку
func Wrap(err error, errorType ErrorType, message string) *AppError {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	return &AppError{
		Type:       errorType,
		Message:    message,
		HTTPStatus: getDefaultHTTPStatus(errorType),
		Cause:      err,
	}
}

// WrapWithCode оборачивает существующую ошибку с кодом
func WrapWithCode(err error, errorType ErrorType, message, code string) *AppError {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	return &AppError{
		Type:       errorType,
		Message:    message,
		Code:       code,
		HTTPStatus: getDefaultHTTPStatus(errorType),
		Cause:      err,
	}
}

// getDefaultHTTPStatus возвращает HTTP статус по умолчанию для типа ошибки
func getDefaultHTTPStatus(errorType ErrorType) int {
	switch errorType {
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeTimeout:
		return http.StatusRequestTimeout
	case ErrorTypeHTTP:
		return http.StatusInternalServerError
	case ErrorTypeDatabase, ErrorTypeKafka, ErrorTypeCache, ErrorTypeRetry, ErrorTypeDLQ, ErrorTypeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Предопределенные ошибки

// Validation errors
var (
	ErrOrderValidationFailed = NewWithCode(
		ErrorTypeValidation,
		"Order validation failed",
		"ORDER_VALIDATION_FAILED",
	)

	ErrInvalidOrderUID = NewWithCode(
		ErrorTypeValidation,
		"Invalid order UID",
		"INVALID_ORDER_UID",
	)

	ErrInvalidTrackNumber = NewWithCode(
		ErrorTypeValidation,
		"Invalid track number",
		"INVALID_TRACK_NUMBER",
	)
)

// Database errors
var (
	ErrDatabaseConnection = NewWithCode(
		ErrorTypeDatabase,
		"Database connection failed",
		"DB_CONNECTION_FAILED",
	)

	ErrOrderNotFound = NewWithCode(
		ErrorTypeNotFound,
		"Order not found",
		"ORDER_NOT_FOUND",
	)

	ErrOrderSaveFailed = NewWithCode(
		ErrorTypeDatabase,
		"Failed to save order",
		"ORDER_SAVE_FAILED",
	)
)

// Kafka errors
var (
	ErrKafkaConnection = NewWithCode(
		ErrorTypeKafka,
		"Kafka connection failed",
		"KAFKA_CONNECTION_FAILED",
	)

	ErrMessageConsumeFailed = NewWithCode(
		ErrorTypeKafka,
		"Failed to consume message",
		"MESSAGE_CONSUME_FAILED",
	)

	ErrMessageProduceFailed = NewWithCode(
		ErrorTypeKafka,
		"Failed to produce message",
		"MESSAGE_PRODUCE_FAILED",
	)
)

// Cache errors
var (
	ErrCacheConnection = NewWithCode(
		ErrorTypeCache,
		"Cache connection failed",
		"CACHE_CONNECTION_FAILED",
	)

	ErrCacheSetFailed = NewWithCode(
		ErrorTypeCache,
		"Failed to set cache value",
		"CACHE_SET_FAILED",
	)

	ErrCacheGetFailed = NewWithCode(
		ErrorTypeCache,
		"Failed to get cache value",
		"CACHE_GET_FAILED",
	)
)

// Retry errors
var (
	ErrRetryExhausted = NewWithCode(
		ErrorTypeRetry,
		"Retry attempts exhausted",
		"RETRY_EXHAUSTED",
	)

	ErrRetryFailed = NewWithCode(
		ErrorTypeRetry,
		"Retry operation failed",
		"RETRY_FAILED",
	)
)

// DLQ errors
var (
	ErrDLQSendFailed = NewWithCode(
		ErrorTypeDLQ,
		"Failed to send message to DLQ",
		"DLQ_SEND_FAILED",
	)

	ErrDLQProcessFailed = NewWithCode(
		ErrorTypeDLQ,
		"Failed to process DLQ message",
		"DLQ_PROCESS_FAILED",
	)
)
