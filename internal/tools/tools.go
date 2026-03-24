package tools

import (
	"context"
)

type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input string) (string, error)
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
