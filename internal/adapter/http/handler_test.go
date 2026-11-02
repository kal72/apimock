package httpadapter

import (
	"apimock/internal/config"
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testRepository struct {
	workspaces []domain.Workspace
	endpoints  []domain.Endpoint
}

func (r *testRepository) Workspaces(context.Context) ([]domain.Workspace, error) {
	return r.workspaces, nil
}
func (r *testRepository) SaveWorkspace(_ context.Context, w domain.Workspace) error {
	r.workspaces = append(r.workspaces, w)
	return nil
}
func (r *testRepository) DeleteWorkspace(context.Context, string) error { return nil }
func (r *testRepository) Endpoints(_ context.Context, wid string) ([]domain.Endpoint, error) {
	out := []domain.Endpoint{}
	for _, e := range r.endpoints {
		if e.WorkspaceID == wid {
			out = append(out, e)
		}
	}
	return out, nil
}
func (r *testRepository) SaveEndpoint(_ context.Context, e domain.Endpoint) error {
	r.endpoints = append(r.endpoints, e)
	return nil
}
func (r *testRepository) DeleteEndpoint(context.Context, string, string) error { return nil }
func testHandler() http.Handler {
	return configuredHandler(config.Config{}).Routes()
}
func configuredHandler(settings config.Config) Handler {
	return Handler{Config: settings, Service: usecase.Service{Repo: &testRepository{workspaces: []domain.Workspace{{ID: "w1", Slug: "demo", Name: "Demo"}}, endpoints: []domain.Endpoint{{ID: "e1", WorkspaceID: "w1", Name: "Product", Method: "GET", Path: "/products/:id", Flow: domain.DefaultFlow()}}}}, Assets: http.NotFoundHandler()}
}
func TestTransport(t *testing.T) {
	h := testHandler()
	tests := []struct {
		name, method, path, body, origin string
		status                           int
	}{
		{"page", "GET", "/app", "", "", 200},
		{"mock parameter", "GET", "/public/demo/products/123", "", "", 200},
		{"missing workspace", "GET", "/public/missing/products/123", "", "", 404},
		{"method mismatch", "POST", "/public/demo/products/123", `{}`, "", 404},
		{"invalid payload", "POST", "/api/workspaces", `{"name":"Demo","slug":"BAD SLUG"}`, "", 400},
		{"trailing JSON", "POST", "/api/workspaces", `{} {}`, "", 400},
		{"cross-origin write", "POST", "/api/workspaces", `{}`, "https://evil.example", 403},
		{"create workspace", "POST", "/api/workspaces", `{"name":"Demo","slug":"new-demo"}`, "", 200},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
		})
	}
}
func TestPlayground(t *testing.T) {
	payload := map[string]any{"flow": domain.DefaultFlow(), "method": "GET", "params": map[string]string{"id": "1"}}
	b, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	testHandler().ServeHTTP(w, httptest.NewRequest("POST", "/api/test", strings.NewReader(string(b))))
	var result usecase.Result
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || result.Status != 200 || len(result.Trace) != 2 {
		t.Fatalf("unexpected response %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateEndpointWithoutFlowStartsConnected(t *testing.T) {
	h := configuredHandler(config.Config{})
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, httptest.NewRequest("POST", "/api/workspaces/w1/endpoints", strings.NewReader(`{"name":"Default flow","method":"GET","path":"/connected"}`)))
	if w.Code != 200 {
		t.Fatalf("create failed: %d %s", w.Code, w.Body.String())
	}
	var endpoint domain.Endpoint
	if err := json.Unmarshal(w.Body.Bytes(), &endpoint); err != nil {
		t.Fatal(err)
	}
	if err := endpoint.Flow.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(endpoint.Flow.Edges) != 1 || endpoint.Flow.Edges[0].From != "request" || endpoint.Flow.Edges[0].To != "response" {
		t.Fatalf("initial flow is disconnected: %+v", endpoint.Flow)
	}
	w = httptest.NewRecorder()
	h.Routes().ServeHTTP(w, httptest.NewRequest("GET", "/public/demo/connected", nil))
	if w.Code != 200 {
		t.Fatalf("initial endpoint is not callable: %d %s", w.Code, w.Body.String())
	}
}
