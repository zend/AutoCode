package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ContextLoader loads context files from the config directory
type ContextLoader struct {
	configDir  string // ~/.config/autocode
	projectDir string // Current working directory
}

// NewContextLoader creates a new context loader
func NewContextLoader(configDir, projectDir string) *ContextLoader {
	return &ContextLoader{
		configDir:  configDir,
		projectDir: projectDir,
	}
}

// GetConfigDir returns the config directory path
func GetConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "~"
	}
	return filepath.Join(homeDir, ".config", "autocode")
}

// EnsureConfigDir creates the config directory and subdirectories
func EnsureConfigDir(configDir string) error {
	subdirs := []string{
		"sessions",
		"memory",
	}

	// Create main config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	// Create subdirectories
	for _, subdir := range subdirs {
		path := filepath.Join(configDir, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("create subdir %s: %w", subdir, err)
		}
	}

	return nil
}

// CreateDefaultFiles creates default context files if they don't exist
func CreateDefaultFiles(configDir string) error {
	defaultFiles := map[string]string{
		"ENVIRONMENT.md": `# Environment Information

## System
- OS: Linux
- Shell: bash

## Preferences
- Editor: vim
- Language: zh-CN

## Notes
Add your environment details and preferences here.
`,
		"SKILLS.md": `# Skills

List your available skills and capabilities here.

## Example
- Code review
- Bug fixing
- Documentation
`,
		"MEMORY.md": `# Long-term Memory

This file contains persistent memories across sessions.

Format:
- [Topic](memory/detail.md) — Brief description
`,
		"banner.txt": `    #                         #####
   # #   #    # #####  ####  #     #  ####  #####  ######
  #   #  #    #   #   #    # #       #    # #    # #
 #     # #    #   #   #    # #       #    # #    # #####
 ####### #    #   #   #    # #       #    # #    # #
 #     # #    #   #   #    # #     # #    # #    # #
 #     #  ####    #    ####   #####   ####  #####  ######

`,
	}

	for filename, content := range defaultFiles {
		path := filepath.Join(configDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("create default file %s: %w", filename, err)
			}
		}
	}

	return nil
}

// LoadSystemPrompt reads SYSTEM_PROMPT.md, returns default if not exists
func (cl *ContextLoader) LoadSystemPrompt() string {
	path := filepath.Join(cl.configDir, "SYSTEM_PROMPT.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "" // Return empty to use default system prompt
	}
	return string(data)
}

// LoadMemory reads MEMORY.md (long-term memory index)
func (cl *ContextLoader) LoadMemory() string {
	path := filepath.Join(cl.configDir, "MEMORY.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadShortTermMemory reads memory/YYYY-MM-DD.md (today's short-term memory)
func (cl *ContextLoader) LoadShortTermMemory() string {
	today := time.Now().Format("2006-01-02")
	path := filepath.Join(cl.configDir, "memory", today+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadYesterdayMemory reads yesterday's short-term memory
func (cl *ContextLoader) LoadYesterdayMemory() string {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	path := filepath.Join(cl.configDir, "memory", yesterday+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadEnvironment reads ENVIRONMENT.md
func (cl *ContextLoader) LoadEnvironment() string {
	path := filepath.Join(cl.configDir, "ENVIRONMENT.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadSkills reads SKILLS.md
func (cl *ContextLoader) LoadSkills() string {
	path := filepath.Join(cl.configDir, "SKILLS.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadBanner reads banner.txt for startup display
func (cl *ContextLoader) LoadBanner() string {
	path := filepath.Join(cl.configDir, "banner.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadProjectClaudeMd finds and reads project CLAUDE.md
func (cl *ContextLoader) LoadProjectClaudeMd() string {
	// Search order: .claude/CLAUDE.md, CLAUDE.md, parent dirs
	searchPaths := []string{
		filepath.Join(cl.projectDir, ".claude", "CLAUDE.md"),
		filepath.Join(cl.projectDir, "CLAUDE.md"),
	}

	// Also search parent directories (up to 3 levels)
	dir := cl.projectDir
	for i := 0; i < 3; i++ {
		parent := filepath.Dir(dir)
		if parent == dir || parent == "/" {
			break
		}
		searchPaths = append(searchPaths,
			filepath.Join(parent, ".claude", "CLAUDE.md"),
			filepath.Join(parent, "CLAUDE.md"),
		)
		dir = parent
	}

	for _, path := range searchPaths {
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data)
		}
	}

	return ""
}

// BuildFullPrompt combines all context files in priority order
// Priority (lowest to highest):
// 1. Default system prompt (passed in)
// 2. SYSTEM_PROMPT.md (user override)
// 3. ENVIRONMENT.md
// 4. SKILLS.md
// 5. Short-term memory (today)
// 6. MEMORY.md (long-term)
// 7. Project CLAUDE.md
func (cl *ContextLoader) BuildFullPrompt(defaultPrompt string) string {
	var parts []string

	// 1. Default or user system prompt
	customPrompt := cl.LoadSystemPrompt()
	if customPrompt != "" {
		parts = append(parts, customPrompt)
	} else if defaultPrompt != "" {
		parts = append(parts, defaultPrompt)
	}

	// 2. Environment
	if env := cl.LoadEnvironment(); env != "" {
		parts = append(parts, "\n\n## Environment\n\n"+env)
	}

	// 3. Skills
	if skills := cl.LoadSkills(); skills != "" {
		parts = append(parts, "\n\n## Skills\n\n"+skills)
	}

	// 4. Short-term memory (today)
	if stm := cl.LoadShortTermMemory(); stm != "" {
		parts = append(parts, "\n\n## Current Session Context\n\n"+stm)
	}

	// 5. Long-term memory
	if ltm := cl.LoadMemory(); ltm != "" {
		parts = append(parts, "\n\n## Memory\n\n"+ltm)
	}

	// 6. Project CLAUDE.md (highest priority)
	if claudeMd := cl.LoadProjectClaudeMd(); claudeMd != "" {
		parts = append(parts, "\n\n## Project Instructions\n\n"+claudeMd)
	}

	return strings.Join(parts, "")
}

// EstimateTokens estimates token count from character count
// Uses 4 chars ≈ 1 token approximation
func EstimateTokens(text string) int {
	return len(text) / 4
}