## Why

The current TUI uses a tree-view layout which can be overwhelming and doesn't match modern CLI conventions. Refactoring to a Claude Code-style interface will provide a cleaner, more intuitive chat-like experience that users are familiar with from modern AI coding assistants.

## What Changes

- **BREAKING**: Replace tree-view layout with chat-style message history
- Add message bubbles for user and assistant with distinct styling
- Move input to bottom with a persistent "> " prompt (Claude Code style)
- Add syntax highlighting for code blocks in responses
- Implement scrollable message history with viewport component
- Add message timestamps and metadata display
- Support markdown rendering for assistant responses
- Add copy-to-clipboard hints for code blocks
- Update keyboard shortcuts to match Claude Code conventions (Enter to submit, Ctrl+C to quit)
- Remove tree navigation (Up/Down arrows now scroll history)

## Capabilities

### New Capabilities
- `tui-chat-interface`: Chat-style message display with bubbles and history
- `tui-syntax-highlighting`: Code block syntax highlighting in responses
- `tui-markdown-rendering`: Markdown parsing and rendering for assistant messages

### Modified Capabilities
- `tui-tree-view`: Remove tree navigation, replace with chat history (REMOVED functionality)

## Impact

- `internal/tui/model.go`: Complete rewrite of View() and Update() logic
- `internal/tui/message.go`: New file for message types and rendering
- `go.mod`: Add `charmbracelet/glamour` for markdown rendering
- `go.mod`: Add `charmbracelet/lipgloss` style definitions (already present)
- User experience: Significantly different interaction model - chat instead of tree navigation
