package agent

import "time"

// AgentEvent is the interface for all events sent from Agent to TUI
type AgentEvent interface {
	Type() string
}

// ThinkingEvent is sent while agent is reasoning
type ThinkingEvent struct {
	Step      int
	Thought   string
	Streaming bool // true while receiving tokens
}

func (e ThinkingEvent) Type() string { return "thinking" }

// ToolStartEvent is sent when agent starts executing a tool
type ToolStartEvent struct {
	Step   int
	Action string
	Input  map[string]interface{}
}

func (e ToolStartEvent) Type() string { return "tool_start" }

// ToolCompleteEvent is sent when tool execution finishes
type ToolCompleteEvent struct {
	Step     int
	Action   string
	Output   string
	Error    string
	Duration time.Duration
}

func (e ToolCompleteEvent) Type() string { return "tool_complete" }

// StepCompleteEvent is sent when a step finishes (including interruption)
type StepCompleteEvent struct {
	Step        int
	Finished    bool
	Interrupted bool
	Result      string
}

func (e StepCompleteEvent) Type() string { return "step_complete" }

// TaskRequest is sent from TUI to Agent to start a task
type TaskRequest struct {
	Task string
}

// CancelRequest is sent from TUI to Agent to cancel current operation
type CancelRequest struct{}
