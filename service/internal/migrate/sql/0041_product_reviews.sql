CREATE TABLE IF NOT EXISTS product_reviews (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  user_uuid UUID NOT NULL,
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending',
  is_verified_purchase BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS product_reviews_product_idx ON product_reviews(product_uuid);
CREATE INDEX IF NOT EXISTS product_reviews_user_idx ON product_reviews(user_uuid);
CREATE INDEX IF NOT EXISTS product_reviews_shop_idx ON product_reviews(shop_uuid);
CREATE INDEX IF NOT EXISTS product_reviews_status_idx ON product_reviews(status);
