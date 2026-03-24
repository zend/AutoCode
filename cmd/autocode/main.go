package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the application state
type Model struct {
	ready   bool
	width   int
	height  int
	message string
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1).
			MarginBottom(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Margin(1, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
)

func initialModel() Model {
	return Model{
		message: "AutoCode - ReAct Agent Framework",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			m.message = "Ready to code!"
		}
	}

	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	title := titleStyle.Render("🦞 AutoCode")
	content := boxStyle.Render(m.message)
	help := helpStyle.Render("Press q to quit • Enter to start")

	return fmt.Sprintf("%s\n%s\n%s", title, content, help)
}

func main() {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
