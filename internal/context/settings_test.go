package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDefaultSettings(t *testing.T) {
	settings := GetDefaultSettings()

	// Test LLM defaults
	if settings.LLM.Provider != "anthropic" {
		t.Errorf("Expected default provider 'anthropic', got '%s'", settings.LLM.Provider)
	}
	if settings.LLM.Model != "claude-sonnet-4-6" {
		t.Errorf("Expected default model 'claude-sonnet-4-6', got '%s'", settings.LLM.Model)
	}
	if settings.LLM.MaxTokens != 4096 {
		t.Errorf("Expected default max tokens 4096, got %d", settings.LLM.MaxTokens)
	}
	if settings.LLM.Temperature != 0.7 {
		t.Errorf("Expected default temperature 0.7, got %f", settings.LLM.Temperature)
	}
	if settings.LLM.Timeout != 120 {
		t.Errorf("Expected default timeout 120, got %d", settings.LLM.Timeout)
	}

	// Test Context defaults
	if settings.Context.MaxContextTokens != 100000 {
		t.Errorf("Expected default max context tokens 100000, got %d", settings.Context.MaxContextTokens)
	}
	if settings.Context.MaxToolOutput != 10000 {
		t.Errorf("Expected default max tool output 10000, got %d", settings.Context.MaxToolOutput)
	}
	if settings.Context.HistoryWindowSize != 20 {
		t.Errorf("Expected default history window size 20, got %d", settings.Context.HistoryWindowSize)
	}

	// Test Memory defaults
	if !settings.Memory.EnableShortTerm {
		t.Error("Expected EnableShortTerm to be true")
	}
	if settings.Memory.ShortMemoryRetentionDays != 7 {
		t.Errorf("Expected ShortMemoryRetentionDays 7, got %d", settings.Memory.ShortMemoryRetentionDays)
	}
	if !settings.Memory.EnableLongTerm {
		t.Error("Expected EnableLongTerm to be true")
	}
	if settings.Memory.MaxMemoryLines != 200 {
		t.Errorf("Expected MaxMemoryLines 200, got %d", settings.Memory.MaxMemoryLines)
	}
	if settings.Memory.AutoArchive {
		t.Error("Expected AutoArchive to be false by default")
	}

	// Test Session defaults
	if !settings.Session.EnablePersistence {
		t.Error("Expected EnablePersistence to be true")
	}
	if settings.Session.SessionRetentionDays != 30 {
		t.Errorf("Expected SessionRetentionDays 30, got %d", settings.Session.SessionRetentionDays)
	}
	if settings.Session.AutoResume {
		t.Error("Expected AutoResume to be false by default")
	}

	// Test UI defaults
	if !settings.UI.ShowContextStats {
		t.Error("Expected ShowContextStats to be true")
	}
	if !settings.UI.ShowMemoryUsage {
		t.Error("Expected ShowMemoryUsage to be true")
	}
	if settings.UI.VerboseEvents {
		t.Error("Expected VerboseEvents to be false by default")
	}
}

func TestSettingsManagerLoad(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Test loading non-existent file (should use defaults)
	sm := NewSettingsManager(tmpDir)
	err := sm.Load()
	if err != nil {
		t.Fatalf("Load failed for non-existent file: %v", err)
	}

	settings := sm.Get()
	if settings.LLM.Provider != "anthropic" {
		t.Errorf("Expected default provider after loading non-existent file, got '%s'", settings.LLM.Provider)
	}

	// Test loading existing file
	data := `{
		"llm": {
			"provider": "openai",
			"model": "gpt-4",
			"maxTokens": 8192,
			"temperature": 0.5
		},
		"context": {
			"maxContextTokens": 50000,
			"maxToolOutput": 5000,
			"historyWindowSize": 10
		}
	}`

	if err := os.WriteFile(settingsPath, []byte(data), 0644); err != nil {
		t.Fatalf("Failed to write test settings: %v", err)
	}

	sm2 := NewSettingsManager(tmpDir)
	if err := sm2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	loaded := sm2.Get()
	if loaded.LLM.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", loaded.LLM.Provider)
	}
	if loaded.LLM.MaxTokens != 8192 {
		t.Errorf("Expected max tokens 8192, got %d", loaded.LLM.MaxTokens)
	}
	if loaded.Context.MaxContextTokens != 50000 {
		t.Errorf("Expected max context tokens 50000, got %d", loaded.Context.MaxContextTokens)
	}
}

func TestSettingsManagerSave(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSettingsManager(tmpDir)

	// Modify settings
	sm.settings.LLM.Provider = "openai"
	sm.settings.LLM.Model = "gpt-4-turbo"
	sm.settings.Context.MaxContextTokens = 80000

	// Save
	if err := sm.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	settingsPath := filepath.Join(tmpDir, "settings.json")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Fatal("Settings file was not created")
	}

	// Load into new manager and verify
	sm2 := NewSettingsManager(tmpDir)
	if err := sm2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	loaded := sm2.Get()
	if loaded.LLM.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", loaded.LLM.Provider)
	}
	if loaded.LLM.Model != "gpt-4-turbo" {
		t.Errorf("Expected model 'gpt-4-turbo', got '%s'", loaded.LLM.Model)
	}
	if loaded.Context.MaxContextTokens != 80000 {
		t.Errorf("Expected max context tokens 80000, got %d", loaded.Context.MaxContextTokens)
	}
}

func TestSettingsManagerUpdate(t *testing.T) {
	sm := NewSettingsManager(t.TempDir())

	updates := Settings{
		LLM: LLMSettings{
			Provider:    "anthropic",
			Model:       "claude-opus-4",
			Temperature: 0.9,
		},
		Context: ContextSettings{
			MaxContextTokens: 200000,
		},
	}

	if err := sm.Update(updates); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	settings := sm.Get()
	if settings.LLM.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got '%s'", settings.LLM.Provider)
	}
	if settings.LLM.Model != "claude-opus-4" {
		t.Errorf("Expected model 'claude-opus-4', got '%s'", settings.LLM.Model)
	}
	if settings.LLM.Temperature != 0.9 {
		t.Errorf("Expected temperature 0.9, got %f", settings.LLM.Temperature)
	}
	if settings.Context.MaxContextTokens != 200000 {
		t.Errorf("Expected max context tokens 200000, got %d", settings.Context.MaxContextTokens)
	}
}

func TestSettingsManagerResetToDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSettingsManager(tmpDir)

	// Modify settings
	sm.settings.LLM.Provider = "openai"
	sm.settings.LLM.Model = "gpt-4"
	sm.settings.Context.MaxContextTokens = 50000

	// Reset
	if err := sm.ResetToDefaults(); err != nil {
		t.Fatalf("ResetToDefaults failed: %v", err)
	}

	settings := sm.Get()
	if settings.LLM.Provider != "anthropic" {
		t.Errorf("Expected default provider after reset, got '%s'", settings.LLM.Provider)
	}
	if settings.Context.MaxContextTokens != 100000 {
		t.Errorf("Expected default max context tokens after reset, got %d", settings.Context.MaxContextTokens)
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	// Test Anthropic detection
	os.Setenv("ANTHROPIC_AUTH_TOKEN", "test-token")
	os.Unsetenv("OPENAI_API_KEY")

	sm := NewSettingsManager(t.TempDir())
	sm.ApplyEnvOverrides()

	if sm.Get().LLM.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic' after env detection, got '%s'", sm.Get().LLM.Provider)
	}

	// Test OpenAI detection
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")
	os.Setenv("OPENAI_API_KEY", "test-key")

	sm2 := NewSettingsManager(t.TempDir())
	sm2.ApplyEnvOverrides()

	if sm2.Get().LLM.Provider != "openai" {
		t.Errorf("Expected provider 'openai' after env detection, got '%s'", sm2.Get().LLM.Provider)
	}

	// Cleanup
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")
	os.Unsetenv("OPENAI_API_KEY")
}

func TestApplyCLIFlags(t *testing.T) {
	sm := NewSettingsManager(t.TempDir())

	sm.ApplyCLIFlags(
		"anthropic",           // provider
		"claude-opus-4",       // model
		"https://custom.api",  // baseURL
		8192,                  // maxTokens
		0.5,                   // temperature
		50000,                 // maxContextTokens
		5000,                  // maxToolOutput
		15,                    // historyWindow
		true,                  // noMemory
		14,                    // shortMemoryDays
		true,                  // autoArchive
		true,                  // noSession
		60,                    // sessionDays
		true,                  // autoResume
	)

	settings := sm.Get()
	if settings.LLM.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got '%s'", settings.LLM.Provider)
	}
	if settings.LLM.Model != "claude-opus-4" {
		t.Errorf("Expected model 'claude-opus-4', got '%s'", settings.LLM.Model)
	}
	if settings.LLM.BaseURL != "https://custom.api" {
		t.Errorf("Expected baseURL 'https://custom.api', got '%s'", settings.LLM.BaseURL)
	}
	if settings.LLM.MaxTokens != 8192 {
		t.Errorf("Expected maxTokens 8192, got %d", settings.LLM.MaxTokens)
	}
	if settings.LLM.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", settings.LLM.Temperature)
	}
	if settings.Context.MaxContextTokens != 50000 {
		t.Errorf("Expected maxContextTokens 50000, got %d", settings.Context.MaxContextTokens)
	}
	if settings.Memory.EnableShortTerm {
		t.Error("Expected EnableShortTerm to be false after --no-memory")
	}
	if settings.Session.EnablePersistence {
		t.Error("Expected EnablePersistence to be false after --no-session")
	}
	if !settings.Session.AutoResume {
		t.Error("Expected AutoResume to be true after --auto-resume")
	}
}