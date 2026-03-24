package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

type Agent struct {
	client   *llm.Client
	registry *tools.ToolRegistry
	baseDir  string
	maxSteps int
	history  []Message
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

func New(client *llm.Client, baseDir string) *Agent {
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
	}
}

func (a *Agent) SetMaxSteps(max int) {
	a.maxSteps = max
}

func (a *Agent) Run(ctx context.Context, task string) (string, error) {
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
		llmMessages := make([]llm.Message, len(messages))
		copy(llmMessages, messages)
		for _, h := range a.history {
			llmMessages = append(llmMessages, llm.Message{
				Role:    h.Role,
				Content: h.Content,
			})
		}

		resp, err := a.client.Chat(ctx, llm.ChatRequest{
			Model:    "gpt-4",
			Messages: llmMessages,
		})
		if err != nil {
			return "", fmt.Errorf("llm chat: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no response from llm")
		}

		assistantMsg := resp.Choices[0].Message.Content
		a.history = append(a.history, Message{
			Role:    "assistant",
			Content: assistantMsg,
		})

		agentResp, err := a.parseResponse(assistantMsg)
		if err != nil {
			observation := fmt.Sprintf("Error parsing response: %v. Please respond with valid JSON.", err)
			a.history = append(a.history, Message{
				Role:    "user",
				Content: observation,
			})
			continue
		}

		if agentResp.Finish {
			return agentResp.Result, nil
		}

		observation, err := a.executeTool(ctx, agentResp)
		if err != nil {
			observation = fmt.Sprintf("Error: %v", err)
		}

		a.history = append(a.history, Message{
			Role:    "user",
			Content: fmt.Sprintf("Observation: %s", observation),
		})
	}

	return "", fmt.Errorf("max steps (%d) exceeded", a.maxSteps)
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

func (a *Agent) executeTool(ctx context.Context, resp *AgentResponse) (string, error) {
	if resp.Action == "" {
		return "", fmt.Errorf("no action specified")
	}

	tool, ok := a.registry.Get(resp.Action)
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", resp.Action)
	}

	inputBytes, err := json.Marshal(resp.ActionInput)
	if err != nil {
		return "", fmt.Errorf("marshal action input: %w", err)
	}

	return tool.Execute(ctx, string(inputBytes))
}

func (a *Agent) GetHistory() []Message {
	return a.history
}
