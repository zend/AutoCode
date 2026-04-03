## 1. Create Mock Agent

- [x] 1.1 Create mock LLMClient implementation for testing
- [x] 1.2 Implement StreamChat method on mock
- [x] 1.3 Implement Chat method on mock
- [x] 1.4 Add cancel tracking to mock agent

## 2. Create Message Tests

- [x] 2.1 Create `internal/tui/message_test.go`
- [x] 2.2 Test NewUserMessage creates correct message
- [x] 2.3 Test NewAssistantMessage creates correct message
- [x] 2.4 Test AppendContent appends correctly
- [x] 2.5 Test FormatTimestamp returns correct format
- [x] 2.6 Test IsUser returns correct value
- [x] 2.7 Test IsAssistant returns correct value

## 3. Create Model Tests - Initialization

- [x] 3.1 Create `internal/tui/model_test.go`
- [x] 3.2 Test NewModel creates model with correct defaults
- [x] 3.3 Test Init initializes renderer
- [x] 3.4 Test Init returns listenForEvents command

## 4. Create Model Tests - Keyboard Input

- [x] 4.1 Test Enter key creates user message when input not empty
- [x] 4.2 Test Enter key does nothing when input empty
- [x] 4.3 Test Escape key cancels when running
- [x] 4.4 Test Escape key clears input when not running
- [x] 4.5 Test Backspace removes last character
- [x] 4.6 Test Backspace on empty input does nothing
- [x] 4.7 Test Ctrl+C returns quit command
- [x] 4.8 Test regular keys append to input

## 5. Create Model Tests - Window Events

- [x] 5.1 Test WindowSizeMsg initializes viewport on first call
- [x] 5.2 Test WindowSizeMsg updates viewport on subsequent calls
- [x] 5.3 Test viewport dimensions are set correctly

## 6. Create Model Tests - Agent Events

- [x] 6.1 Test ThinkingEvent creates assistant message
- [x] 6.2 Test ThinkingEvent updates existing assistant message
- [x] 6.3 Test ToolStartEvent appends tool info
- [x] 6.4 Test ToolCompleteEvent appends output
- [x] 6.5 Test ToolCompleteEvent with error appends error
- [x] 6.6 Test StepCompleteEvent with Finished=true stops running
- [x] 6.7 Test StepCompleteEvent with Interrupted=true stops running

## 7. Create Model Tests - Viewport Scrolling

- [x] 7.1 Test Up arrow scrolls viewport up
- [x] 7.2 Test Down arrow scrolls viewport down
- [x] 7.3 Test PgUp scrolls half page up
- [x] 7.4 Test PgDown scrolls half page down

## 8. Create Model Tests - Message Rendering

- [x] 8.1 Test renderMessages with no messages shows welcome
- [x] 8.2 Test renderMessage for user message
- [x] 8.3 Test renderMessage for assistant message
- [x] 8.4 Test renderInput shows prompt and cursor
- [x] 8.5 Test renderInput shows running state

## 9. Run Tests and Verify

- [x] 9.1 Run all TUI tests: `go test ./internal/tui/... -v`
- [x] 9.2 Check test coverage: `go test ./internal/tui/... -cover`
- [x] 9.3 Fix any failing tests
- [x] 9.4 Run all tests: `go test ./...`
- [x] 9.5 Build binary: `go build -o bin/autocode ./cmd/autocode`
