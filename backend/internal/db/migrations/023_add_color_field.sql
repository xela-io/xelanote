-- Migration 023: Add color field to folders and notes
-- Color is stored as hex string (#RRGGBB) or NULL for no color

ALTER TABLE folders ADD COLUMN color TEXT DEFAULT NULL;
ALTER TABLE notes ADD COLUMN color TEXT DEFAULT NULL;
