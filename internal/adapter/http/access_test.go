package httpadapter

import (
	"apimock/internal/config"
	"apimock/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdministratorBoundary(t *testing.T) {
	c := config.Config{Environment: "production", PublicBaseURL: "https://mock.example.com", AdminUsername: "admin", AdminPassword: "test-password-long-enough"}
	h := configuredHandler(c).Routes()
	for _, tc := range []struct {
		name, path, username, password string
		want                           int
	}{
		{"protected UI", "/app", "", "", 401},
		{"protected read API", "/api/workspaces", "", "", 401},
		{"protected settings", "/api/config", "", "", 401},
		{"invalid credentials", "/api/workspaces", "admin", "wrong", 401},
		{"authenticated UI", "/app", "admin", c.AdminPassword, 200},
		{"authenticated API", "/api/workspaces", "admin", c.AdminPassword, 200},
		{"public mock", "/public/demo/products/123", "", "", 200},
		{"mock authorization is independent", "/public/demo/products/123", "other", "mock-token", 200},
		{"health without authentication", "/healthz", "", "", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "https://mock.example.com"+tc.path, nil)
			if tc.username != "" {
				r.SetBasicAuth(tc.username, tc.password)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if tc.want == 401 && w.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("missing login challenge")
			}
		})
	}
	r := httptest.NewRequest("GET", "https://unexpected.example/api/workspaces", nil)
	r.SetBasicAuth(c.AdminUsername, c.AdminPassword)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 421 {
		t.Fatalf("unexpected host status: %d", w.Code)
	}
	r = httptest.NewRequest("GET", "https://mock.example.com/api/config", nil)
	r.SetBasicAuth(c.AdminUsername, c.AdminPassword)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), c.PublicBaseURL) || strings.Contains(w.Body.String(), c.AdminPassword) || strings.Contains(w.Body.String(), "admin") {
		t.Fatalf("invalid public config: %s", w.Body.String())
	}
}

func TestMockCORS(t *testing.T) {
	c := config.Config{CORSOrigins: []string{"https://frontend.example.com"}, CORSHeaders: []string{"Content-Type", "Authorization", "X-Api-Key"}, AdminUsername: "admin", AdminPassword: "test-password-long-enough"}
	h := configuredHandler(c).Routes()
	for _, tc := range []struct {
		name, method, path, origin, requestedMethod, headers string
		want                                                 int
		allowed                                              bool
	}{
		{"JSON preflight", "OPTIONS", "/public/demo/products/123", "https://frontend.example.com", "POST", "content-type, authorization", 204, true},
		{"allowed actual response", "GET", "/public/demo/products/123", "https://frontend.example.com", "", "", 200, true},
		{"CORS on error", "GET", "/public/demo/missing", "https://frontend.example.com", "", "", 404, true},
		{"rejected origin", "GET", "/public/demo/products/123", "https://evil.example", "", "", 403, false},
		{"rejected lookalike origin", "OPTIONS", "/public/demo/products/123", "https://frontend.example.com.evil", "GET", "", 403, false},
		{"rejected header", "OPTIONS", "/public/demo/products/123", "https://frontend.example.com", "POST", "X-Unlisted", 403, false},
		{"rejected method", "OPTIONS", "/public/demo/products/123", "https://frontend.example.com", "TRACE", "", 403, false},
		{"no origin curl", "GET", "/public/demo/products/123", "", "", "", 200, false},
		{"management has no public CORS", "OPTIONS", "/api/workspaces", "https://frontend.example.com", "POST", "content-type", 401, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("Access-Control-Request-Method", tc.requestedMethod)
			r.Header.Set("Access-Control-Request-Headers", tc.headers)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if got := w.Header().Get("Access-Control-Allow-Origin"); (got != "") != tc.allowed || (tc.allowed && got != tc.origin) {
				t.Fatalf("unexpected allowed origin: %q", got)
			}
			if w.Header().Get("Access-Control-Allow-Credentials") != "" {
				t.Fatal("credentialed CORS must remain disabled")
			}
		})
	}
}

func TestWildcardCORSAndExplicitOptions(t *testing.T) {
	h := configuredHandler(config.Config{CORSOrigins: []string{"*"}})
	repo := h.Service.Repo.(*testRepository)
	f := domain.DefaultFlow()
	f.Nodes[1].Config.Status = 202
	repo.endpoints = append(repo.endpoints, domain.Endpoint{ID: "options", WorkspaceID: "w1", Method: "OPTIONS", Path: "/products/:id", Flow: f})
	r := httptest.NewRequest("OPTIONS", "/public/demo/products/123", nil)
	r.Header.Set("Origin", "https://any.example")
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, r)
	if w.Code != 202 || w.Header().Get("Access-Control-Allow-Origin") != "https://any.example" {
		t.Fatalf("explicit OPTIONS was intercepted: %d %v", w.Code, w.Header())
	}
}

func TestManagementRejectsCrossOriginWithValidCredentials(t *testing.T) {
	c := config.Config{Environment: "production", PublicBaseURL: "https://mock.example.com", AdminUsername: "admin", AdminPassword: "test-password-long-enough"}
	for _, origin := range []string{"https://frontend.example.com", "http://mock.example.com"} {
		r := httptest.NewRequest("POST", "https://mock.example.com/api/workspaces", strings.NewReader(`{}`))
		r.SetBasicAuth(c.AdminUsername, c.AdminPassword)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		configuredHandler(c).Routes().ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("origin %s returned %d", origin, w.Code)
		}
	}
}

func TestInternalPublicRoutes(t *testing.T) {
	c := config.Config{Environment: "internal", PublicBaseURL: "http://apimock.internal:8080", AdminUsername: "admin", AdminPassword: "test-password-long-enough"}
	h := configuredHandler(c)
	repo := h.Service.Repo.(*testRepository)
	repo.endpoints = append(repo.endpoints, domain.Endpoint{ID: "root", WorkspaceID: "w1", Method: "GET", Path: "/", Flow: domain.DefaultFlow()})
	for _, tc := range []struct {
		method, path, host, origin string
		auth                       bool
		want                       int
	}{
		{"GET", "/public/demo/products/123", "apimock.internal:8080", "", false, 200},
		{"GET", "/public/demo/products/123", "apimock.internal:8080", "http://apimock.internal:8080", false, 200},
		{"GET", "/public/demo/", "apimock.internal:8080", "", false, 200},
		{"GET", "/api/workspaces", "apimock.internal:8080", "", false, 401},
		{"GET", "/api/workspaces", "apimock.internal:8080", "", true, 200},
		{"GET", "/public/demo/products/123", "unexpected.internal:8080", "", false, 421},
		{"GET", "/mock/demo/products/123", "apimock.internal:8080", "", true, 404},
		{"POST", "/api/workspaces", "apimock.internal:8080", "http://other.internal", true, 403},
	} {
		r := httptest.NewRequest(tc.method, "http://"+tc.host+tc.path, nil)
		r.Header.Set("Origin", tc.origin)
		if tc.auth {
			r.SetBasicAuth(c.AdminUsername, c.AdminPassword)
		}
		w := httptest.NewRecorder()
		h.Routes().ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s %s host=%s: status %d, want %d", tc.method, tc.path, tc.host, w.Code, tc.want)
		}
	}
}
