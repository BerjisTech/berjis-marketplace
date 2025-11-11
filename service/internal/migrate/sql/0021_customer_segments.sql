CREATE TABLE IF NOT EXISTS customer_segments (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  rules JSONB,
  is_dynamic BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (shop_uuid, name)
);

CREATE TABLE IF NOT EXISTS customer_segment_members (
  segment_uuid UUID NOT NULL REFERENCES customer_segments(uuid) ON DELETE CASCADE,
  customer_uuid UUID NOT NULL REFERENCES customers(uuid) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (segment_uuid, customer_uuid)
);

CREATE INDEX IF NOT EXISTS customer_segments_shop_idx ON customer_segments(shop_uuid);
CREATE INDEX IF NOT EXISTS customer_segment_members_customer_idx ON customer_segment_members(customer_uuid);
