package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/zend/AutoCode/internal/agent"
	appctx "github.com/zend/AutoCode/internal/context"
	"github.com/zend/AutoCode/internal/llm"
	"github.com/zend/AutoCode/internal/tui"
)

func init() {
	// Set lipgloss color profile to ANSI256 to avoid OSC escape sequences
	// that some terminals display as garbage characters
	lipgloss.SetColorProfile(termenv.ANSI256)
}

func main() {
	// Initialize traffic logger
	if err := llm.InitTrafficLogger("traffic.log"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to init traffic logger: %v\n", err)
	}
	defer llm.CloseTrafficLogger()

	// Command-line flags
	var (
		// LLM options
		provider    = flag.String("provider", "", "LLM provider (anthropic/openai)")
		model       = flag.String("model", "", "Model name")
		baseURL     = flag.String("base-url", "", "API base URL")
		temperature = flag.Float64("temperature", 0, "Generation temperature")

		// Context options
		maxContextTokens = flag.Int("max-context-tokens", 0, "Max context tokens")
		maxToolOutput    = flag.Int("max-tool-output", 0, "Max tool output bytes")
		historyWindow    = flag.Int("history-window", 0, "History window size")

		// Memory options
		noMemory        = flag.Bool("no-memory", false, "Disable memory")
		shortMemoryDays = flag.Int("short-memory-days", 0, "Short-term memory retention days")
		autoArchive     = flag.Bool("auto-archive", false, "Auto-archive old memory")

		// Session options
		noSession    = flag.Bool("no-session", false, "Disable session persistence")
		sessionDays  = flag.Int("session-days", 0, "Session retention days")
		autoResume   = flag.Bool("auto-resume", false, "Auto-resume last session")
		resumeSession = flag.String("resume", "", "Resume specific session ID")

		// Other options
		configDir     = flag.String("config-dir", "", "Config directory")
		loadYesterday = flag.Bool("load-yesterday", false, "Load yesterday's memory")
		verbose       = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	// Determine config directory
	configPath := *configDir
	if configPath == "" {
		configPath = appctx.GetConfigDir()
	}

	// Ensure config directory exists
	if err := appctx.EnsureConfigDir(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create config directory: %v\n", err)
		os.Exit(1)
	}

	// Create default files if needed
	if err := appctx.CreateDefaultFiles(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create default files: %v\n", err)
	}

	// Display welcome banner
	loader := appctx.NewContextLoader(configPath, ".")
	banner := loader.LoadBanner()
	if banner != "" {
		fmt.Print(banner)
	}

	// Load settings
	settings := appctx.NewSettingsManager(configPath)
	if err := settings.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load settings: %v\n", err)
	}

	// Apply environment variable overrides
	settings.ApplyEnvOverrides()

	// Apply command-line flags
	settings.ApplyCLIFlags(
		*provider, *model, *baseURL,
		0, *temperature,
		*maxContextTokens, *maxToolOutput, *historyWindow,
		*noMemory, *shortMemoryDays, *autoArchive,
		*noSession, *sessionDays, *autoResume,
	)

	// Get effective settings
	s := settings.Get()

	// Detect provider based on environment variables and settings
	anthropicToken := os.Getenv("ANTHROPIC_AUTH_TOKEN")
	openaiKey := os.Getenv("OPENAI_API_KEY")

	var client agent.LLMClient
	var providerName string
	var modelName string

	// Determine provider from settings or environment
	effectiveProvider := s.LLM.Provider
	if effectiveProvider == "" {
		if anthropicToken != "" {
			effectiveProvider = "anthropic"
		} else if openaiKey != "" {
			effectiveProvider = "openai"
		}
	}

	switch effectiveProvider {
	case "anthropic":
		base := s.LLM.BaseURL
		if base == "" {
			if envURL := os.Getenv("ANTHROPIC_BASE_URL"); envURL != "" {
				base = envURL
			} else {
				base = "https://api.anthropic.com/v1"
			}
		}
		token := anthropicToken
		if token == "" {
			fmt.Fprintln(os.Stderr, "Error: ANTHROPIC_AUTH_TOKEN not set")
			os.Exit(1)
		}
		anthropicClient := llm.NewAnthropicClient(base, token)
		if s.LLM.Model != "" {
			anthropicClient.SetModel(s.LLM.Model)
			modelName = s.LLM.Model
		} else {
			modelName = "claude-sonnet-4-6"
		}
		client = anthropicClient
		providerName = "Anthropic"

	case "openai":
		base := s.LLM.BaseURL
		if base == "" {
			if envURL := os.Getenv("OPENAI_BASE_URL"); envURL != "" {
				base = envURL
			} else {
				base = "https://api.openai.com/v1"
			}
		}
		key := openaiKey
		if key == "" {
			fmt.Fprintln(os.Stderr, "Error: OPENAI_API_KEY not set")
			os.Exit(1)
		}
		client = llm.NewClient(base, key)
		if s.LLM.Model != "" {
			modelName = s.LLM.Model
		} else {
			modelName = "gpt-4"
		}
		providerName = "OpenAI"

	default:
		fmt.Fprintln(os.Stderr, "Error: No LLM provider configured")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Please set one of the following environment variables:")
		fmt.Fprintln(os.Stderr, "  - ANTHROPIC_AUTH_TOKEN: For Anthropic Claude models")
		fmt.Fprintln(os.Stderr, "  - OPENAI_API_KEY: For OpenAI GPT models")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Or configure in ~/.config/autocode/settings.json")
		os.Exit(1)
	}

	// Print verbose info
	if *verbose {
		fmt.Printf("Provider: %s\n", providerName)
		fmt.Printf("Model: %s\n", modelName)
		fmt.Printf("Config dir: %s\n", configPath)
		fmt.Printf("Max context tokens: %d\n", s.Context.MaxContextTokens)
		fmt.Printf("Memory enabled: %v\n", s.Memory.EnableShortTerm)
		fmt.Printf("Session persistence: %v\n", s.Session.EnablePersistence)
	}

	// Create Agent with settings
	opts := []agent.AgentOption{
		agent.WithSettings(settings),
	}
	if *configDir != "" {
		opts = append(opts, agent.WithConfigDir(configPath))
	}

	agentInstance := agent.New(client, ".", opts...)

	// Handle session resume
	if *resumeSession != "" {
		// Resume specific session - would need session manager access
		fmt.Printf("Resuming session: %s\n", *resumeSession)
	} else if *autoResume || s.Session.AutoResume {
		// Auto-resume last session
		if *verbose {
			fmt.Println("Auto-resume enabled")
		}
	}

	// Load yesterday's memory if requested
	if *loadYesterday {
		loader := appctx.NewContextLoader(configPath, ".")
		yesterday := loader.LoadYesterdayMemory()
		if yesterday != "" && *verbose {
			fmt.Println("Yesterday's memory loaded")
		}
	}

	// Create TUI Model
	tuiModel := tui.NewModel(agentInstance, providerName, modelName)

	// Run TUI (without alt screen - shell-style output)
	p := tea.NewProgram(tuiModel)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}