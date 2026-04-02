package targeting

import "testing"

func TestParseURLDetails(t *testing.T) {
	details := Parse("https://demo.example:3000/apps")

	if details.URL != "https://demo.example:3000/apps" {
		t.Fatalf("expected URL to be preserved, got %q", details.URL)
	}
	if details.Host != "demo.example" {
		t.Fatalf("expected host demo.example, got %q", details.Host)
	}
	if details.HostPort != "demo.example:3000" {
		t.Fatalf("expected hostport demo.example:3000, got %q", details.HostPort)
	}
	if details.Port != "3000" {
		t.Fatalf("expected port 3000, got %q", details.Port)
	}
	if details.Origin != "https://demo.example:3000" {
		t.Fatalf("expected origin https://demo.example:3000, got %q", details.Origin)
	}
	if details.Path != "/apps" {
		t.Fatalf("expected path /apps, got %q", details.Path)
	}
}

func TestApplyOverrideSetsDerivedVars(t *testing.T) {
	vars := map[string]any{}
	ApplyOverride(vars, "https://demo.example:3000/path")

	if got := vars["target"]; got != "https://demo.example:3000/path" {
		t.Fatalf("expected target to be stored, got %#v", got)
	}
	if got := vars["target_host"]; got != "demo.example" {
		t.Fatalf("expected target_host demo.example, got %#v", got)
	}
	if got := vars["target_hostport"]; got != "demo.example:3000" {
		t.Fatalf("expected target_hostport demo.example:3000, got %#v", got)
	}
	if got := vars["target_port"]; got != "3000" {
		t.Fatalf("expected target_port 3000, got %#v", got)
	}
	if got := vars["target_origin"]; got != "https://demo.example:3000" {
		t.Fatalf("expected target_origin https://demo.example:3000, got %#v", got)
	}
}

func TestEnsurePreservesExistingValuesAndBackfillsMissingFields(t *testing.T) {
	vars := map[string]any{
		"target_url":  "https://demo.example:3000/path",
		"target_host": "demo.example",
	}

	Ensure(vars)

	if got := vars["target_host"]; got != "demo.example" {
		t.Fatalf("expected target_host to be preserved, got %#v", got)
	}
	if got := vars["target_hostport"]; got != "demo.example:3000" {
		t.Fatalf("expected target_hostport demo.example:3000, got %#v", got)
	}
	if got := vars["target_port"]; got != "3000" {
		t.Fatalf("expected target_port 3000, got %#v", got)
	}
	if got := vars["target_path"]; got != "/path" {
		t.Fatalf("expected target_path /path, got %#v", got)
	}
}
