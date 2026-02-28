-- Shopping lists (user can have multiple lists)
CREATE TABLE IF NOT EXISTS shopping_lists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK(length(name) <= 200),
    color TEXT CHECK(length(color) <= 20),
    is_archived INTEGER NOT NULL DEFAULT 0,
    display_order INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, name)
);
CREATE INDEX IF NOT EXISTS idx_shopping_lists_user
    ON shopping_lists(user_id, is_archived, display_order);

-- Shopping items (user derived via list_id -> shopping_lists.user_id)
CREATE TABLE IF NOT EXISTS shopping_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    list_id INTEGER NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK(length(name) <= 300),
    quantity REAL CHECK(quantity IS NULL OR quantity >= 0),
    unit TEXT CHECK(length(unit) <= 50),
    category TEXT CHECK(length(category) <= 100),
    category_order INTEGER NOT NULL DEFAULT 99,
    parent_id INTEGER REFERENCES shopping_items(id) ON DELETE CASCADE,
    is_checked INTEGER NOT NULL DEFAULT 0,
    checked_at TEXT,
    display_order INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    added_by_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    source_recipe_id TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_shopping_items_list
    ON shopping_items(list_id, is_checked, display_order);
CREATE INDEX IF NOT EXISTS idx_shopping_items_parent
    ON shopping_items(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_shopping_items_category
    ON shopping_items(list_id, category_order, display_order);

-- Trigger: parent_id must belong to same list_id + max 1 nesting level
CREATE TRIGGER IF NOT EXISTS trg_shopping_items_parent_validate_insert
BEFORE INSERT ON shopping_items
WHEN NEW.parent_id IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'parent_id must belong to same list_id')
    WHERE (SELECT list_id FROM shopping_items WHERE id = NEW.parent_id) != NEW.list_id;
    SELECT RAISE(ABORT, 'maximum nesting depth exceeded (1 level)')
    WHERE (SELECT parent_id FROM shopping_items WHERE id = NEW.parent_id) IS NOT NULL;
END;

CREATE TRIGGER IF NOT EXISTS trg_shopping_items_parent_validate_update
BEFORE UPDATE ON shopping_items
WHEN NEW.parent_id IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'item cannot be its own parent')
    WHERE NEW.parent_id = NEW.id;
    SELECT RAISE(ABORT, 'parent_id must belong to same list_id')
    WHERE (SELECT list_id FROM shopping_items WHERE id = NEW.parent_id) != NEW.list_id;
    SELECT RAISE(ABORT, 'maximum nesting depth exceeded (1 level)')
    WHERE (SELECT parent_id FROM shopping_items WHERE id = NEW.parent_id) IS NOT NULL;
END;

-- Favorites (per user, independent of lists)
CREATE TABLE IF NOT EXISTS shopping_favorites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK(length(name) <= 300),
    default_quantity REAL CHECK(default_quantity IS NULL OR default_quantity >= 0),
    default_unit TEXT CHECK(length(default_unit) <= 50),
    category TEXT CHECK(length(category) <= 100),
    usage_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, name)
);
CREATE INDEX IF NOT EXISTS idx_shopping_favorites_user
    ON shopping_favorites(user_id, usage_count DESC);

-- Live sharing (default role: editor)
CREATE TABLE IF NOT EXISTS shopping_list_shares (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    list_id INTEGER NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
    owner_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shared_with_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'editor' CHECK (role IN ('viewer', 'editor')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(list_id, shared_with_user_id),
    CHECK (owner_user_id != shared_with_user_id)
);
CREATE INDEX IF NOT EXISTS idx_shopping_shares_shared_with
    ON shopping_list_shares(shared_with_user_id);
CREATE INDEX IF NOT EXISTS idx_shopping_shares_list
    ON shopping_list_shares(list_id);

-- Enable shopping feature for existing users
INSERT OR IGNORE INTO user_features (user_id, feature, enabled, updated_at)
SELECT id, 'shopping', 1, datetime('now') FROM users;
