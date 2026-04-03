package tui

import (
	"time"
)

// MessageRole represents the role of a message (user or assistant)
type MessageRole int

const (
	UserMessage      MessageRole = 0
	AssistantMessage MessageRole = 1
)

// Message represents a single message in the chat history
type Message struct {
	Role      MessageRole
	Content   string
	Timestamp time.Time
}

// NewUserMessage creates a new user message
func NewUserMessage(content string) Message {
	return Message{
		Role:      UserMessage,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// NewAssistantMessage creates a new assistant message
func NewAssistantMessage(content string) Message {
	return Message{
		Role:      AssistantMessage,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// AppendContent appends content to the message (used for streaming)
func (m *Message) AppendContent(content string) {
	m.Content += content
}

// FormatTimestamp returns a formatted timestamp string
func (m Message) FormatTimestamp() string {
	return m.Timestamp.Format("15:04:05")
}

// IsUser returns true if this is a user message
func (m Message) IsUser() bool {
	return m.Role == UserMessage
}

// IsAssistant returns true if this is an assistant message
func (m Message) IsAssistant() bool {
	return m.Role == AssistantMessage
}
