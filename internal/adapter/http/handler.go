package httpadapter

import (
	"apimock/internal/config"
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"apimock/internal/web"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
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
	mux := http.NewServeMux()
	h.identityRoutes(mux)
	h.workspaceRoutes(mux)
	workspaceRoute := func(pattern string, handler http.HandlerFunc) { mux.HandleFunc(pattern, h.workspaceGuard(handler)) }
	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		respond(w, map[string]string{"public_base_url": h.Config.PublicBaseURL}, nil)
	})
	mux.Handle("GET /assets/", h.Assets)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/app", http.StatusSeeOther) })
	mux.HandleFunc("GET /app", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := web.Page().Render(r.Context(), w); err != nil {
			slog.Error("render page", "error", err)
		}
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if h.Ready != nil {
			if err := h.Ready(r); err != nil {
				http.Error(w, "database unavailable", 503)
				return
			}
		}
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /app/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<span class="status-dot"></span> Server connected`))
	})
	workspaceRoute("GET /api/workspaces", func(w http.ResponseWriter, r *http.Request) {
		if h.Auth != nil {
			items, err := h.Service.WorkspaceRepository().VisibleWorkspaces(r.Context(), principal(r).User)
			respond(w, items, err)
			return
		}
		items, err := h.Service.Repo.Workspaces(r.Context())
		respond(w, items, err)
	})
	workspaceRoute("POST /api/workspaces", func(w http.ResponseWriter, r *http.Request) {
		var item domain.Workspace
		if !decode(w, r, &item) {
			return
		}
		item.ID = ""
		if h.Auth != nil {
			saved, err := h.Service.CreateWorkspace(r.Context(), item, principal(r).User)
			respond(w, saved, err)
			return
		}
		saved, err := h.Service.SaveWorkspace(r.Context(), item)
		respond(w, saved, err)
	})
	workspaceRoute("PUT /api/workspaces/{wid}", func(w http.ResponseWriter, r *http.Request) {
		var item domain.Workspace
		if !decode(w, r, &item) {
			return
		}
		item.ID = r.PathValue("wid")
		saved, err := h.Service.SaveWorkspace(r.Context(), item)
		respond(w, saved, err)
	})
	workspaceRoute("DELETE /api/workspaces/{wid}", func(w http.ResponseWriter, r *http.Request) {
		respond(w, map[string]bool{"deleted": true}, h.Service.Repo.DeleteWorkspace(r.Context(), r.PathValue("wid")))
	})
	workspaceRoute("GET /api/workspaces/{wid}/endpoints", func(w http.ResponseWriter, r *http.Request) {
		items, err := h.Service.Repo.Endpoints(r.Context(), r.PathValue("wid"))
		respond(w, items, err)
	})
	save := func(w http.ResponseWriter, r *http.Request) {
		var item domain.Endpoint
		if !decode(w, r, &item) {
			return
		}
		item.WorkspaceID = r.PathValue("wid")
		item.ID = r.PathValue("eid")
		saved, err := h.Service.SaveEndpoint(r.Context(), item)
		respond(w, saved, err)
	}
	workspaceRoute("POST /api/workspaces/{wid}/endpoints", save)
	workspaceRoute("PUT /api/workspaces/{wid}/endpoints/{eid}", save)
	workspaceRoute("DELETE /api/workspaces/{wid}/endpoints/{eid}", func(w http.ResponseWriter, r *http.Request) {
		respond(w, map[string]bool{"deleted": true}, h.Service.Repo.DeleteEndpoint(r.Context(), r.PathValue("wid"), r.PathValue("eid")))
	})
	mux.HandleFunc("POST /api/test", func(w http.ResponseWriter, r *http.Request) {
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
	mock := func(w http.ResponseWriter, r *http.Request) {
		request, err := parseMockRequest(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := h.Service.Execute(r.Context(), r.PathValue("slug"), request)
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
	mux.HandleFunc("/public/{slug}/{path...}", mock)
	return middleware(h.access(mux))
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
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
