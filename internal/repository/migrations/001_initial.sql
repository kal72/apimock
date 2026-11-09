CREATE TABLE IF NOT EXISTS workspaces (
 id text PRIMARY KEY,
 name text NOT NULL,
 slug text NOT NULL UNIQUE,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS endpoints (
 id text PRIMARY KEY,
 workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 name text NOT NULL,
 method text NOT NULL,
 path text NOT NULL,
 flow jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(workspace_id,method,path)
);
CREATE UNIQUE INDEX IF NOT EXISTS endpoints_route_unique
ON endpoints(workspace_id,method,(regexp_replace(path, ':[^/]+', ':param', 'g')));
