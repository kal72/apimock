package usecase

import (
	"apimock/internal/domain"
	"context"
	"errors"
	"net/http"
	"testing"
)

func conditionalFlow() domain.Flow {
	f := domain.DefaultFlow()
	f.Nodes = append(f.Nodes, domain.Node{ID: "check", Type: "condition", Config: domain.Config{Field: "params.id", Operator: "equals", Value: "123"}}, domain.Node{ID: "missing", Type: "response", Config: domain.Config{Status: 404, Body: `{"error":"missing"}`}})
	f.Edges = []domain.Edge{{From: "request", To: "check"}, {From: "check", To: "response", Branch: "true"}, {From: "check", To: "missing", Branch: "false"}}
	return f
}
func TestEvaluateBranches(t *testing.T) {
	for _, tc := range []struct {
		id     string
		status int
		last   string
	}{{"123", 200, "response"}, {"other", 404, "missing"}} {
		t.Run(tc.id, func(t *testing.T) {
			result, err := Evaluate(context.Background(), conditionalFlow(), Request{}, map[string]string{"id": tc.id})
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != tc.status || result.Trace[len(result.Trace)-1] != tc.last {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}
}
func TestInvalidFlows(t *testing.T) {
	tests := map[string]func(*domain.Flow){"cycle": func(f *domain.Flow) {
		f.Nodes[1].Type = "headers"
		f.Edges = append(f.Edges, domain.Edge{From: "response", To: "request"})
	}, "dangling": func(f *domain.Flow) { f.Edges[0].To = "gone" }, "unreachable": func(f *domain.Flow) {
		f.Nodes = append(f.Nodes, domain.Node{ID: "unused", Type: "response", Config: domain.Config{Status: 200}})
	}, "missing condition branch": func(f *domain.Flow) { *f = conditionalFlow(); f.Edges = f.Edges[:2] }, "duplicate id": func(f *domain.Flow) { f.Nodes[1].ID = "request" }, "header injection": func(f *domain.Flow) { f.Nodes[1].Config.Headers["X-Test"] = "bad\r\nX-Injected: 1" }, "invalid json": func(f *domain.Flow) { f.Nodes[1].Config.Body = "{" }}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			f := domain.DefaultFlow()
			mutate(&f)
			if f.Validate() == nil {
				t.Fatal("expected validation failure")
			}
		})
	}
}
func TestFieldTypes(t *testing.T) {
	r := Request{Headers: http.Header{"Authorization": []string{"token"}}, Query: map[string][]string{"page": {"2"}}, Body: map[string]any{"user": map[string]any{"active": true}}}
	for _, tc := range []struct {
		key, want string
		exists    bool
	}{{"headers.authorization", "token", true}, {"query.page", "2", true}, {"body.user.active", "true", true}, {"body.user.missing", "", false}, {"params.id", "123", true}} {
		got, ok := field(tc.key, r, map[string]string{"id": "123"})
		if got != tc.want || ok != tc.exists {
			t.Errorf("%s: %q %v", tc.key, got, ok)
		}
	}
}
func TestDelayHonorsCancellation(t *testing.T) {
	f := domain.DefaultFlow()
	f.Nodes = append(f.Nodes, domain.Node{ID: "wait", Type: "delay", Config: domain.Config{DelayMS: 10000}})
	f.Edges = []domain.Edge{{From: "request", To: "wait"}, {From: "wait", To: "response"}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Evaluate(ctx, f, Request{}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
func TestMatchAndSpecificity(t *testing.T) {
	if _, ok := match("/products/:id", "/products/"); ok {
		t.Fatal("empty parameter matched")
	}
	if _, ok := match("/products/:id", "/products/1/extra"); ok {
		t.Fatal("extra segment matched")
	}
	if specificity("/products/search") <= specificity("/products/:id") {
		t.Fatal("static route must win")
	}
}

func TestResponseDelayHonorsCancellation(t *testing.T) {
	f := domain.DefaultFlow()
	f.Nodes[1].Config.DelayMS = 10000
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Evaluate(ctx, f, Request{}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("response delay ignored cancellation: %v", err)
	}
}

func TestResponseDelayValidation(t *testing.T) {
	for _, delay := range []int{-1, 10001} {
		f := domain.DefaultFlow()
		f.Nodes[1].Config.DelayMS = delay
		if err := f.Validate(); err == nil {
			t.Fatalf("accepted invalid response delay: %d", delay)
		}
	}
}
