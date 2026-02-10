-- Recipe collection sharing: allows users to share cookbooks with other users
CREATE TABLE recipe_collection_shares (
    id INTEGER PRIMARY KEY,
    collection_id INTEGER NOT NULL,
    owner_user_id INTEGER NOT NULL,
    shared_with_user_id INTEGER NOT NULL,
    role TEXT NOT NULL DEFAULT 'viewer' CHECK (role IN ('viewer', 'editor')),
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (collection_id) REFERENCES recipe_collections(id) ON DELETE CASCADE,
    FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (shared_with_user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(collection_id, shared_with_user_id)
);

CREATE INDEX idx_collection_shares_shared_with ON recipe_collection_shares(shared_with_user_id);
CREATE INDEX idx_collection_shares_collection ON recipe_collection_shares(collection_id);
CREATE INDEX idx_collection_shares_access ON recipe_collection_shares(shared_with_user_id, collection_id);
