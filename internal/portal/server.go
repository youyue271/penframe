package portal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"penframe/internal/asset"
	"penframe/internal/config"
	"penframe/internal/domain"
	"penframe/internal/executor"
	"penframe/internal/parser"
	"penframe/internal/project"
	"penframe/internal/storage"
	"penframe/internal/targeting"
	"penframe/internal/tooling"
	"penframe/internal/workflow"
)

const defaultExternalToolsRoot = "/mnt/h/tools/Penetration/tools"
const defaultToolFileMaxDepth = 4
const defaultToolFileLimit = 400
const maxToolFileLimit = 2000
const defaultRunTimeout = 30 * time.Minute
const minRunTimeout = 30 * time.Second
const maxRunTimeout = 24 * time.Hour

type Server struct {
	mu           sync.RWMutex
	toolsPath    string
	workflowPath string
	externalRoot string
	tools        map[string]domain.ToolDefinition
	workflow     domain.Workflow
	runner       *workflow.Runner
	store        *storage.MemoryStore
	events       *eventBroker
	mux          *http.ServeMux
	assets       *asset.Store
	projects     *project.Store
	expExecutor  *executor.ExpExecutor
	handler      http.Handler
}

type stateResponse struct {
	Workflow     domain.Workflow         `json:"workflow"`
	Tools        []domain.ToolDefinition `json:"tools"`
	Paths        configPaths             `json:"paths"`
	ExternalRoot string                  `json:"external_root"`
	LatestRun    *storage.StoredRun      `json:"latest_run,omitempty"`
	RecentRuns   []storage.StoredRun     `json:"recent_runs"`
	WorkflowMeta workflowMeta            `json:"workflow_meta"`
}

type configPaths struct {
	Tools    string `json:"tools"`
	Workflow string `json:"workflow"`
}

type workflowMeta struct {
	NodeCount  int      `json:"node_count"`
	EdgeCount  int      `json:"edge_count"`
	EntryNodes []string `json:"entry_nodes"`
}

type runResponse struct {
	Run   storage.StoredRun `json:"run"`
	Error string            `json:"error,omitempty"`
}

type runRequest struct {
	Target         string         `json:"target"`
	Vars           map[string]any `json:"vars"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
}

type runsResponse struct {
	Runs []storage.StoredRun `json:"runs"`
}

type toolFileEntry struct {
	Name         string `json:"name"`
	RelativePath string `json:"relative_path"`
	AbsolutePath string `json:"absolute_path"`
	Category     string `json:"category"`
	Extension    string `json:"extension"`
	Kind         string `json:"kind"`
	SizeBytes    int64  `json:"size_bytes"`
}

type toolFilesResponse struct {
	Root      string          `json:"root"`
	Limit     int             `json:"limit"`
	Truncated bool            `json:"truncated"`
	Files     []toolFileEntry `json:"files"`
}

func NewServer(toolsPath, workflowPath string) (*Server, error) {
	return newServerWithExternalRoot(toolsPath, workflowPath, defaultExternalToolsRoot, "")
}

// NewServerWithExpURL creates a server with the exp executor pointing to the given URL.
func NewServerWithExpURL(toolsPath, workflowPath, expURL string) (*Server, error) {
	return newServerWithExternalRoot(toolsPath, workflowPath, defaultExternalToolsRoot, expURL)
}

func newServerWithExternalRoot(toolsPath, workflowPath, externalRoot, expURL string) (*Server, error) {
	tools, wf, runner, err := loadRuntime(toolsPath, workflowPath)
	if err != nil {
		return nil, err
	}

	projectStore, err := project.NewStore(filepath.Join(".penframe"))
	if err != nil {
		return nil, err
	}
	runStore, err := storage.NewFileStore(filepath.Join(".penframe"))
	if err != nil {
		return nil, err
	}

	server := &Server{
		toolsPath:    toolsPath,
		workflowPath: workflowPath,
		externalRoot: externalRoot,
		tools:        tools,
		workflow:     wf,
		runner:       runner,
		store:        runStore,
		events:       newEventBroker(),
		mux:          http.NewServeMux(),
		assets:       asset.NewStore(),
		projects:     projectStore,
	}

	if expURL != "" {
		exp := executor.NewExpExecutor(expURL)
		server.expExecutor = &exp
	}

	if err := server.routes(); err != nil {
		return nil, err
	}
	server.handler = corsMiddleware(server.mux)
	return server, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) routes() error {
	// Existing API routes.
	s.mux.HandleFunc("/api/state", s.handleState)
	s.mux.HandleFunc("/api/events", s.handleEvents)
	s.mux.HandleFunc("/api/run", s.handleRun)
	s.mux.HandleFunc("/api/reload", s.handleReload)
	s.mux.HandleFunc("/api/runs", s.handleRuns)
	s.mux.HandleFunc("/api/runs/", s.handleRunByID)
	s.mux.HandleFunc("/api/config/tool-paths", s.handleToolPathConfig)
	s.mux.HandleFunc("/api/tool-files", s.handleToolFiles)

	// New API routes for asset graph, scan control, and exploit.
	s.mux.HandleFunc("/api/assets", s.handleAssets)
	s.mux.HandleFunc("/api/assets/", s.handleAssetsByRun)
	s.mux.HandleFunc("/api/hosts", s.handleListHosts)
	s.mux.HandleFunc("/api/hosts/", s.handleHostPorts)
	s.mux.HandleFunc("/api/ports/", s.handlePortDetails)
	s.mux.HandleFunc("/api/scan", s.handleScan)
	s.mux.HandleFunc("/api/scan/", s.handleScanAction)
	s.mux.HandleFunc("/api/tasks", s.handleTasks)
	s.mux.HandleFunc("/api/projects", s.handleProjects)
	s.mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/targets") {
			if strings.HasSuffix(path, "/targets") { // /api/projects/{id}/targets
				s.handleProjectTargets(w, r)
			} else { // /api/projects/{id}/targets/{tid}
				s.handleProjectTarget(w, r)
			}
		} else {
			s.handleDeleteProject(w, r)
		}
	})
	s.mux.HandleFunc("/api/exploit", s.handleExploit)
	s.mux.HandleFunc("/api/exploits", s.handleExploitsList)
	s.mux.HandleFunc("/api/logs", s.handleLogs)

	s.mux.HandleFunc("/healthz", s.handleHealth)

	// Serve a minimal JSON fallback for root (Vue dev server handles the real UI).
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			writeJSON(w, http.StatusOK, map[string]string{
				"service": "penframe-portal",
				"status":  "ok",
				"ui":      "use Vue dev server at http://localhost:5173",
			})
			return
		}
		http.NotFound(w, r)
	})
	return nil
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, s.currentStateResponse())
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	request, err := parseRunRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), resolveRunTimeout(request.TimeoutSeconds))
	defer cancel()
	runID := fmt.Sprintf("run-%d", time.Now().UTC().UnixNano())
	observedCtx := workflow.WithEventObserver(ctx, workflow.EventObserverFunc(func(event workflow.Event) {
		if event.Summary != nil {
			storedRun, ok := s.store.GetStoredRun(runID)
			if ok {
				storedRun.Summary = *event.Summary
				_ = s.store.SaveRun(storedRun)
			} else {
				_ = s.store.Save(runID, *event.Summary)
			}
		}
		s.events.Publish(newStreamEvent(runID, event))
	}))

	_, _, _, wfBase, runner := s.snapshotRuntime()
	wf := applyRunRequest(wfBase, request)
	summary, runErr := runner.Run(observedCtx, wf)

	run := storage.StoredRun{
		ID:      runID,
		Summary: summary,
	}
	if err := s.store.SaveRun(run); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("save run: %w", err))
		return
	}
	response := runResponse{Run: run}
	if runErr != nil {
		response.Error = runErr.Error()
	}
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("当前响应不支持事件流"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	events, unsubscribe := s.events.Subscribe()
	defer unsubscribe()

	if err := writeSSE(w, readyEvent()); err != nil {
		return
	}
	flusher.Flush()

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writeSSE(w, event); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := io.WriteString(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func resolveRunTimeout(timeoutSeconds int) time.Duration {
	if timeoutSeconds <= 0 {
		return defaultRunTimeout
	}
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout < minRunTimeout {
		return minRunTimeout
	}
	if timeout > maxRunTimeout {
		return maxRunTimeout
	}
	return timeout
}

func parseRunRequest(r *http.Request) (runRequest, error) {
	if r.Body == nil {
		return runRequest{}, nil
	}
	defer r.Body.Close()

	var payload runRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		if errors.Is(err, io.EOF) {
			return runRequest{}, nil
		}
		return runRequest{}, fmt.Errorf("请求体不是有效 JSON：%w", err)
	}
	return payload, nil
}

func applyRunRequest(base domain.Workflow, request runRequest) domain.Workflow {
	wf := base
	wf.GlobalVars = cloneVars(base.GlobalVars)

	for key, value := range request.Vars {
		wf.GlobalVars[key] = value
	}

	target := strings.TrimSpace(request.Target)
	if target == "" {
		targeting.Ensure(wf.GlobalVars)
		return wf
	}
	targeting.ApplyOverride(wf.GlobalVars, target)
	return wf
}

func cloneVars(vars map[string]any) map[string]any {
	if vars == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(vars))
	for key, value := range vars {
		cloned[key] = value
	}
	return cloned
}

func extractTargetHost(raw string) string {
	return targeting.Parse(raw).Host
}

func normalizeTargetURL(raw string) string {
	return targeting.NormalizeURL(raw)
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	limit := 20
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			writeError(w, http.StatusBadRequest, fmt.Errorf("limit 参数必须是正整数"))
			return
		}
		limit = parsedLimit
	}

	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	targetID := strings.TrimSpace(r.URL.Query().Get("target_id"))

	runs := s.store.List(limit)
	if projectID != "" || targetID != "" {
		runs = s.store.ListByFilter(projectID, targetID, limit)
	}

	writeJSON(w, http.StatusOK, runsResponse{Runs: runs})
}

func (s *Server) handleRunByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	runID := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	if runID == "" || strings.Contains(runID, "/") {
		writeError(w, http.StatusNotFound, fmt.Errorf("未找到运行记录"))
		return
	}

	run, ok := s.store.GetStoredRun(runID)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("未找到运行记录 %q", runID))
		return
	}
	writeJSON(w, http.StatusOK, runResponse{Run: run})
}

func (s *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	toolsPath, workflowPath, _, _, _ := s.snapshotRuntime()
	if err := s.reloadRuntimeFromDisk(toolsPath, workflowPath); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, s.currentStateResponse())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	_, _, _, wf, _ := s.snapshotRuntime()
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"workflow": wf.Name,
	})
}

func (s *Server) handleToolFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	limit := defaultToolFileLimit
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, fmt.Errorf("limit 参数必须是正整数"))
			return
		}
		if parsed > maxToolFileLimit {
			parsed = maxToolFileLimit
		}
		limit = parsed
	}

	root := s.externalToolsRoot()
	files, truncated, err := listToolFiles(root, defaultToolFileMaxDepth, limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toolFilesResponse{
		Root:      filepath.Clean(root),
		Limit:     limit,
		Truncated: truncated,
		Files:     files,
	})
}

func (s *Server) currentStateResponse() stateResponse {
	toolsPath, workflowPath, tools, wf, _ := s.snapshotRuntime()
	response := stateResponse{
		Workflow:     wf,
		Tools:        sortedTools(tools),
		Paths:        configPaths{Tools: filepath.Clean(toolsPath), Workflow: filepath.Clean(workflowPath)},
		ExternalRoot: filepath.Clean(s.externalToolsRoot()),
		RecentRuns:   s.store.List(8),
		WorkflowMeta: summarizeWorkflow(wf),
	}
	if latest, ok := s.store.Latest(); ok {
		response.LatestRun = &latest
	}
	return response
}

func (s *Server) snapshotRuntime() (string, string, map[string]domain.ToolDefinition, domain.Workflow, *workflow.Runner) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.toolsPath, s.workflowPath, s.tools, s.workflow, s.runner
}

func (s *Server) externalToolsRoot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.externalRoot
}

func (s *Server) reloadRuntimeFromDisk(toolsPath, workflowPath string) error {
	tools, wf, runner, err := loadRuntime(toolsPath, workflowPath)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.tools = tools
	s.workflow = wf
	s.runner = runner
	s.mu.Unlock()
	return nil
}

func loadRuntime(toolsPath, workflowPath string) (map[string]domain.ToolDefinition, domain.Workflow, *workflow.Runner, error) {
	tools, err := config.LoadToolCatalog(toolsPath)
	if err != nil {
		return nil, domain.Workflow{}, nil, fmt.Errorf("load tools: %w", err)
	}
	wf, err := config.LoadWorkflow(workflowPath)
	if err != nil {
		return nil, domain.Workflow{}, nil, fmt.Errorf("load workflow: %w", err)
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
	return tools, wf, runner, nil
}

func summarizeWorkflow(wf domain.Workflow) workflowMeta {
	incoming := make(map[string]int, len(wf.Nodes))
	for _, node := range wf.Nodes {
		incoming[node.ID] = 0
	}
	for _, edge := range wf.Edges {
		incoming[edge.To]++
	}
	entryNodes := make([]string, 0, len(incoming))
	for _, node := range wf.Nodes {
		if incoming[node.ID] == 0 {
			entryNodes = append(entryNodes, node.ID)
		}
	}
	sort.Strings(entryNodes)
	return workflowMeta{
		NodeCount:  len(wf.Nodes),
		EdgeCount:  len(wf.Edges),
		EntryNodes: entryNodes,
	}
}

func sortedTools(tools map[string]domain.ToolDefinition) []domain.ToolDefinition {
	list := make([]domain.ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		list = append(list, tool)
	}
	sort.Slice(list, func(i, j int) bool {
		return strings.Compare(list[i].Name, list[j].Name) < 0
	})
	return list
}

func listToolFiles(root string, maxDepth, maxEntries int) ([]toolFileEntry, bool, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, false, fmt.Errorf("读取外部工具目录失败：%w", err)
	}
	if !info.IsDir() {
		return nil, false, fmt.Errorf("外部工具目录不是文件夹：%s", root)
	}
	if maxEntries <= 0 {
		maxEntries = defaultToolFileLimit
	}

	limitHitErr := errors.New("tool file listing reached max entries")
	files := make([]toolFileEntry, 0, min(maxEntries, 64))
	truncated := false

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		depth := strings.Count(filepath.ToSlash(relPath), "/") + 1
		if entry.IsDir() {
			if maxDepth > 0 && depth > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if maxDepth > 0 && depth > maxDepth {
			return nil
		}
		if len(files) >= maxEntries {
			truncated = true
			return limitHitErr
		}

		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}

		category := "根目录"
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		if len(parts) > 1 {
			category = parts[0]
		}
		files = append(files, toolFileEntry{
			Name:         fileInfo.Name(),
			RelativePath: filepath.ToSlash(relPath),
			AbsolutePath: filepath.Clean(path),
			Category:     category,
			Extension:    strings.ToLower(filepath.Ext(fileInfo.Name())),
			Kind:         classifyToolFile(fileInfo.Name()),
			SizeBytes:    fileInfo.Size(),
		})
		return nil
	})
	if err != nil && !errors.Is(err, limitHitErr) {
		return nil, false, fmt.Errorf("遍历外部工具目录失败：%w", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return strings.Compare(files[i].RelativePath, files[j].RelativePath) < 0
	})
	return files, truncated, nil
}

func classifyToolFile(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".exe"), strings.HasSuffix(lower, ".jar"), strings.HasSuffix(lower, ".bat"), strings.HasSuffix(lower, ".cmd"), strings.HasSuffix(lower, ".ps1"), strings.HasSuffix(lower, ".sh"), strings.HasSuffix(lower, ".py"):
		return "可启动文件"
	case strings.HasSuffix(lower, ".zip"), strings.HasSuffix(lower, ".7z"), strings.HasSuffix(lower, ".rar"), strings.HasSuffix(lower, ".tar"), strings.HasSuffix(lower, ".gz"):
		return "压缩包"
	case strings.HasSuffix(lower, ".md"), strings.HasSuffix(lower, ".txt"), strings.HasSuffix(lower, ".json"), strings.HasSuffix(lower, ".yaml"), strings.HasSuffix(lower, ".yml"):
		return "文档"
	default:
		return "文件"
	}
}

func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
		"error": fmt.Sprintf("请求方法不允许，请使用 %s", allowed),
	})
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}
