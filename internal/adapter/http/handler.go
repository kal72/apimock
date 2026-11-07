package httpadapter

import (
	"apimock/internal/config"
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"apimock/internal/web"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	Auth    *Identity
	Config  config.Config
	Service usecase.Service
	Assets  http.Handler
	Ready   func(*http.Request) error
}

func (h Handler) Routes() http.Handler {
	r := chi.NewRouter()
	getHead(r, "/healthz", h.health)
	r.Group(func(management chi.Router) {
		management.Use(h.managementAccess)
		h.publicIdentityRoutes(management)
		getHead(management, "/assets/*", h.Assets.ServeHTTP)
		getHead(management, "/assets", func(w http.ResponseWriter, r *http.Request) {
			u := *r.URL
			u.Path = "/assets/"
			http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
		})
		management.Group(func(protected chi.Router) {
			protected.Use(h.requireSession)
			h.privateIdentityRoutes(protected)
			h.pageRoutes(protected)
			h.workspaceRoutes(protected)
			h.playgroundRoutes(protected)
		})
	})
	r.Route("/public", func(public chi.Router) {
		public.Use(h.mockCORS)
		public.HandleFunc("/{slug}", func(w http.ResponseWriter, r *http.Request) {
			u := *r.URL
			u.Path += "/"
			if u.RawPath != "" {
				u.RawPath += "/"
			}
			http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
		})
		public.HandleFunc("/{slug}/*", h.mock)
		public.Handle("/*", http.NotFoundHandler())
	})
	r.NotFound(h.routingFallback(http.NotFoundHandler()).ServeHTTP)
	r.MethodNotAllowed(h.routingFallback(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/public/") {
			http.NotFound(w, request)
			return
		}
		methodNotAllowed(r).ServeHTTP(w, request)
	})).ServeHTTP)
	return middleware(h.access(canonicalPath(r)))
}
func (h Handler) health(w http.ResponseWriter, r *http.Request) {
	if h.Ready != nil {
		if err := h.Ready(r); err != nil {
			http.Error(w, "database unavailable", 503)
			return
		}
	}
	w.Write([]byte("ok"))
}
func (h Handler) pageRoutes(r chi.Router) {
	for _, page := range []string{"endpoints", "environments", "mocks", "logs", "settings"} {
		getHead(r, "/app/v2/"+page, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := web.V2Page(page).Render(r.Context(), w); err != nil {
				slog.Error("render v2 mockup", "error", err)
			}
		})
	}
	getHead(r, "/app/v2", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/app/v2/endpoints", http.StatusSeeOther)
	})

	getHead(r, "/api/config", func(w http.ResponseWriter, r *http.Request) {
		respond(w, map[string]string{"public_base_url": h.Config.PublicBaseURL}, nil)
	})
	getHead(r, "/", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/app", http.StatusSeeOther) })
	getHead(r, "/app", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := web.Page().Render(r.Context(), w); err != nil {
			slog.Error("render page", "error", err)
		}
	})
	getHead(r, "/docs/generators", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := web.GeneratorDocumentation().Render(r.Context(), w); err != nil {
			slog.Error("render generator documentation", "error", err)
		}
	})
	getHead(r, "/app/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<span class="status-dot"></span> Server connected`))
	})
}
func (h Handler) playgroundRoutes(r chi.Router) {
	r.Post("/api/test", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Flow     domain.Flow         `json:"flow"`
			Method   string              `json:"method"`
			Path     string              `json:"path"`
			Headers  map[string]string   `json:"headers"`
			Body     any                 `json:"body"`
			BodyMode string              `json:"body_mode"`
			RawBody  string              `json:"raw_body"`
			Form     map[string][]string `json:"form"`
			Params   map[string]string   `json:"params"`
			Query    map[string][]string `json:"query"`
		}
		if !decode(w, r, &input) {
			return
		}
		headers := http.Header{}
		for k, v := range input.Headers {
			headers.Set(k, v)
		}
		request, err := playgroundRequest(input.Body, input.BodyMode, input.RawBody, input.Form)
		if err != nil {
			respond(w, nil, validationError{err.Error()})
			return
		}
		request.Method = input.Method
		request.Path = input.Path
		request.Headers = headers
		request.Query = input.Query
		result, err := usecase.Evaluate(r.Context(), input.Flow, request, input.Params)
		if err != nil {
			respond(w, nil, err)
			return
		}
		selected := ""
		if len(result.Trace) > 0 {
			selected = result.Trace[len(result.Trace)-1]
		}
		respond(w, struct {
			usecase.Result
			Conditions []usecase.ConditionCheck `json:"conditions"`
			ResponseID string                   `json:"response_id"`
		}{result, usecase.ExplainConditions(input.Flow, request, input.Params, result.Trace), selected}, nil)
	})
}
func (h Handler) mock(w http.ResponseWriter, r *http.Request) {

	request, err := parseMockRequest(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := h.Service.Execute(r.Context(), routeParam(r, "slug"), request)
	if errors.Is(err, usecase.ErrResponseRender) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"Mock response could not be rendered; check its template in Playground"}`))
		return
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	for k, v := range result.Headers {
		w.Header().Set(k, v)
	}
	if r.Header.Get("Origin") != "" {
		w.Header().Add("Vary", "Origin")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(result.Status)
	if r.Method != "HEAD" {
		w.Write([]byte(result.Body))
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	d.UseNumber()
	if err := d.Decode(v); err != nil {
		respond(w, nil, validationError{"invalid JSON: " + err.Error()})
		return false
	}
	if err := d.Decode(new(any)); err != io.EOF {
		respond(w, nil, validationError{"request must contain exactly one JSON value"})
		return false
	}
	return true
}

type validationError struct{ message string }

func (e validationError) Error() string { return e.message }
func respond(w http.ResponseWriter, v any, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		status := 400
		switch {
		case errors.Is(err, usecase.ErrResponseRender):
			status = 422
		case errors.Is(err, usecase.ErrUnauthorized), errors.Is(err, usecase.ErrCredentials):
			status = 401
		case errors.Is(err, usecase.ErrWorkspaceAccess), errors.Is(err, usecase.ErrForbidden), errors.Is(err, errCSRF):
			status = 403
		case errors.Is(err, usecase.ErrSetupClosed), errors.Is(err, usecase.ErrLastAdmin):
			status = 409
		case errors.Is(err, errRateLimited):
			status = 429
		case errors.Is(err, domain.ErrNotFound):
			status = 404
		case errors.Is(err, domain.ErrConflict):
			status = 409
		default:
			if !isValidation(err) {
				status = 500
				slog.Error("request failed", "error", err)
			}
		}
		w.WriteHeader(status)
		message := err.Error()
		if status == 500 {
			message = "Internal server error"
		}
		json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}
	json.NewEncoder(w).Encode(v)
}

// Validation is mapped explicitly at use-case boundaries by the wrapper below.
func isValidation(err error) bool {
	var v validationError
	return errors.As(err, &v) || errors.Is(err, domain.ErrValidation)
}
func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		start := time.Now()
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("panic", "error", recovered)
				http.Error(w, "internal server error", 500)
			}
			slog.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
