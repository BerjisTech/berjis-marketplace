ALTER TABLE discounts
  ADD COLUMN IF NOT EXISTS minimum_subtotal_cents BIGINT NOT NULL DEFAULT 0 CHECK (minimum_subtotal_cents >= 0),
  ADD COLUMN IF NOT EXISTS free_shipping BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS buy_quantity INTEGER,
  ADD COLUMN IF NOT EXISTS get_quantity INTEGER,
  ADD COLUMN IF NOT EXISTS get_percentage NUMERIC(5,2) NOT NULL DEFAULT 100 CHECK (get_percentage >= 0);

CREATE INDEX IF NOT EXISTS discounts_free_shipping_idx ON discounts(shop_uuid) WHERE free_shipping = true;
