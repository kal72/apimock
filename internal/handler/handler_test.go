package handler

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
	var explanation struct {
		ResponseID string                   `json:"response_id"`
		Conditions []usecase.ConditionCheck `json:"conditions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &explanation); err != nil || explanation.ResponseID != "response" || explanation.Conditions == nil {
		t.Fatalf("missing playground diagnostics: %s", w.Body.String())
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

func TestDynamicPublicAndPlaygroundResponses(t *testing.T) {
	h := configuredHandler(config.Config{})
	repo := h.Service.Repo.(*testRepository)
	e := &repo.endpoints[0]
	e.Flow.Nodes[1].Config.ResponseMode = "dynamic"
	e.Flow.Nodes[1].Config.Body = `{"id":"{{ request.path.id }}","name":"{{ request.query.name }}"}`
	handler := h.Routes()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/public/demo/products/42?name=Alice", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"id":"42"`) || !strings.Contains(w.Body.String(), `"name":"Alice"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/public/demo/products/42", nil))
	if w.Code != 500 || strings.Contains(w.Body.String(), "request.query.name") {
		t.Fatal("public error exposed template internals", w.Code, w.Body.String())
	}
	payload, _ := json.Marshal(map[string]any{"flow": e.Flow, "params": map[string]string{"id": "42"}})
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("POST", "/api/test", strings.NewReader(string(payload))))
	if w.Code != 422 || !strings.Contains(w.Body.String(), "request.query.name") {
		t.Fatal("playground error lacks details", w.Code, w.Body.String())
	}
	e.Flow.Nodes[1].Config.Body = `"{{ unknown.execute }}"`
	payload, _ = json.Marshal(e)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("PUT", "/api/workspaces/w1/endpoints/e1", strings.NewReader(string(payload))))
	if w.Code != 400 {
		t.Fatal("invalid dynamic template saved", w.Code, w.Body.String())
	}
}

func TestGeneratorDocumentation(t *testing.T) {
	w := httptest.NewRecorder()
	testHandler().ServeHTTP(w, httptest.NewRequest("GET", "/docs/generators", nil))
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Fatal("documentation page not rendered")
	}
	if strings.Count(body, `data-available="true"`) != 4 || strings.Count(body, `data-available="false"`) != 25 {
		t.Fatal("generator catalog availability is incorrect")
	}
	if strings.Count(body, ">Copy JSON</button>") != 4 || !strings.Contains(body, "not runnable yet") {
		t.Fatal("planned generators must not have runnable copy controls")
	}
	for _, name := range []string{"generate.uuid", "generate.timestamp", "generate.integer", "generate.choice", "generate.person.full_name"} {
		if !strings.Contains(body, name) {
			t.Fatalf("missing generator %s", name)
		}
	}
}
