package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"penframe/internal/config"
	"penframe/internal/executor"
	"penframe/internal/parser"
	"penframe/internal/tooling"
	"penframe/internal/workflow"
)

func main() {
	toolsPath := flag.String("tools", "examples/mvp/tools.yaml", "path to tool catalog")
	workflowPath := flag.String("workflow", "examples/mvp/workflow.yaml", "path to workflow definition")
	flag.Parse()

	tools, err := config.LoadToolCatalog(*toolsPath)
	if err != nil {
		exitf("load tools: %v", err)
	}
	wf, err := config.LoadWorkflow(*workflowPath)
	if err != nil {
		exitf("load workflow: %v", err)
	}

	runner := workflow.NewRunner(
		tooling.NewRegistry(tools),
		executor.NewRegistry(executor.NewMockExecutor(), executor.NewLocalExecutor()),
		parser.NewEngine(),
		workflow.NewMiniExprEvaluator(),
	)

	summary, err := runner.Run(context.Background(), wf)
	if err != nil {
		exitf("run workflow: %v", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		exitf("write summary: %v", err)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
