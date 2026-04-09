package runtime

import "testing"

func TestRenderStringSupportsJSONHelper(t *testing.T) {
	rendered, err := RenderString(`{{ json .assets.security.cve_findings }}`, map[string]any{
		"assets": map[string]any{
			"security": map[string]any{
				"cve_findings": []any{
					map[string]any{
						"template_id": "cve-2025-55182-behavior-check",
						"target":      "https://target.example",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderString returned error: %v", err)
	}
	expected := `[{"target":"https://target.example","template_id":"cve-2025-55182-behavior-check"}]`
	if rendered != expected {
		t.Fatalf("expected %q, got %q", expected, rendered)
	}
}
