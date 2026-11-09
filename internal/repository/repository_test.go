package repository

import (
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
)

func TestRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for PostgreSQL integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	r := Repository{pool}
	if err = r.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	w := domain.Workspace{ID: usecase.ID(), Name: "Integration", Slug: "test-" + usecase.ID()}
	if err = r.SaveWorkspace(ctx, w); err != nil {
		t.Fatal(err)
	}
	defer r.DeleteWorkspace(ctx, w.ID)
	e := domain.Endpoint{ID: usecase.ID(), WorkspaceID: w.ID, Name: "Product", Method: "GET", Path: "/products/:id", Flow: domain.DefaultFlow()}
	e.Flow.Nodes[0].Config.Example = &domain.RequestExample{Params: map[string]string{"id": "123"}, Query: map[string][]string{"page": {"2"}}, Headers: map[string]string{"X-Api-Key": "demo"}, Body: map[string]any{"active": true}}
	e.Flow.Nodes[1].Config.ResponseMode = "dynamic"
	e.Flow.Nodes[1].Config.ResponseSeed = "integration"
	e.Flow.Nodes[1].Config.Body = `{"id":"{{ request.path.id }}"}`
	if err = r.SaveEndpoint(ctx, e); err != nil {
		t.Fatal(err)
	}
	items, err := r.Endpoints(ctx, w.ID)
	if err != nil || len(items) != 1 || items[0].Flow.Nodes[1].Config.Status != 200 {
		t.Fatalf("roundtrip: %+v %v", items, err)
	}
	if items[0].Flow.Nodes[1].Config.ResponseMode != "dynamic" || items[0].Flow.Nodes[1].Config.ResponseSeed != "integration" {
		t.Fatal("dynamic settings did not persist")
	}
	example := items[0].Flow.Nodes[0].Config.Example
	if example == nil || example.Params["id"] != "123" || example.Query["page"][0] != "2" || example.Headers["X-Api-Key"] != "demo" || example.Body["active"] != true {
		t.Fatalf("request example did not persist: %+v", example)
	}
	e.ID = usecase.ID()
	if err = r.SaveEndpoint(ctx, e); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected duplicate conflict, got %v", err)
	}
	if err = r.DeleteWorkspace(ctx, w.ID); err != nil {
		t.Fatal(err)
	}
	items, err = r.Endpoints(ctx, w.ID)
	if err != nil || len(items) != 0 {
		t.Fatalf("cascade: %+v %v", items, err)
	}
}
