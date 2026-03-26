package parser

import (
	"testing"

	"penframe/internal/domain"
)

func TestParseSavesStructuredAssets(t *testing.T) {
	engine := NewEngine()
	assets := map[string]any{}
	rules := domain.ParserRuleSet{
		Tool: "discovery",
		Rules: []domain.ParserRule{
			{
				Name:   "http",
				Regex:  `HTTP\s+(?P<host>\S+):(?P<port>\d+)`,
				SaveTo: "assets.services.http",
			},
		},
	}

	records, err := engine.Parse(rules, "HTTP 127.0.0.1:8080", assets)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 parsed record, got %d", len(records))
	}
	services, ok := assets["services"].(map[string]any)
	if !ok {
		t.Fatalf("expected services map, got %T", assets["services"])
	}
	httpAssets, ok := services["http"].([]any)
	if !ok {
		t.Fatalf("expected http assets list, got %T", services["http"])
	}
	if len(httpAssets) != 1 {
		t.Fatalf("expected 1 http asset, got %d", len(httpAssets))
	}
}
