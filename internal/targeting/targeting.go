package targeting

import (
	"fmt"
	"net/url"
	"strings"
)

type Details struct {
	Raw      string
	URL      string
	Host     string
	HostPort string
	Port     string
	Scheme   string
	Origin   string
	Path     string
}

func ApplyOverride(vars map[string]any, raw string) {
	if vars == nil {
		return
	}

	details := Parse(raw)
	if details.Raw == "" {
		return
	}

	vars["target"] = details.Raw
	setIfNonEmpty(vars, "target_url", details.URL)
	setIfNonEmpty(vars, "target_host", details.Host)
	setIfNonEmpty(vars, "target_hostport", details.HostPort)
	setIfNonEmpty(vars, "target_port", details.Port)
	setIfNonEmpty(vars, "target_scheme", details.Scheme)
	setIfNonEmpty(vars, "target_origin", details.Origin)
	setIfNonEmpty(vars, "target_path", details.Path)
}

func Ensure(vars map[string]any) {
	if vars == nil {
		return
	}

	candidate := firstNonEmptyVar(vars, "target_url", "target", "url", "target_host", "host")
	if candidate == "" {
		return
	}

	details := Parse(candidate)
	setDefault(vars, "target", details.Raw)
	setDefault(vars, "target_url", details.URL)
	setDefault(vars, "target_host", details.Host)
	setDefault(vars, "target_hostport", details.HostPort)
	setDefault(vars, "target_port", details.Port)
	setDefault(vars, "target_scheme", details.Scheme)
	setDefault(vars, "target_origin", details.Origin)
	setDefault(vars, "target_path", details.Path)
}

func Parse(raw string) Details {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return Details{}
	}

	normalizedURL := NormalizeURL(candidate)
	parsed, err := url.Parse(normalizedURL)
	if err != nil {
		return Details{
			Raw: candidate,
			URL: normalizedURL,
		}
	}

	scheme := parsed.Scheme
	host := parsed.Hostname()
	hostPort := parsed.Host
	port := parsed.Port()
	if port == "" {
		switch strings.ToLower(scheme) {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}

	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}

	origin := ""
	if scheme != "" && hostPort != "" {
		origin = scheme + "://" + hostPort
	}

	return Details{
		Raw:      candidate,
		URL:      normalizedURL,
		Host:     host,
		HostPort: hostPort,
		Port:     port,
		Scheme:   scheme,
		Origin:   origin,
		Path:     path,
	}
}

func NormalizeURL(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return ""
	}
	if strings.Contains(candidate, "://") {
		return candidate
	}
	return "https://" + candidate
}

func firstNonEmptyVar(vars map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := vars[key]
		if !ok {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" {
			return text
		}
	}
	return ""
}

func setDefault(vars map[string]any, key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if existing, ok := vars[key]; ok && strings.TrimSpace(fmt.Sprint(existing)) != "" {
		return
	}
	vars[key] = value
}

func setIfNonEmpty(vars map[string]any, key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	vars[key] = value
}
