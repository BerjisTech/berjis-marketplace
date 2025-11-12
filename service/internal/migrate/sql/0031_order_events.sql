CREATE TABLE IF NOT EXISTS order_events (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  order_uuid UUID NOT NULL REFERENCES orders(uuid) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  message TEXT NOT NULL DEFAULT '',
  metadata JSONB,
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS order_events_order_idx ON order_events(order_uuid);
CREATE INDEX IF NOT EXISTS order_events_type_idx ON order_events(event_type);
