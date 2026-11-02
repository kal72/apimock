package domain

import "testing"

func TestTypedConditionValidation(t *testing.T) {
	base := Config{ConditionType: "header", Match: "all", Rules: []ConditionRule{{Field: "X-Status", Operator: "equals", Value: "active"}, {Field: "x-status", Operator: "equals", Value: "inactive"}}}
	if ValidateCondition(base) == nil {
		t.Fatal("contradictory header AND must fail ignoring header name case")
	}
	base.Match = "any"
	if err := ValidateCondition(base); err != nil {
		t.Fatal(err)
	}
	base.Match = "all"
	base.Rules[1] = base.Rules[0]
	if err := ValidateCondition(base); err != nil {
		t.Fatalf("duplicate warning must not be a blocking validation error: %v", err)
	}
	for _, rule := range []ConditionRule{{Field: "email", Operator: "equals", Value: " ", Required: true}, {Field: "email", Operator: "empty", Required: true}, {Field: "", Operator: "exists"}} {
		c := Config{ConditionType: "query", Match: "all", Rules: []ConditionRule{rule}}
		if ValidateCondition(c) == nil {
			t.Fatalf("accepted invalid rule: %+v", rule)
		}
	}
	for _, c := range []Config{{ConditionType: "body", Match: "all", BodyMode: "raw", RawFormat: "text", Rules: []ConditionRule{{Field: "email", Operator: "exists"}}}, {ConditionType: "body", Match: "all", BodyMode: "invalid", Rules: []ConditionRule{{Field: "id", Operator: "exists"}}}, {ConditionType: "mixed", Match: "all", Rules: []ConditionRule{{Field: "id", Operator: "exists"}}}} {
		if ValidateCondition(c) == nil {
			t.Fatalf("accepted invalid typed condition: %+v", c)
		}
	}
}
func TestTypedPathFieldsMustExistOnRoute(t *testing.T) {
	f := DefaultFlow()
	f.Nodes = append(f.Nodes, Node{ID: "check", Type: "condition", Config: Config{ConditionType: "path", Match: "all", Rules: []ConditionRule{{Field: "missing", Operator: "exists", Required: true}}}})
	f.Edges = []Edge{{From: "request", To: "check"}, {From: "check", To: "response", Branch: "true"}, {From: "check", To: "response", Branch: "false"}}
	e := Endpoint{Name: "Example", Method: "GET", Path: "/users/:id", Flow: f}
	if e.Validate() == nil {
		t.Fatal("accepted nonexistent path parameter")
	}
	e.Flow.Nodes[2].Config.Rules[0].Field = "id"
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
}
