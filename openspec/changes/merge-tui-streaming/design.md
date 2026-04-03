## Context

The AutoCode project has two divergent implementations:

**Master branch**: Basic placeholder TUI with an agent that returns string responses. The TUI is not connected to the agent - it just displays a static message.

**TUI-streaming worktree** (`.worktrees/tui-streaming/`): Complete event-driven implementation with:
- `internal/agent/events.go` - Event type definitions
- `internal/llm/streaming.go` - SSE streaming client
- `internal/tui/model.go` - Full TUI with tree view
- `internal/tui/tree.go` - Tree visualization structure
- `internal/agent/agent.go` - Event-driven agent with cancellation

The goal is to merge the mature streaming implementation into master.

## Goals / Non-Goals

**Goals:**
- Port event-driven agent architecture from worktree
- Port streaming LLM support with SSE parsing
- Port full TUI integration with tree visualization
- Port cancellation support (Escape key interrupt)
- Port visual state indicators and navigation
- Ensure all existing tests pass after changes
- Maintain backward compatibility where possible

**Non-Goals:**
- New features beyond what's in the worktree
- Refactoring worktree code (port as-is, refactor later)
- Changing tool implementations
- Adding new LLM providers
- Persistent history or session management

## Decisions

### 1. Agent Interface: Event-driven over callback
**Decision**: Agent publishes events via channel (`EventChannel()`) rather than accepting callbacks.

**Rationale**: 
- Channels are more idiomatic Go
- Easier integration with Bubble Tea's message-based architecture
- Simpler to test (can read from channel)

**Alternative considered**: Callback functions - rejected as less flexible for TUI updates

### 2. LLM Streaming: Separate StreamChat method
**Decision**: Add `StreamChat()` alongside existing `Chat()` method, not replace it.

**Rationale**:
- Allows non-streaming use cases (testing, batch processing)
- Minimal change to existing code
- Both can share HTTP client configuration

### 3. Cancellation: Channel-based with sync.Once
**Decision**: Use `cancelCh` channel closed via `sync.Once` to signal cancellation.

**Rationale**:
- Simple, works across goroutines
- `sync.Once` prevents double-close panics
- Can be checked via `select` at cancellation points

### 4. TUI Architecture: Separate tui package
**Decision**: Keep TUI in `internal/tui/` package, not inline in main.

**Rationale**:
- Separation of concerns
- Testable components
- Can reuse TUI with different agents later

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Breaking Agent API changes | Document in proposal; this is internal API so acceptable |
| Goroutine leaks in streaming | Ensure stream draining on cancellation; use defer close |
| Event channel blocking | Use select with timeout (100ms) to drop events if channel full |
| Test failures | Update agent tests to use new event-driven interface |
| Worktree code may have bugs | Review all ported code; run tests after each component |

## Migration Plan

1. Port `internal/agent/events.go` - Event type definitions
2. Port `internal/llm/streaming.go` - SSE streaming
3. Update `internal/agent/agent.go` - Event-driven implementation
4. Add `internal/tui/tree.go` - Tree structure
5. Add `internal/tui/model.go` - TUI model
6. Update `cmd/autocode/main.go` - Wire up components
7. Update tests - Adapt to new interfaces
8. Run full test suite

Rollback: Revert to master branch if issues found (all changes are additive or in separate files).

## Open Questions

- Should we keep the old `Chat()` synchronous interface for backward compatibility? → **Decision**: Yes, add `StreamChat()` as new method
- Should the TUI show raw JSON thoughts or formatted? → **Decision**: Port as-is from worktree (formatted)
