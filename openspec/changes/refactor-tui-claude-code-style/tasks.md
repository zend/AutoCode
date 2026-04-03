## 1. Setup Dependencies

- [x] 1.1 Add `charmbracelet/glamour` to go.mod for markdown rendering
- [x] 1.2 Add `charmbracelet/bubbles` to go.mod for viewport component
- [x] 1.3 Run `go mod tidy` to update dependencies

## 2. Create Message Types

- [x] 2.1 Create `internal/tui/message.go` with Message struct
- [x] 2.2 Define Message struct with Role, Content, Timestamp fields
- [x] 2.3 Define MessageType constants (UserMessage, AssistantMessage)
- [x] 2.4 Add message formatting methods

## 3. Refactor Model Structure

- [x] 3.1 Replace Tree with messages slice in Model struct
- [x] 3.2 Add viewport field for scrollable message area
- [x] 3.3 Add glamour renderer field for markdown
- [x] 3.4 Update NewModel constructor

## 4. Implement Chat View

- [x] 4.1 Rewrite View() for chat layout with input at bottom
- [x] 4.2 Implement renderMessages() for message history
- [x] 4.3 Add user message bubble styling
- [x] 4.4 Add assistant message bubble styling
- [x] 4.5 Implement timestamp display

## 5. Add Viewport Scrolling

- [x] 5.1 Integrate viewport.BubbleTea component
- [x] 5.2 Handle viewport sizing on WindowSizeMsg
- [x] 5.3 Bind Up/Down arrows to scroll history
- [x] 5.4 Auto-scroll to bottom on new messages

## 6. Add Markdown Rendering

- [x] 6.1 Initialize glamour renderer with dark theme
- [x] 6.2 Render assistant messages through glamour
- [x] 6.3 Handle rendering errors gracefully
- [x] 6.4 Support streaming content updates

## 7. Add Syntax Highlighting

- [x] 7.1 Configure glamour with syntax highlighting
- [x] 7.2 Style code blocks with distinct background
- [x] 7.3 Add code block borders

## 8. Update Input Handling

- [x] 8.1 Move input to bottom with "> " prompt
- [x] 8.2 Update handleKey() for new keyboard shortcuts
- [x] 8.3 Remove tree navigation key bindings
- [x] 8.4 Add submit on Enter

## 9. Update Event Handling

- [x] 9.1 Convert agent events to chat messages
- [x] 9.2 Create user message on task submit
- [x] 9.3 Create assistant message on thinking events
- [x] 9.4 Append tool results to assistant message
- [x] 9.5 Handle streaming content updates

## 10. Styling

- [x] 10.1 Define user message styles
- [x] 10.2 Define assistant message styles
- [x] 10.3 Define input prompt styles
- [x] 10.4 Remove tree-specific styles

## 11. Testing

- [x] 11.1 Run all unit tests: `go test ./...`
- [x] 11.2 Build binary: `go build -o bin/autocode ./cmd/autocode`
- [x] 11.3 Test chat interface manually
- [x] 11.4 Test scrolling with long conversations
- [x] 11.5 Test markdown rendering
- [x] 11.6 Test syntax highlighting
