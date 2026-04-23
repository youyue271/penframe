package cveindex

import (
	"encoding/json"
	"sort"
	"strings"
)

// ExpModuleInfo is the subset of Python exploit module info we need.
type ExpModuleInfo struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	CVE             string   `json:"cve"`
	Severity        string   `json:"severity"`
	Tags            []string `json:"tags"`
	SupportsCheck   bool     `json:"supports_check"`
	SupportsExploit bool     `json:"supports_exploit"`
	SupportsCommand bool     `json:"supports_command"`
	ExploitKind     string   `json:"exploit_kind"`
	DefaultCommand  string   `json:"default_command"`
}

// CVEEntry is the unified view of a CVE combining nuclei templates and exp module.
type CVEEntry struct {
	ID        string           `json:"id"`
	Vendor    string           `json:"vendor"`
	Severity  string           `json:"severity"`
	Tags      []string         `json:"tags"`
	Templates []NucleiTemplate `json:"templates"`
	ExpModule *ExpModuleInfo   `json:"exp_module"`
	Tested    bool             `json:"tested"`
}

// Aggregate merges nuclei templates, exp modules, and tested status into CVEEntry list.
func Aggregate(templates []NucleiTemplate, expModulesJSON []byte, tested map[string]bool) []CVEEntry {
	entries := make(map[string]*CVEEntry)

	for _, t := range templates {
		key := t.CVEID
		e, ok := entries[key]
		if !ok {
			e = &CVEEntry{
				ID:     key,
				Vendor: t.Vendor,
			}
			entries[key] = e
		}
		e.Templates = append(e.Templates, t)
		mergeTags(e, t.Tags)
		if higherSeverity(t.Severity, e.Severity) {
			e.Severity = t.Severity
		}
	}

	var expList struct {
		Exploits []json.RawMessage `json:"exploits"`
	}
	if len(expModulesJSON) > 0 {
		_ = json.Unmarshal(expModulesJSON, &expList)
	}
	for _, raw := range expList.Exploits {
		var mod ExpModuleInfo
		if err := json.Unmarshal(raw, &mod); err != nil {
			continue
		}
		cveID := strings.ToUpper(mod.CVE)
		if cveID == "" {
			cveID = strings.ToUpper(strings.ReplaceAll(mod.ID, "-", "-"))
			if !strings.HasPrefix(cveID, "CVE-") {
				continue
			}
		}
		e, ok := entries[cveID]
		if !ok {
			e = &CVEEntry{ID: cveID}
			entries[cveID] = e
		}
		e.ExpModule = &mod
		mergeTags(e, strings.Join(mod.Tags, ","))
		if higherSeverity(mod.Severity, e.Severity) {
			e.Severity = mod.Severity
		}
	}

	for id, e := range entries {
		e.Tested = tested[id]
	}

	result := make([]CVEEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, *e)
	}
	sort.Slice(result, func(i, j int) bool {
		si := severityRank(result[i].Severity)
		sj := severityRank(result[j].Severity)
		if si != sj {
			return si > sj
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func mergeTags(e *CVEEntry, tagStr string) {
	existing := make(map[string]bool, len(e.Tags))
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

var severityOrder = map[string]int{
	"critical": 4, "high": 3, "medium": 2, "low": 1, "info": 0,
}

func severityRank(s string) int {
	return severityOrder[strings.ToLower(s)]
}

func higherSeverity(a, b string) bool {
	return severityRank(a) > severityRank(b)
}
