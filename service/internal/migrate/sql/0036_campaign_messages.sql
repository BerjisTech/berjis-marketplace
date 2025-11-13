CREATE TABLE IF NOT EXISTS campaign_messages (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  campaign_uuid UUID NOT NULL REFERENCES marketing_campaigns(uuid) ON DELETE CASCADE,
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  subject TEXT NOT NULL,
  body TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'scheduled',
  scheduled_at TIMESTAMPTZ NOT NULL,
  send_after TIMESTAMPTZ,
  sent_at TIMESTAMPTZ,
  error TEXT,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS campaign_messages_campaign_idx ON campaign_messages(campaign_uuid);
CREATE INDEX IF NOT EXISTS campaign_messages_status_idx ON campaign_messages(status, scheduled_at);
