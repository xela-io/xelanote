-- =============================================================
-- AI-ENABLED FLAGS FÜR CLAUDE API INTEGRATION
-- =============================================================
-- Ermöglicht Opt-In für Cloud-KI (Claude API) pro Notiz/Ordner.
-- ai_enabled=false -> Nur lokale KI (Ollama)
-- ai_enabled=true  -> Cloud-KI erlaubt (wenn API-Key vorhanden)

-- 1. Notes: ai_enabled Flag
-- Default 0 (false) = Safe Default, nur lokale KI
ALTER TABLE notes ADD COLUMN ai_enabled INTEGER DEFAULT 0;

-- 2. Folders: Default für neue Notizen in diesem Ordner
-- Neue Notizen erben diesen Wert (außer Root-Notizen -> immer 0)
ALTER TABLE folders ADD COLUMN ai_enabled_default INTEGER DEFAULT 0;

-- 3. Index für Performance bei Provider-Router Queries
-- Filtert auf user_id + ai_enabled für schnelle Lookups
CREATE INDEX IF NOT EXISTS idx_notes_ai_enabled
ON notes(user_id, ai_enabled)
WHERE is_deleted = 0;

-- 4. Index für GetNoteTitlesAIEnabled (Link-Suggestions mit Claude)
-- Nur freigegebene Notizen werden bei Claude-Requests geladen
CREATE INDEX IF NOT EXISTS idx_notes_ai_enabled_titles
ON notes(user_id, title)
WHERE is_deleted = 0 AND ai_enabled = 1;
