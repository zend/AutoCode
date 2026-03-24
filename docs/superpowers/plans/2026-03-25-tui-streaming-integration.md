# TUI Streaming Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Connect the TUI to the ReAct Agent with real-time streaming LLM responses displayed in an expandable tree view, supporting interrupt capability.

**Architecture:** Event-driven async architecture where Agent publishes events via channels, TUI consumes them in Bubble Tea's Update loop. SSE streaming client yields tokens via channel. Context cancellation propagates for interrupt support.

**Tech Stack:** Go, Bubble Tea (TUI), standard HTTP client with SSE parsing

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/llm/streaming.go` | NEW: SSE streaming client, token channel, StreamChat method |
| `internal/llm/streaming_test.go` | NEW: Tests for SSE parsing, cancellation |
| `internal/agent/events.go` | NEW: Event types (ThinkingEvent, ToolStartEvent, etc.) |
| `internal/agent/agent.go` | MODIFY: Refactor Run() to publish events, add context cancellation |
| `internal/agent/agent_test.go` | MODIFY: Update tests for event-driven architecture |
| `internal/tui/model.go` | NEW: TUI Model with tree state, event channel |
| `internal/tui/tree.go` | NEW: Tree node rendering, expand/collapse logic |
| `internal/tui/events.go` | NEW: Event handling from agent |
| `cmd/autocode/main.go` | MODIFY: Wire TUI to Agent, handle event loop |

---

## Task 1: SSE Streaming Client

**Files:**
- Create: `internal/llm/streaming.go`
- Create: `internal/llm/streaming_test.go`

- [ ] **Step 1: Write the failing test for SSE parsing**

```go
package llm

import (
    "strings"
    "testing"
)

func TestParseSSEStream(t *testing.T) {
    input := `data: {"token": "Hello"}
data: {"token": " World"}
data: [DONE]
`
    reader := strings.NewReader(input)
    tokens, err := ParseSSEStream(reader)

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    expected := []string{"Hello", " World"}
    if len(tokens) != len(expected) {
        t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
    }

    for i, exp := range expected {
        if tokens[i] != exp {
            t.Errorf("token %d: expected %q, got %q", i, exp, tokens[i])
        }
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/llm -run TestParseSSEStream -v`

Expected: FAIL with "ParseSSEStream not defined"

- [ ] **Step 3: Write minimal SSE parser**

```go
package llm

import (
    "bufio"
    "encoding/json"
    "io"
    "strings"
)

type SSEEvent struct {
    Token string `json:"token"`
}

func ParseSSEStream(reader io.Reader) ([]string, error) {
    var tokens []string
    scanner := bufio.NewScanner(reader)

    for scanner.Scan() {
        line := scanner.Text()
        if !strings.HasPrefix(line, "data: ") {
            continue
        }

        data := strings.TrimPrefix(line, "data: ")
        if data == "[DONE]" {
            break
        }

        var event SSEEvent
        if err := json.Unmarshal([]byte(data), &event); err != nil {
            continue // Skip malformed events
        }

        tokens = append(tokens, event.Token)
    }

    return tokens, scanner.Err()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/llm -run TestParseSSEStream -v`

Expected: PASS

- [ ] **Step 5: Write failing test for StreamChat method**

```go
func TestStreamChatCancellation(t *testing.T) {
    client := NewClient("http://localhost:8080", "test-key")

    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately

    req := ChatRequest{Model: "gpt-4", Messages: []Message{{Role: "user", Content: "hi"}}}
    ch, err := client.StreamChat(ctx, req)

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Should receive done or error quickly
    select {
    case event := <-ch:
        if !event.Done && event.Error == nil {
            t.Error("expected done or error after cancellation")
        }
    case <-time.After(100 * time.Millisecond):
        // Expected - channel should close
    }
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./internal/llm -run TestStreamChatCancellation -v`

Expected: FAIL with "StreamChat method not defined"

- [ ] **Step 7: Implement StreamChat method**

```go
package llm

import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
)

type StreamEvent struct {
    Token string
    Done  bool
    Error error
}

func (c *Client) StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
    req.Stream = true

    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("do request: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        resp.Body.Close()
        return nil, fmt.Errorf("api error: %s", resp.Status)
    }

    ch := make(chan StreamEvent)

    go func() {
        defer resp.Body.Close()
        defer close(ch)

        scanner := bufio.NewScanner(resp.Body)
        for scanner.Scan() {
            line := scanner.Text()
            if !strings.HasPrefix(line, "data: ") {
                continue
            }

            data := strings.TrimPrefix(line, "data: ")
            if data == "[DONE]" {
                ch <- StreamEvent{Done: true}
                return
            }

            // Parse choice delta for token
            var streamResp struct {
                Choices []struct {
                    Delta struct {
                        Content string `json:"content"`
                    } `json:"delta"`
                } `json:"choices"`
            }

            if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
                continue
            }

            if len(streamResp.Choices) > 0 {
                token := streamResp.Choices[0].Delta.Content
                if token != "" {
                    ch <- StreamEvent{Token: token}
                }
            }
        }

        if err := scanner.Err(); err != nil {
            ch <- StreamEvent{Error: err}
        }
    }()

    return ch, nil
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./internal/llm -run TestStreamChatCancellation -v`

Expected: PASS (or skip if no server - make test conditional)

- [ ] **Step 9: Commit**

```bash
git add internal/llm/streaming.go internal/llm/streaming_test.go
git commit -m "feat(llm): add SSE streaming client with StreamChat method

- Parse SSE format with data: prefix
- Yield tokens via channel
- Support context cancellation
- Add unit tests for SSE parsing

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Event Types

**Files:**
- Create: `internal/agent/events.go`

- [ ] **Step 1: Write event type definitions**

```go
package agent

import "time"

// AgentEvent is the interface for all events sent from Agent to TUI
type AgentEvent interface {
    Type() string
}

// ThinkingEvent is sent while agent is reasoning
type ThinkingEvent struct {
    Step      int
    Thought   string
    Streaming bool // true while receiving tokens
}

func (e ThinkingEvent) Type() string { return "thinking" }

// ToolStartEvent is sent when agent starts executing a tool
type ToolStartEvent struct {
    Step   int
    Action string
    Input  map[string]interface{}
}

func (e ToolStartEvent) Type() string { return "tool_start" }

// ToolCompleteEvent is sent when tool execution finishes
type ToolCompleteEvent struct {
    Step     int
    Action   string
    Output   string
    Error    string
    Duration time.Duration
}

func (e ToolCompleteEvent) Type() string { return "tool_complete" }

// StepCompleteEvent is sent when a step finishes (including interruption)
type StepCompleteEvent struct {
    Step        int
    Finished    bool
    Interrupted bool
    Result      string
}

func (e StepCompleteEvent) Type() string { return "step_complete" }

// TaskRequest is sent from TUI to Agent to start a task
type TaskRequest struct {
    Task string
}

// CancelRequest is sent from TUI to Agent to cancel current operation
type CancelRequest struct{}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/agent`

Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/agent/events.go
git commit -m "feat(agent): add event types for TUI-Agent communication

- ThinkingEvent for streaming thoughts
- ToolStartEvent and ToolCompleteEvent for tool lifecycle
- StepCompleteEvent with interruption support
- TaskRequest and CancelRequest for TUI-to-Agent

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Refactor Agent for Event-Driven Architecture

**Files:**
- Modify: `internal/agent/agent.go`
- Modify: `internal/agent/agent_test.go`

- [ ] **Step 1: Update agent struct to include event channel**

```go
type Agent struct {
    client    *llm.Client
    registry  *tools.ToolRegistry
    baseDir   string
    maxSteps  int
    model     string
    history   []Message
    eventCh   chan AgentEvent  // NEW: events to TUI
    cancelCh  chan struct{}     // NEW: cancellation signal
}
```

- [ ] **Step 2: Add EventChannel getter method**

```go
func (a *Agent) EventChannel() <-chan AgentEvent {
    return a.eventCh
}

func (a *Agent) Cancel() {
    close(a.cancelCh)
}
```

- [ ] **Step 3: Refactor Run method to publish events**

Replace the existing `Run` method with an event-driven version. Add these imports to `internal/agent/agent.go` if not already present:
```go
import (
    "fmt"
    "strings"
    "time"
    // ... other existing imports
)
```

Then replace `Run` and add helper methods:

```go
func (a *Agent) Run(ctx context.Context, task string) error {
    a.history = append(a.history, Message{
        Role:    "user",
        Content: task,
    })

    step := 0
    for step < a.maxSteps {
        select {
        case <-a.cancelCh:
            a.publishEvent(StepCompleteEvent{
                Step:        step,
                Interrupted: true,
            })
            return fmt.Errorf("agent interrupted by user")
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        // Build messages for LLM
        messages := make([]llm.Message, 0, len(a.history)+1)
        messages = append(messages, llm.Message{
            Role:    "system",
            Content: systemPrompt,
        })
        for _, h := range a.history {
            messages = append(messages, llm.Message{
                Role:    h.Role,
                Content: h.Content,
            })
        }

        // Stream thinking
        thought, err := a.streamThinking(ctx, messages, step)
        if err != nil {
            return err
        }

        // Parse response and execute
        agentResp, err := a.parseResponse(thought)
        if err != nil {
            a.history = append(a.history, Message{
                Role:    "user",
                Content: fmt.Sprintf("Error parsing response: %v. Please respond with valid JSON.", err),
            })
            continue
        }

        if agentResp.Finish {
            a.publishEvent(StepCompleteEvent{
                Step:     step,
                Finished: true,
                Result:   agentResp.Result,
            })
            return nil
        }

        // Execute tool
        if err := a.executeToolStep(ctx, agentResp, step); err != nil {
            return err
        }

        step++
    }

    a.publishEvent(StepCompleteEvent{
        Step: step,
        Result: "max steps exceeded",
    })
    return fmt.Errorf("max steps (%d) exceeded", a.maxSteps)
}

func (a *Agent) streamThinking(ctx context.Context, messages []llm.Message, step int) (string, error) {
    req := llm.ChatRequest{
        Model:    a.model,
        Messages: messages,
    }

    ch, err := a.client.StreamChat(ctx, req)
    if err != nil {
        return "", err
    }

    var thought strings.Builder
    ticker := time.NewTicker(50 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case event, ok := <-ch:
            if !ok {
                a.publishEvent(ThinkingEvent{
                    Step:      step,
                    Thought:   thought.String(),
                    Streaming: false,
                })
                return thought.String(), nil
            }

            if event.Error != nil {
                return "", event.Error
            }

            if event.Done {
                a.publishEvent(ThinkingEvent{
                    Step:      step,
                    Thought:   thought.String(),
                    Streaming: false,
                })
                return thought.String(), nil
            }

            thought.WriteString(event.Token)

        case <-ticker.C:
            if thought.Len() > 0 {
                a.publishEvent(ThinkingEvent{
                    Step:      step,
                    Thought:   thought.String(),
                    Streaming: true,
                })
            }

        case <-a.cancelCh:
            return "", fmt.Errorf("interrupted")
        }
    }
}

func (a *Agent) executeToolStep(ctx context.Context, resp *AgentResponse, step int) error {
    a.publishEvent(ToolStartEvent{
        Step:   step,
        Action: resp.Action,
        Input:  resp.ActionInput,
    })

    start := time.Now()
    observation, err := a.executeTool(ctx, resp)
    duration := time.Since(start)

    event := ToolCompleteEvent{
        Step:     step,
        Action:   resp.Action,
        Output:   observation,
        Duration: duration,
    }

    if err != nil {
        event.Error = err.Error()
        observation = fmt.Sprintf("Error: %v", err)
    }

    a.publishEvent(event)

    a.history = append(a.history, Message{
        Role:    "user",
        Content: fmt.Sprintf("Observation: %s", observation),
    })

    return nil
}

func (a *Agent) publishEvent(event AgentEvent) {
    select {
    case a.eventCh <- event:
    default:
        // Channel full, drop event (shouldn't happen with proper TUI)
    }
}
```

- [ ] **Step 4: Update constructor to initialize channels**

```go
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
        model:    "gpt-4",
        history:  make([]Message, 0),
        eventCh:  make(chan AgentEvent, 100),
        cancelCh: make(chan struct{}),
    }
}
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./internal/agent`

Expected: SUCCESS

- [ ] **Step 6: Commit**

```bash
git add internal/agent/agent.go
git commit -m "feat(agent): refactor for event-driven architecture

- Add event channel for TUI communication
- Implement streamThinking with 50ms throttling
- Publish ThinkingEvent, ToolStartEvent, ToolCompleteEvent
- Add Cancel() method for interruption
- Remove synchronous Run return value (now event-based)

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Tree Data Structure

**Files:**
- Create: `internal/tui/tree.go`

- [ ] **Step 1: Write tree node definitions**

```go
package tui

import (
    "fmt"
    "strings"
    "time"
)

type NodeState int

const (
    StateThinking    NodeState = 0
    StateExecuting   NodeState = 1
    StateComplete    NodeState = 2
    StateError       NodeState = 3
    StateInterrupted NodeState = 4
)

type TreeNode struct {
    Step      int
    Thought   string
    Action    string
    Input     string
    Output    string
    Error     string
    Duration  time.Duration
    Expanded  bool
    State     NodeState
}

func (n *TreeNode) Title() string {
    switch n.State {
    case StateThinking:
        return fmt.Sprintf("Step %d: Thinking...", n.Step)
    case StateExecuting:
        return fmt.Sprintf("Step %d: %s", n.Step, n.Action)
    case StateComplete:
        return fmt.Sprintf("Step %d: Complete", n.Step)
    case StateError:
        return fmt.Sprintf("Step %d: Error", n.Step)
    case StateInterrupted:
        return fmt.Sprintf("Step %d: Interrupted", n.Step)
    default:
        return fmt.Sprintf("Step %d", n.Step)
    }
}

func (n *TreeNode) Preview() string {
    lines := strings.Split(n.Thought, "\n")
    if len(lines) > 0 && lines[0] != "" {
        preview := lines[0]
        if len(preview) > 60 {
            preview = preview[:57] + "..."
        }
        return preview
    }
    return "..."
}

func (n *TreeNode) FullContent() string {
    var b strings.Builder

    b.WriteString(fmt.Sprintf("Thought:\n%s\n\n", n.Thought))

    if n.Action != "" {
        b.WriteString(fmt.Sprintf("Action: %s\n", n.Action))
        if n.Input != "" {
            b.WriteString(fmt.Sprintf("Input: %s\n", n.Input))
        }
    }

    if n.Output != "" {
        b.WriteString(fmt.Sprintf("\nOutput:\n%s\n", n.Output))
    }

    if n.Error != "" {
        b.WriteString(fmt.Sprintf("\nError: %s\n", n.Error))
    }

    if n.Duration > 0 {
        b.WriteString(fmt.Sprintf("\nDuration: %v\n", n.Duration))
    }

    return b.String()
}

type Tree struct {
    nodes []*TreeNode
}

func NewTree() *Tree {
    return &Tree{
        nodes: make([]*TreeNode, 0),
    }
}

func (t *Tree) AddNode(node *TreeNode) {
    t.nodes = append(t.nodes, node)
}

func (t *Tree) GetNode(step int) *TreeNode {
    for _, n := range t.nodes {
        if n.Step == step {
            return n
        }
    }
    return nil
}

func (t *Tree) LastNode() *TreeNode {
    if len(t.nodes) == 0 {
        return nil
    }
    return t.nodes[len(t.nodes)-1]
}

func (t *Tree) Nodes() []*TreeNode {
    return t.nodes
}

func (t *Tree) ToggleExpand(step int) {
    node := t.GetNode(step)
    if node != nil {
        node.Expanded = !node.Expanded
    }
}
```

- [ ] **Step 2: Write test for tree**

```go
package tui

import (
    "strings"
    "testing"
)

func TestTreeNodeTitle(t *testing.T) {
    node := &TreeNode{Step: 1, State: StateThinking}
    if !strings.Contains(node.Title(), "Step 1") {
        t.Error("title should contain step number")
    }

    node.State = StateExecuting
    node.Action = "read"
    if !strings.Contains(node.Title(), "read") {
        t.Error("title should contain action")
    }
}

func TestTreeAddNode(t *testing.T) {
    tree := NewTree()
    node := &TreeNode{Step: 0}
    tree.AddNode(node)

    if len(tree.Nodes()) != 1 {
        t.Error("expected 1 node")
    }

    if tree.GetNode(0) != node {
        t.Error("GetNode returned wrong node")
    }
}

func TestTreeToggleExpand(t *testing.T) {
    tree := NewTree()
    node := &TreeNode{Step: 0, Expanded: false}
    tree.AddNode(node)

    tree.ToggleExpand(0)
    if !node.Expanded {
        t.Error("expected expanded")
    }

    tree.ToggleExpand(0)
    if node.Expanded {
        t.Error("expected collapsed")
    }
}
```

- [ ] **Step 3: Run tests to verify**

Run: `go test ./internal/tui -v`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/tui/tree.go
git commit -m "feat(tui): add tree data structure

- TreeNode with State, Thought, Action, Output, Error
- Tree container with AddNode, GetNode, ToggleExpand
- Node title and preview generation
- Full content rendering for expanded view

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: TUI Model with Event Handling

**Files:**
- Create: `internal/tui/model.go`

- [ ] **Step 1: Write TUI Model implementation**

```go
package tui

import (
    "context"
    "fmt"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "github.com/zend/AutoCode/internal/agent"
)

type Model struct {
    tree        *Tree
    agent       *agent.Agent
    width       int
    height      int
    ready       bool
    input       string
    running     bool
    finished    bool
    result      string
    cursorStep  int  // for keyboard navigation
}

func NewModel(agent *agent.Agent) *Model {
    return &Model{
        tree:   NewTree(),
        agent:  agent,
    }
}

func (m Model) Init() tea.Cmd {
    return listenForEvents(m.agent.EventChannel())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.ready = true
        return m, nil

    case tea.KeyMsg:
        return m.handleKey(msg)

    case agent.AgentEvent:
        return m.handleAgentEvent(msg)
    }

    return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+c", "q":
        return m, tea.Quit

    case "esc":
        if m.running {
            m.agent.Cancel()
            m.running = false
        }
        return m, nil

    case "enter":
        if !m.running && m.input != "" {
            m.startTask()
            return m, m.runAgent()
        }

    case "backspace":
        if len(m.input) > 0 {
            m.input = m.input[:len(m.input)-1]
        }

    case "up":
        if m.cursorStep > 0 {
            m.cursorStep--
        }

    case "down":
        if m.cursorStep < len(m.tree.Nodes())-1 {
            m.cursorStep++
        }

    case " ", "e":
        m.tree.ToggleExpand(m.cursorStep)

    default:
        if !m.running {
            m.input += msg.String()
        }
    }

    return m, nil
}

func (m *Model) startTask() {
    m.running = true
    m.finished = false
    m.result = ""
    m.tree = NewTree()
}

func (m Model) runAgent() tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        err := m.agent.Run(ctx, m.input)
        if err != nil {
            // Error is handled via events
        }
        return agentRunComplete{err: err}
    }
}

func (m Model) handleAgentEvent(evt agent.AgentEvent) (tea.Model, tea.Cmd) {
    switch e := evt.(type) {
    case agent.ThinkingEvent:
        node := m.tree.GetNode(e.Step)
        if node == nil {
            node = &TreeNode{Step: e.Step, State: StateThinking}
            m.tree.AddNode(node)
        }
        node.Thought = e.Thought
        if !e.Streaming {
            // Thought complete, waiting for action
        }

    case agent.ToolStartEvent:
        node := m.tree.GetNode(e.Step)
        if node != nil {
            node.State = StateExecuting
            node.Action = e.Action
            node.Input = fmt.Sprintf("%v", e.Input)
        }

    case agent.ToolCompleteEvent:
        node := m.tree.GetNode(e.Step)
        if node != nil {
            if e.Error != "" {
                node.State = StateError
                node.Error = e.Error
            } else {
                node.State = StateComplete
            }
            node.Output = e.Output
            node.Duration = e.Duration
        }

    case agent.StepCompleteEvent:
        if e.Finished {
            m.finished = true
            m.result = e.Result
            m.running = false
        }
        if e.Interrupted {
            m.running = false
            node := m.tree.LastNode()
            if node != nil {
                node.State = StateInterrupted
            }
        }
    }

    return m, listenForEvents(m.agent.EventChannel())
}

type agentRunComplete struct {
    err error
}

func listenForEvents(ch <-chan agent.AgentEvent) tea.Cmd {
    return func() tea.Msg {
        evt := <-ch
        return evt
    }
}

func (m Model) View() string {
    if !m.ready {
        return "\n  Initializing..."
    }

    var sections []string

    // Title
    sections = append(sections, titleStyle.Render(" AutoCode"))

    // Tree
    sections = append(sections, m.renderTree())

    // Input or Result
    if m.finished {
        sections = append(sections, resultStyle.Render("Result: "+m.result))
    } else if !m.running {
        sections = append(sections, inputStyle.Render("> "+m.input+"_"))
    }

    // Help
    sections = append(sections, helpStyle.Render(m.helpText()))

    return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderTree() string {
    var lines []string

    for _, node := range m.tree.Nodes() {
        line := node.Title()

        if node.Step == m.cursorStep {
            line = "> " + line
        } else {
            line = "  " + line
        }

        if node.Expanded {
            line = "▼ " + line[2:]
            lines = append(lines, line)
            content := node.FullContent()
            lines = append(lines, indent(content))
        } else {
            line = "▶ " + line[2:]
            lines = append(lines, line)
            lines = append(lines, "    "+previewStyle.Render(node.Preview()))
        }
    }

    if len(lines) == 0 && m.running {
        lines = append(lines, "Thinking...")
    }

    return treeStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) helpText() string {
    if m.running {
        return "Press Esc to interrupt • Ctrl+C to quit"
    }
    return "Type task and press Enter • ↑/↓ navigate • Space/e expand • q quit"
}

func indent(s string) string {
    lines := strings.Split(s, "\n")
    for i, line := range lines {
        lines[i] = "    " + line
    }
    return strings.Join(lines, "\n")
}

// Styles
var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("15")).
        Background(lipgloss.Color("62")).
        Padding(0, 1).
        MarginBottom(1)

    treeStyle = lipgloss.NewStyle().
        MarginLeft(1)

    previewStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("241")).
        Italic(true)

    inputStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("15"))

    resultStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("42")).
        Padding(1)

    helpStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("241")).
        MarginTop(1)
)
```

- [ ] **Step 2: Add missing import**

Add to imports:
```go
"strings"
```

- [ ] **Step 3: Verify compilation**

First verify agent package compiles:
Run: `go build ./internal/agent`

Then verify tui package:
Run: `go build ./internal/tui`

Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add internal/tui/model.go
git commit -m "feat(tui): add TUI Model with event handling

- tea.Model implementation with Init/Update/View
- Handle AgentEvents and update tree
- Keyboard navigation (↑/↓, Space/e expand)
- Input handling for new tasks
- Esc key to interrupt running agent
- Lipgloss styles for rendering

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Wire TUI to Agent in Main

**Files:**
- Modify: `cmd/autocode/main.go`

- [ ] **Step 1: Replace main.go content**

```go
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
    // Get API credentials from environment
    baseURL := os.Getenv("OPENAI_BASE_URL")
    if baseURL == "" {
        baseURL = "https://api.openai.com/v1"
    }
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        fmt.Fprintln(os.Stderr, "Error: OPENAI_API_KEY not set")
        os.Exit(1)
    }

    // Create LLM client
    client := llm.NewClient(baseURL, apiKey)

    // Create Agent
    agentInstance := agent.New(client, ".")

    // Create TUI Model
    model := tui.NewModel(agentInstance)

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
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/autocode`

Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add cmd/autocode/main.go
git commit -m "feat(main): wire TUI to Agent

- Create LLM client with env-based config
- Create Agent instance
- Create TUI Model and run with tea.NewProgram
- Exit with error if OPENAI_API_KEY not set

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 7: Add Tests for Agent Event Handling

**Files:**
- Modify: `internal/agent/agent_test.go`

- [ ] **Step 1: Add test for event publishing**

```go
import (
    "context"
    "testing"
    "time"

    "github.com/zend/AutoCode/internal/llm"
)

func TestAgentPublishesEvents(t *testing.T) {
    // Create mock LLM client
    client := &mockStreamingClient{}
    agent := New(client, ".")

    // Run agent in background
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    go func() {
        agent.Run(ctx, "test task")
    }()

    // Collect events
    var events []AgentEvent
    timeout := time.After(50 * time.Millisecond)

    done := false
    for !done {
        select {
        case evt := <-agent.EventChannel():
            events = append(events, evt)
            if _, ok := evt.(StepCompleteEvent); ok {
                done = true
            }
        case <-timeout:
            done = true
        }
    }

    if len(events) == 0 {
        t.Error("expected events to be published")
    }
}
```

- [ ] **Step 2: Add mock streaming client**

The mock implements the interface that Agent expects from the LLM client. The Agent uses `StreamChat()` and `Chat()` methods:

```go
// Mock client for testing - implements the interface Agent needs
type mockStreamingClient struct {
    tokens []string
}

func (m *mockStreamingClient) StreamChat(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamEvent, error) {
    ch := make(chan llm.StreamEvent)
    go func() {
        defer close(ch)
        for _, token := range m.tokens {
            ch <- llm.StreamEvent{Token: token}
        }
        ch <- llm.StreamEvent{Done: true}
    }()
    return ch, nil
}

func (m *mockStreamingClient) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
    return &llm.ChatResponse{}, nil
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/agent -v`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/agent/agent_test.go
git commit -m "test(agent): add event publishing test

- Test that agent publishes events during execution
- Mock streaming client for testing

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 8: Documentation and Final Verification

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update README with new features**

```markdown
## Features

- ReAct Agent with streaming LLM responses
- TUI with expandable tree view showing agent execution
- Real-time token streaming in thought nodes
- Interrupt agent with Esc key
- Tools: Read/Write/Grep/Shell

## Configuration

Set environment variables:
- `OPENAI_API_KEY` - Your API key (required)
- `OPENAI_BASE_URL` - API endpoint (optional, defaults to OpenAI)

## Usage

```
↑/↓     Navigate steps
Space/e Expand/collapse step
Esc     Interrupt running agent
q       Quit
```
```

- [ ] **Step 2: Build and verify**

Run: `go build -o bin/autocode ./cmd/autocode`

Expected: SUCCESS

- [ ] **Step 3: Run all tests**

Run: `go test ./...`

Expected: PASS

- [ ] **Step 4: Final commit**

```bash
git add README.md
git commit -m "docs: update README with TUI streaming features

- Document streaming LLM responses
- Document tree view and controls
- Add configuration section
- Add usage instructions

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Success Criteria

- [ ] TUI displays agent execution as expandable tree
- [ ] LLM tokens stream live within thinking nodes
- [ ] User can interrupt agent with Esc key
- [ ] UI remains responsive during long operations
- [ ] Errors are displayed clearly in tree
- [ ] Tree supports expand/collapse per node
- [ ] All tests pass
- [ ] Application builds successfully
