package httpadapter

import (
	"net/http"
	"strings"
)

func (h Handler) workspaceGuard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.Auth != nil && r.PathValue("wid") != "" {
			required := "viewer"
			if r.Method != "GET" {
				required = "editor"
				if r.PathValue("eid") == "" && !strings.Contains(r.URL.Path, "/endpoints") {
					required = "owner"
				}
			}
			if strings.Contains(r.URL.Path, "/members") {
				required = "owner"
			}
			if err := h.Service.WorkspaceAccess(r.Context(), r.PathValue("wid"), principal(r).User, required); err != nil {
				respond(w, nil, err)
				return
			}
		}
		next(w, r)
	}
}
func (h Handler) workspaceRoutes(mux *http.ServeMux) {
	if h.Auth == nil {
		return
	}
	mux.HandleFunc("GET /api/workspaces/{wid}/members", h.workspaceGuard(func(w http.ResponseWriter, r *http.Request) {
		items, err := h.Service.WorkspaceRepository().WorkspaceMembers(r.Context(), r.PathValue("wid"))
		respond(w, items, err)
	}))
	mux.HandleFunc("PUT /api/workspaces/{wid}/members/{username}", h.workspaceGuard(func(w http.ResponseWriter, r *http.Request) {
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
		err := h.Service.WorkspaceRepository().SetWorkspaceMember(r.Context(), r.PathValue("wid"), strings.ToLower(strings.TrimSpace(r.PathValue("username"))), input.Role, principal(r).User)
		respond(w, map[string]bool{"saved": err == nil}, err)
	}))
	mux.HandleFunc("DELETE /api/workspaces/{wid}/members/{username}", h.workspaceGuard(func(w http.ResponseWriter, r *http.Request) {
		err := h.Service.WorkspaceRepository().SetWorkspaceMember(r.Context(), r.PathValue("wid"), r.PathValue("username"), "", principal(r).User)
		respond(w, map[string]bool{"deleted": err == nil}, err)
	}))
}
