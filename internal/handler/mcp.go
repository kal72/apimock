package handler

import (
	"apimock/internal/mcp"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func (h Handler) getMCPServer() *mcp.Server {
	if h.MCPServer != nil {
		return h.MCPServer
	}
	return mcp.NewServer(h.Service, h.Config)
}

func (h Handler) mcpRoutes(r chi.Router) {
	srv := h.getMCPServer()
	r.Route("/api/mcp", func(m chi.Router) {
		m.Options("/*", func(w http.ResponseWriter, r *http.Request) {
			mcpCORS(w, r)
		})
		m.Get("/sse", h.mcpSSE(srv))
		m.Post("/messages", h.mcpMessages(srv))
	})
}

func mcpCORS(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Api-Key, Cache-Control")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

func (h Handler) authenticateMCP(r *http.Request) bool {
	token := extractMCPToken(r)

	// 1. User-generated MCP API Key or active user session token via Auth service
	if token != "" && h.Auth != nil {
		if _, err := h.Auth.Service.AuthenticateAPIKey(r.Context(), token); err == nil {
			return true
		}
		if _, err := h.Auth.Service.Authenticate(r.Context(), token); err == nil {
			return true
		}
	}

	// 2. Frictionless development fallback (only in development when no token is provided)
	if h.Config.Environment == "development" && token == "" {
		return true
	}

	return false
}

func extractMCPToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	if key := r.Header.Get("X-Api-Key"); key != "" {
		return strings.TrimSpace(key)
	}
	if token := r.URL.Query().Get("token"); token != "" {
		return strings.TrimSpace(token)
	}
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

func writeSSE(w io.Writer, event string, data string) {
	if event != "" {
		fmt.Fprintf(w, "event: %s\n", event)
	}
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	fmt.Fprintf(w, "\n")
}

func (h Handler) mcpSSE(srv *mcp.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mcpCORS(w, r)

		if !h.authenticateMCP(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "MCP authentication required; provide Authorization: Bearer <key> or ?token=<key>",
			})
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		sess := srv.CreateSession()
		defer srv.CloseSession(sess.ID)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		endpointURI := fmt.Sprintf("/api/mcp/messages?sessionId=%s", sess.ID)
		if token := extractMCPToken(r); token != "" {
			endpointURI += fmt.Sprintf("&token=%s", url.QueryEscape(token))
		}

		writeSSE(w, "endpoint", endpointURI)
		flusher.Flush()

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
			case msg, ok := <-sess.Messages:
				if !ok {
					return
				}
				writeSSE(w, "message", string(msg))
				flusher.Flush()
			}
		}
	}
}

func (h Handler) mcpMessages(srv *mcp.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mcpCORS(w, r)

		if !h.authenticateMCP(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "MCP authentication required",
			})
			return
		}

		sessionID := r.URL.Query().Get("sessionId")
		if sessionID == "" {
			http.Error(w, "Missing sessionId query parameter", http.StatusBadRequest)
			return
		}

		sess, ok := srv.GetSession(sessionID)
		if !ok {
			http.Error(w, "Session not found or expired", http.StatusNotFound)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		responses, err := srv.HandleMessage(r.Context(), body)
		if err != nil {
			slog.Error("mcp handle message error", "error", err)
		}

		for _, resp := range responses {
			b, err := json.Marshal(resp)
			if err == nil {
				sess.Send(b)
			}
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
