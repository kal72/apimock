package httpadapter

import (
	"apimock/internal/adapter/postgres"
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWorkspaceIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for workspace integration tests")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	schema := "workspace_test_" + usecase.ID()
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
	repo := postgres.Repository{Pool: pool}
	if err = repo.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	users := map[string]domain.User{}
	tokens := map[string]string{}
	for i, name := range []string{"admin", "owner", "viewer", "editor", "outsider"} {
		role := "user"
		if name == "admin" {
			role = "admin"
		}
		u := domain.User{ID: usecase.ID(), Username: name, Name: name, Role: role, Active: true}
		users[name] = u
		if _, err = pool.Exec(ctx, `INSERT INTO app_users(id,username,name,role,password_hash) VALUES($1,$2,$2,$3,'test-hash')`, u.ID, name, role); err != nil {
			t.Fatal(err)
		}
		token := strings.Repeat(fmt.Sprint(i), 43)
		tokens[name] = token
		if err = repo.CreateSession(ctx, usecase.TokenHash(token), u.ID, "test-hash", time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	h := Handler{Auth: &Identity{Service: usecase.AuthService{Repo: repo}}, Service: usecase.Service{Repo: repo}, Assets: http.NotFoundHandler()}.Routes()
	call := func(actor, method, path string, body any, status int) *httptest.ResponseRecorder {
		t.Helper()
		data, _ := json.Marshal(body)
		req := httptest.NewRequest(method, "http://mock.test"+path, strings.NewReader(string(data)))
		if actor != "" {
			req.AddCookie(&http.Cookie{Name: sessionCookie, Value: tokens[actor]})
			req.Header.Set("X-CSRF-Token", usecase.CSRFToken(tokens[actor]))
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("%s %s as %s: got %d want %d: %s", method, path, actor, w.Code, status, w.Body.String())
		}
		return w
	}
	w := call("owner", "POST", "/api/workspaces", map[string]string{"name": "Private", "slug": "private"}, 200)
	var ws domain.Workspace
	json.Unmarshal(w.Body.Bytes(), &ws)
	base := "/api/workspaces/" + ws.ID
	call("owner", "PUT", base+"/members/viewer", map[string]string{"role": "viewer"}, 200)
	call("owner", "PUT", base+"/members/editor", map[string]string{"role": "editor"}, 200)
	for _, actor := range []string{"owner", "viewer", "editor", "admin"} {
		call(actor, "GET", base+"/endpoints", nil, 200)
	}
	call("outsider", "GET", base+"/endpoints", nil, 403)
	listed := call("outsider", "GET", "/api/workspaces", nil, 200)
	if listed.Body.String() != "[]\n" {
		t.Fatal("workspace leaked to outsider")
	}
	endpoint := domain.Endpoint{Name: "Test", Method: "GET", Path: "/ok", Flow: domain.DefaultFlow()}
	for _, actor := range []string{"viewer", "outsider"} {
		call(actor, "POST", base+"/endpoints", endpoint, 403)
	}
	w = call("editor", "POST", base+"/endpoints", endpoint, 200)
	var ep domain.Endpoint
	json.Unmarshal(w.Body.Bytes(), &ep)
	for _, actor := range []string{"viewer", "outsider"} {
		call(actor, "PUT", base+"/endpoints/"+ep.ID, ep, 403)
		call(actor, "DELETE", base+"/endpoints/"+ep.ID, nil, 403)
	}
	call("editor", "PUT", base+"/endpoints/"+ep.ID, ep, 200)
	for _, actor := range []string{"viewer", "editor", "outsider"} {
		call(actor, "GET", base+"/members", nil, 403)
		call(actor, "PUT", base, map[string]string{"name": "Denied", "slug": "denied"}, 403)
		call(actor, "DELETE", base, nil, 403)
		call(actor, "PUT", base+"/members/outsider", map[string]string{"role": "owner"}, 403)
	}
	call("owner", "PUT", base+"/members/owner", map[string]string{"role": "viewer"}, 400)
	call("admin", "DELETE", base+"/members/owner", nil, 400)
	if err = repo.DeleteUser(ctx, users["owner"].ID, users["admin"].ID); err == nil {
		t.Fatal("deleted last workspace owner")
	}
	inactive := users["owner"]
	inactive.Active = false
	if err = repo.SaveUser(ctx, inactive, users["admin"].ID, false); err == nil {
		t.Fatal("deactivated last workspace owner")
	}
	call("owner", "PUT", base+"/members/missing", map[string]string{"role": "viewer"}, 404)
	call("owner", "PUT", base+"/members/viewer", map[string]string{"role": "invalid"}, 400)
	call("owner", "PUT", base+"/members/viewer", map[string]string{"role": ""}, 400)
	call("", "GET", "/public/private/ok", nil, 200)
	call("viewer", "POST", "/api/test", map[string]any{"flow": domain.DefaultFlow()}, 200)
	call("owner", "DELETE", base+"/members/viewer", nil, 200)
	call("viewer", "GET", base+"/endpoints", nil, 403)
	// Concurrent owner changes cannot remove both owners.
	call("owner", "PUT", base+"/members/editor", map[string]string{"role": "owner"}, 200)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, name := range []string{"owner", "editor"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			results <- repo.SetWorkspaceMember(ctx, ws.ID, name, "viewer", users["admin"])
		}(name)
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected one owner demotion, got %d", successes)
	}
	call("admin", "DELETE", base, nil, 200)
	call("admin", "GET", base+"/endpoints", nil, 404)
	// Upgrade legacy workspaces assigns the earliest active administrator.
	legacy := domain.Workspace{ID: usecase.ID(), Name: "Legacy", Slug: "legacy"}
	if err = repo.SaveWorkspace(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `DROP TABLE workspace_members; DELETE FROM schema_migrations WHERE version='003_workspace_members.sql'`); err != nil {
		t.Fatal(err)
	}
	if err = repo.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	members, err := repo.WorkspaceMembers(ctx, legacy.ID)
	if err != nil || len(members) != 1 || members[0].UserID != users["admin"].ID || members[0].Role != "owner" {
		t.Fatalf("legacy ownership: %+v %v", members, err)
	}
	if err = repo.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
}
