CREATE TABLE IF NOT EXISTS recipe_images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    image_url TEXT NOT NULL CHECK(length(image_url) <= 2048),
    caption TEXT CHECK(length(caption) <= 500),
    display_order INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_recipe_images_note ON recipe_images(note_id, display_order, id);
CREATE INDEX IF NOT EXISTS idx_recipe_images_user ON recipe_images(user_id);
