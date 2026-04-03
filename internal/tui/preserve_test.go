package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestMessagesPreservedAcrossRuns tests that messages are preserved across multiple runs
func TestMessagesPreservedAcrossRuns(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true

	// Simulate first interaction
	model.input = "first message"
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	m1, _ := model.handleKey(enterMsg)
	model = m1.(*Model)

	if len(model.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(model.messages))
	}

	// Add assistant response
	model.messages = append(model.messages, NewAssistantMessage("Response 1"))

	// Simulate second interaction
	model.input = "second message"
	model.running = false // Reset for next run
	m2, _ := model.handleKey(enterMsg)
	model = m2.(*Model)

	if len(model.messages) != 3 {
		t.Fatalf("expected 3 messages, got %d: %+v", len(model.messages), model.messages)
	}

	// Check all messages are preserved
	if model.messages[0].Content != "first message" {
		t.Errorf("message 0: expected 'first message', got %q", model.messages[0].Content)
	}
	if model.messages[1].Content != "Response 1" {
		t.Errorf("message 1: expected 'Response 1', got %q", model.messages[1].Content)
	}
	if model.messages[2].Content != "second message" {
		t.Errorf("message 2: expected 'second message', got %q", model.messages[2].Content)
	}

	// Verify renderMessages includes all messages
	rendered := model.renderMessages()
	if rendered == "" {
		t.Error("renderMessages returned empty string")
	}
	t.Logf("Rendered length: %d", len(rendered))
	t.Logf("Rendered content:\n%s", rendered)
}

// TestViewportContentUpdated tests that viewport content is updated correctly
func TestViewportContentUpdated(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	// Initialize viewport
	model.viewport.Width = 80
	model.viewport.Height = 20

	// Add messages
	model.messages = append(model.messages, NewUserMessage("Message 1"))
	rendered1 := model.renderMessages()
	t.Logf("renderMessages length 1: %d", len(rendered1))

	model.messages = append(model.messages, NewAssistantMessage("Response 1"))
	rendered2 := model.renderMessages()
	t.Logf("renderMessages length 2: %d", len(rendered2))

	// Second user message
	model.messages = append(model.messages, NewUserMessage("Message 2"))
	rendered3 := model.renderMessages()
	t.Logf("renderMessages length 3: %d", len(rendered3))

	// Check that renderMessages grows
	if len(rendered3) <= len(rendered1) {
		t.Errorf("renderMessages should grow: %d -> %d", len(rendered1), len(rendered3))
	}
}
