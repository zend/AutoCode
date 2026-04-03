package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAnthropicClient(t *testing.T) {
	client := NewAnthropicClient("https://api.anthropic.com", "test-token")

	if client == nil {
		t.Fatal("expected client, got nil")
	}
	if client.baseURL != "https://api.anthropic.com" {
		t.Errorf("expected baseURL https://api.anthropic.com, got %s", client.baseURL)
	}
	if client.authToken != "test-token" {
		t.Errorf("expected authToken test-token, got %s", client.authToken)
	}
	if client.model != "claude-3-sonnet-20240229" {
		t.Errorf("expected default model claude-3-sonnet-20240229, got %s", client.model)
	}
}

func TestNewAnthropicClient_DefaultBaseURL(t *testing.T) {
	client := NewAnthropicClient("", "test-token")

	if client.baseURL != "https://api.anthropic.com/v1" {
		t.Errorf("expected default baseURL https://api.anthropic.com/v1, got %s", client.baseURL)
	}
}

func TestAnthropicClient_SetModel(t *testing.T) {
	client := NewAnthropicClient("", "test-token")
	client.SetModel("claude-3-opus-20240229")

	if client.model != "claude-3-opus-20240229" {
		t.Errorf("expected model claude-3-opus-20240229, got %s", client.model)
	}
}

func TestAnthropicClient_Chat(t *testing.T) {
	// Create a mock server that simulates Anthropic API responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("expected /v1/messages, got %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("x-api-key") != "test-token" {
			t.Errorf("expected x-api-key header, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version header, got %s", r.Header.Get("anthropic-version"))
		}

		// Parse request
		var req AnthropicMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			return
		}

		// Return mock response
		response := AnthropicMessageResponse{
			ID:   "msg_0123456789",
			Type: "message",
			Role: "assistant",
			Content: []AnthropicContentBlock{
				{Type: "text", Text: "Hello, I am Claude."},
			},
			Model:      "claude-3-sonnet-20240229",
			StopReason: "end_turn",
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{
				InputTokens:  10,
				OutputTokens: 5,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-token")

	resp, err := client.Chat(context.Background(), ChatRequest{
		Model: "claude-3-sonnet-20240229",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.ID != "msg_0123456789" {
		t.Errorf("expected ID msg_0123456789, got %s", resp.ID)
	}
	if len(resp.Choices) == 0 {
		t.Fatal("expected at least one choice")
	}
	if resp.Choices[0].Message.Content != "Hello, I am Claude." {
		t.Errorf("expected content 'Hello, I am Claude.', got %s", resp.Choices[0].Message.Content)
	}
}

func TestAnthropicClient_Chat_AuthenticationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type": "error", "error": {"type": "authentication_error", "message": "Invalid API key"}}`))
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "invalid-token")

	_, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	})

	if err == nil {
		t.Fatal("expected error for authentication failure")
	}
	if err.Error() != "anthropic api error (401): invalid api key" {
		t.Errorf("expected authentication error, got: %v", err)
	}
}

func TestAnthropicClient_Chat_RateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("retry-after", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"type": "error", "error": {"type": "rate_limit_error", "message": "Rate limit exceeded"}}`))
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-token")

	_, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	})

	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	expected := "anthropic api error (429): rate limited, retry after 60"
	if err.Error() != expected {
		t.Errorf("expected rate limit error with retry-after, got: %v", err)
	}
}

func TestAnthropicClient_Chat_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-token")

	_, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	})

	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestAnthropicClient_StreamChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != "test-token" {
			t.Errorf("expected x-api-key header")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Write SSE events
		fmt.Fprintln(w, "event: content_block_delta")
		fmt.Fprintln(w, "data: {\"delta\": {\"text\": \"Hello\"}}")
		fmt.Fprintln(w, "")

		fmt.Fprintln(w, "event: content_block_delta")
		fmt.Fprintln(w, "data: {\"delta\": {\"text\": \",\"}}")
		fmt.Fprintln(w, "")

		fmt.Fprintln(w, "event: content_block_delta")
		fmt.Fprintln(w, "data: {\"delta\": {\"text\": \" world!\"}}")
		fmt.Fprintln(w, "")

		fmt.Fprintln(w, "event: message_stop")
		fmt.Fprintln(w, "data: {}")
		fmt.Fprintln(w, "")

		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-token")

	streamCh, err := client.StreamChat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("StreamChat failed: %v", err)
	}

	// Collect tokens
	tokens := make([]string, 0)
	done := false
	for event := range streamCh {
		if event.Error != nil {
			t.Fatalf("stream error: %v", event.Error)
		}
		if event.Done {
			done = true
			break
		}
		tokens = append(tokens, event.Token)
	}

	if !done {
		t.Error("expected stream to signal done")
	}

	expected := []string{"Hello", ",", " world!"}
	if len(tokens) != len(expected) {
		t.Errorf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i, token := range tokens {
		if token != expected[i] {
			t.Errorf("token %d: expected %q, got %q", i, expected[i], token)
		}
	}
}

func TestAnthropicClient_StreamChat_ErrorEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintln(w, "event: error")
		fmt.Fprintln(w, "data: {\"error\": {\"type\": \"overloaded_error\", \"message\": \"Server overloaded\"}}")
		fmt.Fprintln(w, "")

		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-token")

	streamCh, err := client.StreamChat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("StreamChat failed: %v", err)
	}

	// Wait for error event
	var streamErr error
	for event := range streamCh {
		if event.Error != nil {
			streamErr = event.Error
			break
		}
	}

	if streamErr == nil {
		t.Error("expected error event in stream")
	}
}

func TestAnthropicClient_StreamChat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"type": "error", "error": {"type": "api_error", "message": "Service unavailable"}}`))
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-token")

	_, err := client.StreamChat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	})

	if err == nil {
		t.Fatal("expected error for HTTP error")
	}
}

func TestAnthropicClient_StreamChat_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintln(w, "event: content_block_delta")
		fmt.Fprintln(w, "data: {invalid json}")
		fmt.Fprintln(w, "")

		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-token")

	streamCh, err := client.StreamChat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("StreamChat failed: %v", err)
	}

	// Wait for error
	var streamErr error
	for event := range streamCh {
		if event.Error != nil {
			streamErr = event.Error
			break
		}
	}

	if streamErr == nil {
		t.Error("expected parse error in stream")
	}
}

func TestAnthropicClient_Chat_ConvertsMessages(t *testing.T) {
	var capturedReq AnthropicMessageRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedReq)

		response := AnthropicMessageResponse{
			ID:      "msg_test",
			Type:    "message",
			Role:    "assistant",
			Content: []AnthropicContentBlock{{Type: "text", Text: "Response"}},
			Model:   "claude-3-sonnet-20240229",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-token")
	client.Chat(context.Background(), ChatRequest{
		Model: "claude-3-sonnet-20240229",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
			{Role: "user", Content: "How are you?"},
		},
	})

	// Verify messages were converted correctly
	if len(capturedReq.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(capturedReq.Messages))
	}
	if capturedReq.Messages[0].Role != "user" || capturedReq.Messages[0].Content != "Hello" {
		t.Errorf("first message mismatch: %+v", capturedReq.Messages[0])
	}
	if capturedReq.Messages[1].Role != "assistant" || capturedReq.Messages[1].Content != "Hi there" {
		t.Errorf("second message mismatch: %+v", capturedReq.Messages[1])
	}
	if capturedReq.Model != "claude-3-sonnet-20240229" {
		t.Errorf("expected model claude-3-sonnet-20240229, got %s", capturedReq.Model)
	}
}

func TestAnthropicClient_Chat_ResponseConversion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := AnthropicMessageResponse{
			ID:         "msg_abc123",
			Type:       "message",
			Role:       "assistant",
			Content:    []AnthropicContentBlock{{Type: "text", Text: "The answer is 42."}},
			Model:      "claude-3-sonnet-20240229",
			StopReason: "end_turn",
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{
				InputTokens:  25,
				OutputTokens: 10,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-token")
	resp, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "What is the answer?"}},
	})

	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	// Verify response conversion
	if resp.ID != "msg_abc123" {
		t.Errorf("expected ID msg_abc123, got %s", resp.ID)
	}
	if resp.Model != "claude-3-sonnet-20240229" {
		t.Errorf("expected model claude-3-sonnet-20240229, got %s", resp.Model)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "The answer is 42." {
		t.Errorf("expected content 'The answer is 42.', got %s", resp.Choices[0].Message.Content)
	}
	if resp.Choices[0].FinishReason != "end_turn" {
		t.Errorf("expected finish_reason end_turn, got %s", resp.Choices[0].FinishReason)
	}
	if resp.Usage.PromptTokens != 25 {
		t.Errorf("expected prompt_tokens 25, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 10 {
		t.Errorf("expected completion_tokens 10, got %d", resp.Usage.CompletionTokens)
	}
	if resp.Usage.TotalTokens != 35 {
		t.Errorf("expected total_tokens 35, got %d", resp.Usage.TotalTokens)
	}
}

func TestAnthropicClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-token")
	client.httpClient.Timeout = 100 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Chat(ctx, ChatRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	})

	if err == nil {
		t.Error("expected timeout error")
	}
}
