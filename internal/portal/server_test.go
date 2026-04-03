package portal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"penframe/internal/domain"
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

func TestEventsEndpointStreamsRunLifecycle(t *testing.T) {
	server := newTestServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil).WithContext(ctx)
	recorder := newStreamRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.ServeHTTP(recorder, req)
	}()

	waitForRecorderContent(t, recorder, "event: portal_ready", time.Second)
	executeRun(t, server)
	waitForRecorderContent(t, recorder, "event: run_finished", time.Second)

	cancel()
	<-done

	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("expected event stream content type, got %q", got)
	}

	received, err := parseSSEEvents(strings.NewReader(recorder.BodyString()))
	if err != nil {
		t.Fatalf("parseSSEEvents returned error: %v", err)
	}

	filtered := make([]streamEvent, 0, len(received))
	for _, event := range received {
		if event.Type == "" || event.Type == eventTypePortalReady {
			continue
		}
		filtered = append(filtered, event)
	}

	received = filtered
	if len(received) != 6 {
		t.Fatalf("expected 6 lifecycle events, got %d", len(received))
	}

	gotTypes := make([]string, 0, len(received))
	for _, event := range received {
		gotTypes = append(gotTypes, event.Type)
	}
	wantTypes := []string{
		"run_started",
		"node_started",
		"node_finished",
		"node_started",
		"node_finished",
		"run_finished",
	}
	if strings.Join(gotTypes, ",") != strings.Join(wantTypes, ",") {
		t.Fatalf("unexpected event types: got %v want %v", gotTypes, wantTypes)
	}

	if received[0].Summary == nil || received[0].Summary.Status != "running" {
		t.Fatalf("expected running summary in first event, got %#v", received[0].Summary)
	}
	if received[1].Node == nil || received[1].Node.Status != "running" {
		t.Fatalf("expected running node in second event, got %#v", received[1].Node)
	}
	if received[2].Summary == nil || received[2].Summary.Stats.SucceededNodes == 0 {
		t.Fatalf("expected node_finished event to carry updated summary stats, got %#v", received[2].Summary)
	}
	if received[5].Summary == nil || received[5].Summary.Status != "succeeded" {
		t.Fatalf("expected succeeded summary in final event, got %#v", received[5].Summary)
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
		"",
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
	if got := response.Run.Summary.Vars["target_hostport"]; got != "demo.example:3000" {
		t.Fatalf("expected target_hostport demo.example:3000, got %#v", got)
	}
	if got := response.Run.Summary.Vars["target_origin"]; got != "https://demo.example:3000" {
		t.Fatalf("expected target_origin https://demo.example:3000, got %#v", got)
	}
	if got := response.Run.Summary.Vars["target_port"]; got != "3000" {
		t.Fatalf("expected target_port 3000, got %#v", got)
	}
	if got := response.Run.Summary.Vars["target_url"]; got != "https://demo.example:3000/path" {
		t.Fatalf("expected target_url override, got %#v", got)
	}
}

func TestScanEndpointReturnsInitialRunningRun(t *testing.T) {
	server := newTestServer(t)
	body := bytes.NewBufferString(`{"target":"192.0.2.10","strategy":"custom","phases":["port_scan","vuln_scan"]}`)

	req := httptest.NewRequest(http.MethodPost, "/api/scan", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", recorder.Code)
	}

	var response scanResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal scan response: %v", err)
	}
	if response.RunID == "" {
		t.Fatal("expected run id to be populated")
	}
	if response.Run == nil {
		t.Fatal("expected initial run payload in scan response")
	}
	if response.Run.ID != response.RunID {
		t.Fatalf("expected run payload id %q, got %q", response.RunID, response.Run.ID)
	}
	if response.Run.Summary.Status != domain.RunStatusRunning {
		t.Fatalf("expected initial run status %q, got %q", domain.RunStatusRunning, response.Run.Summary.Status)
	}
	if response.Run.Summary.Stats.TotalNodes == 0 {
		t.Fatal("expected initial run summary to carry workflow node count")
	}
	if len(response.Tasks) == 0 {
		t.Fatal("expected initial tasks to be returned")
	}
	if response.Tasks[0].NodeID == "" {
		t.Fatal("expected workflow-backed tasks to include node_id")
	}

	storedRun, ok := server.store.GetStoredRun(response.RunID)
	if !ok {
		t.Fatal("expected initial run to be saved")
	}
	if storedRun.Summary.Status != domain.RunStatusRunning && storedRun.Summary.Status != domain.RunStatusSucceeded {
		t.Fatalf("expected stored run to exist with running/succeeded status, got %q", storedRun.Summary.Status)
	}
}

func TestAssetsEndpointProjectsStoredRunHierarchy(t *testing.T) {
	server := newTestServer(t)
	runID := executeRun(t, server).Run.ID

	req := httptest.NewRequest(http.MethodGet, "/api/assets/"+runID, nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response struct {
		RunID   string             `json:"run_id"`
		Summary map[string]int     `json:"summary"`
		Hosts   []domain.AssetHost `json:"hosts"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal assets response: %v", err)
	}
	if response.RunID != runID {
		t.Fatalf("expected run id %q, got %q", runID, response.RunID)
	}
	if response.Summary["hosts"] < 2 {
		t.Fatalf("expected projected hosts, got %#v", response.Summary)
	}
	if response.Summary["ports"] < 2 {
		t.Fatalf("expected projected ports, got %#v", response.Summary)
	}
	if len(response.Hosts) == 0 {
		t.Fatal("expected hosts in projected asset graph")
	}
	foundPort := false
	for _, host := range response.Hosts {
		if len(host.Ports) > 0 {
			foundPort = true
			break
		}
	}
	if !foundPort {
		t.Fatal("expected at least one host to contain ports")
	}
}

func TestTasksEndpointCanFilterByRunID(t *testing.T) {
	server := newTestServer(t)

	firstBody := bytes.NewBufferString(`{"target":"192.0.2.11","strategy":"discovery"}`)
	firstReq := httptest.NewRequest(http.MethodPost, "/api/scan", firstBody)
	firstReq.Header.Set("Content-Type", "application/json")
	firstRecorder := httptest.NewRecorder()
	server.ServeHTTP(firstRecorder, firstReq)

	var firstResponse scanResponse
	if err := json.Unmarshal(firstRecorder.Body.Bytes(), &firstResponse); err != nil {
		t.Fatalf("unmarshal first scan response: %v", err)
	}

	secondBody := bytes.NewBufferString(`{"target":"192.0.2.12","strategy":"recon"}`)
	secondReq := httptest.NewRequest(http.MethodPost, "/api/scan", secondBody)
	secondReq.Header.Set("Content-Type", "application/json")
	secondRecorder := httptest.NewRecorder()
	server.ServeHTTP(secondRecorder, secondReq)

	filterReq := httptest.NewRequest(http.MethodGet, "/api/tasks?run_id="+firstResponse.RunID, nil)
	filterRecorder := httptest.NewRecorder()
	server.ServeHTTP(filterRecorder, filterReq)

	if filterRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", filterRecorder.Code)
	}

	var payload struct {
		Tasks []*domain.ScanTask `json:"tasks"`
	}
	if err := json.Unmarshal(filterRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal filtered tasks response: %v", err)
	}
	if len(payload.Tasks) != len(firstResponse.Tasks) {
		t.Fatalf("expected %d tasks for the first run, got %d", len(firstResponse.Tasks), len(payload.Tasks))
	}
	for _, task := range payload.Tasks {
		if task.ParentID != firstResponse.RunID {
			t.Fatalf("expected task parent %q, got %q", firstResponse.RunID, task.ParentID)
		}
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

func parseSSEEvents(reader io.Reader) ([]streamEvent, error) {
	scanner := bufio.NewScanner(reader)
	currentType := ""
	var events []streamEvent
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			currentType = strings.TrimSpace(strings.TrimPrefix(line, "event: "))
		case strings.HasPrefix(line, "data: "):
			var event streamEvent
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
				return nil, err
			}
			if event.Type == "" {
				event.Type = currentType
			}
			events = append(events, event)
			currentType = ""
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func waitForRecorderContent(t *testing.T, recorder *streamRecorder, needle string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(recorder.BodyString(), needle) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in recorder body", needle)
}

type streamRecorder struct {
	mu     sync.Mutex
	header http.Header
	body   bytes.Buffer
	code   int
}

func newStreamRecorder() *streamRecorder {
	return &streamRecorder{
		header: make(http.Header),
		code:   http.StatusOK,
	}
}

func (r *streamRecorder) Header() http.Header {
	return r.header
}

func (r *streamRecorder) WriteHeader(statusCode int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.code = statusCode
}

func (r *streamRecorder) Write(data []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.body.Write(data)
}

func (r *streamRecorder) Flush() {}

func (r *streamRecorder) BodyString() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.body.String()
}
