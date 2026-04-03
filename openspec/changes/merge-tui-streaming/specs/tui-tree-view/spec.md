## ADDED Requirements

### Requirement: TUI displays agent execution as tree
The TUI SHALL display agent execution steps as an interactive tree view.

#### Scenario: Steps appear as tree nodes
- **WHEN** the agent publishes events
- **THEN** each step SHALL appear as a node in the tree with state indicator

#### Scenario: Tree shows step states
- **WHEN** viewing the tree
- **THEN** each node SHALL show its current state: thinking (○), executing (◐), complete (●), error (✗), or interrupted (⏹)

### Requirement: TUI supports navigation
The TUI SHALL support keyboard navigation of the tree.

#### Scenario: Navigate with arrow keys
- **WHEN** the user presses Up or Down arrow keys
- **THEN** the cursor SHALL move to the previous or next step

#### Scenario: Expand and collapse nodes
- **WHEN** the user presses Space or E on a node
- **THEN** the node SHALL toggle between expanded and collapsed state

### Requirement: TUI shows step details
The TUI SHALL show detailed information for expanded steps.

#### Scenario: Expanded step shows thought
- **WHEN** a thinking step is expanded
- **THEN** the TUI SHALL display the agent's thought content

#### Scenario: Expanded step shows tool I/O
- **WHEN** a tool execution step is expanded
- **THEN** the TUI SHALL display the tool input and output

### Requirement: TUI supports task input and cancellation
The TUI SHALL allow users to enter tasks and cancel running tasks.

#### Scenario: Enter task
- **WHEN** no task is running and user presses Enter
- **THEN** the TUI SHALL start the agent with the entered task

#### Scenario: Cancel task
- **WHEN** a task is running and user presses Escape
- **THEN** the TUI SHALL call agent.Cancel() and show interruption state

### Requirement: TUI handles streaming thoughts
The TUI SHALL update the display in real-time as thoughts stream in.

#### Scenario: Live thought updates
- **WHEN** the agent streams thinking tokens
- **THEN** the TUI SHALL update the thought display without waiting for the complete response
