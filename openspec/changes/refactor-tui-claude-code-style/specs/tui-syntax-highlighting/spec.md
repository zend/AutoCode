## ADDED Requirements

### Requirement: Code blocks have syntax highlighting
Code blocks in assistant responses SHALL have syntax highlighting based on the language.

#### Scenario: Go code highlighted
- **WHEN** the assistant responds with a Go code block (```go)
- **THEN** the code SHALL be displayed with Go syntax highlighting

#### Scenario: Python code highlighted
- **WHEN** the assistant responds with a Python code block (```python)
- **THEN** the code SHALL be displayed with Python syntax highlighting

#### Scenario: JavaScript code highlighted
- **WHEN** the assistant responds with a JavaScript code block (```javascript)
- **THEN** the code SHALL be displayed with JavaScript syntax highlighting

#### Scenario: Plain text blocks
- **WHEN** the assistant responds with a code block without language (```)
- **THEN** the code SHALL be displayed as plain text without highlighting

### Requirement: Code blocks are visually distinct
Code blocks SHALL be visually separated from surrounding text.

#### Scenario: Code block background
- **WHEN** viewing a code block
- **THEN** it SHALL have a distinct background color from regular text

#### Scenario: Code block border
- **WHEN** viewing a code block
- **THEN** it SHALL have a subtle border or padding to separate it
