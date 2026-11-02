package domain

import (
	"errors"
	"fmt"
	"strings"
)

type ConditionRule struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value,omitempty"`
	Required bool   `json:"required"`
}

func ValidateCondition(c Config) error {
	if c.ConditionType == "" && len(c.Rules) == 0 {
		prefix, name, ok := strings.Cut(c.Field, ".")
		if !ok || name == "" || (prefix != "headers" && prefix != "query" && prefix != "params" && prefix != "body") {
			return errors.New("legacy condition fields must use headers., query., params., or body.")
		}
		if c.Operator != "equals" && c.Operator != "contains" && c.Operator != "exists" {
			return errors.New("invalid legacy condition operator")
		}
		return nil
	}
	if c.ConditionType != "header" && c.ConditionType != "query" && c.ConditionType != "path" && c.ConditionType != "body" {
		return errors.New("condition type must be header, query, path, or body")
	}
	if c.Match != "all" && c.Match != "any" {
		return errors.New("condition matching must be all (AND) or any (OR)")
	}
	if len(c.Rules) < 1 || len(c.Rules) > 50 {
		return errors.New("condition nodes must have 1–50 field rules")
	}
	if c.ConditionType == "body" {
		if c.BodyMode != "form" && c.BodyMode != "raw" {
			return errors.New("body mode must be form or raw")
		}
		if c.BodyMode == "raw" && c.RawFormat != "json" && c.RawFormat != "text" {
			return errors.New("raw body format must be json or text")
		}
	}
	for _, r := range c.Rules {
		if strings.TrimSpace(r.Field) == "" || len(r.Field) > 256 {
			return errors.New("condition field name is required (maximum 256 characters)")
		}
		if c.ConditionType == "header" && !validHeader(r.Field, "") {
			// Request conditions may inspect response-restricted names, but never invalid tokens.
			for _, ch := range r.Field {
				if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", ch)) {
					return errors.New("invalid condition header name")
				}
			}
		}
		if c.ConditionType == "body" && c.BodyMode == "raw" && c.RawFormat == "text" && r.Field != "$raw" {
			return errors.New("raw text uses the $raw field")
		}
		if c.ConditionType == "body" && c.BodyMode == "raw" && c.RawFormat == "json" && r.Field != "$" {
			for _, part := range strings.Split(r.Field, ".") {
				if part == "" {
					return errors.New("invalid JSON field path")
				}
			}
		}
		switch r.Operator {
		case "equals", "not_equals", "contains", "exists", "empty":
		default:
			return errors.New("invalid condition operator")
		}
		if r.Required && (r.Operator == "empty" || r.Operator == "equals" && strings.TrimSpace(r.Value) == "") {
			return fmt.Errorf("required field %s cannot use an empty-value check", r.Field)
		}
	}
	if c.Match == "all" {
		for i, a := range c.Rules {
			for _, b := range c.Rules[i+1:] {
				same := a.Field == b.Field
				if c.ConditionType == "header" {
					same = strings.EqualFold(a.Field, b.Field)
				}
				if !same {
					continue
				}
				if a.Operator == "equals" && b.Operator == "equals" && a.Value != b.Value || a.Operator == "equals" && b.Operator == "not_equals" && a.Value == b.Value || b.Operator == "equals" && a.Operator == "not_equals" && a.Value == b.Value || a.Required && b.Operator == "empty" || b.Required && a.Operator == "empty" || a.Operator == "equals" && b.Operator == "contains" && !strings.Contains(a.Value, b.Value) || b.Operator == "equals" && a.Operator == "contains" && !strings.Contains(b.Value, a.Value) {
					return fmt.Errorf("AND conflict for field %s: rules cannot be satisfied together", a.Field)
				}
			}
		}
	}
	return nil
}
