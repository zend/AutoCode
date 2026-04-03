## ADDED Requirements

### Requirement: Anthropic API client supports Claude models
The LLM client SHALL support Anthropic's Messages API for chat completions.

#### Scenario: Send chat request to Anthropic
- **WHEN** the application is configured with ANTHROPIC_AUTH_TOKEN
- **THEN** the client SHALL send requests to Anthropic's /v1/messages endpoint

#### Scenario: Authenticate with x-api-key header
- **WHEN** making API requests
- **THEN** the client SHALL use x-api-key header with ANTHROPIC_AUTH_TOKEN

#### Scenario: Use Claude 3 Sonnet as default model
- **WHEN** no specific model is configured
- **THEN** the client SHALL default to claude-3-sonnet-20240229

### Requirement: Anthropic streaming responses
The client SHALL support streaming responses via Anthropic's SSE format.

#### Scenario: Stream returns token channel
- **WHEN** StreamChat() is called with Anthropic provider
- **THEN** it SHALL return a channel that receives tokens as they arrive

#### Scenario: Handle Anthropic SSE format
- **WHEN** the API returns SSE events with event: content_block_delta
- **THEN** the client SHALL parse each event and extract the delta text

#### Scenario: Stream signals completion
- **WHEN** the API sends message_stop event
- **THEN** the client SHALL send a StreamEvent with Done=true and close the channel

### Requirement: Environment variable configuration
The application SHALL support ANTHROPIC_AUTH_TOKEN and ANTHROPIC_BASE_URL environment variables.

#### Scenario: Detect Anthropic from environment
- **WHEN** ANTHROPIC_AUTH_TOKEN is set and OPENAI_API_KEY is not set
- **THEN** the application SHALL use Anthropic as the LLM provider

#### Scenario: Custom base URL
- **WHEN** ANTHROPIC_BASE_URL is set
- **THEN** the client SHALL use that URL instead of the default https://api.anthropic.com

### Requirement: Error handling
The Anthropic client SHALL handle API errors gracefully.

#### Scenario: Invalid API key
- **WHEN** the API returns 401 Unauthorized
- **THEN** the client SHALL return an error indicating authentication failed

#### Scenario: Rate limiting
- **WHEN** the API returns 429 Too Many Requests
- **THEN** the client SHALL return an error with the retry-after header value if present

#### Scenario: Malformed response
- **WHEN** the API returns unexpected JSON structure
- **THEN** the client SHALL return a parse error without crashing
