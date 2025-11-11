CREATE TABLE IF NOT EXISTS product_variants (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  sku TEXT NOT NULL,
  title TEXT NOT NULL,
  option_values JSONB,
  price_cents BIGINT NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
  compare_at_cents BIGINT,
  stock BIGINT NOT NULL DEFAULT 0 CHECK (stock >= 0),
  barcode TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (product_uuid, sku)
);

CREATE INDEX IF NOT EXISTS product_variants_product_idx ON product_variants(product_uuid);
