CREATE TABLE IF NOT EXISTS shipping_zones (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  name TEXT NOT NULL,
  countries TEXT[] NOT NULL DEFAULT '{}',
  is_rest_of_world BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS shipping_zones_shop_idx ON shipping_zones(shop_uuid);

CREATE TABLE IF NOT EXISTS shipping_rates (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  zone_uuid UUID NOT NULL REFERENCES shipping_zones(uuid) ON DELETE CASCADE,
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  name TEXT NOT NULL,
  price_cents BIGINT NOT NULL DEFAULT 0,
  min_weight_grams INTEGER,
  max_weight_grams INTEGER,
  min_order_cents BIGINT,
  max_order_cents BIGINT,
  free_above_cents BIGINT,
  estimated_days_min INTEGER,
  estimated_days_max INTEGER,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS shipping_rates_zone_idx ON shipping_rates(zone_uuid);
CREATE INDEX IF NOT EXISTS shipping_rates_shop_idx ON shipping_rates(shop_uuid);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_rate_uuid UUID REFERENCES shipping_rates(uuid);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_address JSONB;
