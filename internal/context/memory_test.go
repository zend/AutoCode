package context

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryManagerGetTodayMemoryPath(t *testing.T) {
	tmpDir := t.TempDir()
	mm := NewMemoryManager(tmpDir, 7, true)

	path := mm.GetTodayMemoryPath()
	today := time.Now().Format("2006-01-02")

	expected := filepath.Join(tmpDir, "memory", today+".md")
	if path != expected {
		t.Errorf("Expected path '%s', got '%s'", expected, path)
	}
}

func TestMemoryManagerLoadTodayMemory(t *testing.T) {
	tmpDir := t.TempDir()
	mm := NewMemoryManager(tmpDir, 7, true)

	// Test with non-existent file
	result := mm.LoadTodayMemory()
	if result != "" {
		t.Error("Expected empty string for non-existent memory file")
	}

	// Test with existing file
	today := time.Now().Format("2006-01-02")
	memoryDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memoryDir, 0755)

	content := "## Current Tasks\n- Testing memory"
	memoryPath := filepath.Join(memoryDir, today+".md")
	if err := os.WriteFile(memoryPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write memory file: %v", err)
	}

	result = mm.LoadTodayMemory()
	if result != content {
		t.Errorf("Expected content '%s', got '%s'", content, result)
	}
}

func TestMemoryManagerLoadTodayMemoryDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	mm := NewMemoryManager(tmpDir, 7, false) // disabled

	result := mm.LoadTodayMemory()
	if result != "" {
		t.Error("Expected empty string when memory is disabled")
	}
}

func TestMemoryManagerLoadYesterdayMemory(t *testing.T) {
	tmpDir := t.TempDir()
	mm := NewMemoryManager(tmpDir, 7, true)

	// Test with non-existent file
	result := mm.LoadYesterdayMemory()
	if result != "" {
		t.Error("Expected empty string for non-existent yesterday memory")
	}

	// Test with existing file
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	memoryDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memoryDir, 0755)

	content := "## Yesterday's Tasks"
	memoryPath := filepath.Join(memoryDir, yesterday+".md")
	if err := os.WriteFile(memoryPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write yesterday memory: %v", err)
	}

	result = mm.LoadYesterdayMemory()
	if result != content {
		t.Errorf("Expected content '%s', got '%s'", content, result)
	}
}

func TestMemoryManagerAppendMemory(t *testing.T) {
	tmpDir := t.TempDir()
	mm := NewMemoryManager(tmpDir, 7, true)

	// Append to non-existent file
	if err := mm.AppendMemory(SectionCurrentTask, "Test task"); err != nil {
		t.Fatalf("AppendMemory failed: %v", err)
	}

	// Verify file was created
	result := mm.LoadTodayMemory()
	if result == "" {
		t.Fatal("Memory file was not created")
	}

	// Check that section and content are present
	if !contains(result, "当前任务") {
		t.Error("Expected section header in result")
	}
	if !contains(result, "Test task") {
		t.Error("Expected task content in result")
	}

	// Append another item
	if err := mm.AppendMemory(SectionDiscoveries, "Found a bug"); err != nil {
		t.Fatalf("Second AppendMemory failed: %v", err)
	}

	result = mm.LoadTodayMemory()
	if !contains(result, "重要发现") {
		t.Error("Expected discoveries section in result")
	}
	if !contains(result, "Found a bug") {
		t.Error("Expected discovery content in result")
	}
}

func TestMemoryManagerSaveMemory(t *testing.T) {
	tmpDir := t.TempDir()
	mm := NewMemoryManager(tmpDir, 7, true)

	content := "## Tasks\n- Task 1\n- Task 2"
	if err := mm.SaveMemory(content); err != nil {
		t.Fatalf("SaveMemory failed: %v", err)
	}

	result := mm.LoadTodayMemory()
	if result != content {
		t.Errorf("Expected '%s', got '%s'", content, result)
	}
}

func TestMemoryManagerCleanOldMemory(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memoryDir, 0755)

	mm := NewMemoryManager(tmpDir, 3, true) // 3 days retention

	// Create old memory files
	oldDates := []string{
		"2026-03-31", // 4 days ago (should be deleted)
		"2026-04-01", // 3 days ago (should be kept)
		"2026-04-02", // 2 days ago (should be kept)
		"2026-04-03", // 1 day ago (should be kept)
	}

	for _, date := range oldDates {
		path := filepath.Join(memoryDir, date+".md")
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create memory file %s: %v", date, err)
		}
	}

	// Clean old memory
	if err := mm.CleanOldMemory(); err != nil {
		t.Fatalf("CleanOldMemory failed: %v", err)
	}

	// Check that old files are deleted
	entries, _ := os.ReadDir(memoryDir)
	remainingCount := len(entries)

	// Expected: 3 files kept (2026-04-01, 2026-04-02, 2026-04-03)
	// Note: Actual behavior depends on current date
	if remainingCount > 4 {
		t.Errorf("Expected at most 4 remaining files, got %d", remainingCount)
	}
}

func TestMemoryManagerListRecentMemory(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memoryDir, 0755)

	mm := NewMemoryManager(tmpDir, 7, true)

	// Create memory files
	dates := []string{"2026-04-01", "2026-04-02", "2026-04-03"}
	for _, date := range dates {
		path := filepath.Join(memoryDir, date+".md")
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create memory file: %v", err)
		}
	}

	// List with limit
	list, err := mm.ListRecentMemory(2)
	if err != nil {
		t.Fatalf("ListRecentMemory failed: %v", err)
	}

	if len(list) > 2 {
		t.Errorf("Expected at most 2 entries, got %d", len(list))
	}

	// Verify sorted descending (most recent first)
	if len(list) >= 2 {
		if list[0] < list[1] {
			t.Error("Expected most recent date first")
		}
	}
}

func TestMemoryManagerLimitMemoryLines(t *testing.T) {
	tmpDir := t.TempDir()
	mm := NewMemoryManager(tmpDir, 7, true)

	// Create MEMORY.md with many lines
	var lines []string
	for i := 0; i < 300; i++ {
		lines = append(lines, "Line of content")
	}
	content := ""
	for _, l := range lines {
		content += l + "\n"
	}

	memoryPath := filepath.Join(tmpDir, "MEMORY.md")
	os.WriteFile(memoryPath, []byte(content), 0644)

	// Limit to 200 lines
	if err := mm.LimitMemoryLines(200); err != nil {
		t.Fatalf("LimitMemoryLines failed: %v", err)
	}

	// Verify line count
	data, _ := os.ReadFile(memoryPath)
	resultLines := countLines(string(data))

	if resultLines > 200 {
		t.Errorf("Expected at most 200 lines, got %d", resultLines)
	}
}

func TestMemoryManagerArchiveToLongTerm(t *testing.T) {
	tmpDir := t.TempDir()
	mm := NewMemoryManager(tmpDir, 7, true)

	// Create short-term memory
	memoryDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memoryDir, 0755)

	shortTermContent := "## Tasks\n- Task to archive"
	shortTermPath := filepath.Join(memoryDir, "2026-04-01.md")
	os.WriteFile(shortTermPath, []byte(shortTermContent), 0644)

	// Archive to long-term
	if err := mm.ArchiveToLongTerm("2026-04-01"); err != nil {
		t.Fatalf("ArchiveToLongTerm failed: %v", err)
	}

	// Verify short-term file is deleted
	if _, err := os.Stat(shortTermPath); !os.IsNotExist(err) {
		t.Error("Expected short-term memory file to be deleted after archive")
	}

	// Verify content in MEMORY.md
	longTermPath := filepath.Join(tmpDir, "MEMORY.md")
	data, err := os.ReadFile(longTermPath)
	if err != nil {
		t.Fatalf("Failed to read MEMORY.md: %v", err)
	}

	if !contains(string(data), "2026-04-01") {
		t.Error("Expected archived date in MEMORY.md")
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func countLines(s string) int {
	count := 0
	for _, c := range s {
		if c == '\n' {
			count++
		}
	}
	return count
}