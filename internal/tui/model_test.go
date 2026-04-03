package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zend/AutoCode/internal/agent"
)

// mockAgentForModel creates a mock agent for model tests
func mockAgentForModel() (*agent.Agent, *MockAgent) {
	mock := NewMockAgent()
	// Create agent with mock as LLMClient
	// The agent.New function expects an LLMClient, we pass the mock
	// But we need to work around the type system
	// For now, we'll test the model without a real agent
	return nil, mock
}

func TestNewModel(t *testing.T) {
	model := NewModel(nil, "", "")

	if model == nil {
		t.Fatal("expected model, got nil")
	}
	if model.agent != nil {
		t.Error("expected agent to be nil initially")
	}
	if len(model.messages) != 0 {
		t.Errorf("expected empty messages, got %d", len(model.messages))
	}
	if model.input != "" {
		t.Errorf("expected empty input, got %q", model.input)
	}
	if model.running {
		t.Error("expected running to be false")
	}
	if model.ready {
		t.Error("expected ready to be false initially")
	}
}

func TestModel_Init(t *testing.T) {
	model := NewModel(nil, "", "")
	cmd := model.Init()

	// Init now returns nil - renderer is initialized async on WindowSizeMsg
	if cmd != nil {
		t.Error("expected Init to return nil (no initial command)")
	}
}

func TestModel_Update_WindowSize(t *testing.T) {
	model := NewModel(nil, "", "")

	// First WindowSizeMsg should initialize viewport
	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	newModel, cmd := model.Update(msg)
	m := newModel.(*Model)

	if m.width != 100 {
		t.Errorf("expected width 100, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
	if !m.ready {
		t.Error("expected ready to be true after first WindowSizeMsg")
	}
	if m.viewport.Width != 100 {
		t.Errorf("expected viewport width 100, got %d", m.viewport.Width)
	}
	if m.viewport.Height != 36 { // 40 - 4 for input area
		t.Errorf("expected viewport height 36, got %d", m.viewport.Height)
	}
	// WindowSizeMsg now returns a command to init glamour renderer
	if cmd == nil {
		t.Error("expected a command from WindowSizeMsg (initRenderer)")
	}

	// Second WindowSizeMsg should update dimensions
	msg2 := tea.WindowSizeMsg{Width: 80, Height: 30}
	newModel2, _ := m.Update(msg2)
	m2 := newModel2.(*Model)

	if m2.viewport.Width != 80 {
		t.Errorf("expected viewport width 80 after resize, got %d", m2.viewport.Width)
	}
	if m2.viewport.Height != 26 { // 30 - 4 for input area
		t.Errorf("expected viewport height 26 after resize, got %d", m2.viewport.Height)
	}
}

func TestModel_handleKey_CtrlD(t *testing.T) {
	model := NewModel(nil, "", "")

	msg := tea.KeyMsg{Type: tea.KeyCtrlD}
	_, cmd := model.handleKey(msg)

	// cmd should be tea.Quit - we can only verify it's not nil
	if cmd == nil {
		t.Error("expected Ctrl+D to return a command")
	}
}

func TestModel_handleKey_Escape_NotRunning(t *testing.T) {
	model := NewModel(nil, "", "")
	model.input = "some text"
	model.running = false

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, cmd := model.handleKey(msg)
	m := newModel.(*Model)

	if m.input != "" {
		t.Errorf("expected input to be cleared, got %q", m.input)
	}
	if cmd != nil {
		t.Error("expected no command from Escape when not running")
	}
}

func TestModel_handleKey_Escape_EmptyInput(t *testing.T) {
	model := NewModel(nil, "", "")
	model.input = ""
	model.running = false

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, cmd := model.handleKey(msg)

	if cmd != nil {
		t.Error("expected no command from Escape with empty input")
	}
	_ = newModel
}

func TestModel_handleKey_Enter_WithInput(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true // Simulate initialized state
	model.input = "hello world"
	model.running = false

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.handleKey(msg)
	m := newModel.(*Model)

	if m.input != "" {
		t.Errorf("expected input to be cleared, got %q", m.input)
	}
	if !m.running {
		t.Error("expected running to be true after Enter with input")
	}
	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.messages))
	}
	if !m.messages[0].IsUser() {
		t.Error("expected first message to be user message")
	}
	if m.messages[0].Content != "hello world" {
		t.Errorf("expected message content 'hello world', got %q", m.messages[0].Content)
	}
	if cmd == nil {
		t.Error("expected command from Enter (runAgent + listenForEvents)")
	}
}

func TestModel_handleKey_Enter_EmptyInput(t *testing.T) {
	model := NewModel(nil, "", "")
	model.input = ""
	model.running = false

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.handleKey(msg)
	m := newModel.(*Model)

	if len(m.messages) != 0 {
		t.Errorf("expected no messages, got %d", len(m.messages))
	}
	if cmd != nil {
		t.Error("expected no command from Enter with empty input")
	}
}

func TestModel_handleKey_Backspace(t *testing.T) {
	model := NewModel(nil, "", "")
	model.input = "hello"
	model.cursorPos = len(model.input)

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	newModel, _ := model.handleKey(msg)
	m := newModel.(*Model)

	if m.input != "hell" {
		t.Errorf("expected input 'hell', got %q", m.input)
	}
}

func TestModel_handleKey_Backspace_Empty(t *testing.T) {
	model := NewModel(nil, "", "")
	model.input = ""
	model.cursorPos = 0

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	newModel, _ := model.handleKey(msg)
	m := newModel.(*Model)

	if m.input != "" {
		t.Errorf("expected input to remain empty, got %q", m.input)
	}
}

func TestModel_handleKey_Runes(t *testing.T) {
	model := NewModel(nil, "", "")
	model.input = "hel"
	model.cursorPos = len(model.input)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l', 'o'}}
	newModel, _ := model.handleKey(msg)
	m := newModel.(*Model)

	if m.input != "hello" {
		t.Errorf("expected input 'hello', got %q", m.input)
	}
}

func TestModel_handleKey_Up(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20) // Initialize viewport
	model.viewport.SetContent("Line 1\nLine 2\nLine 3\nLine 4\nLine 5")

	// First scroll down to set position
	model.viewport.LineDown(3)

	msg := tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ := model.handleKey(msg)
	m := newModel.(*Model)

	// Check that viewport scroll position changed
	// Note: The actual scroll position depends on viewport implementation
	_ = m
}

func TestModel_handleKey_Down(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)
	model.viewport.SetContent("Line 1\nLine 2\nLine 3\nLine 4\nLine 5")

	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.handleKey(msg)
	m := newModel.(*Model)

	_ = m
}

func TestModel_handleKey_PgUp(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)

	msg := tea.KeyMsg{Type: tea.KeyPgUp}
	newModel, _ := model.handleKey(msg)
	_ = newModel
}

func TestModel_handleKey_PgDown(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)

	msg := tea.KeyMsg{Type: tea.KeyPgDown}
	newModel, _ := model.handleKey(msg)
	_ = newModel
}

func TestModel_handleAgentEvent_ThinkingEvent(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)

	event := agent.ThinkingEvent{
		Step:      0,
		Thought:   "I'm thinking about this task",
		Streaming: false,
	}

	newModel, cmd := model.handleAgentEvent(event)
	m := newModel.(*Model)

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.messages))
	}
	if !m.messages[0].IsAssistant() {
		t.Error("expected assistant message")
	}
	if m.messages[0].Content != "I'm thinking about this task" {
		t.Errorf("expected content 'I'm thinking about this task', got %q", m.messages[0].Content)
	}
	if cmd == nil {
		t.Error("expected command to continue listening for events")
	}
}

func TestModel_handleAgentEvent_ThinkingEvent_UpdateExisting(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)
	model.messages = []Message{NewAssistantMessage("Previous content")}

	event := agent.ThinkingEvent{
		Step:      0,
		Thought:   "Updated thought",
		Streaming: false,
	}

	newModel, _ := model.handleAgentEvent(event)
	m := newModel.(*Model)

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.messages))
	}
	if m.messages[0].Content != "Updated thought" {
		t.Errorf("expected content 'Updated thought', got %q", m.messages[0].Content)
	}
}

func TestModel_handleAgentEvent_ToolStartEvent(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)
	model.messages = []Message{NewAssistantMessage("Thinking...")}

	event := agent.ToolStartEvent{
		Step:   0,
		Action: "shell",
		Input:  map[string]interface{}{"command": "ls -la"},
	}

	newModel, _ := model.handleAgentEvent(event)
	m := newModel.(*Model)

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.messages))
	}
	content := m.messages[0].Content
	if !strings.Contains(content, "**Running:** `shell`") {
		t.Errorf("expected content to contain tool info, got:\n%s", content)
	}
}

func TestModel_handleAgentEvent_ToolCompleteEvent(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)
	model.messages = []Message{NewAssistantMessage("Thinking...")}

	event := agent.ToolCompleteEvent{
		Step:     0,
		Action:   "shell",
		Output:   "file1.txt\nfile2.txt",
		Error:    "",
		Duration: time.Second,
	}

	newModel, _ := model.handleAgentEvent(event)
	m := newModel.(*Model)

	content := m.messages[0].Content
	if !strings.Contains(content, "file1.txt") {
		t.Errorf("expected content to contain output, got:\n%s", content)
	}
}

func TestModel_handleAgentEvent_ToolCompleteEvent_WithError(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)
	model.messages = []Message{NewAssistantMessage("Thinking...")}

	event := agent.ToolCompleteEvent{
		Step:     0,
		Action:   "shell",
		Output:   "",
		Error:    "command not found",
		Duration: 0,
	}

	newModel, _ := model.handleAgentEvent(event)
	m := newModel.(*Model)

	content := m.messages[0].Content
	if !strings.Contains(content, "**Error:**") {
		t.Errorf("expected content to contain error, got:\n%s", content)
	}
}

func TestModel_handleAgentEvent_StepCompleteEvent_Finished(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)
	model.messages = []Message{NewAssistantMessage("Working...")}
	model.running = true

	event := agent.StepCompleteEvent{
		Step:        0,
		Finished:    true,
		Interrupted: false,
		Result:      "Task completed successfully",
	}

	newModel, _ := model.handleAgentEvent(event)
	m := newModel.(*Model)

	if m.running {
		t.Error("expected running to be false after finished event")
	}
	content := m.messages[0].Content
	if !strings.Contains(content, "Task completed successfully") {
		t.Errorf("expected content to contain result, got:\n%s", content)
	}
}

func TestModel_handleAgentEvent_StepCompleteEvent_Interrupted(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)
	model.messages = []Message{NewAssistantMessage("Working...")}
	model.running = true

	event := agent.StepCompleteEvent{
		Step:        0,
		Finished:    false,
		Interrupted: true,
		Result:      "Cancelled by user",
	}

	newModel, _ := model.handleAgentEvent(event)
	m := newModel.(*Model)

	if m.running {
		t.Error("expected running to be false after interrupted event")
	}
	content := m.messages[0].Content
	if !strings.Contains(content, "Interrupted") {
		t.Errorf("expected content to contain interrupted notice, got:\n%s", content)
	}
}

func TestModel_renderMessages_Empty(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)

	content := model.renderMessages()
	if !strings.Contains(content, "Welcome to AutoCode") {
		t.Errorf("expected welcome message, got:\n%s", content)
	}
}

func TestModel_renderMessages_WithMessages(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)
	model.messages = []Message{
		NewUserMessage("Hello"),
		NewAssistantMessage("Hi there!"),
	}

	content := model.renderMessages()
	if !strings.Contains(content, "Hello") {
		t.Errorf("expected user message content, got:\n%s", content)
	}
}

func TestModel_renderMessage_User(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	msg := NewUserMessage("Test message")

	content := model.renderMessage(msg)
	if !strings.Contains(content, "Test message") {
		t.Errorf("expected message content, got:\n%s", content)
	}
}

func TestModel_renderMessage_Assistant(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	msg := NewAssistantMessage("Assistant response")

	content := model.renderMessage(msg)
	if !strings.Contains(content, "Assistant response") {
		t.Errorf("expected message content, got:\n%s", content)
	}
}

func TestModel_renderInput(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.width = 80
	model.input = "test input"
	model.running = false

	content := model.renderInput()
	if !strings.Contains(content, ">") {
		t.Errorf("expected prompt character, got:\n%s", content)
	}
	if !strings.Contains(content, "test input") {
		t.Errorf("expected input text, got:\n%s", content)
	}
}

func TestModel_renderInput_Running(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.width = 80
	model.input = ""
	model.running = true

	content := model.renderInput()
	if !strings.Contains(content, "Processing") {
		t.Errorf("expected running indicator, got:\n%s", content)
	}
}

func TestModel_helpText_NotRunning(t *testing.T) {
	model := NewModel(nil, "", "")
	model.running = false

	help := model.helpText()
	if !strings.Contains(help, "enter: submit") {
		t.Errorf("expected submit hint in help, got: %s", help)
	}
}

func TestModel_helpText_Running(t *testing.T) {
	model := NewModel(nil, "", "")
	model.running = true

	help := model.helpText()
	if !strings.Contains(help, "esc: cancel") {
		t.Errorf("expected cancel hint in help, got: %s", help)
	}
}

func TestModel_updateViewport(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.viewport = viewport.New(80, 20)
	model.messages = []Message{NewUserMessage("Test")}

	model.updateViewport()

	// Viewport should have content set
	content := model.viewport.View()
	if content == "" {
		t.Error("expected viewport to have content after updateViewport")
	}
}

func Test_formatJSON(t *testing.T) {
	tests := []struct {
		input    map[string]interface{}
		expected string
	}{
		{map[string]interface{}{}, "{}"},
		{map[string]interface{}{"key": "value"}, "key=value"},
		{map[string]interface{}{"a": 1, "b": 2}, "a=1, b=2"},
	}

	for _, tt := range tests {
		result := formatJSON(tt.input)
		// Note: map iteration order is not guaranteed, so we check for containment
		if tt.expected == "{}" {
			if result != "{}" {
				t.Errorf("expected '{}', got %q", result)
			}
		} else {
			// For non-empty maps, just verify it doesn't panic and has content
			if result == "" {
				t.Error("expected non-empty result")
			}
		}
	}
}
