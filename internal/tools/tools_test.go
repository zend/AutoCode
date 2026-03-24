package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadTool_File(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	tool := NewReadTool(tmpDir)

	result, err := tool.Execute(context.Background(), `{"path": "test.txt"}`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !contains(result, "line1") || !contains(result, "line5") {
		t.Errorf("expected file content in result, got: %s", result)
	}
}

func TestReadTool_Offset(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	tool := NewReadTool(tmpDir)

	result, err := tool.Execute(context.Background(), `{"path": "test.txt", "offset": 2, "limit": 2}`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if contains(result, "line1") {
		t.Error("should not contain line1")
	}
	if !contains(result, "line2") || !contains(result, "line3") {
		t.Errorf("expected line2 and line3, got: %s", result)
	}
}

func TestReadTool_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	tool := NewReadTool(tmpDir)

	result, err := tool.Execute(context.Background(), `{"path": ".", "is_dir": true}`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !contains(result, "file.txt") || !contains(result, "subdir") {
		t.Errorf("expected directory listing, got: %s", result)
	}
}

func TestReadTool_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewReadTool(tmpDir)

	_, err := tool.Execute(context.Background(), `{"path": "../etc/passwd"}`)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestWriteTool_Create(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewWriteTool(tmpDir)

	result, err := tool.Execute(context.Background(), `{"path": "new.txt", "create": true, "content": "hello world"}`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !contains(result, "Created") {
		t.Errorf("expected created message, got: %s", result)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "new.txt"))
	if err != nil {
		t.Fatalf("read created file: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got: %s", data)
	}
}

func TestWriteTool_Edit(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "edit.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	tool := NewWriteTool(tmpDir)

	result, err := tool.Execute(context.Background(), `{"path": "edit.txt", "old_string": "world", "new_string": "golang"}`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !contains(result, "Edited") {
		t.Errorf("expected edited message, got: %s", result)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("read edited file: %v", err)
	}
	if string(data) != "hello golang" {
		t.Errorf("expected 'hello golang', got: %s", data)
	}
}

func TestWriteTool_MultipleOccurrences(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "multi.txt")
	if err := os.WriteFile(testFile, []byte("foo bar foo"), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	tool := NewWriteTool(tmpDir)

	_, err := tool.Execute(context.Background(), `{"path": "multi.txt", "old_string": "foo", "new_string": "baz"}`)
	if err == nil {
		t.Error("expected error for multiple occurrences")
	}
}

func TestGrepTool_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world\nfoo bar\nhello again"), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	tool := NewGrepTool(tmpDir)

	result, err := tool.Execute(context.Background(), `{"pattern": "hello"}`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !contains(result, "test.txt:1:") || !contains(result, "test.txt:3:") {
		t.Errorf("expected matches on lines 1 and 3, got: %s", result)
	}
}

func TestGrepTool_Extension(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("func main()"), 0644); err != nil {
		t.Fatalf("write go file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("func other()"), 0644); err != nil {
		t.Fatalf("write txt file: %v", err)
	}

	tool := NewGrepTool(tmpDir)

	result, err := tool.Execute(context.Background(), `{"pattern": "func", "ext": ".go"}`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if contains(result, "test.txt") {
		t.Error("should not match txt file")
	}
	if !contains(result, "test.go") {
		t.Error("should match go file")
	}
}

func TestShellTool_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewShellTool(tmpDir)

	result, err := tool.Execute(context.Background(), `{"command": "echo hello"}`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !contains(result, "hello") {
		t.Errorf("expected 'hello' in output, got: %s", result)
	}
}

func TestShellTool_WorkDir(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}

	tool := NewShellTool(tmpDir)

	result, err := tool.Execute(context.Background(), `{"command": "pwd", "work_dir": "sub"}`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !contains(result, "sub") {
		t.Errorf("expected working directory 'sub', got: %s", result)
	}
}

func TestShellTool_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewShellTool(tmpDir)

	result, err := tool.Execute(context.Background(), `{"command": "sleep 0.1", "timeout": 1}`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !contains(result, "Exit Code: 0") {
		t.Errorf("expected exit code 0, got: %s", result)
	}
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()
	tool := NewReadTool("/tmp")
	registry.Register(tool)

	if _, ok := registry.Get("read"); !ok {
		t.Error("expected to find read tool")
	}

	list := registry.List()
	if len(list) != 1 {
		t.Errorf("expected 1 tool, got %d", len(list))
	}
}

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
