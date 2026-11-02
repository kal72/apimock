package usecase

import (
	"apimock/internal/domain"
	"encoding/json"
	"net/http"
	"testing"
)

func group(kind, match string, rules ...domain.ConditionRule) domain.Config {
	return domain.Config{ConditionType: kind, Match: match, BodyMode: "raw", RawFormat: "json", Rules: rules}
}
func TestRequiredValueSemantics(t *testing.T) {
	for _, tc := range []struct {
		name         string
		value        any
		exists, want bool
	}{
		{"missing", nil, false, false}, {"null", nil, true, false}, {"empty string", "", true, false}, {"spaces", " \t\n", true, false}, {"empty array", []any{}, true, false}, {"empty object", map[string]any{}, true, false}, {"zero", json.Number("0"), true, true}, {"false", false, true, true}, {"text null", "null", true, true}, {"nonempty array", []any{0}, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ruleMatches(domain.ConditionRule{Operator: "exists", Required: true}, tc.value, tc.exists)
			if got != tc.want {
				t.Fatalf("required got %v want %v", got, tc.want)
			}
		})
	}
	if !ruleMatches(domain.ConditionRule{Operator: "exists"}, nil, true) {
		t.Fatal("exists must accept a present null without required")
	}
	if !ruleMatches(domain.ConditionRule{Operator: "empty"}, nil, false) {
		t.Fatal("empty must include absent fields")
	}
}
func TestTypedGroupsANDORAndSources(t *testing.T) {
	headerRules := []domain.ConditionRule{{Field: "authorization", Operator: "exists", Required: true}, {Field: "X-Role", Operator: "equals", Value: "admin", Required: true}}
	r := Request{Headers: http.Header{"Authorization": {""}, "X-Role": {"admin"}}, Query: map[string][]string{"page": {"2"}}, Body: map[string]any{"user": map[string]any{"active": false}}, BodyFormat: "json", BodyPresent: true}
	if conditionMatches(group("header", "all", headerRules...), r, nil) {
		t.Fatal("AND must reject empty required authorization")
	}
	if !conditionMatches(group("header", "any", headerRules...), r, nil) {
		t.Fatal("OR must accept the other matching clause")
	}
	r.Headers.Set("Authorization", "Bearer demo")
	if !conditionMatches(group("header", "all", headerRules...), r, nil) {
		t.Fatal("case-insensitive header AND should match")
	}
	if !conditionMatches(group("query", "all", domain.ConditionRule{Field: "page", Operator: "equals", Value: "2", Required: true}), r, nil) {
		t.Fatal("query did not match")
	}
	if !conditionMatches(group("path", "all", domain.ConditionRule{Field: "id", Operator: "equals", Value: "123", Required: true}), r, map[string]string{"id": "123"}) {
		t.Fatal("path did not match")
	}
	if !conditionMatches(group("body", "all", domain.ConditionRule{Field: "user.active", Operator: "equals", Value: "false", Required: true}), r, nil) {
		t.Fatal("false JSON boolean must remain a valid required value")
	}
	r.Query["page"] = []string{"", "2"}
	if conditionMatches(group("query", "all", domain.ConditionRule{Field: "page", Operator: "exists", Required: true}), r, nil) {
		t.Fatal("scalar query conditions must consistently inspect the first value")
	}
}
func TestBodyFormRawAndJSONArrays(t *testing.T) {
	jsonConfig := group("body", "all", domain.ConditionRule{Field: "items.0.id", Operator: "equals", Value: "0", Required: true})
	r := Request{BodyFormat: "json", BodyPresent: true, JSONBody: map[string]any{"items": []any{map[string]any{"id": json.Number("0")}}}}
	if !conditionMatches(jsonConfig, r, nil) {
		t.Fatal("nested JSON arrays must support numeric path segments")
	}
	form := group("body", "all", domain.ConditionRule{Field: "email", Operator: "exists", Required: true})
	form.BodyMode = "form"
	r = Request{BodyFormat: "form", Form: map[string][]string{"email": {"a@example.com"}}, BodyPresent: true}
	if !conditionMatches(form, r, nil) || conditionMatches(jsonConfig, r, nil) {
		t.Fatal("Form and Raw JSON conditions must be isolated")
	}
	text := group("body", "all", domain.ConditionRule{Field: "$raw", Operator: "contains", Value: "hello", Required: true})
	text.RawFormat = "text"
	r = Request{BodyFormat: "text", RawBody: "hello world", BodyPresent: true}
	if !conditionMatches(text, r, nil) || conditionMatches(form, r, nil) {
		t.Fatal("Raw Text did not match or leaked into Form")
	}
	r.RawBody = "   "
	if conditionMatches(text, r, nil) {
		t.Fatal("required text must reject whitespace")
	}
}
