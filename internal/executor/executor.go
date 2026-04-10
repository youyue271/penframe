package executor

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

type HTTPExecutor struct {
	client *http.Client
}

func NewLocalExecutor() LocalExecutor {
	return LocalExecutor{
		shellPath:      defaultShellPath(),
		powerShellPath: "powershell.exe",
		timeout:        30 * time.Minute,
	}
}

func (LocalExecutor) Name() string {
	return "local"
}

func NewHTTPExecutor() HTTPExecutor {
	return HTTPExecutor{}
}

func (HTTPExecutor) Name() string {
	return "http"
}

func (e HTTPExecutor) Execute(ctx context.Context, node domain.WorkflowNode, _ domain.ToolDefinition, renderedInputs map[string]any, _ string) (domain.ExecutionResult, error) {
	targetURL := strings.TrimSpace(fmt.Sprint(renderedInputs["url"]))
	headersFile := strings.TrimSpace(fmt.Sprint(renderedInputs["headers_file"]))
	bodyFile := strings.TrimSpace(fmt.Sprint(renderedInputs["body_file"]))
	if targetURL == "" {
		return domain.ExecutionResult{}, fmt.Errorf("node %q is missing inputs.url", node.ID)
	}
	if headersFile == "" {
		return domain.ExecutionResult{}, fmt.Errorf("node %q is missing inputs.headers_file", node.ID)
	}
	if bodyFile == "" {
		return domain.ExecutionResult{}, fmt.Errorf("node %q is missing inputs.body_file", node.ID)
	}

	reqCtx := ctx
	cancel := func() {}
	if timeoutSeconds := parseTimeoutSeconds(renderedInputs["max_time"]); timeoutSeconds > 0 {
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			reqCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		}
	}
	defer cancel()

	cleanDeclaredOutputFiles([]string{headersFile, bodyFile})
	if err := ensureParentDir(headersFile); err != nil {
		return domain.ExecutionResult{}, err
	}
	if err := ensureParentDir(bodyFile); err != nil {
		return domain.ExecutionResult{}, err
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("build http request: %w", err)
	}
	mergedProxyEnv := mergeEnvMap(os.Environ(), node.Env)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           proxyFuncFromEnvMap(mergedProxyEnv),
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	if e.client != nil {
		cloned := *e.client
		if cloned.CheckRedirect == nil {
			cloned.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
		}
		client = &cloned
	}

	resp, err := client.Do(req)
	if err != nil {
		result := domain.ExecutionResult{
			Metadata: map[string]any{
				"launcher":    "http",
				"url":         targetURL,
				"timed_out":   errors.Is(reqCtx.Err(), context.DeadlineExceeded),
				"status_code": 0,
			},
		}
		return result, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("read http response body: %w", err)
	}
	headersData := formatHTTPResponseHeaders(resp)

	if err := os.WriteFile(headersFile, headersData, 0o644); err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("write headers file %s: %w", headersFile, err)
	}
	if err := os.WriteFile(bodyFile, bodyData, 0o644); err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("write body file %s: %w", bodyFile, err)
	}

	metadata := map[string]any{
		"launcher":     "http",
		"url":          targetURL,
		"status_code":  resp.StatusCode,
		"timed_out":    errors.Is(reqCtx.Err(), context.DeadlineExceeded),
		"stdout_bytes": 0,
	}
	outputFiles := readDeclaredOutputFilesByPath([]string{headersFile, bodyFile})
	if len(outputFiles) > 0 {
		metadata["output_files"] = outputFiles
	}
	return domain.ExecutionResult{
		Stdout:   "",
		Metadata: metadata,
	}, nil
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
	shellPath := resolveShellPath(e.shellPath, node.Shell)
	if usePowerShell {
		resolved, err := preparePowerShellCommand(command)
		if err != nil {
			return domain.ExecutionResult{}, fmt.Errorf("prepare powershell command: %w", err)
		}
		powerShellCommand = resolved
	}
	cmd := e.buildCommand(execCtx, command, powerShellCommand, shellPath, usePowerShell)
	cmd.Env = mergeCommandEnv(os.Environ(), node.Env)
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
		"shell_path":   shellPath,
		"env_keys":     envKeys(node.Env),
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

func (e LocalExecutor) buildCommand(ctx context.Context, renderedCommand, powerShellCommand, shellPath string, usePowerShell bool) *exec.Cmd {
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
	return exec.CommandContext(ctx, shellPath, "-lc", renderedCommand)
}

func defaultShellPath() string {
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		return shell
	}
	return "sh"
}

func resolveShellPath(defaultPath, override string) string {
	if selected := strings.TrimSpace(override); selected != "" {
		return selected
	}
	if selected := strings.TrimSpace(defaultPath); selected != "" {
		return selected
	}
	return "sh"
}

func mergeCommandEnv(base []string, overrides map[string]any) []string {
	if len(overrides) == 0 {
		return append([]string(nil), base...)
	}

	merged := mergeEnvMap(base, overrides)

	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+merged[key])
	}
	return result
}

func mergeEnvMap(base []string, overrides map[string]any) map[string]string {
	merged := make(map[string]string, len(base)+len(overrides))
	for _, entry := range base {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		merged[key] = value
	}
	for key, value := range overrides {
		merged[key] = envValueString(value)
	}
	return merged
}

func proxyFuncFromEnvMap(env map[string]string) func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		if req == nil || req.URL == nil {
			return nil, nil
		}
		return proxyURLFromEnv(req.URL, env)
	}
}

func proxyURLFromEnv(target *url.URL, env map[string]string) (*url.URL, error) {
	if target == nil {
		return nil, nil
	}

	lookup := func(keys ...string) string {
		for _, key := range keys {
			if value := strings.TrimSpace(env[key]); value != "" {
				return value
			}
		}
		return ""
	}

	var raw string
	switch strings.ToLower(target.Scheme) {
	case "https":
		raw = lookup("HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy")
	case "http":
		raw = lookup("HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy")
	default:
		raw = lookup("ALL_PROXY", "all_proxy")
	}
	if raw == "" {
		return nil, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func envKeys(values map[string]any) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func envValueString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func parseTimeoutSeconds(value any) int {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" {
		return 0
	}
	seconds, err := strconv.Atoi(text)
	if err != nil || seconds <= 0 {
		return 0
	}
	return seconds
}

func ensureParentDir(path string) error {
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent directory %q: %w", dir, err)
	}
	return nil
}

func formatHTTPResponseHeaders(resp *http.Response) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, "%s %s\n", resp.Proto, resp.Status)
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Fprintf(&out, "%s: %s\n", key, value)
		}
	}
	out.WriteString("\n")
	return out.Bytes()
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

var outputFlagPattern = regexp.MustCompile(`(?i)(?:^|\s)(-oN|-oX|-oG|-oA|-o|-D|--output|--dump-header)\s+("([^"]+)"|'([^']+)'|(\S+))`)

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
		if flag == "-oa" {
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
