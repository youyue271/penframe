package portal

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"penframe/internal/cveindex"
)

func (s *Server) handleCVEs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	nucleiRoot := filepath.Join("cve", "nuclei", "cves")
	templates := cveindex.ScanNucleiTemplates(nucleiRoot)

	var expData []byte
	if exp := s.getExpExecutor(); exp != nil {
		data, err := exp.ListExploits(r.Context())
		if err == nil {
			expData = data
		}
	}

	tested := s.cveStore.TestedSet()
	entries := aggregate(templates, expData, tested)

	filter := r.URL.Query().Get("filter")
	switch filter {
	case "tested":
		entries = filterEntries(entries, func(e cveJSON) bool { return e.Tested })
	case "untested":
		entries = filterEntries(entries, func(e cveJSON) bool { return !e.Tested })
	}

	if tag := r.URL.Query().Get("tag"); tag != "" {
		entries = filterEntries(entries, func(e cveJSON) bool {
			for _, t := range e.Tags {
				if strings.EqualFold(t, tag) {
					return true
				}
			}
			return false
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"cves": entries, "count": len(entries)})
}

func (s *Server) handleCVETested(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/cves/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] != "tested" {
		http.NotFound(w, r)
		return
	}
	cveID := strings.ToUpper(parts[0])

	switch r.Method {
	case http.MethodPut:
		if err := s.cveStore.SetTested(cveID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"cve_id": cveID, "tested": true})
	case http.MethodDelete:
		if err := s.cveStore.SetUntested(cveID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"cve_id": cveID, "tested": false})
	default:
		w.Header().Set("Allow", "PUT, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCVETags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	nucleiRoot := filepath.Join("cve", "nuclei", "cves")
	templates := cveindex.ScanNucleiTemplates(nucleiRoot)

	var expData []byte
	if exp := s.getExpExecutor(); exp != nil {
		data, err := exp.ListExploits(r.Context())
		if err == nil {
			expData = data
		}
	}

	entries := aggregate(templates, expData, nil)

	counts := map[string]int{}
	for _, e := range entries {
		for _, t := range e.Tags {
			counts[t]++
		}
	}

	type tagInfo struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	tags := make([]tagInfo, 0, len(counts))
	for name, c := range counts {
		tags = append(tags, tagInfo{Name: name, Count: c})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Count > tags[j].Count })

	writeJSON(w, http.StatusOK, map[string]any{"tags": tags})
}

// --- aggregation (inline, avoids extra package dependency for JSON shaping) ---

type cveJSON struct {
	ID        string            `json:"id"`
	Vendor    string            `json:"vendor"`
	Severity  string            `json:"severity"`
	Tags      []string          `json:"tags"`
	Templates []templateJSON    `json:"templates"`
	ExpModule json.RawMessage   `json:"exp_module"`
	Tested    bool              `json:"tested"`
}

type templateJSON struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Severity string `json:"severity"`
	CVEID    string `json:"cve_id"`
	Vendor   string `json:"vendor"`
	Tags     string `json:"tags"`
}

func aggregate(templates []cveindex.NucleiTemplate, expData []byte, tested map[string]bool) []cveJSON {
	entries := map[string]*cveJSON{}

	for _, t := range templates {
		e, ok := entries[t.CVEID]
		if !ok {
			e = &cveJSON{ID: t.CVEID, Vendor: t.Vendor, Tags: []string{}}
			entries[t.CVEID] = e
		}
		e.Templates = append(e.Templates, templateJSON{
			ID: t.ID, Name: t.Name, Severity: t.Severity,
			CVEID: t.CVEID, Vendor: t.Vendor, Tags: t.Tags,
		})
		addTags(e, t.Tags)
		if sevRank(t.Severity) > sevRank(e.Severity) {
			e.Severity = t.Severity
		}
	}

	var expList struct {
		Exploits []json.RawMessage `json:"exploits"`
	}
	if len(expData) > 0 {
		_ = json.Unmarshal(expData, &expList)
	}
	for _, raw := range expList.Exploits {
		var mod struct {
			ID       string   `json:"id"`
			CVE      string   `json:"cve"`
			Severity string   `json:"severity"`
			Tags     []string `json:"tags"`
		}
		if json.Unmarshal(raw, &mod) != nil {
			continue
		}
		cveID := strings.ToUpper(mod.CVE)
		if cveID == "" {
			continue
		}
		e, ok := entries[cveID]
		if !ok {
			e = &cveJSON{ID: cveID, Tags: []string{}}
			entries[cveID] = e
		}
		e.ExpModule = raw
		addTags(e, strings.Join(mod.Tags, ","))
		if sevRank(mod.Severity) > sevRank(e.Severity) {
			e.Severity = mod.Severity
		}
	}

	if tested == nil {
		tested = map[string]bool{}
	}

	result := make([]cveJSON, 0, len(entries))
	for id, e := range entries {
		e.Tested = tested[id]
		if e.Templates == nil {
			e.Templates = []templateJSON{}
		}
		result = append(result, *e)
	}

	sort.Slice(result, func(i, j int) bool {
		si, sj := sevRank(result[i].Severity), sevRank(result[j].Severity)
		if si != sj {
			return si > sj
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func filterEntries(entries []cveJSON, fn func(cveJSON) bool) []cveJSON {
	var out []cveJSON
	for _, e := range entries {
		if fn(e) {
			out = append(out, e)
		}
	}
	return out
}

func addTags(e *cveJSON, tagStr string) {
	existing := map[string]bool{}
	for _, t := range e.Tags {
		existing[t] = true
	}
	for _, t := range strings.Split(tagStr, ",") {
		t = strings.TrimSpace(t)
		if t != "" && !existing[t] {
			e.Tags = append(e.Tags, t)
			existing[t] = true
		}
	}
}

var sevMap = map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1, "info": 0}

func sevRank(s string) int { return sevMap[strings.ToLower(s)] }
