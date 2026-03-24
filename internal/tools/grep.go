package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type GrepInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
	Ext     string `json:"ext,omitempty"`
	MaxLine int    `json:"max_line,omitempty"`
}

type GrepResult struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

type GrepTool struct {
	baseDir string
}

func NewGrepTool(baseDir string) *GrepTool {
	return &GrepTool{baseDir: baseDir}
}

func (t *GrepTool) Name() string {
	return "grep"
}

func (t *GrepTool) Description() string {
	return "Search for pattern in files. Respects .gitignore, ignores binary/compiled files, filters very long lines."
}

func (t *GrepTool) Execute(ctx context.Context, input string) (string, error) {
	var req GrepInput
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}

	searchPath := t.baseDir
	if req.Path != "" {
		searchPath = filepath.Join(t.baseDir, req.Path)
	}
	searchPath = filepath.Clean(searchPath)

	if !strings.HasPrefix(searchPath, t.baseDir) {
		return "", fmt.Errorf("path must be within base directory")
	}

	maxLineLen := req.MaxLine
	if maxLineLen <= 0 {
		maxLineLen = 500
	}

	re, err := regexp.Compile(req.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid pattern: %w", err)
	}

	var results []GrepResult
	ignorePatterns := t.loadGitignore(searchPath)

	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if t.shouldIgnore(filepath.Base(path), ignorePatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		if t.shouldIgnore(filepath.Base(path), ignorePatterns) {
			return nil
		}

		if req.Ext != "" && !strings.HasSuffix(path, req.Ext) {
			return nil
		}

		matches, err := t.grepFile(path, re, maxLineLen)
		if err != nil {
			return nil
		}
		results = append(results, matches...)
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("walk directory: %w", err)
	}

	return t.formatResults(results), nil
}

func (t *GrepTool) loadGitignore(dir string) []string {
	gitignorePath := filepath.Join(t.baseDir, ".gitignore")
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

func (t *GrepTool) shouldIgnore(name string, patterns []string) bool {
	defaultIgnores := []string{
		".git", ".svn", ".hg",
		"node_modules", "vendor",
		"bin", "dist", "build",
		"*.exe", "*.dll", "*.so", "*.dylib",
		"*.test", "*.out", "*.o", "*.a",
		"*.pyc", "__pycache__",
		".DS_Store", "Thumbs.db",
		"*.min.js", "*.min.css",
		"*.svg", "*.png", "*.jpg", "*.jpeg", "*.gif", "*.ico",
		"*.pdf", "*.zip", "*.tar", "*.gz",
		"go.sum", "package-lock.json", "yarn.lock",
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

func (t *GrepTool) grepFile(path string, re *regexp.Regexp, maxLineLen int) ([]GrepResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	relPath, _ := filepath.Rel(t.baseDir, path)

	var results []GrepResult
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if len(line) > maxLineLen {
			continue
		}

		if t.isBinaryLine(line) {
			return nil, nil
		}

		if re.MatchString(line) {
			results = append(results, GrepResult{
				File:    relPath,
				Line:    lineNum,
				Content: line,
			})
		}
	}

	return results, scanner.Err()
}

func (t *GrepTool) isBinaryLine(line string) bool {
	nonPrintable := 0
	for _, r := range line {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			nonPrintable++
		}
	}
	return nonPrintable > 0
}

func (t *GrepTool) formatResults(results []GrepResult) string {
	if len(results) == 0 {
		return "No matches found.\n"
	}

	var sb strings.Builder
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("%s:%d: %s\n", r.File, r.Line, r.Content))
	}
	sb.WriteString(fmt.Sprintf("\nFound %d match(es) in %d file(s)\n",
		len(results), t.countUniqueFiles(results)))
	return sb.String()
}

func (t *GrepTool) countUniqueFiles(results []GrepResult) int {
	files := make(map[string]bool)
	for _, r := range results {
		files[r.File] = true
	}
	return len(files)
}
