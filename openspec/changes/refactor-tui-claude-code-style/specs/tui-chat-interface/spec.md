## ADDED Requirements

### Requirement: TUI displays messages in chat format
The TUI SHALL display conversation history as a scrollable chat interface with user and assistant messages.

#### Scenario: User message appears in chat
- **WHEN** the user submits a task
- **THEN** the message SHALL appear in the chat history with user styling

#### Scenario: Assistant message appears in chat
- **WHEN** the agent responds
- **THEN** the message SHALL appear in the chat history with assistant styling

#### Scenario: Messages are scrollable
- **WHEN** the conversation exceeds the screen height
- **THEN** the user SHALL be able to scroll through message history

### Requirement: Input area at bottom with Claude Code style
The TUI SHALL have a persistent input area at the bottom with a "> " prompt.

#### Scenario: Input always visible
- **WHEN** viewing the TUI at any time
- **THEN** the input prompt SHALL be visible at the bottom

#### Scenario: Input uses "> " prompt
- **WHEN** the TUI renders the input area
- **THEN** it SHALL show "> " before the user's typed text

### Requirement: Distinct styling for user and assistant
User and assistant messages SHALL have visually distinct styling.

#### Scenario: User message styling
- **WHEN** viewing a user message
- **THEN** it SHALL be styled differently from assistant messages (e.g., right-aligned or different color)

#### Scenario: Assistant message styling
- **WHEN** viewing an assistant message
- **THEN** it SHALL be styled with the assistant's branding color

### Requirement: Message timestamps
Messages SHALL display timestamps showing when they were sent.

#### Scenario: User message shows timestamp
- **WHEN** viewing a user message
- **THEN** it SHALL display the time the message was sent

#### Scenario: Assistant message shows timestamp
- **WHEN** viewing an assistant message
- **THEN** it SHALL display the time the response started
