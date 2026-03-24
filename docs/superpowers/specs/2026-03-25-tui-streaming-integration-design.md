# TUI Streaming Integration Design

## Overview

Connect the TUI to the ReAct Agent with real-time streaming LLM responses displayed in an expandable tree view. Support interrupting the agent mid-execution.

## Goals

- Display agent execution as an expandable tree (thought → action → result)
- Stream LLM tokens live within tree nodes during thinking
- Allow user to interrupt agent with Esc key
- Maintain responsive UI during long operations

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      TUI Layer                          │
│  ┌─────────────┐  ┌─────────────┐  ┌────────────────┐  │
│  │  Tree View  │  │ Input Box   │  │ Status Bar     │  │
│  │  (bubbletea)│  │             │  │                │  │
│  └──────┬──────┘  └─────────────┘  └────────────────┘  │
└─────────┼───────────────────────────────────────────────┘
          │
┌─────────┼───────────────────────────────────────────────┐
│         │              Agent Layer                      │
│  ┌────────────────────────────────────────────┐        │
│  │           Event Bus (chan)                 │        │
│  │  ┌─────────┐ ┌─────────┐ ┌───────────────┐ │        │
│  │  │Thinking │ │ToolExec │ │StepComplete   │ │        │
│  │  │Token    │ │Start    │ │Interrupt      │ │        │
│  │  └─────────┘ └─────────┘ └───────────────┘ │        │
│  └────────────────────────────────────────────┘        │
│         │                                               │
│         ▼                                               │
│  ┌─────────────────────────────────────────┐           │
│  │         Agent Controller                │           │
│  │  - Manages execution loop               │           │
│  │  - Handles cancellation (context)       │           │
│  └─────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────┘
          │
┌─────────┼───────────────────────────────────────────────┐
│         │              LLM Layer                        │
│         ▼                                               │
│  ┌─────────────────────────────────────────┐           │
│  │     Streaming Client (SSE)              │           │
│  │  - HTTP client with SSE parser          │           │
│  │  - Yields tokens via channel            │           │
│  └─────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────┘
```

## Event Types

Events flow from Agent to TUI via a channel:

```go
type AgentEvent interface {
    Type() string
}

type ThinkingEvent struct {
    Step      int
    Thought   string // accumulated thought so far
    Streaming bool   // true while receiving tokens
}

type ToolStartEvent struct {
    Step    int
    Action  string
    Input   map[string]interface{}
}

type ToolCompleteEvent struct {
    Step      int
    Action    string
    Output    string
    Error     string
    Duration  time.Duration
}

type StepCompleteEvent struct {
    Step      int
    Finished  bool
    Result    string // only if Finished=true
}
```

## Data Structures

### Tree Node

```go
type TreeNode struct {
    Step       int
    Thought    string
    Action     string
    Input      string
    Output     string
    Error      string
    Duration   time.Duration
    Expanded   bool
    State      NodeState
}

type NodeState int
const (
    StateThinking  NodeState = 0
    StateExecuting NodeState = 1
    StateComplete  NodeState = 2
    StateError     NodeState = 3
)
```

## Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| TUI Model | Holds tree state, current input, event channel. Implements `tea.Model`. |
| Tree View | Renders expandable nodes. Shows: thought preview → (expand) → full thought + action + input → output/error. |
| Agent Controller | Manages ReAct loop. Listens for cancellation. Publishes events. |
| Streaming Client | HTTP SSE parser. Yields tokens via channel. Supports cancellation via context. |
| Tool Registry | (existing) Executes tools synchronously, returns results. |

## Key Interactions

1. **User submits task** → TUI sends to Agent Controller via channel → Agent starts loop
2. **Agent thinking** → Stream tokens via `ThinkingEvent` → TUI updates node in real-time
3. **Agent chooses action** → `ToolStartEvent` → TUI marks node as "executing"
4. **Tool completes** → `ToolCompleteEvent` → TUI updates with output/error
5. **Step done** → `StepCompleteEvent` → TUI adds new node for next step (or shows result if finished)
6. **User presses Esc** → TUI sends cancel signal → Agent Controller cancels LLM context

## Streaming Implementation

### SSE Client

```go
func (c *Client) StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
    // Send request with stream: true
    // Parse SSE data: lines starting with "data: "
    // Yield tokens via channel
    // Support cancellation via context
}

type StreamEvent struct {
    Token string
    Done  bool
    Error error
}
```

### Token Throttling

Accumulate tokens in a string builder. Send `ThinkingEvent` every 50ms to avoid flooding TUI. Final event marks `Streaming: false`.

## Error Handling

| Scenario | Handling |
|----------|----------|
| LLM API error | Agent publishes `ToolCompleteEvent{Error: "..."}`, TUI shows error node |
| Invalid JSON from LLM | Agent retries up to 3 times, includes parse error as observation |
| Tool timeout | Context timeout per tool, returns error to agent |
| Path validation | Existing `validatePath` rejects paths outside baseDir |
| Streaming interrupted | Context cancellation propagates, TUI shows "(interrupted)" indicator |
| Max steps exceeded | Agent publishes event, TUI shows "max steps reached" |

## Cancellation Flow

```
User presses Esc
    ↓
TUI sets cancel flag
    ↓
Agent Controller checks context.Done()
    ↓
Cancels LLM request (HTTP client with context)
    ↓
Publishes StepCompleteEvent{Interrupted: true}
    ↓
TUI shows current node with "⚠ interrupted" badge
    ↓
Returns to input mode for new task
```

## Testing Strategy

| Component | Test Approach |
|-----------|---------------|
| Streaming Client | Mock SSE server, verify tokens and cancellation |
| Agent Controller | Mock LLM client, verify event sequence |
| TUI Tree View | Snapshot tests for rendering, verify expansion logic |
| Integration | End-to-end with mock LLM, verify full flow |

## UI Layout

```
┌────────────────────────────────────────┐
│ 🦞 AutoCode                    [steps] │
├────────────────────────────────────────┤
│ ▼ Step 1: Read project files           │
│   Thought: I need to understand the    │
│   project structure. Let me start by   │
│   reading the main file...             │
│   ┌────────────────────────────────┐   │
│   │ Action: read                   │   │
│   │ Input: {"path": "main.go"}     │   │
│   │ Output: (success - 45 lines)   │   │
│   └────────────────────────────────┘   │
│                                        │
│ ▶ Step 2: Analyzing code...            │
│                                        │
│                                        │
├────────────────────────────────────────┤
│ > Type task and press Enter            │
│ Press Esc to interrupt • q to quit     │
└────────────────────────────────────────┘
```

## Implementation Order

1. Add streaming support to LLM client (SSE parser)
2. Create event types and event bus
3. Refactor Agent to publish events
4. Update TUI to receive events and render tree
5. Add interrupt support (context cancellation)
6. Add expansion/collapse to tree nodes
7. Polish: throttling, error states, help text

## Success Criteria

- [ ] TUI displays agent execution as expandable tree
- [ ] LLM tokens stream live within thinking nodes
- [ ] User can interrupt agent with Esc key
- [ ] UI remains responsive during long operations
- [ ] Errors are displayed clearly in tree
- [ ] Tree supports expand/collapse per node
