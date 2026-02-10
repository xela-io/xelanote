-- Migration 035: Add encryption_default to folders table
-- Allows folders to control whether new notes are created encrypted or not.
-- Default is 1 (encrypted) to maintain backward compatibility.
ALTER TABLE folders ADD COLUMN encryption_default INTEGER NOT NULL DEFAULT 1;
