package workflow

import "testing"

func TestMiniExprEvaluator(t *testing.T) {
	evaluator := NewMiniExprEvaluator()
	env := map[string]any{
		"assets": map[string]any{
			"services": map[string]any{
				"http": []any{"a", "b"},
			},
		},
		"results": map[string]any{
			"discover": map[string]any{
				"record_count": 2,
			},
		},
	}

	ok, err := evaluator.Evaluate("len(assets.services.http) > 0 && results.discover.record_count == 2", env)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected expression to be true")
	}
}
