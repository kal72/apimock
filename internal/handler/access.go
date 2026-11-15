package handler

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

func (h Handler) access(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		if h.Config.ManagedDeployment() && !strings.EqualFold(r.Host, h.Config.PublicHost()) {
			http.Error(w, "unexpected host", http.StatusMisdirectedRequest)
			return
		}
		// Run mock preflight before canonical redirects, preserving existing clients.
		if strings.HasPrefix(r.URL.Path, "/public/") {
			h.mockCORS(next).ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h Handler) managementAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if h.Auth == nil && h.Config.AdminUsername != "" {
			username, password, ok := r.BasicAuth()
			gotUser, gotPassword := sha256.Sum256([]byte(username)), sha256.Sum256([]byte(password))
			wantUser, wantPassword := sha256.Sum256([]byte(h.Config.AdminUsername)), sha256.Sum256([]byte(h.Config.AdminPassword))
			userMatch := subtle.ConstantTimeCompare(gotUser[:], wantUser[:])
			passwordMatch := subtle.ConstantTimeCompare(gotPassword[:], wantPassword[:])
			if !ok || userMatch&passwordMatch != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="Mocklab management", charset="UTF-8"`)
				http.Error(w, "administrator authentication required", http.StatusUnauthorized)
				return
			}
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			origin := r.Header.Get("Origin")
			if origin != "" && !h.sameOrigin(r, origin) {
				http.Error(w, "cross-origin management write denied", http.StatusForbidden)
				return
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				http.Error(w, "cross-site management write denied", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

const corsMethods = "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS"

func allowedMethod(method string) bool {
	switch method {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		return true
	}
	return false
}

type mockCORSKey struct{}

func (h Handler) mockCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if applied, _ := r.Context().Value(mockCORSKey{}).(bool); applied {
			next.ServeHTTP(w, r)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), mockCORSKey{}, true))
		w.Header().Add("Vary", "Origin")
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !h.Config.AllowsOrigin(origin) && !h.sameOrigin(r, origin) {
			http.Error(w, "mock origin is not allowed", http.StatusForbidden)
			return
		}
		preflight := r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
		if preflight {
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")
			if !allowedMethod(r.Header.Get("Access-Control-Request-Method")) {
				http.Error(w, "mock method is not allowed", http.StatusForbidden)
				return
			}
			for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
				header = strings.TrimSpace(header)
				if header != "" && (!h.Config.AllowsHeader(header)) {
					http.Error(w, "mock header is not allowed", http.StatusForbidden)
					return
				}
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", corsMethods)
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(h.Config.CORSHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		next.ServeHTTP(w, r)
	})
}

func (h Handler) sameOrigin(r *http.Request, origin string) bool {
	if h.Config.ManagedDeployment() {
		return origin == h.Config.PublicBaseURL
	}
	return origin == h.Config.PublicBaseURL || origin == "http://"+r.Host || origin == "https://"+r.Host
}

// Error fallbacks preserve the public mock boundary without exposing management handlers.
func (h Handler) routingFallback(next http.Handler) http.Handler {
	management := h.managementAccess(h.requireSession(next))
	public := h.mockCORS(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/public/") || strings.HasPrefix(r.URL.Path, "/share/") {
			public.ServeHTTP(w, r)
			return
		}
		management.ServeHTTP(w, r)
	})
}
