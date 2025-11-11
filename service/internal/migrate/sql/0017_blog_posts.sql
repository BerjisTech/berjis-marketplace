CREATE TABLE IF NOT EXISTS blog_posts (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  author_uuid UUID,
  title TEXT NOT NULL,
  slug TEXT NOT NULL,
  excerpt TEXT NOT NULL DEFAULT '',
  content TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  published_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (shop_uuid, slug)
);

CREATE INDEX IF NOT EXISTS blog_posts_shop_idx ON blog_posts(shop_uuid);
CREATE INDEX IF NOT EXISTS blog_posts_status_idx ON blog_posts(shop_uuid, status);
