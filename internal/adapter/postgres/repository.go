package postgres

import (
	"apimock/internal/domain"
	"bytes"
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

func (r Repository) Environments(ctx context.Context, userID string) ([]domain.Environment, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id,name FROM user_environments WHERE user_id=$1 ORDER BY name,id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Environment{}
	for rows.Next() {
		var item domain.Environment
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		variableRows, err := r.Pool.Query(ctx, `SELECT key,kind,value FROM environment_variables WHERE environment_id=$1 ORDER BY key`, item.ID)
		if err != nil {
			return nil, err
		}
		for variableRows.Next() {
			var variable domain.EnvironmentVariable
			if err := variableRows.Scan(&variable.Key, &variable.Kind, &variable.Value); err != nil {
				variableRows.Close()
				return nil, err
			}
			if variable.Kind == "secret" {
				variable.Value = nil
				variable.HasValue = true
			}
			item.Variables = append(item.Variables, variable)
		}
		if err := variableRows.Err(); err != nil {
			variableRows.Close()
			return nil, err
		}
		variableRows.Close()
		out = append(out, item)
	}
	return out, rows.Err()
}
func (r Repository) SaveEnvironment(ctx context.Context, userID string, item domain.Environment) error {
	if item.ID == "" {
		return errors.New("environment ID is required")
	}
	tag, err := r.Pool.Exec(ctx, `INSERT INTO user_environments(id,user_id,name) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,updated_at=now() WHERE user_environments.user_id=EXCLUDED.user_id`, item.ID, userID, item.Name)
	if err == nil && tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return dbError(err)
}
func (r Repository) DeleteEnvironment(ctx context.Context, userID, id string) error {
	tag, err := r.Pool.Exec(ctx, `DELETE FROM user_environments WHERE id=$1 AND user_id=$2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return dbError(err)
}
func (r Repository) SaveEnvironmentVariable(ctx context.Context, userID, environmentID string, variable domain.EnvironmentVariable) error {
	var exists bool
	if err := r.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_environments WHERE id=$1 AND user_id=$2)`, environmentID, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return domain.ErrNotFound
	}
	_, err := r.Pool.Exec(ctx, `INSERT INTO environment_variables(environment_id,key,kind,value) VALUES($1,$2,$3,$4) ON CONFLICT(environment_id,key) DO UPDATE SET kind=EXCLUDED.kind,value=EXCLUDED.value,updated_at=now()`, environmentID, variable.Key, variable.Kind, []byte(variable.Value))
	return dbError(err)
}
func (r Repository) DeleteEnvironmentVariable(ctx context.Context, userID, environmentID, key string) error {
	tag, err := r.Pool.Exec(ctx, `DELETE FROM environment_variables v USING user_environments e WHERE v.environment_id=e.id AND e.id=$1 AND e.user_id=$2 AND v.key=$3`, environmentID, userID, key)
	if err == nil && tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return dbError(err)
}
func (r Repository) EnvironmentValues(ctx context.Context, userID, environmentID string) (map[string]any, error) {
	var exists bool
	if err := r.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_environments WHERE id=$1 AND user_id=$2)`, environmentID, userID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrNotFound
	}
	rows, err := r.Pool.Query(ctx, `SELECT key,value FROM environment_variables WHERE environment_id=$1`, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := map[string]any{}
	for rows.Next() {
		var key string
		var raw []byte
		if err := rows.Scan(&key, &raw); err != nil {
			return nil, err
		}
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		values[key] = value
	}
	return values, rows.Err()
}
