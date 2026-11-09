package httpadapter

import (
	"apimock/internal/domain"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (h Handler) environmentRoutes(r chi.Router) {
	if h.Auth == nil {
		return
	}
	getHead(r, "/api/environments", func(w http.ResponseWriter, r *http.Request) {
		items, err := h.Service.EnvironmentRepository().Environments(r.Context(), principal(r).User.ID)
		respond(w, items, err)
	})
	r.Post("/api/environments", func(w http.ResponseWriter, r *http.Request) {
		var item domain.Environment
		if !decode(w, r, &item) {
			return
		}
		item.ID = ""
		saved, err := h.Service.SaveEnvironment(r.Context(), principal(r).User.ID, item)
		respond(w, saved, err)
	})
	r.Put("/api/environments/{environmentID}", func(w http.ResponseWriter, r *http.Request) {
		var item domain.Environment
		if !decode(w, r, &item) {
			return
		}
		item.ID = routeParam(r, "environmentID")
		saved, err := h.Service.SaveEnvironment(r.Context(), principal(r).User.ID, item)
		respond(w, saved, err)
	})
	r.Delete("/api/environments/{environmentID}", func(w http.ResponseWriter, r *http.Request) {
		err := h.Service.EnvironmentRepository().DeleteEnvironment(r.Context(), principal(r).User.ID, routeParam(r, "environmentID"))
		respond(w, map[string]bool{"deleted": err == nil}, err)
	})
	r.Put("/api/environments/{environmentID}/variables/{key}", func(w http.ResponseWriter, r *http.Request) {
		var item domain.EnvironmentVariable
		if !decode(w, r, &item) {
			return
		}
		item.Key = routeParam(r, "key")
		err := h.Service.SaveEnvironmentVariable(r.Context(), principal(r).User.ID, routeParam(r, "environmentID"), item)
		respond(w, map[string]bool{"saved": err == nil}, err)
	})
	r.Delete("/api/environments/{environmentID}/variables/{key}", func(w http.ResponseWriter, r *http.Request) {
		err := h.Service.EnvironmentRepository().DeleteEnvironmentVariable(r.Context(), principal(r).User.ID, routeParam(r, "environmentID"), routeParam(r, "key"))
		respond(w, map[string]bool{"deleted": err == nil}, err)
	})
}
