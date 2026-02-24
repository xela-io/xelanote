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

## M2 - Feature & Quality Sprint (shipped)

**Status: Abgeschlossen** (2026-02-21)

What's shipped:
- ✅ Virtual Root (Migration 025); Hardcoded Root eliminiert, 13 Unit-Tests
- ✅ Cookie-Auth Hardening (SEC-006); Tokens nur in HttpOnly Cookies
- ✅ Mobile UX Improvements (Sidebar, Zoom-Fix, Toolbar Scrolling, Hanging Indent, Dark Mode)
- ✅ i18n Phase 1-3; ~604 Keys pro Locale (Dialoge, Editor, Admin, MarkdownGuide)
- ✅ AI Actions (8 Text-Transformationen via Claude/Gemini/ChatGPT)
- ✅ Offline Write Mode Phase 1 (IndexedDB Queue, Background Sync, Konflikterkennung)
- ✅ CI/CD: Forgejo Actions Auto-Deploy mit Auto-Rollback
- ✅ Note Sharing Phase 1 MVP (Viewer/Editor Rollen, User-Suche)
- ✅ Folder Sharing (implizite Permission-Vererbung, Placements)
- ✅ Collection Sharing (Kochbuch-Sharing mit 3-Tier Prioritaetskette)
- ✅ Encryption Toggle (Notizen entschluesseln/verschluesseln, Folder Defaults)
- ✅ CI Guidelines Enforcement (golangci-lint, Layer-Violation Ratchet, Security Checks)
- ✅ Rezepte (Structured Recipes, AI Import, Collections, Sharing, Dietary Preferences)
- ✅ Canvas (Infinite Canvas, JSON Canvas spec v1.0, Note Preview, Keyboard Shortcuts)
- ✅ Refactoring Sprint (live-preview→8 Module, codemirror→5 Module, markdown→8 Module, 200+ Tests)
- ✅ Layer Violations auf 0 reduziert (von 37 Baseline)

Done when: Alle Items implementiert. ✅

## M3 - UI Redesign & Preview (aktuell)

**Status: In Arbeit** (seit 2026-02-22)

Erledigt:
- ✅ Sidebar Redesign (Obsidian-Style Icon Strip, Active Indicators, Section Labels)
- ✅ Home Page Redesign (Activity Stats, Continue Working, All-Notes Listing)
- ✅ Editor Redesign (Frosted-Glass Toolbar, Surface Tokens, Card Panels)
- ✅ Mobile Bottom Navigation Bar (Notes, Search, More Tabs)
- ✅ Shared Design System (ui-panel, ui-list-item, ui-button etc. in app.css)
- ✅ Recipe Editor Overhaul (Inline Editing, Drag-and-Drop, Smart Parsing)
- ✅ Inline Title Editing (Bear/Apple Notes Style)
- ✅ PWA Improvements (Portrait Lock, Display Override, iOS Viewport Fix)
- ✅ Mobile Sidebar Push-and-Blur Effect

Offen:
- Live Preview Optimization (Shiki, KaTeX, Mermaid, Web Worker, Idiomorph, Scroll Sync)
- Templates/Snippets UI + Slash Palette

Abhaengigkeiten:
- Live Preview Optimization baut auf dem Refactoring Sprint (M2) auf

Risiken:
- Web Worker Rendering muss korrekt mit Encryption zusammenspielen

Done when:
- Live Preview Optimization fertig und getaggt
- Templates/Snippets UI ist nutzbar (Manager/Selector/Palette)

## Later - Nice-to-have

Ziele:
- Multi-Tab Editing + Split View (Obsidian-Style); **Status:** Detailliert geplant (6 Phasen), bestehende Stores vorhanden (tabs.svelte.ts, split-pane.svelte.ts); Cut-Kriterium: Phase 0-2 (Tabs ohne Split) fuer erstes Release; Refs: frontend/src/lib/stores/tabs.svelte.ts, frontend/src/lib/stores/split-pane.svelte.ts
- Offline Write Mode Phase 2 (Tags/Folders/Rename/Trash offline, Queue-UI); Phase 1 erledigt; Refs: frontend/src/lib/offline/
- Note Sharing Phase 3 (oeffentliche Links, Benachrichtigungen); Phase 1+2 erledigt (Folder+Collection Sharing shipped); Refs: backend/internal/db/sharing.go
- Hybrid Editor (Preview default, Markdown nur im aktiven Block); Refs: docs/planning/
- Generisches Listenarten-System (ToDo, Shopping etc.); Refs: docs/planning/list-types.md
- Plugins / Git Sync / Collaboration; Refs: docs/ideas.md

Abhaengigkeiten:
- Keine harten Abhaengigkeiten bekannt.

Risiken:
- Performance-Optimierungen brauchen Messungen.

Done when:
- Features sind implementiert und dokumentiert.
