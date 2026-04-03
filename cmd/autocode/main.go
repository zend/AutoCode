package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/zend/AutoCode/internal/agent"
	"github.com/zend/AutoCode/internal/llm"
	"github.com/zend/AutoCode/internal/tui"
)

func main() {
	// Initialize traffic logger
	if err := llm.InitTrafficLogger("traffic.log"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to init traffic logger: %v\n", err)
	}
	defer llm.CloseTrafficLogger()

	// Detect provider based on environment variables
	anthropicToken := os.Getenv("ANTHROPIC_AUTH_TOKEN")
	openaiKey := os.Getenv("OPENAI_API_KEY")

	var client agent.LLMClient
	var providerName string
	var modelName string

	if anthropicToken != "" {
		// Use Anthropic provider
		baseURL := os.Getenv("ANTHROPIC_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.anthropic.com/v1"
		}
		anthropicClient := llm.NewAnthropicClient(baseURL, anthropicToken)
		// Set custom model if provided
		modelName = os.Getenv("ANTHROPIC_MODEL")
		if modelName != "" {
			anthropicClient.SetModel(modelName)
		} else {
			modelName = "claude-3-sonnet-20240229" // default
		}
		client = anthropicClient
		providerName = "Anthropic"
	} else if openaiKey != "" {
		// Use OpenAI provider
		baseURL := os.Getenv("OPENAI_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		client = llm.NewClient(baseURL, openaiKey)
		providerName = "OpenAI"
		modelName = "gpt-4" // default
	} else {
		fmt.Fprintln(os.Stderr, "Error: No LLM provider configured")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Please set one of the following environment variables:")
		fmt.Fprintln(os.Stderr, "  - ANTHROPIC_AUTH_TOKEN: For Anthropic Claude models")
		fmt.Fprintln(os.Stderr, "  - OPENAI_API_KEY: For OpenAI GPT models")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Optional environment variables:")
		fmt.Fprintln(os.Stderr, "  - ANTHROPIC_BASE_URL: Custom Anthropic API endpoint (default: https://api.anthropic.com)")
		fmt.Fprintln(os.Stderr, "  - ANTHROPIC_MODEL: Model to use (default: claude-3-sonnet-20240229)")
		fmt.Fprintln(os.Stderr, "  - OPENAI_BASE_URL: Custom OpenAI API endpoint (default: https://api.openai.com/v1)")
		os.Exit(1)
	}

	// Create Agent
	agentInstance := agent.New(client, ".")

	// Create TUI Model
	model := tui.NewModel(agentInstance, providerName, modelName)

	// Run TUI
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
