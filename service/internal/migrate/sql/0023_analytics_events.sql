CREATE TABLE IF NOT EXISTS analytics_events (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID REFERENCES shops(uuid) ON DELETE CASCADE,
  session_id UUID,
  user_uuid UUID,
  event_name TEXT NOT NULL,
  payload JSONB,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS analytics_events_shop_idx ON analytics_events(shop_uuid);
CREATE INDEX IF NOT EXISTS analytics_events_user_idx ON analytics_events(user_uuid);
CREATE INDEX IF NOT EXISTS analytics_events_name_idx ON analytics_events(event_name);
