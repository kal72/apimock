CREATE TABLE app_api_keys (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    name text NOT NULL,
    key_hash text UNIQUE NOT NULL,
    prefix text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz
);
CREATE INDEX app_api_keys_user_idx ON app_api_keys(user_id);
CREATE INDEX app_api_keys_hash_idx ON app_api_keys(key_hash);
