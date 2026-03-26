package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadToolCatalogResolvesParserPathsRelativeToCatalog(t *testing.T) {
	root := t.TempDir()
	catalogDir := filepath.Join(root, "catalog")
	parserPath := filepath.Join(catalogDir, "parsers", "demo.yaml")

	if err := os.MkdirAll(filepath.Dir(parserPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(parserPath, []byte("tool: demo\nrules: []\n"), 0o644); err != nil {
		t.Fatalf("WriteFile parser returned error: %v", err)
	}

	catalogPath := filepath.Join(catalogDir, "tools.yaml")
	catalogYAML := []byte("tools:\n  demo:\n    category: discovery\n    parser: parsers/demo.yaml\n")
	if err := os.WriteFile(catalogPath, catalogYAML, 0o644); err != nil {
		t.Fatalf("WriteFile catalog returned error: %v", err)
	}

	tools, err := LoadToolCatalog(catalogPath)
	if err != nil {
		t.Fatalf("LoadToolCatalog returned error: %v", err)
	}

	if got := tools["demo"].Parser; got != parserPath {
		t.Fatalf("expected parser path %q, got %q", parserPath, got)
	}
}

func TestLoadWorkflowResolvesMockFixturePathsRelativeToWorkflow(t *testing.T) {
	root := t.TempDir()
	workflowDir := filepath.Join(root, "workflow")
	fixturePath := filepath.Join(workflowDir, "fixtures", "stdout.txt")

	if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(fixturePath, []byte("fixture output"), 0o644); err != nil {
		t.Fatalf("WriteFile fixture returned error: %v", err)
	}

	workflowPath := filepath.Join(workflowDir, "workflow.yaml")
	workflowYAML := []byte("name: test-workflow\nnodes:\n  - id: mock-node\n    tool: demo\n    executor: mock\n    mock:\n      stdout_file: fixtures/stdout.txt\n")
	if err := os.WriteFile(workflowPath, workflowYAML, 0o644); err != nil {
		t.Fatalf("WriteFile workflow returned error: %v", err)
	}

	wf, err := LoadWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("LoadWorkflow returned error: %v", err)
	}

	if len(wf.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(wf.Nodes))
	}
	if got := wf.Nodes[0].Mock.StdoutFile; got != fixturePath {
		t.Fatalf("expected fixture path %q, got %q", fixturePath, got)
	}
}
