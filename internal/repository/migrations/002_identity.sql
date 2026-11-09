CREATE TABLE app_users (
 id text PRIMARY KEY,
 username text UNIQUE NOT NULL CHECK (username=lower(username)),
 name text NOT NULL,
 role text NOT NULL CHECK(role IN ('admin','user')),
 active boolean NOT NULL DEFAULT true,
 password_hash text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app_sessions (
 token_hash text PRIMARY KEY,
 user_id text NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
 expires_at timestamptz NOT NULL
);
CREATE INDEX app_sessions_user_idx ON app_sessions(user_id);
CREATE INDEX app_sessions_expiry_idx ON app_sessions(expires_at);
