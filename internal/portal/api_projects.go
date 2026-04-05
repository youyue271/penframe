package portal

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type createProjectRequest struct {
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

	if req.Name == "" || req.URL == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("name and url are required"))
		return
	}

	p, err := s.projects.Add(req.Name, req.URL)
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

	id := extractPathSegment(r.URL.Path, "/api/projects/")
	if id == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("project id is required"))
		return
	}

	if err := s.projects.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to delete project: %w", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
