package context

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestGenerateSessionId(t *testing.T) {
	id := generateSessionId()

	// Check format: YYYY-MM-DD-HHMMSS-RAND
	pattern := `^\d{4}-\d{2}-\d{2}-\d{6}-[a-f0-9]{6}$`
	matched, err := regexp.MatchString(pattern, id)
	if err != nil {
		t.Fatalf("Regex error: %v", err)
	}

	if !matched {
		t.Errorf("Session ID '%s' does not match expected format", id)
	}
}

func TestNewSession(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir, true)

	session := sm.NewSession("/project/test")

	if session.ID == "" {
		t.Error("Session ID is empty")
	}

	if session.ProjectDir != "/project/test" {
		t.Errorf("Expected project dir '/project/test', got '%s'", session.ProjectDir)
	}

	if session.StartTime.IsZero() {
		t.Error("Session start time is zero")
	}

	if len(session.Messages) != 0 {
		t.Errorf("Expected empty messages, got %d", len(session.Messages))
	}
}

func TestNewSessionDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir, false) // disabled

	session := sm.NewSession("/project/test")

	// Session should still be created in memory
	if session == nil {
		t.Error("Expected non-nil session even when disabled")
	}
}

func TestSessionManagerAddMessage(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir, true)

	sm.NewSession("/project/test")

	// Add user message
	sm.AddMessage("user", "Hello")

	session := sm.GetCurrent()
	if len(session.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(session.Messages))
	}

	if session.Messages[0].Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", session.Messages[0].Role)
	}

	if session.Messages[0].Content != "Hello" {
		t.Errorf("Expected content 'Hello', got '%s'", session.Messages[0].Content)
	}

	// Add assistant message
	sm.AddMessage("assistant", "Hi there!")

	if len(session.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(session.Messages))
	}
}

func TestIsGreeting(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
	}{
		{"hi", true},
		{"Hi!", true},
		{"Hello", true},
		{"你好", true},
		{"您好", true},
		{"Hey!", true},
		{"What is the task?", false},
		{"Can you fix this bug?", false},
		{"Implement a new feature", false},
		{"This is a longer message that is definitely not a greeting", false},
		{"hi, can you help me fix this?", false}, // Contains task keyword "fix"
	}

	for _, test := range tests {
		result := isGreeting(test.content)
		if result != test.expected {
			t.Errorf("isGreeting(%q) = %v, expected %v", test.content, result, test.expected)
		}
	}
}

func TestSessionManagerSaveAndLoadSession(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir, true)

	// Create and save session
	session := sm.NewSession("/project/test")
	sm.AddMessage("user", "Task 1")
	sm.AddMessage("assistant", "Working on it")

	summary := SessionSummary{
		TotalSteps: 1,
		ToolsUsed:  []string{"read", "write"},
		Result:     "Success",
	}

	if err := sm.CloseSession(summary); err != nil {
		t.Fatalf("CloseSession failed: %v", err)
	}

	// Load the session
	loaded, err := sm.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}

	if loaded.ID != session.ID {
		t.Errorf("Expected session ID '%s', got '%s'", session.ID, loaded.ID)
	}

	if len(loaded.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(loaded.Messages))
	}

	if loaded.Summary.Result != "Success" {
		t.Errorf("Expected result 'Success', got '%s'", loaded.Summary.Result)
	}
}

func TestSessionManagerListSessions(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir, true)

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		_ = sm.NewSession("/project/test")
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
		sm.AddMessage("user", "Task")
		sm.CloseSession(SessionSummary{Result: "Done"})
	}

	// List sessions
	metas, err := sm.ListSessions(10)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(metas) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(metas))
	}

	// Verify sorted descending (most recent first)
	if len(metas) >= 2 {
		if metas[0].StartTime.Before(metas[1].StartTime) {
			t.Error("Expected sessions sorted by time descending")
		}
	}
}

func TestSessionManagerListSessionsWithLimit(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir, true)

	// Create 5 sessions
	for i := 0; i < 5; i++ {
		_ = sm.NewSession("/project/test")
		time.Sleep(1 * time.Millisecond)
		sm.AddMessage("user", "Task")
		sm.CloseSession(SessionSummary{})
	}

	// List with limit
	metas, err := sm.ListSessions(2)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(metas) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(metas))
	}
}

func TestSessionManagerResumeSession(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir, true)

	// Create and save session
	session := sm.NewSession("/project/test")
	sm.AddMessage("user", "Original task")
	sm.CloseSession(SessionSummary{Result: "In progress"})

	// Resume in a new manager
	sm2 := NewSessionManager(tmpDir, true)
	if err := sm2.ResumeSession(session.ID); err != nil {
		t.Fatalf("ResumeSession failed: %v", err)
	}

	// Verify resumed session
	resumed := sm2.GetCurrent()
	if resumed == nil {
		t.Fatal("Expected non-nil current session after resume")
	}

	if len(resumed.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(resumed.Messages))
	}

	if resumed.Messages[0].Content != "Original task" {
		t.Errorf("Expected message 'Original task', got '%s'", resumed.Messages[0].Content)
	}
}

func TestSessionManagerCleanOldSessions(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions")
	os.MkdirAll(sessionsDir, 0755)

	sm := NewSessionManager(tmpDir, true)

	// Create old session files manually
	oldSession := `{
		"sessionId": "2026-03-01-120000-abc123",
		"startTime": "2026-03-01T12:00:00Z",
		"projectDir": "/test",
		"messages": []
	}`
	os.WriteFile(filepath.Join(sessionsDir, "2026-03-01-120000-abc123.json"), []byte(oldSession), 0644)

	recentSession := `{
		"sessionId": "2026-04-03-120000-def456",
		"startTime": "2026-04-03T12:00:00Z",
		"projectDir": "/test",
		"messages": []
	}`
	os.WriteFile(filepath.Join(sessionsDir, "2026-04-03-120000-def456.json"), []byte(recentSession), 0644)

	// Clean sessions older than 7 days
	if err := sm.CleanOldSessions(7); err != nil {
		t.Fatalf("CleanOldSessions failed: %v", err)
	}

	// Check that old session is deleted
	entries, _ := os.ReadDir(sessionsDir)
	for _, e := range entries {
		if e.Name() == "2026-03-01-120000-abc123.json" {
			t.Error("Expected old session to be deleted")
		}
	}
}