## ADDED Requirements

### Requirement: TUI has comprehensive test coverage
The TUI package SHALL have unit tests covering all major functionality.

#### Scenario: Message creation tested
- **WHEN** creating user and assistant messages
- **THEN** the tests SHALL verify correct initialization

#### Scenario: Message content appended
- **WHEN** appending content to a message
- **THEN** the tests SHALL verify the content is added

#### Scenario: Message timestamps formatted
- **WHEN** formatting message timestamps
- **THEN** the tests SHALL verify correct time format

### Requirement: Keyboard input handling tested
The TUI SHALL have tests for all keyboard shortcuts.

#### Scenario: Enter key submits input
- **WHEN** user presses Enter with text in input
- **THEN** the tests SHALL verify a user message is created

#### Scenario: Escape cancels when running
- **WHEN** user presses Escape while agent is running
- **THEN** the tests SHALL verify Cancel() is called on the agent

#### Scenario: Escape clears input when not running
- **WHEN** user presses Escape with text in input and agent not running
- **THEN** the tests SHALL verify input is cleared

#### Scenario: Backspace removes characters
- **WHEN** user presses Backspace
- **THEN** the tests SHALL verify last character is removed from input

#### Scenario: Up arrow scrolls up
- **WHEN** user presses Up arrow
- **THEN** the tests SHALL verify viewport scrolls up

#### Scenario: Down arrow scrolls down
- **WHEN** user presses Down arrow
- **THEN** the tests SHALL verify viewport scrolls down

#### Scenario: Ctrl+C quits
- **WHEN** user presses Ctrl+C
- **THEN** the tests SHALL verify tea.Quit command is returned

### Requirement: Agent event handling tested
The TUI SHALL have tests for agent event handling.

#### Scenario: Thinking event creates assistant message
- **WHEN** a ThinkingEvent is received
- **THEN** the tests SHALL verify an assistant message is created

#### Scenario: Tool start event appends tool info
- **WHEN** a ToolStartEvent is received
- **THEN** the tests SHALL verify tool information is appended to message

#### Scenario: Tool complete event appends output
- **WHEN** a ToolCompleteEvent is received
- **THEN** the tests SHALL verify tool output is appended to message

#### Scenario: Step complete finishes task
- **WHEN** a StepCompleteEvent with Finished=true is received
- **THEN** the tests SHALL verify running state is set to false

#### Scenario: Interrupted event handled
- **WHEN** a StepCompleteEvent with Interrupted=true is received
- **THEN** the tests SHALL verify running state is set to false and interruption noted

### Requirement: Window resize tested
The TUI SHALL handle window resize events.

#### Scenario: Window size initializes viewport
- **WHEN** first WindowSizeMsg is received
- **THEN** the tests SHALL verify viewport is initialized with correct dimensions

#### Scenario: Window resize updates viewport
- **WHEN** subsequent WindowSizeMsg is received
- **THEN** the tests SHALL verify viewport dimensions are updated

### Requirement: Mock agent for testing
The TUI tests SHALL use a mock agent implementation.

#### Scenario: Mock agent implements LLMClient
- **WHEN** creating a mock agent
- **THEN** it SHALL satisfy the agent.LLMClient interface

#### Scenario: Mock agent tracks Cancel calls
- **WHEN** Cancel() is called on mock agent
- **THEN** the tests SHALL verify the cancel was recorded
