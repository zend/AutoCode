## 1. Event Types

- [x] 1.1 Create `internal/agent/events.go` with AgentEvent interface
- [x] 1.2 Define ThinkingEvent with Step, Thought, Streaming fields
- [x] 1.3 Define ToolStartEvent with Step, Action, Input fields
- [x] 1.4 Define ToolCompleteEvent with Step, Action, Output, Error, Duration fields
- [x] 1.5 Define StepCompleteEvent with Step, Finished, Interrupted, Result fields
- [x] 1.6 Run tests to verify events compile

## 2. LLM Streaming

- [x] 2.1 Create `internal/llm/streaming.go` with StreamEvent struct
- [x] 2.2 Implement StreamChat method with SSE parsing
- [x] 2.3 Handle data: [DONE] signal
- [x] 2.4 Handle API error responses
- [x] 2.5 Handle malformed JSON gracefully
- [x] 2.6 Run streaming tests

## 3. Agent Refactor

- [x] 3.1 Add LLMClient interface to agent.go
- [x] 3.2 Add eventCh field to Agent struct
- [x] 3.3 Add cancelCh and cancelOnce fields
- [x] 3.4 Implement EventChannel() method
- [x] 3.5 Implement Cancel() method with sync.Once
- [x] 3.6 Implement publishEvent() with timeout
- [x] 3.7 Refactor Run() to use streaming
- [x] 3.8 Implement streamThinking() method
- [x] 3.9 Implement executeToolWithEvents() method
- [x] 3.10 Update agent tests for new interface

## 4. Tree Structure

- [x] 4.1 Create `internal/tui/tree.go` with TreeNode struct
- [x] 4.2 Define node states: StateThinking, StateExecuting, StateComplete, StateError, StateInterrupted
- [x] 4.3 Implement Tree type with AddNode, GetNode, Nodes methods
- [x] 4.4 Implement ToggleExpand for tree nodes
- [x] 4.5 Implement Title(), Preview(), FullContent() methods
- [x] 4.6 Run tree tests

## 5. TUI Model

- [x] 5.1 Create `internal/tui/model.go` with Model struct
- [x] 5.2 Implement NewModel constructor
- [x] 5.3 Implement Init() with listenForEvents
- [x] 5.4 Implement Update() with WindowSizeMsg handling
- [x] 5.5 Implement handleKey() with all key bindings
- [x] 5.6 Implement handleAgentEvent() for all event types
- [x] 5.7 Implement runAgent() tea.Cmd
- [x] 5.8 Implement listenForEvents() tea.Cmd
- [x] 5.9 Implement View() with title, tree, input, help
- [x] 5.10 Implement renderTree() with indentation
- [x] 5.11 Define all lipgloss styles
- [x] 5.12 Run model tests

## 6. Main Integration

- [x] 6.1 Update `cmd/autocode/main.go` imports
- [x] 6.2 Initialize agent with LLM client
- [x] 6.3 Create TUI model with agent
- [x] 6.4 Update tea.Program initialization
- [x] 6.5 Test full application startup

## 7. Test Verification

- [x] 7.1 Run all unit tests: `go test ./...`
- [x] 7.2 Fix any failing tests
- [x] 7.3 Build binary: `go build -o bin/autocode ./cmd/autocode`
- [x] 7.4 Verify TUI launches without error
- [x] 7.5 Check agent can receive and cancel tasks
