package postgres

import (
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
)

func (r Repository) VisibleWorkspaces(ctx context.Context, actor domain.User) ([]domain.Workspace, error) {
	rows, err := r.Pool.Query(ctx, `SELECT w.id,w.name,w.slug,CASE WHEN $2='admin' THEN 'owner' ELSE m.role END FROM workspaces w LEFT JOIN workspace_members m ON m.workspace_id=w.id AND m.user_id=$1 WHERE $2='admin' OR m.user_id IS NOT NULL ORDER BY w.name,w.id`, actor.ID, actor.Role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Workspace{}
	for rows.Next() {
		var w domain.Workspace
		if err = rows.Scan(&w.ID, &w.Name, &w.Slug, &w.Role); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
func (r Repository) WorkspaceRole(ctx context.Context, wid string, actor domain.User) (string, error) {
	var role string
	err := r.Pool.QueryRow(ctx, `SELECT CASE WHEN $3='admin' THEN 'owner' ELSE COALESCE((SELECT role FROM workspace_members WHERE workspace_id=w.id AND user_id=$2),'') END FROM workspaces w WHERE w.id=$1`, wid, actor.ID, actor.Role).Scan(&role)
	if err == pgx.ErrNoRows {
		return "", domain.ErrNotFound
	}
	return role, err
}
func (r Repository) CreateOwnedWorkspace(ctx context.Context, w domain.Workspace, uid string) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO workspaces(id,name,slug) VALUES($1,$2,$3)`, w.ID, w.Name, w.Slug); err != nil {
		return dbError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO workspace_members VALUES($1,$2,'owner')`, w.ID, uid); err != nil {
		return dbError(err)
	}
	return tx.Commit(ctx)
}
func (r Repository) WorkspaceMembers(ctx context.Context, wid string) ([]domain.WorkspaceMember, error) {
	rows, err := r.Pool.Query(ctx, `SELECT u.id,u.username,u.name,m.role,u.active FROM workspace_members m JOIN app_users u ON u.id=m.user_id WHERE m.workspace_id=$1 ORDER BY m.role,u.username`, wid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.WorkspaceMember{}
	for rows.Next() {
		var m domain.WorkspaceMember
		if err = rows.Scan(&m.UserID, &m.Username, &m.Name, &m.Role, &m.Active); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (r Repository) SetWorkspaceMember(ctx context.Context, wid, username, role string, actor domain.User) error {
	if role != "" && role != "viewer" && role != "editor" && role != "owner" {
		return fmt.Errorf("%w: invalid workspace role", domain.ErrValidation)
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Serialize membership changes with account deletion and deactivation.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(67822002)`); err != nil {
		return err
	}
	var allowed bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app_users u WHERE u.id=$2 AND u.active AND (u.role='admin' OR EXISTS(SELECT 1 FROM workspace_members m WHERE m.workspace_id=$1 AND m.user_id=u.id AND m.role='owner')))`, wid, actor.ID).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return usecase.ErrWorkspaceAccess
	}
	var uid string
	if err = tx.QueryRow(ctx, `SELECT id FROM app_users WHERE username=$1 AND (active OR $2='')`, username, role).Scan(&uid); err == pgx.ErrNoRows {
		return domain.ErrNotFound
	} else if err != nil {
		return err
	}
	var last bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND role='owner') AND (SELECT count(*) FROM workspace_members m JOIN app_users u ON u.id=m.user_id WHERE m.workspace_id=$1 AND m.role='owner' AND u.active)<=1`, wid, uid).Scan(&last); err != nil {
		return err
	}
	if last && role != "owner" {
		return fmt.Errorf("%w: add another active owner before removing the last owner", domain.ErrValidation)
	}
	if role == "" {
		_, err = tx.Exec(ctx, `DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wid, uid)
	} else {
		_, err = tx.Exec(ctx, `INSERT INTO workspace_members VALUES($1,$2,$3) ON CONFLICT(workspace_id,user_id) DO UPDATE SET role=EXCLUDED.role`, wid, uid, role)
	}
	if err != nil {
		return dbError(err)
	}
	return tx.Commit(ctx)
}
func protectWorkspaceOwner(ctx context.Context, tx pgx.Tx, uid string) error {
	var last bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members m WHERE m.user_id=$1 AND m.role='owner' AND NOT EXISTS(SELECT 1 FROM workspace_members o JOIN app_users u ON u.id=o.user_id WHERE o.workspace_id=m.workspace_id AND o.role='owner' AND o.user_id<>$1 AND u.active))`, uid).Scan(&last)
	if err != nil {
		return err
	}
	if last {
		return fmt.Errorf("%w: assign another active workspace owner before deleting or deactivating this user", domain.ErrValidation)
	}
	return nil
}
