package tools

import (
	"context"
	"path/filepath"
	"strings"
)

type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input string) (string, error)
}

func validatePath(baseDir, path string) bool {
	cleanBase := filepath.Clean(baseDir)
	cleanPath := filepath.Clean(path)
	if cleanPath == cleanBase {
		return true
	}
	if !strings.HasSuffix(cleanBase, string(filepath.Separator)) {
		cleanBase += string(filepath.Separator)
	}
	return strings.HasPrefix(cleanPath+string(filepath.Separator), cleanBase)
}

type ToolRegistry struct {
	tools map[string]Tool
}

func NewRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *ToolRegistry) List() []Tool {
	result := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}
