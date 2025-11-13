CREATE TABLE IF NOT EXISTS order_fulfillments (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  order_uuid UUID NOT NULL REFERENCES orders(uuid) ON DELETE CASCADE,
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  location_uuid UUID REFERENCES inventory_locations(uuid) ON DELETE SET NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  tracking_number TEXT,
  tracking_url TEXT,
  shipping_carrier TEXT,
  label_url TEXT,
  label_data JSONB,
  label_generated_at TIMESTAMPTZ,
  notes TEXT,
  shipped_at TIMESTAMPTZ,
  delivered_at TIMESTAMPTZ,
  cancelled_at TIMESTAMPTZ,
  created_by UUID,
  updated_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS order_fulfillments_order_idx ON order_fulfillments(order_uuid);
CREATE INDEX IF NOT EXISTS order_fulfillments_shop_idx ON order_fulfillments(shop_uuid);
CREATE INDEX IF NOT EXISTS order_fulfillments_status_idx ON order_fulfillments(status);

CREATE TABLE IF NOT EXISTS order_fulfillment_items (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  fulfillment_uuid UUID NOT NULL REFERENCES order_fulfillments(uuid) ON DELETE CASCADE,
  order_item_uuid UUID NOT NULL REFERENCES order_items(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS order_fulfillment_items_fulfillment_idx ON order_fulfillment_items(fulfillment_uuid);
CREATE INDEX IF NOT EXISTS order_fulfillment_items_order_item_idx ON order_fulfillment_items(order_item_uuid);
CREATE INDEX IF NOT EXISTS order_fulfillment_items_product_idx ON order_fulfillment_items(product_uuid);
