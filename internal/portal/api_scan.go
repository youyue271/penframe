package portal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"penframe/internal/domain"
	"penframe/internal/scanner"
	"penframe/internal/storage"
	"penframe/internal/workflow"
)

type scanRequest struct {
	Target         string            `json:"target"`
	Strategy       string            `json:"strategy"` // full/discovery/recon/custom
	Phases         []string          `json:"phases,omitempty"`
	Tools          map[string]string `json:"tools,omitempty"`
	Vars           map[string]any    `json:"vars,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
	ProjectID      string            `json:"project_id,omitempty"`
	TargetID       string            `json:"target_id,omitempty"`
}

type scanResponse struct {
	RunID string              `json:"run_id"`
	Tasks []*domain.ScanTask  `json:"tasks"`
	Input scanner.ParsedInput `json:"input"`
	Run   *storage.StoredRun  `json:"run,omitempty"`
	Error string              `json:"error,omitempty"`
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req scanRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
			return
		}
	}

	if req.Target == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("target is required"))
		return
	}

	// Classify input.
	input := scanner.ClassifyInput(req.Target)

	runID := fmt.Sprintf("scan-%d", time.Now().UTC().UnixNano())
	initialRun := s.newInitialScanRun(runID, req)

	// Store project/target context if provided
	if req.ProjectID != "" || req.TargetID != "" {
		initialRun.ProjectID = req.ProjectID
		initialRun.TargetID = req.TargetID
	}

	// Generate initial tasks.
	tasks := s.newInitialScanTasks(runID, req)
	for _, t := range tasks {
		s.assets.AddTask(t)
	}

	// Create asset graph for this run.
	s.assets.GetOrCreate(runID, req.Target)

	if err := s.store.SaveRun(initialRun); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("save scan run: %w", err))
		return
	}
	if req.TargetID != "" {
		_ = s.projects.UpdateTargetLastScanned(req.TargetID, time.Now().UTC())
	}

	// Also run the existing workflow for backward compatibility.
	go s.executeScanWorkflow(runID, req)

	writeJSON(w, http.StatusCreated, scanResponse{
		RunID: runID,
		Tasks: tasks,
		Input: input,
		Run:   &initialRun,
	})
}

func (s *Server) handleScanAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	// Expect /api/scan/{id}/action
	path := r.URL.Path
	seg := extractPathSegment(path, "/api/scan/")
	// seg should be like "scan-xxx/action"
	parts := splitFirst(seg, "/")
	if len(parts) < 2 || parts[1] != "action" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("expected /api/scan/{id}/action"))
		return
	}

	var body struct {
		Action string `json:"action"` // scan_ports, scan_paths, scan_vulns, exploit, skip
		Target string `json:"target"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		json.NewDecoder(r.Body).Decode(&body)
	}

	// Create a follow-up task.
	taskType := domain.ScanTypePortScan
	switch body.Action {
	case "scan_ports":
		taskType = domain.ScanTypePortScan
	case "scan_paths":
		taskType = domain.ScanTypePathScan
	case "scan_vulns":
		taskType = domain.ScanTypeVulnScan
	case "exploit":
		taskType = domain.ScanTypeExploit
	case "skip":
		writeJSON(w, http.StatusOK, map[string]string{"status": "skipped"})
		return
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unknown action %q", body.Action))
		return
	}

	task := &domain.ScanTask{
		ID:       fmt.Sprintf("task-%s-%d", taskType, time.Now().UnixNano()),
		Type:     taskType,
		Target:   body.Target,
		Status:   domain.ScanTaskPending,
		ParentID: parts[0],
	}
	s.assets.AddTask(task)

	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) executeScanWorkflow(runID string, req scanRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), resolveRunTimeout(req.TimeoutSeconds))
	defer cancel()

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

		switch event.Type {
		case workflow.EventNodeStarted:
			if event.Node != nil {
				s.assets.UpdateTaskByRunNode(runID, event.Node.NodeID, domain.ScanTaskRunning, "")
			}
		case workflow.EventNodeFinished:
			if event.Node != nil {
				s.assets.UpdateTaskByRunNode(runID, event.Node.NodeID, mapNodeStatusToTaskStatus(event.Node.Status), event.Node.Error)
			}
		case workflow.EventRunFinished:
			if event.Summary != nil && event.Summary.Status == domain.RunStatusSucceeded {
				s.assets.FinalizeTasksByRun(runID, domain.ScanTaskSkipped, "")
			} else {
				errMsg := "scan workflow failed"
				if event.Summary != nil && event.Summary.Error != "" {
					errMsg = event.Summary.Error
				}
				s.assets.FinalizeTasksByRun(runID, domain.ScanTaskFailed, errMsg)
			}
		}

		s.events.Publish(newStreamEvent(runID, event))
	}))

	_, _, _, wfBase, runner := s.snapshotRuntime()
	wf := applyRunRequest(wfBase, runRequest{
		Target: req.Target,
		Vars:   req.Vars,
	})
	summary, runErr := runner.Run(observedCtx, wf)

	run := storage.StoredRun{
		ID:        runID,
		ProjectID: req.ProjectID,
		TargetID:  req.TargetID,
		Summary:   summary,
	}
	_ = s.store.SaveRun(run)

	if runErr != nil {
		errMsg := runErr.Error()
		if summary.Error != "" {
			errMsg = summary.Error
		}
		s.assets.FinalizeTasksByRun(runID, domain.ScanTaskFailed, errMsg)
	}

	// Publish final event.
	if runErr != nil {
		s.events.Publish(streamEvent{
			Type:      "scan_error",
			RunID:     runID,
			Timestamp: time.Now().UTC().UnixMilli(),
		})
	}
}

func buildCustomStrategy(phases []string) scanner.Strategy {
	strat := scanner.Strategy{
		HostDiscovery: false,
		PortScan:      false,
		PathScan:      false,
		VulnScan:      false,
		Exploit:       false,
	}
	for _, phase := range phases {
		switch phase {
		case "host_discovery":
			strat.HostDiscovery = true
		case "port_scan":
			strat.PortScan = true
		case "path_scan":
			strat.PathScan = true
		case "vuln_scan":
			strat.VulnScan = true
		case "exploit":
			strat.Exploit = true
		}
	}
	return strat
}

func (s *Server) newInitialScanRun(runID string, req scanRequest) storage.StoredRun {
	_, _, _, wfBase, _ := s.snapshotRuntime()
	wf := applyRunRequest(wfBase, runRequest{
		Target: req.Target,
		Vars:   req.Vars,
	})
	startedAt := time.Now().UTC()
	return storage.StoredRun{
		ID: runID,
		Summary: domain.RunSummary{
			Workflow:       wf.Name,
			Status:         domain.RunStatusRunning,
			StartedAt:      startedAt,
			Vars:           cloneVars(wf.GlobalVars),
			Assets:         map[string]any{},
			NodeResults:    map[string]domain.NodeRunResult{},
			ExecutionOrder: []string{},
			Stats: domain.RunStats{
				TotalNodes: len(wf.Nodes),
			},
		},
	}
}

func splitFirst(s, sep string) []string {
	idx := 0
	for i := range s {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			return []string{s[:i], s[i+len(sep):]}
		}
		idx++
	}
	return []string{s}
}

func (s *Server) newInitialScanTasks(runID string, req scanRequest) []*domain.ScanTask {
	_, _, tools, wfBase, _ := s.snapshotRuntime()
	wf := applyRunRequest(wfBase, runRequest{
		Target: req.Target,
		Vars:   req.Vars,
	})
	tasks := buildWorkflowTasks(runID, strings.TrimSpace(req.Target), wf, tools)
	if len(tasks) > 0 {
		return tasks
	}

	input := scanner.ClassifyInput(req.Target)
	strat := scanner.FullStrategy()
	switch req.Strategy {
	case "discovery":
		strat = scanner.DiscoveryOnlyStrategy()
	case "recon":
		strat = scanner.ReconStrategy()
	case "custom":
		strat = buildCustomStrategy(req.Phases)
	}
	return scanner.GenerateInitialTasks(input, strat, runID)
}

func buildWorkflowTasks(runID, target string, wf domain.Workflow, tools map[string]domain.ToolDefinition) []*domain.ScanTask {
	tasks := make([]*domain.ScanTask, 0, len(wf.Nodes))
	seqBase := time.Now().UTC().UnixNano()
	for idx, node := range wf.Nodes {
		if tool, ok := tools[node.Tool]; ok && tool.Category == "orchestration" {
			continue
		}
		taskTarget := strings.TrimSpace(target)
		if taskTarget == "" {
			taskTarget = node.ID
		}
		tasks = append(tasks, &domain.ScanTask{
			ID:       fmt.Sprintf("task-node-%d-%d", seqBase, idx),
			Type:     node.Tool,
			Target:   taskTarget,
			Status:   domain.ScanTaskPending,
			ParentID: runID,
			NodeID:   node.ID,
		})
	}
	return tasks
}

func mapNodeStatusToTaskStatus(status string) string {
	switch status {
	case domain.NodeStatusSucceeded:
		return domain.ScanTaskDone
	case domain.NodeStatusFailed:
		return domain.ScanTaskFailed
	case domain.NodeStatusSkipped:
		return domain.ScanTaskSkipped
	case domain.NodeStatusRunning:
		return domain.ScanTaskRunning
	default:
		return domain.ScanTaskPending
	}
}
