CREATE TABLE IF NOT EXISTS transfers (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  source_location_uuid UUID REFERENCES inventory_locations(uuid) ON DELETE SET NULL,
  destination_location_uuid UUID REFERENCES inventory_locations(uuid) ON DELETE SET NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  notes TEXT NOT NULL DEFAULT '',
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS transfer_items (
  transfer_uuid UUID NOT NULL REFERENCES transfers(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  quantity BIGINT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (transfer_uuid, product_uuid)
);

CREATE INDEX IF NOT EXISTS transfers_shop_idx ON transfers(shop_uuid);
