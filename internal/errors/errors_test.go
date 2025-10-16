package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(ErrorTypeValidation, "test message")

	if err.Type != ErrorTypeValidation {
		t.Errorf("Expected type %s, got %s", ErrorTypeValidation, err.Type)
	}

	if err.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", err.Message)
	}

	if err.HTTPStatus != http.StatusBadRequest {
		t.Errorf("Expected HTTP status %d, got %d", http.StatusBadRequest, err.HTTPStatus)
	}
}

func TestNewWithCode(t *testing.T) {
	err := NewWithCode(ErrorTypeDatabase, "test message", "TEST_CODE")

	if err.Type != ErrorTypeDatabase {
		t.Errorf("Expected type %s, got %s", ErrorTypeDatabase, err.Type)
	}

	if err.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", err.Message)
	}

	if err.Code != "TEST_CODE" {
		t.Errorf("Expected code 'TEST_CODE', got '%s'", err.Code)
	}

	if err.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("Expected HTTP status %d, got %d", http.StatusInternalServerError, err.HTTPStatus)
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := Wrap(originalErr, ErrorTypeKafka, "wrapped message")

	if wrappedErr.Type != ErrorTypeKafka {
		t.Errorf("Expected type %s, got %s", ErrorTypeKafka, wrappedErr.Type)
	}

	if wrappedErr.Message != "wrapped message" {
		t.Errorf("Expected message 'wrapped message', got '%s'", wrappedErr.Message)
	}

	if wrappedErr.Cause != originalErr {
		t.Error("Cause is not the original error")
	}

	if wrappedErr.Unwrap() != originalErr {
		t.Error("Unwrap() does not return the original error")
	}
}

func TestWrapWithCode(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := WrapWithCode(originalErr, ErrorTypeCache, "wrapped message", "WRAP_CODE")

	if wrappedErr.Type != ErrorTypeCache {
		t.Errorf("Expected type %s, got %s", ErrorTypeCache, wrappedErr.Type)
	}

	if wrappedErr.Message != "wrapped message" {
		t.Errorf("Expected message 'wrapped message', got '%s'", wrappedErr.Message)
	}

	if wrappedErr.Code != "WRAP_CODE" {
		t.Errorf("Expected code 'WRAP_CODE', got '%s'", wrappedErr.Code)
	}

	if wrappedErr.Cause != originalErr {
		t.Error("Cause is not the original error")
	}
}

func TestWrapNil(t *testing.T) {
	wrappedErr := Wrap(nil, ErrorTypeInternal, "wrapped message")
	if wrappedErr != nil {
		t.Error("Wrap(nil) should return nil")
	}
}

func TestWrapAppError(t *testing.T) {
	originalErr := New(ErrorTypeValidation, "original app error")
	wrappedErr := Wrap(originalErr, ErrorTypeKafka, "wrapped message")

	// Wrap should return the original AppError unchanged
	if wrappedErr != originalErr {
		t.Error("Wrap(AppError) should return the original AppError")
	}
}

func TestError(t *testing.T) {
	err := New(ErrorTypeDatabase, "test message")
	errorString := err.Error()

	expected := "database: test message"
	if errorString != expected {
		t.Errorf("Expected error string '%s', got '%s'", expected, errorString)
	}
}

func TestErrorWithCause(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := Wrap(originalErr, ErrorTypeKafka, "wrapped message")
	errorString := wrappedErr.Error()

	expected := "kafka: wrapped message (caused by: original error)"
	if errorString != expected {
		t.Errorf("Expected error string '%s', got '%s'", expected, errorString)
	}
}

func TestGetDefaultHTTPStatus(t *testing.T) {
	tests := []struct {
		errorType      ErrorType
		expectedStatus int
	}{
		{ErrorTypeValidation, http.StatusBadRequest},
		{ErrorTypeNotFound, http.StatusNotFound},
		{ErrorTypeTimeout, http.StatusRequestTimeout},
		{ErrorTypeDatabase, http.StatusInternalServerError},
		{ErrorTypeKafka, http.StatusInternalServerError},
		{ErrorTypeCache, http.StatusInternalServerError},
		{ErrorTypeRetry, http.StatusInternalServerError},
		{ErrorTypeDLQ, http.StatusInternalServerError},
		{ErrorTypeInternal, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		err := New(tt.errorType, "test")
		if err.HTTPStatus != tt.expectedStatus {
			t.Errorf("Expected HTTP status %d for %s, got %d", tt.expectedStatus, tt.errorType, err.HTTPStatus)
		}
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		err          *AppError
		expectedType ErrorType
		expectedCode string
	}{
		{ErrOrderValidationFailed, ErrorTypeValidation, "ORDER_VALIDATION_FAILED"},
		{ErrInvalidOrderUID, ErrorTypeValidation, "INVALID_ORDER_UID"},
		{ErrInvalidTrackNumber, ErrorTypeValidation, "INVALID_TRACK_NUMBER"},
		{ErrDatabaseConnection, ErrorTypeDatabase, "DB_CONNECTION_FAILED"},
		{ErrOrderNotFound, ErrorTypeNotFound, "ORDER_NOT_FOUND"},
		{ErrOrderSaveFailed, ErrorTypeDatabase, "ORDER_SAVE_FAILED"},
		{ErrKafkaConnection, ErrorTypeKafka, "KAFKA_CONNECTION_FAILED"},
		{ErrMessageConsumeFailed, ErrorTypeKafka, "MESSAGE_CONSUME_FAILED"},
		{ErrMessageProduceFailed, ErrorTypeKafka, "MESSAGE_PRODUCE_FAILED"},
		{ErrCacheConnection, ErrorTypeCache, "CACHE_CONNECTION_FAILED"},
		{ErrCacheSetFailed, ErrorTypeCache, "CACHE_SET_FAILED"},
		{ErrCacheGetFailed, ErrorTypeCache, "CACHE_GET_FAILED"},
		{ErrRetryExhausted, ErrorTypeRetry, "RETRY_EXHAUSTED"},
		{ErrRetryFailed, ErrorTypeRetry, "RETRY_FAILED"},
		{ErrDLQSendFailed, ErrorTypeDLQ, "DLQ_SEND_FAILED"},
		{ErrDLQProcessFailed, ErrorTypeDLQ, "DLQ_PROCESS_FAILED"},
	}

	for _, tt := range tests {
		if tt.err.Type != tt.expectedType {
			t.Errorf("Expected type %s for %s, got %s", tt.expectedType, tt.err.Code, tt.err.Type)
		}

		if tt.err.Code != tt.expectedCode {
			t.Errorf("Expected code %s, got %s", tt.expectedCode, tt.err.Code)
		}

		if tt.err.Message == "" {
			t.Error("Error message should not be empty")
		}
	}
}
