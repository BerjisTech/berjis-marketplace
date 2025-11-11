CREATE TABLE IF NOT EXISTS discounts (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  name TEXT NOT NULL,
  code TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  discount_type TEXT NOT NULL DEFAULT 'percentage',
  amount_cents BIGINT NOT NULL DEFAULT 0 CHECK (amount_cents >= 0),
  percentage NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (percentage >= 0),
  starts_at TIMESTAMPTZ,
  ends_at TIMESTAMPTZ,
  usage_limit_total INTEGER,
  usage_limit_per_customer INTEGER,
  auto_apply BOOLEAN NOT NULL DEFAULT FALSE,
  status TEXT NOT NULL DEFAULT 'draft',
  applies_to JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (shop_uuid, code)
);

CREATE INDEX IF NOT EXISTS discounts_shop_idx ON discounts(shop_uuid);
CREATE INDEX IF NOT EXISTS discounts_active_idx ON discounts(shop_uuid) WHERE status='active';

CREATE TABLE IF NOT EXISTS discount_products (
  discount_uuid UUID NOT NULL REFERENCES discounts(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (discount_uuid, product_uuid)
);

CREATE INDEX IF NOT EXISTS discount_products_product_idx ON discount_products(product_uuid);
