CREATE TABLE IF NOT EXISTS media_assets (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID REFERENCES shops(uuid) ON DELETE SET NULL,
  owner_uuid UUID,
  filename TEXT NOT NULL,
  path TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  size_bytes BIGINT NOT NULL DEFAULT 0,
  checksum TEXT,
  metadata JSONB,
  visibility TEXT NOT NULL DEFAULT 'private',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS media_assets_shop_idx ON media_assets(shop_uuid);
CREATE INDEX IF NOT EXISTS media_assets_owner_idx ON media_assets(owner_uuid);
CREATE INDEX IF NOT EXISTS media_assets_visibility_idx ON media_assets(visibility);
