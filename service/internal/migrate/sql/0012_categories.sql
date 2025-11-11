CREATE TABLE IF NOT EXISTS categories (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  parent_uuid UUID REFERENCES categories(uuid) ON DELETE SET NULL,
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  sort_order INTEGER NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (shop_uuid, slug)
);

CREATE INDEX IF NOT EXISTS categories_shop_idx ON categories(shop_uuid);
CREATE INDEX IF NOT EXISTS categories_parent_idx ON categories(parent_uuid);
CREATE INDEX IF NOT EXISTS categories_active_idx ON categories(shop_uuid, is_active);

ALTER TABLE products
  ADD COLUMN IF NOT EXISTS category_uuid UUID REFERENCES categories(uuid) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS products_category_uuid_idx ON products(category_uuid);

CREATE TABLE IF NOT EXISTS category_products (
  category_uuid UUID NOT NULL REFERENCES categories(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  is_primary BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (category_uuid, product_uuid)
);

CREATE INDEX IF NOT EXISTS category_products_product_idx ON category_products(product_uuid);
