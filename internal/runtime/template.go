package runtime

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	texttemplate "text/template"
)

func RenderValue(value any, ctx map[string]any) (any, error) {
	switch typed := value.(type) {
	case string:
		if !strings.Contains(typed, "{{") {
			return typed, nil
		}
		return RenderString(typed, ctx)
	case map[string]any:
		rendered := make(map[string]any, len(typed))
		for key, nested := range typed {
			value, err := RenderValue(nested, ctx)
			if err != nil {
				return nil, err
			}
			rendered[key] = value
		}
		return rendered, nil
	case []any:
		rendered := make([]any, len(typed))
		for idx, nested := range typed {
			value, err := RenderValue(nested, ctx)
			if err != nil {
				return nil, err
			}
			rendered[idx] = value
		}
		return rendered, nil
	default:
		valueOf := reflect.ValueOf(value)
		if valueOf.IsValid() && valueOf.Kind() == reflect.Map {
			rendered := make(map[string]any, valueOf.Len())
			iter := valueOf.MapRange()
			for iter.Next() {
				key := fmt.Sprint(iter.Key().Interface())
				value, err := RenderValue(iter.Value().Interface(), ctx)
				if err != nil {
					return nil, err
				}
				rendered[key] = value
			}
			return rendered, nil
		}
		return value, nil
	}
}

func RenderString(raw string, ctx map[string]any) (string, error) {
	tmpl, err := texttemplate.New("value").Option("missingkey=error").Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", raw, err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, ctx); err != nil {
		return "", fmt.Errorf("render template %q: %w", raw, err)
	}
	return strings.TrimSpace(out.String()), nil
}
