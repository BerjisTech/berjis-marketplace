CREATE TABLE IF NOT EXISTS collections (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  title TEXT NOT NULL,
  slug TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  is_automatic BOOLEAN NOT NULL DEFAULT FALSE,
  rules JSONB,
  sort_order INTEGER NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (shop_uuid, slug)
);

CREATE INDEX IF NOT EXISTS collections_shop_idx ON collections(shop_uuid);
CREATE INDEX IF NOT EXISTS collections_active_idx ON collections(shop_uuid) WHERE is_active;

CREATE TABLE IF NOT EXISTS collection_products (
  collection_uuid UUID NOT NULL REFERENCES collections(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (collection_uuid, product_uuid)
);

CREATE INDEX IF NOT EXISTS collection_products_product_idx ON collection_products(product_uuid);
