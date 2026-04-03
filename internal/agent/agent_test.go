package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/zend/AutoCode/internal/llm"
)

// mockLLMClient implements LLMClient for testing
type mockLLMClient struct {
	streamEvents []llm.StreamEvent
	chatResponse *llm.ChatResponse
	chatError    error
	streamError  error
}

func (m *mockLLMClient) StreamChat(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	if m.streamError != nil {
		return nil, m.streamError
	}

	ch := make(chan llm.StreamEvent, len(m.streamEvents)+1)
	for _, e := range m.streamEvents {
		ch <- e
	}
	close(ch)
	return ch, nil
}

func (m *mockLLMClient) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return m.chatResponse, m.chatError
}

func TestNew(t *testing.T) {
	client := &mockLLMClient{}
	agent := New(client, "/tmp")

	if agent == nil {
		t.Fatal("expected agent, got nil")
	}
	if agent.baseDir != "/tmp" {
		t.Errorf("expected baseDir /tmp, got %s", agent.baseDir)
	}
	if agent.maxSteps != 50 {
		t.Errorf("expected maxSteps 50, got %d", agent.maxSteps)
	}
	if agent.client != client {
		t.Error("expected client to be set")
	}
	if agent.eventCh == nil {
		t.Error("expected eventCh to be initialized")
	}
}

func TestSetMaxSteps(t *testing.T) {
	client := &mockLLMClient{}
	agent := New(client, "/tmp")
	agent.SetMaxSteps(100)
	if agent.maxSteps != 100 {
		t.Errorf("expected maxSteps 100, got %d", agent.maxSteps)
	}
}

func TestEventChannel(t *testing.T) {
	client := &mockLLMClient{}
	agent := New(client, "/tmp")

	ch := agent.EventChannel()
	if ch == nil {
		t.Error("expected event channel, got nil")
	}
}

func TestCancel(t *testing.T) {
	client := &mockLLMClient{}
	agent := New(client, "/tmp")
	agent.cancelCh = make(chan struct{})

	// Cancel should close the channel
	agent.Cancel()

	select {
	case <-agent.cancelCh:
		// Good, channel is closed
	case <-time.After(100 * time.Millisecond):
		t.Error("cancelCh was not closed after Cancel()")
	}

	// Calling Cancel again should not panic (sync.Once protects it)
	agent.Cancel()
}

func TestParseResponse(t *testing.T) {
	client := &mockLLMClient{}
	agent := New(client, "/tmp")

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*AgentResponse) bool
	}{
		{
			name:  "valid action response",
			input: `{"thought": "I need to read a file", "action": "read", "action_input": {"path": "test.txt"}}`,
			check: func(r *AgentResponse) bool {
				return r.Thought == "I need to read a file" && r.Action == "read"
			},
		},
		{
			name:  "valid finish response",
			input: `{"thought": "task completed", "finish": true, "result": "Done"}`,
			check: func(r *AgentResponse) bool {
				return r.Finish && r.Result == "Done"
			},
		},
		{
			name:  "response with markdown",
			input: "Here's my response:\n\n```json\n{\"thought\": \"test\", \"action\": \"shell\", \"action_input\": {\"command\": \"ls\"}}\n```",
			check: func(r *AgentResponse) bool {
				return r.Thought == "test" && r.Action == "shell"
			},
		},
		{
			name:    "no JSON",
			input:   "Just a plain text response",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `{"thought": "test", "action": }`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := agent.parseResponse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !tt.check(resp) {
				t.Errorf("response check failed: %+v", resp)
			}
		})
	}
}

func TestRun_CompletesSuccessfully(t *testing.T) {
	// Create mock client that returns a finish response
	client := &mockLLMClient{
		streamEvents: []llm.StreamEvent{
			{Token: `{"thought": "task completed", "finish": true, "result": "Task done"}`},
			{Done: true},
		},
	}

	agent := New(client, t.TempDir())
	agent.SetMaxSteps(10)

	// Collect events in background
	events := make([]AgentEvent, 0)
	eventCh := agent.EventChannel()

	err := agent.Run(context.Background(), "Test task")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Collect all pending events (don't wait for close)
	for {
		select {
		case event := <-eventCh:
			if event == nil {
				goto done
			}
			events = append(events, event)
		case <-time.After(100 * time.Millisecond):
			goto done
		}
	}
done:

	// Should have at least one step complete event
	foundComplete := false
	for _, e := range events {
		if sce, ok := e.(StepCompleteEvent); ok && sce.Finished {
			foundComplete = true
			if sce.Result != "Task done" {
				t.Errorf("expected result 'Task done', got: %s", sce.Result)
			}
		}
	}
	if !foundComplete {
		t.Error("expected a StepCompleteEvent with Finished=true")
	}
}

func TestRun_MaxStepsExceeded(t *testing.T) {
	// Create mock client that never finishes
	client := &mockLLMClient{
		streamEvents: []llm.StreamEvent{
			{Token: `{"thought": "still thinking", "action": "shell", "action_input": {"command": "echo test"}}`},
			{Done: true},
		},
	}

	agent := New(client, t.TempDir())
	agent.SetMaxSteps(3)

	err := agent.Run(context.Background(), "Test task")
	if err == nil {
		t.Fatal("expected error for max steps exceeded")
	}
	if err.Error() != "max steps (3) exceeded" {
		t.Errorf("expected max steps error, got: %v", err)
	}
}

func TestRun_LLMError(t *testing.T) {
	client := &mockLLMClient{
		streamError: context.DeadlineExceeded,
	}

	agent := New(client, t.TempDir())

	err := agent.Run(context.Background(), "Test task")
	if err == nil {
		t.Fatal("expected error from LLM")
	}
}

func TestRun_Cancellation(t *testing.T) {
	// Create a blocking mock client
	client := &blockingMockClient{}

	agent := New(client, t.TempDir())
	agent.SetMaxSteps(10)

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		agent.Cancel()
	}()

	err := agent.Run(context.Background(), "Test task")
	// Cancellation returns "interrupted" error from streamThinking
	if err == nil || err.Error() != "interrupted" {
		t.Errorf("expected 'interrupted' error on cancellation, got: %v", err)
	}
}

// blockingMockClient blocks until context is cancelled
type blockingMockClient struct{}

func (m *blockingMockClient) StreamChat(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	ch := make(chan llm.StreamEvent)
	go func() {
		defer close(ch)
		// Block until context is cancelled or timeout
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
		}
	}()
	return ch, nil
}

func (m *blockingMockClient) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return nil, nil
}

func TestGetHistory(t *testing.T) {
	client := &mockLLMClient{
		streamEvents: []llm.StreamEvent{
			{Token: `{"thought": "done", "finish": true, "result": "result"}`},
			{Done: true},
		},
	}

	agent := New(client, t.TempDir())

	// History should be empty before Run
	history := agent.GetHistory()
	if len(history) != 0 {
		t.Errorf("expected empty history, got %d messages", len(history))
	}

	agent.Run(context.Background(), "Test task")

	// History should have messages after Run
	history = agent.GetHistory()
	if len(history) == 0 {
		t.Error("expected history after Run, got none")
	}
}

func TestExecuteToolWithEvents(t *testing.T) {
	client := &mockLLMClient{}
	agent := New(client, t.TempDir())

	tests := []struct {
		name    string
		resp    *AgentResponse
		wantErr bool
	}{
		{
			name:    "no action",
			resp:    &AgentResponse{Thought: "test"},
			wantErr: true,
		},
		{
			name:    "unknown tool",
			resp:    &AgentResponse{Thought: "test", Action: "unknown"},
			wantErr: true,
		},
		{
			name: "valid tool",
			resp: &AgentResponse{
				Thought:     "test",
				Action:      "shell",
				ActionInput: map[string]interface{}{"command": "echo hello"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := agent.executeToolWithEvents(ctx, 0, tt.resp)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAgentResponse_JSON(t *testing.T) {
	// Test that AgentResponse can be marshaled and unmarshaled
	original := AgentResponse{
		Thought:     "test thought",
		Action:      "read",
		ActionInput: map[string]interface{}{"path": "test.txt"},
		Finish:      false,
		Result:      "",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded AgentResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Thought != original.Thought {
		t.Errorf("thought mismatch: %s vs %s", decoded.Thought, original.Thought)
	}
	if decoded.Action != original.Action {
		t.Errorf("action mismatch: %s vs %s", decoded.Action, original.Action)
	}
}

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name     string
		event    AgentEvent
		expected string
	}{
		{
			name:     "thinking event",
			event:    ThinkingEvent{Step: 1, Thought: "test"},
			expected: "thinking",
		},
		{
			name:     "tool start event",
			event:    ToolStartEvent{Step: 1, Action: "read"},
			expected: "tool_start",
		},
		{
			name:     "tool complete event",
			event:    ToolCompleteEvent{Step: 1, Action: "read", Output: "result"},
			expected: "tool_complete",
		},
		{
			name:     "step complete event",
			event:    StepCompleteEvent{Step: 1, Finished: true, Result: "done"},
			expected: "step_complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.Type(); got != tt.expected {
				t.Errorf("Type() = %v, want %v", got, tt.expected)
			}
		})
	}
}
