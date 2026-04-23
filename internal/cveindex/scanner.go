package cveindex

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var cveIDPattern = regexp.MustCompile(`(?i)CVE-\d{4}-\d{4,}`)

// NucleiTemplate holds metadata parsed from a nuclei YAML template.
type NucleiTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	CVEID       string `json:"cve_id"`
	Vendor      string `json:"vendor"`
	Tags        string `json:"tags"`
	FilePath    string `json:"file_path"`
}

type nucleiYAML struct {
	ID   string `yaml:"id"`
	Info struct {
		Name           string `yaml:"name"`
		Severity       string `yaml:"severity"`
		Description    string `yaml:"description"`
		Tags           string `yaml:"tags"`
		Classification struct {
			CVEID string `yaml:"cve-id"`
		} `yaml:"classification"`
	} `yaml:"info"`
}

// ScanNucleiTemplates walks cveRoot/cves/**/*.yaml and parses each template.
func ScanNucleiTemplates(nucleiCVERoot string) []NucleiTemplate {
	var templates []NucleiTemplate

	_ = filepath.WalkDir(nucleiCVERoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var doc nucleiYAML
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil
		}

		cveID := strings.ToUpper(doc.Info.Classification.CVEID)
		if cveID == "" {
			cveID = extractCVEID(doc.ID)
		}
		if cveID == "" {
			return nil
		}

		vendor := filepath.Base(filepath.Dir(path))

		templates = append(templates, NucleiTemplate{
			ID:          doc.ID,
			Name:        doc.Info.Name,
			Severity:    doc.Info.Severity,
			Description: doc.Info.Description,
			CVEID:       cveID,
			Vendor:      vendor,
			Tags:        doc.Info.Tags,
			FilePath:    path,
		})
		return nil
	})

	return templates
}

func extractCVEID(s string) string {
	m := cveIDPattern.FindString(strings.ToUpper(s))
	return m
}
