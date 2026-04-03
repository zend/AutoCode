## ADDED Requirements

### Requirement: LLM client supports streaming responses
The LLM client SHALL support streaming responses via Server-Sent Events (SSE).

#### Scenario: Stream chat returns token channel
- **WHEN** StreamChat() is called with Stream=true
- **THEN** it SHALL return a channel that receives tokens as they arrive from the API

#### Scenario: Stream handles SSE format
- **WHEN** the API returns SSE data lines (data: {...})
- **THEN** the client SHALL parse each line and extract the delta content

#### Scenario: Stream signals completion
- **WHEN** the API sends data: [DONE]
- **THEN** the client SHALL send a StreamEvent with Done=true and close the channel

### Requirement: Stream handles errors gracefully
The streaming implementation SHALL handle network and parse errors without crashing.

#### Scenario: API returns error status
- **WHEN** the HTTP response status is not 200 OK
- **THEN** StreamChat SHALL return an error with the status and response body

#### Scenario: Malformed JSON in stream
- **WHEN** a stream line contains malformed JSON
- **THEN** the client SHALL send a StreamEvent with the Error field set

#### Scenario: Network interruption
- **WHEN** the connection is interrupted during streaming
- **THEN** the client SHALL send a StreamEvent with the error and close the channel
