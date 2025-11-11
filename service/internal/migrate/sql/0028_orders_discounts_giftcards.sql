ALTER TABLE orders ADD COLUMN IF NOT EXISTS shop_uuid UUID REFERENCES shops(uuid);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS subtotal_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS discount_uuid UUID REFERENCES discounts(uuid);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS discount_code TEXT;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS discount_amount_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS gift_card_uuid UUID REFERENCES gift_cards(uuid);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS gift_card_code TEXT;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS gift_card_amount_cents BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS discount_redemptions (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  discount_uuid UUID NOT NULL REFERENCES discounts(uuid) ON DELETE CASCADE,
  order_uuid UUID NOT NULL REFERENCES orders(uuid) ON DELETE CASCADE,
  user_uuid UUID NOT NULL,
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  amount_cents BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS discount_redemptions_discount_idx ON discount_redemptions(discount_uuid);
CREATE INDEX IF NOT EXISTS discount_redemptions_user_idx ON discount_redemptions(user_uuid);
