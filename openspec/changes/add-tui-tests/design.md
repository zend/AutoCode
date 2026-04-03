## Context

The TUI package has been refactored to a chat-style interface but lacks test coverage. Testing Bubble Tea applications requires understanding the Model-Update-View pattern and how to simulate user input and window events.

## Goals / Non-Goals

**Goals:**
- Achieve >80% test coverage for TUI package
- Test message creation and manipulation
- Test keyboard input handling (Enter, Esc, arrows, etc.)
- Test window resize handling
- Test agent event handling
- Test viewport scrolling behavior
- Create mock agent for isolated testing

**Non-Goals:**
- UI integration tests (visual testing)
- Testing actual rendering output
- Testing glamour markdown rendering (external library)
- Performance benchmarks

## Decisions

### 1. Testing Approach: Unit tests with mocked dependencies
**Decision**: Create mock agent that satisfies LLMClient interface for isolated TUI testing.

**Rationale**:
- TUI tests should not depend on real agent or LLM
- Mock allows deterministic testing of event handling
- Follows Go testing best practices

### 2. Test Organization: Separate test files by component
**Decision**: Create model_test.go for Model tests, message_test.go for Message tests.

**Rationale**:
- Clear separation of concerns
- Easier to find relevant tests
- Matches Go convention

### 3. Event Simulation: Use tea.Msg directly
**Decision**: Simulate user input by creating tea.KeyMsg and tea.WindowSizeMsg directly.

**Rationale**:
- Standard Bubble Tea testing approach
- No need for external testing libraries
- Direct control over test inputs

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Viewport behavior hard to test | Test viewport state indirectly through scroll position |
| Glamour rendering not testable | Skip testing glamour output, focus on input/output |
| Race conditions in async events | Use synchronous test patterns |

## Migration Plan

1. Create mock agent implementation
2. Create message_test.go for Message type tests
3. Create model_test.go for Model tests
4. Add tests for each keyboard shortcut
5. Add tests for agent event handling
6. Run tests and ensure >80% coverage
7. Fix any bugs discovered during testing
