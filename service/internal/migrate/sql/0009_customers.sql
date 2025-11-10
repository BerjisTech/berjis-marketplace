CREATE TABLE IF NOT EXISTS customers (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  user_uuid UUID NULL,
  email TEXT NOT NULL,
  first_name TEXT NOT NULL DEFAULT '',
  last_name TEXT NOT NULL DEFAULT '',
  phone TEXT NOT NULL DEFAULT '',
  tags TEXT[] NOT NULL DEFAULT '{}',
  notes TEXT NOT NULL DEFAULT '',
  marketing_opt_in BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE customers
  ADD CONSTRAINT customers_shop_email_unique UNIQUE (shop_uuid, email);

CREATE INDEX IF NOT EXISTS customers_shop_idx ON customers(shop_uuid);
CREATE INDEX IF NOT EXISTS customers_user_idx ON customers(user_uuid);
CREATE INDEX IF NOT EXISTS customers_email_idx ON customers(LOWER(email));
CREATE INDEX IF NOT EXISTS customers_tags_idx ON customers USING GIN (tags);
