package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"penframe/internal/domain"
)

// ExpExecutor delegates execution to the Python exploit service.
type ExpExecutor struct {
	baseURL string
	client  *http.Client
}

// NewExpExecutor creates an ExpExecutor pointing at the Python FastAPI service.
func NewExpExecutor(baseURL string) ExpExecutor {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8787"
	}
	return ExpExecutor{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (ExpExecutor) Name() string { return "exp" }

type expRequest struct {
	Executor string         `json:"executor"`
	Target   string         `json:"target"`
	Entry    string         `json:"entry"`
	Finding  string         `json:"finding,omitempty"`
	Command  string         `json:"command,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type expResponse struct {
	Accepted  bool   `json:"accepted"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Result    any    `json:"result,omitempty"`
}

func (e ExpExecutor) Execute(ctx context.Context, node domain.WorkflowNode, _ domain.ToolDefinition, renderedInputs map[string]any, _ string) (domain.ExecutionResult, error) {
	target := inputStr(renderedInputs, "target")
	entry := inputStr(renderedInputs, "entry")
	if target == "" {
		return domain.ExecutionResult{}, fmt.Errorf("node %q: exp executor requires inputs.target", node.ID)
	}
	if entry == "" {
		entry = target
	}

	payload := expRequest{
		Executor: inputStr(renderedInputs, "exploit_id"),
		Target:   target,
		Entry:    entry,
		Finding:  inputStr(renderedInputs, "finding"),
		Command:  inputStr(renderedInputs, "command"),
	}
	if payload.Executor == "" {
		payload.Executor = inputStr(renderedInputs, "executor_name")
	}
	if payload.Executor == "" {
		payload.Executor = "auto"
	}
	payload.Metadata = collectProxyMetadata(renderedInputs)

	body, err := json.Marshal(payload)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("marshal exp request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/api/v1/execute", bytes.NewReader(body))
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("build exp request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return domain.ExecutionResult{
			Metadata: map[string]any{
				"launcher": "exp",
				"url":      e.baseURL,
				"error":    err.Error(),
			},
		}, fmt.Errorf("exp service call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("read exp response: %w", err)
	}

	var result expResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return domain.ExecutionResult{
			Stdout: string(respBody),
			Metadata: map[string]any{
				"launcher":    "exp",
				"status_code": resp.StatusCode,
			},
		}, fmt.Errorf("decode exp response: %w", err)
	}

	metadata := map[string]any{
		"launcher":    "exp",
		"url":         e.baseURL,
		"status_code": resp.StatusCode,
		"accepted":    result.Accepted,
		"exp_status":  result.Status,
		"request_id":  result.RequestID,
	}
	if result.Result != nil {
		metadata["result"] = result.Result
	}

	stdout := result.Message
	if stdout == "" {
		stdout = string(respBody)
	}

	execResult := domain.ExecutionResult{
		Stdout:   stdout,
		Metadata: metadata,
	}

	if resp.StatusCode >= 400 {
		return execResult, fmt.Errorf("exp service returned %d: %s", resp.StatusCode, result.Message)
	}

	return execResult, nil
}

// ListExploits calls the Python service's list endpoint.
func (e ExpExecutor) ListExploits(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.baseURL+"/api/v1/exploits", nil)
	if err != nil {
		return nil, err
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// CheckExploit calls the Python service's check endpoint.
func (e ExpExecutor) CheckExploit(ctx context.Context, target, exploitID string) ([]byte, error) {
	return e.CheckExploitWithOptions(ctx, target, exploitID, nil)
}

func (e ExpExecutor) CheckExploitWithOptions(ctx context.Context, target, exploitID string, options map[string]any) ([]byte, error) {
	payload := map[string]any{
		"target":     target,
		"exploit_id": exploitID,
		"options":    options,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/api/v1/check", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// RunExploit calls the Python service's execute endpoint for manual portal use.
func (e ExpExecutor) RunExploit(ctx context.Context, target, exploitID, command string) ([]byte, error) {
	return e.RunExploitWithOptions(ctx, target, exploitID, command, nil)
}

func (e ExpExecutor) RunExploitWithOptions(ctx context.Context, target, exploitID, command string, options map[string]any) ([]byte, error) {
	payload := expRequest{
		Executor: exploitID,
		Target:   target,
		Entry:    target,
		Command:  command,
		Options:  options,
	}
	if payload.Executor == "" {
		payload.Executor = "auto"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/api/v1/execute", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func inputStr(inputs map[string]any, key string) string {
	if inputs == nil {
		return ""
	}
	v, ok := inputs[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func collectProxyMetadata(inputs map[string]any) map[string]any {
	httpProxy := inputStr(inputs, "http_proxy")
	socks5Proxy := inputStr(inputs, "socks5_proxy")
	if httpProxy == "" && socks5Proxy == "" {
		return nil
	}
	metadata := map[string]any{}
	if httpProxy != "" {
		metadata["http_proxy"] = httpProxy
	}
	if socks5Proxy != "" {
		metadata["socks5_proxy"] = socks5Proxy
	}
	return metadata
}
