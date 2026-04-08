package asset

import (
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"

	"penframe/internal/domain"
)

var nextID atomic.Int64

func genID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, nextID.Add(1))
}

// CytoscapeElement is a node or edge for Cytoscape.js rendering.
type CytoscapeElement struct {
	Group   string         `json:"group"`            // "nodes" or "edges"
	Data    map[string]any `json:"data"`
	Classes string         `json:"classes,omitempty"`
}

// Graph holds the hierarchical asset tree:
//   Target -> Host -> Port -> Path
//                          -> Vuln
type Graph struct {
	TargetRaw string
	Hosts     []*domain.AssetHost
	hostIndex map[string]*domain.AssetHost // ip -> host
	portIndex map[string]*domain.AssetPort // portID -> port
	pathIndex map[string]*domain.AssetPath // pathID -> path
	vulnIndex map[string]*domain.AssetVuln // vulnID -> vuln
}

// NewGraph creates an empty asset graph for a given target.
func NewGraph(target string) *Graph {
	return &Graph{
		TargetRaw: target,
		hostIndex: make(map[string]*domain.AssetHost),
		portIndex: make(map[string]*domain.AssetPort),
		pathIndex: make(map[string]*domain.AssetPath),
		vulnIndex: make(map[string]*domain.AssetVuln),
	}
}

// AddHost adds or returns an existing host by IP.
func (g *Graph) AddHost(ip, hostname, source string) *domain.AssetHost {
	if existing, ok := g.hostIndex[ip]; ok {
		if hostname != "" && existing.Hostname == "" {
			existing.Hostname = hostname
		}
		return existing
	}
	host := &domain.AssetHost{
		ID:       genID("host"),
		IP:       ip,
		Hostname: hostname,
		Status:   "alive",
		Source:   source,
	}
	g.Hosts = append(g.Hosts, host)
	g.hostIndex[ip] = host
	return host
}

// FindHost returns a host by IP.
func (g *Graph) FindHost(ip string) (*domain.AssetHost, bool) {
	h, ok := g.hostIndex[ip]
	return h, ok
}

// AddPort adds a port to a host (deduplicated by host + port number + protocol).
func (g *Graph) AddPort(hostID string, port int, protocol, service, banner, source string) *domain.AssetPort {
	host := g.findHostByID(hostID)
	if host == nil {
		return nil
	}
	for i := range host.Ports {
		p := &host.Ports[i]
		if p.Port == port && p.Protocol == protocol {
			if service != "" && p.Service == "" {
				p.Service = service
			}
			if banner != "" && p.Banner == "" {
				p.Banner = banner
			}
			return p
		}
	}
	portEntry := domain.AssetPort{
		ID:       genID("port"),
		HostID:   hostID,
		Port:     port,
		Protocol: protocol,
		Service:  service,
		Banner:   banner,
		Source:   source,
	}
	host.Ports = append(host.Ports, portEntry)
	g.portIndex[portEntry.ID] = &host.Ports[len(host.Ports)-1]
	return &host.Ports[len(host.Ports)-1]
}

// AddPath adds a discovered path under a port.
func (g *Graph) AddPath(portID string, path string, statusCode int, title, tech, source string) *domain.AssetPath {
	port := g.portIndex[portID]
	if port == nil {
		return nil
	}
	for i := range port.Paths {
		p := &port.Paths[i]
		if p.Path == path {
			if statusCode != 0 {
				p.StatusCode = statusCode
			}
			if title != "" {
				p.Title = title
			}
			return p
		}
	}
	pathEntry := domain.AssetPath{
		ID:         genID("path"),
		PortID:     portID,
		Path:       path,
		StatusCode: statusCode,
		Title:      title,
		Tech:       tech,
		Source:     source,
	}
	port.Paths = append(port.Paths, pathEntry)
	g.pathIndex[pathEntry.ID] = &port.Paths[len(port.Paths)-1]
	return &port.Paths[len(port.Paths)-1]
}

// AddVuln adds a vulnerability under a port or path.
func (g *Graph) AddVuln(portID, pathID string, cve, name, severity, source, detail string, expAvail bool) *domain.AssetVuln {
	vuln := domain.AssetVuln{
		ID:       genID("vuln"),
		PortID:   portID,
		PathID:   pathID,
		CVE:      cve,
		Name:     name,
		Severity: severity,
		Source:   source,
		ExpAvail: expAvail,
		Detail:   detail,
	}

	// Attach to port's Vulns if port exists.
	if p, ok := g.portIndex[portID]; ok {
		p.Vulns = append(p.Vulns, vuln)
		g.vulnIndex[vuln.ID] = &p.Vulns[len(p.Vulns)-1]
		return &p.Vulns[len(p.Vulns)-1]
	}
	g.vulnIndex[vuln.ID] = &vuln
	return &vuln
}

// HostCount returns the number of discovered hosts.
func (g *Graph) HostCount() int { return len(g.Hosts) }

// PortCount returns the total number of discovered ports.
func (g *Graph) PortCount() int {
	count := 0
	for _, h := range g.Hosts {
		count += len(h.Ports)
	}
	return count
}

// PathCount returns the total number of discovered paths.
func (g *Graph) PathCount() int {
	count := 0
	for _, h := range g.Hosts {
		for _, p := range h.Ports {
			count += len(p.Paths)
		}
	}
	return count
}

// VulnCount returns the total number of discovered vulnerabilities.
func (g *Graph) VulnCount() int {
	count := 0
	for _, h := range g.Hosts {
		for _, p := range h.Ports {
			count += len(p.Vulns)
		}
	}
	return count
}

// ExploitableVulnPathCount returns the deduplicated number of exploitable vulnerability paths.
func (g *Graph) ExploitableVulnPathCount() int {
	type vulnSummaryGroup struct {
		expAvailable bool
		hits         map[string]struct{}
	}

	groups := make(map[string]*vulnSummaryGroup)
	targetSite := g.targetSite()

	for _, h := range g.Hosts {
		for _, p := range h.Ports {
			hitSite := targetSite
			if hitSite == "" {
				hitSite = fmt.Sprintf("%s:%d", h.IP, p.Port)
			}

			for i := range p.Vulns {
				v := &p.Vulns[i]
				groupKey := hitSite + "::" + v.Name
				hitKey := hitSite + "::" + g.vulnPath(v.PathID)
				group, ok := groups[groupKey]
				if !ok {
					group = &vulnSummaryGroup{hits: make(map[string]struct{})}
					groups[groupKey] = group
				}
				group.hits[hitKey] = struct{}{}
				group.expAvailable = group.expAvailable || v.ExpAvail
			}
		}
	}

	count := 0
	for _, group := range groups {
		if group.expAvailable {
			count += len(group.hits)
		}
	}
	return count
}

func (g *Graph) targetSite() string {
	raw := strings.TrimSpace(g.TargetRaw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return strings.ToLower(parsed.Scheme + "://" + parsed.Host)
	}
	return strings.TrimRight(strings.ToLower(raw), "/")
}

func (g *Graph) vulnPath(pathID string) string {
	if pathID == "" {
		return "/"
	}
	path, ok := g.pathIndex[pathID]
	if !ok || strings.TrimSpace(path.Path) == "" {
		return "/"
	}
	return path.Path
}

// ToCytoscapeJSON exports the graph in Cytoscape.js compatible format.
func (g *Graph) ToCytoscapeJSON() []CytoscapeElement {
	var elements []CytoscapeElement

	// Target root node.
	targetID := "target-root"
	elements = append(elements, CytoscapeElement{
		Group: "nodes",
		Data: map[string]any{
			"id":    targetID,
			"label": g.TargetRaw,
			"type":  "target",
		},
		Classes: "target",
	})

	for _, host := range g.Hosts {
		label := host.IP
		if host.Hostname != "" {
			label = host.Hostname + " (" + host.IP + ")"
		}
		elements = append(elements, CytoscapeElement{
			Group: "nodes",
			Data: map[string]any{
				"id":     host.ID,
				"label":  label,
				"type":   "host",
				"ip":     host.IP,
				"status": host.Status,
			},
			Classes: "host",
		})
		elements = append(elements, CytoscapeElement{
			Group: "edges",
			Data: map[string]any{
				"id":     "e-" + targetID + "-" + host.ID,
				"source": targetID,
				"target": host.ID,
			},
		})

		for i := range host.Ports {
			port := &host.Ports[i]
			portLabel := fmt.Sprintf("%d/%s", port.Port, port.Protocol)
			if port.Service != "" {
				portLabel += " (" + port.Service + ")"
			}
			elements = append(elements, CytoscapeElement{
				Group: "nodes",
				Data: map[string]any{
					"id":       port.ID,
					"label":    portLabel,
					"type":     "port",
					"port":     port.Port,
					"protocol": port.Protocol,
					"service":  port.Service,
				},
				Classes: "port",
			})
			elements = append(elements, CytoscapeElement{
				Group: "edges",
				Data: map[string]any{
					"id":     "e-" + host.ID + "-" + port.ID,
					"source": host.ID,
					"target": port.ID,
				},
			})

			for j := range port.Paths {
				p := &port.Paths[j]
				pathLabel := p.Path
				if p.StatusCode != 0 {
					pathLabel += fmt.Sprintf(" [%d]", p.StatusCode)
				}
				elements = append(elements, CytoscapeElement{
					Group: "nodes",
					Data: map[string]any{
						"id":          p.ID,
						"label":       pathLabel,
						"type":        "path",
						"path":        p.Path,
						"status_code": p.StatusCode,
						"title":       p.Title,
					},
					Classes: "path",
				})
				elements = append(elements, CytoscapeElement{
					Group: "edges",
					Data: map[string]any{
						"id":     "e-" + port.ID + "-" + p.ID,
						"source": port.ID,
						"target": p.ID,
					},
				})
			}

			for k := range port.Vulns {
				v := &port.Vulns[k]
				vulnLabel := v.Name
				if v.CVE != "" {
					vulnLabel = v.CVE + ": " + v.Name
				}
				classes := "vuln vuln-" + v.Severity
				if v.ExpAvail {
					classes += " exploitable"
				}
				elements = append(elements, CytoscapeElement{
					Group: "nodes",
					Data: map[string]any{
						"id":            v.ID,
						"label":         vulnLabel,
						"type":          "vuln",
						"cve":           v.CVE,
						"severity":      v.Severity,
						"exp_available": v.ExpAvail,
					},
					Classes: classes,
				})
				parent := port.ID
				if v.PathID != "" {
					parent = v.PathID
				}
				elements = append(elements, CytoscapeElement{
					Group: "edges",
					Data: map[string]any{
						"id":     "e-" + parent + "-" + v.ID,
						"source": parent,
						"target": v.ID,
					},
				})
			}
		}
	}

	return elements
}

// Summary returns a quick overview of the graph.
func (g *Graph) Summary() map[string]int {
	return map[string]int{
		"hosts": g.HostCount(),
		"ports": g.PortCount(),
		"paths": g.PathCount(),
		"vulns": g.ExploitableVulnPathCount(),
	}
}

func (g *Graph) findHostByID(id string) *domain.AssetHost {
	for _, h := range g.Hosts {
		if h.ID == id {
			return h
		}
	}
	return nil
}
