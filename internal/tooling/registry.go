package tooling

import (
	"fmt"

	"penframe/internal/domain"
)

type Registry struct {
	tools map[string]domain.ToolDefinition
}

func NewRegistry(tools map[string]domain.ToolDefinition) *Registry {
	cloned := make(map[string]domain.ToolDefinition, len(tools))
	for name, tool := range tools {
		cloned[name] = tool
	}
	return &Registry{tools: cloned}
}

func (r *Registry) Get(name string) (domain.ToolDefinition, error) {
	tool, ok := r.tools[name]
	if !ok {
		return domain.ToolDefinition{}, fmt.Errorf("tool %q is not registered", name)
	}
	return tool, nil
}
