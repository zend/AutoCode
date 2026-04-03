## Why

The master branch currently has a basic placeholder TUI that doesn't connect to the agent. Meanwhile, the `tui-streaming` worktree contains a fully functional event-driven architecture with real-time streaming, cancellation support, and visual step tracking. Merging these features will make AutoCode actually usable as an interactive coding agent.

## What Changes

- **Event-driven agent architecture**: Agent publishes events (ThinkingEvent, ToolStartEvent, ToolCompleteEvent, StepCompleteEvent) instead of returning strings
- **Streaming LLM support**: Real-time token streaming via SSE with `StreamChat()` method
- **Full TUI integration**: Connected TUI model with tree visualization of agent reasoning steps
- **Cancellation support**: Users can interrupt agent execution with Escape key
- **Visual state indicators**: Real-time status (thinking/executing/complete/error/interrupted) with color-coded indicators
- **Navigation**: Up/Down keys to navigate steps, Space/E to expand/collapse details
- **BREAKING**: Agent `Run()` method signature changes from returning `(string, error)` to `error` with events via `EventChannel()`
- **BREAKING**: Agent now requires `LLMClient` interface instead of concrete `*llm.Client`

## Capabilities

### New Capabilities
- `agent-events`: Event-driven agent architecture for real-time TUI updates
- `llm-streaming`: Server-sent events (SSE) streaming for LLM responses
- `tui-tree-view`: Interactive tree visualization of agent reasoning steps

### Modified Capabilities
- None (no existing specs to modify)

## Impact

- `internal/agent/agent.go`: Complete rewrite to event-driven model
- `internal/llm/`: New `streaming.go` with SSE parsing
- `internal/tui/`: New package with model, tree view, and event handlers
- `cmd/autocode/main.go`: Updated to wire up agent and TUI
- `go.mod`: No new dependencies (already has Bubble Tea)
