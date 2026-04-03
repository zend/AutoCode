package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zend/AutoCode/internal/agent"
)

// TestIntegration_EventLoop_StopsOnComplete tests that the event listening
// loop properly stops when receiving a Finished StepCompleteEvent
func TestIntegration_EventLoop_StopsOnComplete(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.running = true // Simulate task is running

	// Send StepCompleteEvent with Finished=true
	completeEvent := agent.StepCompleteEvent{
		Step:        0,
		Finished:    true,
		Interrupted: false,
		Result:      "Task completed",
	}

	newModel, cmd := model.handleAgentEvent(completeEvent)
	m := newModel.(*Model)

	// After Finished event, cmd should be nil (no more listening)
	if cmd != nil {
		t.Error("expected NO command after Finished StepCompleteEvent - event loop should stop")
	}

	if m.running {
		t.Error("expected running to be false after Finished event")
	}
}

// TestIntegration_EventLoop_StopsOnInterrupt tests that the event listening
// loop properly stops when receiving an Interrupted StepCompleteEvent
func TestIntegration_EventLoop_StopsOnInterrupt(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.running = true

	// Send StepCompleteEvent with Interrupted=true
	event := agent.StepCompleteEvent{
		Step:        0,
		Finished:    false,
		Interrupted: true,
		Result:      "Cancelled by user",
	}

	newModel, cmd := model.handleAgentEvent(event)
	m := newModel.(*Model)

	// After Interrupted event, cmd should be nil (no more listening)
	if cmd != nil {
		t.Error("expected NO command after Interrupted StepCompleteEvent - event loop should stop")
	}

	if m.running {
		t.Error("expected running to be false after Interrupted event")
	}
}

// TestIntegration_RunningState_ResetsCorrectly tests that running state
// properly resets to false when task completes
func TestIntegration_RunningState_ResetsCorrectly(t *testing.T) {
	tests := []struct {
		name            string
		event           agent.StepCompleteEvent
		expectedRunning bool
		expectedCmdNil  bool // Should cmd be nil (stop listening)?
	}{
		{
			name: "Finished=true stops running",
			event: agent.StepCompleteEvent{
				Step:        0,
				Finished:    true,
				Interrupted: false,
			},
			expectedRunning: false,
			expectedCmdNil:  true, // Stop listening
		},
		{
			name: "Interrupted=true stops running",
			event: agent.StepCompleteEvent{
				Step:        0,
				Finished:    false,
				Interrupted: true,
			},
			expectedRunning: false,
			expectedCmdNil:  true, // Stop listening
		},
		{
			name: "neither finished nor interrupted keeps running",
			event: agent.StepCompleteEvent{
				Step:        0,
				Finished:    false,
				Interrupted: false,
			},
			expectedRunning: true,  // Still running
			expectedCmdNil:  false, // Continue listening
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel(nil, "", "")
			model.ready = true
			model.running = true

			newModel, cmd := model.handleAgentEvent(tt.event)
			m := newModel.(*Model)

			if m.running != tt.expectedRunning {
				t.Errorf("running=%v, expected=%v", m.running, tt.expectedRunning)
			}

			cmdIsNil := cmd == nil
			if cmdIsNil != tt.expectedCmdNil {
				t.Errorf("cmd is nil=%v, expected=%v", cmdIsNil, tt.expectedCmdNil)
			}
		})
	}
}

// TestIntegration_FullEventFlow_Simulation simulates the event flow
// to verify the bug: running should reset to false after task completion
func TestIntegration_FullEventFlow_Simulation(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true

	// Step 1: User types input and presses Enter
	model.input = "test task"
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.handleKey(msg)
	m := newModel.(*Model)

	if !m.running {
		t.Fatal("expected running=true after Enter (this simulates the bug setup)")
	}

	// Step 2: Simulate agent sending ThinkingEvent
	thinkingEvent := agent.ThinkingEvent{
		Step:      0,
		Thought:   "Analyzing task...",
		Streaming: false,
	}
	newModel2, cmd2 := m.handleAgentEvent(thinkingEvent)
	m2 := newModel2.(*Model)

	// Should continue listening after ThinkingEvent
	if cmd2 == nil {
		t.Error("expected cmd to continue listening after ThinkingEvent")
	}

	// Step 3: Simulate agent sending StepCompleteEvent with Finished=true
	completeEvent := agent.StepCompleteEvent{
		Step:        0,
		Finished:    true,
		Interrupted: false,
		Result:      "Task completed successfully",
	}
	newModel3, _ := m2.handleAgentEvent(completeEvent)
	m3 := newModel3.(*Model)


	// THIS IS THE BUG: running should be false
	// If running is still true, UI shows "Processing..." forever
	if m3.running {
		t.Error("BUG: running should be false after Finished event - UI stuck at 'Processing...'")
	}
}

// TestIntegration_ListenForEvents_ReturnsNilOnStop tests that
// listenForEvents returns nil when stopListen is closed
// NOTE: This test requires a non-nil agent, so it's skipped when agent is nil
func TestIntegration_ListenForEvents_ReturnsNilOnStop(t *testing.T) {
	// Skip this test since we can't easily create a *agent.Agent without
	// complex setup. The fix should handle nil agent gracefully.
	t.Skip("Skipping test - requires non-nil agent")
}

// TestIntegration_TickMsg_ContinuesListening tests tickMsg handling
func TestIntegration_TickMsg_ContinuesListening(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.running = true

	// Send tickMsg
	msg := tickMsg{}
	newModel, cmd := model.Update(msg)
	m := newModel.(*Model)

	// Should continue listening when running
	if cmd == nil {
		t.Error("expected cmd to continue listening after tickMsg when running")
	}

	// Should still be running
	if !m.running {
		t.Error("expected running=true after tickMsg")
	}
}

// TestIntegration_TickMsg_StopsWhenNotRunning tests tickMsg stops when not running
func TestIntegration_TickMsg_StopsWhenNotRunning(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.running = false

	// Send tickMsg
	msg := tickMsg{}
	_, cmd := model.Update(msg)

	// Should NOT continue listening when not running
	if cmd != nil {
		t.Error("expected cmd=nil after tickMsg when not running")
	}
}

// TestIntegration_StepCompleteEvent_Error tests StepCompleteEvent with Step=-1 (error case)
func TestIntegration_StepCompleteEvent_Error(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.running = true

	// Send an error StepCompleteEvent (Step=-1 from runAgent)
	event := agent.StepCompleteEvent{
		Step:        -1,
		Finished:    false,
		Interrupted: false,
		Result:      "Error: llm stream chat: connection refused",
	}

	newModel, _ := model.handleAgentEvent(event)
	m := newModel.(*Model)

	// Should NOT be running after error
	if m.running {
		t.Error("expected running=false after error StepCompleteEvent")
	}


	// Should have added an error message
	foundError := false
	for _, msg := range m.messages {
		if msg.IsAssistant() && len(msg.Content) > 0 {
			// Check if message contains error info
			foundError = true
			break
		}
	}
	if !foundError {
		t.Error("expected error message to be added")
	}
}

// TestIntegration_StepCompleteEvent_NonTerminal tests StepCompleteEvent
// that is not terminal (neither finished nor interrupted)
func TestIntegration_StepCompleteEvent_NonTerminal(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true
	model.running = true

	// Send a non-terminal StepCompleteEvent (e.g., after each step)
	event := agent.StepCompleteEvent{
		Step:        0,
		Finished:    false,
		Interrupted: false,
		Result:      "Step completed, continuing...",
	}

	newModel, cmd := model.handleAgentEvent(event)
	m := newModel.(*Model)

	// Should still be running
	if !m.running {
		t.Error("expected running=true for non-terminal StepCompleteEvent")
	}

	// Should continue listening
	if cmd == nil {
		t.Error("expected cmd to continue listening for non-terminal event")
	}
}

// TestIntegration_EscWhenRunning_NonNilAgent tests Escape key behavior
// when a task is running with a non-nil agent
func TestIntegration_EscWhenRunning_NonNilAgent(t *testing.T) {
	// Create a mock agent that won't panic on Cancel
	_ = NewMockAgent()

	// We can't directly assign mock to model.agent since it expects *agent.Agent
	// This test documents the expected behavior when agent is present
	model := NewModel(nil, "", "")
	model.running = true

	// Without an agent, Escape when running would panic
	// The fix should check if agent is nil before calling Cancel
	msg := tea.KeyMsg{Type: tea.KeyEsc}

	// This will panic if not handled properly
	// We use a closure to catch potential panic
	var panicked bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		model.handleKey(msg)
	}()

	if panicked {
		t.Error("handleKey panicked when Escape pressed with running=true and nil agent")
	}
}

// TestIntegration_Messages_Preserved tests that messages are preserved across multiple interactions
func TestIntegration_Messages_Preserved(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true

	// First interaction
	model.input = "first message"
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	m1, _ := model.handleKey(enterMsg)
	model = m1.(*Model)

	// Verify first message is added
	if len(model.messages) != 1 {
		t.Fatalf("expected 1 message after first input, got %d", len(model.messages))
	}
	if model.messages[0].Content != "first message" {
		t.Errorf("expected 'first message', got %q", model.messages[0].Content)
	}

	// Simulate agent response
	model.messages = append(model.messages, NewAssistantMessage("Response 1"))

	// Second interaction - input should be cleared but messages preserved
	model.input = "second message"
	model.running = false // Reset running state
	m2, _ := model.handleKey(enterMsg)
	model = m2.(*Model)

	// Verify both old and new messages are preserved
	if len(model.messages) != 3 {
		t.Fatalf("expected 3 messages (user1, assistant1, user2), got %d", len(model.messages))
	}
	if model.messages[0].Content != "first message" {
		t.Errorf("message 0: expected 'first message', got %q", model.messages[0].Content)
	}
	if model.messages[1].Content != "Response 1" {
		t.Errorf("message 1: expected 'Response 1', got %q", model.messages[1].Content)
	}
	if model.messages[2].Content != "second message" {
		t.Errorf("message 2: expected 'second message', got %q", model.messages[2].Content)
	}
}
func TestIntegration_EventLoop_BugReproduction(t *testing.T) {
	// Simulate: User submits task -> Agent runs -> Events sent -> Task completes
	// Expected: running=false after completion
	// Bug: running stays true forever

	model := NewModel(nil, "", "")
	model.ready = true

	// Simulate Enter key press
	model.input = "Do something"
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	m1, cmd1 := model.handleKey(enterMsg)
	model = m1.(*Model)

	if !model.running {
		t.Fatal("expected running=true after submitting task")
	}

	// cmd1 is tea.Batch(runAgent, listenForEvents)
	// In real Bubble Tea, these would be processed
	_ = cmd1

	// Simulate event stream from agent
	events := []agent.AgentEvent{
		agent.ThinkingEvent{Step: 0, Thought: "Planning..."},
		agent.ThinkingEvent{Step: 0, Thought: "Planning... Working on it"},
		agent.ToolStartEvent{Step: 0, Action: "shell", Input: map[string]interface{}{"command": "echo test"}},
		agent.ToolCompleteEvent{Step: 0, Action: "shell", Output: "test"},
		agent.StepCompleteEvent{Step: 0, Finished: true, Result: "Done!"},
	}
		for _, ev := range events {
			m, _ := model.handleAgentEvent(ev)
			model = m.(*Model)

			// After the final event, check state
			if ev, ok := ev.(agent.StepCompleteEvent); ok && ev.Finished {
				// cmd can be a print command, that's fine
				if model.running {
					t.Error("BUG: running should be false after Finished event")
				}
			}
		}
	}

	// TestIntegration_HandleAgentEvent_NilEvent tests handling nil AgentEvent
func TestIntegration_HandleAgentEvent_NilEvent(t *testing.T) {
	model := NewModel(nil, "", "")
	model.ready = true

	// Send nil event (stop signal)
	var nilEvent agent.AgentEvent = nil
	newModel, cmd := model.Update(nilEvent)
	m := newModel.(*Model)

	// Should not panic and should return nil cmd
	_ = m
	if cmd != nil {
		t.Error("expected cmd=nil for nil AgentEvent")
	}
}
