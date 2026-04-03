## Context

The AutoCode application currently has a single LLM client implementation in `internal/llm/client.go` that assumes an OpenAI-compatible API. The streaming implementation in `internal/llm/streaming.go` also uses OpenAI's SSE format. To support Anthropic's Claude models, we need to add a separate client that handles Anthropic's native Messages API.

Anthropic uses:
- Different authentication (x-api-key header instead of Bearer token)
- Different API endpoint (/v1/messages instead of /v1/chat/completions)
- Different request/response format (messages array with role/content instead of chat completions)
- Different streaming format (event types in SSE)

## Goals / Non-Goals

**Goals:**
- Add Anthropic API client with Messages API support
- Support streaming responses via Anthropic's SSE format
- Allow users to switch between OpenAI and Anthropic via environment variables
- Maintain backward compatibility with existing OpenAI setup

**Non-Goals:**
- Supporting other LLM providers (Gemini, Cohere, etc.)
- Changing the agent's interface (Agent uses LLMClient interface, stays unchanged)
- Implementing Anthropic's legacy text completion API (only Messages API)
- Fine-grained model selection UI (handled via environment variables)

## Decisions

### 1. Provider Selection: Environment variable-based
**Decision**: Use presence of `ANTHROPIC_AUTH_TOKEN` to detect Anthropic vs OpenAI.

**Rationale**:
- Simple, no configuration files needed
- Clear separation - one or the other
- Easy to switch between providers

**Alternative considered**: Configuration file with provider field - rejected as overkill for a CLI tool.

### 2. Client Architecture: Separate structs with shared interface
**Decision**: Create separate `AnthropicClient` struct alongside existing `Client`, both satisfying `LLMClient` interface.

**Rationale**:
- Clean separation of provider-specific logic
- No conditional spaghetti code in HTTP handling
- Easy to test each client independently

### 3. Streaming Format: Native Anthropic SSE handling
**Decision**: Implement Anthropic's event-based SSE format in a separate streaming method.

**Rationale**:
- Anthropic uses `event: content_block_delta` format vs OpenAI's `data:` prefix
- Different event types need different parsing
- Keeping them separate prevents fragile abstraction

### 4. Model Selection: Default to Claude 3 Sonnet
**Decision**: Use `claude-3-sonnet-20240229` as default Anthropic model.

**Rationale**:
- Good balance of capability and cost
- Supports streaming and tool use (for future)

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| API drift between OpenAI and Anthropic | Keep clients separate, monitor API changelogs |
| Testing requires real API keys | Add mock server support for tests |
| Environment variable confusion | Clear error messages when neither provider is configured |

## Migration Plan

No migration needed - this is additive. Existing OpenAI setups continue working unchanged.

New users can:
1. Set `ANTHROPIC_AUTH_TOKEN` and optionally `ANTHROPIC_BASE_URL`
2. Run the application - it auto-detects Anthropic

## Open Questions

- Should we support both providers simultaneously? → **Decision**: No, one provider per run
- Should the TUI show which provider is active? → **Decision**: Yes, show in title bar
