package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ReadInput struct {
	Path     string `json:"path"`
	IsDir    bool   `json:"is_dir,omitempty"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	ShowTime bool   `json:"show_time,omitempty"`
}

type ReadTool struct {
	baseDir string
}

func NewReadTool(baseDir string) *ReadTool {
	return &ReadTool{baseDir: baseDir}
}

func (t *ReadTool) Name() string {
	return "read"
}

func (t *ReadTool) Description() string {
	return "Read file or directory. For directories, returns tree structure respecting .gitignore. For files, returns content with line numbers (max 100 lines)."
}

func (t *ReadTool) Execute(ctx context.Context, input string) (string, error) {
	var req ReadInput
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}

	path := filepath.Join(t.baseDir, req.Path)
	path = filepath.Clean(path)

	if !validatePath(t.baseDir, path) {
		return "", fmt.Errorf("path must be within base directory")
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat path: %w", err)
	}

	if info.IsDir() {
		return t.readDirectory(path, req.ShowTime)
	}
	return t.readFile(path, req)
}

func (t *ReadTool) readDirectory(path string, showTime bool) (string, error) {
	ignorePatterns := t.loadGitignore(path)
	var result strings.Builder
	result.WriteString(path + "/\n")
	t.walkDir(path, "", ignorePatterns, showTime, &result)
	return result.String(), nil
}

func (t *ReadTool) loadGitignore(dir string) []string {
	gitignorePath := filepath.Join(dir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return nil
	}

	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

func (t *ReadTool) shouldIgnore(name string, patterns []string) bool {
	defaultIgnores := []string{
		".git", ".svn", ".hg",
		"node_modules", "vendor",
		"bin", "dist", "build",
		"*.exe", "*.dll", "*.so", "*.dylib",
		"*.test", "*.out",
		"*.pyc", "__pycache__",
		".DS_Store", "Thumbs.db",
		"*.min.js", "*.min.css",
		"go.sum",
	}

	allPatterns := append(defaultIgnores, patterns...)

	for _, pattern := range allPatterns {
		if strings.HasPrefix(pattern, "*.") {
			ext := pattern[1:]
			if strings.HasSuffix(name, ext) {
				return true
			}
		}
		if name == pattern {
			return true
		}
	}
	return false
}

func (t *ReadTool) walkDir(path, prefix string, patterns []string, showTime bool, result *strings.Builder) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for i, entry := range entries {
		if t.shouldIgnore(entry.Name(), patterns) {
			continue
		}

		isLast := i == len(entries)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		fullPath := filepath.Join(path, entry.Name())
		var timeStr string
		if showTime {
			if info, err := entry.Info(); err == nil {
				timeStr = fmt.Sprintf(" [%s]", info.ModTime().Format("2006-01-02 15:04"))
			}
		}

		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("%s%s/%s\n", prefix+connector, entry.Name(), timeStr))
			newPrefix := prefix
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
			t.walkDir(fullPath, newPrefix, patterns, showTime, result)
		} else {
			result.WriteString(fmt.Sprintf("%s%s%s\n", prefix+connector, entry.Name(), timeStr))
		}
	}
}

func (t *ReadTool) readFile(path string, req ReadInput) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat file: %w", err)
	}

	if info.Size() > 10*1024*1024 {
		return "", fmt.Errorf("file too large (max 10MB)")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	offset := req.Offset
	if offset < 1 {
		offset = 1
	}
	if offset > totalLines {
		offset = totalLines
	}

	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	end := offset + limit - 1
	if end > totalLines {
		end = totalLines
	}

	var result strings.Builder

	if req.ShowTime {
		result.WriteString(fmt.Sprintf("// File: %s\n", path))
		result.WriteString(fmt.Sprintf("// Size: %d bytes\n", info.Size()))
		result.WriteString(fmt.Sprintf("// Modified: %s\n", info.ModTime().Format(time.RFC3339)))
		result.WriteString(fmt.Sprintf("// Lines: %d (showing %d-%d)\n\n", totalLines, offset, end))
	}

	for i := offset - 1; i < end && i < len(lines); i++ {
		result.WriteString(fmt.Sprintf("%d: %s\n", i+1, lines[i]))
	}

	if end < totalLines {
		result.WriteString(fmt.Sprintf("\n... (%d more lines)\n", totalLines-end))
	}

	return result.String(), nil
}
