package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"penframe/internal/domain"
)

func LoadToolCatalog(path string) (map[string]domain.ToolDefinition, error) {
	var catalog domain.ToolCatalog
	if err := loadYAML(path, &catalog); err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(path)
	for name, tool := range catalog.Tools {
		tool.Name = name
		tool.Parser = resolveRelativePath(baseDir, tool.Parser)
		catalog.Tools[name] = tool
	}
	return catalog.Tools, nil
}

func LoadWorkflow(path string) (domain.Workflow, error) {
	var workflow domain.Workflow
	if err := loadYAML(path, &workflow); err != nil {
		return domain.Workflow{}, err
	}
	baseDir := filepath.Dir(path)
	for idx, node := range workflow.Nodes {
		node.Mock.StdoutFile = resolveRelativePath(baseDir, node.Mock.StdoutFile)
		workflow.Nodes[idx] = node
	}
	return workflow, nil
}

func LoadParserRuleSet(path string) (domain.ParserRuleSet, error) {
	var ruleSet domain.ParserRuleSet
	if err := loadYAML(path, &ruleSet); err != nil {
		return domain.ParserRuleSet{}, err
	}
	return ruleSet, nil
}

func loadYAML(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func resolveRelativePath(baseDir, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(filepath.Join(baseDir, path))
}
