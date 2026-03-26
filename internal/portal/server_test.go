package portal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStateEndpointReturnsWorkflowAndTools(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response stateResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal state response: %v", err)
	}
	if response.Workflow.Name != "safe-orchestrator-mvp" {
		t.Fatalf("expected workflow name safe-orchestrator-mvp, got %q", response.Workflow.Name)
	}
	if len(response.Tools) < 4 {
		t.Fatalf("expected at least 4 tools, got %d", len(response.Tools))
	}
	if response.WorkflowMeta.NodeCount != 2 {
		t.Fatalf("expected 2 nodes, got %d", response.WorkflowMeta.NodeCount)
	}
}

func TestRunEndpointExecutesWorkflowAndPersistsLatestRun(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/run", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", recorder.Code)
	}

	var response runResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal run response: %v", err)
	}
	if response.Run.ID == "" {
		t.Fatal("expected run id to be populated")
	}
	if response.Run.Summary.Status != "succeeded" {
		t.Fatalf("expected succeeded run, got %q", response.Run.Summary.Status)
	}
	if len(response.Run.Summary.ExecutionOrder) != 2 {
		t.Fatalf("expected 2 executed nodes, got %d", len(response.Run.Summary.ExecutionOrder))
	}

	latest, ok := server.store.Latest()
	if !ok {
		t.Fatal("expected latest run to be saved")
	}
	if latest.ID != response.Run.ID {
		t.Fatalf("expected latest run id %q, got %q", response.Run.ID, latest.ID)
	}
}

func TestRunsEndpointsExposeStoredRuns(t *testing.T) {
	server := newTestServer(t)
	runID := executeRun(t, server).Run.ID

	listReq := httptest.NewRequest(http.MethodGet, "/api/runs?limit=1", nil)
	listRecorder := httptest.NewRecorder()
	server.ServeHTTP(listRecorder, listReq)

	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listRecorder.Code)
	}

	var listResponse runsResponse
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("unmarshal runs response: %v", err)
	}
	if len(listResponse.Runs) != 1 {
		t.Fatalf("expected 1 listed run, got %d", len(listResponse.Runs))
	}
	if listResponse.Runs[0].ID != runID {
		t.Fatalf("expected listed run id %q, got %q", runID, listResponse.Runs[0].ID)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/runs/"+runID, nil)
	getRecorder := httptest.NewRecorder()
	server.ServeHTTP(getRecorder, getReq)

	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", getRecorder.Code)
	}

	var getResponse runResponse
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &getResponse); err != nil {
		t.Fatalf("unmarshal run response: %v", err)
	}
	if getResponse.Run.ID != runID {
		t.Fatalf("expected fetched run id %q, got %q", runID, getResponse.Run.ID)
	}
}

func TestReloadEndpointRefreshesWorkflowDefinition(t *testing.T) {
	root := repoRoot(t)
	tempDir := t.TempDir()
	configRoot := filepath.Join(tempDir, "mvp")
	copyDir(t, filepath.Join(root, "examples", "mvp"), configRoot)

	workflowPath := filepath.Join(configRoot, "workflow.yaml")
	workflowData, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("ReadFile workflow returned error: %v", err)
	}
	updatedWorkflow := strings.Replace(string(workflowData), "name: safe-orchestrator-mvp", "name: 重新加载后的工作流", 1)
	if err := os.WriteFile(workflowPath, []byte(updatedWorkflow), 0o644); err != nil {
		t.Fatalf("WriteFile workflow returned error: %v", err)
	}

	server, err := NewServer(
		filepath.Join(configRoot, "tools.yaml"),
		workflowPath,
	)
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}

	reloadReq := httptest.NewRequest(http.MethodPost, "/api/reload", nil)
	reloadRecorder := httptest.NewRecorder()
	server.ServeHTTP(reloadRecorder, reloadReq)

	if reloadRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", reloadRecorder.Code)
	}

	var response stateResponse
	if err := json.Unmarshal(reloadRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal reload response: %v", err)
	}
	if response.Workflow.Name != "重新加载后的工作流" {
		t.Fatalf("expected reloaded workflow name, got %q", response.Workflow.Name)
	}
}

func TestToolFilesEndpointListsExternalFiles(t *testing.T) {
	root := repoRoot(t)
	externalRoot := t.TempDir()

	mustWriteFile(t, filepath.Join(externalRoot, "ffuf", "ffuf.exe"), []byte("binary"))
	mustWriteFile(t, filepath.Join(externalRoot, "ffuf", "README.md"), []byte("docs"))
	mustWriteFile(t, filepath.Join(externalRoot, "03 getshell", "antsword.zip"), []byte("archive"))
	mustWriteFile(t, filepath.Join(externalRoot, "deep", "a", "b", "c", "skip.txt"), []byte("too deep"))

	server, err := newServerWithExternalRoot(
		filepath.Join(root, "examples", "mvp", "tools.yaml"),
		filepath.Join(root, "examples", "mvp", "workflow.yaml"),
		externalRoot,
	)
	if err != nil {
		t.Fatalf("newServerWithExternalRoot returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tool-files", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response toolFilesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal tool files response: %v", err)
	}
	if response.Root != filepath.Clean(externalRoot) {
		t.Fatalf("expected root %q, got %q", filepath.Clean(externalRoot), response.Root)
	}
	if len(response.Files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(response.Files))
	}

	entries := make(map[string]toolFileEntry, len(response.Files))
	for _, entry := range response.Files {
		entries[entry.RelativePath] = entry
	}

	ffufEntry, ok := entries["ffuf/ffuf.exe"]
	if !ok {
		t.Fatal("expected ffuf/ffuf.exe to be listed")
	}
	if ffufEntry.Kind != "可启动文件" {
		t.Fatalf("expected executable kind, got %q", ffufEntry.Kind)
	}
	if ffufEntry.Category != "ffuf" {
		t.Fatalf("expected category ffuf, got %q", ffufEntry.Category)
	}

	readmeEntry, ok := entries["ffuf/README.md"]
	if !ok {
		t.Fatal("expected ffuf/README.md to be listed")
	}
	if readmeEntry.Kind != "文档" {
		t.Fatalf("expected document kind, got %q", readmeEntry.Kind)
	}

	if _, ok := entries["deep/a/b/c/skip.txt"]; ok {
		t.Fatal("expected files deeper than max depth to be skipped")
	}
}

func TestRunEndpointPersistsFailedRuns(t *testing.T) {
	root := repoRoot(t)
	tempDir := t.TempDir()
	configRoot := filepath.Join(tempDir, "mvp")
	copyDir(t, filepath.Join(root, "examples", "mvp"), configRoot)

	workflowPath := filepath.Join(configRoot, "workflow.yaml")
	workflowData, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("ReadFile workflow returned error: %v", err)
	}
	updatedWorkflow := strings.Replace(string(workflowData), "stdout_file: fixtures/discovery.txt", "stdout_file: fixtures/missing.txt", 1)
	if err := os.WriteFile(workflowPath, []byte(updatedWorkflow), 0o644); err != nil {
		t.Fatalf("WriteFile workflow returned error: %v", err)
	}

	server, err := NewServer(
		filepath.Join(configRoot, "tools.yaml"),
		workflowPath,
	)
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/run", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", recorder.Code)
	}

	var response runResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal run response: %v", err)
	}
	if response.Error == "" {
		t.Fatal("expected failed run response to include an error")
	}
	if response.Run.Summary.Status != "failed" {
		t.Fatalf("expected failed run status, got %q", response.Run.Summary.Status)
	}
	if _, ok := server.store.Latest(); !ok {
		t.Fatal("expected failed run to be stored")
	}
}

func TestRunEndpointAppliesTargetOverride(t *testing.T) {
	server := newTestServer(t)
	body := bytes.NewBufferString(`{"target":"https://demo.example:3000/path"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/run", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", recorder.Code)
	}

	var response runResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal run response: %v", err)
	}
	if got := response.Run.Summary.Vars["target"]; got != "https://demo.example:3000/path" {
		t.Fatalf("expected target override to be applied, got %#v", got)
	}
	if got := response.Run.Summary.Vars["target_host"]; got != "demo.example" {
		t.Fatalf("expected target_host demo.example, got %#v", got)
	}
	if got := response.Run.Summary.Vars["target_url"]; got != "https://demo.example:3000/path" {
		t.Fatalf("expected target_url override, got %#v", got)
	}
}

func TestResolveRunTimeout(t *testing.T) {
	if got := resolveRunTimeout(0); got != defaultRunTimeout {
		t.Fatalf("expected default timeout %v, got %v", defaultRunTimeout, got)
	}
	if got := resolveRunTimeout(5); got != minRunTimeout {
		t.Fatalf("expected min timeout %v, got %v", minRunTimeout, got)
	}
	if got := resolveRunTimeout(3600); got != time.Hour {
		t.Fatalf("expected one hour timeout, got %v", got)
	}
	if got := resolveRunTimeout(200000); got != maxRunTimeout {
		t.Fatalf("expected max timeout %v, got %v", maxRunTimeout, got)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	root := repoRoot(t)
	server, err := NewServer(
		filepath.Join(root, "examples", "mvp", "tools.yaml"),
		filepath.Join(root, "examples", "mvp", "workflow.yaml"),
	)
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}
	return server
}

func executeRun(t *testing.T, server *Server) runResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/run", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", recorder.Code)
	}
	var response runResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal run response: %v", err)
	}
	return response
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0o644)
	}); err != nil {
		t.Fatalf("copyDir returned error: %v", err)
	}
}
