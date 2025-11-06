-- Enable uuid extension if available
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Shops owned by users (Core API user UUIDs recorded here)
CREATE TABLE IF NOT EXISTS shops (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  owner_uuid UUID NOT NULL,
  description TEXT DEFAULT '',
  public BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS shops_owner_idx ON shops(owner_uuid);

-- Simple product model (variants can come later)
CREATE TABLE IF NOT EXISTS products (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  title TEXT NOT NULL,
  slug TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
  currency TEXT NOT NULL DEFAULT 'USD',
  stock BIGINT NOT NULL DEFAULT 0,
  image_url TEXT NULL,
  published BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (shop_uuid, slug)
);
CREATE INDEX IF NOT EXISTS products_shop_idx ON products(shop_uuid);
CREATE INDEX IF NOT EXISTS products_published_idx ON products(published);

