package context

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MemorySection represents a section in short-term memory
type MemorySection string

const (
	SectionCurrentTask   MemorySection = "## 当前任务"
	SectionDiscoveries   MemorySection = "## 重要发现"
	SectionFrequentFiles MemorySection = "## 常用文件"
	SectionTodo          MemorySection = "## 待完成"
)

// MemoryManager manages short-term memory files
type MemoryManager struct {
	configDir     string
	retentionDays int
	enabled       bool
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(configDir string, retentionDays int, enabled bool) *MemoryManager {
	return &MemoryManager{
		configDir:     configDir,
		retentionDays: retentionDays,
		enabled:       enabled,
	}
}

// GetTodayMemoryPath returns the path to today's memory file
func (mm *MemoryManager) GetTodayMemoryPath() string {
	today := time.Now().Format("2006-01-02")
	return filepath.Join(mm.configDir, "memory", today+".md")
}

// GetMemoryPath returns the path for a specific date
func (mm *MemoryManager) GetMemoryPath(date string) string {
	return filepath.Join(mm.configDir, "memory", date+".md")
}

// LoadTodayMemory loads today's short-term memory
func (mm *MemoryManager) LoadTodayMemory() string {
	if !mm.enabled {
		return ""
	}
	path := mm.GetTodayMemoryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadYesterdayMemory loads yesterday's short-term memory
func (mm *MemoryManager) LoadYesterdayMemory() string {
	if !mm.enabled {
		return ""
	}
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	path := mm.GetMemoryPath(yesterday)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// AppendMemory adds content to a specific section in today's memory
func (mm *MemoryManager) AppendMemory(section MemorySection, content string) error {
	if !mm.enabled {
		return nil
	}

	path := mm.GetTodayMemoryPath()

	// Read existing content or create new
	var existing string
	if data, err := os.ReadFile(path); err == nil {
		existing = string(data)
	}

	// Find or create the section
	newContent := mm.appendToSection(existing, section, content)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(newContent), 0644)
}

// appendToSection adds content to a specific section
func (mm *MemoryManager) appendToSection(existing string, section MemorySection, content string) string {
	lines := strings.Split(existing, "\n")
	sectionStr := string(section)

	// Find the section
	sectionIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, sectionStr) {
			sectionIdx = i
			break
		}
	}

	// If section doesn't exist, add it
	if sectionIdx < 0 {
		if existing != "" && !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		return existing + sectionStr + "\n- " + content + "\n"
	}

	// Insert content after the section header
	newLines := make([]string, 0, len(lines)+2)
	newLines = append(newLines, lines[:sectionIdx+1]...)
	newLines = append(newLines, "- "+content)
	newLines = append(newLines, lines[sectionIdx+1:]...)

	return strings.Join(newLines, "\n")
}

// SaveMemory saves content to today's memory file
func (mm *MemoryManager) SaveMemory(content string) error {
	if !mm.enabled {
		return nil
	}

	path := mm.GetTodayMemoryPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(content), 0644)
}

// CleanOldMemory removes memory files older than retention days
func (mm *MemoryManager) CleanOldMemory() error {
	memoryDir := filepath.Join(mm.configDir, "memory")

	entries, err := os.ReadDir(memoryDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -mm.retentionDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Parse date from filename (YYYY-MM-DD.md)
		name := entry.Name()
		if len(name) < 10 {
			continue
		}

		dateStr := name[:10]
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if fileDate.Before(cutoff) {
			path := filepath.Join(memoryDir, name)
			if err := os.Remove(path); err != nil {
				fmt.Printf("Warning: failed to remove old memory %s: %v\n", name, err)
			}
		}
	}

	return nil
}

// ArchiveToLongTerm moves a date's memory to long-term memory
func (mm *MemoryManager) ArchiveToLongTerm(date string) error {
	// Read the short-term memory
	path := mm.GetMemoryPath(date)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read memory: %w", err)
	}

	// Append to MEMORY.md
	memoryPath := filepath.Join(mm.configDir, "MEMORY.md")
	var existing string
	if existingData, err := os.ReadFile(memoryPath); err == nil {
		existing = string(existingData)
	}

	// Add archive entry
	archiveEntry := fmt.Sprintf("\n## Archived %s\n\n%s\n", date, string(data))
	newContent := existing + archiveEntry

	if err := os.WriteFile(memoryPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write memory: %w", err)
	}

	// Remove the short-term memory file
	return os.Remove(path)
}

// ListRecentMemory lists recent memory files
func (mm *MemoryManager) ListRecentMemory(limit int) ([]string, error) {
	memoryDir := filepath.Join(mm.configDir, "memory")

	entries, err := os.ReadDir(memoryDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry.Name()[:len(entry.Name())-3])
		}
	}

	// Sort descending (most recent first)
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}

	return files, nil
}

// LimitMemoryLines limits MEMORY.md to maxLines
func (mm *MemoryManager) LimitMemoryLines(maxLines int) error {
	memoryPath := filepath.Join(mm.configDir, "MEMORY.md")

	data, err := os.ReadFile(memoryPath)
	if err != nil {
		return nil // File doesn't exist, nothing to limit
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) <= maxLines {
		return nil
	}

	// Keep the last maxLines
	newLines := lines[len(lines)-maxLines:]
	newContent := strings.Join(newLines, "\n")

	return os.WriteFile(memoryPath, []byte(newContent), 0644)
}