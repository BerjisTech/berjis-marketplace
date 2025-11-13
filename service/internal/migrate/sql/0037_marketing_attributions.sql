CREATE TABLE IF NOT EXISTS marketing_attributions (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  user_uuid UUID NOT NULL,
  order_uuid UUID REFERENCES orders(uuid) ON DELETE SET NULL,
  source TEXT,
  medium TEXT,
  campaign TEXT,
  term TEXT,
  content TEXT,
  referrer TEXT,
  landing_page TEXT,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS marketing_attributions_user_idx ON marketing_attributions(user_uuid, created_at DESC);
CREATE INDEX IF NOT EXISTS marketing_attributions_order_idx ON marketing_attributions(order_uuid);
