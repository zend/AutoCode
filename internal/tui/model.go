package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/zend/AutoCode/internal/agent"
)

// tickMsg is sent when listenForEvents times out to prevent blocking
type tickMsg struct{}

// Model implements tea.Model for the chat-style TUI
type Model struct {
	agent        *agent.Agent
	viewport     viewport.Model
	messages     []Message
	renderer     *glamour.TermRenderer
	input        string
	running      bool
	width        int
	height       int
	ready        bool
	glamourErr   error
	stopListen   chan struct{}      // Channel to stop event listening
	cancelCtx    context.CancelFunc // Function to cancel agent context
	providerName string             // Provider name (Anthropic/OpenAI)
	modelName    string             // Model name
}

// NewModel creates a new TUI model with chat interface
func NewModel(agent *agent.Agent, providerName, modelName string) *Model {
	return &Model{
		agent:        agent,
		messages:     make([]Message, 0),
		input:        "",
		providerName: providerName,
		modelName:    modelName,
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	// Initialize glamour renderer with dark theme
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		m.glamourErr = err
	}
	m.renderer = renderer

	return m.listenForEvents()
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			// Initialize viewport with available height minus input area
			m.viewport = viewport.New(msg.Width, msg.Height-4)
			m.viewport.SetContent(m.renderMessages())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 4
		}
		m.viewport.SetContent(m.renderMessages())

	case tea.KeyMsg:
		return m.handleKey(msg)

	case agent.AgentEvent:
		if msg == nil {
			// Stop signal received, don't continue listening
			return m, nil
		}
		return m.handleAgentEvent(msg)

	case tickMsg:
		// Timeout occurred, continue listening if still running
		if m.running {
			return m, m.listenForEvents()
		}
		return m, nil
	}

	// Handle viewport updates
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleKey handles keyboard input
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlD:
		return m, tea.Quit

	case tea.KeyEsc:
		if m.running {
			if m.agent != nil {
				m.agent.Cancel()
			}
			if m.cancelCtx != nil {
				m.cancelCtx()
			}
			m.running = false
			return m, nil
		}
		// Clear input if not running
		if m.input != "" {
			m.input = ""
		}
		return m, nil

	case tea.KeyEnter:
		if m.input != "" && !m.running {
			task := m.input
			m.input = ""
			m.running = true

			// Add user message
			m.messages = append(m.messages, NewUserMessage(task))
			m.updateViewport()

			return m, tea.Batch(m.runAgent(task), m.listenForEvents())
		}
		return m, nil

	case tea.KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
		return m, nil

	case tea.KeyUp:
		// Scroll viewport up
		m.viewport.LineUp(1)
		return m, nil

	case tea.KeyDown:
		// Scroll viewport down
		m.viewport.LineDown(1)
		return m, nil

	case tea.KeyPgUp:
		m.viewport.HalfViewUp()
		return m, nil

	case tea.KeyPgDown:
		m.viewport.HalfViewDown()
		return m, nil

	case tea.KeyRunes:
		m.input += msg.String()
		return m, nil
	}

	return m, nil
}

// handleAgentEvent processes events from the agent
func (m *Model) handleAgentEvent(event agent.AgentEvent) (tea.Model, tea.Cmd) {
	switch e := event.(type) {
	case agent.ThinkingEvent:
		// Find or create assistant message
		if len(m.messages) == 0 || !m.messages[len(m.messages)-1].IsAssistant() {
			m.messages = append(m.messages, NewAssistantMessage(""))
		}

		// Update the last assistant message
		lastMsg := &m.messages[len(m.messages)-1]
		lastMsg.Content = e.Thought
		m.updateViewport()

	case agent.ToolStartEvent:
		// Append tool info to the last assistant message
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].IsAssistant() {
			lastMsg := &m.messages[len(m.messages)-1]
			toolInfo := fmt.Sprintf("\n\n**Tool:** %s\n", e.Action)
			if e.Input != nil {
				toolInfo += fmt.Sprintf("**Input:** `%s`\n", formatJSON(e.Input))
			}
			lastMsg.Content += toolInfo
			m.updateViewport()
		}

	case agent.ToolCompleteEvent:
		// Append tool result to the last assistant message
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].IsAssistant() {
			lastMsg := &m.messages[len(m.messages)-1]
			if e.Error != "" {
				lastMsg.Content += fmt.Sprintf("\n**Error:** %s\n", e.Error)
			} else {
				lastMsg.Content += fmt.Sprintf("\n**Output:**\n```\n%s\n```\n", e.Output)
			}
			m.updateViewport()
		}

	case agent.StepCompleteEvent:
		if e.Finished {
			m.running = false
			// Finalize the last assistant message with result
			if len(m.messages) > 0 && m.messages[len(m.messages)-1].IsAssistant() {
				lastMsg := &m.messages[len(m.messages)-1]
				if e.Result != "" {
					lastMsg.Content += fmt.Sprintf("\n\n**Result:** %s\n", e.Result)
				}
				m.updateViewport()
			}
			// Don't continue listening - task is complete
			return m, nil
		} else if e.Interrupted {
			m.running = false
			if len(m.messages) > 0 && m.messages[len(m.messages)-1].IsAssistant() {
				lastMsg := &m.messages[len(m.messages)-1]
				lastMsg.Content += fmt.Sprintf("\n\n*Interrupted: %s*\n", e.Result)
				m.updateViewport()
			}
			// Don't continue listening - task was interrupted
			return m, nil
		} else if e.Step < 0 {
			// Error case (Step = -1 from runAgent) - display error and stop
			m.running = false
			if e.Result != "" {
				// Add error as assistant message
				m.messages = append(m.messages, NewAssistantMessage(fmt.Sprintf("**Error:** %s", e.Result)))
				m.updateViewport()
			}
			// Don't continue listening - there was an error
			return m, nil
		}
		// Otherwise it's an intermediate step - continue listening
	}

	return m, m.listenForEvents()
}

// formatJSON formats a map as compact JSON
func formatJSON(input map[string]interface{}) string {
	if len(input) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(input))
	for k, v := range input {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, ", ")
}

// updateViewport refreshes the viewport content
func (m *Model) updateViewport() {
	m.viewport.SetContent(m.renderMessages())
	// Auto-scroll to bottom
	m.viewport.GotoBottom()
}

// runAgent runs the agent in a goroutine
func (m *Model) runAgent(task string) tea.Cmd {
	// Create stop channel for this run
	m.stopListen = make(chan struct{})

	return func() tea.Msg {
		// Use a context with timeout and cancellation
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		m.cancelCtx = cancel
		defer func() {
			m.cancelCtx = nil
		}()

		err := m.agent.Run(ctx, task)

		// Close stop channel to signal event listening to stop
		if m.stopListen != nil {
			close(m.stopListen)
		}

		if err != nil {
			return agent.StepCompleteEvent{
				Step:     -1,
				Finished: false,
				Result:   fmt.Sprintf("Error: %v", err),
			}
		}
		return agent.StepCompleteEvent{
			Step:     -1,
			Finished: true,
			Result:   "Task completed",
		}
	}
}

// listenForEvents listens for events from the agent
func (m *Model) listenForEvents() tea.Cmd {
	return func() tea.Msg {
		if m.agent == nil {
			return nil
		}
		eventCh := m.agent.EventChannel()

		// Use select to listen for events or stop signal
		// Also include a timeout to prevent blocking forever
		select {
		case event := <-eventCh:
			return event
		case <-m.stopListen:
			// Stop listening, return nil
			return nil
		case <-time.After(100 * time.Millisecond):
			// Timeout to prevent blocking forever
			// Return a tick to re-schedule listening
			return tickMsg{}
		}
	}
}

// View renders the TUI
func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var b strings.Builder

	// Header with provider info
	providerInfo := ""
	if m.providerName != "" && m.modelName != "" {
		providerInfo = fmt.Sprintf("  [%s: %s]", m.providerName, m.modelName)
	} else if m.providerName != "" {
		providerInfo = fmt.Sprintf("  [%s]", m.providerName)
	}
	header := headerStyle.Render("  AutoCode Agent — Chat Mode" + providerInfo)
	b.WriteString(header)
	b.WriteString("\n")

	// Viewport with messages
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Input area at bottom
	inputLine := m.renderInput()
	b.WriteString(inputLine)

	// Help text
	b.WriteString(helpStyle.Render(m.helpText()))

	return b.String()
}

// renderMessages renders the message history
func (m *Model) renderMessages() string {
	if len(m.messages) == 0 {
		return welcomeStyle.Render("Welcome to AutoCode! Type a task and press Enter to start.")
	}

	var b strings.Builder
	for _, msg := range m.messages {
		b.WriteString(m.renderMessage(msg))
		b.WriteString("\n\n")
	}

	return b.String()
}

// renderMessage renders a single message
func (m *Model) renderMessage(msg Message) string {
	var b strings.Builder

	if msg.IsUser() {
		// User message
		timestamp := userTimestampStyle.Render(msg.FormatTimestamp())
		b.WriteString(timestamp)
		b.WriteString("\n")

		// Wrap content in bubble style
		content := userMessageStyle.Render(msg.Content)
		b.WriteString(content)
	} else {
		// Assistant message
		timestamp := assistantTimestampStyle.Render(msg.FormatTimestamp())
		b.WriteString(timestamp)
		b.WriteString("\n")

		// Add assistant prefix
		b.WriteString(assistantPrefixStyle.Render("  AutoCode"))
		b.WriteString("\n")

		// Render markdown content using glamour if available
		if m.renderer != nil {
			rendered, err := m.renderer.Render(msg.Content)
			if err == nil {
				// Wrap in assistant style
				b.WriteString(assistantMessageStyle.Render(rendered))
			} else {
				b.WriteString(assistantMessageStyle.Render(msg.Content))
			}
		} else {
			b.WriteString(assistantMessageStyle.Render(msg.Content))
		}
	}

	return b.String()
}

// renderInput renders the input prompt at the bottom
func (m *Model) renderInput() string {
	var b strings.Builder

	// Separator line
	b.WriteString(separatorStyle.Render(strings.Repeat("─", m.width)))
	b.WriteString("\n")

	// Input prompt with "> " like Claude Code
	prompt := inputPromptStyle.Render("> ")
	input := inputStyle.Render(m.input)
	cursor := cursorStyle.Render("▋")

	if m.running {
		// Show spinner/status when running
		b.WriteString(prompt)
		b.WriteString(runningStyle.Render("Processing... (Press Esc to cancel)"))
	} else {
		b.WriteString(prompt)
		b.WriteString(input)
		b.WriteString(cursor)
	}

	b.WriteString("\n")

	return b.String()
}

// helpText returns context-sensitive help
func (m *Model) helpText() string {
	if m.running {
		return "  esc: cancel • ctrl+d: quit"
	}
	return "  enter: submit • esc: clear • ↑/↓: scroll • ctrl+d: quit"
}

// Styles
var (
	// Header
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#D97757")).
			Background(lipgloss.Color("#1A1A1A")).
			Padding(0, 1)

	// Welcome message
	welcomeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true).
			Margin(2, 4)

	// User message styles
	userMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E0E0E0")).
				Background(lipgloss.Color("#2D3748")).
				Padding(0, 1).
				MarginLeft(20).
				MarginRight(2).
				Width(60)

	userTimestampStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#606060")).
				Align(lipgloss.Right).
				MarginLeft(20).
				MarginRight(2)

	// Assistant message styles
	assistantMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E0E0E0")).
				MarginLeft(2).
				MarginRight(20)

	assistantTimestampStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#606060")).
				MarginLeft(2)

	assistantPrefixStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D97757")).
				Bold(true).
				MarginLeft(2)

	// Input styles
	inputPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D97757")).
				Bold(true)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D97757"))

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#404040"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060"))
)
