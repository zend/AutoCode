package context

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Session represents a conversation session
type Session struct {
	ID         string          `json:"sessionId"`
	StartTime  time.Time       `json:"startTime"`
	EndTime    time.Time       `json:"endTime,omitempty"`
	ProjectDir string          `json:"projectDir"`
	Messages   []MsgEntry      `json:"messages"`
	Summary    SessionSummary  `json:"summary,omitempty"`
}

// MsgEntry represents a message in a session
type MsgEntry struct {
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	Timestamp  time.Time `json:"timestamp"`
	IsGreeting bool      `json:"isGreeting,omitempty"`
}

// SessionSummary summarizes a session
type SessionSummary struct {
	TotalSteps int      `json:"totalSteps"`
	ToolsUsed  []string `json:"toolsUsed"`
	Result     string   `json:"result"`
}

// SessionMeta is metadata for listing sessions
type SessionMeta struct {
	ID         string    `json:"id"`
	StartTime  time.Time `json:"startTime"`
	ProjectDir string    `json:"projectDir"`
	Result     string    `json:"result"`
}

// SessionManager manages session persistence
type SessionManager struct {
	configDir string
	current   *Session
	enabled   bool
}

// NewSessionManager creates a new session manager
func NewSessionManager(configDir string, enabled bool) *SessionManager {
	return &SessionManager{
		configDir: configDir,
		enabled:   enabled,
	}
}

// generateSessionId creates a unique session ID
// Format: YYYY-MM-DD-HHMMSS-RAND
func generateSessionId() string {
	now := time.Now()
	timestamp := now.Format("2006-01-02-150405")

	// Generate 6 random hex characters
	b := make([]byte, 3)
	rand.Read(b)
	random := hex.EncodeToString(b)

	return fmt.Sprintf("%s-%s", timestamp, random)
}

// NewSession creates a new session
func (sm *SessionManager) NewSession(projectDir string) *Session {
	if !sm.enabled {
		return &Session{
			ID:         generateSessionId(),
			StartTime:  time.Now(),
			ProjectDir: projectDir,
			Messages:   []MsgEntry{},
		}
	}

	session := &Session{
		ID:         generateSessionId(),
		StartTime:  time.Now(),
		ProjectDir: projectDir,
		Messages:   []MsgEntry{},
	}

	sm.current = session
	return session
}

// GetCurrent returns the current session
func (sm *SessionManager) GetCurrent() *Session {
	return sm.current
}

// AddMessage adds a message to the current session
func (sm *SessionManager) AddMessage(role, content string) {
	if sm.current == nil {
		return
	}

	msg := MsgEntry{
		Role:       role,
		Content:    content,
		Timestamp:  time.Now(),
		IsGreeting: isGreeting(content),
	}

	sm.current.Messages = append(sm.current.Messages, msg)
}

// isGreeting detects if a message is a pure greeting
func isGreeting(content string) bool {
	// Length check
	if len(content) >= 50 {
		return false
	}

	// Normalize
	content = strings.ToLower(strings.TrimSpace(content))

	// Pattern match
	greetings := []string{"hi", "hello", "hey", "你好", "嗨", "您好", "hola", "早上好", "下午好", "晚上好"}
	for _, g := range greetings {
		if content == g || content == g+"!" || content == g+"." {
			// Check for task keywords
			keywords := []string{"task", "fix", "code", "bug", "implement", "add", "create", "update", "delete", "read", "write", "file", "help", "问题", "任务", "修复", "实现", "创建"}
			for _, kw := range keywords {
				if strings.Contains(content, kw) {
					return false
				}
			}
			return true
		}
	}

	return false
}

// CloseSession ends the current session and saves it
func (sm *SessionManager) CloseSession(summary SessionSummary) error {
	if sm.current == nil || !sm.enabled {
		return nil
	}

	sm.current.EndTime = time.Now()
	sm.current.Summary = summary

	return sm.SaveSession()
}

// SaveSession writes the current session to a file
func (sm *SessionManager) SaveSession() error {
	if sm.current == nil {
		return nil
	}

	sessionsDir := filepath.Join(sm.configDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}

	path := filepath.Join(sessionsDir, sm.current.ID+".json")
	data, err := json.MarshalIndent(sm.current, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadSession loads a session from a file
func (sm *SessionManager) LoadSession(id string) (*Session, error) {
	path := filepath.Join(sm.configDir, "sessions", id+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	return &session, nil
}

// ResumeSession loads and sets a session as current
func (sm *SessionManager) ResumeSession(id string) error {
	session, err := sm.LoadSession(id)
	if err != nil {
		return err
	}

	sm.current = session
	return nil
}

// ListSessions lists recent sessions sorted by time
func (sm *SessionManager) ListSessions(limit int) ([]SessionMeta, error) {
	sessionsDir := filepath.Join(sm.configDir, "sessions")

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var metas []SessionMeta
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(sessionsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		metas = append(metas, SessionMeta{
			ID:         session.ID,
			StartTime:  session.StartTime,
			ProjectDir: session.ProjectDir,
			Result:     session.Summary.Result,
		})
	}

	// Sort by start time descending
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].StartTime.After(metas[j].StartTime)
	})

	if limit > 0 && len(metas) > limit {
		metas = metas[:limit]
	}

	return metas, nil
}

// AutoResume loads the most recent session if auto-resume is enabled
func (sm *SessionManager) AutoResume() error {
	metas, err := sm.ListSessions(1)
	if err != nil {
		return err
	}

	if len(metas) == 0 {
		return nil
	}

	return sm.ResumeSession(metas[0].ID)
}

// CleanOldSessions removes session files older than retention days
func (sm *SessionManager) CleanOldSessions(retentionDays int) error {
	sessionsDir := filepath.Join(sm.configDir, "sessions")

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Parse session ID for date (YYYY-MM-DD-HHMMSS-RAND)
		name := entry.Name()
		if len(name) < 10 {
			continue
		}

		// Extract date part
		dateStr := name[:10]
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if fileDate.Before(cutoff) {
			path := filepath.Join(sessionsDir, name)
			if err := os.Remove(path); err != nil {
				fmt.Printf("Warning: failed to remove old session %s: %v\n", name, err)
			}
		}
	}

	return nil
}