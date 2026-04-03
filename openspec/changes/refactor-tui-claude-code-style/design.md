## Context

The current TUI (`internal/tui/model.go`) uses a tree-view layout that displays agent execution steps in a hierarchical tree structure. While this provides visibility into the agent's reasoning process, it doesn't match the conversational interaction pattern users expect from modern AI coding assistants like Claude Code.

Claude Code's interface features:
- Chat-style message history with distinct user/assistant bubbles
- Input at the bottom with a minimal "> " prompt
- Syntax-highlighted code blocks
- Scrollable conversation history
- Clean, distraction-free design

## Goals / Non-Goals

**Goals:**
- Replace tree-view with chat-style message history
- Implement message bubbles with distinct user/assistant styling
- Add syntax highlighting for code blocks
- Support markdown rendering in assistant responses
- Move input to bottom with Claude Code-style "> " prompt
- Make conversation history scrollable
- Maintain all existing functionality (streaming, cancellation, events)

**Non-Goals:**
- Supporting multiple concurrent conversations
- Persistent conversation history between sessions
- Customizable themes or colors
- Mouse support (keyboard-only interaction)
- Inline editing of previous messages

## Decisions

### 1. Layout: Input at bottom, messages above
**Decision**: Fixed input area at bottom (~3 lines), scrollable message history above.

**Rationale**:
- Matches Claude Code and chat conventions
- Messages naturally flow bottom-up
- Input always visible and accessible

**Alternative considered**: Variable input like a text editor - rejected as unfamiliar for chat interface.

### 2. Message Rendering: Glamour for markdown
**Decision**: Use `charmbracelet/glamour` for markdown rendering with syntax highlighting.

**Rationale**:
- Already handles markdown + syntax highlighting
- Supports multiple color themes
- Widely used in Charm ecosystem

**Alternative considered**: Custom markdown parser - rejected as too much work for same result.

### 3. Message Storage: In-memory slice
**Decision**: Store messages as slice of structs with role, content, timestamp.

**Rationale**:
- Simple, matches Bubble Tea patterns
- Easy to render with lipgloss
- No persistence needed (Non-Goal)

### 4. Scrolling: Bubble Tea viewport
**Decision**: Use `charmbracelet/bubbles/viewport` for scrollable message area.

**Rationale**:
- Standard Bubble Tea component
- Handles paging, scrolling, dimensions
- Integrates well with lipgloss

### 5. Streaming: Append to last message
**Decision**: Stream tokens append to the last assistant message in real-time.

**Rationale**:
- Natural chat behavior
- Simpler than creating new messages mid-stream
- Matches Claude Code behavior

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Glamour adds significant dependency | Acceptable for rich markdown rendering |
| Loss of tree visibility for debugging | Acceptable - focus on user experience |
| Terminal width constraints | Viewport handles wrapping, Glamour adapts |
| Performance with large outputs | Viewport only renders visible content |

## Migration Plan

1. Replace tree.go with message.go (new message types)
2. Rewrite model.go View() for chat layout
3. Update event handling to create messages instead of tree nodes
4. Add viewport integration for scrolling
5. Integrate Glamour for markdown rendering
6. Update styles for chat bubbles
7. Test all existing functionality

## Open Questions

- Should we keep the tree view as an optional debug view? → **Decision**: No, simplify to chat only
- What markdown theme to use? → **Decision**: Dark theme matching terminal
