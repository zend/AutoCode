## Why

The TUI package currently has no test coverage, making it difficult to ensure correctness when making changes. Adding comprehensive tests will verify all TUI functionality works as expected and prevent regressions during future development.

## What Changes

- Create `internal/tui/model_test.go` with comprehensive test coverage
- Test message creation and formatting
- Test viewport scrolling behavior
- Test keyboard input handling
- Test agent event handling and message conversion
- Test markdown rendering integration
- Add mock agent for isolated TUI testing
- Verify all keyboard shortcuts work correctly

## Capabilities

### New Capabilities
- `tui-testing`: Comprehensive test suite for TUI components

### Modified Capabilities
- None

## Impact

- `internal/tui/model_test.go`: New test file with full coverage
- `internal/tui/message_test.go`: New test file for message types
- Test-only dependencies: No production code changes
