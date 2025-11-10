CREATE TABLE IF NOT EXISTS store_invitations (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  store_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  email TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'staff',
  token TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'pending',
  invited_by UUID NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '7 days'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  accepted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS store_invitations_store_idx ON store_invitations(store_uuid);
CREATE INDEX IF NOT EXISTS store_invitations_token_idx ON store_invitations(token);
