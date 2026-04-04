package context

import (
	"strings"
	"testing"
)

func TestContextManagerEstimateTokens(t *testing.T) {
	cm := NewContextManager(100000, 10000, 20, 0)

	tests := []struct {
		text     string
		minToken int
		maxToken int
	}{
		{"", 0, 0},
		{"a", 0, 1},
		{"abcd", 1, 1},
		{"abcdefgh", 2, 2},
		{"This is a test message.", 5, 6},
	}

	for _, test := range tests {
		result := cm.EstimateTokens(test.text)
		if result < test.minToken || result > test.maxToken {
			t.Errorf("EstimateTokens(%q) = %d, expected between %d and %d",
				test.text, result, test.minToken, test.maxToken)
		}
	}
}

func TestContextManagerIsGreeting(t *testing.T) {
	cm := NewContextManager(100000, 10000, 20, 0)

	// Test greetings
	greetings := []string{"hi", "Hi!", "Hello", "你好", "您好", "Hey!"}
	for _, g := range greetings {
		if !cm.IsGreeting(g) {
			t.Errorf("Expected '%s' to be detected as greeting", g)
		}
	}

	// Test non-greetings
	nonGreetings := []string{
		"Can you fix this bug?",
		"Implement a new feature",
		"Read the file",
		"This is a longer message that is definitely not a greeting",
		"hi, can you help me?", // Contains task keyword "help"
	}
	for _, ng := range nonGreetings {
		if cm.IsGreeting(ng) {
			t.Errorf("Expected '%s' to NOT be detected as greeting", ng)
		}
	}
}

func TestContextManagerFilterGreetings(t *testing.T) {
	cm := NewContextManager(100000, 10000, 20, 0)

	messages := []MsgEntry{
		{Role: "user", Content: "hi"},                     // Greeting
		{Role: "assistant", Content: "Hello!"},            // Greeting
		{Role: "user", Content: "Fix this bug"},           // Task
		{Role: "assistant", Content: "I'll help with that"},
		{Role: "user", Content: "你好"},                    // Greeting
		{Role: "assistant", Content: "Working on it"},
	}

	filtered, count := cm.FilterGreetings(messages)

	if count != 3 {
		t.Errorf("Expected 3 greetings filtered, got %d", count)
	}

	if len(filtered) != 3 {
		t.Errorf("Expected 3 remaining messages, got %d", len(filtered))
	}

	// Verify the task message is kept
	found := false
	for _, m := range filtered {
		if m.Content == "Fix this bug" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected task message to be preserved")
	}
}

func TestContextManagerPreserveFirstMessage(t *testing.T) {
	cm := NewContextManager(100000, 10000, 20, 0)

	messages := []MsgEntry{
		{Role: "user", Content: "First message"},
		{Role: "assistant", Content: "Response"},
		{Role: "user", Content: "Second message"},
	}

	result := cm.PreserveFirstMessage(messages)

	if len(result) != len(messages) {
		t.Errorf("PreserveFirstMessage should not change message count")
	}

	if result[0].Content != "First message" {
		t.Error("Expected first message to be preserved")
	}
}

func TestContextManagerApplyWindow(t *testing.T) {
	// Test with window size 3
	cm := NewContextManager(100000, 10000, 3, 0)

	// Create more messages than window size
	messages := []MsgEntry{
		{Role: "user", Content: "First"},      // Should be preserved
		{Role: "assistant", Content: "R1"},
		{Role: "user", Content: "M2"},
		{Role: "assistant", Content: "R2"},
		{Role: "user", Content: "M3"},         // Recent
		{Role: "assistant", Content: "R3"},    // Recent
	}

	result := cm.ApplyWindow(messages)

	// Should have: First (preserved) + recent 2 (window-1 for first)
	if len(result) > 4 {
		t.Errorf("Expected at most 4 messages after windowing, got %d", len(result))
	}

	// First message should be preserved
	if result[0].Content != "First" {
		t.Error("Expected first message to be preserved")
	}
}

func TestContextManagerApplyWindowSmallInput(t *testing.T) {
	cm := NewContextManager(100000, 10000, 10, 0)

	// Less messages than window size
	messages := []MsgEntry{
		{Role: "user", Content: "M1"},
		{Role: "assistant", Content: "R1"},
	}

	result := cm.ApplyWindow(messages)

	if len(result) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(result))
	}
}

func TestContextManagerTrimHistory(t *testing.T) {
	cm := NewContextManager(10000, 1000, 10, 1000) // max 10000 tokens, 1000 static

	// Create messages
	messages := []MsgEntry{
		{Role: "user", Content: "hi"},                     // Greeting (filtered)
		{Role: "assistant", Content: "Hello!"},            // Greeting (filtered)
		{Role: "user", Content: "Fix the bug in main.go"}, // First valid task
		{Role: "assistant", Content: "I'll help you fix the bug."},
		{Role: "user", Content: "Check the file"},
	}

	result, stats := cm.TrimHistory(messages)

	// Check stats
	if stats.FilteredGreetings == 0 {
		t.Error("Expected some greetings to be filtered")
	}

	if !stats.PreservedFirst {
		t.Error("Expected first valid message to be preserved")
	}

	// Verify result format
	for _, m := range result {
		if m.Role != "user" && m.Role != "assistant" {
			t.Errorf("Unexpected role: %s", m.Role)
		}
		if m.Content == "" {
			t.Error("Empty content in result")
		}
	}
}

func TestContextManagerCompressToolOutput(t *testing.T) {
	cm := NewContextManager(100000, 100, 20, 0) // max 100 bytes

	// Small output - no compression
	smallOutput := "This is small"
	result := cm.CompressToolOutput(smallOutput)
	if result != smallOutput {
		t.Error("Small output should not be compressed")
	}

	// Large output - should be compressed
	largeOutput := strings.Repeat("a", 200)
	result = cm.CompressToolOutput(largeOutput)

	if len(result) >= len(largeOutput) {
		t.Error("Large output should be compressed")
	}

	if !strings.Contains(result, "截断") {
		t.Error("Expected truncation indicator in result")
	}
}

func TestContextManagerCompressToolOutputWithLimit(t *testing.T) {
	cm := NewContextManager(100000, 10000, 20, 0)

	largeOutput := strings.Repeat("x", 1000)
	result := cm.CompressToolOutputWithLimit(largeOutput, 100)

	if len(result) >= 1000 {
		t.Error("Output should be compressed to fit limit")
	}

	if !strings.Contains(result, "截断") {
		t.Error("Expected truncation indicator")
	}
}

func TestContextManagerTokenLimit(t *testing.T) {
	// Very restrictive token limit
	cm := NewContextManager(100, 1000, 10, 0) // Only 100 tokens max

	// Create messages that exceed the limit
	messages := []MsgEntry{
		{Role: "user", Content: strings.Repeat("test ", 50)},  // ~250 chars = ~62 tokens
		{Role: "assistant", Content: strings.Repeat("response ", 50)}, // ~400 chars = ~100 tokens
		{Role: "user", Content: strings.Repeat("more ", 50)},  // Would exceed limit
	}

	result, stats := cm.TrimHistory(messages)

	// Should be truncated due to token limit
	totalTokens := 0
	for _, m := range result {
		totalTokens += cm.EstimateTokens(m.Content)
	}

	if totalTokens > stats.AvailableTokens {
		t.Errorf("Total tokens %d exceeds available %d", totalTokens, stats.AvailableTokens)
	}
}

func TestContextManagerGetSetStaticTokens(t *testing.T) {
	cm := NewContextManager(100000, 10000, 20, 5000)

	if cm.GetStaticTokens() != 5000 {
		t.Errorf("Expected static tokens 5000, got %d", cm.GetStaticTokens())
	}

	cm.SetStaticTokens(8000)

	if cm.GetStaticTokens() != 8000 {
		t.Errorf("Expected static tokens 8000, got %d", cm.GetStaticTokens())
	}
}