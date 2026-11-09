CREATE TABLE user_environments (
 id text PRIMARY KEY,
 user_id text NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
 name text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(user_id, name)
);

CREATE TABLE environment_variables (
 environment_id text NOT NULL REFERENCES user_environments(id) ON DELETE CASCADE,
 key text NOT NULL,
 kind text NOT NULL CHECK (kind IN ('string', 'number', 'boolean', 'secret')),
 value jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(environment_id, key)
);

CREATE INDEX user_environments_user_id_name_idx ON user_environments(user_id, name);
