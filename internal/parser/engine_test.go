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

func TestParseTrimsFieldsAndDeduplicatesRecords(t *testing.T) {
	engine := NewEngine()
	assets := map[string]any{}
	rules := domain.ParserRuleSet{
		Tool: "headers",
		Rules: []domain.ParserRule{
			{
				Name:   "powered-by",
				Regex:  `(?m)^x-powered-by:\s*(?P<value>.+)$`,
				SaveTo: "assets.web.powered_by",
			},
		},
	}

	stdout := "x-powered-by: Next.js\r\nx-powered-by: Next.js\r\n"
	records, err := engine.Parse(rules, stdout, assets)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 unique parsed record, got %d", len(records))
	}

	web, ok := assets["web"].(map[string]any)
	if !ok {
		t.Fatalf("expected web map, got %T", assets["web"])
	}
	poweredBy, ok := web["powered_by"].([]any)
	if !ok {
		t.Fatalf("expected powered_by list, got %T", web["powered_by"])
	}
	if len(poweredBy) != 1 {
		t.Fatalf("expected 1 unique powered_by asset, got %d", len(poweredBy))
	}
	record, ok := poweredBy[0].(map[string]string)
	if !ok {
		t.Fatalf("expected powered_by record map, got %T", poweredBy[0])
	}
	if record["value"] != "Next.js" {
		t.Fatalf("expected trimmed value Next.js, got %#v", record["value"])
	}
}

func TestParseExtractsDifyFingerprintsFromMarkup(t *testing.T) {
	engine := NewEngine()
	assets := map[string]any{}
	rules := domain.ParserRuleSet{
		Tool: "curl_fetch",
		Rules: []domain.ParserRule{
			{
				Name:   "dify_console_api_prefix",
				Regex:  `(?i)data-api-prefix="(?P<api_prefix>[^"]+)"`,
				SaveTo: "assets.web.fingerprints.dify_console_api",
			},
			{
				Name:   "dify_public_api_prefix",
				Regex:  `(?i)data-pub(?:l)?ic-api-prefix="(?P<api_prefix>[^"]+)"`,
				SaveTo: "assets.web.fingerprints.dify_public_api",
			},
			{
				Name:   "dify_marketplace_url_prefix",
				Regex:  `(?i)data-marketplace-url-prefix="(?P<url_prefix>[^"]+)"`,
				SaveTo: "assets.web.fingerprints.dify_marketplace_url",
			},
			{
				Name:   "dify_public_edition",
				Regex:  `(?i)data-public-edition="(?P<edition>[^"]+)"`,
				SaveTo: "assets.web.fingerprints.dify_edition",
			},
			{
				Name:   "robots_noindex",
				Regex:  `(?i)<meta\s+name="robots"\s+content="(?P<directive>[^"]+)"`,
				SaveTo: "assets.web.metadata.robots",
			},
			{
				Name:   "dify_product_name",
				Regex:  `(?i)<title>\s*(?P<product>Dify)\s*</title>`,
				SaveTo: "assets.web.fingerprints.products",
			},
			{
				Name:   "react_rsc_flight_stream",
				Regex:  `(?i)(?P<feature>__next_f\.push)`,
				SaveTo: "assets.web.fingerprints.react_rsc",
			},
		},
	}

	stdout := `
HTTP/2 200
<meta name="robots" content="noindex"/>
<title>Dify</title>
<body data-api-prefix="http://127.0.0.1:5001/console/api"
      data-pubic-api-prefix="http://127.0.0.1:5001/api"
      data-marketplace-url-prefix="https://marketplace.dify.ai"
      data-public-edition="SELF_HOSTED">
<script>self.__next_f.push([1,"payload"])</script>
`

	records, err := engine.Parse(rules, stdout, assets)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(records) != 7 {
		t.Fatalf("expected 7 parsed records, got %d", len(records))
	}

	web, ok := assets["web"].(map[string]any)
	if !ok {
		t.Fatalf("expected web map, got %T", assets["web"])
	}
	fingerprints, ok := web["fingerprints"].(map[string]any)
	if !ok {
		t.Fatalf("expected fingerprints map, got %T", web["fingerprints"])
	}
	edition, ok := fingerprints["dify_edition"].([]any)
	if !ok || len(edition) != 1 {
		t.Fatalf("expected dify edition asset, got %#v", fingerprints["dify_edition"])
	}
	products, ok := fingerprints["products"].([]any)
	if !ok || len(products) != 1 {
		t.Fatalf("expected product asset, got %#v", fingerprints["products"])
	}
	rsc, ok := fingerprints["react_rsc"].([]any)
	if !ok || len(rsc) != 1 {
		t.Fatalf("expected react_rsc asset, got %#v", fingerprints["react_rsc"])
	}
}

func TestParseExtractsNucleiJSONLFindings(t *testing.T) {
	engine := NewEngine()
	assets := map[string]any{}
	rules := domain.ParserRuleSet{
		Tool: "nuclei_scan",
		Rules: []domain.ParserRule{
			{
				Name:   "nuclei_finding_jsonl",
				Regex:  `(?m)^\{[^\n]*"template-id":"(?P<template_id>[^"]+)"[^\n]*"name":"(?P<name>[^"]+)"[^\n]*"severity":"(?P<severity>info|low|medium|high|critical|unknown)"[^\n]*"matched-at":"(?P<target>[^"]+)"[^\n]*\}$`,
				SaveTo: "assets.discovery.nuclei_findings",
			},
			{
				Name:   "nuclei_cve_jsonl",
				Regex:  `(?mi)^\{[^\n]*"template-id":"(?P<template_id>cve-[^"]+)"[^\n]*"name":"(?P<name>[^"]+)"[^\n]*"severity":"(?P<severity>info|low|medium|high|critical|unknown)"[^\n]*"matched-at":"(?P<target>[^"]+)"[^\n]*\}$`,
				SaveTo: "assets.security.cve_findings",
			},
		},
	}

	stdout := `
{"template-id":"dify-nextjs-http-fingerprint","info":{"name":"Dify / Next.js HTTP Fingerprint","severity":"info"},"matched-at":"https://target.example/apps"}
{"template-id":"cve-2025-55182-possible-surface","info":{"name":"Possible Exposure Prerequisites For CVE-2025-55182","severity":"info"},"matched-at":"https://target.example/signin"}
`

	records, err := engine.Parse(rules, stdout, assets)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 parsed records, got %d", len(records))
	}

	discovery, ok := assets["discovery"].(map[string]any)
	if !ok {
		t.Fatalf("expected discovery map, got %T", assets["discovery"])
	}
	findings, ok := discovery["nuclei_findings"].([]any)
	if !ok || len(findings) != 2 {
		t.Fatalf("expected 2 nuclei findings, got %#v", discovery["nuclei_findings"])
	}

	security, ok := assets["security"].(map[string]any)
	if !ok {
		t.Fatalf("expected security map, got %T", assets["security"])
	}
	cves, ok := security["cve_findings"].([]any)
	if !ok || len(cves) != 1 {
		t.Fatalf("expected 1 cve finding, got %#v", security["cve_findings"])
	}
}
