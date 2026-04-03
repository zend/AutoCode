## 1. Anthropic Client Core

- [x] 1.1 Create `internal/llm/anthropic.go` with AnthropicClient struct
- [x] 1.2 Define Anthropic request/response types (Message, MessageRequest, MessageResponse)
- [x] 1.3 Implement NewAnthropicClient constructor with baseURL and authToken
- [x] 1.4 Implement Chat method for non-streaming requests
- [x] 1.5 Add authentication with x-api-key header
- [x] 1.6 Handle Anthropic-specific error responses

## 2. Anthropic Streaming

- [x] 2.1 Define AnthropicStreamEvent type for SSE parsing
- [x] 2.2 Implement StreamChat method with SSE parsing
- [x] 2.3 Handle event: content_block_delta events
- [x] 2.4 Handle message_stop completion signal
- [x] 2.5 Handle Anthropic-specific error events (event: error)
- [x] 2.6 Run streaming tests

## 3. Provider Selection

- [x] 3.1 Update `cmd/autocode/main.go` to check environment variables
- [x] 3.2 Implement provider detection (ANTHROPIC_AUTH_TOKEN vs OPENAI_API_KEY)
- [x] 3.3 Create LLMClient interface instance based on provider
- [x] 3.4 Add clear error message when neither provider is configured
- [x] 3.5 Support ANTHROPIC_BASE_URL override

## 4. Testing

- [x] 4.1 Create `internal/llm/anthropic_test.go` with mock server
- [x] 4.2 Test Chat method with mock Anthropic API
- [x] 4.3 Test StreamChat with mock SSE stream
- [x] 4.4 Test error handling (401, 429, malformed JSON)
- [x] 4.5 Run all unit tests: `go test ./...`
- [x] 4.6 Build binary: `go build -o bin/autocode ./cmd/autocode`
