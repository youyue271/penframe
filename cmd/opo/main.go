package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"penframe/internal/config"
	"penframe/internal/executor"
	"penframe/internal/parser"
	"penframe/internal/targeting"
	"penframe/internal/tooling"
	"penframe/internal/workflow"
)

func main() {
	toolsPath := flag.String("tools", "", "path to tool catalog (required)")
	workflowPath := flag.String("workflow", "", "path to workflow definition (required)")
	target := flag.String("target", "", "override target URL or host for this run")
	timeoutSeconds := flag.Int("timeout", 0, "workflow timeout in seconds (0 keeps the default context)")
	var overrides multiFlag
	flag.Var(&overrides, "var", "override workflow variable as key=value; repeatable")
	flag.Parse()
	if strings.TrimSpace(*toolsPath) == "" || strings.TrimSpace(*workflowPath) == "" {
		exitf("both -tools and -workflow are required")
	}

	tools, err := config.LoadToolCatalog(*toolsPath)
	if err != nil {
		exitf("load tools: %v", err)
	}
	wf, err := config.LoadWorkflow(*workflowPath)
	if err != nil {
		exitf("load workflow: %v", err)
	}
	if wf.GlobalVars == nil {
		wf.GlobalVars = map[string]any{}
	}
	for _, override := range overrides {
		key, value, err := parseOverride(override)
		if err != nil {
			exitf("parse -var %q: %v", override, err)
		}
		wf.GlobalVars[key] = value
	}
	if strings.TrimSpace(*target) != "" {
		targeting.ApplyOverride(wf.GlobalVars, *target)
	} else {
		targeting.Ensure(wf.GlobalVars)
	}

	runner := workflow.NewRunner(
		tooling.NewRegistry(tools),
		executor.NewRegistry(
			executor.NewMockExecutor(),
			executor.NewLocalExecutor(),
			executor.NewHTTPExecutor(),
			executor.NewExpExecutor(""),
		),
		parser.NewEngine(),
		workflow.NewMiniExprEvaluator(),
	)

	ctx := context.Background()
	if *timeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*timeoutSeconds)*time.Second)
		defer cancel()
	}

	summary, err := runner.Run(ctx, wf)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if encodeErr := encoder.Encode(summary); encodeErr != nil {
		exitf("write summary: %v", encodeErr)
	}
	if err != nil {
		exitf("run workflow: %v", err)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

type multiFlag []string

func (f *multiFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *multiFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func parseOverride(raw string) (string, any, error) {
	key, value, ok := strings.Cut(raw, "=")
	if !ok {
		return "", nil, fmt.Errorf("expected key=value")
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", nil, fmt.Errorf("key must not be empty")
	}

	value = strings.TrimSpace(value)
	if parsed, err := strconv.ParseBool(value); err == nil {
		return key, parsed, nil
	}
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return key, parsed, nil
	}
	if parsed, err := strconv.ParseFloat(value, 64); err == nil {
		return key, parsed, nil
	}
	if strings.EqualFold(value, "null") {
		return key, nil, nil
	}
	return key, value, nil
}
