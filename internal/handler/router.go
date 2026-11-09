package handler

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
)

// getHead preserves ServeMux GET routes accepting HEAD, without altering mock methods.
func getHead(r chi.Router, pattern string, handler http.HandlerFunc) {
	r.Get(pattern, handler)
	r.Head(pattern, handler)
}

// routeParam decodes escaped-path captures once, matching the previous transport contract.
func routeParam(r *http.Request, key string) string {
	if key == "path" {
		key = "*"
	}
	value := chi.URLParam(r, key)
	if r.URL.RawPath != "" {
		decoded, err := url.PathUnescape(value)
		if err == nil {
			return decoded
		}
	}
	return value
}

// canonicalPath preserves redirects for literal dot segments and repeated slashes.
// Escaped segments remain escaped while routing, so %2F cannot create management routes.
func canonicalPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		escaped := r.URL.EscapedPath()
		clean := path.Clean(escaped)
		if strings.HasSuffix(escaped, "/") && clean != "/" {
			clean += "/"
		}
		if r.Method == http.MethodConnect {
			clean = escaped
		}
		if clean != escaped {
			if router, ok := next.(chi.Router); ok && !strings.HasSuffix(clean, "/") {
				ctx := chi.NewRouteContext()
				matched := router.Match(ctx, r.Method, clean)
				if (!matched && router.Match(chi.NewRouteContext(), r.Method, clean+"/")) || (matched && ctx.RoutePattern() == "/public/{slug}") {
					clean += "/"
				}
			}
			target, err := url.ParseRequestURI(clean)
			if err != nil {
				http.Error(w, "invalid request path", 400)
				return
			}
			target.RawQuery = r.URL.RawQuery
			http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
			return
		}
		// Chi matches RawPath when present. Canonicalize segment encoding without
		// turning an encoded slash into a routing separator.
		if r.URL.RawPath != "" {
			segments := strings.Split(r.URL.RawPath, "/")
			for i, segment := range segments {
				decoded, err := url.PathUnescape(segment)
				if err != nil {
					http.Error(w, "invalid request path", 400)
					return
				}
				segments[i] = url.PathEscape(decoded)
			}
			u := *r.URL
			u.RawPath = strings.Join(segments, "/")
			r2 := r.Clone(r.Context())
			r2.URL = &u
			r = r2
		}
		next.ServeHTTP(w, r)
	})
}

func methodNotAllowed(router chi.Router) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed := []string{}
		routePath := r.URL.RawPath
		if routePath == "" {
			routePath = r.URL.Path
		}
		for _, method := range []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "CONNECT", "TRACE"} {
			if router.Match(chi.NewRouteContext(), method, routePath) {
				allowed = append(allowed, method)
			}
		}
		sort.Strings(allowed)
		if len(allowed) > 0 {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
		}
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	})
}
