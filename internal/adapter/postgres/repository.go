package postgres

import (
	"apimock/internal/domain"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Repository struct{ Pool *pgxpool.Pool }

func (r Repository) Migrate(ctx context.Context) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(67822001)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, file := range files {
		var applied bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, file.Name()).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		sql, readErr := migrations.ReadFile("migrations/" + file.Name())
		if readErr != nil {
			return readErr
		}
		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, file.Name()); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func dbError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return domain.ErrConflict
		case "23503":
			return domain.ErrNotFound
		}
	}
	return err
}
func (r Repository) Workspaces(ctx context.Context) ([]domain.Workspace, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id,name,slug FROM workspaces ORDER BY name,id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Workspace{}
	for rows.Next() {
		var w domain.Workspace
		if err = rows.Scan(&w.ID, &w.Name, &w.Slug); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
func (r Repository) SaveWorkspace(ctx context.Context, w domain.Workspace) error {
	_, err := r.Pool.Exec(ctx, `INSERT INTO workspaces(id,name,slug) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,slug=EXCLUDED.slug`, w.ID, w.Name, w.Slug)
	return dbError(err)
}
func (r Repository) DeleteWorkspace(ctx context.Context, id string) error {
	tag, err := r.Pool.Exec(ctx, "DELETE FROM workspaces WHERE id=$1", id)
	if err == nil && tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return err
}
func (r Repository) Endpoints(ctx context.Context, wid string) ([]domain.Endpoint, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id,workspace_id,name,method,path,flow FROM endpoints WHERE workspace_id=$1 ORDER BY path,method", wid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Endpoint{}
	for rows.Next() {
		var e domain.Endpoint
		var flow []byte
		if err = rows.Scan(&e.ID, &e.WorkspaceID, &e.Name, &e.Method, &e.Path, &flow); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(flow, &e.Flow); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func (r Repository) SaveEndpoint(ctx context.Context, e domain.Endpoint) error {
	flow, err := json.Marshal(e.Flow)
	if err != nil {
		return err
	}
	tag, err := r.Pool.Exec(ctx, `INSERT INTO endpoints(id,workspace_id,name,method,path,flow) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,method=EXCLUDED.method,path=EXCLUDED.path,flow=EXCLUDED.flow WHERE endpoints.workspace_id=EXCLUDED.workspace_id`, e.ID, e.WorkspaceID, e.Name, e.Method, e.Path, flow)
	if err == nil && tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return dbError(err)
}
func (r Repository) DeleteEndpoint(ctx context.Context, wid, id string) error {
	tag, err := r.Pool.Exec(ctx, "DELETE FROM endpoints WHERE workspace_id=$1 AND id=$2", wid, id)
	if err == nil && tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return err
}
