-- Add admin flag to users table
ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_users_is_admin ON users(is_admin);

-- Make the first user (lowest ID) an admin
UPDATE users SET is_admin = 1 WHERE id = (SELECT MIN(id) FROM users);
