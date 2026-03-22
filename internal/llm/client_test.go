package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://api.example.com/v1", "test-key")
	if client == nil {
		t.Fatal("expected client, got nil")
	}
	if client.baseURL != "https://api.example.com/v1" {
		t.Errorf("expected baseURL https://api.example.com/v1, got %s", client.baseURL)
	}
	if client.apiKey != "test-key" {
		t.Errorf("expected apiKey test-key, got %s", client.apiKey)
	}
}

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected path /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization header 'Bearer test-key', got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "test-id",
			"object": "chat.completion",
			"created": 1234567890,
			"model": "gpt-4",
			"choices": [{
				"index": 0,
				"message": {"role": "assistant", "content": "Hello!"},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	resp, err := client.Chat(context.Background(), ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "user", Content: "Hi"},
		},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello!" {
		t.Errorf("expected content 'Hello!', got %s", resp.Choices[0].Message.Content)
	}
}