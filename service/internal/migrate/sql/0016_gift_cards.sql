CREATE TABLE IF NOT EXISTS gift_cards (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  code TEXT NOT NULL,
  balance_cents BIGINT NOT NULL DEFAULT 0 CHECK (balance_cents >= 0),
  original_balance_cents BIGINT NOT NULL DEFAULT 0 CHECK (original_balance_cents >= 0),
  currency TEXT NOT NULL DEFAULT 'USD',
  issued_to_email TEXT,
  note TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'draft',
  expires_at TIMESTAMPTZ,
  issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  redeemed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (shop_uuid, code)
);

CREATE INDEX IF NOT EXISTS gift_cards_shop_idx ON gift_cards(shop_uuid);
CREATE INDEX IF NOT EXISTS gift_cards_status_idx ON gift_cards(shop_uuid, status);

CREATE TABLE IF NOT EXISTS gift_card_transactions (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  gift_card_uuid UUID NOT NULL REFERENCES gift_cards(uuid) ON DELETE CASCADE,
  change_cents BIGINT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS gift_card_transactions_card_idx ON gift_card_transactions(gift_card_uuid);
