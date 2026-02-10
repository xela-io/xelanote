# Migration 024 Bug Fix Notes

## Problem Discovery (2026-01-25)

Nach dem ersten Deployment von Migration 024 auf Staging wurde ein Bug entdeckt:

**Symptom:** User konnten keine Notizen mit demselben Titel erstellen, auch wenn diese in verschiedenen Ordnern liegen sollten.

**Root Cause:** Ein alter globaler Index `idx_notes_title_norm` existierte noch in der Datenbank, obwohl Migration 010 ihn hätte löschen sollen.

## Index-Analyse

Die Datenbank hatte nach Migration 024 (initial) **zwei** UNIQUE Indexes:

1. ✅ `idx_notes_user_folder_title_norm` (neu, korrekt)
   - Constraint: `(user_id, folder_path, title_norm)`
   - Scope: Per-folder uniqueness

2. ❌ `idx_notes_title_norm` (alt, problematisch)
   - Constraint: `(title_norm)` global
   - Source: `schema.sql` (sehr alte Definition)
   - Sollte gelöscht sein durch: Migration 010

## Warum existierte der alte Index noch?

Migration 010 löschte `idx_notes_title_norm`:

```sql
-- Migration 010, Zeile 8:
DROP INDEX IF EXISTS idx_notes_title_norm;
```

**Aber:** `schema.sql` hatte ihn weiterhin definiert:

```sql
-- schema.sql, Zeile 21:
CREATE UNIQUE INDEX IF NOT EXISTS idx_notes_title_norm ON notes(title_norm) WHERE is_deleted = 0;
```

**Vermutung:** Bei frischen Datenbanken wurde `schema.sql` NACH den Migrationen angewendet (oder schema.sql wurde verwendet statt Migrationen), wodurch der alte Index wiederhergestellt wurde.

## Fix

Migration 024 wurde aktualisiert, um **defensiv BEIDE** alte Indexes zu löschen:

```sql
-- STEP 2: Drop ALL old unique indexes (defense in depth)
DROP INDEX IF EXISTS idx_notes_title_norm;        -- Global (schema.sql)
DROP INDEX IF EXISTS idx_notes_user_title_norm;   -- User-scoped (Migration 010)
```

## Deployment-Notizen

### Staging (<STAGING_URL>)
- Migration 024 initial lief, aber mit altem Index
- Manueller Fix: `DROP INDEX IF EXISTS idx_notes_title_norm;`
- Updated Migration committed für Production

### Production (xelanote.com)
- Wird mit fixed Migration deployed
- Sollte keine manuellen Eingriffe benötigen

## Lessons Learned

1. **Defense in Depth:** Migrations sollten ALLE möglichen alten Indexes löschen, nicht nur den erwarteten
2. **schema.sql vs Migrations:** Unclear welche Quelle zuerst läuft bei frischen Installs
3. **Verification:** Nach Migration immer alle Indexes prüfen, nicht nur neue

## Testing

Nach dem Fix verifiziert:
- ✅ Notiz "Commands" in `/Code/Linux` existierte bereits
- ✅ Notiz "Commands" in `/personal` konnte erfolgreich erstellt werden
- ✅ Notiz "Commands" nochmal in `/personal` → 409 Conflict (korrekt)
