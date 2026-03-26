package workflow

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"penframe/internal/config"
	"penframe/internal/domain"
	"penframe/internal/executor"
	"penframe/internal/parser"
	runtimepkg "penframe/internal/runtime"
	"penframe/internal/tooling"
)

type Runner struct {
	tools         *tooling.Registry
	executors     *executor.Registry
	parser        *parser.Engine
	evaluator     Evaluator
	parserCacheMu sync.RWMutex
	parserCache   map[string]domain.ParserRuleSet
}

func NewRunner(tools *tooling.Registry, executors *executor.Registry, parserEngine *parser.Engine, evaluator Evaluator) *Runner {
	return &Runner{
		tools:       tools,
		executors:   executors,
		parser:      parserEngine,
		evaluator:   evaluator,
		parserCache: make(map[string]domain.ParserRuleSet),
	}
}

func (r *Runner) Run(ctx context.Context, wf domain.Workflow) (domain.RunSummary, error) {
	runVars, err := prepareRunVars(wf.GlobalVars)
	if err != nil {
		return domain.RunSummary{}, fmt.Errorf("prepare run vars: %w", err)
	}

	startedAt := time.Now().UTC()
	summary := domain.RunSummary{
		Workflow:    wf.Name,
		Status:      domain.RunStatusRunning,
		StartedAt:   startedAt,
		Vars:        runVars,
		Assets:      map[string]any{},
		NodeResults: make(map[string]domain.NodeRunResult, len(wf.Nodes)),
		Stats: domain.RunStats{
			TotalNodes: len(wf.Nodes),
		},
	}
	nodes := make(map[string]domain.WorkflowNode, len(wf.Nodes))
	incoming := make(map[string]int, len(wf.Nodes))
	adjacency := make(map[string][]domain.WorkflowEdge)

	for _, node := range wf.Nodes {
		if node.ID == "" {
			return finishSummaryWithError(summary, fmt.Errorf("workflow contains a node with empty id"))
		}
		if _, exists := nodes[node.ID]; exists {
			return finishSummaryWithError(summary, fmt.Errorf("workflow contains duplicate node id %q", node.ID))
		}
		nodes[node.ID] = node
		incoming[node.ID] = 0
	}
	for _, edge := range wf.Edges {
		if _, ok := nodes[edge.From]; !ok {
			return finishSummaryWithError(summary, fmt.Errorf("edge references unknown source node %q", edge.From))
		}
		if _, ok := nodes[edge.To]; !ok {
			return finishSummaryWithError(summary, fmt.Errorf("edge references unknown target node %q", edge.To))
		}
		adjacency[edge.From] = append(adjacency[edge.From], edge)
		incoming[edge.To]++
	}
	if err := r.validateWorkflowDependencies(nodes); err != nil {
		return finishSummaryWithError(summary, err)
	}
	if err := validateAcyclic(wf.Name, incoming, adjacency); err != nil {
		return finishSummaryWithError(summary, err)
	}

	resultEnv := make(map[string]any, len(wf.Nodes))

	remainingIncoming := cloneIntMap(incoming)
	blocked := make(map[string]bool, len(wf.Nodes))
	blockedReasons := make(map[string]string, len(wf.Nodes))
	queue := make([]string, 0, len(wf.Nodes))
	for nodeID, count := range remainingIncoming {
		if count == 0 {
			queue = append(queue, nodeID)
		}
	}

	resolved := make(map[string]bool, len(wf.Nodes))
	recordNodeResult := func(nodeResult domain.NodeRunResult, appendExecution bool) {
		summary.NodeResults[nodeResult.NodeID] = nodeResult
		if appendExecution {
			summary.ExecutionOrder = append(summary.ExecutionOrder, nodeResult.NodeID)
		}
		resultEnv[nodeResult.NodeID] = buildResultEnv(nodeResult)
		resolved[nodeResult.NodeID] = true
		refreshRunStats(&summary)
	}
	skipNode := func(nodeID, reason string) {
		if _, exists := summary.NodeResults[nodeID]; exists {
			return
		}
		node := nodes[nodeID]
		now := time.Now().UTC()
		recordNodeResult(domain.NodeRunResult{
			NodeID:         node.ID,
			Tool:           node.Tool,
			Executor:       node.Executor,
			Status:         domain.NodeStatusSkipped,
			SkipReason:     reason,
			StartedAt:      now,
			FinishedAt:     now,
			DurationMillis: 0,
		}, false)
	}
	propagateResolution := func(sourceID string, sourceExecuted bool) error {
		resolveQueue := []string{sourceID}
		for len(resolveQueue) > 0 {
			currentID := resolveQueue[0]
			resolveQueue = resolveQueue[1:]

			for _, edge := range adjacency[currentID] {
				allow := false
				blockReason := blockedReasons[edge.To]
				if sourceExecuted && currentID == sourceID {
					var err error
					allow, err = r.evaluator.Evaluate(edge.Condition, map[string]any{
						"vars":    summary.Vars,
						"assets":  summary.Assets,
						"results": resultEnv,
					})
					if err != nil {
						return fmt.Errorf("evaluate edge %s -> %s: %w", edge.From, edge.To, err)
					}
					if !allow && blockReason == "" {
						blockReason = fmt.Sprintf("incoming edge %q -> %q did not satisfy its condition", edge.From, edge.To)
					}
				} else if blockReason == "" {
					blockReason = fmt.Sprintf("upstream node %q was skipped", edge.From)
				}
				if !allow {
					blocked[edge.To] = true
					blockedReasons[edge.To] = blockReason
				}

				remainingIncoming[edge.To]--
				if remainingIncoming[edge.To] > 0 {
					continue
				}
				if !blocked[edge.To] {
					queue = append(queue, edge.To)
					continue
				}
				if resolved[edge.To] {
					continue
				}
				skipNode(edge.To, blockedReasons[edge.To])
				resolveQueue = append(resolveQueue, edge.To)
			}

			sourceExecuted = false
		}
		return nil
	}

	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		if resolved[nodeID] {
			continue
		}
		node := nodes[nodeID]
		nodeStartedAt := time.Now().UTC()

		tool, err := r.tools.Get(node.Tool)
		if err != nil {
			recordNodeResult(domain.NodeRunResult{
				NodeID:         node.ID,
				Tool:           node.Tool,
				Executor:       node.Executor,
				Status:         domain.NodeStatusFailed,
				Error:          err.Error(),
				StartedAt:      nodeStartedAt,
				FinishedAt:     time.Now().UTC(),
				DurationMillis: time.Since(nodeStartedAt).Milliseconds(),
			}, true)
			return finishSummaryWithError(summary, fmt.Errorf("resolve tool for node %q: %w", node.ID, err))
		}
		execImpl, err := r.executors.Get(node.Executor)
		if err != nil {
			recordNodeResult(domain.NodeRunResult{
				NodeID:         node.ID,
				Tool:           node.Tool,
				Executor:       node.Executor,
				Status:         domain.NodeStatusFailed,
				Error:          err.Error(),
				StartedAt:      nodeStartedAt,
				FinishedAt:     time.Now().UTC(),
				DurationMillis: time.Since(nodeStartedAt).Milliseconds(),
			}, true)
			return finishSummaryWithError(summary, fmt.Errorf("resolve executor for node %q: %w", node.ID, err))
		}

		templateCtx := map[string]any{
			"vars":    summary.Vars,
			"assets":  summary.Assets,
			"results": resultEnv,
			"node": map[string]any{
				"id":       node.ID,
				"tool":     node.Tool,
				"executor": node.Executor,
			},
		}

		renderedInputs, err := renderInputs(node.Inputs, templateCtx)
		if err != nil {
			nodeFinishedAt := time.Now().UTC()
			recordNodeResult(domain.NodeRunResult{
				NodeID:         node.ID,
				Tool:           node.Tool,
				Executor:       node.Executor,
				Status:         domain.NodeStatusFailed,
				Inputs:         map[string]any{},
				Error:          fmt.Sprintf("render inputs: %v", err),
				StartedAt:      nodeStartedAt,
				FinishedAt:     nodeFinishedAt,
				DurationMillis: nodeFinishedAt.Sub(nodeStartedAt).Milliseconds(),
			}, true)
			return finishSummaryWithError(summary, fmt.Errorf("render inputs for node %q: %w", node.ID, err))
		}

		var renderedCommand string
		if tool.CmdTemplate != "" {
			commandCtx := cloneMap(templateCtx)
			commandCtx["inputs"] = renderedInputs
			renderedCommand, err = runtimepkg.RenderString(tool.CmdTemplate, commandCtx)
			if err != nil {
				nodeFinishedAt := time.Now().UTC()
				recordNodeResult(domain.NodeRunResult{
					NodeID:          node.ID,
					Tool:            node.Tool,
					Executor:        node.Executor,
					Status:          domain.NodeStatusFailed,
					RenderedCommand: renderedCommand,
					Inputs:          renderedInputs,
					Error:           fmt.Sprintf("render command: %v", err),
					StartedAt:       nodeStartedAt,
					FinishedAt:      nodeFinishedAt,
					DurationMillis:  nodeFinishedAt.Sub(nodeStartedAt).Milliseconds(),
				}, true)
				return finishSummaryWithError(summary, fmt.Errorf("render command for node %q: %w", node.ID, err))
			}
		}

		execResult, err := execImpl.Execute(ctx, node, tool, renderedInputs, renderedCommand)
		if err != nil {
			nodeFinishedAt := time.Now().UTC()
			recordNodeResult(domain.NodeRunResult{
				NodeID:          node.ID,
				Tool:            node.Tool,
				Executor:        node.Executor,
				Status:          domain.NodeStatusFailed,
				RenderedCommand: renderedCommand,
				Inputs:          renderedInputs,
				Error:           fmt.Sprintf("execute node: %v", err),
				StartedAt:       nodeStartedAt,
				FinishedAt:      nodeFinishedAt,
				DurationMillis:  nodeFinishedAt.Sub(nodeStartedAt).Milliseconds(),
			}, true)
			return finishSummaryWithError(summary, fmt.Errorf("execute node %q: %w", node.ID, err))
		}

		var records []domain.ParsedRecord
		if tool.Parser != "" {
			ruleSet, err := r.loadParserRuleSet(tool.Parser)
			if err != nil {
				nodeFinishedAt := time.Now().UTC()
				recordNodeResult(domain.NodeRunResult{
					NodeID:          node.ID,
					Tool:            node.Tool,
					Executor:        node.Executor,
					Status:          domain.NodeStatusFailed,
					RenderedCommand: renderedCommand,
					Inputs:          renderedInputs,
					Stdout:          execResult.Stdout,
					Metadata:        execResult.Metadata,
					Error:           fmt.Sprintf("load parser: %v", err),
					StartedAt:       nodeStartedAt,
					FinishedAt:      nodeFinishedAt,
					DurationMillis:  nodeFinishedAt.Sub(nodeStartedAt).Milliseconds(),
				}, true)
				return finishSummaryWithError(summary, fmt.Errorf("load parser for node %q: %w", node.ID, err))
			}
			records, err = r.parser.Parse(ruleSet, parserInput(execResult), summary.Assets)
			if err != nil {
				nodeFinishedAt := time.Now().UTC()
				recordNodeResult(domain.NodeRunResult{
					NodeID:          node.ID,
					Tool:            node.Tool,
					Executor:        node.Executor,
					Status:          domain.NodeStatusFailed,
					RenderedCommand: renderedCommand,
					Inputs:          renderedInputs,
					Stdout:          execResult.Stdout,
					Metadata:        execResult.Metadata,
					Error:           fmt.Sprintf("parse output: %v", err),
					StartedAt:       nodeStartedAt,
					FinishedAt:      nodeFinishedAt,
					DurationMillis:  nodeFinishedAt.Sub(nodeStartedAt).Milliseconds(),
				}, true)
				return finishSummaryWithError(summary, fmt.Errorf("parse node %q output: %w", node.ID, err))
			}
		}
		nodeFinishedAt := time.Now().UTC()

		recordNodeResult(domain.NodeRunResult{
			NodeID:          node.ID,
			Tool:            node.Tool,
			Executor:        node.Executor,
			Status:          domain.NodeStatusSucceeded,
			RenderedCommand: renderedCommand,
			Inputs:          renderedInputs,
			Stdout:          execResult.Stdout,
			Metadata:        execResult.Metadata,
			Records:         records,
			RecordCount:     len(records),
			DurationMillis:  nodeFinishedAt.Sub(nodeStartedAt).Milliseconds(),
			StartedAt:       nodeStartedAt,
			FinishedAt:      nodeFinishedAt,
		}, true)

		if err := propagateResolution(node.ID, true); err != nil {
			return finishSummaryWithError(summary, err)
		}
	}

	summary.FinishedAt = time.Now().UTC()
	summary.Status = domain.RunStatusSucceeded
	refreshRunStats(&summary)
	return summary, nil
}

func renderInputs(inputs map[string]any, ctx map[string]any) (map[string]any, error) {
	if len(inputs) == 0 {
		return map[string]any{}, nil
	}
	rendered, err := runtimepkg.RenderValue(inputs, ctx)
	if err != nil {
		return nil, err
	}
	result, ok := rendered.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rendered inputs have unexpected type %T", rendered)
	}
	return result, nil
}

func prepareRunVars(globalVars map[string]any) (map[string]any, error) {
	vars := cloneMap(globalVars)

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve current directory: %w", err)
	}
	outputRoot := filepath.Join(cwd, "output")

	targetRaw := resolveTargetIdentifier(vars)
	targetDir := sanitizeTargetIdentifier(targetRaw)
	outputDir := filepath.Join(outputRoot, targetDir)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory %q: %w", outputDir, err)
	}

	vars["output_root"] = outputRoot
	vars["output_target"] = targetDir
	vars["output_dir"] = outputDir

	if windowsDir, err := toWindowsPath(outputDir); err == nil && windowsDir != "" {
		vars["output_dir_windows"] = windowsDir
	}
	return vars, nil
}

func resolveTargetIdentifier(vars map[string]any) string {
	for _, key := range []string{"output_target", "target_host", "target_url", "target", "host", "url"} {
		if value, ok := vars[key]; ok {
			if text := strings.TrimSpace(fmt.Sprint(value)); text != "" {
				return text
			}
		}
	}
	return "default-target"
}

var targetSegmentSanitizer = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func sanitizeTargetIdentifier(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "default-target"
	}
	sanitized := targetSegmentSanitizer.ReplaceAllString(raw, "_")
	sanitized = strings.Trim(sanitized, "._-")
	if sanitized == "" {
		return "default-target"
	}
	if len(sanitized) > 80 {
		return sanitized[:80]
	}
	return sanitized
}

func toWindowsPath(path string) (string, error) {
	cmd := exec.Command("wslpath", "-w", path)
	data, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func parserInput(result domain.ExecutionResult) string {
	parts := make([]string, 0, 2)
	stdout := strings.TrimSpace(result.Stdout)
	if stdout != "" {
		parts = append(parts, stdout)
	}

	for _, content := range outputFileContents(result.Metadata) {
		text := strings.TrimSpace(content)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}

	return strings.Join(parts, "\n\n")
}

func outputFileContents(metadata map[string]any) []string {
	if len(metadata) == 0 {
		return nil
	}

	raw, ok := metadata["output_files"]
	if !ok || raw == nil {
		return nil
	}

	var contents []string
	appendContent := func(entry map[string]any) {
		if entry == nil {
			return
		}
		content, ok := entry["content"].(string)
		if !ok || content == "" {
			return
		}
		contents = append(contents, content)
	}

	switch value := raw.(type) {
	case []map[string]any:
		for _, entry := range value {
			appendContent(entry)
		}
	case []any:
		for _, item := range value {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			appendContent(entry)
		}
	}
	return contents
}

func cloneIntMap(input map[string]int) map[string]int {
	cloned := make(map[string]int, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func buildResultEnv(nodeResult domain.NodeRunResult) map[string]any {
	return map[string]any{
		"status":           nodeResult.Status,
		"tool":             nodeResult.Tool,
		"executor":         nodeResult.Executor,
		"record_count":     nodeResult.RecordCount,
		"rendered_command": nodeResult.RenderedCommand,
		"metadata":         nodeResult.Metadata,
		"error":            nodeResult.Error,
		"skip_reason":      nodeResult.SkipReason,
	}
}

func refreshRunStats(summary *domain.RunSummary) {
	stats := domain.RunStats{
		TotalNodes: summary.Stats.TotalNodes,
	}
	for _, nodeResult := range summary.NodeResults {
		switch nodeResult.Status {
		case domain.NodeStatusSucceeded:
			stats.ExecutedNodes++
			stats.SucceededNodes++
		case domain.NodeStatusFailed:
			stats.ExecutedNodes++
			stats.FailedNodes++
		case domain.NodeStatusSkipped:
			stats.SkippedNodes++
		}
	}
	summary.Stats = stats
}

func finishSummaryWithError(summary domain.RunSummary, err error) (domain.RunSummary, error) {
	summary.Status = domain.RunStatusFailed
	summary.Error = err.Error()
	summary.FinishedAt = time.Now().UTC()
	refreshRunStats(&summary)
	return summary, err
}

func (r *Runner) validateWorkflowDependencies(nodes map[string]domain.WorkflowNode) error {
	checkedParsers := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		tool, err := r.tools.Get(node.Tool)
		if err != nil {
			return fmt.Errorf("node %q references an unknown tool: %w", node.ID, err)
		}
		if _, err := r.executors.Get(node.Executor); err != nil {
			return fmt.Errorf("node %q references an unknown executor: %w", node.ID, err)
		}
		if tool.Parser != "" && !checkedParsers[tool.Parser] {
			if _, err := r.loadParserRuleSet(tool.Parser); err != nil {
				return fmt.Errorf("tool %q parser %q is invalid: %w", tool.Name, tool.Parser, err)
			}
			checkedParsers[tool.Parser] = true
		}
	}
	return nil
}

func (r *Runner) loadParserRuleSet(path string) (domain.ParserRuleSet, error) {
	r.parserCacheMu.RLock()
	if ruleSet, ok := r.parserCache[path]; ok {
		r.parserCacheMu.RUnlock()
		return ruleSet, nil
	}
	r.parserCacheMu.RUnlock()

	ruleSet, err := config.LoadParserRuleSet(path)
	if err != nil {
		return domain.ParserRuleSet{}, err
	}

	r.parserCacheMu.Lock()
	r.parserCache[path] = ruleSet
	r.parserCacheMu.Unlock()
	return ruleSet, nil
}

func validateAcyclic(workflowName string, incoming map[string]int, adjacency map[string][]domain.WorkflowEdge) error {
	remainingIncoming := cloneIntMap(incoming)
	queue := make([]string, 0, len(remainingIncoming))
	for nodeID, count := range remainingIncoming {
		if count == 0 {
			queue = append(queue, nodeID)
		}
	}

	visited := 0
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		visited++

		for _, edge := range adjacency[nodeID] {
			remainingIncoming[edge.To]--
			if remainingIncoming[edge.To] == 0 {
				queue = append(queue, edge.To)
			}
		}
	}

	if visited != len(remainingIncoming) {
		return fmt.Errorf("workflow %q must be a DAG; cycle detected", workflowName)
	}
	return nil
}
