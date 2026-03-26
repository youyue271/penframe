package workflow

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Evaluator interface {
	Evaluate(condition string, env map[string]any) (bool, error)
}

type MiniExprEvaluator struct{}

func NewMiniExprEvaluator() MiniExprEvaluator {
	return MiniExprEvaluator{}
}

func (MiniExprEvaluator) Evaluate(condition string, env map[string]any) (bool, error) {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true, nil
	}
	return evalBoolExpr(condition, env)
}

func evalBoolExpr(input string, env map[string]any) (bool, error) {
	input = strings.TrimSpace(trimOuterParens(input))
	if parts := splitTopLevel(input, "||"); len(parts) > 1 {
		for _, part := range parts {
			ok, err := evalBoolExpr(part, env)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	if parts := splitTopLevel(input, "&&"); len(parts) > 1 {
		for _, part := range parts {
			ok, err := evalBoolExpr(part, env)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	}
	return evalClause(input, env)
}

func evalClause(input string, env map[string]any) (bool, error) {
	input = strings.TrimSpace(trimOuterParens(input))
	if input == "true" {
		return true, nil
	}
	if input == "false" {
		return false, nil
	}

	for _, op := range []string{"==", "!=", ">=", "<=", ">", "<"} {
		if left, right, ok := splitComparison(input, op); ok {
			return compare(left, op, right, env)
		}
	}

	value, err := resolveOperand(input, env)
	if err != nil {
		return false, err
	}
	return truthy(value), nil
}

func compare(leftRaw, op, rightRaw string, env map[string]any) (bool, error) {
	left, err := resolveOperand(leftRaw, env)
	if err != nil {
		return false, err
	}
	right, err := resolveOperand(rightRaw, env)
	if err != nil {
		return false, err
	}

	if leftNumber, leftOK := toFloat(left); leftOK {
		if rightNumber, rightOK := toFloat(right); rightOK {
			switch op {
			case "==":
				return leftNumber == rightNumber, nil
			case "!=":
				return leftNumber != rightNumber, nil
			case ">":
				return leftNumber > rightNumber, nil
			case "<":
				return leftNumber < rightNumber, nil
			case ">=":
				return leftNumber >= rightNumber, nil
			case "<=":
				return leftNumber <= rightNumber, nil
			}
		}
	}

	leftString := fmt.Sprint(left)
	rightString := fmt.Sprint(right)
	switch op {
	case "==":
		return leftString == rightString, nil
	case "!=":
		return leftString != rightString, nil
	default:
		return false, fmt.Errorf("operator %q requires numeric operands", op)
	}
}

func resolveOperand(input string, env map[string]any) (any, error) {
	input = strings.TrimSpace(trimOuterParens(input))
	if input == "true" {
		return true, nil
	}
	if input == "false" {
		return false, nil
	}
	if unquoted, err := strconv.Unquote(input); err == nil {
		return unquoted, nil
	}
	if number, err := strconv.ParseFloat(input, 64); err == nil {
		return number, nil
	}
	if strings.HasPrefix(input, "len(") && strings.HasSuffix(input, ")") {
		path := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(input, "len("), ")"))
		value, err := resolvePath(path, env)
		if err != nil {
			return nil, err
		}
		return float64(lengthOf(value)), nil
	}
	if strings.HasPrefix(input, "exists(") && strings.HasSuffix(input, ")") {
		path := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(input, "exists("), ")"))
		_, err := resolvePath(path, env)
		return err == nil, nil
	}
	return resolvePath(input, env)
}

func resolvePath(path string, env map[string]any) (any, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}
	segments := strings.Split(path, ".")
	var current any = env
	for _, segment := range segments {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("path %q stopped at non-map value %T", path, current)
		}
		next, ok := currentMap[segment]
		if !ok {
			return nil, fmt.Errorf("path %q not found", path)
		}
		current = next
	}
	return current, nil
}

func splitComparison(input, operator string) (string, string, bool) {
	depth := 0
	for idx := 0; idx < len(input)-len(operator)+1; idx++ {
		switch input[idx] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 && strings.HasPrefix(input[idx:], operator) {
			left := strings.TrimSpace(input[:idx])
			right := strings.TrimSpace(input[idx+len(operator):])
			if left != "" && right != "" {
				return left, right, true
			}
		}
	}
	return "", "", false
}

func splitTopLevel(input, operator string) []string {
	var parts []string
	depth := 0
	start := 0
	for idx := 0; idx < len(input)-len(operator)+1; idx++ {
		switch input[idx] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 && strings.HasPrefix(input[idx:], operator) {
			parts = append(parts, strings.TrimSpace(input[start:idx]))
			start = idx + len(operator)
			idx += len(operator) - 1
		}
	}
	if len(parts) == 0 {
		return []string{input}
	}
	parts = append(parts, strings.TrimSpace(input[start:]))
	return parts
}

func trimOuterParens(input string) string {
	for strings.HasPrefix(input, "(") && strings.HasSuffix(input, ")") {
		trimmed := strings.TrimSpace(input[1 : len(input)-1])
		if balanced(trimmed) {
			input = trimmed
			continue
		}
		break
	}
	return input
}

func balanced(input string) bool {
	depth := 0
	for _, ch := range input {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

func toFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func lengthOf(value any) int {
	if value == nil {
		return 0
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return rv.Len()
	default:
		return 0
	}
}

func truthy(value any) bool {
	if value == nil {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed != ""
	}
	if number, ok := toFloat(value); ok {
		return number != 0
	}
	return lengthOf(value) > 0
}
