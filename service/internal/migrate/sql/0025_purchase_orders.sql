CREATE TABLE IF NOT EXISTS suppliers (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  name TEXT NOT NULL,
  contact_email TEXT,
  phone TEXT,
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (shop_uuid, name)
);

CREATE TABLE IF NOT EXISTS purchase_orders (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  supplier_uuid UUID REFERENCES suppliers(uuid) ON DELETE SET NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  expected_at TIMESTAMPTZ,
  notes TEXT NOT NULL DEFAULT '',
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS purchase_order_items (
  purchase_order_uuid UUID NOT NULL REFERENCES purchase_orders(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  quantity BIGINT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
  cost_cents BIGINT NOT NULL DEFAULT 0 CHECK (cost_cents >= 0),
  received_quantity BIGINT NOT NULL DEFAULT 0 CHECK (received_quantity >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (purchase_order_uuid, product_uuid)
);

CREATE INDEX IF NOT EXISTS suppliers_shop_idx ON suppliers(shop_uuid);
CREATE INDEX IF NOT EXISTS purchase_orders_shop_idx ON purchase_orders(shop_uuid);
CREATE INDEX IF NOT EXISTS purchase_order_items_product_idx ON purchase_order_items(product_uuid);
