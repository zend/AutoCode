package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zend/AutoCode/internal/llm"
	"github.com/zend/AutoCode/internal/tools"
)

const systemPrompt = `You are an AI coding agent that follows the ReAct (Reasoning + Acting) pattern.

You have access to the following tools:
1. read - Read files or directories
2. write - Write or edit files
3. grep - Search for patterns in files
4. shell - Execute shell commands

For each step, you should:
1. Think about what to do next
2. Choose and execute a tool
3. Observe the result
4. Repeat until the task is complete

Format your responses as JSON:
{"thought": "your reasoning", "action": "tool_name", "action_input": {"param": "value"}}

When the task is complete, respond with:
{"thought": "task completed", "finish": true, "result": "final result"}`

// LLMClient defines the interface for LLM operations needed by the Agent
type LLMClient interface {
	StreamChat(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamEvent, error)
	Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error)
}

type Agent struct {
	client     LLMClient
	registry   *tools.ToolRegistry
	baseDir    string
	maxSteps   int
	history    []Message
	eventCh    chan AgentEvent // NEW: events to TUI
	cancelCh   chan struct{}   // NEW: cancellation signal
	cancelOnce sync.Once       // NEW: ensures Cancel() only closes channel once
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AgentResponse struct {
	Thought     string                 `json:"thought"`
	Action      string                 `json:"action,omitempty"`
	ActionInput map[string]interface{} `json:"action_input,omitempty"`
	Finish      bool                   `json:"finish,omitempty"`
	Result      string                 `json:"result,omitempty"`
}

func New(client LLMClient, baseDir string) *Agent {
	registry := tools.NewRegistry()
	registry.Register(tools.NewReadTool(baseDir))
	registry.Register(tools.NewWriteTool(baseDir))
	registry.Register(tools.NewGrepTool(baseDir))
	registry.Register(tools.NewShellTool(baseDir))

	return &Agent{
		client:   client,
		registry: registry,
		baseDir:  baseDir,
		maxSteps: 50,
		history:  make([]Message, 0),
		eventCh:  make(chan AgentEvent, 100),
	}
}

func (a *Agent) SetMaxSteps(max int) {
	a.maxSteps = max
}

// EventChannel returns the channel for receiving agent events
func (a *Agent) EventChannel() <-chan AgentEvent {
	return a.eventCh
}

// Cancel signals the agent to stop processing
func (a *Agent) Cancel() {
	a.cancelOnce.Do(func() { close(a.cancelCh) })
}

// publishEvent safely sends an event to the event channel with timeout
func (a *Agent) publishEvent(event AgentEvent) {
	select {
	case a.eventCh <- event:
	case <-time.After(100 * time.Millisecond):
		// Channel blocked or closed, drop event
	}
}

// Run starts the agent processing in an event-driven manner
// Events are published to eventCh instead of returning results directly
func (a *Agent) Run(ctx context.Context, task string) error {
	a.cancelCh = make(chan struct{}) // Reset for each run
	// Reset cancelOnce so Cancel() can be called again
	a.cancelOnce = sync.Once{}

	a.history = append(a.history, Message{
		Role:    "user",
		Content: task,
	})

	messages := make([]llm.Message, 0, len(a.history)+1)
	messages = append(messages, llm.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	for i := 0; i < a.maxSteps; i++ {
		// Check for cancellation at start of each step
		select {
		case <-a.cancelCh:
			a.publishEvent(StepCompleteEvent{
				Step:        i,
				Finished:    false,
				Interrupted: true,
				Result:      "Cancelled by user",
			})
			return nil
		default:
		}

		llmMessages := make([]llm.Message, len(messages))
		copy(llmMessages, messages)
		for _, h := range a.history {
			llmMessages = append(llmMessages, llm.Message{
				Role:    h.Role,
				Content: h.Content,
			})
		}

		// Stream LLM response
		streamCh, err := a.client.StreamChat(ctx, llm.ChatRequest{
			Model:    "gpt-4",
			Messages: llmMessages,
		})
		if err != nil {
			return fmt.Errorf("llm stream chat: %w", err)
		}

		// Collect streaming response and publish thinking events
		var assistantMsg strings.Builder
		err = a.streamThinking(i, streamCh, &assistantMsg)
		if err != nil {
			return err
		}

		content := assistantMsg.String()
		a.history = append(a.history, Message{
			Role:    "assistant",
			Content: content,
		})

		agentResp, err := a.parseResponse(content)
		if err != nil {
			observation := fmt.Sprintf("Error parsing response: %v. Please respond with valid JSON.", err)
			a.history = append(a.history, Message{
				Role:    "user",
				Content: observation,
			})
			a.publishEvent(StepCompleteEvent{
				Step:        i,
				Finished:    false,
				Interrupted: false,
				Result:      observation,
			})
			continue
		}

		if agentResp.Finish {
			a.publishEvent(StepCompleteEvent{
				Step:        i,
				Finished:    true,
				Interrupted: false,
				Result:      agentResp.Result,
			})
			return nil
		}

		// Execute tool with events
		observation, err := a.executeToolWithEvents(ctx, i, agentResp)
		if err != nil {
			observation = fmt.Sprintf("Error: %v", err)
		}

		a.history = append(a.history, Message{
			Role:    "user",
			Content: fmt.Sprintf("Observation: %s", observation),
		})

		a.publishEvent(StepCompleteEvent{
			Step:        i,
			Finished:    false,
			Interrupted: false,
			Result:      observation,
		})
	}

	// Max steps exceeded
	a.publishEvent(StepCompleteEvent{
		Step:        a.maxSteps,
		Finished:    false,
		Interrupted: false,
		Result:      fmt.Sprintf("max steps (%d) exceeded", a.maxSteps),
	})
	return fmt.Errorf("max steps (%d) exceeded", a.maxSteps)
}

// streamThinking processes the LLM stream and publishes ThinkingEvents
func (a *Agent) streamThinking(step int, streamCh <-chan llm.StreamEvent, builder *strings.Builder) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var currentThought strings.Builder
	lastPublishLen := 0

	for {
		select {
		case <-a.cancelCh:
			// Drain streamCh to prevent goroutine leak
			go func() {
				for range streamCh {
				}
			}()
			return fmt.Errorf("interrupted")
		case event, ok := <-streamCh:
			if !ok {
				// Stream closed, publish final thinking event
				publishFinal := currentThought.Len() > lastPublishLen
				if publishFinal {
					a.publishEvent(ThinkingEvent{
						Step:      step,
						Thought:   currentThought.String(),
						Streaming: false,
					})
				}
				builder.WriteString(currentThought.String())
				return nil
			}

			if event.Error != nil {
				return event.Error
			}

			if event.Done {
				// Final thinking event
				publishFinal := currentThought.Len() > lastPublishLen
				if publishFinal {
					a.publishEvent(ThinkingEvent{
						Step:      step,
						Thought:   currentThought.String(),
						Streaming: false,
					})
				}
				builder.WriteString(currentThought.String())
				return nil
			}

			currentThought.WriteString(event.Token)

		case <-ticker.C:
			// Throttle events to every 50ms
			if currentThought.Len() > lastPublishLen {
				a.publishEvent(ThinkingEvent{
					Step:      step,
					Thought:   currentThought.String(),
					Streaming: true,
				})
				lastPublishLen = currentThought.Len()
			}
		}
	}
}

// executeToolWithEvents executes a tool and publishes ToolStartEvent and ToolCompleteEvent
func (a *Agent) executeToolWithEvents(ctx context.Context, step int, resp *AgentResponse) (string, error) {
	if resp.Action == "" {
		return "", fmt.Errorf("no action specified")
	}

	tool, ok := a.registry.Get(resp.Action)
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", resp.Action)
	}

	// Publish ToolStartEvent
	a.publishEvent(ToolStartEvent{
		Step:   step,
		Action: resp.Action,
		Input:  resp.ActionInput,
	})

	inputBytes, err := json.Marshal(resp.ActionInput)
	if err != nil {
		a.publishEvent(ToolCompleteEvent{
			Step:   step,
			Action: resp.Action,
			Output: "",
			Error:  fmt.Sprintf("marshal action input: %v", err),
		})
		return "", fmt.Errorf("marshal action input: %w", err)
	}

	startTime := time.Now()
	output, err := tool.Execute(ctx, string(inputBytes))
	duration := time.Since(startTime)

	// Publish ToolCompleteEvent
	toolErr := ""
	if err != nil {
		toolErr = err.Error()
	}
	a.publishEvent(ToolCompleteEvent{
		Step:     step,
		Action:   resp.Action,
		Output:   output,
		Error:    toolErr,
		Duration: duration,
	})

	return output, err
}

func (a *Agent) parseResponse(content string) (*AgentResponse, error) {
	content = strings.TrimSpace(content)

	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd < jsonStart {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	jsonStr := content[jsonStart : jsonEnd+1]

	var resp AgentResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	return &resp, nil
}

func (a *Agent) GetHistory() []Message {
	return a.history
}
