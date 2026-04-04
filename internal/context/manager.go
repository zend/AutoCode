package context

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/zend/AutoCode/internal/llm"
)

// ContextStats holds statistics about context filtering
type ContextStats struct {
	TotalMessages   int
	FilteredGreetings int
	PreservedFirst  bool
	WindowApplied   int
	TotalTokens     int
	AvailableTokens int
}

// ContextManager manages conversation history filtering
type ContextManager struct {
	maxTokens     int
	maxToolOutput int
	windowSize    int
	staticTokens  int
}

// NewContextManager creates a new context manager
func NewContextManager(maxTokens, maxToolOutput, windowSize, staticTokens int) *ContextManager {
	return &ContextManager{
		maxTokens:     maxTokens,
		maxToolOutput: maxToolOutput,
		windowSize:    windowSize,
		staticTokens:  staticTokens,
	}
}

// EstimateTokens estimates token count from text
// Uses 4 chars ≈ 1 token approximation
func (cm *ContextManager) EstimateTokens(text string) int {
	return len(text) / 4
}

// IsGreeting checks if a message is a pure greeting
func (cm *ContextManager) IsGreeting(content string) bool {
	return isGreeting(content)
}

// FilterGreetings removes greeting messages from history
func (cm *ContextManager) FilterGreetings(messages []MsgEntry) ([]MsgEntry, int) {
	var filtered []MsgEntry
	greetingCount := 0

	for _, msg := range messages {
		if msg.IsGreeting || isGreeting(msg.Content) {
			greetingCount++
			continue
		}
		filtered = append(filtered, msg)
	}

	return filtered, greetingCount
}

// PreserveFirstMessage ensures the first non-greeting message is preserved
func (cm *ContextManager) PreserveFirstMessage(messages []MsgEntry) []MsgEntry {
	if len(messages) == 0 {
		return messages
	}

	// First message is already preserved if it exists
	// This function ensures we don't drop it during windowing
	return messages
}

// ApplyWindow keeps only the most recent N messages
func (cm *ContextManager) ApplyWindow(messages []MsgEntry) []MsgEntry {
	if len(messages) <= cm.windowSize {
		return messages
	}

	// Keep the first message (task definition) + recent messages
	if len(messages) == 0 {
		return messages
	}

	first := messages[0]
	recent := messages[len(messages)-cm.windowSize+1:]

	result := make([]MsgEntry, 0, len(recent)+1)
	result = append(result, first)
	result = append(result, recent...)

	return result
}

// TrimHistory applies all filtering steps to history
func (cm *ContextManager) TrimHistory(messages []MsgEntry) ([]llm.Message, ContextStats) {
	stats := ContextStats{
		TotalMessages: len(messages),
	}

	// Step 1: Filter greetings
	filtered, greetingsFiltered := cm.FilterGreetings(messages)
	stats.FilteredGreetings = greetingsFiltered

	// Step 2: Preserve first message (implicit in window application)
	filtered = cm.PreserveFirstMessage(filtered)
	stats.PreservedFirst = len(filtered) > 0

	// Step 3: Apply time window
	beforeWindow := len(filtered)
	filtered = cm.ApplyWindow(filtered)
	stats.WindowApplied = beforeWindow - len(filtered)

	// Step 4: Check token limit
	availableTokens := cm.maxTokens - cm.staticTokens
	stats.AvailableTokens = availableTokens

	// Convert to llm.Message and check tokens
	var result []llm.Message
	totalTokens := 0

	for _, msg := range filtered {
		tokens := cm.EstimateTokens(msg.Content)
		if totalTokens+tokens > availableTokens {
			// Would exceed limit, stop adding
			break
		}
		result = append(result, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
		totalTokens += tokens
	}

	stats.TotalTokens = totalTokens

	return result, stats
}

// CompressToolOutput truncates large tool outputs
func (cm *ContextManager) CompressToolOutput(output string) string {
	if len(output) <= cm.maxToolOutput {
		return output
	}

	// Keep first half and add truncation notice
	halfLen := cm.maxToolOutput / 2
	truncated := output[:halfLen]
	remaining := len(output) - halfLen

	return fmt.Sprintf("%s\n\n... [截断: 原长度 %d 字节，剩余 %d 字节已省略]\n",
		truncated, len(output), remaining)
}

// CompressToolOutputWithLimit truncates with a specific limit
func (cm *ContextManager) CompressToolOutputWithLimit(output string, limit int) string {
	if len(output) <= limit {
		return output
	}

	halfLen := limit / 2
	truncated := output[:halfLen]
	remaining := len(output) - halfLen

	return fmt.Sprintf("%s\n\n... [截断: 原长度 %d 字节，剩余 %d 字节已省略]\n",
		truncated, len(output), remaining)
}

// GetStaticTokens returns the tokens used by static context
func (cm *ContextManager) GetStaticTokens() int {
	return cm.staticTokens
}

// SetStaticTokens sets the tokens used by static context
func (cm *ContextManager) SetStaticTokens(tokens int) {
	cm.staticTokens = tokens
}

// greetingPattern matches common greeting patterns
var greetingPattern = regexp.MustCompile(`^(hi|hello|hey|hola|你好|嗨|您好|早上好|下午好|晚上好)[!.]*$`)

// taskKeywords indicates the message has actual task content
var taskKeywords = []string{
	"task", "fix", "code", "bug", "implement", "add", "create",
	"update", "delete", "read", "write", "file", "help",
	"error", "issue", "problem", "change", "refactor",
	"test", "build", "run", "install", "config",
	"问题", "任务", "修复", "实现", "创建", "更新", "删除",
	"文件", "代码", "错误", "测试", "运行",
}

// containsTaskKeyword checks if text contains task keywords
func containsTaskKeyword(text string) bool {
	lowerText := strings.ToLower(text)
	for _, kw := range taskKeywords {
		if strings.Contains(lowerText, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}