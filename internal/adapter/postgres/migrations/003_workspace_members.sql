CREATE TABLE workspace_members (
 workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 user_id text NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
 role text NOT NULL CHECK(role IN ('viewer','editor','owner')),
 PRIMARY KEY(workspace_id,user_id)
);
CREATE INDEX workspace_members_user_idx ON workspace_members(user_id);
INSERT INTO workspace_members(workspace_id,user_id,role)
SELECT w.id,u.id,'owner' FROM workspaces w CROSS JOIN LATERAL
 (SELECT id FROM app_users WHERE role='admin' AND active ORDER BY created_at,id LIMIT 1) u;
