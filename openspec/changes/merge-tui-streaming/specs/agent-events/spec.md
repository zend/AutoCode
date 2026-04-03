## ADDED Requirements

### Requirement: Agent publishes events during execution
The agent SHALL publish structured events during task execution to enable real-time UI updates.

#### Scenario: Agent publishes thinking event
- **WHEN** the agent starts reasoning about a step
- **THEN** it SHALL publish a ThinkingEvent with the current step number and thought content

#### Scenario: Agent publishes tool start event
- **WHEN** the agent decides to execute a tool
- **THEN** it SHALL publish a ToolStartEvent with the step number, action name, and input parameters

#### Scenario: Agent publishes tool complete event
- **WHEN** a tool finishes execution
- **THEN** it SHALL publish a ToolCompleteEvent with the step number, output, error (if any), and duration

#### Scenario: Agent publishes step complete event
- **WHEN** a step completes (success, failure, or interruption)
- **THEN** it SHALL publish a StepCompleteEvent with the final status

### Requirement: Agent supports cancellation
The agent SHALL support cancellation of in-progress tasks via a Cancel method.

#### Scenario: User cancels during thinking
- **WHEN** the user calls Cancel() while the agent is streaming thoughts
- **THEN** the agent SHALL stop processing and publish a StepCompleteEvent with Interrupted=true

#### Scenario: User cancels during tool execution
- **WHEN** the user calls Cancel() while a tool is executing
- **THEN** the agent SHALL allow current tool to complete but stop before next step

### Requirement: Event channel never blocks indefinitely
The agent SHALL drop events rather than block if the event channel is full.

#### Scenario: Slow consumer
- **WHEN** the event consumer is slower than event production
- **THEN** events SHALL be dropped after 100ms timeout rather than blocking the agent
