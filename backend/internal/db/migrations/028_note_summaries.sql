-- Migration 028: Add LLM summary fields to notes table
-- Adds support for AI-generated summaries with encryption support

-- Summary fields
ALTER TABLE notes ADD COLUMN summary TEXT;
ALTER TABLE notes ADD COLUMN encrypted_summary TEXT;
ALTER TABLE notes ADD COLUMN summary_encrypted INTEGER DEFAULT 0;

-- Content hash for change detection (SHA256, first 16 chars)
ALTER TABLE notes ADD COLUMN content_hash TEXT;

-- Timestamp when summary was generated (ISO8601/RFC3339 format)
ALTER TABLE notes ADD COLUMN summary_generated_at TEXT;

-- Index for efficient lookup of notes needing summary updates
-- Covers: undeleted notes with content_hash for change detection
CREATE INDEX IF NOT EXISTS idx_notes_content_hash ON notes(user_id, content_hash) WHERE is_deleted = 0;

-- Index for finding notes that need summary generation
-- Covers: undeleted, unencrypted notes without summaries
CREATE INDEX IF NOT EXISTS idx_notes_pending_summary ON notes(user_id, content_encrypted, summary_generated_at) WHERE is_deleted = 0 AND content_encrypted = 0;
