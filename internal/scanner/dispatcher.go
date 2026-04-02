package scanner

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync/atomic"

	"penframe/internal/domain"
)

var taskSeq atomic.Int64

func genTaskID(taskType string) string {
	return fmt.Sprintf("task-%s-%d", taskType, taskSeq.Add(1))
}

// InputType classifies the target input.
type InputType int

const (
	InputTypeUnknown InputType = iota
	InputTypeIP
	InputTypeCIDR
	InputTypeDomain
	InputTypeURLNoPort
	InputTypeURLWithPort
)

// ParsedInput holds classified target information.
type ParsedInput struct {
	Raw      string
	Type     InputType
	Host     string
	Port     string
	Scheme   string
	URL      string
	Network  *net.IPNet // only for CIDR
}

// ClassifyInput determines the type of target input.
func ClassifyInput(raw string) ParsedInput {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ParsedInput{Raw: raw, Type: InputTypeUnknown}
	}

	// Check CIDR first.
	if _, network, err := net.ParseCIDR(raw); err == nil {
		return ParsedInput{
			Raw:     raw,
			Type:    InputTypeCIDR,
			Host:    raw,
			Network: network,
		}
	}

	// Check plain IP.
	if ip := net.ParseIP(raw); ip != nil {
		return ParsedInput{
			Raw:  raw,
			Type: InputTypeIP,
			Host: raw,
		}
	}

	// Try parsing as URL.
	candidate := raw
	if !strings.Contains(candidate, "://") {
		candidate = "https://" + candidate
	}
	parsed, err := url.Parse(candidate)
	if err != nil {
		return ParsedInput{Raw: raw, Type: InputTypeDomain, Host: raw}
	}

	host := parsed.Hostname()
	port := parsed.Port()
	scheme := parsed.Scheme

	// If input was just a hostname (no scheme in original).
	if !strings.Contains(raw, "://") && port == "" {
		// Could be domain or IP.
		if ip := net.ParseIP(host); ip != nil {
			return ParsedInput{
				Raw:  raw,
				Type: InputTypeIP,
				Host: host,
			}
		}
		return ParsedInput{
			Raw:    raw,
			Type:   InputTypeDomain,
			Host:   host,
			Scheme: scheme,
			URL:    candidate,
		}
	}

	if port != "" {
		return ParsedInput{
			Raw:    raw,
			Type:   InputTypeURLWithPort,
			Host:   host,
			Port:   port,
			Scheme: scheme,
			URL:    candidate,
		}
	}
	return ParsedInput{
		Raw:    raw,
		Type:   InputTypeURLNoPort,
		Host:   host,
		Scheme: scheme,
		URL:    candidate,
	}
}

// Strategy determines which scan phases to generate.
type Strategy struct {
	HostDiscovery bool
	PortScan      bool
	PathScan      bool
	VulnScan      bool
	Exploit       bool
}

// FullStrategy enables all scan phases.
func FullStrategy() Strategy {
	return Strategy{
		HostDiscovery: true,
		PortScan:      true,
		PathScan:      true,
		VulnScan:      true,
		Exploit:       true,
	}
}

// DiscoveryOnlyStrategy enables only discovery phases.
func DiscoveryOnlyStrategy() Strategy {
	return Strategy{
		HostDiscovery: true,
		PortScan:      true,
		PathScan:      true,
		VulnScan:      false,
		Exploit:       false,
	}
}

// ReconStrategy enables discovery and vuln scanning.
func ReconStrategy() Strategy {
	return Strategy{
		HostDiscovery: true,
		PortScan:      true,
		PathScan:      true,
		VulnScan:      true,
		Exploit:       false,
	}
}

// GenerateInitialTasks creates the initial set of scan tasks for a target
// based on the input type and strategy.
func GenerateInitialTasks(input ParsedInput, strat Strategy, runID string) []*domain.ScanTask {
	var tasks []*domain.ScanTask

	// Seed task is always generated.
	seed := &domain.ScanTask{
		ID:       genTaskID(domain.ScanTypeSeed),
		Type:     domain.ScanTypeSeed,
		Target:   input.Raw,
		Status:   domain.ScanTaskPending,
		ParentID: runID,
	}
	tasks = append(tasks, seed)

	switch input.Type {
	case InputTypeCIDR:
		// CIDR -> host discovery -> per host port scan -> ...
		if strat.HostDiscovery {
			tasks = append(tasks, &domain.ScanTask{
				ID:       genTaskID(domain.ScanTypeHostDiscovery),
				Type:     domain.ScanTypeHostDiscovery,
				Target:   input.Raw,
				Status:   domain.ScanTaskPending,
				ParentID: runID,
			})
		}
		// Port scan tasks will be generated dynamically after host discovery.

	case InputTypeIP:
		// Direct IP -> port scan
		if strat.PortScan {
			tasks = append(tasks, &domain.ScanTask{
				ID:       genTaskID(domain.ScanTypePortScan),
				Type:     domain.ScanTypePortScan,
				Target:   input.Host,
				Status:   domain.ScanTaskPending,
				ParentID: runID,
			})
		}

	case InputTypeDomain, InputTypeURLNoPort:
		// Domain or URL without explicit port -> host discovery + port scan
		if strat.HostDiscovery {
			tasks = append(tasks, &domain.ScanTask{
				ID:       genTaskID(domain.ScanTypeHostDiscovery),
				Type:     domain.ScanTypeHostDiscovery,
				Target:   input.Host,
				Status:   domain.ScanTaskPending,
				ParentID: runID,
			})
		}
		if strat.PortScan {
			tasks = append(tasks, &domain.ScanTask{
				ID:       genTaskID(domain.ScanTypePortScan),
				Type:     domain.ScanTypePortScan,
				Target:   input.Host,
				Status:   domain.ScanTaskPending,
				ParentID: runID,
			})
		}

	case InputTypeURLWithPort:
		// URL with explicit port -> skip port discovery, go to path scan
		if strat.PathScan {
			tasks = append(tasks, &domain.ScanTask{
				ID:       genTaskID(domain.ScanTypePathScan),
				Type:     domain.ScanTypePathScan,
				Target:   input.URL,
				Status:   domain.ScanTaskPending,
				ParentID: runID,
			})
		}
		if strat.VulnScan {
			tasks = append(tasks, &domain.ScanTask{
				ID:       genTaskID(domain.ScanTypeVulnScan),
				Type:     domain.ScanTypeVulnScan,
				Target:   input.URL,
				Status:   domain.ScanTaskPending,
				ParentID: runID,
			})
		}
		if strat.Exploit {
			tasks = append(tasks, &domain.ScanTask{
				ID:       genTaskID(domain.ScanTypeExploit),
				Type:     domain.ScanTypeExploit,
				Target:   input.URL,
				Status:   domain.ScanTaskPending,
				ParentID: runID,
			})
		}
	}

	return tasks
}

// GenerateFollowUpTasks creates next-phase tasks after a scan completes.
// For example, after port scan finds port 3000, generate path scan + vuln scan for it.
func GenerateFollowUpTasks(completedTask *domain.ScanTask, discoveredTargets []string, strat Strategy, runID string) []*domain.ScanTask {
	var tasks []*domain.ScanTask

	switch completedTask.Type {
	case domain.ScanTypeHostDiscovery:
		// Each discovered host gets a port scan.
		if strat.PortScan {
			for _, host := range discoveredTargets {
				tasks = append(tasks, &domain.ScanTask{
					ID:       genTaskID(domain.ScanTypePortScan),
					Type:     domain.ScanTypePortScan,
					Target:   host,
					Status:   domain.ScanTaskPending,
					ParentID: runID,
				})
			}
		}

	case domain.ScanTypePortScan:
		// Each discovered port gets path scan + vuln scan.
		for _, target := range discoveredTargets {
			if strat.PathScan {
				tasks = append(tasks, &domain.ScanTask{
					ID:       genTaskID(domain.ScanTypePathScan),
					Type:     domain.ScanTypePathScan,
					Target:   target,
					Status:   domain.ScanTaskPending,
					ParentID: runID,
				})
			}
			if strat.VulnScan {
				tasks = append(tasks, &domain.ScanTask{
					ID:       genTaskID(domain.ScanTypeVulnScan),
					Type:     domain.ScanTypeVulnScan,
					Target:   target,
					Status:   domain.ScanTaskPending,
					ParentID: runID,
				})
			}
		}

	case domain.ScanTypeVulnScan:
		// Discovered vulns with exp_available get exploit tasks.
		if strat.Exploit {
			for _, target := range discoveredTargets {
				tasks = append(tasks, &domain.ScanTask{
					ID:       genTaskID(domain.ScanTypeExploit),
					Type:     domain.ScanTypeExploit,
					Target:   target,
					Status:   domain.ScanTaskPending,
					ParentID: runID,
				})
			}
		}
	}

	return tasks
}
