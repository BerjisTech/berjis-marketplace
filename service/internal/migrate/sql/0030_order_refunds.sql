ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS refund_total_cents BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS refunded_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS order_refunds (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  order_uuid UUID NOT NULL REFERENCES orders(uuid) ON DELETE CASCADE,
  amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
  reason TEXT NOT NULL DEFAULT '',
  processed_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS order_refunds_order_idx ON order_refunds(order_uuid);
