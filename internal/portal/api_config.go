package portal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"penframe/internal/config"
	"penframe/internal/domain"
)

type toolPathEntry struct {
	Tool      string `json:"tool"`
	Label     string `json:"label"`
	VarName   string `json:"var_name"`
	Value     string `json:"value"`
	Default   string `json:"default"`
	Source    string `json:"source"`
}

type toolPathConfigResponse struct {
	WorkflowPath string          `json:"workflow_path"`
	Items        []toolPathEntry `json:"items"`
}

type updateToolPathConfigRequest struct {
	Paths map[string]string `json:"paths"`
}

func (s *Server) handleToolPathConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.currentToolPathConfig())
	case http.MethodPut:
		s.handleUpdateToolPathConfig(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPut)
		methodNotAllowed(w, http.MethodGet)
	}
}

func (s *Server) handleUpdateToolPathConfig(w http.ResponseWriter, r *http.Request) {
	var req updateToolPathConfigRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
			return
		}
	}

	toolsPath, workflowPath, tools, wf, _ := s.snapshotRuntime()
	allowed := configurableToolPathMap(tools, wf.GlobalVars)
	for _, entry := range proxyConfigEntries(wf.GlobalVars) {
		allowed[entry.VarName] = entry
	}
	updates := make(map[string]string, len(req.Paths))
	for key, value := range req.Paths {
		item, ok := allowed[key]
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			trimmed = item.Default
		}
		updates[key] = trimmed
	}
	if len(updates) == 0 {
		writeJSON(w, http.StatusOK, s.currentToolPathConfig())
		return
	}

	if err := config.UpdateWorkflowGlobalVars(workflowPath, updates); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("update workflow tool paths: %w", err))
		return
	}
	if err := s.reloadRuntimeFromDisk(toolsPath, workflowPath); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, s.currentToolPathConfig())
}

func (s *Server) currentToolPathConfig() toolPathConfigResponse {
	_, workflowPath, tools, wf, _ := s.snapshotRuntime()
	items := configurableToolPaths(tools, wf.GlobalVars)
	return toolPathConfigResponse{
		WorkflowPath: workflowPath,
		Items:        items,
	}
}

func configurableToolPaths(tools map[string]domain.ToolDefinition, vars map[string]any) []toolPathEntry {
	itemsByVar := configurableToolPathMap(tools, vars)
	for _, entry := range proxyConfigEntries(vars) {
		itemsByVar[entry.VarName] = entry
	}
	keys := make([]string, 0, len(itemsByVar))
	for key := range itemsByVar {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]toolPathEntry, 0, len(keys))
	for _, key := range keys {
		items = append(items, itemsByVar[key])
	}
	return items
}

func configurableToolPathMap(tools map[string]domain.ToolDefinition, vars map[string]any) map[string]toolPathEntry {
	items := make(map[string]toolPathEntry)
	for name, tool := range tools {
		varName := strings.TrimSpace(fmt.Sprint(tool.Metadata["binary_var"]))
		if varName == "" {
			continue
		}
		defaultValue := strings.TrimSpace(fmt.Sprint(tool.Metadata["binary_path"]))
		label := strings.TrimSpace(fmt.Sprint(tool.Metadata["binary_label"]))
		if label == "" {
			label = name
		}
		currentValue := mapValueString(vars, varName)
		if currentValue == "" {
			currentValue = defaultValue
		}
		if existing, ok := items[varName]; ok {
			if existing.Value == "" && currentValue != "" {
				existing.Value = currentValue
			}
			items[varName] = existing
			continue
		}
		items[varName] = toolPathEntry{
			Tool:    name,
			Label:   label,
			VarName: varName,
			Value:   currentValue,
			Default: defaultValue,
			Source:  strings.TrimSpace(fmt.Sprint(tool.Metadata["source"])),
		}
	}
	return items
}

func proxyConfigEntries(vars map[string]any) []toolPathEntry {
	return []toolPathEntry{
		{
			Tool:    "global",
			Label:   "HTTP Proxy",
			VarName: "http_proxy",
			Value:   mapValueString(vars, "http_proxy"),
			Default: "",
			Source:  "workflow",
		},
		{
			Tool:    "global",
			Label:   "SOCKS5 Proxy",
			VarName: "socks5_proxy",
			Value:   mapValueString(vars, "socks5_proxy"),
			Default: "",
			Source:  "workflow",
		},
	}
}

func mapValueString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
