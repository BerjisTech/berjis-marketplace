-- Carts
CREATE TABLE IF NOT EXISTS carts (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_uuid UUID NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS cart_items (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  cart_uuid UUID NOT NULL REFERENCES carts(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  added_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (cart_uuid, product_uuid)
);

-- Orders
CREATE TABLE IF NOT EXISTS orders (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_uuid UUID NOT NULL,
  total_cents BIGINT NOT NULL CHECK (total_cents >= 0),
  currency TEXT NOT NULL DEFAULT 'USD',
  status TEXT NOT NULL DEFAULT 'pending',
  shipping_address TEXT,
  payment_method TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_items (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  order_uuid UUID NOT NULL REFERENCES orders(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid),
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  price_cents BIGINT NOT NULL CHECK (price_cents >= 0)
);

