CREATE TABLE IF NOT EXISTS marketing_campaigns (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  name TEXT NOT NULL,
  channel TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  budget_cents BIGINT NOT NULL DEFAULT 0 CHECK (budget_cents >= 0),
  spend_cents BIGINT NOT NULL DEFAULT 0 CHECK (spend_cents >= 0),
  starts_at TIMESTAMPTZ,
  ends_at TIMESTAMPTZ,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (shop_uuid, name)
);

CREATE INDEX IF NOT EXISTS marketing_campaigns_shop_idx ON marketing_campaigns(shop_uuid);
CREATE INDEX IF NOT EXISTS marketing_campaigns_status_idx ON marketing_campaigns(shop_uuid, status);
