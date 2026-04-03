package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AnthropicClient is a client for Anthropic's Messages API
type AnthropicClient struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
	model      string
}

// AnthropicMessage represents a message in Anthropic's API
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicMessageRequest represents a request to Anthropic's Messages API
type AnthropicMessageRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

// AnthropicMessageResponse represents a response from Anthropic's Messages API
type AnthropicMessageResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Content      []AnthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason,omitempty"`
	StopSequence string                  `json:"stop_sequence,omitempty"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// AnthropicContentBlock represents a content block in the response
type AnthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// NewAnthropicClient creates a new Anthropic API client
func NewAnthropicClient(baseURL, authToken string) *AnthropicClient {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	return &AnthropicClient{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		model: "claude-3-sonnet-20240229",
	}
}

// SetModel sets the model to use for requests
func (c *AnthropicClient) SetModel(model string) {
	c.model = model
}

// Chat sends a chat completion request to Anthropic
func (c *AnthropicClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()

	// Convert OpenAI format to Anthropic format
	anthropicReq := AnthropicMessageRequest{
		Model:       c.model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	// Set default max_tokens if not provided (required by Anthropic API)
	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = 4096
	}

	// Convert messages
	for _, msg := range req.Messages {
		role := msg.Role
		// Anthropic API doesn't support "system" role in messages array
		// Convert system messages to "user" role (or skip them)
		if role == "system" {
			role = "user"
		}
		anthropicReq.Messages = append(anthropicReq.Messages, AnthropicMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Log request
	LogRequest("anthropic", c.model, anthropicReq)

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		LogError("anthropic", c.model, err)
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Construct the full URL, handling both standard and custom endpoints
	// Standard: https://api.anthropic.com/v1/messages
	// Custom: user-provided base URL may or may not have /v1 suffix
	url := c.baseURL
	if !strings.HasSuffix(url, "/v1") && !strings.Contains(url, "/v1/") {
		url = url + "/v1"
	}
	url = url + "/messages"

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.authToken)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, respBody, resp.Header)
	}

	var anthropicResp AnthropicMessageResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Convert Anthropic response to OpenAI format
	chatResp := &ChatResponse{
		ID:      anthropicResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   anthropicResp.Model,
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}

	// Extract text content from content blocks
	var content strings.Builder
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			content.WriteString(block.Text)
		}
	}

	chatResp.Choices = []struct {
		Index        int     `json:"index"`
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	}{
		{
			Index: 0,
			Message: Message{
				Role:    "assistant",
				Content: content.String(),
			},
			FinishReason: anthropicResp.StopReason,
		},
	}

	// Log response
	LogResponse("anthropic", c.model, anthropicResp, time.Since(startTime))

	return chatResp, nil
}

// handleErrorResponse handles Anthropic-specific error responses
func (c *AnthropicClient) handleErrorResponse(statusCode int, body []byte, headers http.Header) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("anthropic api error (401): invalid api key")
	case http.StatusTooManyRequests:
		retryAfter := headers.Get("retry-after")
		if retryAfter != "" {
			return fmt.Errorf("anthropic api error (429): rate limited, retry after %s", retryAfter)
		}
		return fmt.Errorf("anthropic api error (429): rate limited")
	default:
		return fmt.Errorf("anthropic api error (status %d): %s", statusCode, string(body))
	}
}

// StreamChat sends a streaming chat request to Anthropic
func (c *AnthropicClient) StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	startTime := time.Now()

	anthropicReq := AnthropicMessageRequest{
		Model:       c.model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	// Set default max_tokens if not provided (required by Anthropic API)
	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = 4096
	}

	// Convert messages
	for _, msg := range req.Messages {
		role := msg.Role
		// Anthropic API doesn't support "system" role in messages array
		// Convert system messages to "user" role
		if role == "system" {
			role = "user"
		}
		anthropicReq.Messages = append(anthropicReq.Messages, AnthropicMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Log request
	LogRequest("anthropic", c.model, anthropicReq)

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		LogError("anthropic", c.model, err)
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Construct the full URL, handling both standard and custom endpoints
	// Standard: https://api.anthropic.com/v1/messages
	// Custom: user-provided base URL may or may not have /v1 suffix
	url := c.baseURL
	if !strings.HasSuffix(url, "/v1") && !strings.Contains(url, "/v1/") {
		url = url + "/v1"
	}
	url = url + "/messages"

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.authToken)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, c.handleErrorResponse(resp.StatusCode, body, resp.Header)
	}

	ch := make(chan StreamEvent, 10)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		reader := bufio.NewReader(resp.Body)
		var eventType string
		var fullContent strings.Builder // Collect full response for logging

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					ch <- StreamEvent{Error: fmt.Errorf("read stream: %w", err)}
					LogError("anthropic", c.model, err)
				} else {
					// Log complete response
					LogResponse("anthropic", c.model, map[string]string{
						"content": fullContent.String(),
					}, time.Since(startTime))
				}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Parse event type
			if strings.HasPrefix(line, "event:") {
				eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}

			// Parse data
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

				switch eventType {
				case "content_block_delta":
					var deltaEvent struct {
						Delta struct {
							Text string `json:"text"`
						} `json:"delta"`
					}
					if err := json.Unmarshal([]byte(data), &deltaEvent); err != nil {
						ch <- StreamEvent{Error: fmt.Errorf("parse delta: %w", err)}
						continue
					}
					if deltaEvent.Delta.Text != "" {
						fullContent.WriteString(deltaEvent.Delta.Text)
						ch <- StreamEvent{Token: deltaEvent.Delta.Text}
					}

				case "message_stop":
					ch <- StreamEvent{Done: true}
					// Log complete response
					LogResponse("anthropic", c.model, map[string]string{
						"content": fullContent.String(),
					}, time.Since(startTime))
					return

				case "error":
					err := fmt.Errorf("anthropic stream error: %s", data)
					ch <- StreamEvent{Error: err}
					LogError("anthropic", c.model, err)
					return
				}
			}
		}
	}()

	return ch, nil
}
