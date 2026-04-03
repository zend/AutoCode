package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/zend/AutoCode/internal/agent"
)

// tickMsg is sent when listenForEvents times out to prevent blocking
type tickMsg struct{}

// rendererMsg is sent when glamour renderer is ready
type rendererMsg struct {
	renderer *glamour.TermRenderer
	err      error
}

// Model implements tea.Model for the chat-style TUI
type Model struct {
	agent           *agent.Agent
	messages        []Message
	renderer        *glamour.TermRenderer
	input           string
	cursorPos       int      // Cursor position in input
	inputHistory    []string // History of user inputs
	historyIndex    int      // Current position in history (-1 = new input)
	running         bool
	width           int
	ready           bool
	glamourErr      error
	stopListen      chan struct{}      // Channel to stop event listening
	cancelCtx       context.CancelFunc // Function to cancel agent context
	providerName    string             // Provider name (Anthropic/OpenAI)
	modelName       string             // Model name
	currentStep     int                // Current agent step for tracking message updates
	printedMessages int                // Number of messages already printed to terminal
}

// NewModel creates a new TUI model with chat interface
func NewModel(agent *agent.Agent, providerName, modelName string) *Model {
	return &Model{
		agent:        agent,
		messages:     make([]Message, 0),
		input:        "",
		cursorPos:    0,
		inputHistory: make([]string, 0),
		historyIndex: -1,
		providerName: providerName,
		modelName:    modelName,
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	// No initial command needed - wait for WindowSizeMsg to initialize
	return nil
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		if !m.ready {
			m.ready = true
		}
		// Update glamour word wrap asynchronously
		return m, m.initRenderer(msg.Width - 4)

	case rendererMsg:
		// Glamour renderer ready
		if msg.err == nil {
			m.renderer = msg.renderer
		}
		return m, nil

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

	return m, nil
}

// handleKey handles keyboard input
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlD:
		// Only quit if input is empty and not running
		if m.input == "" && !m.running {
			return m, tea.Quit
		}
		return m, nil

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
			m.cursorPos = 0
		}
		return m, nil

	case tea.KeyCtrlU:
		// Clear input line
		m.input = ""
		m.cursorPos = 0
		m.historyIndex = -1
		return m, nil

	case tea.KeyCtrlW:
		// Delete word before cursor
		if m.cursorPos > 0 {
			// Find start of word
			pos := m.cursorPos - 1
			for pos > 0 && m.input[pos-1] == ' ' {
				pos--
			}
			for pos > 0 && m.input[pos-1] != ' ' {
				pos--
			}
			m.input = m.input[:pos] + m.input[m.cursorPos:]
			m.cursorPos = pos
		}
		return m, nil

	case tea.KeyCtrlA:
		// Move to beginning of line
		m.cursorPos = 0
		return m, nil

	case tea.KeyCtrlE:
		// Move to end of line
		m.cursorPos = len(m.input)
		return m, nil

	case tea.KeyEnter:
		if m.input != "" && !m.running {
			task := m.input
			m.input = ""
			m.cursorPos = 0
			m.running = true
			m.currentStep = 0 // Reset step tracking for new task

			// Add to history
			m.inputHistory = append(m.inputHistory, task)
			m.historyIndex = -1

			// Add user message
			m.messages = append(m.messages, NewUserMessage(task))

			return m, tea.Batch(m.updateViewport(), m.runAgent(task), m.listenForEvents())
		}
		return m, nil

	case tea.KeyBackspace:
		if m.cursorPos > 0 {
			m.input = m.input[:m.cursorPos-1] + m.input[m.cursorPos:]
			m.cursorPos--
		}
		return m, nil

	case tea.KeyUp:
		// Navigate history when not running
		if !m.running && len(m.inputHistory) > 0 {
			if m.historyIndex < len(m.inputHistory)-1 {
				m.historyIndex++
				m.input = m.inputHistory[len(m.inputHistory)-1-m.historyIndex]
				m.cursorPos = len(m.input)
			}
		}
		return m, nil

	case tea.KeyDown:
		// Navigate history when not running
		if !m.running && m.historyIndex >= 0 {
			if m.historyIndex > 0 {
				m.historyIndex--
				m.input = m.inputHistory[len(m.inputHistory)-1-m.historyIndex]
				m.cursorPos = len(m.input)
			} else {
				m.historyIndex = -1
				m.input = ""
				m.cursorPos = 0
			}
		}
		return m, nil

	case tea.KeyLeft:
		if m.cursorPos > 0 {
			m.cursorPos--
		}
		return m, nil

	case tea.KeyRight:
		if m.cursorPos < len(m.input) {
			m.cursorPos++
		}
		return m, nil

	case tea.KeyRunes:
		// Insert at cursor position
		runes := msg.String()
		m.input = m.input[:m.cursorPos] + runes + m.input[m.cursorPos:]
		m.cursorPos += len(runes)
		m.historyIndex = -1
		return m, nil

	case tea.KeySpace:
		// Insert space at cursor position
		m.input = m.input[:m.cursorPos] + " " + m.input[m.cursorPos:]
		m.cursorPos++
		m.historyIndex = -1
		return m, nil
	}

	return m, nil
}

// handleAgentEvent processes events from the agent
func (m *Model) handleAgentEvent(event agent.AgentEvent) (tea.Model, tea.Cmd) {
	switch e := event.(type) {
	case agent.ThinkingEvent:
		// Create new message if this is a new step or no assistant message exists
		needNewMsg := len(m.messages) == 0 || !m.messages[len(m.messages)-1].IsAssistant()

		// If step changed, create new message for this step
		if !needNewMsg && m.currentStep != e.Step {
			needNewMsg = true
			m.currentStep = e.Step
		}

		if needNewMsg {
			m.messages = append(m.messages, NewAssistantMessage(""))
			m.currentStep = e.Step
		}

		// Update the last assistant message with formatted thought content
		// Don't print yet - wait for step completion
		lastMsg := &m.messages[len(m.messages)-1]
		lastMsg.Content = formatThinkingContent(e.Thought)

	case agent.ToolStartEvent:
		// Append tool info to the last assistant message
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].IsAssistant() {
			lastMsg := &m.messages[len(m.messages)-1]
			toolInfo := fmt.Sprintf("\n\n🔧 **Running:** `%s`", e.Action)
			if e.Input != nil && len(e.Input) > 0 {
				toolInfo += fmt.Sprintf(" → %s", formatJSON(e.Input))
			}
			lastMsg.Content += toolInfo
		}

	case agent.ToolCompleteEvent:
		// Append tool result to the last assistant message
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].IsAssistant() {
			lastMsg := &m.messages[len(m.messages)-1]
			if e.Error != "" {
				lastMsg.Content += fmt.Sprintf("\n❌ **Error:** %s", e.Error)
			} else {
				output := e.Output
				if len(output) > 500 {
					output = output[:500] + "... (truncated)"
				}
				lastMsg.Content += fmt.Sprintf("\n✓ **Result:**\n```\n%s\n```", output)
			}
		}

	case agent.StepCompleteEvent:
		// Handle different step completion states
		if e.Finished {
			m.running = false
			// Result is already shown in the message content via formatThinkingContent
			// Only add result if not already present
			if e.Result != "" && len(m.messages) > 0 && m.messages[len(m.messages)-1].IsAssistant() {
				lastMsg := &m.messages[len(m.messages)-1]
				// Check if result already shown (contains ✅ Done or similar)
				if !strings.Contains(lastMsg.Content, "✅") && !strings.Contains(lastMsg.Content, e.Result) {
					lastMsg.Content += fmt.Sprintf("\n\n**Result:** %s", e.Result)
				}
			}
			// Print and stop - task is complete
			return m, m.updateViewport()
		} else if e.Interrupted {
			m.running = false
			// Add interrupted notice
			if len(m.messages) > 0 && m.messages[len(m.messages)-1].IsAssistant() {
				lastMsg := &m.messages[len(m.messages)-1]
				lastMsg.Content += fmt.Sprintf("\n\n*Interrupted: %s*", e.Result)
			}
			// Print and stop - task was interrupted
			return m, m.updateViewport()
		} else if e.Step < 0 {
			// Error case
			m.running = false
			if e.Result != "" && !strings.Contains(e.Result, "Task completed") {
				m.messages = append(m.messages, NewAssistantMessage(fmt.Sprintf("**Error:** %s", e.Result)))
			}
			// Print and stop - there was an error
			return m, m.updateViewport()
		}
		// Intermediate step - print current state and continue listening
		return m, tea.Batch(m.updateViewport(), m.listenForEvents())
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

// thinkingResponse represents the parsed LLM response structure
type thinkingResponse struct {
	Thought     string                 `json:"thought"`
	Action      string                 `json:"action,omitempty"`
	ActionInput map[string]interface{} `json:"action_input,omitempty"`
	Response    string                 `json:"response,omitempty"`
	Finish      bool                   `json:"finish,omitempty"`
	Result      string                 `json:"result,omitempty"`
}

// formatThinkingContent parses JSON content and formats it nicely for display
func formatThinkingContent(content string) string {
	content = strings.TrimSpace(content)

	// Try to parse as JSON
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd < jsonStart {
		// Not JSON, return as-is
		return content
	}

	jsonStr := content[jsonStart : jsonEnd+1]
	var resp thinkingResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		// Parse failed, return raw content
		return content
	}

	// Build formatted output
	var result strings.Builder

	// Show thought with a nice prefix
	if resp.Thought != "" {
		result.WriteString("💭 *Thinking:* ")
		result.WriteString(resp.Thought)
		result.WriteString("\n")
	}

	// Show response if present (for conversational replies)
	if resp.Response != "" {
		result.WriteString("\n")
		result.WriteString(resp.Response)
	}

	// Show action if it's a real tool (not "none")
	if resp.Action != "" && resp.Action != "none" {
		result.WriteString(fmt.Sprintf("\n\n🔧 *Action:* `%s`", resp.Action))
		if len(resp.ActionInput) > 0 {
			result.WriteString(fmt.Sprintf(" (%s)", formatJSON(resp.ActionInput)))
		}
	}

	// Show finish/result only if no response was shown (tool-based completion)
	// For conversational replies, response already contains the answer
	if resp.Finish && resp.Result != "" && resp.Response == "" {
		result.WriteString(fmt.Sprintf("\n\n✅ *Done:* %s", resp.Result))
	}

	return result.String()
}

// printMsg is a message that triggers printing to terminal
type printMsg struct {
	content string
}

// updateViewport returns a command to print new messages for shell-style output
func (m *Model) updateViewport() tea.Cmd {
	// Collect new messages to print
	var lines []string
	for i := m.printedMessages; i < len(m.messages); i++ {
		msg := m.messages[i]
		lines = append(lines, m.renderMessage(msg), "")
	}
	m.printedMessages = len(m.messages)

	if len(lines) == 0 {
		return nil
	}

	// Return a command that prints the messages
	return tea.Println(strings.Join(lines, "\n"))
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

		// Don't send duplicate StepCompleteEvent - agent already sends one
		// Only send error event if there was an actual error
		if err != nil {
			// Close stop channel to signal event listening to stop
			if m.stopListen != nil {
				close(m.stopListen)
			}
			return agent.StepCompleteEvent{
				Step:     -1,
				Finished: false,
				Result:   fmt.Sprintf("Error: %v", err),
			}
		}
		// Return nil instead of duplicate event - agent's event will be processed
		// Note: We don't close stopListen here because agent's event still needs to be processed
		return nil
	}
}

// initRenderer creates a glamour renderer asynchronously
func (m *Model) initRenderer(width int) tea.Cmd {
	return func() tea.Msg {
		// Use dark style instead of auto-detect to avoid OSC escape sequences
		renderer, err := glamour.NewTermRenderer(
			glamour.WithStylePath("dark"),
			glamour.WithWordWrap(width),
		)
		return rendererMsg{renderer: renderer, err: err}
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
		case event, ok := <-eventCh:
			if !ok {
				// Channel closed
				return nil
			}
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

// View renders the TUI (shell-style: only input line)
func (m *Model) View() string {
	if !m.ready {
		return ""
	}

	var b strings.Builder

	// Input area
	inputLine := m.renderInput()
	b.WriteString(inputLine)

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060"))
	b.WriteString(helpStyle.Render(m.helpText()))

	return b.String()
}

// helpText returns context-sensitive help
func (m *Model) renderMessage(msg Message) string {
	var b strings.Builder

	// Timestamp style
	tsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060"))

	// User style
	userStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4A90D9")).
		Bold(true)

	// Assistant style
	assistantStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D97757")).
		Bold(true)

	if msg.IsUser() {
		// User message: [time] You: content
		b.WriteString(tsStyle.Render(msg.FormatTimestamp()))
		b.WriteString(" ")
		b.WriteString(userStyle.Render("You:"))
		b.WriteString(" ")
		b.WriteString(msg.Content)
	} else {
		// Assistant message: [time] AutoCode: content
		b.WriteString(tsStyle.Render(msg.FormatTimestamp()))
		b.WriteString(" ")
		b.WriteString(assistantStyle.Render("AutoCode:"))

		// Render markdown content using glamour if available
		if m.renderer != nil {
			rendered, err := m.renderer.Render(msg.Content)
			if err == nil {
				b.WriteString(rendered)
			} else {
				b.WriteString("\n")
				b.WriteString(msg.Content)
			}
		} else {
			b.WriteString("\n")
			b.WriteString(msg.Content)
		}
	}

	return b.String()
}

// renderInput renders the input prompt
func (m *Model) renderInput() string {
	var b strings.Builder

	// Dynamic styles for input area
	inputPromptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D97757")).
		Bold(true)
	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E0E0E0"))
	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D97757"))
	runningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Italic(true)
	providerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060"))

	// Show provider info in prompt
	providerInfo := ""
	if m.providerName != "" && m.modelName != "" {
		providerInfo = providerStyle.Render(fmt.Sprintf("[%s:%s] ", m.providerName, m.modelName))
	}

	// Input prompt
	prompt := inputPromptStyle.Render("> ")

	if m.running {
		// Show status when running
		b.WriteString(providerInfo)
		b.WriteString(prompt)
		b.WriteString(runningStyle.Render("Processing... (Esc to cancel)"))
	} else {
		b.WriteString(providerInfo)
		b.WriteString(prompt)
		// Render input with cursor at correct position
		beforeCursor := inputStyle.Render(m.input[:m.cursorPos])
		cursor := cursorStyle.Render("▋")
		afterCursor := inputStyle.Render(m.input[m.cursorPos:])
		b.WriteString(beforeCursor)
		b.WriteString(cursor)
		b.WriteString(afterCursor)
	}

	b.WriteString("\n")

	return b.String()
}

// helpText returns context-sensitive help
func (m *Model) helpText() string {
	if m.running {
		return "esc: cancel • ctrl+d: quit"
	}
	return "enter: submit • ↑/↓: history • ctrl+a/e: home/end • ctrl+u: clear • ctrl+d: quit"
}
