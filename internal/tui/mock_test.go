package tui

import (
	"context"
	"sync"

	"github.com/zend/AutoCode/internal/agent"
	"github.com/zend/AutoCode/internal/llm"
)

// MockAgent implements agent.LLMClient for testing
type MockAgent struct {
	cancelCalled bool
	eventCh      chan agent.AgentEvent
	mutex        sync.Mutex
}

// NewMockAgent creates a new mock agent for testing
func NewMockAgent() *MockAgent {
	return &MockAgent{
		eventCh: make(chan agent.AgentEvent, 100),
	}
}

// StreamChat implements the LLMClient interface
func (m *MockAgent) StreamChat(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	ch := make(chan llm.StreamEvent)
	close(ch)
	return ch, nil
}

// Chat implements the LLMClient interface
func (m *MockAgent) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{
		ID:    "mock-response",
		Model: "mock-model",
		Choices: []struct {
			Index        int         `json:"index"`
			Message      llm.Message `json:"message"`
			FinishReason string      `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: llm.Message{
					Role:    "assistant",
					Content: "Mock response",
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

// Cancel marks the agent as cancelled
func (m *MockAgent) Cancel() {
	m.mutex.Lock()
	m.cancelCalled = true
	m.mutex.Unlock()
}

// WasCancelCalled returns true if Cancel was called
func (m *MockAgent) WasCancelCalled() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.cancelCalled
}

// ResetCancel resets the cancel flag
func (m *MockAgent) ResetCancel() {
	m.mutex.Lock()
	m.cancelCalled = false
	m.mutex.Unlock()
}

// EventChannel returns the event channel
func (m *MockAgent) EventChannel() <-chan agent.AgentEvent {
	return m.eventCh
}

// SendEvent sends an event to the event channel
func (m *MockAgent) SendEvent(event agent.AgentEvent) {
	m.eventCh <- event
}

// CloseEventChannel closes the event channel
func (m *MockAgent) CloseEventChannel() {
	close(m.eventCh)
}
