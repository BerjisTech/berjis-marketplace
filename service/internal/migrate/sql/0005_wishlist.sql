CREATE TABLE IF NOT EXISTS wishlists (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_uuid UUID NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wishlist_items (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  wishlist_uuid UUID NOT NULL REFERENCES wishlists(uuid) ON DELETE CASCADE,
  product_uuid UUID NOT NULL REFERENCES products(uuid) ON DELETE CASCADE,
  added_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (wishlist_uuid, product_uuid)
);
