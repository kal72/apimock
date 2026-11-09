package handler

import (
	"apimock/internal/config"
	"apimock/internal/domain"
	"apimock/internal/repository"
	"apimock/internal/usecase"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestIdentityIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for identity integration tests")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	schema := "identity_test_" + usecase.ID()
	if _, err = root.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer root.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo := repository.Repository{Pool: pool}
	if err = repo.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	auth := usecase.AuthService{Repo: repo}
	h := Handler{Auth: &Identity{Service: auth}, Config: config.Config{}, Service: usecase.Service{Repo: repo}, Assets: http.NotFoundHandler()}.Routes()
	call := func(method, path, body string, cookie *http.Cookie, csrf string) (*httptest.ResponseRecorder, map[string]any) {
		req := httptest.NewRequest(method, "http://mock.test"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		data := map[string]any{}
		json.Unmarshal(w.Body.Bytes(), &data)
		return w, data
	}
	expect := func(w *httptest.ResponseRecorder, status int) {
		t.Helper()
		if w.Code != status {
			t.Fatalf("got %d want %d: %s", w.Code, status, w.Body.String())
		}
	}
	hxRequest := httptest.NewRequest("GET", "http://mock.test/app/status", nil)
	hxRequest.Header.Set("HX-Request", "true")
	hxResponse := httptest.NewRecorder()
	h.ServeHTTP(hxResponse, hxRequest)
	if hxResponse.Header().Get("HX-Redirect") != "/login" {
		t.Fatal("expired HTMX requests must navigate to login")
	}
	w, _ := call("GET", "/docs/generators", "", nil, "")
	expect(w, 303)
	w, _ = call("GET", "/app", "", nil, "")
	expect(w, 303)
	w, _ = call("GET", "/api/workspaces", "", nil, "")
	expect(w, 401)
	setup := `{"username":"admin","name":"Administrator","password":"test-admin-password-2026"}`
	req := httptest.NewRequest("POST", "http://mock.test/api/auth/setup", strings.NewReader(setup))
	req.Header.Set("Origin", "https://evil.example")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	expect(w, 403)
	w, _ = call("POST", "/api/auth/setup", setup, nil, "")
	expect(w, 200)
	w, _ = call("POST", "/api/auth/setup", setup, nil, "")
	expect(w, 409)
	w, _ = call("POST", "/api/auth/login", `{"username":"admin","password":"wrong"}`, nil, "")
	expect(w, 401)
	login := func(username, password string) (*http.Cookie, string) {
		t.Helper()
		body, _ := json.Marshal(map[string]string{"username": username, "password": password})
		w, data := call("POST", "/api/auth/login", string(body), nil, "")
		expect(w, 200)
		cookie := w.Result().Cookies()[0]
		if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.MaxAge <= 0 {
			t.Fatal("unsafe session cookie")
		}
		if strings.Contains(w.Body.String(), "password_hash") {
			t.Fatal("hash leaked")
		}
		return cookie, data["csrf_token"].(string)
	}
	if err = auth.Bootstrap(ctx, "admin", "must-not-reset-existing-password"); err != nil {
		t.Fatal(err)
	}
	admin, csrf := login("admin", "test-admin-password-2026")
	w, me := call("GET", "/api/auth/me", "", admin, "")
	expect(w, 200)
	adminID := me["user"].(map[string]any)["id"].(string)
	userInput := `{"username":"member","name":"Member","role":"user","active":true,"password":"member-password-2026"}`
	w, _ = call("POST", "/api/users", userInput, admin, "")
	expect(w, 403)
	w, data := call("POST", "/api/users", userInput, admin, csrf)
	expect(w, 200)
	uid := data["id"].(string)
	member, memberCSRF := login("member", "member-password-2026")
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		path := "/api/users"
		if method == "PUT" || method == "DELETE" {
			path += "/" + adminID
		}
		w, _ = call(method, path, userInput, member, memberCSRF)
		expect(w, 403)
	}
	w, _ = call("GET", "/api/workspaces", "", member, "")
	expect(w, 200)
	w, _ = call("PUT", "/api/users/"+adminID, `{"username":"admin","name":"Admin","role":"user","active":true}`, admin, csrf)
	expect(w, 400)
	if err = repo.DeleteUser(ctx, adminID, adminID); err != usecase.ErrLastAdmin {
		t.Fatalf("last admin protection: %v", err)
	}
	w, _ = call("PUT", "/api/users/"+uid, `{"username":"member","name":"Member","role":"admin","active":true}`, admin, csrf)
	expect(w, 200)
	w, _ = call("GET", "/api/auth/me", "", member, "")
	expect(w, 401)
	promoted, promotedCSRF := login("member", "member-password-2026")
	w, _ = call("GET", "/api/users", "", promoted, "")
	expect(w, 200)
	w, _ = call("POST", "/api/auth/logout", `{}`, promoted, promotedCSRF)
	expect(w, 200)
	w, _ = call("GET", "/api/auth/me", "", promoted, "")
	expect(w, 401)
	w, _ = call("PUT", "/api/users/"+uid, `{"username":"member","name":"Member","role":"user","active":false}`, admin, csrf)
	expect(w, 200)
	w, _ = call("POST", "/api/auth/login", `{"username":"member","password":"member-password-2026"}`, nil, "")
	expect(w, 401)
	// Expired sessions are rejected even when the browser retains its cookie.
	_, err = pool.Exec(ctx, "UPDATE app_sessions SET expires_at=$1 WHERE user_id=$2", time.Now().Add(-time.Hour), adminID)
	if err != nil {
		t.Fatal(err)
	}
	w, _ = call("GET", "/api/auth/me", "", admin, "")
	expect(w, 401)
	admin, csrf = login("admin", "test-admin-password-2026")
	oldAdmin, err := repo.UserByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	w, _ = call("POST", "/api/auth/password", `{"current_password":"test-admin-password-2026","new_password":"new-admin-password-2026"}`, admin, csrf)
	expect(w, 200)
	w, _ = call("GET", "/api/auth/me", "", admin, "")
	expect(w, 401)
	if err = repo.CreateSession(ctx, "stale-login-token", adminID, oldAdmin.PasswordHash, time.Now().Add(time.Hour)); err != usecase.ErrCredentials {
		t.Fatalf("stale login survived password change: %v", err)
	}
	admin, csrf = login("admin", "new-admin-password-2026")
	w, _ = call("DELETE", "/api/users/"+uid, "", admin, csrf)
	expect(w, 200)
	// Public mock endpoints stay independent of the management session.
	svc := usecase.Service{Repo: repo}
	workspace, err := svc.SaveWorkspace(ctx, domain.Workspace{Name: "Public", Slug: "public-auth-test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.SaveEndpoint(ctx, domain.Endpoint{WorkspaceID: workspace.ID, Name: "Demo", Method: "GET", Path: "/ok", Flow: domain.DefaultFlow()})
	if err != nil {
		t.Fatal(err)
	}
	w, _ = call("GET", "/public/public-auth-test/ok", "", nil, "")
	expect(w, 200)
	body, _ := io.ReadAll(w.Result().Body)
	if len(body) == 0 {
		t.Fatal("mock body missing")
	}
	// Concurrent administrators cannot demote each other into a zero-admin state.
	actorA, err := repo.UserByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	actorB, err := auth.SaveUser(ctx, actorA, "", usecase.UserInput{Username: "second-admin", Name: "Second admin", Role: "admin", Active: true, Password: "second-admin-password-2026"})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := auth.SaveUser(ctx, actorA, actorB.ID, usecase.UserInput{Username: actorB.Username, Name: actorB.Name, Role: "user", Active: true})
		results <- err
	}()
	go func() {
		defer wg.Done()
		_, err := auth.SaveUser(ctx, actorB, actorA.ID, usecase.UserInput{Username: actorA.Username, Name: actorA.Name, Role: "user", Active: true})
		results <- err
	}()
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected one serialized role change, got %d", successes)
	}
	users, err := repo.Users(ctx)
	if err != nil {
		t.Fatal(err)
	}
	admins := 0
	for _, u := range users {
		if u.Active && u.Role == "admin" {
			admins++
		}
	}
	if admins != 1 {
		t.Fatalf("lost last administrator: %d", admins)
	}

}
