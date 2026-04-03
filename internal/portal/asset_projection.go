package portal

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"penframe/internal/asset"
	"penframe/internal/domain"
	"penframe/internal/storage"
)

type projectedAssetGraph struct {
	graph  *asset.Graph
	vars   map[string]any
	assets map[string]any
}

type endpointInfo struct {
	host     string
	hostname string
	port     int
	path     string
}

func buildAssetGraphFromRun(run storage.StoredRun) *asset.Graph {
	target := firstStringFromAnyMap(run.Summary.Vars, "target", "target_url", "target_host", "target_origin")
	if target == "" {
		target = run.ID
	}

	projection := projectedAssetGraph{
		graph:  asset.NewGraph(target),
		vars:   run.Summary.Vars,
		assets: run.Summary.Assets,
	}
	projection.seedFromTarget()
	projection.ingestServiceRecords("services.http", "http")
	projection.ingestServiceRecords("services.database", "database")
	projection.ingestHostRecords("discovery.hosts")
	projection.ingestEntryRecords("discovery.entries")
	projection.ingestEntryRecords("web.pages")
	projection.ingestEntryRecords("web.redirects")
	projection.ingestEntryRecords("paths.discovered")
	projection.ingestOpenPortRecords("network.open_ports")
	projection.ingestVulnRecords("discovery.nuclei_findings", "nuclei")
	projection.ingestVulnRecords("security.cve_findings", "nuclei")
	projection.ingestVulnRecords("security.vulnerabilities", "scanner")
	return projection.graph
}

func (p projectedAssetGraph) seedFromTarget() {
	targetURL := firstStringFromAnyMap(p.vars, "target_url", "target")
	if info := parseEndpoint(targetURL); info.host != "" {
		host := p.graph.AddHost(info.host, info.hostname, "target")
		if info.port > 0 {
			port := p.graph.AddPort(host.ID, info.port, "tcp", "", "", "target")
			if info.path != "" {
				p.graph.AddPath(port.ID, info.path, 0, "", "", "target")
			}
		}
		return
	}

	hostValue := firstStringFromAnyMap(p.vars, "target_host", "host")
	if hostValue == "" {
		return
	}
	hostInfo := parseEndpoint(hostValue)
	if hostInfo.host == "" {
		return
	}
	host := p.graph.AddHost(hostInfo.host, hostInfo.hostname, "target")
	if portValue := parseInt(firstStringFromAnyMap(p.vars, "target_port")); portValue > 0 {
		p.graph.AddPort(host.ID, portValue, "tcp", "", "", "target")
	}
}

func (p projectedAssetGraph) ingestServiceRecords(path, service string) {
	for _, record := range recordMapsAtPath(p.assets, path) {
		info := parseEndpoint(firstStringFromRecord(record, "host", "hostport", "origin", "target"))
		if info.host == "" {
			continue
		}
		host := p.graph.AddHost(info.host, info.hostname, path)
		portValue := parseInt(firstStringFromRecord(record, "port"))
		if portValue <= 0 {
			portValue = info.port
		}
		if portValue <= 0 {
			continue
		}
		p.graph.AddPort(host.ID, portValue, "tcp", service, firstStringFromRecord(record, "version"), path)
	}
}

func (p projectedAssetGraph) ingestHostRecords(path string) {
	for _, record := range recordMapsAtPath(p.assets, path) {
		info := parseEndpoint(firstStringFromRecord(record, "hostport", "origin", "host"))
		if info.host == "" {
			continue
		}
		host := p.graph.AddHost(info.host, info.hostname, path)
		if info.port > 0 {
			p.graph.AddPort(host.ID, info.port, "tcp", "", "", path)
		}
	}
}

func (p projectedAssetGraph) ingestEntryRecords(path string) {
	for _, record := range recordMapsAtPath(p.assets, path) {
		info := parseEndpoint(firstStringFromRecord(record, "url", "redirect_url", "location", "target"))
		if info.host == "" {
			continue
		}
		host := p.graph.AddHost(info.host, info.hostname, path)
		if info.port <= 0 {
			info.port = p.defaultPort(host)
		}
		if info.port <= 0 {
			continue
		}
		port := p.graph.AddPort(host.ID, info.port, "tcp", "", "", path)
		pathValue := info.path
		if pathValue == "" {
			pathValue = "/"
		}
		p.graph.AddPath(
			port.ID,
			pathValue,
			parseInt(firstStringFromRecord(record, "status_code")),
			firstStringFromRecord(record, "title"),
			firstStringFromRecord(record, "tech"),
			path,
		)
	}
}

func (p projectedAssetGraph) ingestOpenPortRecords(path string) {
	for _, record := range recordMapsAtPath(p.assets, path) {
		portValue, protocol := parsePortProtocol(firstStringFromRecord(record, "port"))
		if portValue <= 0 {
			continue
		}

		info := parseEndpoint(firstStringFromRecord(record, "host", "target"))
		var host *domain.AssetHost
		if info.host != "" {
			host = p.graph.AddHost(info.host, info.hostname, path)
		} else {
			host = p.defaultHost()
		}
		if host == nil {
			continue
		}

		p.graph.AddPort(
			host.ID,
			portValue,
			protocol,
			firstStringFromRecord(record, "service"),
			firstStringFromRecord(record, "version"),
			path,
		)
	}
}

func (p projectedAssetGraph) ingestVulnRecords(path, source string) {
	for _, record := range recordMapsAtPath(p.assets, path) {
		info := parseEndpoint(firstStringFromRecord(record, "target", "url", "entry", "hostport", "host"))
		var host *domain.AssetHost
		if info.host != "" {
			host = p.graph.AddHost(info.host, info.hostname, path)
		} else {
			host = p.defaultHost()
		}
		if host == nil {
			continue
		}

		portValue := info.port
		if portValue <= 0 {
			portValue = p.defaultPort(host)
		}
		if portValue <= 0 {
			continue
		}

		port := p.graph.AddPort(host.ID, portValue, "tcp", "", "", path)
		pathID := ""
		if info.path != "" {
			pathID = p.graph.AddPath(port.ID, info.path, 0, "", "", path).ID
		}

		templateID := firstStringFromRecord(record, "cve", "template_id")
		name := firstStringFromRecord(record, "name", "detail", "status", "template_id")
		if name == "" {
			name = templateID
		}

		p.graph.AddVuln(
			port.ID,
			pathID,
			templateID,
			name,
			firstStringFromRecord(record, "severity"),
			source,
			firstStringFromRecord(record, "detail", "target"),
			strings.HasPrefix(strings.ToLower(templateID), "cve-"),
		)
	}
}

func (p projectedAssetGraph) defaultHost() *domain.AssetHost {
	if info := parseEndpoint(firstStringFromAnyMap(p.vars, "target_host", "host")); info.host != "" {
		return p.graph.AddHost(info.host, info.hostname, "target")
	}
	if len(p.graph.Hosts) > 0 {
		return p.graph.Hosts[0]
	}
	return nil
}

func (p projectedAssetGraph) defaultPort(host *domain.AssetHost) int {
	if host == nil {
		return 0
	}
	if portValue := parseInt(firstStringFromAnyMap(p.vars, "target_port")); portValue > 0 {
		return portValue
	}
	if len(host.Ports) > 0 {
		return host.Ports[0].Port
	}
	return 0
}

func recordMapsAtPath(root map[string]any, path string) []map[string]string {
	if len(root) == 0 || strings.TrimSpace(path) == "" {
		return nil
	}

	cursor := any(root)
	for _, segment := range strings.Split(path, ".") {
		next, ok := cursor.(map[string]any)
		if !ok {
			return nil
		}
		value, exists := next[segment]
		if !exists {
			return nil
		}
		cursor = value
	}

	switch typed := cursor.(type) {
	case []any:
		records := make([]map[string]string, 0, len(typed))
		for _, item := range typed {
			if record := toStringMap(item); len(record) > 0 {
				records = append(records, record)
			}
		}
		return records
	case []map[string]any:
		records := make([]map[string]string, 0, len(typed))
		for _, item := range typed {
			if record := toStringMap(item); len(record) > 0 {
				records = append(records, record)
			}
		}
		return records
	case []map[string]string:
		return typed
	default:
		return nil
	}
}

func toStringMap(value any) map[string]string {
	switch typed := value.(type) {
	case map[string]string:
		return typed
	case map[string]any:
		result := make(map[string]string, len(typed))
		for key, item := range typed {
			result[key] = strings.TrimSpace(fmt.Sprint(item))
		}
		return result
	default:
		return nil
	}
}

func parseEndpoint(raw string) endpointInfo {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return endpointInfo{}
	}

	if strings.Contains(candidate, "://") {
		parsed, err := url.Parse(candidate)
		if err == nil && parsed.Hostname() != "" {
			portValue := parseInt(parsed.Port())
			if portValue <= 0 {
				switch strings.ToLower(parsed.Scheme) {
				case "https":
					portValue = 443
				case "http":
					portValue = 80
				}
			}
			pathValue := parsed.EscapedPath()
			if pathValue == "" {
				pathValue = "/"
			}
			return endpointInfo{
				host: parsed.Hostname(),
				port: portValue,
				path: pathValue,
			}
		}
	}

	if hostValue, hostname, portValue, ok := splitHostPort(candidate); ok {
		return endpointInfo{
			host:     hostValue,
			hostname: hostname,
			port:     portValue,
		}
	}

	hostValue, hostname := normalizeHost(candidate)
	return endpointInfo{
		host:     hostValue,
		hostname: hostname,
	}
}

func splitHostPort(raw string) (string, string, int, bool) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" || strings.Contains(candidate, "/") {
		return "", "", 0, false
	}

	if hostValue, portValue, err := net.SplitHostPort(candidate); err == nil {
		parsedPort := parseInt(portValue)
		if parsedPort > 0 {
			host, hostname := normalizeHost(hostValue)
			return host, hostname, parsedPort, true
		}
	}

	idx := strings.LastIndex(candidate, ":")
	if idx <= 0 || idx >= len(candidate)-1 {
		return "", "", 0, false
	}
	parsedPort := parseInt(candidate[idx+1:])
	if parsedPort <= 0 {
		return "", "", 0, false
	}
	host, hostname := normalizeHost(candidate[:idx])
	if host == "" {
		return "", "", 0, false
	}
	return host, hostname, parsedPort, true
}

func normalizeHost(raw string) (string, string) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return "", ""
	}

	if strings.HasSuffix(candidate, ")") {
		if idx := strings.LastIndex(candidate, " ("); idx > 0 {
			label := strings.TrimSpace(candidate[:idx])
			hostValue := strings.TrimSpace(strings.TrimSuffix(candidate[idx+2:], ")"))
			if hostValue != "" {
				return hostValue, label
			}
		}
	}

	return candidate, ""
}

func parsePortProtocol(raw string) (int, string) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return 0, "tcp"
	}
	parts := strings.SplitN(candidate, "/", 2)
	portValue := parseInt(parts[0])
	protocol := "tcp"
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		protocol = strings.TrimSpace(parts[1])
	}
	return portValue, protocol
}

func firstStringFromAnyMap(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if values == nil {
			return ""
		}
		value, ok := values[key]
		if !ok {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func firstStringFromRecord(record map[string]string, keys ...string) string {
	for _, key := range keys {
		text := strings.TrimSpace(record[key])
		if text != "" {
			return text
		}
	}
	return ""
}

func parseInt(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return value
}
