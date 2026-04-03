## Why

The application currently only supports OpenAI-compatible APIs. Adding Anthropic API support allows users to leverage Claude models for agent tasks, providing an alternative LLM provider with different capabilities and pricing.

## What Changes

- Add Anthropic API client implementation with native message API support
- Support `ANTHROPIC_AUTH_TOKEN` and `ANTHROPIC_BASE_URL` environment variables
- Implement streaming response handling for Anthropic's SSE format
- Add provider detection in main.go to choose between OpenAI and Anthropic based on environment
- **BREAKING**: Environment variable naming changes from `OPENAI_API_KEY` to `ANTHROPIC_AUTH_TOKEN` for Anthropic users

## Capabilities

### New Capabilities
- `anthropic-api`: Anthropic API client with Claude model support and streaming

### Modified Capabilities
- `llm-streaming`: Extend to support Anthropic's SSE streaming format alongside OpenAI

## Impact

- `internal/llm/`: New `anthropic.go` with Anthropic-specific client
- `cmd/autocode/main.go`: Provider selection logic based on environment variables
- `go.mod`: No new dependencies (uses existing HTTP client)
- Environment variables: Adds `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`
