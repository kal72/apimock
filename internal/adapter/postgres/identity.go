package postgres

import (
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

const userColumns = "id,username,name,role,active,password_hash,created_at"

func scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Username, &u.Name, &u.Role, &u.Active, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = domain.ErrNotFound
	}
	return u, err
}
func (r Repository) HasUsers(ctx context.Context) (bool, error) {
	var exists bool
	err := r.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM app_users)").Scan(&exists)
	return exists, err
}
func (r Repository) CreateFirstUser(ctx context.Context, u domain.User) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(67822002)"); err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM app_users)").Scan(&exists); err != nil {
		return err
	}
	if exists {
		return usecase.ErrSetupClosed
	}
	_, err = tx.Exec(ctx, "INSERT INTO app_users(id,username,name,role,active,password_hash) VALUES($1,$2,$3,'admin',true,$4)", u.ID, u.Username, u.Name, u.PasswordHash)
	if err != nil {
		return dbError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO workspace_members SELECT w.id,$1,'owner' FROM workspaces w WHERE NOT EXISTS(SELECT 1 FROM workspace_members m WHERE m.workspace_id=w.id)`, u.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r Repository) Users(ctx context.Context) ([]domain.User, error) {
	rows, err := r.Pool.Query(ctx, "SELECT "+userColumns+" FROM app_users ORDER BY username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
func (r Repository) UserByUsername(ctx context.Context, name string) (domain.User, error) {
	return scanUser(r.Pool.QueryRow(ctx, "SELECT "+userColumns+" FROM app_users WHERE username=$1", name))
}
func lockAdmin(ctx context.Context, tx pgx.Tx, actor string) error {
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(67822002)"); err != nil {
		return err
	}
	var allowed bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM app_users WHERE id=$1 AND role='admin' AND active)", actor).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return usecase.ErrForbidden
	}
	return nil
}
func protectLastAdmin(ctx context.Context, tx pgx.Tx, id string, removing bool) error {
	if !removing {
		return nil
	}
	var last bool
	err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM app_users WHERE id=$1 AND active AND role='admin') AND (SELECT count(*) FROM app_users WHERE active AND role='admin')<=1", id).Scan(&last)
	if err != nil {
		return err
	}
	if last {
		return usecase.ErrLastAdmin
	}
	return nil
}
func (r Repository) SaveUser(ctx context.Context, u domain.User, actor string, create bool) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockAdmin(ctx, tx, actor); err != nil {
		return err
	}
	if create {
		_, err = tx.Exec(ctx, "INSERT INTO app_users(id,username,name,role,active,password_hash) VALUES($1,$2,$3,$4,$5,$6)", u.ID, u.Username, u.Name, u.Role, u.Active, u.PasswordHash)
	} else {
		old, readErr := scanUser(tx.QueryRow(ctx, "SELECT "+userColumns+" FROM app_users WHERE id=$1 FOR UPDATE", u.ID))
		if readErr != nil {
			return readErr
		}
		if err = protectLastAdmin(ctx, tx, u.ID, u.Role != "admin" || !u.Active); err != nil {
			return err
		}
		if actor == u.ID && (u.Role != "admin" || !u.Active) {
			return usecase.ErrLastAdmin
		}
		if !u.Active {
			if err = protectWorkspaceOwner(ctx, tx, u.ID); err != nil {
				return err
			}
		}
		hash := u.PasswordHash
		if hash == "" {
			hash = old.PasswordHash
		}
		_, err = tx.Exec(ctx, "UPDATE app_users SET username=$2,name=$3,role=$4,active=$5,password_hash=$6 WHERE id=$1", u.ID, u.Username, u.Name, u.Role, u.Active, hash)
		if err == nil && (hash != old.PasswordHash || u.Role != old.Role || u.Active != old.Active || u.Username != old.Username) {
			_, err = tx.Exec(ctx, "DELETE FROM app_sessions WHERE user_id=$1", u.ID)
		}
	}
	if err != nil {
		return dbError(err)
	}
	return tx.Commit(ctx)
}
func (r Repository) DeleteUser(ctx context.Context, id, actor string) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockAdmin(ctx, tx, actor); err != nil {
		return err
	}
	if err = protectLastAdmin(ctx, tx, id, true); err != nil {
		return err
	}
	if actor == id {
		return usecase.ErrLastAdmin
	}
	if err = protectWorkspaceOwner(ctx, tx, id); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, "DELETE FROM app_users WHERE id=$1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit(ctx)
}
func (r Repository) CreateSession(ctx context.Context, hash, id, passwordHash string, expires time.Time) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	u, err := scanUser(tx.QueryRow(ctx, "SELECT "+userColumns+" FROM app_users WHERE id=$1 FOR UPDATE", id))
	if err != nil {
		return err
	}
	if !u.Active || u.PasswordHash != passwordHash {
		return usecase.ErrCredentials
	}
	if _, err = tx.Exec(ctx, "DELETE FROM app_sessions WHERE expires_at<=now()"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO app_sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)", hash, id, expires); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r Repository) SessionUser(ctx context.Context, hash string) (domain.User, error) {
	return scanUser(r.Pool.QueryRow(ctx, "SELECT u.id,u.username,u.name,u.role,u.active,u.password_hash,u.created_at FROM app_users u JOIN app_sessions s ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now() AND u.active", hash))
}
func (r Repository) DeleteSession(ctx context.Context, hash string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM app_sessions WHERE token_hash=$1", hash)
	return err
}
func (r Repository) ChangePassword(ctx context.Context, id, old, next string) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, "UPDATE app_users SET password_hash=$3 WHERE id=$1 AND password_hash=$2 AND active", id, old, next)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return usecase.ErrCredentials
	}
	if _, err = tx.Exec(ctx, "DELETE FROM app_sessions WHERE user_id=$1", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
