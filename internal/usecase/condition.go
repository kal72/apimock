package usecase

import (
	"apimock/internal/domain"
	"encoding/json"
	"strconv"
	"strings"
)

func conditionMatches(c domain.Config, r Request, params map[string]string) bool {
	if c.ConditionType == "" && len(c.Rules) == 0 {
		value, exists := field(c.Field, r, params)
		switch c.Operator {
		case "equals":
			return exists && value == c.Value
		case "contains":
			return exists && strings.Contains(value, c.Value)
		case "exists":
			return exists
		}
		return false
	}
	if c.ConditionType == "body" {
		format := r.BodyFormat
		if format == "" {
			format = "json"
		}
		if c.BodyMode == "form" && format != "form" || c.BodyMode == "raw" && c.RawFormat != format {
			return false
		}
	}
	for _, rule := range c.Rules {
		value, exists := conditionValue(c, rule.Field, r, params)
		matched := ruleMatches(rule, value, exists)
		if c.Match == "all" && !matched {
			return false
		}
		if c.Match == "any" && matched {
			return true
		}
	}
	return c.Match == "all"
}

func ruleMatches(rule domain.ConditionRule, value any, exists bool) bool {
	empty := emptyValue(value)
	if rule.Required && (!exists || empty) {
		return false
	}
	text := valueText(value)
	switch rule.Operator {
	case "exists":
		return exists
	case "empty":
		return !exists || empty
	case "equals":
		return exists && text == rule.Value
	case "not_equals":
		return exists && text != rule.Value
	case "contains":
		return exists && strings.Contains(text, rule.Value)
	}
	return false
}
func emptyValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	}
	return false
}
func valueText(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	b, _ := json.Marshal(value)
	return string(b)
}
func first(values []string, exists bool) (any, bool) {
	if !exists || len(values) == 0 {
		return nil, false
	}
	return values[0], true
}
func conditionValue(c domain.Config, key string, r Request, params map[string]string) (any, bool) {
	switch c.ConditionType {
	case "header":
		for name, values := range r.Headers {
			if strings.EqualFold(name, key) {
				return first(values, true)
			}
		}
	case "query":
		values, ok := r.Query[key]
		return first(values, ok)
	case "path":
		value, ok := params[key]
		return value, ok
	case "body":
		if c.BodyMode == "form" {
			values, ok := r.Form[key]
			return first(values, ok)
		}
		if c.RawFormat == "text" {
			return r.RawBody, r.BodyPresent || r.RawBody != ""
		}
		var root any = r.JSONBody
		if root == nil {
			root = r.Body
		}
		if root == nil && !r.BodyPresent {
			return nil, false
		}
		if key == "$" {
			return root, true
		}
		return jsonField(root, key)
	}
	return nil, false
}
func jsonField(value any, path string) (any, bool) {
	for _, part := range strings.Split(path, ".") {
		switch object := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = object[part]
			if !ok {
				return nil, false
			}
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(object) {
				return nil, false
			}
			value = object[index]
		default:
			return nil, false
		}
	}
	return value, true
}

// ConditionCheck explains a playground request without changing execution order.
type ConditionCheck struct {
	ID        string      `json:"id"`
	Label     string      `json:"label"`
	Matched   bool        `json:"matched"`
	Evaluated bool        `json:"evaluated"`
	Reason    string      `json:"reason"`
	Rules     []RuleCheck `json:"rules"`
}
type RuleCheck struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Required bool   `json:"required"`
	Matched  bool   `json:"matched"`
	Exists   bool   `json:"exists"`
}

func ExplainConditions(f domain.Flow, r Request, params map[string]string, trace []string) []ConditionCheck {
	visited := map[string]bool{}
	for _, id := range trace {
		visited[id] = true
	}
	checks := []ConditionCheck{}
	for _, n := range f.Nodes {
		if n.Type != "condition" {
			continue
		}
		c := n.Config
		check := ConditionCheck{ID: n.ID, Label: n.Label, Matched: conditionMatches(c, r, params), Evaluated: visited[n.ID], Rules: []RuleCheck{}}
		if c.ConditionType != "" {
			format := r.BodyFormat
			if format == "" {
				format = "json"
			}
			mismatch := c.ConditionType == "body" && (c.BodyMode == "form" && format != "form" || c.BodyMode == "raw" && c.RawFormat != format)
			if mismatch {
				check.Reason = "Request body format does not match this condition."
			} else {
				for _, rule := range c.Rules {
					value, exists := conditionValue(c, rule.Field, r, params)
					check.Rules = append(check.Rules, RuleCheck{Field: rule.Field, Operator: rule.Operator, Required: rule.Required, Matched: ruleMatches(rule, value, exists), Exists: exists})
				}
			}
		}
		if check.Reason == "" {
			if check.Matched {
				check.Reason = "Condition matches this request."
			} else {
				check.Reason = "Condition does not match this request."
			}
		}
		if !check.Evaluated {
			check.Reason += " Not visited during execution; this is a diagnostic check only."
		}
		checks = append(checks, check)
	}
	return checks
}
