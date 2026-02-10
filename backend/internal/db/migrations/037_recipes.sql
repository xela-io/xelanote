-- Recipe metadata (1:1 with note)
CREATE TABLE IF NOT EXISTS recipe_metadata (
    note_id TEXT NOT NULL PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    servings INTEGER DEFAULT 4 CHECK(servings BETWEEN 1 AND 999),
    prep_time_minutes INTEGER CHECK(prep_time_minutes >= 0 AND prep_time_minutes <= 99999),
    cook_time_minutes INTEGER CHECK(cook_time_minutes >= 0 AND cook_time_minutes <= 99999),
    source_url TEXT CHECK(length(source_url) <= 2048),
    difficulty TEXT CHECK(difficulty IN ('easy', 'medium', 'hard')),
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

-- Recipe ingredients (1:N with note)
CREATE TABLE IF NOT EXISTS recipe_ingredients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount REAL CHECK(amount >= 0),
    amount_text TEXT CHECK(length(amount_text) <= 100),
    unit TEXT CHECK(length(unit) <= 50),
    name TEXT NOT NULL CHECK(length(name) <= 200),
    group_name TEXT CHECK(length(group_name) <= 100),
    display_order INTEGER DEFAULT 0,
    optional INTEGER DEFAULT 0,
    scalable INTEGER DEFAULT 1,
    created_at TEXT DEFAULT (datetime('now'))
);

-- Recipe collections (cookbooks, owner-only)
CREATE TABLE IF NOT EXISTS recipe_collections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK(length(name) <= 200),
    description TEXT CHECK(length(description) <= 1000),
    color TEXT CHECK(length(color) <= 20),
    display_order INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    UNIQUE(user_id, name)
);

-- Recipe collection items (M:N between collections and notes)
CREATE TABLE IF NOT EXISTS recipe_collection_items (
    collection_id INTEGER NOT NULL REFERENCES recipe_collections(id) ON DELETE CASCADE,
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    display_order INTEGER DEFAULT 0,
    added_at TEXT DEFAULT (datetime('now')),
    PRIMARY KEY (collection_id, note_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_recipe_metadata_user ON recipe_metadata(user_id);
CREATE INDEX IF NOT EXISTS idx_recipe_ingredients_note ON recipe_ingredients(note_id, display_order, id);
CREATE INDEX IF NOT EXISTS idx_recipe_ingredients_user ON recipe_ingredients(user_id);
CREATE INDEX IF NOT EXISTS idx_recipe_collections_user ON recipe_collections(user_id, display_order);
CREATE INDEX IF NOT EXISTS idx_notes_recipe_type ON notes(user_id, note_type, updated_at)
    WHERE note_type = 'recipe' AND is_deleted = 0;

-- Note: updated_at is managed by the Go application layer (not triggers)
-- to avoid conflicts with optimistic locking and ensure millisecond precision.
