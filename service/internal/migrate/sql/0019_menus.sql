CREATE TABLE IF NOT EXISTS menus (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  shop_uuid UUID NOT NULL REFERENCES shops(uuid) ON DELETE CASCADE,
  name TEXT NOT NULL,
  handle TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (shop_uuid, handle)
);

CREATE TABLE IF NOT EXISTS menu_items (
  uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  menu_uuid UUID NOT NULL REFERENCES menus(uuid) ON DELETE CASCADE,
  parent_uuid UUID REFERENCES menu_items(uuid) ON DELETE CASCADE,
  title TEXT NOT NULL,
  url TEXT NOT NULL,
  target TEXT NOT NULL DEFAULT '_self',
  position INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS menu_items_menu_idx ON menu_items(menu_uuid);
CREATE INDEX IF NOT EXISTS menu_items_parent_idx ON menu_items(parent_uuid);
