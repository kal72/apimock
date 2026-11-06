package httpadapter

import (
	"apimock/internal/config"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChiParameterAndRedirectCompatibility(t *testing.T) {
	baseline := http.NewServeMux()
	baseline.HandleFunc("GET /api/workspaces/{wid}/endpoints", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.PathValue("wid"))) })
	baseline.HandleFunc("/public/{slug}/{path...}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.PathValue("slug") + "|" + r.PathValue("path")))
	})
	router := chi.NewRouter()
	getHead(router, "/api/workspaces/{wid}/endpoints", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(routeParam(r, "wid"))) })
	router.HandleFunc("/public/{slug}/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(routeParam(r, "slug") + "|" + routeParam(r, "path")))
	})
	actual := canonicalPath(router)
	for _, path := range []string{
		"/api/workspaces/w1/endpoints", "/api/workspaces/w%31/endpoints", "/%61pi/workspaces/w1/endpoints",
		"/api/workspaces/w%2F1/endpoints", "/api/workspaces/w%252F1/endpoints", "/api/workspaces/%2e%2e/endpoints",
		"/public/demo/", "/public/demo/a%2Fb", "/public/demo/a%252Fb", "/public/demo/a%25b", "/public/demo/a+b", "/public/demo/a%20b",
		"/public/d%65mo/path", "/public/demo/./path?a=1", "/public/demo/../path", "/public//demo/products/123", "/api//workspaces/w1/endpoints",
	} {
		t.Run(path, func(t *testing.T) {
			a, b := httptest.NewRecorder(), httptest.NewRecorder()
			baseline.ServeHTTP(a, httptest.NewRequest("GET", path, nil))
			actual.ServeHTTP(b, httptest.NewRequest("GET", path, nil))
			if a.Code != b.Code || a.Header().Get("Location") != b.Header().Get("Location") || a.Body.String() != b.Body.String() {
				t.Fatalf("ServeMux: %d %q %q; Chi: %d %q %q", a.Code, a.Header().Get("Location"), a.Body.String(), b.Code, b.Header().Get("Location"), b.Body.String())
			}
		})
	}
}
func TestChiMethodsAndPublicWildcard(t *testing.T) {
	h := testHandler()
	for _, tc := range []struct {
		method, path string
		status       int
		allow        string
	}{
		{"HEAD", "/app", 200, ""}, {"HEAD", "/api/workspaces", 200, ""}, {"HEAD", "/docs/generators", 200, ""},
		{"PATCH", "/app", 405, "GET, HEAD"}, {"OPTIONS", "/api/workspaces", 405, "GET, HEAD, POST"},
		{"GET", "/assets", 307, ""}, {"GET", "/public/demo", 307, ""}, {"GET", "/api/workspaces/", 404, ""},
		{"GET", "/mock/demo/products/123", 404, ""}, {"GET", "/public/demo/missing", 404, ""},
		{"BREW", "/public/demo/products/123", 404, ""},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status || w.Header().Get("Allow") != tc.allow {
			t.Fatalf("%s %s: %d %v", tc.method, tc.path, w.Code, w.Header())
		}
	}
}
func TestWorkspacePermissionFailsClosed(t *testing.T) {
	for _, role := range []string{"viewer", "editor", "owner", "invalid"} {
		called := false
		h := Handler{}
		handler := h.requireWorkspace(role)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/no-parameters", nil))
		if w.Code != 403 || called {
			t.Fatalf("missing workspace skipped policy %s", role)
		}
	}
}
func TestEncodedPathsCannotBypassManagementBoundary(t *testing.T) {
	cfg := config.Config{AdminUsername: "admin", AdminPassword: "long-test-password"}
	h := configuredHandler(cfg).Routes()
	for _, path := range []string{"/%61pi/workspaces", "/api%2Fworkspaces", "/api/workspaces/w1%2Fmembers/endpoints"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 401 {
			t.Fatalf("unauthenticated encoded path %s returned %d", path, w.Code)
		}
	}
	encoded := httptest.NewRecorder()
	h.ServeHTTP(encoded, httptest.NewRequest("GET", "/public%2Fdemo%2F..%2Fapi%2Fusers", nil))
	if encoded.Code != http.StatusNotFound {
		t.Fatalf("encoded separators reached a handler: %d", encoded.Code)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/public/demo/../../api/workspaces", nil))
	if w.Code != 307 || w.Header().Get("Location") != "/api/workspaces" {
		t.Fatal(w.Code, w.Header())
	}
	redirected := httptest.NewRecorder()
	h.ServeHTTP(redirected, httptest.NewRequest("GET", w.Header().Get("Location"), nil))
	if redirected.Code != 401 {
		t.Fatal("canonical redirect bypassed authentication")
	}
}
func TestForwardedHeadersDoNotChangeLoginLimiterIdentity(t *testing.T) {
	a := Identity{}
	for i := 0; i < 21; i++ {
		r := httptest.NewRequest("POST", "/api/auth/login", nil)
		r.RemoteAddr = "192.0.2.10:1234"
		r.Header.Set("X-Forwarded-For", strings.Repeat("1", i+1))
		r.Header.Set("X-Real-IP", "198.51.100.1")
		limited := a.limited(r, strings.Repeat("u", i+1))
		if limited != (i == 20) {
			t.Fatalf("limiter changed identity: attempt %d", i)
		}
	}
}
