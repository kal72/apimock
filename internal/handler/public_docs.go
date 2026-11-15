package handler

import (
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"apimock/internal/web"
	"errors"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	"time"
)

func (h Handler) publicDocsRoutes(r chi.Router) {
	getHead(r, "/share/{slug}", h.publicDocsPage)
	getHead(r, "/share/{slug}/spec", h.publicDocsSpec)
	r.Post("/share/{slug}/playground", h.publicDocsPlayground)
}

func (h Handler) publicDocsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	slug := routeParam(r, "slug")
	workspace, endpoints, err := h.Service.PublicWorkspace(r.Context(), slug)
	if errors.Is(err, domain.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		if r.Method != http.MethodHead {
			if renderErr := web.PublicDocsDisabledPage(slug).Render(r.Context(), w); renderErr != nil {
				slog.Error("render disabled public docs page", "error", renderErr)
			}
		}
		return
	}
	if err != nil {
		slog.Error("load public workspace", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	if err := web.PublicDocsPage(workspace, endpoints, h.Config.PublicBaseURL).Render(r.Context(), w); err != nil {
		slog.Error("render public docs page", "error", err)
	}
}

func (h Handler) publicDocsSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	workspace, endpoints, err := h.Service.PublicWorkspace(r.Context(), routeParam(r, "slug"))
	if err != nil {
		respond(w, nil, err)
		return
	}
	respond(w, web.PublicDocsPayload{
		Workspace:     workspace,
		Endpoints:     endpoints,
		PublicBaseURL: h.Config.PublicBaseURL,
	}, nil)
}

func (h Handler) publicDocsPlayground(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if origin := r.Header.Get("Origin"); origin != "" && !h.sameOrigin(r, origin) && !h.Config.AllowsOrigin(origin) {
		respond(w, nil, usecase.ErrForbidden)
		return
	}
	_, endpoints, err := h.Service.PublicWorkspace(r.Context(), routeParam(r, "slug"))
	if err != nil {
		respond(w, nil, err)
		return
	}
	var input struct {
		EndpointID string              `json:"endpoint_id"`
		Headers    map[string]string   `json:"headers"`
		Body       any                 `json:"body"`
		BodyMode   string              `json:"body_mode"`
		RawBody    string              `json:"raw_body"`
		Form       map[string][]string `json:"form"`
		Params     map[string]string   `json:"params"`
		Query      map[string][]string `json:"query"`
	}
	if !decode(w, r, &input) {
		return
	}
	var target *domain.Endpoint
	for i := range endpoints {
		if endpoints[i].ID == input.EndpointID {
			target = &endpoints[i]
			break
		}
	}
	if target == nil {
		respond(w, nil, domain.ErrNotFound)
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
	request.Method = target.Method
	request.Path = target.Path
	request.Headers = headers
	request.Query = input.Query

	start := time.Now()
	result, err := usecase.EvaluateWithEnvironment(r.Context(), target.Flow, request, input.Params, nil)
	if err != nil {
		respond(w, nil, err)
		return
	}
	durationMS := int(time.Since(start).Milliseconds())
	selected := ""
	if len(result.Trace) > 0 {
		selected = result.Trace[len(result.Trace)-1]
	}
	respond(w, struct {
		usecase.Result
		Conditions []usecase.ConditionCheck `json:"conditions"`
		ResponseID string                   `json:"response_id"`
		DurationMS int                      `json:"duration_ms"`
	}{
		Result:     result,
		Conditions: usecase.ExplainConditions(target.Flow, request, input.Params, result.Trace),
		ResponseID: selected,
		DurationMS: durationMS,
	}, nil)
}
