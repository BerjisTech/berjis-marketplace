CREATE TABLE IF NOT EXISTS payment_settings (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  provider TEXT NOT NULL DEFAULT 'stripe',
  stripe_publishable_key TEXT NOT NULL DEFAULT '',
  stripe_secret_key TEXT NOT NULL DEFAULT '',
  stripe_webhook_secret TEXT NOT NULL DEFAULT '',
  is_active BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (shop_uuid, provider)
);

CREATE INDEX IF NOT EXISTS payment_settings_shop_idx ON payment_settings(shop_uuid);

CREATE TABLE IF NOT EXISTS payment_intents (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  order_uuid UUID NOT NULL REFERENCES orders(uuid) ON DELETE CASCADE,
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  stripe_payment_intent_id TEXT NOT NULL DEFAULT '',
  client_secret TEXT NOT NULL DEFAULT '',
  amount_cents BIGINT NOT NULL DEFAULT 0,
  currency TEXT NOT NULL DEFAULT 'USD',
  status TEXT NOT NULL DEFAULT 'requires_payment_method',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS payment_intents_order_idx ON payment_intents(order_uuid);
CREATE INDEX IF NOT EXISTS payment_intents_stripe_idx ON payment_intents(stripe_payment_intent_id);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_intent_uuid UUID REFERENCES payment_intents(uuid);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS stripe_payment_intent_id TEXT;
