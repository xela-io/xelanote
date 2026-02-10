# ROADMAP

Primaere Quelle fuer Milestones. TODO.md ist die Quelle fuer konkrete Tasks.

## M0 - Current State (shipped)

What's shipped:
- Multi-User Auth (JWT + Refresh Rotation + Cookie-Fallback); Refs: backend/internal/api/auth.go, backend/internal/service/auth.go, backend/internal/api/cookies.go
- Notes CRUD + Rename + Backlinks; Refs: backend/internal/api/notes.go, backend/internal/service/notes.go, backend/internal/db/links.go
- Version History + Restore; Refs: backend/internal/api/versions.go, backend/internal/db/migrations/011_note_versions.sql, frontend/src/lib/components/VersionHistoryDialog.svelte
- Trash + Undo/Redo UI; Refs: backend/internal/api/notes.go, frontend/src/routes/trash/+page.svelte, frontend/src/lib/stores/history.svelte.ts
- Graph View; Refs: backend/internal/api/graph.go, frontend/src/routes/graph/+page.svelte
- Folder Tree + Drag/Drop (Folder-Reorder); Refs: backend/internal/api/folders.go, frontend/src/lib/components/UnifiedTree.svelte
- Uploads (user-scoped) + Cookie-Auth fuer Images; Refs: backend/internal/api/uploads.go, backend/internal/api/cookies.go
- Templates/Snippets Backend + API + Stores; Refs: backend/internal/api/templates.go, backend/internal/api/snippets.go, frontend/src/lib/stores/templates.svelte.ts
- WebSocket Updates fuer Notes; Refs: backend/internal/api/websocket.go, frontend/src/lib/stores/websocket.svelte.ts
- Tags API + Filter UI + Assignment UI; Refs: backend/internal/api/tags.go, frontend/src/lib/components/FilterMenu.svelte, frontend/src/lib/components/TagEditor.svelte
- Offline Read Mode (PWA + Read-only Guard); Refs: frontend/vite.config.ts, frontend/src/lib/components/OfflineBanner.svelte, frontend/src/lib/api.ts
- Offline Write Mode Phase 1 (IndexedDB Queue + Background Sync + Conflict Resolution); Refs: frontend/src/lib/offline/, frontend/src/lib/components/ConflictDialog.svelte, frontend/src/lib/api.ts
- Note Sharing Phase 1 (Viewer + Editor Rollen, User-Suche, Share-Dialog); Refs: backend/internal/api/sharing.go, backend/internal/service/sharing.go, backend/internal/db/sharing.go
- Encryption Toggle (Notizen entschluesseln/verschluesseln, Folder Encryption Default); Refs: backend/internal/api/notes.go, backend/internal/api/folders.go, frontend/src/lib/components/EditorMoreMenu.svelte

Abhaengigkeiten: -
Risiken: -
Done when: Im Code vorhanden (siehe Refs).

## M1 - Stabilisierung (shipped)

**Status: Abgeschlossen** (2026-01-19)

What's shipped:
- Search Snippet XSS fix (sichere Snippets + keine untrusted {@html}); Refs: backend/internal/db/search.go, frontend/src/routes/search/+page.svelte
- Import error handling (ErrNotFound check); Refs: backend/internal/api/import.go, backend/internal/db/errors.go
- Tag Assignment UI (Note Tags setzen/entfernen); Refs: backend/internal/api/tags.go, frontend/src/lib/components/TagEditor.svelte
- 409-Konflikt-Erkennung beim Speichern; Refs: frontend/src/lib/api.ts, frontend/src/lib/stores/notes.svelte.ts
- Settings Speichern (Preferences, Email, Password); Refs: backend/internal/api/users.go, backend/internal/db/migrations/015_user_preferences.sql
- Mobile Versionshistorie (Tab-Navigation); Refs: frontend/src/lib/components/VersionHistoryDialog.svelte, docs/mobile-version-history.md
- iOS Autokorrektur; Refs: frontend/src/lib/editor/codemirror.ts
- Security Hardening (Rate-Limiting, CSP Headers, Non-root Docker); Refs: backend/internal/api/ratelimit.go, backend/internal/api/security.go, Dockerfile

Done when: Alle Items aus TODO.md Done-Liste sind implementiert.

## M2 - Next Feature Cluster

**Status: In Arbeit**

Erledigt:
- ✅ Virtual Root (Migration 025); Hardcoded Root-Folder id=1 eliminiert, top-level Folders haben parent_id=NULL, 13 Unit-Tests, E2E-Tests; Refs: backend/internal/db/migrations/025_virtual_root.sql, backend/internal/db/folders_test.go
- ✅ Cookie-Auth Hardening (SEC-006); Tokens nur in HttpOnly Cookies, localStorage entfernt; Refs: backend/internal/api/cookies.go
- ✅ Mobile UX Improvements (Sidebar, Zoom-Fix, Editor-Toolbar Scrolling, List Hanging Indent, Dark Mode Farben); Refs: frontend/src/lib/components/Sidebar.svelte, frontend/src/lib/editor/codemirror.ts, frontend/src/lib/components/Editor.svelte, frontend/src/app.css
- ✅ i18n Phase 1 (Dialoge + Editor); 7 Komponenten vollständig internationalisiert (~55 neue Keys pro Locale), ICU MessageFormat, {@html} Support, 468 Keys pro Locale-Datei; Refs: frontend/src/lib/components/*.svelte, frontend/src/lib/locales/en.json, frontend/src/lib/locales/de.json
- ✅ i18n Phase 2 (MarkdownGuideDialog Content Arrays); 54 neue Keys pro Locale, Content Arrays mit `$derived` fuer Locale-Reaktivitaet, Info-Boxen internationalisiert, 522 Keys pro Locale-Datei; Refs: frontend/src/lib/components/MarkdownGuideDialog.svelte, frontend/src/lib/locales/en.json, frontend/src/lib/locales/de.json
- ✅ i18n Phase 3 (Admin Page); 82 neue Keys pro Locale unter `page.admin` Namespace (tabs, dashboard, users, activity, settings, delete_dialog), formatDate() nutzt `$locale`, dynamisches Key-Pattern fuer getActionLabel(), ~604 Keys pro Locale-Datei; Refs: frontend/src/routes/admin/+page.svelte, frontend/src/lib/locales/en.json, frontend/src/lib/locales/de.json
- ✅ AI Actions (Text-Transformation via LLM); 8 Aktionen (Format, Summarize, Expand, Translate DE/EN, Formal, Informal, Custom), per-Note opt-in via `ai_enabled`, Diff-Preview vor Anwenden, Konflikterkennung, Encrypted-Notes-Support, Prompt-Injection-Schutz (Sandwich-Pattern), Validierung (10 Chars min, 50KB max), Claude/Gemini Provider; Refs: backend/internal/api/notes.go (Endpoints), backend/internal/llm/prompts.go (Prompts), backend/internal/service/summarize.go (Service), frontend/src/lib/components/AIActionsDropdown.svelte, frontend/src/lib/components/AITransformDialog.svelte, frontend/src/lib/api.ts
- ✅ Offline Write Mode Phase 1 (IndexedDB Queue + Background Sync); Notes erstellen/bearbeiten/verschieben/loeschen offline, verschluesselte IndexedDB-Queue, automatischer Sync bei Reconnect, Konflikterkennung (HTTP 409) mit manuellem Resolution-Dialog, Offline-Status-Pill in Editor-Toolbar, Paranoid Mode read-only, Tab-Safety via navigator.locks, Queue-Optimierung, Temp-ID-System mit URL-Rewriting; Phase 2 (Tags/Folders/Rename/Trash offline) steht noch aus; Refs: frontend/src/lib/offline/, frontend/src/lib/components/ConflictDialog.svelte, frontend/src/lib/api.ts, frontend/src/lib/stores/notes.svelte.ts
- ✅ CI/CD: Forgejo Actions Auto-Deploy fuer Staging; Push auf `forgejo main` triggered automatischen Build + Deploy auf Staging (<STAGING_URL>), SHA-pinned Checkout, Pre-Flight Checks, Docker Build mit SHA-Tags, Zero-Downtime Deploy, Health Checks, Auto-Rollback, Security-Hardening (read-only FS, cap-drop ALL); Refs: .forgejo/workflows/deploy-staging.yml, scripts/setup-forgejo-runner.sh
- ✅ Note Sharing Phase 1 MVP; Notizen mit anderen Benutzern teilen (Viewer + Editor Rollen), 8 API-Endpoints, DB-Migration 034 (note_shares + folder_shares Tabellen), ShareNoteDialog mit User-Suche, SharedWithMeList, Sidebar Count-Badge, /shared Route, 22 i18n Keys, 10 DB-Tests, E2E-verschluesselte Notizen nicht teilbar; Phase 2 (Folder Sharing) steht noch aus; Refs: backend/internal/db/sharing.go, backend/internal/service/sharing.go, backend/internal/api/sharing.go, backend/internal/db/migrations/034_note_sharing.sql
- ✅ Encryption Toggle fuer Notizen und Ordner; Einzelne Notizen ueber More-Menu entschluesseln/verschluesseln, Ordner als "unverschluesselt" markieren (neue Notizen ohne E2E), automatische Share-Entfernung beim Verschluesseln, 3 neue API-Endpoints, DB-Migration 035 (encryption_default), 7 neue DB-Tests; Refs: backend/internal/db/migrations/035_encryption_toggle.sql, backend/internal/api/notes.go, backend/internal/api/folders.go, frontend/src/lib/components/EditorMoreMenu.svelte, frontend/src/lib/stores/notes.svelte.ts

Offen:
- Templates/Snippets UI + Slash Palette; Refs: TEMPLATES_SNIPPETS_STATUS.md, frontend/src/lib/stores/snippets.svelte.ts

Abhaengigkeiten:
- (Root-Handling erledigt - keine Migration bestehender Pfade noetig)

Risiken:
- (Keine bekannten Risiken mehr)

Done when:
- Templates/Snippets UI ist nutzbar (Manager/Selector/Palette)
- ~~i18n für alle User-facing Components abgeschlossen~~ ✅ (Phase 1-3 erledigt, ~604 Keys pro Locale)

## Later - Nice-to-have

Ziele:
- Multi-Tab Editing + Split View (Obsidian-Style); **Status:** Detailliert geplant (6 Phasen), bestehende Stores vorhanden (tabs.svelte.ts, split-pane.svelte.ts), Notes-Store muss refactored werden; Cut-Kriterium: Phase 0-2 (Tabs ohne Split) fuer erstes Release; **Plan:** `.claude/plans/immutable-finding-harbor.md`; Refs: frontend/src/lib/stores/tabs.svelte.ts, frontend/src/lib/stores/split-pane.svelte.ts
- Live Preview (Obsidian-Style); **Status:** Verschoben (kritische Analyse hat fundamentale Probleme aufgedeckt). Markdown-Syntax verstecken, formatierten Text anzeigen, Syntax bei Cursor-Nähe wieder sichtbar. **Richtiger Ansatz:** `Decoration.replace()` statt CSS-Tricks (font-size:0 zerstört Cursor-Positionierung). Aufwand: 3-4 Wochen. Alternative: Split-View beibehalten; Refs: frontend/src/lib/editor/spell-check.ts, frontend/src/lib/editor/codemirror.ts
- ~~Color Syntax Support~~; **Status:** ✅ Implementiert (Feature-Flag enabled), `{color:VALUE}text{/color}` Syntax, ColorPicker UI; Refs: frontend/src/lib/editor/markdown.ts, frontend/src/lib/components/ColorPickerPopover.svelte, docs/markdown-guide.md
- ~~Auto-Sort Task Lists~~; **Status:** ✅ Implementiert, Automatisches Sortieren beim Checkbox-Toggle (checked → Ende, unchecked → Anfang), atomare Undo-Transaktion, Listen-Grenzen-Erkennung; Refs: frontend/src/lib/components/Editor.svelte (toggleTaskByIndex, calculateTargetPosition, findTaskListBoundary), docs/editor-features.md
- Drag & Drop fuer Task-Reihenfolge; **Status:** Backlog, Manuelle Neuordnung von Tasks per Drag & Drop (Ergänzung zu Auto-Sort, optional deaktivierbar); Refs: docs/editor-features.md (Zukünftige Erweiterungen)
- Note Reordering (display_order fuer Notes); Refs: backend/internal/db/migrations/003_add_order_fields.sql, frontend/src/lib/components/UnifiedTree.svelte
- ~~Tree Virtual Scrolling~~; **Status:** ✅ Implementiert (Commits 0ccb80b + 08ee054), Opt-in via Settings (Experimental); Refs: frontend/src/lib/components/VirtualizedTree.svelte, frontend/src/routes/settings/+page.svelte
- Note Sharing Phase 2 (Folder Sharing, oeffentliche Links, Benachrichtigungen); **Status:** Tabelle `folder_shares` bereits in Migration 034 vorbereitet, Phase 1 MVP shipped; Refs: backend/internal/db/migrations/034_note_sharing.sql, backend/internal/db/sharing.go
- Plugins / Git Sync / Collaboration; Refs: docs/ideas.md, CLAUDE.md

Abhaengigkeiten:
- Keine harten Abhaengigkeiten bekannt.

Risiken:
- Performance-Optimierungen brauchen Messungen.

Done when:
- Features sind implementiert und dokumentiert.
