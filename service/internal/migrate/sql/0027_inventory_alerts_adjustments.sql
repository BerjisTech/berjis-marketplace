CREATE TABLE IF NOT EXISTS inventory_adjustments (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  inventory_level_uuid UUID NOT NULL REFERENCES inventory_levels(uuid) ON DELETE CASCADE,
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  location_uuid UUID REFERENCES inventory_locations(uuid) ON DELETE SET NULL,
  user_uuid UUID,
  delta_quantity BIGINT NOT NULL DEFAULT 0,
  delta_reserved BIGINT NOT NULL DEFAULT 0,
  resulting_quantity BIGINT NOT NULL,
  resulting_reserved BIGINT NOT NULL,
  reason TEXT NOT NULL DEFAULT 'manual',
  note TEXT NOT NULL DEFAULT '',
  adjustment_source TEXT NOT NULL DEFAULT 'manual',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS inventory_adjustments_level_idx ON inventory_adjustments(inventory_level_uuid);
CREATE INDEX IF NOT EXISTS inventory_adjustments_shop_created_idx ON inventory_adjustments(shop_uuid, created_at DESC);

CREATE TABLE IF NOT EXISTS inventory_alerts (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  inventory_level_uuid UUID NOT NULL REFERENCES inventory_levels(uuid) ON DELETE CASCADE,
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  location_uuid UUID REFERENCES inventory_locations(uuid) ON DELETE SET NULL,
  quantity BIGINT NOT NULL,
  safety_stock BIGINT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  triggered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  resolved_at TIMESTAMPTZ,
  note TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS inventory_alerts_level_idx ON inventory_alerts(inventory_level_uuid);
CREATE INDEX IF NOT EXISTS inventory_alerts_shop_status_idx ON inventory_alerts(shop_uuid, status);
CREATE UNIQUE INDEX IF NOT EXISTS inventory_alerts_open_unique_idx
  ON inventory_alerts(inventory_level_uuid)
  WHERE status = 'open';
