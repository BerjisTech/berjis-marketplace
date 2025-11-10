CREATE TABLE IF NOT EXISTS user_profiles (
  user_uuid UUID PRIMARY KEY,
  display_name TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL DEFAULT '',
  phone TEXT NOT NULL DEFAULT '',
  avatar_url TEXT,
  timezone TEXT NOT NULL DEFAULT 'UTC',
  marketing_opt_in BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_addresses (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_uuid UUID NOT NULL REFERENCES user_profiles(user_uuid) ON DELETE CASCADE,
  label TEXT NOT NULL DEFAULT 'Primary',
  recipient_name TEXT NOT NULL DEFAULT '',
  line1 TEXT NOT NULL,
  line2 TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL,
  region TEXT NOT NULL,
  postal_code TEXT NOT NULL,
  country TEXT NOT NULL,
  phone TEXT NOT NULL DEFAULT '',
  is_default_shipping BOOLEAN NOT NULL DEFAULT FALSE,
  is_default_billing BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS user_addresses_user_idx ON user_addresses(user_uuid);
CREATE INDEX IF NOT EXISTS user_addresses_default_shipping_idx ON user_addresses(user_uuid) WHERE is_default_shipping;
CREATE INDEX IF NOT EXISTS user_addresses_default_billing_idx ON user_addresses(user_uuid) WHERE is_default_billing;
