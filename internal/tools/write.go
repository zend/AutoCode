package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type WriteInput struct {
	Path          string `json:"path"`
	OldString     string `json:"old_string"`
	NewString     string `json:"new_string"`
	ExpectedCount int    `json:"expected_count,omitempty"`
	ModTime       string `json:"mod_time,omitempty"`
	Create        bool   `json:"create,omitempty"`
	Content       string `json:"content,omitempty"`
}

type WriteTool struct {
	baseDir string
}

func NewWriteTool(baseDir string) *WriteTool {
	return &WriteTool{baseDir: baseDir}
}

func (t *WriteTool) Name() string {
	return "write"
}

func (t *WriteTool) Description() string {
	return "Write or edit files. For editing: requires exact old_string match and validates mod_time. Runs lint before confirming success."
}

func (t *WriteTool) Execute(ctx context.Context, input string) (string, error) {
	var req WriteInput
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}

	path := filepath.Join(t.baseDir, req.Path)
	path = filepath.Clean(path)

	if !strings.HasPrefix(path, t.baseDir) {
		return "", fmt.Errorf("path must be within base directory")
	}

	if req.Create && req.Content != "" {
		return t.createFile(path, req)
	}

	return t.editFile(path, req)
}

func (t *WriteTool) createFile(path string, req WriteInput) (string, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("file already exists: %s", path)
	}

	if err := os.WriteFile(path, []byte(req.Content), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	if err := t.runLint(path); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("lint failed, file removed: %w", err)
	}

	info, _ := os.Stat(path)
	return fmt.Sprintf("Created: %s (%d bytes)\n", path, info.Size()), nil
}

func (t *WriteTool) editFile(path string, req WriteInput) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat file: %w", err)
	}

	if req.ModTime != "" {
		expectedTime, err := time.Parse(time.RFC3339, req.ModTime)
		if err != nil {
			return "", fmt.Errorf("parse mod_time: %w", err)
		}
		if !info.ModTime().Equal(expectedTime) {
			return "", fmt.Errorf("file modified since reading (expected %s, got %s)",
				expectedTime.Format(time.RFC3339), info.ModTime().Format(time.RFC3339))
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	content := string(data)

	count := strings.Count(content, req.OldString)
	if count == 0 {
		return "", fmt.Errorf("old_string not found in file")
	}

	if req.ExpectedCount > 0 && count != req.ExpectedCount {
		return "", fmt.Errorf("expected %d occurrences, found %d", req.ExpectedCount, count)
	}

	if count > 1 && req.ExpectedCount == 0 {
		return "", fmt.Errorf("old_string appears %d times, please provide more context or set expected_count", count)
	}

	newContent := strings.Replace(content, req.OldString, req.NewString, 1)
	if count > 1 {
		newContent = strings.Replace(content, req.OldString, req.NewString, count)
	}

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	if err := t.runLint(path); err != nil {
		if err := os.WriteFile(path, data, 0644); err != nil {
			return "", fmt.Errorf("lint failed and rollback failed: %w", err)
		}
		return "", fmt.Errorf("lint failed, changes reverted: %w", err)
	}

	newInfo, _ := os.Stat(path)
	return fmt.Sprintf("Edited: %s (%d bytes -> %d bytes, %d replacement(s))\n",
		path, len(data), newInfo.Size(), count), nil
}

func (t *WriteTool) runLint(path string) error {
	ext := filepath.Ext(path)

	switch ext {
	case ".go":
		return t.runGoLint(path)
	case ".js", ".ts", ".jsx", ".tsx":
		return t.runJSLint(path)
	}
	return nil
}

func (t *WriteTool) runGoLint(path string) error {
	cmd := exec.Command("go", "fmt", path)
	cmd.Dir = t.baseDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go fmt: %w\n%s", err, output)
	}

	cmd = exec.Command("go", "vet", path)
	cmd.Dir = t.baseDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go vet: %w\n%s", err, output)
	}

	return nil
}

func (t *WriteTool) runJSLint(path string) error {
	return nil
}
