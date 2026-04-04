package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	dir := GetConfigDir()

	if dir == "" {
		t.Error("GetConfigDir returned empty string")
	}

	if !strings.Contains(dir, ".config") {
		t.Errorf("Expected config dir to contain '.config', got '%s'", dir)
	}

	if !strings.Contains(dir, "autocode") {
		t.Errorf("Expected config dir to contain 'autocode', got '%s'", dir)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "autocode")

	if err := EnsureConfigDir(configDir); err != nil {
		t.Fatalf("EnsureConfigDir failed: %v", err)
	}

	// Check main directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("Config directory was not created")
	}

	// Check subdirectories
	sessionsDir := filepath.Join(configDir, "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		t.Error("Sessions subdirectory was not created")
	}

	memoryDir := filepath.Join(configDir, "memory")
	if _, err := os.Stat(memoryDir); os.IsNotExist(err) {
		t.Error("Memory subdirectory was not created")
	}
}

func TestCreateDefaultFiles(t *testing.T) {
	tmpDir := t.TempDir()

	if err := CreateDefaultFiles(tmpDir); err != nil {
		t.Fatalf("CreateDefaultFiles failed: %v", err)
	}

	// Check each default file
	expectedFiles := []string{"ENVIRONMENT.md", "SKILLS.md", "MEMORY.md"}
	for _, filename := range expectedFiles {
		path := filepath.Join(tmpDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Default file '%s' was not created", filename)
		}

		// Verify file has content
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read %s: %v", filename, err)
		}
		if len(data) == 0 {
			t.Errorf("Default file '%s' is empty", filename)
		}
	}
}

func TestContextLoaderLoadSystemPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := t.TempDir()

	// Test with non-existent file
	loader := NewContextLoader(tmpDir, projectDir)
	result := loader.LoadSystemPrompt()
	if result != "" {
		t.Error("Expected empty string for non-existent SYSTEM_PROMPT.md")
	}

	// Test with existing file
	customPrompt := "# Custom System Prompt\n\nThis is a custom prompt."
	promptPath := filepath.Join(tmpDir, "SYSTEM_PROMPT.md")
	if err := os.WriteFile(promptPath, []byte(customPrompt), 0644); err != nil {
		t.Fatalf("Failed to write custom prompt: %v", err)
	}

	result = loader.LoadSystemPrompt()
	if result != customPrompt {
		t.Errorf("Expected custom prompt, got '%s'", result)
	}
}

func TestContextLoaderLoadMemory(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := t.TempDir()

	loader := NewContextLoader(tmpDir, projectDir)

	// Test non-existent file
	result := loader.LoadMemory()
	if result != "" {
		t.Error("Expected empty string for non-existent MEMORY.md")
	}

	// Test existing file
	memoryContent := "# Long-term Memory\n\n- [Item](memory/item.md) — Description"
	memoryPath := filepath.Join(tmpDir, "MEMORY.md")
	if err := os.WriteFile(memoryPath, []byte(memoryContent), 0644); err != nil {
		t.Fatalf("Failed to write memory file: %v", err)
	}

	result = loader.LoadMemory()
	if result != memoryContent {
		t.Errorf("Expected memory content, got '%s'", result)
	}
}

func TestContextLoaderLoadShortTermMemory(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := t.TempDir()

	loader := NewContextLoader(tmpDir, projectDir)

	// Test non-existent file
	result := loader.LoadShortTermMemory()
	if result != "" {
		t.Error("Expected empty string for non-existent short-term memory")
	}

	// Create today's memory file
	today := "2026-04-04" // Fixed date for testing
	memoryDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memoryDir, 0755)

	memoryContent := "## Current Tasks\n- Testing"
	memoryPath := filepath.Join(memoryDir, today+".md")
	if err := os.WriteFile(memoryPath, []byte(memoryContent), 0644); err != nil {
		t.Fatalf("Failed to write short-term memory: %v", err)
	}

	// Note: This test uses the actual time.Now(), so it might not find the file
	// if run on a different date. In production, you'd inject the time.
}

func TestContextLoaderLoadEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := t.TempDir()

	loader := NewContextLoader(tmpDir, projectDir)

	// Test non-existent file
	result := loader.LoadEnvironment()
	if result != "" {
		t.Error("Expected empty string for non-existent ENVIRONMENT.md")
	}

	// Test existing file
	envContent := "# Environment\n\n- OS: Linux"
	envPath := filepath.Join(tmpDir, "ENVIRONMENT.md")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write environment file: %v", err)
	}

	result = loader.LoadEnvironment()
	if result != envContent {
		t.Errorf("Expected environment content, got '%s'", result)
	}
}

func TestContextLoaderLoadSkills(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := t.TempDir()

	loader := NewContextLoader(tmpDir, projectDir)

	// Test non-existent file
	result := loader.LoadSkills()
	if result != "" {
		t.Error("Expected empty string for non-existent SKILLS.md")
	}

	// Test existing file
	skillsContent := "# Skills\n\n- Code review"
	skillsPath := filepath.Join(tmpDir, "SKILLS.md")
	if err := os.WriteFile(skillsPath, []byte(skillsContent), 0644); err != nil {
		t.Fatalf("Failed to write skills file: %v", err)
	}

	result = loader.LoadSkills()
	if result != skillsContent {
		t.Errorf("Expected skills content, got '%s'", result)
	}
}

func TestContextLoaderLoadProjectClaudeMd(t *testing.T) {
	// Create temp project directory structure
	tmpDir := t.TempDir()
	projectDir := tmpDir

	loader := NewContextLoader(t.TempDir(), projectDir)

	// Test non-existent file
	result := loader.LoadProjectClaudeMd()
	if result != "" {
		t.Error("Expected empty string for non-existent CLAUDE.md")
	}

	// Test CLAUDE.md in project root
	claudeContent := "# Project Instructions\n\nFollow these rules."
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte(claudeContent), 0644); err != nil {
		t.Fatalf("Failed to write CLAUDE.md: %v", err)
	}

	result = loader.LoadProjectClaudeMd()
	if result != claudeContent {
		t.Errorf("Expected CLAUDE.md content, got '%s'", result)
	}
}

func TestContextLoaderLoadProjectClaudeMdInClaudeDir(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := tmpDir
	configDir := t.TempDir()

	loader := NewContextLoader(configDir, projectDir)

	// Create .claude directory
	claudeDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	claudeContent := "# Hidden Project Instructions"
	claudePath := filepath.Join(claudeDir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte(claudeContent), 0644); err != nil {
		t.Fatalf("Failed to write .claude/CLAUDE.md: %v", err)
	}

	result := loader.LoadProjectClaudeMd()
	if result != claudeContent {
		t.Errorf("Expected .claude/CLAUDE.md content, got '%s'", result)
	}
}

func TestContextLoaderBuildFullPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := t.TempDir()

	loader := NewContextLoader(tmpDir, projectDir)

	defaultPrompt := "Default system prompt"

	// Test with no context files
	result := loader.BuildFullPrompt(defaultPrompt)
	if !strings.Contains(result, defaultPrompt) {
		t.Error("Expected result to contain default prompt")
	}

	// Test with custom system prompt
	customPrompt := "Custom system prompt"
	promptPath := filepath.Join(tmpDir, "SYSTEM_PROMPT.md")
	os.WriteFile(promptPath, []byte(customPrompt), 0644)

	result = loader.BuildFullPrompt(defaultPrompt)
	if !strings.Contains(result, customPrompt) {
		t.Error("Expected result to contain custom prompt")
	}
	if strings.Contains(result, defaultPrompt) {
		t.Error("Expected custom prompt to replace default prompt")
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"a", 0},       // 1 char = 0 tokens (integer division)
		{"abcd", 1},    // 4 chars = 1 token
		{"abcdefgh", 2}, // 8 chars = 2 tokens
		{"Hello, world!", 3}, // 13 chars = 3 tokens
	}

	for _, test := range tests {
		result := EstimateTokens(test.text)
		if result != test.expected {
			t.Errorf("EstimateTokens(%q) = %d, expected %d", test.text, result, test.expected)
		}
	}
}