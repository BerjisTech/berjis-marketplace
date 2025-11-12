CREATE TABLE IF NOT EXISTS cart_recoveries (
  uuid UUID PRIMARY KEY,
  cart_uuid UUID NOT NULL REFERENCES carts(uuid) ON DELETE CASCADE,
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  user_uuid UUID NOT NULL,
  customer_uuid UUID REFERENCES customers(uuid) ON DELETE SET NULL,
  customer_email TEXT,
  token TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'pending',
  recovery_url TEXT,
  sent_at TIMESTAMPTZ,
  clicked_at TIMESTAMPTZ,
  converted_order_uuid UUID REFERENCES orders(uuid) ON DELETE SET NULL,
  expires_at TIMESTAMPTZ,
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS cart_recoveries_cart_idx ON cart_recoveries(cart_uuid);
CREATE INDEX IF NOT EXISTS cart_recoveries_shop_status_idx ON cart_recoveries(shop_uuid, status);

CREATE TABLE IF NOT EXISTS order_returns (
  uuid UUID PRIMARY KEY,
  order_uuid UUID NOT NULL REFERENCES orders(uuid) ON DELETE CASCADE,
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  customer_uuid UUID REFERENCES customers(uuid) ON DELETE SET NULL,
  status TEXT NOT NULL DEFAULT 'requested',
  reason TEXT,
  notes TEXT,
  requested_by UUID,
  processed_by UUID,
  restock BOOL NOT NULL DEFAULT false,
  restocked_at TIMESTAMPTZ,
  refund_amount_cents BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_return_items (
  uuid UUID PRIMARY KEY,
  return_uuid UUID NOT NULL REFERENCES order_returns(uuid) ON DELETE CASCADE,
  order_item_uuid UUID NOT NULL REFERENCES order_items(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  quantity INT NOT NULL CHECK (quantity > 0),
  reason TEXT,
  condition TEXT,
  restocked_quantity INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS order_returns_shop_idx ON order_returns(shop_uuid);
CREATE INDEX IF NOT EXISTS order_returns_order_idx ON order_returns(order_uuid);
CREATE INDEX IF NOT EXISTS order_return_items_return_idx ON order_return_items(return_uuid);
