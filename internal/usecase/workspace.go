package usecase

import (
	"apimock/internal/domain"
	"context"
	"fmt"
)

type WorkspaceRepository interface {
	VisibleWorkspaces(context.Context, domain.User) ([]domain.Workspace, error)
	WorkspaceRole(context.Context, string, domain.User) (string, error)
	CreateOwnedWorkspace(context.Context, domain.Workspace, string) error
	WorkspaceMembers(context.Context, string) ([]domain.WorkspaceMember, error)
	SetWorkspaceMember(context.Context, string, string, string, domain.User) error
}

func (s Service) WorkspaceRepository() WorkspaceRepository { return s.Repo.(WorkspaceRepository) }
func (s Service) WorkspaceAccess(ctx context.Context, wid string, actor domain.User, required string) error {
	role, err := s.WorkspaceRepository().WorkspaceRole(ctx, wid, actor)
	if err != nil {
		return err
	}
	rank := map[string]int{"viewer": 1, "editor": 2, "owner": 3}
	if rank[role] < rank[required] {
		return fmt.Errorf("%w: %s workspace access required", ErrWorkspaceAccess, required)
	}
	return nil
}

var ErrWorkspaceAccess = fmt.Errorf("workspace access denied")

func (s Service) CreateWorkspace(ctx context.Context, w domain.Workspace, actor domain.User) (domain.Workspace, error) {
	if err := w.Validate(); err != nil {
		return w, fmt.Errorf("%w: %s", domain.ErrValidation, err)
	}
	w.ID = ID()
	w.Role = "owner"
	return w, s.WorkspaceRepository().CreateOwnedWorkspace(ctx, w, actor.ID)
}
