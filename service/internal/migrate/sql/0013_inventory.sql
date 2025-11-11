CREATE TABLE IF NOT EXISTS inventory_locations (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  name TEXT NOT NULL,
  code TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  is_primary BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (shop_uuid, code)
);

CREATE INDEX IF NOT EXISTS inventory_locations_shop_idx ON inventory_locations(shop_uuid);
CREATE INDEX IF NOT EXISTS inventory_locations_primary_idx ON inventory_locations(shop_uuid) WHERE is_primary;

CREATE TABLE IF NOT EXISTS inventory_levels (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  location_uuid UUID REFERENCES inventory_locations(uuid) ON DELETE SET NULL,
  quantity BIGINT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
  reserved BIGINT NOT NULL DEFAULT 0 CHECK (reserved >= 0),
  safety_stock BIGINT NOT NULL DEFAULT 0 CHECK (safety_stock >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS inventory_levels_shop_idx ON inventory_levels(shop_uuid);
CREATE INDEX IF NOT EXISTS inventory_levels_product_idx ON inventory_levels(product_uuid);
CREATE UNIQUE INDEX IF NOT EXISTS inventory_levels_product_default_idx ON inventory_levels(product_uuid) WHERE location_uuid IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS inventory_levels_product_location_idx ON inventory_levels(product_uuid, location_uuid) WHERE location_uuid IS NOT NULL;
