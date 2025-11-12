ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS draft_source_uuid UUID;

CREATE TABLE IF NOT EXISTS order_drafts (
  uuid UUID PRIMARY KEY,
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  customer_uuid UUID REFERENCES customers(uuid) ON DELETE SET NULL,
  customer_email TEXT,
  customer_name TEXT,
  currency TEXT NOT NULL,
  subtotal_cents BIGINT NOT NULL DEFAULT 0,
  discount_code TEXT,
  discount_amount_cents BIGINT NOT NULL DEFAULT 0,
  gift_card_code TEXT,
  gift_card_amount_cents BIGINT NOT NULL DEFAULT 0,
  shipping_address TEXT,
  payment_method TEXT,
  notes TEXT,
  status TEXT NOT NULL DEFAULT 'open',
  expires_at TIMESTAMPTZ,
  created_by UUID,
  updated_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_draft_items (
  uuid UUID PRIMARY KEY,
  draft_uuid UUID NOT NULL REFERENCES order_drafts(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  variant_uuid UUID REFERENCES product_variants(uuid) ON DELETE SET NULL,
  sku TEXT,
  quantity INT NOT NULL CHECK (quantity > 0),
  price_cents BIGINT NOT NULL,
  currency TEXT NOT NULL,
  line_total_cents BIGINT NOT NULL DEFAULT 0,
  title TEXT,
  variant_title TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS order_drafts_shop_idx ON order_drafts(shop_uuid);
CREATE INDEX IF NOT EXISTS order_draft_items_draft_idx ON order_draft_items(draft_uuid);
