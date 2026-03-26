package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"unicode"

	"penframe/internal/domain"
)

type Executor interface {
	Name() string
	Execute(ctx context.Context, node domain.WorkflowNode, tool domain.ToolDefinition, renderedInputs map[string]any, renderedCommand string) (domain.ExecutionResult, error)
}

type Registry struct {
	executors map[string]Executor
}

func NewRegistry(executors ...Executor) *Registry {
	items := make(map[string]Executor, len(executors))
	for _, exec := range executors {
		items[exec.Name()] = exec
	}
	return &Registry{executors: items}
}

func (r *Registry) Get(name string) (Executor, error) {
	exec, ok := r.executors[name]
	if !ok {
		return nil, fmt.Errorf("executor %q is not registered", name)
	}
	return exec, nil
}

type MockExecutor struct{}

func NewMockExecutor() MockExecutor {
	return MockExecutor{}
}

func (MockExecutor) Name() string {
	return "mock"
}

func (MockExecutor) Execute(_ context.Context, node domain.WorkflowNode, _ domain.ToolDefinition, _ map[string]any, _ string) (domain.ExecutionResult, error) {
	if node.Mock.StdoutFile == "" {
		return domain.ExecutionResult{}, fmt.Errorf("node %q is missing mock.stdout_file", node.ID)
	}
	data, err := os.ReadFile(node.Mock.StdoutFile)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("read mock stdout file %s: %w", node.Mock.StdoutFile, err)
	}
	metadata := make(map[string]any, len(node.Mock.Metadata)+1)
	for key, value := range node.Mock.Metadata {
		metadata[key] = value
	}
	metadata["source"] = node.Mock.StdoutFile
	return domain.ExecutionResult{
		Stdout:   string(data),
		Metadata: metadata,
	}, nil
}

type LocalExecutor struct {
	shellPath      string
	powerShellPath string
	timeout        time.Duration
}

func NewLocalExecutor() LocalExecutor {
	return LocalExecutor{
		shellPath:      "sh",
		powerShellPath: "powershell.exe",
		timeout:        30 * time.Minute,
	}
}

func (LocalExecutor) Name() string {
	return "local"
}

func (e LocalExecutor) Execute(ctx context.Context, node domain.WorkflowNode, _ domain.ToolDefinition, _ map[string]any, renderedCommand string) (domain.ExecutionResult, error) {
	command := strings.TrimSpace(renderedCommand)
	if command == "" {
		return domain.ExecutionResult{}, fmt.Errorf("node %q rendered an empty command", node.ID)
	}

	execCtx := ctx
	cancel := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && e.timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, e.timeout)
	}
	defer cancel()

	usePowerShell := shouldUsePowerShell(command)
	powerShellCommand := command
	if usePowerShell {
		resolved, err := preparePowerShellCommand(command)
		if err != nil {
			return domain.ExecutionResult{}, fmt.Errorf("prepare powershell command: %w", err)
		}
		powerShellCommand = resolved
	}
	cmd := e.buildCommand(execCtx, command, powerShellCommand, usePowerShell)
	commandForReadback := commandForMetadata(command, powerShellCommand, usePowerShell)
	declaredOutputFiles := extractDeclaredOutputFiles(commandForReadback)
	cleanDeclaredOutputFiles(declaredOutputFiles)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	metadata := map[string]any{
		"launcher":     launcherName(usePowerShell),
		"command":      command,
		"command_exec": commandForMetadata(command, powerShellCommand, usePowerShell),
		"stderr":       strings.TrimSpace(stderr.String()),
		"timed_out":    errors.Is(execCtx.Err(), context.DeadlineExceeded),
		"powershell":   usePowerShell,
		"stdout_bytes": stdout.Len(),
	}
	if cmd.ProcessState != nil {
		metadata["exit_code"] = cmd.ProcessState.ExitCode()
	}

	result := domain.ExecutionResult{
		Stdout:   stdout.String(),
		Metadata: metadata,
	}
	outputFiles := readDeclaredOutputFilesByPath(declaredOutputFiles)
	if len(outputFiles) > 0 {
		metadata["output_files"] = outputFiles
	}
	if err != nil {
		return result, fmt.Errorf("run command: %w", err)
	}
	return result, nil
}

func (e LocalExecutor) buildCommand(ctx context.Context, renderedCommand, powerShellCommand string, usePowerShell bool) *exec.Cmd {
	if usePowerShell {
		// Prefix command with call operator so quoted .exe paths execute correctly in PowerShell.
		script := "& " + powerShellCommand
		return exec.CommandContext(
			ctx,
			e.powerShellPath,
			"-NoProfile",
			"-NonInteractive",
			"-ExecutionPolicy",
			"Bypass",
			"-Command",
			script,
		)
	}
	return exec.CommandContext(ctx, e.shellPath, "-lc", renderedCommand)
}

func preparePowerShellCommand(command string) (string, error) {
	executable, remainder, err := splitFirstToken(command)
	if err != nil {
		return "", err
	}
	resolvedExecutable := executable
	if strings.HasPrefix(executable, "/") {
		if converted, convertErr := wslPathToWindows(executable); convertErr == nil {
			resolvedExecutable = converted
		}
	}
	return quoteForPowerShell(resolvedExecutable) + remainder, nil
}

func splitFirstToken(command string) (string, string, error) {
	raw := strings.TrimSpace(command)
	if raw == "" {
		return "", "", fmt.Errorf("empty command")
	}
	if raw[0] == '"' || raw[0] == '\'' {
		quote := raw[0]
		for idx := 1; idx < len(raw); idx++ {
			if raw[idx] == quote {
				return raw[1:idx], raw[idx+1:], nil
			}
		}
		return "", "", fmt.Errorf("unterminated quoted executable in command %q", command)
	}
	for idx, ch := range raw {
		if unicode.IsSpace(ch) {
			return raw[:idx], raw[idx:], nil
		}
	}
	return raw, "", nil
}

func wslPathToWindows(path string) (string, error) {
	cmd := exec.Command("wslpath", "-w", path)
	data, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func quoteForPowerShell(value string) string {
	escaped := strings.ReplaceAll(value, "'", "''")
	return "'" + escaped + "'"
}

func shouldUsePowerShell(command string) bool {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "&") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "&"))
	}
	first := firstToken(trimmed)
	if first == "" {
		return false
	}
	return strings.HasSuffix(strings.ToLower(first), ".exe")
}

func firstToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if raw[0] == '"' || raw[0] == '\'' {
		quote := raw[0]
		for idx := 1; idx < len(raw); idx++ {
			if raw[idx] == quote {
				return raw[1:idx]
			}
		}
		return strings.Trim(raw, `"'`)
	}
	for idx, ch := range raw {
		if unicode.IsSpace(ch) {
			return raw[:idx]
		}
	}
	return raw
}

func launcherName(usePowerShell bool) string {
	if usePowerShell {
		return "powershell"
	}
	return "shell"
}

func commandForMetadata(shellCommand, powerShellCommand string, usePowerShell bool) string {
	if usePowerShell {
		return powerShellCommand
	}
	return shellCommand
}

var outputFlagPattern = regexp.MustCompile(`(?i)-(oN|oX|oG|oA|o)\s+("([^"]+)"|'([^']+)'|(\S+))`)

func readDeclaredOutputFiles(command string) []map[string]any {
	declared := extractDeclaredOutputFiles(command)
	return readDeclaredOutputFilesByPath(declared)
}

func readDeclaredOutputFilesByPath(declared []string) []map[string]any {
	if len(declared) == 0 {
		return nil
	}

	files := make([]map[string]any, 0, len(declared))
	for _, declaredPath := range declared {
		content, sizeBytes, truncated, err := readOutputFileContent(declaredPath)
		entry := map[string]any{
			"path":       declaredPath,
			"size_bytes": sizeBytes,
			"truncated":  truncated,
		}
		if err != nil {
			entry["error"] = err.Error()
		} else {
			entry["content"] = content
		}
		files = append(files, entry)
	}
	return files
}

func cleanDeclaredOutputFiles(declared []string) {
	for _, path := range declared {
		for _, candidate := range readCandidates(path) {
			if candidate == "" {
				continue
			}
			err := os.Remove(candidate)
			if err == nil || errors.Is(err, os.ErrNotExist) {
				continue
			}
		}
	}
}

func extractDeclaredOutputFiles(command string) []string {
	matches := outputFlagPattern.FindAllStringSubmatch(command, -1)
	if len(matches) == 0 {
		return nil
	}

	var files []string
	seen := map[string]struct{}{}

	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		files = append(files, path)
	}

	for _, match := range matches {
		flag := strings.ToLower(strings.TrimSpace(match[1]))
		path := firstNonEmpty(match[3], match[4], match[5])
		if path == "" {
			continue
		}
		if flag == "oa" {
			add(path + ".nmap")
			add(path + ".xml")
			add(path + ".gnmap")
			continue
		}
		add(path)
	}
	return files
}

func readOutputFileContent(path string) (string, int64, bool, error) {
	const maxBytes = 1 << 20 // 1 MiB
	for _, candidate := range readCandidates(path) {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		size := int64(len(data))
		if len(data) > maxBytes {
			return string(data[:maxBytes]), size, true, nil
		}
		return string(data), size, false, nil
	}
	return "", 0, false, fmt.Errorf("read output file failed")
}

func readCandidates(path string) []string {
	candidates := []string{path}
	if isLikelyWindowsPath(path) {
		if wslPath, err := windowsPathToWSL(path); err == nil && wslPath != "" {
			candidates = append([]string{wslPath}, candidates...)
		}
	}
	return candidates
}

func windowsPathToWSL(path string) (string, error) {
	cmd := exec.Command("wslpath", "-u", path)
	data, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func isLikelyWindowsPath(path string) bool {
	if len(path) >= 2 && unicode.IsLetter(rune(path[0])) && path[1] == ':' {
		return true
	}
	return strings.Contains(path, `\`)
}
