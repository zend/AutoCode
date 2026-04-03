package tui

import (
	"strings"
	"testing"
	"time"
)

func TestNewUserMessage(t *testing.T) {
	content := "Hello, world!"
	msg := NewUserMessage(content)

	if msg.Role != UserMessage {
		t.Errorf("expected Role UserMessage, got %v", msg.Role)
	}
	if msg.Content != content {
		t.Errorf("expected Content %q, got %q", content, msg.Content)
	}
	if msg.Timestamp.IsZero() {
		t.Error("expected Timestamp to be set")
	}
}

func TestNewAssistantMessage(t *testing.T) {
	content := "Hello! How can I help you?"
	msg := NewAssistantMessage(content)

	if msg.Role != AssistantMessage {
		t.Errorf("expected Role AssistantMessage, got %v", msg.Role)
	}
	if msg.Content != content {
		t.Errorf("expected Content %q, got %q", content, msg.Content)
	}
	if msg.Timestamp.IsZero() {
		t.Error("expected Timestamp to be set")
	}
}

func TestMessage_AppendContent(t *testing.T) {
	msg := NewUserMessage("Hello")
	msg.AppendContent(", world!")

	expected := "Hello, world!"
	if msg.Content != expected {
		t.Errorf("expected Content %q, got %q", expected, msg.Content)
	}
}

func TestMessage_AppendContent_Empty(t *testing.T) {
	msg := NewUserMessage("")
	msg.AppendContent("test")

	if msg.Content != "test" {
		t.Errorf("expected Content 'test', got %q", msg.Content)
	}
}

func TestMessage_FormatTimestamp(t *testing.T) {
	// Create a message with a known timestamp
	msg := Message{
		Role:      UserMessage,
		Content:   "Test",
		Timestamp: time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC),
	}

	formatted := msg.FormatTimestamp()
	expected := "14:30:45"
	if formatted != expected {
		t.Errorf("expected FormatTimestamp %q, got %q", expected, formatted)
	}
}

func TestMessage_IsUser(t *testing.T) {
	userMsg := NewUserMessage("Hello")
	if !userMsg.IsUser() {
		t.Error("expected IsUser() to return true for user message")
	}
	if userMsg.IsAssistant() {
		t.Error("expected IsAssistant() to return false for user message")
	}
}

func TestMessage_IsAssistant(t *testing.T) {
	assistantMsg := NewAssistantMessage("Hello")
	if !assistantMsg.IsAssistant() {
		t.Error("expected IsAssistant() to return true for assistant message")
	}
	if assistantMsg.IsUser() {
		t.Error("expected IsUser() to return false for assistant message")
	}
}

func TestMessage_TimestampIsRecent(t *testing.T) {
	before := time.Now()
	msg := NewUserMessage("Test")
	after := time.Now()

	if msg.Timestamp.Before(before) || msg.Timestamp.After(after) {
		t.Error("expected Timestamp to be set to current time")
	}
}

func TestMessage_MultipleAppends(t *testing.T) {
	msg := NewAssistantMessage("")
	msg.AppendContent("First")
	msg.AppendContent("Second")
	msg.AppendContent("Third")

	expected := "FirstSecondThird"
	if msg.Content != expected {
		t.Errorf("expected Content %q, got %q", expected, msg.Content)
	}
}

func TestMessage_LongContent(t *testing.T) {
	longContent := strings.Repeat("a", 10000)
	msg := NewUserMessage(longContent)

	if msg.Content != longContent {
		t.Errorf("expected Content length %d, got %d", len(longContent), len(msg.Content))
	}
}
