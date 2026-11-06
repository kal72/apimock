package httpadapter

import (
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
)

func (h Handler) requireWorkspace(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if role != "viewer" && role != "editor" && role != "owner" {
				respond(w, nil, usecase.ErrWorkspaceAccess)
				return
			}
			wid := routeParam(r, "wid")
			if wid == "" {
				respond(w, nil, usecase.ErrWorkspaceAccess)
				return
			}
			if h.Auth != nil {
				if err := h.Service.WorkspaceAccess(r.Context(), wid, principal(r).User, role); err != nil {
					respond(w, nil, err)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
func (h Handler) memberRoutes(r chi.Router) {
	if h.Auth == nil {
		return
	}
	getHead(r.With(h.requireWorkspace("owner")), "/api/workspaces/{wid}/members", func(w http.ResponseWriter, r *http.Request) {
		items, err := h.Service.WorkspaceRepository().WorkspaceMembers(r.Context(), routeParam(r, "wid"))
		respond(w, items, err)
	})
	r.With(h.requireWorkspace("owner")).Put("/api/workspaces/{wid}/members/{username}", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Role string `json:"role"`
		}
		if !decode(w, r, &input) {
			return
		}
		// Empty roles are reserved for the explicit delete action.
		if input.Role == "" {
			respond(w, nil, validationError{"role is required"})
			return
		}
		err := h.Service.WorkspaceRepository().SetWorkspaceMember(r.Context(), routeParam(r, "wid"), strings.ToLower(strings.TrimSpace(routeParam(r, "username"))), input.Role, principal(r).User)
		respond(w, map[string]bool{"saved": err == nil}, err)
	})
	r.With(h.requireWorkspace("owner")).Delete("/api/workspaces/{wid}/members/{username}", func(w http.ResponseWriter, r *http.Request) {
		err := h.Service.WorkspaceRepository().SetWorkspaceMember(r.Context(), routeParam(r, "wid"), routeParam(r, "username"), "", principal(r).User)
		respond(w, map[string]bool{"deleted": err == nil}, err)
	})
}

func (h Handler) workspaceRoutes(r chi.Router) {
	h.memberRoutes(r)
	getHead(r, "/api/workspaces", func(w http.ResponseWriter, r *http.Request) {
		if h.Auth != nil {
			items, err := h.Service.WorkspaceRepository().VisibleWorkspaces(r.Context(), principal(r).User)
			respond(w, items, err)
			return
		}
		items, err := h.Service.Repo.Workspaces(r.Context())
		respond(w, items, err)
	})
	r.Post("/api/workspaces", func(w http.ResponseWriter, r *http.Request) {
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
	r.With(h.requireWorkspace("owner")).Put("/api/workspaces/{wid}", func(w http.ResponseWriter, r *http.Request) {
		var item domain.Workspace
		if !decode(w, r, &item) {
			return
		}
		item.ID = routeParam(r, "wid")
		saved, err := h.Service.SaveWorkspace(r.Context(), item)
		respond(w, saved, err)
	})
	r.With(h.requireWorkspace("owner")).Delete("/api/workspaces/{wid}", func(w http.ResponseWriter, r *http.Request) {
		respond(w, map[string]bool{"deleted": true}, h.Service.Repo.DeleteWorkspace(r.Context(), routeParam(r, "wid")))
	})
	getHead(r.With(h.requireWorkspace("viewer")), "/api/workspaces/{wid}/endpoints", func(w http.ResponseWriter, r *http.Request) {
		items, err := h.Service.Repo.Endpoints(r.Context(), routeParam(r, "wid"))
		respond(w, items, err)
	})
	save := func(w http.ResponseWriter, r *http.Request) {
		var item domain.Endpoint
		if !decode(w, r, &item) {
			return
		}
		item.WorkspaceID = routeParam(r, "wid")
		item.ID = routeParam(r, "eid")
		saved, err := h.Service.SaveEndpoint(r.Context(), item)
		respond(w, saved, err)
	}
	r.With(h.requireWorkspace("editor")).Post("/api/workspaces/{wid}/endpoints", save)
	r.With(h.requireWorkspace("editor")).Put("/api/workspaces/{wid}/endpoints/{eid}", save)
	r.With(h.requireWorkspace("editor")).Delete("/api/workspaces/{wid}/endpoints/{eid}", func(w http.ResponseWriter, r *http.Request) {
		respond(w, map[string]bool{"deleted": true}, h.Service.Repo.DeleteEndpoint(r.Context(), routeParam(r, "wid"), routeParam(r, "eid")))
	})
}
