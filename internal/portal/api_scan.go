package portal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"penframe/internal/domain"
	"penframe/internal/scanner"
	"penframe/internal/storage"
	"penframe/internal/workflow"
)

type scanRequest struct {
	Target         string         `json:"target"`
	Strategy       string         `json:"strategy"` // full/discovery/recon
	Vars           map[string]any `json:"vars,omitempty"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
}

type scanResponse struct {
	RunID  string             `json:"run_id"`
	Tasks  []*domain.ScanTask `json:"tasks"`
	Input  scanner.ParsedInput `json:"input"`
	Run    *storage.StoredRun  `json:"run,omitempty"`
	Error  string             `json:"error,omitempty"`
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

	// Choose strategy.
	strat := scanner.FullStrategy()
	switch req.Strategy {
	case "discovery":
		strat = scanner.DiscoveryOnlyStrategy()
	case "recon":
		strat = scanner.ReconStrategy()
	}

	runID := fmt.Sprintf("scan-%d", time.Now().UTC().UnixNano())

	// Generate initial tasks.
	tasks := scanner.GenerateInitialTasks(input, strat, runID)
	for _, t := range tasks {
		s.assets.AddTask(t)
	}

	// Create asset graph for this run.
	s.assets.GetOrCreate(runID, req.Target)

	// Also run the existing workflow for backward compatibility.
	go s.executeScanWorkflow(runID, req)

	writeJSON(w, http.StatusCreated, scanResponse{
		RunID: runID,
		Tasks: tasks,
		Input: input,
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

	runMarkedRunning := false
	observedCtx := workflow.WithEventObserver(ctx, workflow.EventObserverFunc(func(event workflow.Event) {
		if event.Summary != nil {
			s.store.Save(runID, *event.Summary)
		}

		switch event.Type {
		case workflow.EventRunStarted, workflow.EventNodeStarted:
			if !runMarkedRunning {
				s.assets.UpdatePendingTasksByRun(runID, domain.ScanTaskRunning)
				runMarkedRunning = true
			}
		case workflow.EventRunFinished:
			if event.Summary != nil && event.Summary.Status == domain.RunStatusSucceeded {
				s.assets.FinalizeTasksByRun(runID, domain.ScanTaskDone, "")
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
		ID:      runID,
		Summary: summary,
	}
	s.store.Save(run.ID, run.Summary)

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
