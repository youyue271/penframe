package parser

import (
	"fmt"
	"regexp"
	"strings"

	"penframe/internal/domain"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Parse(ruleSet domain.ParserRuleSet, stdout string, assets map[string]any) ([]domain.ParsedRecord, error) {
	var records []domain.ParsedRecord
	for _, rule := range ruleSet.Rules {
		re, err := regexp.Compile(rule.Regex)
		if err != nil {
			return nil, fmt.Errorf("compile parser rule %q: %w", rule.Name, err)
		}
		groupNames := re.SubexpNames()
		matches := re.FindAllStringSubmatch(stdout, -1)
		for _, match := range matches {
			record := make(map[string]string)
			for idx, value := range match {
				if idx == 0 {
					continue
				}
				name := groupNames[idx]
				if name == "" {
					continue
				}
				record[name] = value
			}
			if err := saveRecord(assets, rule.SaveTo, record); err != nil {
				return nil, fmt.Errorf("save parser rule %q output: %w", rule.Name, err)
			}
			records = append(records, domain.ParsedRecord{
				Rule:   rule.Name,
				Path:   rule.SaveTo,
				Fields: record,
			})
		}
	}
	return records, nil
}

func saveRecord(root map[string]any, path string, record map[string]string) error {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "assets.")
	if path == "" {
		return fmt.Errorf("empty save_to path")
	}

	segments := strings.Split(path, ".")
	cursor := root
	for idx, segment := range segments {
		if segment == "" {
			return fmt.Errorf("invalid segment in save_to path %q", path)
		}
		if idx == len(segments)-1 {
			existing, ok := cursor[segment]
			if !ok {
				cursor[segment] = []any{record}
				return nil
			}
			list, ok := existing.([]any)
			if !ok {
				return fmt.Errorf("path %q already exists with incompatible type %T", path, existing)
			}
			cursor[segment] = append(list, record)
			return nil
		}

		next, ok := cursor[segment]
		if !ok {
			child := map[string]any{}
			cursor[segment] = child
			cursor = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("path %q already exists with incompatible type %T", strings.Join(segments[:idx+1], "."), next)
		}
		cursor = child
	}
	return nil
}
