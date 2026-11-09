package handler

import (
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"apimock/internal/web"
	"context"
	"crypto/subtle"
	"errors"
	"github.com/go-chi/chi/v5"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const sessionCookie = "mocklab_session"

type identityKey struct{}
type identity struct {
	User  domain.User
	Token string
}
type loginAttempt struct {
	Count int
	Until time.Time
}
type Identity struct {
	Service  usecase.AuthService
	mu       sync.Mutex
	attempts map[string]loginAttempt
}

func principal(r *http.Request) identity {
	value, _ := r.Context().Value(identityKey{}).(identity)
	return value
}
func (a *Identity) limited(r *http.Request, username string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	if a.attempts == nil {
		a.attempts = map[string]loginAttempt{}
	}
	for key, value := range a.attempts {
		if now.After(value.Until) {
			delete(a.attempts, key)
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	keys := []string{"ip:" + ip, "user:" + usecase.TokenHash(strings.ToLower(username))}
	for _, key := range keys {
		if a.attempts[key].Count >= 20 {
			return true
		}
	}
	if len(a.attempts) > 10000 {
		return true
	}
	for _, key := range keys {
		v := a.attempts[key]
		if v.Count == 0 {
			v.Until = now.Add(15 * time.Minute)
		}
		v.Count++
		a.attempts[key] = v
	}
	return false
}
func (h Handler) identityAccess(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	path := r.URL.Path
	cookie, err := r.Cookie(sessionCookie)
	token := ""
	if err == nil {
		token = cookie.Value
	}
	u, err := h.Auth.Service.Authenticate(r.Context(), token)
	if err != nil {
		if !errors.Is(err, usecase.ErrUnauthorized) {
			respond(w, nil, err)
			return r, false
		}
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/login")
			w.WriteHeader(http.StatusOK)
		} else if strings.HasPrefix(path, "/api/") {
			respond(w, nil, usecase.ErrUnauthorized)
		} else {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
		return r, false
	}
	if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
		want := usecase.CSRFToken(token)
		got := r.Header.Get("X-CSRF-Token")
		if subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
			respond(w, nil, errCSRF)
			return r, false
		}
	}
	return r.WithContext(context.WithValue(r.Context(), identityKey{}, identity{User: u, Token: token})), true
}
func (h Handler) setSession(w http.ResponseWriter, r *http.Request, token string, clear bool) {
	cookie := &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: r.TLS != nil || strings.HasPrefix(h.Config.PublicBaseURL, "https://"), SameSite: http.SameSiteStrictMode, MaxAge: int(usecase.SessionDuration.Seconds())}
	if clear {
		cookie.MaxAge = -1
		cookie.Expires = time.Unix(1, 0)
	} else {
		cookie.Expires = time.Now().Add(usecase.SessionDuration)
	}
	http.SetCookie(w, cookie)
}
func (h Handler) publicIdentityRoutes(r chi.Router) {
	if h.Auth == nil {
		return
	}
	getHead(r, "/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		web.AuthPage().Render(r.Context(), w)
	})
	getHead(r, "/api/auth/status", func(w http.ResponseWriter, r *http.Request) {
		exists, err := h.Auth.Service.Repo.HasUsers(r.Context())
		respond(w, map[string]bool{"setup_required": !exists}, err)
	})
	r.Post("/api/auth/setup", func(w http.ResponseWriter, r *http.Request) {
		if h.Config.ManagedDeployment() {
			respond(w, nil, usecase.ErrSetupClosed)
			return
		}
		var input usecase.UserInput
		if !decode(w, r, &input) {
			return
		}
		if h.Auth.limited(r, input.Username) {
			w.Header().Set("Retry-After", "900")
			respond(w, nil, errRateLimited)
			return
		}
		_, err := h.Auth.Service.Setup(r.Context(), input)
		respond(w, map[string]bool{"created": err == nil}, err)
	})
	r.Post("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if !decode(w, r, &input) {
			return
		}
		if h.Auth.limited(r, input.Username) {
			w.Header().Set("Retry-After", "900")
			respond(w, nil, errRateLimited)
			return
		}
		u, token, err := h.Auth.Service.Login(r.Context(), input.Username, input.Password)
		if err != nil {
			respond(w, nil, err)
			return
		}
		if old, err := r.Cookie(sessionCookie); err == nil {
			if err = h.Auth.Service.Repo.DeleteSession(r.Context(), usecase.TokenHash(old.Value)); err != nil {
				respond(w, nil, err)
				return
			}
		}
		h.setSession(w, r, token, false)
		respond(w, map[string]any{"user": u, "csrf_token": usecase.CSRFToken(token)}, nil)
	})
}
func (h Handler) privateIdentityRoutes(r chi.Router) {
	if h.Auth == nil {
		return
	}
	getHead(r, "/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		p := principal(r)
		respond(w, map[string]any{"user": p.User, "csrf_token": usecase.CSRFToken(p.Token)}, nil)
	})
	r.Post("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		err := h.Auth.Service.Repo.DeleteSession(r.Context(), usecase.TokenHash(principal(r).Token))
		if err == nil {
			h.setSession(w, r, "", true)
		}
		respond(w, map[string]bool{"logged_out": err == nil}, err)
	})
	r.Post("/api/auth/password", func(w http.ResponseWriter, r *http.Request) {
		if h.Auth.limited(r, "password:"+principal(r).User.ID) {
			respond(w, nil, errRateLimited)
			return
		}
		var input struct {
			Current string `json:"current_password"`
			Next    string `json:"new_password"`
		}
		if !decode(w, r, &input) {
			return
		}
		err := h.Auth.Service.ChangePassword(r.Context(), principal(r).User, input.Current, input.Next)
		if err == nil {
			h.setSession(w, r, "", true)
		}
		respond(w, map[string]bool{"changed": err == nil}, err)
	})
	r.Group(func(r chi.Router) {
		r.Use(h.requireAdmin)
		getHead(r, "/api/users", func(w http.ResponseWriter, r *http.Request) {
			users, err := h.Auth.Service.Repo.Users(r.Context())
			respond(w, users, err)
		})
		save := func(w http.ResponseWriter, r *http.Request) {
			var input usecase.UserInput
			if !decode(w, r, &input) {
				return
			}
			u, err := h.Auth.Service.SaveUser(r.Context(), principal(r).User, routeParam(r, "uid"), input)
			respond(w, u, err)
		}
		r.Post("/api/users", save)
		r.Put("/api/users/{uid}", save)
		r.Delete("/api/users/{uid}", func(w http.ResponseWriter, r *http.Request) {
			err := h.Auth.Service.DeleteUser(r.Context(), principal(r).User, routeParam(r, "uid"))
			respond(w, map[string]bool{"deleted": err == nil}, err)
		})
	})
}

var errRateLimited = errors.New("too many sign-in attempts; try again in 15 minutes")

var errCSRF = errors.New("CSRF validation failed; refresh the page and try again")

func (h Handler) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if principal(r).User.Role != "admin" {
			respond(w, nil, usecase.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h Handler) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.Auth == nil {
			next.ServeHTTP(w, r)
			return
		}
		authenticated, ok := h.identityAccess(w, r)
		if !ok {
			return
		}
		next.ServeHTTP(w, authenticated)
	})
}
