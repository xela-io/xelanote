ALTER TABLE users ADD COLUMN storage_limit_mb INTEGER DEFAULT NULL CHECK(storage_limit_mb IS NULL OR storage_limit_mb >= 0);
