package portal

import (
	"fmt"
	"net/http"
)

type assetGraphResponse struct {
	RunID    string `json:"run_id"`
	Target   string `json:"target"`
	Summary  map[string]int `json:"summary"`
	Elements any    `json:"elements"`
}

type assetHostResponse struct {
	Host any `json:"host"`
}

func (s *Server) handleAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	graph, runID, ok := s.assets.Latest()
	if !ok {
		writeJSON(w, http.StatusOK, assetGraphResponse{
			Summary:  map[string]int{"hosts": 0, "ports": 0, "paths": 0, "vulns": 0},
			Elements: []any{},
		})
		return
	}

	writeJSON(w, http.StatusOK, assetGraphResponse{
		RunID:    runID,
		Target:   graph.TargetRaw,
		Summary:  graph.Summary(),
		Elements: graph.ToCytoscapeJSON(),
	})
}

func (s *Server) handleAssetsByRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	// Expect /api/assets/{runID}
	runID := extractPathSegment(r.URL.Path, "/api/assets/")
	if runID == "" {
		writeError(w, http.StatusBadRequest, errorf("missing run_id"))
		return
	}

	graph, ok := s.assets.Get(runID)
	if !ok {
		writeError(w, http.StatusNotFound, errorf("no assets for run %q", runID))
		return
	}

	writeJSON(w, http.StatusOK, assetGraphResponse{
		RunID:    runID,
		Target:   graph.TargetRaw,
		Summary:  graph.Summary(),
		Elements: graph.ToCytoscapeJSON(),
	})
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	tasks := s.assets.ListTasks()
	result := make([]any, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, t)
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": result})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	// Placeholder: return empty log list.
	writeJSON(w, http.StatusOK, map[string]any{"logs": []any{}})
}

// cors middleware to allow Vue dev server connections.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractPathSegment(path, prefix string) string {
	if len(path) <= len(prefix) {
		return ""
	}
	seg := path[len(prefix):]
	// Strip trailing slash.
	for len(seg) > 0 && seg[len(seg)-1] == '/' {
		seg = seg[:len(seg)-1]
	}
	return seg
}

func errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
