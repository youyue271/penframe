package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"penframe/internal/domain"
	"penframe/internal/executor"
	"penframe/internal/parser"
	"penframe/internal/tooling"
)

type recordingExecutor struct {
	inputsByNode map[string]map[string]any
}

func newRecordingExecutor() *recordingExecutor {
	return &recordingExecutor{
		inputsByNode: map[string]map[string]any{},
	}
}

func (e *recordingExecutor) Name() string {
	return "recording"
}

func (e *recordingExecutor) Execute(_ context.Context, node domain.WorkflowNode, _ domain.ToolDefinition, renderedInputs map[string]any, _ string) (domain.ExecutionResult, error) {
	clonedInputs := make(map[string]any, len(renderedInputs))
	for key, value := range renderedInputs {
		clonedInputs[key] = value
	}
	e.inputsByNode[node.ID] = clonedInputs
	return domain.ExecutionResult{
		Stdout: node.ID,
	}, nil
}

func TestRunnerWaitsForAllIncomingEdgesBeforeSchedulingNode(t *testing.T) {
	execImpl := newRecordingExecutor()
	runner := NewRunner(
		tooling.NewRegistry(map[string]domain.ToolDefinition{
			"scan_a":    {Name: "scan_a"},
			"scan_b":    {Name: "scan_b"},
			"correlate": {Name: "correlate"},
		}),
		executor.NewRegistry(execImpl),
		parser.NewEngine(),
		NewMiniExprEvaluator(),
	)

	wf := domain.Workflow{
		Name: "join-workflow",
		Nodes: []domain.WorkflowNode{
			{ID: "scan_a", Tool: "scan_a", Executor: "recording"},
			{ID: "scan_b", Tool: "scan_b", Executor: "recording"},
			{
				ID:       "correlate",
				Tool:     "correlate",
				Executor: "recording",
				Inputs: map[string]any{
					"a_status": "{{ .results.scan_a.status }}",
					"b_status": "{{ .results.scan_b.status }}",
				},
			},
		},
		Edges: []domain.WorkflowEdge{
			{From: "scan_a", To: "correlate"},
			{From: "scan_b", To: "correlate"},
		},
	}

	summary, err := runner.Run(context.Background(), wf)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(summary.ExecutionOrder) != 3 {
		t.Fatalf("expected 3 executed nodes, got %d", len(summary.ExecutionOrder))
	}
	if summary.ExecutionOrder[2] != "correlate" {
		t.Fatalf("expected correlate to execute last, got order %v", summary.ExecutionOrder)
	}

	correlateInputs := execImpl.inputsByNode["correlate"]
	if correlateInputs["a_status"] != domain.NodeStatusSucceeded {
		t.Fatalf("expected correlate to receive scan_a status, got %#v", correlateInputs["a_status"])
	}
	if correlateInputs["b_status"] != domain.NodeStatusSucceeded {
		t.Fatalf("expected correlate to receive scan_b status, got %#v", correlateInputs["b_status"])
	}
}

func TestRunnerRejectsCyclesHiddenBehindEntryNodes(t *testing.T) {
	runner := NewRunner(
		tooling.NewRegistry(map[string]domain.ToolDefinition{
			"noop": {Name: "noop"},
		}),
		executor.NewRegistry(newRecordingExecutor()),
		parser.NewEngine(),
		NewMiniExprEvaluator(),
	)

	wf := domain.Workflow{
		Name: "cyclic-workflow",
		Nodes: []domain.WorkflowNode{
			{ID: "start", Tool: "noop", Executor: "recording"},
			{ID: "b", Tool: "noop", Executor: "recording"},
			{ID: "c", Tool: "noop", Executor: "recording"},
		},
		Edges: []domain.WorkflowEdge{
			{From: "b", To: "c"},
			{From: "c", To: "b"},
		},
	}

	_, err := runner.Run(context.Background(), wf)
	if err == nil {
		t.Fatal("expected Run to reject cyclic workflow")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestRunnerMarksBlockedNodesAsSkipped(t *testing.T) {
	execImpl := newRecordingExecutor()
	runner := NewRunner(
		tooling.NewRegistry(map[string]domain.ToolDefinition{
			"seed":  {Name: "seed"},
			"child": {Name: "child"},
		}),
		executor.NewRegistry(execImpl),
		parser.NewEngine(),
		NewMiniExprEvaluator(),
	)

	wf := domain.Workflow{
		Name: "skip-workflow",
		Nodes: []domain.WorkflowNode{
			{ID: "seed", Tool: "seed", Executor: "recording"},
			{ID: "child", Tool: "child", Executor: "recording"},
		},
		Edges: []domain.WorkflowEdge{
			{From: "seed", To: "child", Condition: "false"},
		},
	}

	summary, err := runner.Run(context.Background(), wf)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if summary.Status != domain.RunStatusSucceeded {
		t.Fatalf("expected run status %q, got %q", domain.RunStatusSucceeded, summary.Status)
	}
	child, ok := summary.NodeResults["child"]
	if !ok {
		t.Fatal("expected skipped child node result to be recorded")
	}
	if child.Status != domain.NodeStatusSkipped {
		t.Fatalf("expected child status %q, got %q", domain.NodeStatusSkipped, child.Status)
	}
	if child.SkipReason == "" {
		t.Fatal("expected child skip reason to be populated")
	}
	if summary.Stats.SkippedNodes != 1 {
		t.Fatalf("expected 1 skipped node, got %d", summary.Stats.SkippedNodes)
	}
	if len(summary.ExecutionOrder) != 1 || summary.ExecutionOrder[0] != "seed" {
		t.Fatalf("expected only seed to execute, got %v", summary.ExecutionOrder)
	}
}

func TestRunnerReturnsFailedSummaryWithNodeError(t *testing.T) {
	failingExec := failingExecutor{err: fmt.Errorf("boom")}
	runner := NewRunner(
		tooling.NewRegistry(map[string]domain.ToolDefinition{
			"fragile": {Name: "fragile"},
		}),
		executor.NewRegistry(failingExec),
		parser.NewEngine(),
		NewMiniExprEvaluator(),
	)

	wf := domain.Workflow{
		Name: "failure-workflow",
		Nodes: []domain.WorkflowNode{
			{ID: "fragile-node", Tool: "fragile", Executor: "failing"},
		},
	}

	summary, err := runner.Run(context.Background(), wf)
	if err == nil {
		t.Fatal("expected Run to return an error")
	}
	if summary.Status != domain.RunStatusFailed {
		t.Fatalf("expected run status %q, got %q", domain.RunStatusFailed, summary.Status)
	}
	nodeResult, ok := summary.NodeResults["fragile-node"]
	if !ok {
		t.Fatal("expected failed node result to be recorded")
	}
	if nodeResult.Status != domain.NodeStatusFailed {
		t.Fatalf("expected node status %q, got %q", domain.NodeStatusFailed, nodeResult.Status)
	}
	if !strings.Contains(nodeResult.Error, "boom") {
		t.Fatalf("expected node error to contain boom, got %q", nodeResult.Error)
	}
	if summary.Stats.FailedNodes != 1 {
		t.Fatalf("expected 1 failed node, got %d", summary.Stats.FailedNodes)
	}
	if len(summary.ExecutionOrder) != 1 || summary.ExecutionOrder[0] != "fragile-node" {
		t.Fatalf("expected failed node to appear in execution order, got %v", summary.ExecutionOrder)
	}
}

func TestPrepareRunVarsCreatesOutputDirectoryByTarget(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	tempWD := t.TempDir()
	if err := os.Chdir(tempWD); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	vars, err := prepareRunVars(map[string]any{
		"target_host": "example.com:3000",
	})
	if err != nil {
		t.Fatalf("prepareRunVars returned error: %v", err)
	}

	outputRoot, _ := vars["output_root"].(string)
	expectedRoot := filepath.Join(tempWD, "output")
	if outputRoot != expectedRoot {
		t.Fatalf("expected output_root %q, got %q", expectedRoot, outputRoot)
	}

	outputDir, _ := vars["output_dir"].(string)
	expectedDir := filepath.Join(expectedRoot, "example.com_3000")
	if outputDir != expectedDir {
		t.Fatalf("expected output_dir %q, got %q", expectedDir, outputDir)
	}
	if info, err := os.Stat(outputDir); err != nil || !info.IsDir() {
		t.Fatalf("expected output directory to exist, err=%v", err)
	}
}

func TestParserInputIncludesOutputFileContent(t *testing.T) {
	result := domain.ExecutionResult{
		Stdout: "first-line",
		Metadata: map[string]any{
			"output_files": []map[string]any{
				{"path": "/tmp/a.txt", "content": "second-line"},
				{"path": "/tmp/b.txt", "content": "third-line"},
			},
		},
	}

	got := parserInput(result)
	if !strings.Contains(got, "first-line") {
		t.Fatalf("expected parser input to include stdout, got %q", got)
	}
	if !strings.Contains(got, "second-line") || !strings.Contains(got, "third-line") {
		t.Fatalf("expected parser input to include output file contents, got %q", got)
	}
}

func TestOutputFileContentsSupportsUntypedArray(t *testing.T) {
	metadata := map[string]any{
		"output_files": []any{
			map[string]any{"content": "one"},
			map[string]any{"content": "two"},
		},
	}

	got := outputFileContents(metadata)
	if len(got) != 2 {
		t.Fatalf("expected 2 contents, got %d", len(got))
	}
	if got[0] != "one" || got[1] != "two" {
		t.Fatalf("unexpected contents: %#v", got)
	}
}

type failingExecutor struct {
	err error
}

func (failingExecutor) Name() string {
	return "failing"
}

func (e failingExecutor) Execute(_ context.Context, _ domain.WorkflowNode, _ domain.ToolDefinition, _ map[string]any, _ string) (domain.ExecutionResult, error) {
	return domain.ExecutionResult{}, e.err
}
