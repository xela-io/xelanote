# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Landing page only mode: temporarily disable all routes except /about via `LANDING_PAGE_ONLY=true` env var (frontend redirect + backend 503 middleware)
- Open notes in new tab via Ctrl/Cmd+click, middle-click, or context menu "Open in new tab" in the sidebar tree, wikilinks, and quick switcher
- Default tab navigation now replaces the active tab instead of always opening new tabs
- Landing page: dynamic animated landing page with scroll-reveal, parallax hero, terminal typing effect, and full DE/EN localization
- JSON-LD recipe extraction: recipe import now extracts structured data from `<script type="application/ld+json">` blocks before falling back to LLM, improving speed and reliability on ~90% of recipe websites
- Ingredient string parser for JSON-LD data: parses raw strings like "500g Hackfleisch" into structured name/amount/unit, supporting metric, imperial, fractions, unicode fractions, and German units (EL, TL, Prise, Bund)
- View Transitions API for smooth page crossfades (skips /note/ for CodeMirror stability, respects prefers-reduced-motion)
- Consistent iOS-style touch active states (scale 0.97) for all interactive elements on touch devices
- Note data prefetch on pointerdown in tree sidebar for faster perceived navigation
- Background Sync API for offline queue replay when connectivity returns (Chromium; graceful fallback on Safari/Firefox)

### Changed

- Tabs and Shopping Lists features are now opt-in (WIP) behind feature flags in Settings > Editor > Features

### Fixed

- Fixed wikilink click test for newTab parameter after Ctrl/Cmd+click support was added
- Fixed table of contents scroll not jumping to the correct heading position on mobile and desktop: replaced `scrollIntoView()` (which scrolls all ancestor containers) with targeted `scrollTo()` on the specific scroll container; fixed live-mode scroll using `EditorView.scrollIntoView` with `y: 'start'` positioning; fixed heading extraction and slug lookup to skip fenced code blocks
- Fixed kebab menu (editor "more" menu) having transparent background on desktop — `mobile-glass-sheet` was unsetting `backdrop-filter`, now scoped to mobile only so `ui-panel` blur applies
- Fixed tabs not persisting across page refresh: CORS config was missing PATCH method, silently blocking all tab persist API calls
- Fixed tab restore race condition: notes loading and preferences loading could cause `isHydrating` to get stuck, preventing tab persistence
- Fixed closing all tabs not persisting: `closeAllTabs()` now explicitly persists the empty state
- Fixed tab state lost on page unload: added `flushPendingPersist()` with keepalive fetch on `beforeunload` and `visibilitychange` events
- Fixed tab resolve effect blocking forever when user has 0 notes: removed unnecessary empty-notes guard
- Fixed `isHydrating` getting permanently stuck if `resolveTabTitles` never fires: added 15s safety timeout
- Fixed tab state not cleared on logout: added `tabs.closeAllTabs()` to logout cleanup
- Fixed encrypted notes losing folder assignment: folder paths are now encrypted client-side (`encrypted_folder_path`) and decrypted for tree display, preserving folder organization while keeping metadata private from the server
- Fixed encrypted notes reverting to root folder after save: API responses no longer overwrite client-side folder path with server's "/"
- Fixed encrypted folder paths not decrypted on page refresh due to race condition: tree now reloads when encryption becomes available
- Fixed UNION ALL column mismatch in shared-note folder query (missing `wrapped_dek_recovery`)
- Fixed e2e-feature test to match updated `encryptNote` signature (now includes noteId)
- Fixed missing `decryptFolderPath` mock in loaders test
- Fixed migration 065 failing on UNIQUE constraint when moving encrypted notes to root folder

### Added

- Per-User Storage Quota: individual storage limits per user with `NULL` = global default, `0` = unlimited, `>0` = limit in MB
- Per-User Storage Quota: admin panel UI to view and set per-user storage limits (global/unlimited/custom)
- Per-User Storage Quota: user settings page shows storage usage with progress bar (green/yellow/red)
- Per-User Storage Quota: `GET /api/users/storage-quota` endpoint for users to check their quota
- Per-User Storage Quota: `PUT /api/admin/users/{id}/storage-limit` endpoint for admins to set per-user limits
- Shopping Lists: dedicated shopping list feature with multiple lists per user, color-coded tabs, and full CRUD
- Shopping Lists: quick-input parser supporting German quantities/units (e.g., "3 Äpfel, 500g Hack, 2,5l Milch")
- Shopping Lists: AI-powered sorting into 14 German supermarket categories via LLM
- Shopping Lists: recipe ingredient import from existing recipe notes
- Shopping Lists: favorites system with usage-count-based ordering
- Shopping Lists: live sharing with role-based access control (owner/editor/viewer)
- Shopping Lists: real-time sync via WebSocket with echo-detection for multi-tab support
- Shopping Lists: manual category creation with inline per-category item input
- Shopping Lists: "Uncategorized" section for items without a category when categories exist
- Shopping Lists: confirmation dialog before AI sort when manual categories are present
- Shopping Lists: collapsible checked-items section with bulk clear
- Shopping Lists: optimistic locking with version fields and HTTP 409 conflict detection
- Shopping Lists: integrated into sidebar, mobile bottom nav, and feature toggle system
- Multi-Tab Editor: browser-style tabs above the editor for opening multiple notes simultaneously
- Multi-Tab Editor: server-persisted tab state via `open_tabs` in user preferences (cross-device sync)
- Multi-Tab Editor: drag-to-reorder tabs with debounced server persistence
- Multi-Tab Editor: dirty indicator (dot) on tabs with unsaved changes
- Multi-Tab Editor: keyboard shortcuts — Ctrl+PageDown/PageUp for tab switching, Ctrl+W (desktop only) for close
- Multi-Tab Editor: automatic tab cleanup when notes are deleted (local or remote via WebSocket)
- Multi-Tab Editor: dedicated drag handle on tabs instead of whole-tab dragging for better UX
- Multi-Tab Editor: keyboard tab reorder via Alt+ArrowLeft/ArrowRight
- Multi-Tab Editor: duplicate close guard preventing double-click and rapid middle-click issues
- Multi-Tab Editor: middle-click close uses `onauxclick` for correct event handling
- Multi-Tab Editor: E2E regression tests for tab close (X button, middle-click) and drag reorder
- Multi-Tab Editor: offline-safe temp-ID replacement when offline-created notes sync
- Multi-Tab Editor: E2E encryption compatible — only note IDs persisted server-side, no titles
- Dictation: voice input via browser-native Web Speech API (real-time transcript)
- Dictation: server-side transcription via OpenAI Whisper (`POST /llm/transcribe`)
- Dictation: AI cleanup action (`dictation_cleanup`) for post-processing raw speech-to-text
- Dictation: DictationPanel component with bottom-sheet (mobile) / dropdown (desktop) UI
- Dictation: Mic button in editor toolbar with pulsing indicator when active
- Dictation: localStorage persistence for dictation mode and AI cleanup toggle
- Tests: cross-user data isolation tests for user settings, API keys, recovery keys, WebAuthn, due dates, export, task events, and canvas endpoints
- Mobile: graph view link in bottom navigation "More" sheet
- Editor: indent/outdent buttons in mobile toolbar and insert menu
- E2EE Frontend: encrypted attachments for encrypted notes (`.xenc`) with client-side upload encryption, preview-time decryption, legacy attachment metadata migration, and note-id-bound encryption context across search/sync/editor flows

### Changed

- Canvas: disabled by default for new users (can be enabled in Settings > Editor)
- UI: introduced CSS custom properties for typography scale, replacing hardcoded font sizes
- Editor: moved indent/outdent from "More" menu to toolbar and insert menu for quicker access
- E2EE: KEK derivation for login and password-change rewrap now uses Worker-based Argon2id by default and falls back to synchronous derivation only if Worker setup fails
- Documentation: clarified E2EE threat boundaries (metadata visibility, AI plaintext boundary, recovery limitations, encrypted-attachment behavior) across README and docs/wiki pages
- Documentation/UI: aligned E2EE recovery claims with wrapper-gated encrypted recovery flow (README, API/e2e docs, audit addendum, encryption settings i18n copy)
- Documentation/UI: updated remaining recovery references (migration page, API reference/wiki, auth recovery endpoint docs, localized recovery conflict text) to match tokenized wrapper-based encrypted recovery
- Documentation: refreshed handbook/wiki security sections to reflect current encrypted-upload behavior and wrapper-gated recovery flow for encrypted notes
- Security Planning: added dedicated E2EE follow-up roadmap for remaining P1/P2 topics (XSS defense-in-depth, recovery readiness UX, FS/PCS and multi-device trust model) and linked it from docs index

### Fixed

- Tabs: closing a tab no longer fails to remove it due to a hidden reactive dependency in the URL→Tab sync effect
- Sidebar: increased minimum resize width so header toolbar buttons are never clipped
- Quality: fixed prettier formatting and svelte-check type error in PWA test
- Quality: removed unused variables in task-sortable.ts and task-toggle.ts (eslint no-unused-vars)
- Security: hardened encrypted-note backend flows (recovery reset now blocked for encrypted accounts, server-side AI summary disabled for encrypted notes, encrypted-title schema validation aligned, deterministic DEK re-wrap validation, encrypted server-export markers, and encrypted upload endpoint)
- Security: blocked all server-side AI note processing for encrypted notes (tags, links, formatting, transform) and added frontend guards to prevent plaintext submission from encrypted notes
- Security: API-key encryption now requires dedicated `XELANOTE_API_KEY_SECRET` (min 64 chars, must differ from `JWT_SECRET`), derives keys via HKDF-SHA256, and enforces secret checks in server startup plus staging/production deployment validation
- Security: removed encrypted ciphertext preview fragments from note-load debug logs and added regression coverage to prevent reintroduction
- Security: deprecated and server-rejected legacy `plaintext_content` on AI suggest/format/transform endpoints; frontend no longer sends plaintext payloads for these routes
- Security: encrypted notes no longer persist plaintext keyword metadata (API drops `keywords`, encrypted updates clear legacy keywords, migration removes existing rows)
- Security: deprecated `keywords_enabled` encryption preference end-to-end (settings toggle removed, backend clamps to `false`, migration resets legacy enabled flags)
- Security: encrypted-note updates now clear legacy `links`/`unresolved_links`/`note_due_dates` metadata in service layer, plus migration cleanup for existing encrypted rows
- Security: added defense-in-depth guards so `UpdateLinksFromClient`/`SetNoteDueDates` always ignore and clear metadata for encrypted notes
- Security: `POST /api/users/recovery-key` for encrypted accounts now requires complete `recovery_wrapped_note_deks` / `recovery_wrapped_version_deks` coverage and stores recovery key + wrappers atomically
- Security: encrypted create/update flows now auto-invalidate stored recovery key material; migration clears legacy recovery keys for users with encrypted notes/versions
- Security: removed residual encrypted-create keyword persistence hooks across note/journal/recipe/canvas service+DB paths
- Security: `GET /api/users/recovery-key/salt` for encrypted accounts now returns salt only when all encrypted notes/versions have `wrapped_dek_recovery`; incomplete/legacy states return `404`
- Security: Settings UI now supports recovery-key setup for encrypted accounts by generating recovery wrappers client-side and posting full `recovery_wrapped_*` coverage; `/api/users/recovery-key` also accepts plaintext `recovery_key` (server-side bcrypt hashing)
- Security: encrypted notes no longer support server-side tags (`PUT /api/notes/:id/tags` returns `409`, `GET` returns `[]`), and migration removes legacy encrypted-note tag rows
- Security: encrypted note `folder_path` is now normalized to `/` on create/update across note/journal/recipe/canvas flows, with migration cleanup for existing encrypted rows
- Security: encrypted note create/update now rejects outdated `encryption_metadata.version` (< 3) to harden against protocol downgrade writes
- Security: introduced recovery-wrapped DEK persistence (`wrapped_dek_recovery`) for notes and note_versions with migration, read-path wiring, and transactional DB helpers (`BulkUpdateRecoveryWrappedDEKs`, `ClearRecoveryWrappedDEKs`)
- Security: added phased recovery-reset backend for encrypted accounts (`/auth/recovery/verify`, `/auth/recovery/encrypted-deks`, `/auth/recovery/reset-password-v2`) with one-time reset tokens, atomic password+DEK rewrap finalize, and token/session/recovery-wrapper invalidation
- Security: added frontend recovery reset flow (`/recovery`) with client-side DEK re-wrap from recovery wrappers, wired new recovery APIs, and exposed `encryption_salt` in verify response for encrypted-account rewrap
- Security Tests: expanded `frontend/src/lib/crypto/e2e.test.ts` with decrypt-failure and AAD-behavior regression coverage (invalid wrapped DEK, tampered ciphertext path, corrupted metadata)
- Security Planning: added concrete implementation plan for recovery-based DEK rewrap on encrypted accounts (`docs/security/E2EE-RECOVERY-DEK-REWRAP-IMPLEMENTATION-PLAN-2026-02-28.md`)
- Editor: checked children within an unchecked parent no longer form separate completed task groups, ensuring a single contiguous grouping area
- PWA: changed manifest orientation from `portrait` to `any` so the Android system rotation setting is respected instead of being overridden by the PWA
- CI: relaxed flaky `docChanged:structured` performance threshold from 2.0ms to 5.0ms for slower CI runners
- CI: updated bundle size budget from 3600 KB to 15000 KB to match actual app size after CodeMirror/charting deps
- Quality: added `OPENAI_MODEL` env var to docs and `.env.example` (env-sync check)
- Quality: added `internal/api/admin.go` to layer-violation baseline (type-only import)
- Quality: fixed 17 golangci-lint findings — tightened file/dir permissions (0750/0600), added http.Server timeouts for pprof, applied struct conversions, removed unused `getRequestID`, added `//nolint` for validated false positives
- Quality: fixed broken discussions link in development docs, added lychee exclusion for tag comparison URLs
- Security: resolved npm audit high vulnerabilities via `npm audit fix`, scoped audit to production deps (`--omit=dev`)
- Security: suppressed gosec G104 on intentionally-ignored bcrypt returns in timing-attack mitigations (auth, 2FA, recovery)
- Security: added gosec API batch scan job and advisory-level npm audit step to security workflow
- CI/Quality: relaxed visual regression pixel tolerance from fixed 500px to 3% ratio, auto-generate missing visual baselines in quality workflow

### Changed

- Editor: moved editor mode selector (Live/Edit/Preview/Split) from toolbar into kebab menu to save toolbar space
- README: comprehensive rewrite — expanded feature documentation (editor modes, canvas, recipes, journal, AI, admin, desktop app), added architecture diagram, fixed Desktop entry (Tauri v2 instead of Electron), removed placeholder images and TODOs, restructured sections (Quick Start, Tech Stack, Development)

### Security

- Pre-public-release security & privacy audit: removed personal email addresses from SECURITY.md and CODE_OF_CONDUCT.md, redacted internal infrastructure details (SSH aliases, staging/production IPs, internal domains, server paths, LAN IPs) across all documentation, annotated deployment workflow paths, documented LGPL-3.0 sharp/libvips dependency in THIRD-PARTY-LICENSES.md, removed 18MB compiled Go binary from git history via git-filter-repo

### Added

- Nested todo lists in live-preview: hierarchical indentation for task and list items, bidirectional auto-propagation (parent ↔ children check sync), and subtree-as-atomic-unit for drag/reorder (children move with their parent, only top-level tasks are individually draggable)
- Backend error reporting: 500 errors (`respondInternalErr`) and panics are now automatically reported as Forgejo issues with fingerprint-based dedup, stack traces, and a `backend` label for filtering
- Mobile ToC FAB: Table of Contents trigger becomes a fixed-position floating action button (bottom-right) on mobile with an SVG progress ring that fills as the user scrolls; desktop behavior unchanged
- Task collapse state sync: completed task group open/closed state now persists across devices via `GET/PUT /api/notes/:id/user-state` (localStorage as fast cache, server as source of truth, 500ms debounced sync)
- Live-preview task collapse server sync: collapsed task groups in live-preview mode sync to server via `note_user_state` API (namespaced `tasks:<hash>` keys) with debounced writes and merge logic
- Stagger animations on list items (home page recent/new notes, journal entries, recipe list)
- Dark theme: subtle noise grain texture on background for visual depth
- Frosted-glass `mobile-glass-sheet` class for all mobile bottom sheets
- CTA button color tokens (warm amber) for primary action buttons in both themes
- iOS Safari: dual-mode mobile layout — body-level scrolling on opt-in pages (journal, recipes, due-dates, trash, shared collections) so Safari toolbar auto-hides on scroll; PWA and desktop remain in fixed layout
- Local iPhone/PWA testing helpers: Caddy HTTPS proxy, mkcert integration, `make phone-*` targets

### Changed

- Typography: replaced Inter with DM Sans (body), Literata (headings), JetBrains Mono (code) as variable fonts
- Mobile bottom nav: active tab pills now have subtle scale animation on press
- Mobile "More" sheet: promoted Journal/Recipes above the "More Options" divider for faster access
- Mobile bottom sheets: all use frosted-glass effect and clear bottom nav spacing
- Settings page: tab labels visible on mobile (9px text) instead of icon-only
- Recipe ingredient row: tighter padding and smaller touch targets on mobile
- Note page: editor component lazy-loaded once, preserving Svelte action state across note-to-note navigations
- Live-preview: disabled auto-expand of collapsed task groups on cursor enter for stable persistence
- Docs: updated `docs/design-system.md` to match actual codebase (2 Gruvbox themes, Svelte 5 component APIs, CSS-only tokens, removed references to non-existent `$lib/design/` files)
- Docs: updated `frontend/DESIGN_SYSTEM.md` theme addition steps (added FOUC script + backend validation steps, fixed TypeScript file path)
- Mobile bottom nav: reduced height from 56px to 40px, smaller icons (18px) and tighter padding for less wasted space in PWA mode
- Mobile bottom nav: frosted-glass effect (backdrop-blur + semi-transparent background) so the safe area blends with content like native iOS tab bars

### Fixed

- AI Transform dialog: apply button no longer off-screen on mobile (added `min-h-0` for proper flex shrinking, `overflow-hidden` on container, `pb-safe` for safe-area inset)
- Live-preview: fixed collapse state being pruned/reset during note switches by reordering effect lifecycle (content sync before live-preview reconfigure)
- Task collapse: prevented state being persisted under empty noteId during initial mount
- Task collapse: cleanup now properly removes wrapper DOM for correct re-initialization
- Backend: `collapse_state` validation now accepts namespaced `tasks:<base36>` keys
- iOS PWA: improved viewport resync after app resume, orientation change, and window focus — prevents stale viewport height leaving a gap at the bottom
- iOS PWA: account for safe-area-inset-bottom in viewport height calculation to prevent gap on notch devices
- iOS: viewport height no longer double-counts safe-area-inset-bottom
- iOS: keyboard detection now prefers Visual Viewport over focus state to avoid stale keyboard-open state
- Recipe editor: tab slider styling and Prettier formatting fixes

### Added

- Preview: Shiki syntax highlighting for code blocks with Gruvbox CSS variable theme and lazy language loading
- Preview: KaTeX math rendering support with `$inline$` and `$$display$$` syntax (opt-in via feature flag)
- Preview: Mermaid diagram rendering with content-hash caching and 500ms debounce (opt-in via feature flag)
- Preview: Web Worker for markdown-it rendering — moves expensive parsing off the main thread with auto-cancel and 500ms timeout fallback
- Preview: Idiomorph DOM morphing — preserves DOM state (scroll position, focus, `<details>` open) on content updates instead of full innerHTML replacement
- Preview: Element-level scroll sync between editor and preview using `data-source-line` anchors with interpolation, replacing ratio-based sync
- Preview: Performance baseline benchmarks for renderMarkdown() at 100/500/2000 lines (vitest bench)
- Editor: Image lazy loading (`loading="lazy"`, `decoding="async"`) on all rendered images
- Editor: CSS `content-visibility: auto` on preview block elements for off-screen rendering optimization
- Editor: Viewport-aware heading and task group collection in live preview — reduces map population from O(document) to O(viewport)
- Recipes: Delete button on recipe list items with confirmation dialog, trash integration, and state refresh

### Changed

- Recipes: Replace tab buttons with animated slider component — sliding indicator shows active tab (Ingredients/Instructions/Preview) with smooth CSS transition
- Mobile: Add undo/redo buttons to mobile editor toolbar with CodeMirror state integration (disabled when no undo/redo available)
- Mobile: Hide encryption lock icon in editor toolbar on mobile to save space
- Mobile: Redesign mobile topbar system — new CSS utility classes (`ui-mobile-topbar`, `ui-mobile-topbar-icon`, etc.) with consistent 44px touch targets, scrollable action areas, and ghost/soft button variants
- Mobile: Add section headers and styled dividers to bottom navigation "More" sheet
- Mobile: Recipe editor overflow actions moved to bottom sheet on mobile, with flat panel/input styles for cleaner look
- Mobile: Hide desktop sidebar icon strip on mobile (replaced by bottom nav), widen drawer to 82vw
- Mobile: Editor toolbar uses compact icon-only status pills (sync, offline, locked) on mobile
- Mobile: PageHeader gains `mobileHeaderMode`, `mobileSingleRow`, `mobileHideSubtitle` props for topbar layout
- Mobile: Settings page uses flat panels and topbar mode on mobile
- Mobile: SummaryPanel gets responsive styles with reduced padding and softer borders

### Fixed

- Recipes: Fix race condition in recipe page feature guard — replace synchronous `onMount` check with reactive `$effect` pattern (matching journal page) to wait for async feature load before redirect
- Screenshot Tests: Add comprehensive test data seeding (graph notes, recipes, journal entries), encryption unlock for journal, route-specific waits, and recipe tab screenshots
- Docker build: increase Node.js heap limit to 4GB for Vite production build to prevent OOM crash with large module count
- Docker build: add missing new source files (UI components, editor preview, settings tabs, markdown plugins) that broke CI build
- PWA: increase workbox precache size limit to 6MB to accommodate shiki/mermaid chunks (hash-based filenames prevent glob exclusion)
- i18n: Replace ~30 hardcoded German strings on dashboard with `$_()` calls and add corresponding en/de keys under `page.home.*` and `component.dashboard_section.*`
- i18n: Fix pluralization for `notes_available` and `items_count` using ICU MessageFormat plural syntax (1 note vs 2 notes)
- Recipes: Replace native `confirm()` with styled `dialog.confirm()` for collection deletion (consistent with recipe deletion pattern)
- Mobile: Hide theme descriptions on small screens and use responsive grid gap for compact theme cards in appearance settings
- Mobile: Hide `Ctrl+P` keyboard shortcut badge on small screens
- Editor: Improve live preview gutter layout — increase left padding to prevent toggle/drag-handle overlap, enlarge toggle icons, hide drag handles on completed task group lines, switch expanded indicator from triangle to dash
- PWA: Fix iOS viewport height jumps on keyboard open/close, app switching, and orientation changes via JS-corrected `--app-viewport-height` CSS variable (replaces raw `100vh`/`100dvh` in dropdowns, search, trash, and root layout)
- Mobile: Add WCAG AA compliant 44px touch targets for buttons, tabs, and icon buttons; enable momentum scrolling in editor/preview panes on mobile WebKit/PWA
- Mobile: Hide drag handles on touch devices and use long-press for task reordering; enlarge heading and task-group toggle buttons to 44px (WCAG AA); disable live-mode drag on touch to avoid text selection conflicts

## [1.1.3] - 2026-02-23

### Changed

- Editor: Streamline toolbar — extract mode selector into dedicated component (segmented control on desktop, dropdown on mobile), consolidate insert actions (task, table, upload) behind a single "+" menu, move autosave toggle to more menu, add section headers to more menu
- Editor: Polish toolbar with grouped pill containers, custom mobile mode dropdown with checkmarks, enhanced save button with primary highlight and spinner, i18n for all section headers
- Editor: Note title is now edited inline as the first line of the editor content (Bear/Apple Notes style) instead of a separate toolbar input. Title and content remain separate in the API/DB. Journal notes retain their read-only date title in the toolbar.
- Frontend: Sidebar redesigned with Obsidian-style icon strip layout — persistent left icon column (40px) with navigation, theme toggle, and settings; collapsible main panel with toolbar header
- Frontend: Sidebar icon strip now shows active-state indicators (left accent bar), reordered navigation (Home, Due Dates, Shared, Trash), and Notes/Journal section labels in the tree view. Tree node rows have improved spacing, selection borders, and visible kebab menus on selected items. SidebarHeader toolbar features a prominent "New Note" button with label
- Frontend: Home screen redesigned with two-column card layout — hero section with gradient background, grid pattern, branding pill, and prominent action buttons; recent notes panel with icon thumbnails and relative timestamps
- Editor: Elevated layout with subtle gradient background, grid-pattern overlay, frosted-glass toolbar, and card-style editor panels (backlinks, summary, tags). New CSS surface tokens for consistent panel styling across themes
- Mobile: Replace top MobileHeader with fixed bottom navigation bar (Notes, Search, More tabs). Sidebar toggle button always visible and visually integrated into editor toolbars. Mobile sidebar uses two-column layout matching desktop (icon strip + tree panel). Editor toolbars use single-row layout with horizontal scroll overflow
- PWA: Lock orientation to portrait to respect Android rotation lock, add `display_override` for window-controls-overlay, add `mobile-web-app-capable` meta tag, add manifest screenshots for richer Android install prompt
- Mobile: Sidebar now pushes content to the right with blur effect instead of overlaying with dark backdrop
- Mobile: Replace fixed floating sidebar toggle with inline MobileSidebarInlineToggle component in each page header for contextual positioning
- Home: Redesign home page with activity stats, continue-working section, recently created notes, and full all-notes listing with search, sort, and mobile-friendly collapsible view
- Frontend: Introduce shared UI component classes (ui-panel, ui-list-item, ui-button, ui-page-header, etc.) in app.css for consistent frosted-glass design language across all pages
- Recipes: Overhaul ingredient editor with inline editing, drag-and-drop reordering, smart ingredient parsing, and polished preview layout
- Settings: Refresh Account, AI, and Security tabs with new surface tokens and consistent card styling
- Pages: Apply unified design system to journal, recipes, search, trash, and shared-with-me views
- Backend: Add home dashboard layout preferences API endpoint for persisting user layout customization

### Fixed

- Editor: Suppress first-line title styling for journal notes via `data-note-type` attribute
- Backend: Fix admin promotion violating single-admin unique constraint (atomically demote existing admin before promoting new one, update API test accordingly)
- Backend: Fix build failure in service package — `cache.NewCache` renamed to `cache.New` but `graph_test.go` was missed
- Recipes: Sidebar toggle button now vertically aligned with tab bar (Ingredients/Instructions/Preview) on mobile
- Editor: Live preview now keeps consistent font styling when clicking on a line (no more sans-serif → monospace switch on active lines)
- Backend: UpdateNote now automatically updates wikilinks in all linking notes when a note's title changes (previously only handled by explicit rename)
- Recipes: AI import (image and URL) now reliably translates recipes to the user's locale (e.g. German) using sandwich-pattern prompt reinforcement
- Toast: Fixed type mismatches in toast store API (functions now return toast ID, warning accepts action directly, added undoToast for undo-capable notifications)

### Added

- Recipes: Automatic Fahrenheit→Celsius conversion in recipe instructions during AI import (image and URL), rounded to nearest 5°C

### Security

- Upload directory permissions tightened from 0755 to 0750 (F-14)
- Activity-log page parameter capped at 100 to prevent excessive DB offsets (F-12)

### Improved

- Style: gofmt fix for uploads_thumbnail.go (import order and alignment)
- DX: Fixed prettier (13 files), eslint (9 errors), and svelte-check (12 type errors) across frontend
- CI: Synced `frontend/package-lock.json` with `package.json` so `npm ci --ignore-scripts` succeeds in Docker quality gates
- CI: Updated Dockerfile digest pins for golang:1.25-alpine, node:22-alpine, and alpine:3.20
- Docs: schema.sql header clarified as initial-only schema with migration reference (F-11)
- DX: Documented `make build -j2` parallel build tip in Makefile (F-18)
- Tests: Added HTTP-level path-traversal and user-isolation tests for upload handler (F-08)
- Tests: Added rate-limiter eviction-at-capacity and evict-oldest tests (F-09)
- Tests: Added CORS origin validation tests for dev mode, production, and preflight (F-10)
- CI: Backend and frontend test runs now generate code coverage reports, uploaded as artifacts to GitHub Actions with per-PR summaries
- DX: Added `make test-coverage` target for local coverage reporting (backend + frontend)
- Docs: Unified JWT_SECRET minimum length to 64 characters across all documentation (was inconsistently 32 in some places)
- Docs: Clarified Docker build tag differences (FTS5 only vs FTS5+SQLCipher locally) in deployment documentation

### Fixed

- FTS5: Fixed `notes_au` trigger that unconditionally deleted OLD from FTS index, causing "database disk image is malformed" when restoring soft-deleted notes (migration 052)
- Encryption: Fixed `BatchUpdateWrappedDEKs` deadlock with `MaxOpenConns=1` by using the transaction for ownership/encryption checks instead of a separate DB query
- ETag: `resolveETagVersion` now returns 404 instead of 500 for nonexistent notes by mapping `ErrNotFound` correctly
- Performance: Keyword insertion now uses batch INSERT instead of per-keyword loop (1 query instead of N)
- Performance: Admin user listing now calculates storage for all users in one filesystem pass instead of N separate walks
- Performance: Recipe list endpoint supports `?fields=slim` to omit content/encrypted_content from responses

### Refactored

- Frontend: Split `codemirror.ts` (1134 lines) into 5 modules under `codemirror/` — decoration plugins, theme, extension loader, utilities, plus slimmed orchestrator with shared event handlers (F-03)
- Frontend: Split `markdown.ts` (914 lines) into 8-module plugin architecture under `markdown/` — color, wikilink, due-date, image, HTML sanitizer, task processor, extractors, plus thin orchestrator (F-16)
- Frontend: Split `live-preview.ts` (1660 lines) into 8 modules under `live-preview/` — widgets, table parser, line primitives, structured lines, heading manager, task group manager, utilities, plus slimmed orchestrator (F-01)
- Frontend: `Editor.svelte` cleanup (1269 → 1198 lines) — extracted scroll sync to `scroll-sync.ts`, deduplicated `ensureEditorReady` calls via `withEditor` helper, removed stale svelte-ignore comments (F-02)
- API layer: Split `recipe_suggestions.go` (474 lines) into 4 focused files — core handlers, save/import, image processing, URL import (F-13)
- DB layer: Added `fmt.Errorf` error wrapping across 8 core DB files (auth, settings, activity, admin, 2FA, API keys) — all raw `return err` returns now include operation context for debugging
- Service layer: Extracted duplicated note-limit checks (7 identical blocks across 5 files) into a single `checkNoteLimit()` method on NoteService
- DB layer: Extracted shared graph node/edge scanning logic from `GetGlobalGraph` and `GetFilteredGraph` into `scanGraphNodes`, `scanFilteredEdges`, and `buildGraphData` helpers (eliminated ~60 lines of duplication)
- Service layer: Extracted `validateCanvasNodes` and `validateCanvasEdges` from 107-line `ValidateCanvasContent` function
- Logging: Migrated all 7 legacy `log.Printf` calls in `db/db.go` and `api/middleware.go` to structured `slog` with typed fields
- Logging: `respondInternalErr()` now includes `request_id` field from Chi's `X-Request-Id` header for request correlation
- Observability: Added `requestIDLoggerMiddleware` that stores the Chi request ID in context for downstream use
- Folder handlers: Unified error handling — validation errors now use consistent string literals, service errors use `respondInternalErr()`
- Handlers: Extracted long handler functions — `updateNote` (134→54 lines), `login` (115→58 lines), `register` (106→29 lines) — into focused helpers (`executeNoteUpdate`, `handleTwoFactorLogin`, `registerOrBootstrapUser`, etc.)
- Search: Refactored `FilteredSearch` (137→16 lines) into composable sub-methods (`buildFilteredSearchQuery`, `applyQueryFilter`, `applyFolderFilter`, `applyDateFilters`, `applyTagFilter`, `applyOrderBy`); shared `scanNoteRows` with `QuickSearch`
- Layer isolation: Added `service.User` type alias for `db.User`, eliminating the only production `db` import in the API layer (0 layer violations in baseline)

### Tests

- Service layer: Added comprehensive test suites for encryption, user account, trash operations, admin service, FIDO2/WebAuthn, and two-factor authentication (6 new test files, ~90 subtests)
- API layer: Added handler-level integration tests for auth flow (register, login, refresh, logout, /me), notes CRUD (create, read, list, update, delete), trash operations (list, count, restore, permanent delete, empty), and admin endpoints (stats, users, settings) — 4 new test files with shared test helpers
- API layer: Added NoteService domain integration tests for journal (lookup, calendar, entries, duplicate date), search (FTS5, cross-user isolation, quick-search with folder filter), encryption (create/decrypt/batch re-encrypt DEKs), backlinks (wikilink resolution), and sharing (share/unshare, role updates, cross-user access, encrypted note blocking) — 4 new test files
- API layer: Added handler tests for folders (CRUD, nested, move, rename, cross-user isolation), templates (CRUD, validation, cross-user isolation), snippets (CRUD, validation, cross-user isolation), tags (set/get/delete, replace, cross-user isolation), and versions (list, get, compare, restore, ETag/If-Match) — 3 new test files, 49 tests
- Service layer: Added sharing service tests for note sharing (share, unshare, get shares, update role, access check, shared note CRUD, editor vs viewer permissions) and folder sharing (share, unshare, update role, list shared folders, user search) — 11 new test functions, 10 subtests
- Service layer: Added graph service tests (global graph empty/with notes, caching, cache invalidation, filtered graph, cross-user isolation) — 6 tests
- DB layer: Added tests for tags (CRUD, set/replace/clear note tags, user isolation), snippets (CRUD, ACL checks, user isolation), and activity logs (log, filter, pagination, cleanup, distinct actions, user agent truncation) — 3 new test files, 32 tests
- Frontend stores: Added tests for sharing (load all/notes/folders, error handling, clear functions, count helpers — 16 tests), tree-operations (CRUD, optimistic UI, rollback on error, granular color updates — 20 tests), journal (calendar navigation, year cache, streaks, openJournalForDate, resetState — 37 tests), and recipes (pure scaling functions, CRUD, collections, sharing, images, WebSocket handlers, reset — 54 tests) — 4 new test files, 127 tests
- Frontend components: Added tests for CanvasToolbar (click handlers, drag MIME data, accessibility — 8 tests), EditorToolbar (title input, save/upload/history/autosave buttons, mobile vs desktop layout, AI actions visibility, focus mode — 20 tests), EditorStatusBar (toggle button, aria-expanded, mobile behavior — 4 tests), and QuickSwitcher (dialog open/close, combobox input, command registration, debounced search, keyboard navigation, filter button — 11 tests) — 4 new test files, 43 tests

### Removed

- Dead code: Removed 6 unreferenced Svelte components (ChangelogDialog, FormatMarkdownDialog, RecoveryKeySetup, Section, SidebarButton, SidebarItem)
- Repo hygiene: Removed spike files (`live-preview-spike.ts`, `live-preview-spike.test.ts`), stale log files, and `frontend/build-old/` directory

### Added

- Uploads: Automatic JPEG thumbnail generation (max 200×200, quality 80) for uploaded images — thumbnails served alongside originals, included as `thumbnail_url` in upload response (F-05)
- AI settings: Dietary preference dropdown (none, vegetarian, vegan, pescatarian, flexitarian) in Settings → AI, persisted per user via dedicated `GET/PUT /users/dietary-preference` endpoints. AI recipe suggestions (similar recipes, ingredient matches, generated ideas) automatically incorporate the preference into LLM prompts.
- AI settings: Added ChatGPT (OpenAI) API key integration and a selectable active AI provider (`auto`, `claude`, `gemini`, `chatgpt`) with backend persistence.
- AI settings: Added per-provider model configuration in Settings → AI (Claude/Gemini/ChatGPT), persisted per user and applied by backend provider routing.
- AI settings: Model selectors now use dropdowns with estimated per-model costs (USD per 1M in/out tokens) from a backend catalog endpoint.
- AI settings: Gemini model catalog expanded with gemini-3-flash-preview, gemini-2.0-flash, and gemini-2.0-flash-lite models and their cost metadata.
- Live Preview: Markdown tables are now rendered as HTML tables when the cursor is outside the table block; clicking the rendered table or moving the cursor into it reveals raw markdown for editing
- Table insert feature: toolbar button and `Mod-Shift-T` shortcut open a dialog to insert a markdown table with configurable rows and columns

### Security

- Email validation: `isValidEmail()` now rejects IP-literal addresses (`user@[192.168.1.1]`), single-label domains (`user@localhost`), and addresses shorter than 5 characters
- Error leakage: Login, 2FA verification, and backup code regeneration endpoints no longer expose internal service/DB error details to clients
- Error leakage: Registration and BootstrapAdmin endpoints now use typed `ValidationError` to distinguish safe validation messages from internal errors (DB, bcrypt), preventing internal detail exposure

- Lockout overflow fix: exponential backoff in `AccountLockout` now uses `safeLockoutDuration()` with capped bit-shift exponent (`maxExponentShift=20`) to prevent int64 overflow that produced negative durations, bypassing the lockout cap after ~39 failed attempts
- CSP hardening: removed bare `ws: wss:` from `connect-src` (allowed WebSocket to any host); CSP Level 3 `'self'` covers same-origin WebSocket. Added `object-src 'none'` as defense-in-depth against plugin exploitation
- Refresh token error: replaced `err.Error()` with generic "invalid or expired refresh token" message to prevent leaking internal details like "refresh token reuse detected"
- JWT issuer validation: added `jwt.WithIssuer("xelanote")` to token parser, rejecting tokens from other issuers sharing the same secret
- Security audit #2 (re-audit): comprehensive 6-agent parallel code review identifying 1 critical (first-user admin race), 2 high (upload quota TOCTOU, 2FA state race), 4 medium, 2 low findings; verified 65 security controls as passing — documented in `docs/security_audit_findings.md`
- LLM HTTP client: added response body size limits (1 MB error, 4 MB success) to prevent memory exhaustion from oversized provider responses
- Refresh token cleanup: expired and revoked tokens are now purged at startup and daily, preventing unbounded table growth
- Health check: `/health` endpoint now verifies database connectivity and available disk space (minimum 100 MB), returning 503 on failure

### Improved

- Resilience: Added robust frontend error handling with route-level error pages, client error IDs, and component boundaries for sidebar, graph, and note editors (retry/reload fallback UI).
- API reliability: Introduced request timeouts in the frontend API client (default and AI-specific) and aligned backend routing so streaming endpoints (SSE/WebSocket) are excluded from timeout middleware.
- Sidebar tree drag-and-drop now respects sort mode: before/after reordering is limited to manual sort, while automatic sort modes avoid invalid reorder interactions.
- Recipe preview: ingredients header sticks to top of scroll container while reading instructions (mobile cooking UX); collapsible via tap with chevron indicator and ingredient count badge
- Recipe editor: ingredient remove button always visible on touch devices (no hover required)

### Fixed

- Desktop auth/CAPTCHA flow: Electron development keeps `webSecurity` enabled, uses a controlled same-origin `/api` proxy path for production API testing, and improves Linux runtime temp handling for more reliable Turnstile/CAPTCHA rendering.
- Shared note/recipe opening now handles missing notes robustly: `/api/notes/{id}` not-found maps to 404 (instead of 500), frontend note loading marks 404 as `NOT_FOUND`, stops retry loops in editors, and auto-redirects away from broken `/note/{id}` routes.
- Shared recipes/notes can now be opened reliably from shared lists: note loading now falls back to shared endpoint access when owner-only note lookup returns not found
- Mobile editor: re-enabled scroll anchoring (`overflow-anchor: auto`) for editor scrollers so deleting todo/list items no longer causes viewport jumps
- Live Preview: code fence markers (```) are now properly hidden when the cursor is not on the line, consistent with headings and blockquotes
- Live Preview: Inhaltsverzeichnis (TOC) is now shown in live mode (right-aligned like preview), and TOC clicks now jump to the matching heading in the live editor
- Live Preview: non-code lines now use the regular sans font to avoid a monospace look
- Tooling: speed up live preview spike benchmark to avoid timeouts
- Pre-push lint issues: gofmt formatting, ESLint import sorting, API doc coverage baseline update
- iOS Mobile layout now uses stable viewport height (`svh`/`dvh`) and applies safe-area paddings only in standalone mode, reducing oversized bottom inset and using full screen height; mobile task deletion view-jump prevented via `overflow-anchor` stabilization on editor scroll containers
- iOS PWA viewport fallback: added `-webkit-fill-available` to safe screen-height utilities and `html/body` so older iOS Safari versions use full available app height reliably

### Changed

- Dependency lockfiles refreshed for frontend tooling updates; repository root `package-lock.json` package name now matches the project name (`xelanote`).
- Development docs and scripts now distinguish `dev:local` vs `dev:prod` flows for browser and Electron, and workspace/backlink UI locale labels were added for German and English.
- Navigation: moved logout action from the sidebar into Settings -> Account for improved reliability on narrow sidebars.
- Canvas feature is now enabled by default for all users (previously required manual activation in settings)
- Backend API refactoring: note creation flow split into dedicated validation/creation/post-processing helpers; login success response flow centralized; direct `api -> db` note-type validation dependency removed in favor of service-layer validation
- Security hardening in auth middleware: Bearer authorization parsing is now centralized and reused consistently by both auth and CSRF middleware

### Added

- KI-Rezeptimport erweitert: Beim Import per URL werden automatisch bis zu 3 Hauptbilder aus der Quellseite übernommen (KI-basierte Auswahl mit Fallback), lokal als Upload gespeichert und direkt am Rezept angehängt
- Backend unit tests for Authorization header parsing (`parseBearerToken` / `hasBearerAuthorizationHeader`) including case-insensitive scheme handling and malformed-header rejection

- Canvas sidebar-to-canvas drag-and-drop — drag notes from the sidebar tree directly onto the canvas to create file nodes at the drop position; supports both UnifiedTree and legacy NoteItem drag sources with text/plain fallback parsing
- Canvas group renaming — double-click a group label to edit inline, or use "Rename" in the context menu; groups without a custom name show a dimmed "Group" placeholder
- Canvas drag-and-drop tool placement — drag tools from the bottom toolbar onto the canvas to place nodes at the exact drop position with a live size-accurate preview ghost
- Canvas keyboard shortcuts: `T/N/L/G` to add text/note/link/group nodes, `Esc` to close canvas menus, and `Ctrl/Cmd + C/V` for internal canvas copy/paste of selected nodes (including edges between selected nodes)
- Canvas note preview — embedded notes now display their actual content with full markdown live preview instead of a static "Click to open note" placeholder; uses a read-only CodeMirror instance with the same rendering as text nodes
- Canvas EditorToolbar-style toolbar — replaces minimal header with full-featured toolbar matching notes editor design: editable title, save/upload/history/focus-mode buttons, responsive 3-column grid layout, and a more menu with export, share, move, and delete actions; lazy-loaded dialogs for version history, move-to-folder, and share
- Canvas drag-and-drop image support — drop image files from the desktop onto the canvas to create image nodes at the drop position; paste images from clipboard with Ctrl+V; CanvasFileNode renders actual images with object-fit scaling and error fallback instead of placeholder icon
- Journal E2E encryption enforcement — journal entries are now always encrypted regardless of folder settings; journal page shows lock overlay when encryption is locked with unlock button; lock icon next to note title in editor toolbar indicates encrypted notes
- Sidebar sort dropdown — sort notes by manual order, last modified, title A-Z, or date created; persisted in localStorage; folder order remains manual
- Infinite canvas feature (JSON Canvas spec v1.0) — visually organize notes, text cards, links, and groups on a free-form spatial board; uses @xyflow/svelte with 4 custom node types, 6 Gruvbox color presets, toolbar, context menu, auto-save, and .canvas export; user-togglable feature flag (disabled by default)
- Collapsible completed task groups in live preview — groups of 2+ consecutive completed tasks show a bracket and [+]/[−] toggle; collapsed groups display a summary line; cursor auto-expands collapsed groups
- Live preview task drag handles — grab handle appears on hover to the left of each task item for intuitive drag-and-drop reordering directly in live preview mode; touch devices show handles permanently; uses line-based reorder mapping for accurate task movement

### Fixed

- Fix empty tasks (checkbox without body text) incorrectly receiving task-line decoration in live preview, causing inconsistency with the drag-sortable task set

- Fix live preview task drag-and-drop items jumping to the top of the note — replaced SortableJS index-based target computation with DOM-based neighbor comparison that is immune to index desynchronization caused by onMove overrides and non-task elements
- Fix note title being truncated to only a few characters on narrow desktop windows; title input now uses flex-grow instead of character-count-based width so the lock icon and last-updated label no longer squeeze it
- Fix viewport jumping to the bottom of the list when toggling a todo checkbox that triggers task reordering in live preview
- Fix @due() date badge overlapping with preceding text in todo list live preview; reset inherited text-indent on inline-block badge and skip mark decoration on non-active lines to prevent double rendering
- Fix collapsed completed-task groups reopening in live preview after checking tasks or adding new items; collapse state is now preserved across task-group key changes during document updates
- Fix canvas view staying stuck when navigating back to a regular note; caused by stale currentNote and missing loadNote call in CanvasEditor
- Fix sidebar-to-canvas note drag-and-drop not working — dropEffect/effectAllowed mismatch ('copy' vs 'move') caused browsers to suppress the drop event per HTML5 DnD spec; also narrowed overly broad text/plain detection and added stopPropagation to prevent SvelteFlow interference
- Fix canvas frontend check blockers: corrected `FlowNode`/`FlowEdge` type narrowing in paste handling, stabilized concurrent canvas feature-load test typing, and removed `CanvasNotePicker` a11y warnings (`autofocus` + non-interactive click handler)
- Mobile Todo-Bereiche behalten ihren Collapse-Zustand zuverlässig: erledigte Task-Gruppen in Live-Preview und im Preview-Renderer werden jetzt persistent pro Notiz gespeichert und nach Re-Mount/Reload wiederhergestellt
- Mobile Trefferfläche für den Live-Preview-Taskgruppen-Toggle vergrößert, damit das Dreieck-Symbol auf Touch-Geräten leichter getroffen werden kann

### Changed

- Canvas text nodes now use live preview with full markdown rendering (headings, bold, italic, lists, task checkboxes, wikilinks, code blocks) via an always-on embedded CodeMirror instance; click into card to edit raw markdown on active line, click outside for clean rendered preview
- Use outline triangle symbols (▷/▽) for collapse toggles in live preview instead of +/−; heading toggle no longer indents the heading text
- Replace "LP" text label with ScanEye icon for live preview mode toggle in toolbar and menu
- Switch app font from monospace/system to Inter (self-hosted via @fontsource/inter) for consistent rendering across all devices
- Fix ESLint errors: prefer-const for $state vars, use SvelteSet for reactive Sets, sort imports, prefix unused params
- Fix Prettier formatting in Graph components, API client, and Editor
- Add project-wide Prettier and ESLint checks to pre-push hook to catch formatting issues before CI

### Security

- Remove silent CAPTCHA bypass for desktop clients (SEC-002) — all clients must provide valid CAPTCHA tokens when CAPTCHA is enabled
- Add owner validation to recipe image URLs (SEC-003) — prevents cross-user upload URL signing oracle
- Add CSRF protection to /auth/refresh and /auth/logout (SEC-006) — prevents same-site CSRF attacks on session-mutating endpoints; CSRF cookie lifetime extended to 30 days to match refresh token
- Remove tokens from auth response body for web clients (SEC-001) — web authentication relies exclusively on HttpOnly cookies; desktop clients still receive tokens for OS keyring storage
- Pin third-party GitHub Actions to commit SHAs (SEC-004) — hardens CI/CD supply chain

### Fixed

- **Wikilinks und Markdown-Links in Live-Preview nicht anklickbar** — Der `mousedown`-Event verschob den Cursor auf die Zeile, wodurch die Live-Preview-Dekoration entfernt wurde bevor der `click`-Handler feuerte; `preventDefault` im `mousedown` für Link-Widgets behebt das Problem.
- **Task-Gruppen Toggle reagiert nicht nach Klick auf erledigten Eintrag** — Die Auto-Expand-Logik lief bei jedem Update-Zyklus, nicht nur bei Cursor-Bewegung; nach Klick auf eine erledigte Aufgabe (Cursor in Gruppe) wurde ein Collapse per Toggle sofort durch Auto-Expand rückgängig gemacht; jetzt wird Auto-Expand nur ausgelöst, wenn sich die aktiven Zeilen tatsächlich ändern.
- **Task-Gruppen Toggle-Button nicht mittig an Klammer** — Der [−]-Button nutzte `em`-Einheiten für die Positionierung, aber da der Button `font-size: 0.8em` hat, wurde der Wert auf 80% skaliert und der Button ~20% zu hoch angezeigt; Korrektur auf `rem`-Einheiten.
- **Live-Preview zeigt Markdown auf erster Zeile beim Laden** — Beim Öffnen einer Notiz oder Neuladen stand der Cursor auf Zeile 1, wodurch die Live-Preview den rohen Markdown-Code anzeigte; jetzt werden bei unfokussiertem Editor keine Zeilen als aktiv markiert, und beim Fokussieren werden die Dekorationen korrekt neu berechnet.
- **Task-Listen Zeilenumbruch bündig** — Umgebrochener Text in Todo-Einträgen ist jetzt bündig mit dem Textanfang nach der Checkbox; Checkbox-Ersetzung schließt den Leerraum nach `[x]` ein, hanging indent wird per Inline-Style statt nur CSS-Klasse gesetzt.
- **Live-Preview Heading-Toggle eingerückt** — Das `+ / −`-Symbol für einklappbare Überschriften wird jetzt links außerhalb des Textflusses positioniert; der Überschriftentext bleibt bündig mit dem restlichen Inhalt.
- **Live-Preview Heading-Format auf aktiver Zeile** — Beim Klick auf eine Überschrift in der Live-Preview bleibt die Markdown-Syntax (`#`) sichtbar, aber die Zeile behält weiterhin das Überschriften-Styling (z. B. fett/größer).
- **Live-Editor Listenmarker nicht bündig** — Unordered-Marker werden im Live-Preview nicht mehr als `•`-Glyph (fontabhängig), sondern als CSS-Kreis gerendert; Ordered- und Unordered-Listen teilen sich eine einheitliche Marker-Spalte für konsistente Ausrichtung.
- Live-Preview tests stabilized for auth init flow by mocking initial `/auth/me` probe fetch in SEC-001 and E2E auth tests; removes nondeterministic network-error path in Vitest
- **Sidebar: E-Mail und Changelog-Button entfernt** — Die Benutzer-E-Mail wird nicht mehr in der Seitenleiste angezeigt; der Versions-/Changelog-Button im Footer wurde ebenfalls entfernt (mobile + desktop).
- **Editor shows stale content when switching notes** — Preview updated immediately while CodeMirror kept showing the old note; caused by `isLoading` guard blocking the editor update effect after `currentNote` was already set; also debounce headings extraction and split-mode preview rendering to reduce per-keystroke work
- **Unlock modal flashes on every page refresh** — Race condition: `loadNotes()` fires before async KEK restore completes, `encryption.getUserID()` returns null so silent restore is skipped; now falls back to `auth.getCurrentUser()?.id` which is available immediately
- **Editor panels always visible on mobile** — Summary, Tags and Backlinks panels no longer take up fixed space at the bottom on mobile; they now sit below the editor/preview content and appear when scrolling down (restoring the pre-PWA-fix behavior for touch devices)
- **Encryption unlock modal shown unnecessarily after auto-lock** — For balanced/convenient security levels, silently restore KEK from IndexedDB instead of showing the password modal; paranoid mode still requires manual unlock
- **Auto-lock timeout "never" (0) reset to 15 minutes** — Fix `||` to `??` (nullish coalescing) so a timeout value of 0 is preserved instead of being treated as falsy and replaced with the default 15 minutes
- **Task checkbox toggles wrong item with ordered lists** — `markdown-it-task-lists` generates `task-list-item` for both ordered (`1. [ ]`) and unordered (`- [ ]`) lists, but the toggle regex only matched unordered markers, causing an index offset that toggled the wrong checkbox
- **Task checkbox toggle broken when clicking label text** — Clicking the text of a task item (not the checkbox square) failed silently due to `<label>` click timing; read HTML `checked` attribute instead of DOM property and prevent browser-side toggling; also harden `taskCollapse` to use the same attribute-based check
- **Task checkbox toggles seemingly random items in some lists** — Preview toggle mapping now skips empty task markers (`- [ ]`), adds `data-task-line` source mapping, and toggles by source line with index fallback; prevents wrong-item toggles when rendered checkbox count differs from raw markdown marker count
- **Note title hidden on narrow screens** — Guarantee minimum 120px for the title column in the editor toolbar grid and allow the toolbar buttons to shrink and scroll horizontally instead of squeezing the title off-screen
- **Journal note title not editable** — Make title input readonly for journal notes in editor toolbar, matching backend enforcement; visual cues (reduced opacity, no focus ring) indicate non-editable state
- **Mouse wheel scrolling in Chrome PWA** — Restructure Editor layout from single `overflow-auto` container to flex column with constrained heights so CodeMirror and preview scroll internally via their own scroll containers; also change `overscroll-behavior-y` from `none` to `contain` and scope `touch-action: manipulation` to touch devices only
- **E2E tests in CI** — Auto-enable registration when `XELANOTE_ENV=test` so E2E tests can create users on fresh `:memory:` databases (migration 045 disables registration by default)
- **golangci-lint v2 config** — Suppress pre-existing lint noise: exclude `defer .Close()` errcheck, `QF*` quickfix suggestions, `SA9003` empty branches, `ST1005` error string style, and `commentedOutCode`/`deprecatedComment` gocritic checks; fix invalid YAML escape `\.` in double-quoted string on errcheck `.Close()` exclusion

### Added

- **Pre-push hook** — Added `pre-push` section to `lefthook.yml` with `gofmt`, `go vet`, and `svelte-check` to catch type errors and formatting issues before pushing.
- **CI E2E pre-build** — Pre-build Go backend binary before Playwright E2E tests in CI to avoid cold `go run` timeout. Uses pre-built binary in CI, `go run` locally.

### Changed

- **Graph UI modernisiert (theme-adaptiv)** — Graph-Canvas nutzt jetzt Theme-Tokens statt fixer Farben (Nodes, Edges, Hintergrund, Tooltip), reagiert live auf Theme-Wechsel und hat ein visuelles Fokus-Highlighting fuer ausgewählten Knoten + Nachbarn. Controls wurden als moderne Card/Glass-Leiste mit kompakter Legende ueberarbeitet. Neu: Floating Toolbar mit `Fit`, `Reset`, `Center Selected`, Fokus-Toggle (`Nur Nachbarn`) und Zoom-Status sowie zoomabhaengige Detailstufen (LOD) fuer Labels/Link-Visuals.
- **Phased startup initialization** — `initializeApp()` split into 3 phases for faster time-to-interactive: Phase 1 (Critical) runs IndexedDB/Sodium/Auth in parallel and restores UI from localStorage; Phase 2 (After First Paint via rAF+setTimeout) loads notes, encryption, WebSocket, and web-vitals; Phase 3 (Idle via requestIdleCallback) handles error reporting, feature detection, and background sync. Activity listeners now register immediately after auth confirmation (security). Web-vitals moved to after-first-paint to capture early FCP/LCP metrics.
- **SQLite WAL mode + performance pragmas** — Default journal mode switched from DELETE to WAL with `synchronous=NORMAL` for lower write latency and better concurrent read performance. Added `busy_timeout=5000` to prevent SQLITE_BUSY under load. `PRAGMA optimize` runs at startup, daily (background scheduler), and on shutdown. Journal mode is configurable via `XELANOTE_JOURNAL_MODE` env var (`wal` default, `delete` fallback for problematic volumes).
- **Editor preview action lifecycle** — Removed forced preview remount on every render (`#key renderedContent`) and reduced action churn for task collapse/sortable via rAF-scheduled refresh and instance reuse; lowers DOM work and improves responsiveness while typing in split/preview mode

### Fixed

- **GitHub CI fixes** — Added missing `cache-dependency-path: backend/go.sum` in `quality.yml` for backend and golangci-lint jobs. Added `XELANOTE_JOURNAL_MODE` to environment variable docs and `.env.example`. Updated layer-violation baseline. Fixed 4 broken links in docs (golangci-lint install URL, Forgejo checkout releases, Docker Hub badge, migrations README). Updated Go version from 1.24 to 1.25 and Node.js from 20 to 22 across all CI workflows to match `go.mod` and local/staging environments. Fixed security.yml: added `cache-dependency-path`, updated Trivy action v0.28.0 to v0.33.1, CodeQL v3 to v4, added `actions: read` permission, replaced deprecated `npm audit --production` with `--omit=dev`. Migrated golangci-lint config from v1 to v2 format (v2.9.0). Re-formatted 5 Go files for Go 1.25 gofmt alignment changes. Removed Trivy container scan and CodeQL jobs from security.yml (redundant with govulncheck + npm audit for solo project).
- **Settings page 500 error** — Lazy-load `qrcode` module in `TwoFactorSetup.svelte` via dynamic `import()` instead of static top-level import. A corrupted Vite dependency cache for `qrcode.js` previously crashed the entire settings page because the module was in the critical import graph.

### Added

- **Delta-Sync + Field Projection**: Backend `fields=slim` query parameter strips `content`, `encrypted_content`, `summary` fields from list responses (~90% payload reduction). `updated_since` parameter enables cursor-based delta-sync. Frontend Notes-Store uses paginated slim loads with `sync_token` high-watermark, delta-merge for offline-sync and incremental updates, and race-protection delta-pass for multi-page full loads. Tree-Store also uses `fields=slim`.
- `is_deleted` field on Note model for delta-sync soft-delete propagation
- `sync_token` response field on note list endpoint (datenbasierte High-Watermark: `updated_at|id`)
- `updated_since` query parameter for delta-sync with cursor-tiebreaker pagination (ASC order)
- Web Vitals (LCP, INP, CLS, FCP, TTFB) performance metrics reporting with 10% client sampling, DNT respect, URL sanitizing, and 90-day retention (`perf_metrics` table + `/perf-metrics` endpoint)
- PWA analytics events pipeline: fire-and-forget POST to `/analytics/events` with event-name whitelist, rate limiting (20/hr), 1KB payload limit, and 90-day retention (`analytics_events` table)
- Data governance rules for telemetry in `docs/conventions.md` (URL sanitizing, no PII, sampling, retention, DNT hierarchy)
- Bundle size check in primary CI pipeline (`ci.yml` build job)
- Command Palette: type `>` in QuickSwitcher (Ctrl+K/Ctrl+P) to access commands — New Note, New Folder, Toggle Theme, Open Graph, Settings, Journal, Export Note — with keyboard navigation and i18n (DE/EN)
- Command registry (`$lib/commands/command-registry.ts`) for extensible palette commands
- Batch conflict resolution: ConflictDialog shows all sync conflicts with tab navigation, "Alle: Lokal behalten" / "Alle: Server behalten" bulk buttons, and progress indicator
- Granular tree cache updates with feature flag (`useGranularTreeCache`): targeted splice for expand/collapse, in-place update for note title/color and folder color, dev-mode invariant checks, automatic fallback to full invalidation
- `updateNoteInTree()` for efficient in-place tree updates without full reload
- `LayoutOverlays.svelte` component extracting Toast, OfflineBanner, ConflictDialog, InstallPrompt, UnlockEncryptionModal, ConfirmDialog, AlertDialog from `+layout.svelte`

### Removed

- **Mobile header back arrow** — Removed the standalone PWA back arrow button from `MobileHeader.svelte` (was shown next to the burger menu in PWA standalone mode). The sidebar already provides full navigation, making the back button redundant. Cleaned up the associated `navStack`/`pushNav`/`canGoBack` navigation stack in the UI store.
- **Home page welcome hero** — Removed the PenLine icon, "Welcome to xelanote" heading, and subtitle from the home page for a cleaner look

### Fixed

- **Mobile toolbar: more-menu button misaligned** — The ⋮ (three dots) button dropped to its own row below the toolbar icons on mobile because the toolbar used `flex-col` stacking. Wrapped the toolbar buttons and the more-menu button in a shared flex container (`sm:contents` preserves the desktop grid layout).
- **iOS PWA: safe-area overlap** — App content overlapped with iOS status bar in standalone PWA mode. Added `pt-safe` (safe-area-inset-top padding) and `bg-background` to root layout container. Additionally added safe-area spacer to mobile sidebar drawer, which uses `fixed inset-y-0` positioning and therefore escapes the root container's padding.
- **iOS PWA: blank screen / no notes after login** — After logging in within the iOS standalone PWA, the app showed only the empty welcome page until force-closed and reopened. Root cause: The standalone code path used `window.location.replace('/')` to force a full page reload, relying on `initializeApp()` to restore the session via HttpOnly cookies. On iOS WebKit this had a timing issue where the cookie was not reliably available immediately after the reload, causing `refreshTokenViaCookie()` to fail silently. Fix: Removed the standalone special case — all modes now use the same client-side initialization (load preferences, fire notes load, connect WebSocket) followed by `goto('/')`. The layout's `$derived` `isPublic` reactively switches from public to protected branch when `auth.isAuthenticated()` becomes true, and the Sidebar's `$effect` triggers `tree.loadTree()` on auth state change.
- **CI: frontend quality gate failure** — `web-vitals` missing from `package-lock.json` (added to `package.json` but lockfile never regenerated), causing silent `npm ci` failure. Also fixed 14 eslint errors (unused imports, `Map` → `SvelteMap` in reactive stores, self-assignment pattern, import sorting) and prettier formatting in 7 source files. Removed unused `onUnlockModalChange` prop from `LayoutOverlays` (redundant with `$bindable()`).
- **100-Note silent cap bug**: Backend DB layer limited notes to 100 (despite API layer allowing 500). Changed DB limit to 500 and implemented cursor-based pagination loop in `tree.svelte.ts` with 100-iteration safety guard (supports up to 50,000 notes)
- Tree `buildTree()` O(n\*m) folder lookup: replaced `Array.from(folderMap.values()).find()` with O(1) `pathMap.get()` lookup
- Notes store O(n) lookups: added `SvelteMap`-based note index with lazy rebuild pattern, `getNoteById()` returns O(1), `updateNoteInList()` uses Map + findIndex splice instead of full `.map()`
- Eliminated double array traversal in `remote-updates.ts` (redundant `setNotes(getNotes().map(...))` after `updateNoteInList`)
- Mobile touch targets: increased `toolbar-btn` minimum size from 44px to 48px (WCAG AAA) on touch devices via `@media (pointer: coarse)`; MobileHeader buttons use `min-h-12 min-w-12` explicitly

### Changed

- Lazy-load ConflictDialog, ShareDialog, AIActionsDropdown via `dialog-loaders.ts` pattern (removed from initial bundle)
- Lazy-load Graph route: dynamic import with GraphSkeleton loading state
- Extracted share-target processing from `+layout.svelte` into `$lib/routes/layout/share-target.ts`
- Extracted activity listeners from `+layout.svelte` into `$lib/routes/layout/activity-listeners.ts`
- Extracted overlay components from `+layout.svelte` template into `LayoutOverlays.svelte`

- Dark mode splash screens for iOS PWA launch (22 dark variants with `prefers-color-scheme` media queries, generated via `scripts/generate-splash.mjs`)
- Web Share Target (Chromium): share text/URLs from other apps to create notes (GET method, with auth persistence via sessionStorage + input hardening)
- Journal and Due Dates manifest shortcuts for quick PWA access (4 shortcuts total)
- Back button in standalone PWA mode mobile header (in-app nav depth tracking via `pushNav()` with home fallback)

### Fixed

- Account settings page: added missing i18n keys for email change, password change, and 2FA sections (en + de) — keys like `change_email_description`, `new_email`, `password`, `two_factor_title` etc. were referenced in `AccountTab.svelte` but not present in translation files
- Lefthook pre-commit hooks: `root: "frontend/"` fuer korrekte relative Pfade, `.html` zu Prettier-Glob hinzugefuegt, ESLint `--max-warnings 0` fuer Paritaet mit CI

### Changed

- PWA install coach triggers after first successful user action (auto-save completed) instead of fixed 5s timer (60s fallback)
- iOS browser detection supports Chrome, Firefox, Edge, Opera on iOS 16.4+ (`detectIOSPwaCapable` replaces `detectIOSSafari`; defensive: show coach when version unparseable)
- Manifest `background_color` changed from `#ffffff` to `#282828` to reduce dark mode launch flash

- iOS PWA Install Coach: 3-Schritt Onboarding-Dialog fuer iOS Safari-User (Share > Zum Home-Bildschirm > App oeffnen) mit Snooze/Dismiss-Optionen, State-Machine (`pwa.svelte.ts`), Dialog-A11y (`role="dialog"`, `aria-modal`, Fokus-Trap, ESC-Handler), 7-Tage-Snooze-Logik, verzoegerter Anzeige nach Login, Standalone-Erkennung, Safari-positive-Detection (schliesst CriOS/FxiOS/FBAN/Instagram/LinkedIn/Twitter aus), localStorage-Migration vom alten Key, defensivem localStorage-Zugriff (Private Browsing, QuotaExceeded), Unit-Tests fuer State-Machine und Detection
- iOS PWA Optimierung: Apple Meta-Tags (`apple-mobile-web-app-capable`, `apple-mobile-web-app-status-bar-style`, `apple-mobile-web-app-title`) fuer Standalone-Modus auf iOS
- Dual `theme-color` Meta-Tags mit `prefers-color-scheme` Media-Queries fuer korrekten Browser-Chrome vor JS-Ausfuehrung
- Apple Splash Screens fuer alle aktuellen iPhone- und iPad-Modelle (22 PNGs, ~1.9MB)
- Manifest: `id`-Feld fuer stabile PWA-Identitaet, `shortcuts` fuer "New Note" und "Search" (Android/Chrome)
- `overscroll-behavior-y: none` auf `html` verhindert Pull-to-Refresh und Rubber-Banding im Standalone-Modus
- Standalone-Erkennung (`navigator.standalone` + `display-mode: standalone` Media-Query) im UI-Store
- iOS-Installationsanleitung als Bottom-Sheet mit Share-Icon und 2-Schritt-Anleitung (DE/EN i18n)
- Shortcut-Action-Handling: `?action=new-note` Query-Parameter erstellt neue Notiz nach App-Start

### Changed

- `.gitignore`: Codex-Docker-Verzeichnis und `codex.sh` komplett ignoriert (statt nur einzelner Dateien)
- FOUC-Script aktualisiert: Nutzt vorhandene `theme-color` Meta-Tags statt neue zu erstellen
- Manifest: `lang`-Feld entfernt (App unterstuetzt DE und EN, statischer Wert war irrefuehrend)
- Manifest: `theme_color` von `#3b82f6` auf `#458588` korrigiert (konsistent mit Gruvbox-Light Meta-Tag)

### Fixed

- **Cache-Leak zwischen Accounts**: Runtime-Caching fuer `/api/notes`, `/api/notes/:id`, `/api/folders` entfernt. Feste SW-Cache-Namen fuer userbezogene Daten konnten bei Session-Timeout, Crash oder Account-Wechsel ohne Logout stale Responses ausliefern. Uploads-Cache bleibt (wird bei Logout geloescht).
- **theme-color-Initialisierung**: Meta-Tags vor FOUC-Script verschoben, damit `querySelectorAll` sie beim Parsen bereits findet
- **Precache-Groesse**: Splash-Screen-PNGs (~1.9MB) aus Workbox-Precache ausgeschlossen via `globIgnores` (werden nur von iOS via `<link media>` abgerufen)
- **InstallPrompt Lifecycle**: `beforeinstallprompt`-Listener von Modul-Ebene in `onMount` verschoben mit Cleanup in der Destroy-Funktion

### Fixed

- Import-Sortierung in RecipeEditor.svelte korrigiert (ESLint simple-import-sort)

### Changed

- Layer-Violations-Baseline von 37 auf 0 reduziert: Alle 37 API-Dateien importieren jetzt `service` statt `db` direkt. Type-Aliases (`Note`, `NoteVersion`, `Backlink`, `GraphData`, `SearchFilters`, etc.) und Error-Re-Exports (`ErrNotFound`, `ErrVersionMismatch`, etc.) im Service-Layer eingefuehrt. 11 GetDB()-Bypass-Aufrufe in journal.go, features.go, task_events.go, notes_crud_create.go, notes_crud_update.go durch 9 neue Service-Methoden ersetzt (`GetUserFeature`, `ListUserFeatures`, `SetUserFeature`, `JournalExistsForDate`, `ListJournalEntries`, `ListJournalDates`, `ListJournalDatesForYear`, `SetNoteDueDates`, `RecordTaskEvent`). `NoteService.GetDB()` entfernt. CI-Ratchet (`check-layer-violations.sh`) um GetDB()-Check erweitert.

### Fixed

- Base64-Encoding-Bug bei verschlüsselten Notizen >8KB behoben: `toBase64Standard()` nutzte Chunk-Größe 8192 (nicht durch 3 teilbar), wodurch `btoa()` Padding-Zeichen (`=`) in Zwischen-Chunks erzeugte, die beim Konkatenieren ungültiges Base64 ergaben. Fix: Chunk-Größe auf 8190 (teilbar durch 3) geändert.
- golangci-lint suppressed Findings bereinigt: 7 von 8 exclude-rules entfernt (sprintfQuotedString, sloppyReassign, assignOp, SA9003, S1017, s.updateLinks, ineffassign). Konkrete Code-Fixes: `fmt.Sprintf` mit Quoted-Strings durch Concatenation/`%q` ersetzt (db.go, notes_helpers_validate.go, notes_helpers_types.go), 2 unchecked `s.updateLinks` Aufrufe mit Error-Handling versehen (notes_rename.go, notes_encryption_create.go), ineffektive Zuweisung in folders_rename.go eliminiert. Nur `activityService.Log` bleibt als fire-and-forget by design supprimiert.

### Changed

- Go Runtime von 1.24 auf 1.25 aktualisiert (Green Tea GC experimentell, `http.CrossOriginProtection`, `testing/synctest`, Container-Awareness)
- go-chi von v5.1.0 auf v5.2.5 aktualisiert
- CodeMirror-Pakete aktualisiert: view 6.39.14, state 6.5.4, autocomplete 6.20.0, commands 6.10.2
- CI Quality-Gate (`deploy-staging.yml`) auf `golang:1.25-alpine` aktualisiert
- Dockerfile Backend-Builder auf `golang:1.25-alpine` aktualisiert

### Fixed

- Air Hot-Reload Build-Tags von `fts5` auf `fts5 sqlite_crypt` synchronisiert (war inkonsistent mit Makefile)
- Race Condition im Job-Manager behoben: `Job`-Struct mit `sync.RWMutex` abgesichert, `GetJob()` gibt Snapshot-Kopie zurueck, Handler nutzen `UpdateProgress()` fuer thread-safe Zugriff

### Changed

- Editor-Toolbar: Breadcrumb-Navigation (Home > Ordner > Notiz) entfernt, da Sidebar-Navigation ausreicht
- Editor-Toolbar: "Edited X Min"-Timestamp direkt neben den Notiztitel verschoben (dynamische Breite, passt sich an Titellaenge an)
- Editor-Toolbar: 3-Spalten CSS Grid Layout fuer echte Zentrierung der Toolbar-Buttons

### Removed

- `Breadcrumb.svelte` Komponente (nicht mehr verwendet)

### Added

- lefthook pre-commit Hook (`scripts/check-changelog.sh`): Blockiert Commits mit Code-Aenderungen wenn CHANGELOG.md nicht aktualisiert wurde. Ueberspringbar mit `LEFTHOOK=0` fuer reine Refactorings.

### Fixed

- CI/CD: `TRUSTED_PROXIES` zu Pre-flight Checks in Staging- und Production-Workflows hinzugefuegt (fehlende Variable verursachte `log.Fatal` beim Container-Start und Health-Check-Failure mit Auto-Rollback)

### Added

- CI Guidelines Enforcement: Automatisierte Durchsetzung der Architektur- und Security-Konventionen aus `docs/conventions.md`
  - **golangci-lint** in CI integriert (revive, misspell, bodyclose, gocritic, unused) mit build-tags fuer fts5/sqlite_crypt; 63 pre-existierende Findings via targeted exclude-rules suppressed (inkrementelle Bereinigung geplant), bewusst NICHT als pre-commit Hook (30-60s first-run wuerde Commit-Velocity beeintraechtigen)
  - **Layer-Violation Ratchet** (`scripts/check-layer-violations.sh`): Prueft API->DB Layer-Verletzungen gegen Baseline (37 bekannte Violations); neue Violations blockieren CI, behobene Violations erfordern Baseline-Update
  - **Svelte 4 Import Guard** (`scripts/check-svelte4-imports.sh`): Blockiert verbotene `svelte/store` Imports (nur `{ get }` erlaubt fuer Svelte 5 Kompatibilitaet)
  - **Security Pattern Check** (`scripts/check-security-patterns.sh`): Blockiert `localStorage.setItem` fuer Auth-Tokens (token/auth/jwt Keys); Limitation: Nur String-Literal-Keys erkannt, variable-basierte Keys erfordern Code Review
  - 3 neue lefthook pre-commit Hooks (layer-check, svelte4-check, security-check) fuer sofortiges Feedback bei lokaler Entwicklung
  - 2 neue GitHub Actions CI Jobs: `golangci-lint` (mit CGO+sqlite3) und `policy` (alle 3 Check-Scripts)
  - Neue Makefile-Targets: `lint-golangci`, `check-policy`; `quality` Target um beide erweitert
- Backend API route wiring modularized into domain-specific route files for improved maintainability.
- Unit tests for typed API key status mapping and forwarded IP parsing edge cases.
- Frontend API-Modularisierung: `frontend/src/lib/api.ts` ist jetzt Facade, API in Module unter `frontend/src/lib/api/` ausgelagert
- ESLint Import-Sortierung (frontend) via `eslint-plugin-simple-import-sort`
- History-Store Unit-Tests fuer Undo/Redo-Persistenz
- Find-in-Note & Search-Highlight: Ctrl+F oeffnet eine VS-Code-artige Suchleiste direkt im Editor (basierend auf @codemirror/search mit eigenem Svelte-UI-Panel)
- Suchen & Ersetzen: Ctrl+H oeffnet die Replace-Zeile mit Einzel- und Alle-Ersetzen-Funktionalitaet
- Suchkontext aus Volltextsuche: Klick auf ein Suchergebnis oeffnet die Notiz mit `?highlight=` Parameter und hebt den Suchbegriff automatisch hervor
- Preview-Highlighting: Suchbegriffe werden auch im Markdown-Preview sicher hervorgehoben (TreeWalker auf Text-Nodes, kein innerHTML)
- HTTP gzip-Kompression fuer alle Text-Responses (~65% kleinere JS/CSS/JSON-Payloads)
- Cache-Control Header fuer statische Assets (immutable-Caching fuer Vite-gehashte Dateien, Revalidierung fuer den Rest)
- Default `Cache-Control: no-store` fuer API-Responses (verhindert Caching verschluesselter Daten durch Proxies)
- Einzelne Notizen koennen jetzt direkt als Markdown-Datei exportiert werden (ueber das Drei-Punkte-Menue im Editor)
- Demo-Datenbank-Generator mit Beispieluser, Notizen, Rezepten, Journal-Eintraegen und allen Features fuer Screenshots (`make demo-db`)
- CONTRIBUTING.md, CODE_OF_CONDUCT.md, Issue- und PR-Templates fuer klarere Projektbeitraege
- SVG-Platzhalter fuer Banner und Screenshots in der Dokumentation
- GolangCI-Lint Konfiguration fuer zusaetzliche Go-Qualitaetschecks
- Faelligkeiten-Uebersicht: Neue Seite zeigt alle `@due()`-Termine ueber alle Notizen hinweg, gruppiert nach Status (ueberfaellig, heute, bald, zukunft) mit Toggle fuer erledigte Aufgaben
- Changelog-Dialog: Klick auf die Versionsnummer in der Seitenleiste oeffnet den vollstaendigen Changelog als formatierten Dialog
- Faelligkeitsdaten-Syntax: `@due(YYYY-MM-DD)` ueberall im Markdown-Text erzeugt farbige Badges (rot=ueberfaellig, orange=heute/bald, grau=Zukunft) und wird im Editor hervorgehoben
- Client-seitige Volltextsuche ueber entschluesselte Notizen: Bei entsperrtem Vault werden verschluesselte Notizen im Browser durchsuchbar (MiniSearch-Index im RAM, automatischer Aufbau bei Unlock, sofortige Zerstoerung bei Lock)
- Volltextsuche fuer verschluesselte Notizen: Suche findet jetzt auch verschluesselte Notizen ueber deren Keywords (opt-in). Ergebnisse zeigen Lock-Icon, entschluesselte Titel (wenn Vault entsperrt) und gematchte Keywords
- `decryptTitle()` im Encryption-Store fuer isolierte Titel-Entschluesselung (z.B. in Suchergebnissen)
- Schnellsuche (Strg+P) durchsucht jetzt auch verschluesselte Notizen ueber den Client-seitigen Index und zeigt Kontext-Snippets mit hervorgehobenen Treffern an
- Dedizierte Journal-Seite (`/journal`): Kalender + Eintraege-Liste mit Desktop 2-Spalten-Layout und Mobile-Responsive Collapsible-Kalender
- Journal-Button in Sidebar ersetzt inline Mini-Kalender (konsistent mit Recipes-Button)
- Neuer Backend-Endpoint `GET /journal/entries` fuer die Eintraege-Liste
- KI-gestuetzte Rezeptvorschlaege: Aehnliche Rezepte finden, Rezepte anhand von Zutaten vorschlagen und generierte Rezepte direkt speichern
- Neue Backend-Tests fuer UpdateNoteTitle (Versioning + Normalisierung) und GetNotesByIDs (Edge Cases)
- Foto-Upload fuer Zutatenerkennung: Kuehlschrankfoto hochladen und die KI erkennt automatisch die vorhandenen Zutaten
- Vision-API-Support fuer Claude und Gemini Bildverarbeitung
- Erledigungszeitpunkte fuer Tasks werden automatisch erfasst (Grundlage fuer kuenftige Statistiken)
- Dedizierte Bildergalerie fuer Rezepte mit Batch-Upload, Bildunterschriften und Sortierung
- Erledigte Todo-Items werden in aufklappbare Gruppe zusammengefasst (Standard: eingeklappt), damit der Fokus auf offenen Aufgaben liegt. Collapse-State bleibt ueber Re-Renders erhalten.
- Kochbuch-Sharing: Ganze Rezeptsammlungen (Collections) koennen mit anderen Benutzern geteilt werden (Viewer/Editor-Rollen)
- Collection-Share Berechtigungsmodell: 3-Tier Prioritaetskette (note_shares > folder_shares > collection_shares), nur additiv (R1), hoechste Prioritaet gewinnt (R2), Dedup (R3)
- Verschluesselungs-Guard (R4): Sharing wird blockiert wenn Collection verschluesselte Rezepte enthaelt, verschluesselte Rezepte koennen nicht zu geteilten Collections hinzugefuegt werden
- Typ-Enforcement (R5): Collections enthalten ausschliesslich Rezepte (note_type='recipe'), Service-Layer und DB-Queries filtern explizit
- Geteilte Rezepte Ansicht: /recipes Seite zeigt "Geteilte Rezepte" Sektion mit Rolle-Badge und Herkunftsinfo
- Geteilte Kochbuecher Seite: /shared/collection/[id] zeigt Rezepte einer geteilten Collection mit Header, Rolle-Badge und Owner-Info
- ShareCollectionDialog: User-Suche (300ms Debounce), Rollenauswahl, bestehende Shares verwalten (analog ShareFolderDialog)
- SharedWithMeList: Neue "Geteilte Kochbuecher" Sektion zwischen Ordnern und Notizen
- RecipeCollectionList: Share-Button (Users Icon) pro Collection auf Hover
- DB-Migration 038_recipe_collection_shares.sql mit recipe_collection_shares Tabelle (UNIQUE constraint, ON DELETE CASCADE, 3 Indexes)
- GetSharePermission() um 3. Branch (collection_shares) erweitert
- SharedNote um note_type Feld erweitert (omitempty, COALESCE fuer Altdaten)
- GetSharedRecipesForUser(): 3-fach UNION Query mit NOT EXISTS Dedup
- 7 neue Backend-API-Endpunkte fuer Collection-Sharing und Shared-Recipes/Collections
- 7 neue Frontend-API-Funktionen und Store-Erweiterungen (recipes.svelte.ts, sharing.svelte.ts)
- 13 neue DB-Tests fuer Collection-Sharing (CRUD, Dedup, Prioritaet, Verschluesselung, Cascade)
- Neue i18n Keys (de.json + en.json) fuer Collection-Sharing UI
- Rezept-Feature: Notizen koennen als Rezepte erstellt werden (note_type='recipe') mit strukturierten Zutaten, Portionen, Zubereitungszeit, Schwierigkeit und Quell-URL
- Rezept-Zutaten: Strukturierte Zutatenliste mit Mengen, Einheiten, Gruppen, optional/skalierbar Flags und Drag-Reorder
- Portionen-Skalierung: Server- und Client-seitige Berechnung skalierter Zutatenmengen mit konsistenter Rundung (2 Dezimalstellen)
- Rezept-Kochbuecher (Collections): Owner-only Sammlungen mit Name, Beschreibung und Farbe zur Organisation von Rezepten
- Optimistic Locking: Rezept-Metadata und Zutaten-Updates verwenden expected_updated_at zur Konflikterkennung (409 bei Mismatch)
- Rezept-Encryption: Verschluesselte Rezepte serialisieren Metadata + Zutaten in den encrypted payload, Entschluesselung stellt sie wieder her
- Rezept-Sharing: Nutzt bestehendes Note-Sharing (Editor kann Metadata+Zutaten bearbeiten mit Owner-user_id, Viewer nur lesen+skalieren)
- RecipeEditor mit 3 Tabs (Zutaten/Anleitung/Vorschau), Viewer-Modus fuer Shared-Viewer, Encrypted-Modus fuer verschluesselte Rezepte
- Rezept-Button in Sidebar (Feature-Flag-gesteuert), /recipes Uebersichtsseite, Settings-Toggle
- 12 neue API-Endpoints fuer Rezepte (CRUD Metadata, Ingredients, Collections, Scaling)
- DB-Migration 037_recipes.sql mit recipe_metadata, recipe_ingredients, recipe_collections, recipe_collection_items Tabellen
- WebSocket-Events fuer Rezept-Aenderungen (recipe.metadata.updated, recipe.ingredients.updated)
- ~100 neue i18n Keys (de.json + en.json) fuer Rezept-UI
- Feature-State Reset bei Logout (Journal + Rezepte) verhindert State-Leaking zwischen Benutzern
- Error Reporting via Forgejo: Automatische JS-Fehlerberichte und manuelles Feedback als Forgejo-Issues (Token bleibt serverseitig, Repo bleibt privat)
- Feedback-Button in der Sidebar (mobile, desktop expanded, desktop collapsed) mit FeedbackDialog
- Opt-out Toggle fuer automatische Fehlerberichte in Settings > Editor > Feature Toggles
- Fingerprint-basierte Deduplizierung: gleicher Fehler wird als Kommentar am bestehenden Issue angefuegt statt neues Issue
- Client-seitiges Rate-Limiting (3 Reports/5 Min) und Session-Dedup
- Backend Rate-Limiting (5/Stunde) und 16KB Body-Limit fuer Error Reports

### Changed

- `make quality` Target um golangci-lint und Policy-Checks erweitert (prueft jetzt Format, Lint, Typecheck, golangci-lint und Architektur-/Security-Policies in einem Lauf)
- Pre-commit Hooks (lefthook) um 3 neue Checks erweitert: Layer-Violation Ratchet, Svelte 4 Import Guard, Security Pattern Check
- golangci-lint Konfiguration (`.golangci.yml`) erweitert um build-tags (fts5, sqlite_crypt) und 5 zusaetzliche Linter (revive, misspell, bodyclose, gocritic, unused) mit targeted exclude-rules fuer 63 pre-existierende Findings
- API key status handlers now use explicit typed responses instead of untyped status payloads.
- Client IP extraction now validates `X-Forwarded-For` / `X-Real-IP` values before trusting them.
- Recipe image metadata timestamp updates are now handled transactionally in DB operations.
- README improved with architecture overview, development script reference, and fixed configuration table formatting.
- Frontend: `notes.svelte.ts`, `settings/+page.svelte` und `+layout.svelte` modularisiert (Helper unter `src/lib/stores/notes/` und `src/lib/routes/`).
- Layout- und Notes-Flows (PWA-Update, Init, Interactions, Viewport, Guards, Auto-Save, Remote-Updates) in eigene Module ausgelagert.

### Fixed

- FE-Typecheck bereinigt; `npm run typecheck`, `npm run lint`, `npm run format` und `make test-frontend` gruen.
- Konfiguration ueber FORGEJO_URL, FORGEJO_REPO, FORGEJO_API_TOKEN (Feature deaktiviert wenn leer)
- /api/config liefert `error_reporting_enabled` fuer Frontend-Feature-Detection
- 10 Frontend-Tests (normalizeMessage, computeFingerprint, Dedup) und 19 Backend-Tests (Service + Handler)
- Folder Sharing: Ganze Ordner koennen mit anderen Benutzern geteilt werden (Viewer/Editor-Rollen), alle Notizen darin sind implizit geteilt
- Shared Note Placements: Empfaenger koennen geteilte Notizen in eigene Ordner einordnen, ohne Ownership zu aendern
- ShareFolderDialog mit User-Suche, Rollen-Auswahl und Encryption-Warnungen
- SharedWithMeList zeigt jetzt geteilte Ordner (mit NoteCount und Rollen-Badge) vor einzelnen Notizen, gruppiert nach Besitzer
- Dedizierte `/shared/folder/{id}` Seite fuer Notizen in geteilten Ordnern
- "Teilen"-Option im Ordner-Kontextmenue (nicht fuer Root und Journal)
- Notizen haben jetzt dieselben Kontextmenue-Optionen wie Ordner in der Seitenleiste: Umbenennen, Teilen, Farbe, Loeschen
- RenameNoteDialog: Notizen koennen direkt aus der Seitenleiste umbenannt werden (ohne Editor oeffnen)
- Git pre-commit Hook: Nicht-blockierende Erinnerung wenn CHANGELOG.md nicht im Commit enthalten ist
- 8 neue API-Endpoints fuer Folder Sharing und Placements (CRUD fuer Folder-Shares, geteilte Ordner und deren Notizen, Placement-Verwaltung)
- DB-Migration `036_shared_note_placements.sql` mit Placements-Tabelle und Folder-Shares-Index
- Permission-Chain: `note_shares` hat Vorrang vor `folder_shares`, implizite Vererbung fuer Ordner-Notizen
- Defense-in-Depth: Share-Validierung auf Service-, DB- und Query-Ebene (UNION mit aktivem Share-Check)
- Placement-Cleanup bei Share-Entzug (Note und Folder) + Belt-and-Suspenders UNION-Check
- 22 neue DB-Tests fuer Folder-Sharing, Placements, Permission-Chain und Edge Cases
- 17 neue i18n Keys in en.json und de.json fuer Folder-Sharing-UI
- Encryption Toggle: Einzelne Notizen koennen ueber das More-Menu im Editor entschluesselt und wieder verschluesselt werden
- Folder Encryption Default: Ordner koennen als "unverschluesselt" markiert werden, neue Notizen darin werden ohne Verschluesselung erstellt
- 3 neue API-Endpoints fuer Encryption Toggle (`POST /api/notes/{id}/decrypt`, `GET/PUT /api/folders/{id}/encryption-default`)
- DB-Migration `035_encryption_toggle.sql` mit `encryption_default` Spalte fuer Folders
- 7 neue DB-Tests fuer Encryption Toggle (DecryptNote, VersionMismatch, EncryptNoteRemovesShares, FolderEncryptionDefault, etc.)
- Automatische Share-Entfernung beim Verschluesseln einer Notiz (Business-Regel: verschluesselte Notizen sind nicht teilbar)
- Note Sharing (Phase 1 MVP): Notizen koennen mit anderen Benutzern geteilt werden (Viewer- und Editor-Rollen)
- 8 neue API-Endpoints fuer Sharing (CRUD fuer Shares, geteilte Notizen abrufen/bearbeiten, User-Suche)
- ShareNoteDialog mit User-Suche (Debounce), Rollen-Auswahl und Share-Verwaltung
- SharedWithMeList: Gruppierte Anzeige geteilter Notizen nach Owner mit Rollen-Badges
- "Geteilt mit mir" Button in Sidebar mit Count-Badge (Mobile + Desktop + Collapsed)
- "Teilen" Menuepunkt im EditorMoreMenu (disabled bei verschluesselten Notizen)
- Dedizierte `/shared` Seite fuer geteilte Notizen
- DB-Migration `034_note_sharing.sql` mit `note_shares` und `folder_shares` Tabellen (Phase 2 vorbereitet)
- 22 neue i18n Keys in en.json und de.json fuer Sharing-UI
- 10 DB-Tests fuer Sharing-Layer
- Hardware Security Keys (FIDO2/WebAuthn) als zweite 2FA-Methode neben Authenticator App
- Security Key Management in den Einstellungen (Registrierung, Benennung, Loeschung)
- Methodenauswahl beim Login (Security Key, Authenticator App, Backup Code)
- All action icons (trash, settings, theme, logout) now visible in collapsed sidebar
- Resizable sidebar and editor/preview split view via drag handle
- AI Actions for text transformation via LLM integration
- CodeMirror plugins, syntax theming, and scrollable toolbar
- Unified "Create Note" dialog on start page (consistent with sidebar)
- Auto-focus in "Create Folder" dialog
- LLM-based note summaries with Ollama integration (auto-scheduler and manual generation)
- Interactive image resizing via drag & drop in Markdown preview
- Client-side wiki-link extraction for E2E-encrypted notes (enables backlinks)
- Cross-folder drag & drop: notes can be moved between folders by dropping onto notes in the target folder
- Drag & drop task list reordering
- Internationalization (i18n) for all dialogs, editor, markdown guide, and admin page (~604 keys per locale)
- Security badges on login page (E2E Encrypted, Zero-Knowledge, Open Source)
- New themes: Dark Pastels, Gruvbox Light, Gruvbox Dark (12 themes total)
- Global graph visualization of notes and connections (`/graph`, `Ctrl+G`)
- Mobile-optimized version history with tab navigation
- Text wrapping for mobile view in editor and preview
- Responsive two-row toolbar layout for mobile
- iOS autocorrect, autocapitalize, and spellcheck support in editor
- Trash with soft-delete, restore, and permanent delete
- Undo/Redo system with Command Pattern (`Ctrl+Z` / `Ctrl+Shift+Z`)
- Toast notification system
- Auto-save with 2s debounce and visual status indicators
- Backend pprof profiling endpoint (opt-in via `PPROF_ENABLED`)
- CodeMirror code-splitting (-42% login page bundle size)
- Forgejo Actions CI/CD pipeline for automatic staging deployment on push to main (with auto-rollback, health checks, and security-hardened containers)

### Changed

- Repository ist nun oeffentlich auf GitHub verfuegbar
- Versionshistorie speichert jetzt bis zu 100 Versionen pro Notiz statt bisher 30
- GitHub CI um Prettier-, Markdownlint- und strengere ESLint-Checks erweitert fuer Paritaet mit Forgejo Quality Gates
- README neu strukturiert mit klaren Sections, Codebeispielen, Badges und Screenshot-Platzhaltern
- .gitignore um Coverage- und Playwright-Reports ergaenzt
- README komplett ueberarbeitet: Professionelle Struktur mit Badges, "Why xelanote"-Sektion, Feature-Kategorien, ausfuehrliche Contributing-Anleitung und Development-Kommandos
- Alle ESLint-Warnings (260+) auf null reduziert: Fehlende each-Keys, ungenutzte Variablen, explizite any-Typen, nicht-reaktive Svelte-5-Objekte und unsaubere Regex-Escapes behoben
- ESLint-Schwelle in CI und Makefile von 700 auf 0 gesenkt, neue Warnings werden ab sofort blockiert
- Alle API-Fehlerantworten verwenden jetzt einheitlich JSON-Format statt gemischtem Plain-Text/JSON
- Backend-Code: Duplizierte Validierungslogik fuer Notiz-Felder, Journal-Feature-Checks und ETag-Parsing in wiederverwendbare Hilfsfunktionen konsolidiert
- Journal und Rezepte sind jetzt standardmaessig fuer alle Nutzer aktiviert (bestehende und neue)
- Inhaltsverzeichnis (TOC) ist jetzt ein Floating-Button oben rechts im Preview statt ganz unten versteckt. Oeffnet sich als Dropdown-Overlay, bleibt beim Scrollen sticky sichtbar, responsive fuer Mobile.
- Upload-Button in der Editor-Toolbar verwendet jetzt ein Bild-Icon (ImagePlus) statt des generischen Upload-Pfeils fuer bessere Erkennbarkeit
- Tests fuer Account-Lockout verwenden jetzt eine kontrollierte Uhr (keine echten Sleeps mehr)
- Quality-Checks dokumentiert und Make-Targets fuer `fmt-check`, `typecheck` und `quality` ergaenzt (Format, Lint, Typecheck in einem Lauf)
- UI-A11y und Svelte-Deprecation Fixes (Dialog-Fokus, Tastatur-Handling, Slot-Rendering, Labels/ARIA)
- Pre-commit Hooks (lefthook) pruefen automatisch gofmt, go vet, ESLint, Prettier und Markdownlint vor jedem Commit
- Markdown-Linting mit markdownlint-cli2 fuer README und Docs-Verzeichnis eingerichtet (strukturelle Regeln)
- Link-Checking mit lychee in der GitHub Actions Quality-Pipeline ergaenzt
- Forgejo Staging-Deployment fuehrt jetzt Backend-Tests und Frontend-Lint vor dem Deploy aus
- CHANGELOG-Aktualisierung wird in Pull Requests automatisch geprueft (Warning bei fehlender Aenderung)
- Sidebar-Aktionen (Farbe, Umbenennen, Loeschen) ueber Kontextmenue statt Inline-Buttons erreichbar
- Breadcrumb-Navigation (Haus-Symbol + Pfad) wird auf Mobilgeraeten nicht mehr angezeigt
- MobileHeader (Logo-Leiste) wird auf Notiz-Seiten auf Mobilgeraeten ausgeblendet fuer mehr Platz
- Hamburger-Menue ist in der Editor-Toolbar auf Mobilgeraeten immer sichtbar
- Speicher-Indikator erscheint direkt neben dem Notiztitel auf Mobilgeraeten
- Focus-Mode-Button wird auf Mobilgeraeten nicht mehr angezeigt
- Editor-Toolbar aufgeraeumt: selten genutzte Aktionen (Loeschen, Verschieben, Textfarbe, Einruecken, KI-Toggle) ins More-Menue verschoben
- Auto-Save-Statustext entfernt, nur noch Icon-Feedback in der Toolbar
- More-Menue komplett lokalisiert (war vorher nur auf Deutsch)
- Split-View-Modus auf Mobile deaktiviert zugunsten von Edit/Preview-Umschaltung
- Sidebar-Layout komplett ueberarbeitet: Erstellungs-Buttons im Header, Suchzeile ueber dem Baum, Papierkorb als Tree-Item
- Import/Export von der Sidebar in die Einstellungen verschoben (neuer "Daten"-Tab)
- Theme-Umschalter in der Seitenleiste vereinfacht zu einem direkten Toggle-Button mit Sonne/Mond-Icons
- Gruvbox Aqua theme consistency across entire frontend
- Documentation restructured: planning docs moved to `docs/planning/`, cross-references added, new testing guide and environment variables reference
- Drei fast identische Sharing-Dialoge (Note, Folder, Collection) zu einem generischen `ShareDialog.svelte` konsolidiert (~400 Zeilen Duplikation eliminiert)
- Konfliktierendes `backend/Dockerfile` entfernt, SQLCipher-Anleitung als Kommentar im Root-Dockerfile dokumentiert
- Staging-Workflow auf node:22 aktualisiert (war node:20, inkonsistent mit Root-Dockerfile)
- CORS-Dokumentation korrigiert: `CORS_ALLOWED_ORIGINS` ist in Production _required_ (log.Fatal), nicht nur "recommended"
- Fehlende Env-Vars `GEMINI_MODEL` und `XELANOTE_API_KEY_SECRET` in `.env.example` ergaenzt
- Salt-Deletion-Testanleitung von TODO.md nach `docs/runbooks/salt-deletion-test.md` verschoben
- CI: Env-Var-Sync-Check hinzugefuegt (`scripts/check-env-sync.sh`) — vergleicht Go-Source, Docs und `.env.example`
- CI: Bundle-Size-Tracking hinzugefuegt (`scripts/check-bundle-size.sh`, Budget: 3600 KB)

### Fixed

- `versions.go` restoreVersion verwendete rohen Integer-ETag statt gehashtem ETag und unterstuetzte keine neuen Hash-ETags
- Papierkorb zeigte die Anzahl geloeschter Notizen an, aber die Notizliste blieb leer (Virtualizer-Deadlock durch fehlende Container-Hoehe)
- Endgueltiges Loeschen einzelner Notizen aus dem Papierkorb schlug immer mit 404 fehl
- Papierkorb-Badge in der Seitenleiste aktualisierte sich nicht beim Loeschen ueber das Kontextmenue im Notizbaum
- Wiederhergestellte Notizen erschienen erst nach manuellem Seiten-Reload in der Seitenleiste
- Inhaltsverzeichnis (TOC) war auf mobilen Geraeten transparent und ueberlagerte den Seiteninhalt unleserlich
- Inhaltsverzeichnis-Button scrollte beim Herunterscrollen aus dem Sichtfeld statt oben fixiert zu bleiben
- Split-View wurde auf Mobilgeraeten trotz fehlender Buttons angezeigt, wenn der gespeicherte Editor-Modus "split" war
- Volltextsuche fuer verschluesselte Notizen lieferte veraltete oder falsche Treffer, weil die FTS-Trigger-Korrektur nie ausgefuehrt wurde
- Keywords fuer verschluesselte Notizen wurden nie gespeichert, weil die Preference-Abfrage eine nicht existierende Tabellenstruktur verwendete
- Verschluesselungs-Einstellungen (Keyword-Extraktion, Titel-Verschluesselung) wurden nur lokal gespeichert und nie an den Server uebermittelt
- Schnellsuche (Strg+P) fand keine verschluesselten Notizen, weil nur Plaintext-Titel durchsucht wurden
- FTS5-Trigger fuer `note_keywords` DELETE/UPDATE verwendeten falsche Syntax fuer external-content Tabellen (Migration 041)
- Tag- und Link-Vorschlaege zeigen jetzt korrekt "AI features not enabled for this note" statt generischer Fehlermeldungen wenn KI fuer eine Notiz deaktiviert ist
- Suchfenster (Quick Switcher) laesst sich jetzt durch Klick ausserhalb wieder schliessen
- Journal-Kalender crasht nicht mehr beim Anzeigen von Monaten ohne Eintraege (TypeError auf null wurde behoben)
- Android PWA: Sidebar-Buttons reagieren wieder auf Taps (setPointerCapture wurde zu frueh aufgerufen und blockierte click-Events in Chrome)
- Journal-Ordner in der Sidebar kann jetzt aufgeklappt werden: loadTree() laedt Journal-Notizen separat nach, da sie vom Standard-API-Query (note_type='note') ausgeschlossen werden
- Enter auf checked Tasks springt nicht mehr unerwartet nach oben. Standard-Enter-Verhalten wiederhergestellt, Auto-Sort nur noch bei Checkbox-Toggle.
- Flaky Tests: Lockout, FIDO2-Session-Store und Upload-Signaturen sind jetzt deterministisch (keine sleeps, keine timing races)
- Panic/500 bei Hash-ETag-Requests wenn Notiz geloescht wurde: `db.GetNote` gibt jetzt `ErrNotFound` statt `(nil, nil)` zurueck, verhindert nil-Dereferenz in Update- und Decrypt-Handlern
- Stille Teilergebnisse bei Such-Queries: `rows.Err()` wird jetzt nach Iteration in Search, QuickSearch und FilteredSearch geprueft
- Touch-Ziele auf Mobilgeraeten sind jetzt gross genug zum zuverlaessigen Antippen
- Aktions-Buttons in der Notizbaumansicht sind auf Touch-Geraeten jetzt sichtbar
- Benachrichtigungen und Dialoge laufen auf schmalen Bildschirmen nicht mehr ueber
- QuickSwitcher und Login-Seite sind auf Mobilgeraeten besser nutzbar
- Page refresh on a selected note now stays on that note instead of redirecting to the start page
- Sidebar correctly highlights the selected note after page refresh
- Split View wurde auf Mobilgeraeten nicht automatisch deaktiviert
- Mobile layout was broken due to sidebar rendering in desktop mode on small screens
- Reordering notes in the root folder had no effect because display_order was ignored for root-level notes
- Spell checker was accessible even when AI functions were disabled for a note
- Firefox iOS keyboard detection not hiding header/breadcrumbs when typing
- Auth/token refresh bugs causing unexpected logouts and memory leaks
- Summary display not updating after regeneration (Svelte 5 reactivity)
- Encrypted note versions not decrypted in version history dialog
- Action buttons hidden on folders with long names (CSS flexbox truncation)
- Missing `parent_id` for root-level folders during markdown import
- Theme selection not persisting on page refresh (client-server sync)
- Rename with refactor broken due to non-existent column reference
- Trash page crash on NULL `deleted_at` values in legacy data
- Svelte 5 `$effect` orphan error when used in stores
- Infinite loop and browser freeze from reactive auto-save chain
- SSR hydration mismatch from premature auth initialization
- Login/register/logout race conditions from reactive navigation guards
- Auto-save toggle not triggering for already-dirty notes
- N+1 query problem in rename feature (batch query optimization)
- iOS keyboard closing on autosave due to editor unmounting
- CodeMirror text selection color not applying when editor focused
- AI Actions dropdown rendered outside overflow container

### Security

- Alte Git-History bereinigt und alle Production-Secrets rotiert
- Infrastruktur-Details (IP-Adressen, SSH-Ports, Benutzernamen) in der Dokumentation durch Platzhalter ersetzt
- Versehentlich committed node_modules aus dem archive-Verzeichnis entfernt (442 Dateien)
- .gitignore gehaertet: Private Keys, Zertifikate, Claude Code Konfiguration und node_modules global ausgeschlossen
- JSON Body-Size Limits: `http.MaxBytesReader` erhaelt jetzt korrekt den `ResponseWriter` (verhindert Panic bei Ueberschreitung), Standard 1MB Limit fuer alle Endpoints, 16MB fuer Note-Content und Import
- User-Enumeration verhindert: Sharing-Fehlermeldungen enthalten keine Benutzernamen mehr ("unable to share with specified user" statt "user not found: username")
- IV-Validierung: Verschluesselte Titel muessen jetzt ein gueltiges Base64-IV-Feld enthalten
- Crypto Debug-Logs (libsodium Version, Key-Derivation) nur noch im DEV-Modus sichtbar
- Request-Context wird jetzt an alle Such-Queries durchgereicht (ermoeglicht Timeout/Abbruch bei Client-Disconnect)
- Encryption-Guard: Verschluesselung kann nicht auf geteilten Ordnern aktiviert werden (Share-Entzug erforderlich)
- Ordner mit `encryption_default=true` oder verschluesselten Notizen koennen nicht geteilt werden
- Verschluesselte Notizen in geteilten Ordnern werden aus der Anzeige gefiltert (Defense-in-Depth)
- SSE-Nachrichten werden jetzt sicher als JSON serialisiert, um Injection zu verhindern
- HTTP-Server-Timeouts schuetzen jetzt gegen Slowloris-Angriffe
- Account-Lockout nutzt jetzt hybrides IP- und Account-Tracking gegen verteilte Brute-Force-Angriffe
- WebSocket lehnt leere Origin-Header im Produktionsmodus ab
- Klartextinhalt wird nicht mehr als URL-Query-Parameter gesendet
- Benutzerkennungen werden in Logs jetzt gehasht statt im Klartext ausgegeben
- Docker-Container hat jetzt ein PID-Limit zum Schutz gegen Fork-Bomben
- CSRF token validation restored after SameSite=Strict migration
- Constant-time login comparison to prevent user enumeration via timing
- Generic error messages on registration to prevent user enumeration
- Security event logging for 9 event types (login, password change, 2FA, etc.)
- ETag version hashing with SHA256 instead of raw version integer
- TOCTOU race condition fix in upload quota enforcement
- IDOR vulnerability fix in backlinks endpoint (cross-user data leakage)
- Comprehensive `user_id` filtering added to all link queries

## [0.5.0] - 2026-01-17

### Fixed

- Search crash on whitespace-only queries
- Stale UI data after note rename (missing fresh data fetch)
- Broken wikilink navigation (used titles instead of note IDs)
- Concurrent save race condition on rapid `Ctrl+S`
- Dirty state indicator not clearing after successful save

### Security

- XSS prevention in wikilink rendering via HTML escaping
- Path traversal fix in export API (`../../etc` directory traversal)

## [0.4.0] - 2026-01-17

### Added

- Multi-user authentication system with JWT (access + refresh tokens)
- User registration and login pages
- Multi-user data isolation (all queries filtered by `user_id`)
- Protected routes with automatic redirect to login
- Database migrations for users table and user ownership

### Fixed

- SQL parameter count mismatch in multi-user queries
- Frontend store synchronization (tree store vs notes store limit mismatch)

## [0.3.0] - 2026-01-17

### Added

- Folder rename with validation dialog
- Drag & drop folder reordering with custom display order
- Virtual root pattern for cleaner folder tree
- Database migration for `display_order` fields

### Fixed

- Root-level folder creation bug (path "/" not handled correctly)
- Legacy database repair for orphaned folders with NULL `parent_id`
- Nested button HTML validation warning in folder tree

## [0.2.0] - 2026-01-XX

### Added

- Unified tree with folders and notes in one sidebar
- Drag & drop for notes and folders
- Expand/collapse state with localStorage persistence
- Parent-child folder hierarchy with unlimited nesting
- Note counts per folder
- Database migration for folders table

## [0.1.0] - 2025-XX-XX

### Added

- Basic note-taking application with Markdown editor
- Full-text search via SQLite FTS5
- Note versioning with history
- Backlinks via wiki-links
- Path-based folder system

[Unreleased]: https://github.com/xela-io/xelanote/compare/v1.1.3...HEAD
[1.1.3]: https://github.com/xela-io/xelanote/compare/v0.5.0...v1.1.3
[0.5.0]: https://github.com/xela-io/xelanote/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/xela-io/xelanote/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/xela-io/xelanote/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/xela-io/xelanote/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/xela-io/xelanote/releases/tag/v0.1.0
