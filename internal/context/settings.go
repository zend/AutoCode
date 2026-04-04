package context

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings is the main configuration structure
type Settings struct {
	LLM     LLMSettings     `json:"llm"`
	Context ContextSettings `json:"context"`
	Memory  MemorySettings  `json:"memory"`
	Session SessionSettings `json:"session"`
	UI      UISettings      `json:"ui"`
}

// LLMSettings configures the LLM client
type LLMSettings struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	BaseURL     string  `json:"baseUrl"`
	MaxTokens   int     `json:"maxTokens"`
	Temperature float64 `json:"temperature"`
	Timeout     int     `json:"timeout"`
}

// ContextSettings configures context management
type ContextSettings struct {
	MaxContextTokens  int `json:"maxContextTokens"`
	MaxToolOutput     int `json:"maxToolOutput"`
	HistoryWindowSize int `json:"historyWindowSize"`
}

// MemorySettings configures memory management
type MemorySettings struct {
	EnableShortTerm          bool `json:"enableShortTerm"`
	ShortMemoryRetentionDays int  `json:"shortMemoryRetentionDays"`
	EnableLongTerm           bool `json:"enableLongTerm"`
	MaxMemoryLines           int  `json:"maxMemoryLines"`
	AutoArchive              bool `json:"autoArchive"`
}

// SessionSettings configures session persistence
type SessionSettings struct {
	EnablePersistence    bool `json:"enablePersistence"`
	SessionRetentionDays int  `json:"sessionRetentionDays"`
	AutoResume           bool `json:"autoResume"`
}

// UISettings configures UI display
type UISettings struct {
	ShowContextStats bool `json:"showContextStats"`
	ShowMemoryUsage  bool `json:"showMemoryUsage"`
	VerboseEvents    bool `json:"verboseEvents"`
}

// GetDefaultSettings returns the default configuration
func GetDefaultSettings() Settings {
	return Settings{
		LLM: LLMSettings{
			Provider:    "anthropic",
			Model:       "claude-sonnet-4-6",
			BaseURL:     "",
			MaxTokens:   4096,
			Temperature: 0.7,
			Timeout:     120,
		},
		Context: ContextSettings{
			MaxContextTokens:  100000,
			MaxToolOutput:     10000,
			HistoryWindowSize: 20,
		},
		Memory: MemorySettings{
			EnableShortTerm:          true,
			ShortMemoryRetentionDays: 7,
			EnableLongTerm:           true,
			MaxMemoryLines:           200,
			AutoArchive:              false,
		},
		Session: SessionSettings{
			EnablePersistence:    true,
			SessionRetentionDays: 30,
			AutoResume:           false,
		},
		UI: UISettings{
			ShowContextStats: true,
			ShowMemoryUsage:  true,
			VerboseEvents:    false,
		},
	}
}

// SettingsManager manages configuration loading and saving
type SettingsManager struct {
	configDir string
	settings  Settings
}

// NewSettingsManager creates a new settings manager
func NewSettingsManager(configDir string) *SettingsManager {
	return &SettingsManager{
		configDir: configDir,
		settings:  GetDefaultSettings(),
	}
}

// Load reads settings from settings.json
func (sm *SettingsManager) Load() error {
	settingsPath := filepath.Join(sm.configDir, "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, use defaults
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &sm.settings)
}

// Save writes settings to settings.json
func (sm *SettingsManager) Save() error {
	settingsPath := filepath.Join(sm.configDir, "settings.json")

	data, err := json.MarshalIndent(sm.settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, 0644)
}

// Get returns the current settings
func (sm *SettingsManager) Get() Settings {
	return sm.settings
}

// Update merges updates into current settings
func (sm *SettingsManager) Update(updates Settings) error {
	// Merge non-zero values from updates
	if updates.LLM.Provider != "" {
		sm.settings.LLM.Provider = updates.LLM.Provider
	}
	if updates.LLM.Model != "" {
		sm.settings.LLM.Model = updates.LLM.Model
	}
	if updates.LLM.BaseURL != "" {
		sm.settings.LLM.BaseURL = updates.LLM.BaseURL
	}
	if updates.LLM.MaxTokens != 0 {
		sm.settings.LLM.MaxTokens = updates.LLM.MaxTokens
	}
	if updates.LLM.Temperature != 0 {
		sm.settings.LLM.Temperature = updates.LLM.Temperature
	}
	if updates.LLM.Timeout != 0 {
		sm.settings.LLM.Timeout = updates.LLM.Timeout
	}
	if updates.Context.MaxContextTokens != 0 {
		sm.settings.Context.MaxContextTokens = updates.Context.MaxContextTokens
	}
	if updates.Context.MaxToolOutput != 0 {
		sm.settings.Context.MaxToolOutput = updates.Context.MaxToolOutput
	}
	if updates.Context.HistoryWindowSize != 0 {
		sm.settings.Context.HistoryWindowSize = updates.Context.HistoryWindowSize
	}

	// Boolean fields need special handling - we check if they're different from default
	sm.settings.Memory.EnableShortTerm = updates.Memory.EnableShortTerm
	sm.settings.Memory.EnableLongTerm = updates.Memory.EnableLongTerm
	sm.settings.Memory.AutoArchive = updates.Memory.AutoArchive
	sm.settings.Session.EnablePersistence = updates.Session.EnablePersistence
	sm.settings.Session.AutoResume = updates.Session.AutoResume
	sm.settings.UI.ShowContextStats = updates.UI.ShowContextStats
	sm.settings.UI.ShowMemoryUsage = updates.UI.ShowMemoryUsage
	sm.settings.UI.VerboseEvents = updates.UI.VerboseEvents

	if updates.Memory.ShortMemoryRetentionDays != 0 {
		sm.settings.Memory.ShortMemoryRetentionDays = updates.Memory.ShortMemoryRetentionDays
	}
	if updates.Memory.MaxMemoryLines != 0 {
		sm.settings.Memory.MaxMemoryLines = updates.Memory.MaxMemoryLines
	}
	if updates.Session.SessionRetentionDays != 0 {
		sm.settings.Session.SessionRetentionDays = updates.Session.SessionRetentionDays
	}

	return nil
}

// ResetToDefaults resets settings to default values
func (sm *SettingsManager) ResetToDefaults() error {
	sm.settings = GetDefaultSettings()
	return sm.Save()
}

// ApplyEnvOverrides applies environment variable overrides
func (sm *SettingsManager) ApplyEnvOverrides() {
	// Detect provider from environment variables
	if os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" {
		sm.settings.LLM.Provider = "anthropic"
	} else if os.Getenv("OPENAI_API_KEY") != "" {
		sm.settings.LLM.Provider = "openai"
	}

	// Apply base URL overrides
	if url := os.Getenv("ANTHROPIC_BASE_URL"); url != "" && sm.settings.LLM.Provider == "anthropic" {
		sm.settings.LLM.BaseURL = url
	}
	if url := os.Getenv("OPENAI_BASE_URL"); url != "" && sm.settings.LLM.Provider == "openai" {
		sm.settings.LLM.BaseURL = url
	}
}

// ApplyCLIFlags applies command-line flag overrides
func (sm *SettingsManager) ApplyCLIFlags(
	provider, model, baseURL string,
	maxTokens int,
	temperature float64,
	maxContextTokens, maxToolOutput, historyWindow int,
	noMemory bool,
	shortMemoryDays int,
	autoArchive bool,
	noSession bool,
	sessionDays int,
	autoResume bool,
) {
	if provider != "" {
		sm.settings.LLM.Provider = provider
	}
	if model != "" {
		sm.settings.LLM.Model = model
	}
	if baseURL != "" {
		sm.settings.LLM.BaseURL = baseURL
	}
	if maxTokens != 0 {
		sm.settings.LLM.MaxTokens = maxTokens
	}
	if temperature != 0 {
		sm.settings.LLM.Temperature = temperature
	}
	if maxContextTokens != 0 {
		sm.settings.Context.MaxContextTokens = maxContextTokens
	}
	if maxToolOutput != 0 {
		sm.settings.Context.MaxToolOutput = maxToolOutput
	}
	if historyWindow != 0 {
		sm.settings.Context.HistoryWindowSize = historyWindow
	}
	if noMemory {
		sm.settings.Memory.EnableShortTerm = false
		sm.settings.Memory.EnableLongTerm = false
	}
	if shortMemoryDays != 0 {
		sm.settings.Memory.ShortMemoryRetentionDays = shortMemoryDays
	}
	if autoArchive {
		sm.settings.Memory.AutoArchive = true
	}
	if noSession {
		sm.settings.Session.EnablePersistence = false
	}
	if sessionDays != 0 {
		sm.settings.Session.SessionRetentionDays = sessionDays
	}
	if autoResume {
		sm.settings.Session.AutoResume = true
	}
}