package dlq

import (
	"testing"
	"time"

	"wbtest/internal/config"
)

func TestDLQService_SendToDLQ(t *testing.T) {
	config := &config.DLQConfig{
		Enabled:    true,
		Topic:      "test-dlq",
		MaxRetries: 3,
	}

	// Note: This test would require a real Kafka instance or mocking
	// For now, we'll test the NoOpDLQService
	t.Run("noop service when disabled", func(t *testing.T) {
		config.Enabled = false
		service := &NoOpDLQService{}

		message := []byte("test message")
		reason := "test reason"

		err := service.SendToDLQ(message, reason)
		if err != nil {
			t.Errorf("NoOpDLQService.SendToDLQ() error = %v", err)
		}
	})
}

func TestDLQService_ProcessDLQ(t *testing.T) {
	service := &NoOpDLQService{}

	err := service.ProcessDLQ()
	if err != nil {
		t.Errorf("NoOpDLQService.ProcessDLQ() error = %v", err)
	}
}

func TestDLQService_Close(t *testing.T) {
	service := &NoOpDLQService{}

	err := service.Close()
	if err != nil {
		t.Errorf("NoOpDLQService.Close() error = %v", err)
	}
}

func TestDLQMessage_MarshalUnmarshal(t *testing.T) {
	originalMessage := DLQMessage{
		OriginalMessage: []byte("test message"),
		Reason:          "test reason",
		Timestamp:       time.Now(),
		RetryCount:      1,
	}

	// Marshal
	data, err := originalMessage.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to marshal DLQMessage: %v", err)
	}

	// Unmarshal
	var unmarshaledMessage DLQMessage
	err = unmarshaledMessage.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("Failed to unmarshal DLQMessage: %v", err)
	}

	// Compare fields
	if string(unmarshaledMessage.OriginalMessage) != string(originalMessage.OriginalMessage) {
		t.Errorf("OriginalMessage mismatch: got %s, want %s",
			string(unmarshaledMessage.OriginalMessage), string(originalMessage.OriginalMessage))
	}

	if unmarshaledMessage.Reason != originalMessage.Reason {
		t.Errorf("Reason mismatch: got %s, want %s", unmarshaledMessage.Reason, originalMessage.Reason)
	}

	if unmarshaledMessage.RetryCount != originalMessage.RetryCount {
		t.Errorf("RetryCount mismatch: got %d, want %d", unmarshaledMessage.RetryCount, originalMessage.RetryCount)
	}

	// Timestamp comparison (with small tolerance for precision)
	timeDiff := originalMessage.Timestamp.Sub(unmarshaledMessage.Timestamp)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > time.Millisecond {
		t.Errorf("Timestamp mismatch: got %v, want %v", unmarshaledMessage.Timestamp, originalMessage.Timestamp)
	}
}
