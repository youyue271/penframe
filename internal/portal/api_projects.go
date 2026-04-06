package portal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type createProjectRequest struct {
	Name string `json:"name"`
}

type addTargetRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type updateTargetRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListProjects(w, r)
	case http.MethodPost:
		s.handleCreateProject(w, r)
	default:
		w.Header().Set("Allow", fmt.Sprintf("%s, %s", http.MethodGet, http.MethodPost))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	projects := s.projects.List()
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	defer r.Body.Close()

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("name is required"))
		return
	}

	p, err := s.projects.Add(req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to add project: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w, http.MethodDelete)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusBadRequest, fmt.Errorf("project id is required"))
		return
	}

	if err := s.projects.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to delete project: %w", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleProjectTargets(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from path like /api/projects/{id}/targets
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	path = strings.TrimSuffix(path, "/targets")
	projectID := path

	if projectID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("project id is required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleListTargets(w, r, projectID)
	case http.MethodPost:
		s.handleAddTarget(w, r, projectID)
	default:
		w.Header().Set("Allow", fmt.Sprintf("%s, %s", http.MethodGet, http.MethodPost))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleListTargets(w http.ResponseWriter, r *http.Request, projectID string) {
	project, err := s.projects.Get(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"targets": project.Targets})
}

func (s *Server) handleAddTarget(w http.ResponseWriter, r *http.Request, projectID string) {
	var req addTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	defer r.Body.Close()

	if req.Name == "" || req.URL == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("name and url are required"))
		return
	}

	target, err := s.projects.AddTarget(projectID, req.Name, req.URL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to add target: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, target)
}

func (s *Server) handleProjectTarget(w http.ResponseWriter, r *http.Request) {
	// Extract IDs from path like /api/projects/{pid}/targets/{tid}
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.Split(path, "/targets/")
	if len(parts) != 2 || parts[1] == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid path format"))
		return
	}

	projectID := parts[0]
	targetID := parts[1]

	if projectID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("project id and target id are required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetTarget(w, r, projectID, targetID)
	case http.MethodPut:
		s.handleUpdateTarget(w, r, projectID, targetID)
	case http.MethodDelete:
		s.handleDeleteTarget(w, r, projectID, targetID)
	default:
		w.Header().Set("Allow", fmt.Sprintf("%s, %s, %s", http.MethodGet, http.MethodPut, http.MethodDelete))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleUpdateTarget(w http.ResponseWriter, r *http.Request, projectID, targetID string) {
	var req updateTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	defer r.Body.Close()

	if err := s.projects.UpdateTarget(targetID, req.Name, req.URL); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to update target: %w", err))
		return
	}

	target, err := s.projects.GetTarget(targetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, target)
}

func (s *Server) handleGetTarget(w http.ResponseWriter, r *http.Request, projectID, targetID string) {
	target, err := s.projects.GetTarget(targetID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	writeJSON(w, http.StatusOK, target)
}

func (s *Server) handleDeleteTarget(w http.ResponseWriter, r *http.Request, projectID, targetID string) {
	if err := s.projects.DeleteTarget(projectID, targetID); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to delete target: %w", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
