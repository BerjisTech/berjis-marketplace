CREATE TABLE IF NOT EXISTS store_users (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  store_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  user_uuid UUID NOT NULL,
  role TEXT NOT NULL DEFAULT 'staff',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (store_uuid, user_uuid)
);

CREATE INDEX IF NOT EXISTS store_users_store_idx ON store_users(store_uuid);
CREATE INDEX IF NOT EXISTS store_users_user_idx ON store_users(user_uuid);
