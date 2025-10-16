package logger

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "default config",
			config: Config{
				Level:  "info",
				Format: "json",
			},
		},
		{
			name: "text format",
			config: Config{
				Level:  "debug",
				Format: "text",
			},
		},
		{
			name: "invalid level",
			config: Config{
				Level:  "invalid",
				Format: "json",
			},
		},
		{
			name: "invalid format",
			config: Config{
				Level:  "warn",
				Format: "invalid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.config)
			if logger == nil {
				t.Error("Logger is nil")
			}
			if logger.Logger == nil {
				t.Error("Logger.Logger is nil")
			}
		})
	}
}

func TestDefault(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Error("Default logger is nil")
	}
	if logger.Logger == nil {
		t.Error("Default logger.Logger is nil")
	}
}

func TestLogger_WithField(t *testing.T) {
	logger := Default()
	entry := logger.WithField("test", "value")
	if entry == nil {
		t.Error("WithField returned nil entry")
	}
}

func TestLogger_WithFields(t *testing.T) {
	logger := Default()
	fields := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	entry := logger.WithFields(fields)
	if entry == nil {
		t.Error("WithFields returned nil entry")
	}
}

func TestLogger_WithError(t *testing.T) {
	logger := Default()
	err := &testError{message: "test error"}
	entry := logger.WithError(err)
	if entry == nil {
		t.Error("WithError returned nil entry")
	}
}

// testError для тестирования
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}
