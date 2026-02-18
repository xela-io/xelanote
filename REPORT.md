# Full-Repo Audit Report

Date: 2026-02-17
Scope: Entire tracked repository in `/workspace` (backend, frontend, docs, CI/CD, scripts)

## Assumptions And Validation
- Assumption: pre-existing untracked/WIP files are out of scope for behavior conclusions (`frontend/src/lib/api/canvas.ts`, `frontend/src/routes/canvas/`, etc.).
- Assumption: audit focuses on tracked code and docs only; no runtime behavior changes were made.
- Validation run:
  - `make test-frontend` passed (Vitest).
  - `make lint-md` passed.
  - `make test` could not run in this environment (`go: not found`).
  - Policy scripts previously run in-session: env sync, layer checks, svelte4 import checks, security pattern checks (all passed).

## Architecture Map
- Backend runtime entrypoint is `backend/cmd/server/main.go` (flags/env, DB open+migrate, service wiring, router/bootstrap, graceful shutdown): `backend/cmd/server/main.go:20`, `backend/cmd/server/main.go:50`, `backend/cmd/server/main.go:58`, `backend/cmd/server/main.go:129`, `backend/cmd/server/main.go:161`.
- API routing is split into public and protected route groups under `/api`: `backend/internal/api/routes.go:22`, `backend/internal/api/routes.go:23`, `backend/internal/api/routes.go:24`, `backend/internal/api/routes.go:38`, `backend/internal/api/routes.go:67`.
- Public routes include config/changelog/auth recovery/error reports; protected routes are guarded by auth + CSRF middleware: `backend/internal/api/routes.go:40`, `backend/internal/api/routes.go:41`, `backend/internal/api/routes.go:60`, `backend/internal/api/routes.go:61`, `backend/internal/api/routes.go:64`, `backend/internal/api/routes.go:71`, `backend/internal/api/routes.go:73`.
- Protected routes are further composed via registries (resource routes vs utility routes): `backend/internal/api/routes_registry.go:6`, `backend/internal/api/routes_registry.go:20`.
- Layering is explicit: API handlers call services, services call DB (`internal/api` -> `internal/service` -> `internal/db`) and policy checks enforce boundaries (`scripts/check-layer-violations.sh`).
- Core backend dependencies are Go 1.25.0 + Chi v5.2.5 + SQLite driver: `backend/go.mod:3`, `backend/go.mod:6`, `backend/go.mod:11`.
- DB migrations are forward-only SQL files; current set includes 002..047: `backend/internal/db/migrations/046_perf_metrics.sql`, `backend/internal/db/migrations/047_analytics_events.sql`.
- Frontend app bootstrap is in SvelteKit layout load + layout component (`i18n`, auth/init/offline/pwa/ws setup): `frontend/src/routes/+layout.ts:7`, `frontend/src/routes/+layout.svelte:148`, `frontend/src/routes/+layout.svelte:153`.
- Frontend API access is modularized per domain (`frontend/src/lib/api/*.ts`) with shared request client.
- Build orchestration is centralized in root `Makefile` (`build`, `run-backend`, `run-frontend`, `test`, `quality`): `Makefile:7`, `Makefile:23`, `Makefile:27`, `Makefile:31`, `Makefile:144`.
- Production/dev container workflows exist in both GitHub Actions and Forgejo workflows: `.github/workflows/ci.yml:1`, `.github/workflows/quality.yml:1`, `.github/workflows/security.yml:1`, `.forgejo/workflows/deploy-staging.yml:3`, `.forgejo/workflows/deploy-production.yml:3`.
- Git hook guardrails run formatting/lint/policy checks pre-commit and pre-push via Lefthook: `lefthook.yml:5`, `lefthook.yml:46`.
- Docker Compose deployment runs single service with localhost binding, health check, hardening flags: `docker-compose.yml:2`, `docker-compose.yml:9`, `docker-compose.yml:16`, `docker-compose.yml:35`, `docker-compose.yml:39`.
- Frontend Vite dev server proxies `/api` to backend `:8080`; default web dev port is `5173`: `frontend/vite.config.ts:152`, `frontend/vite.config.ts:155`, `frontend/vite.config.ts:156`.
- Tauri dev URL also points at `5173`: `frontend/src-tauri/tauri.conf.json:9`.

## Top Redundancies (Prioritized)

### R1. Template/Snippet CRUD duplicated across 4 layers
- Why redundant:
  - Same list/get/create/update/delete structure is repeated for templates and snippets in API, service, DB, and frontend API modules.
- Evidence:
  - API: `backend/internal/api/templates.go:50`, `backend/internal/api/templates.go:95`, `backend/internal/api/templates.go:124`, `backend/internal/api/templates.go:169`.
  - API mirror: `backend/internal/api/snippets.go:45`, `backend/internal/api/snippets.go:90`, `backend/internal/api/snippets.go:119`, `backend/internal/api/snippets.go:164`.
  - Service cache wrappers: `backend/internal/service/templates.go:36`, `backend/internal/service/snippets.go:36`.
  - DB CRUD: `backend/internal/db/templates.go:34`, `backend/internal/db/snippets.go:33`.
  - Frontend API wrappers: `frontend/src/lib/api/templates.ts:4`, `frontend/src/lib/api/snippets.ts:4`.
- Consolidation target:
  - Introduce shared generic CRUD helpers per layer while keeping public function names stable.
- Risk: medium
- Effort: M
- Refactor sketch:
  - `backend/internal/api/crud_helpers.go`
  - `func listOwned[T any](w http.ResponseWriter, r *http.Request, fetch func(userID int) ([]T, error), key string)`
  - `func getOwned[T any](w http.ResponseWriter, r *http.Request, notFoundMsg string, fetch func(userID, id int) (*T, error))`
  - Update callers in `backend/internal/api/templates.go` and `backend/internal/api/snippets.go` only (first pass).

### R2. Share-management handler flows triplicated (notes/folders/recipe collections)
- Why redundant:
  - Repeated auth checks, ID parsing, `identifier` validation, role validation, and similar error-branch handling.
- Evidence:
  - Notes sharing: `backend/internal/api/sharing_notes.go:12`, `backend/internal/api/sharing_notes.go:31`, `backend/internal/api/sharing_notes.go:35`, `backend/internal/api/sharing_notes.go:129`.
  - Folder sharing: `backend/internal/api/sharing_folders.go:15`, `backend/internal/api/sharing_folders.go:33`, `backend/internal/api/sharing_folders.go:37`, `backend/internal/api/sharing_folders.go:130`.
  - Recipe collection sharing: `backend/internal/api/recipes_collection_shares.go:13`, `backend/internal/api/recipes_collection_shares.go:31`, `backend/internal/api/recipes_collection_shares.go:35`, `backend/internal/api/recipes_collection_shares.go:126`.
- Consolidation target:
  - Shared helper for request validation (`identifier`, role, decode) and shared error mapping helper with resource-specific adapters.
- Risk: medium
- Effort: M/L
- Refactor sketch:
  - `backend/internal/api/sharing_common.go`
  - `func validateShareInput(identifier, role string) error`
  - `func mapShareError(resource string, err error) (status int, msg string)`
  - Update callers in `sharing_notes.go`, `sharing_folders.go`, `recipes_collection_shares.go`.

### R3. Ownership checks repeated in service layer
- Why redundant:
  - Same owner-fetch + equality guard appears in many methods (note/folder/collection).
- Evidence:
  - Notes ownership checks: `backend/internal/service/sharing.go:46`, `backend/internal/service/sharing.go:96`, `backend/internal/service/sharing.go:141`.
  - Folder ownership checks: `backend/internal/service/sharing.go:187`, `backend/internal/service/sharing.go:246`, `backend/internal/service/sharing.go:284`.
  - Collection ownership checks: `backend/internal/service/recipes_collections.go:19`, `backend/internal/service/recipes_collections.go:69`, `backend/internal/service/recipes_collections.go:97`.
- Consolidation target:
  - Add explicit ownership guard helpers in each service.
- Risk: low/medium
- Effort: M
- Refactor sketch:
  - `func (s *SharingService) requireNoteOwner(ownerUserID int, noteID string) error`
  - `func (s *SharingService) requireFolderOwner(ownerUserID, folderID int) error`
  - `func (s *RecipeService) requireCollectionOwner(ownerUserID, collectionID int) error`
  - Replace in existing methods without changing handler contracts.

### R4. Repeated API auth/decode/error boilerplate
- Why redundant:
  - Extremely frequent `getUserID` and `decodeJSON` branches increase maintenance surface.
- Evidence:
  - `getUserID` pattern count: 147 occurrences in `backend/internal/api`.
  - `decodeJSON` request-body pattern count: 65 occurrences in `backend/internal/api`.
  - Samples: `backend/internal/api/templates.go:52`, `backend/internal/api/snippets.go:47`, `backend/internal/api/users_webauthn.go:9`.
- Consolidation target:
  - Add narrow helper wrappers (not middleware rewrite) for low-risk incremental reuse.
- Risk: medium
- Effort: M
- Refactor sketch:
  - `func requireUserID(w http.ResponseWriter, r *http.Request) (int, bool)`
  - `func decodeBody[T any](w http.ResponseWriter, r *http.Request, dst *T) bool`
  - Phase-in file-by-file starting with templates/snippets/sharing handlers.

### R5. Duplicated validation constants for templates/snippets
- Why redundant:
  - Same limits are duplicated in API and DB layers; this can drift silently.
- Evidence:
  - Template constants in DB: `backend/internal/db/templates.go:11`.
  - Template constants in API: `backend/internal/api/templates.go:28`.
  - Snippet constants in DB: `backend/internal/db/snippets.go:11`.
  - Snippet constants in API: `backend/internal/api/snippets.go:28`.
- Consolidation target:
  - Single source for constraints (`internal/db` exported constants or `internal/domain/constraints`).
- Risk: low
- Effort: S
- Refactor sketch:
  - Add `backend/internal/domain/constraints/limits.go`
  - `const MaxTemplateContentSize = ...`
  - Replace local constants in four files and keep error text unchanged.

### R6. Frontend query-string construction repeated across API modules
- Why redundant:
  - Same `URLSearchParams` + `toString()` pattern repeated in many modules.
- Evidence:
  - `frontend/src/lib/api/notes.ts:14`
  - `frontend/src/lib/api/trash.ts:10`
  - `frontend/src/lib/api/graph.ts:10`
  - `frontend/src/lib/api/admin.ts:43`
  - `frontend/src/lib/api/versions.ts:8`
- Consolidation target:
  - Shared query utility helper.
- Risk: low
- Effort: S
- Refactor sketch:
  - `frontend/src/lib/api/query.ts`
  - `export function withQuery(path: string, set: (p: URLSearchParams) => void): string`
  - Replace call-sites in listed modules.

## Documentation Consistency Audit (Prioritized)

### Fixed In This Pass

#### D1. API auth/public endpoint rules were inaccurate
- Discrepancy:
  - Docs claimed all requests except `/health` and `/api/auth/*` need auth.
  - Code has additional public endpoints (`/api/config`, `/api/changelog`, recovery endpoints, `/api/error-reports`) and also protected `/api/auth/me`.
- Source of truth:
  - `backend/internal/api/routes.go:38`, `backend/internal/api/routes.go:40`, `backend/internal/api/routes.go:41`, `backend/internal/api/routes.go:60`, `backend/internal/api/routes.go:61`, `backend/internal/api/routes.go:64`, `backend/internal/api/routes.go:76`.
- Patch applied:
  - `docs/api.md:165` (auth section rewritten with explicit public endpoints).

#### D2. API config response docs were stale
- Discrepancy:
  - Docs lacked `version`, `error_reporting_enabled`, and `captcha_iframe_url`.
- Source of truth:
  - `backend/internal/api/config.go:13`, `backend/internal/api/config.go:17`, `backend/internal/api/config.go:18`, `backend/internal/api/config.go:39`.
- Patch applied:
  - `docs/api.md:245` (response example + notes updated).

#### D3. WebAuthn delete/touch request/response contracts mismatched implementation
- Discrepancy:
  - Docs used JSON body and `message`; code expects `credential_id` query parameter and returns `status`.
- Source of truth:
  - `backend/internal/api/users_webauthn.go:50`, `backend/internal/api/users_webauthn.go:62`, `backend/internal/api/users_webauthn.go:73`, `backend/internal/api/users_webauthn.go:85`.
- Patch applied:
  - `docs/api.md:3939`, `docs/api.md:3950`, `docs/api.md:3974`, `docs/api.md:3984`.

#### D4. API docs ToC link drift and broken anchor
- Discrepancy:
  - Upload path in ToC used outdated `:filename` form; shared-folder-notes anchor was malformed.
- Source of truth:
  - Route: `backend/internal/api/routes.go:98`.
  - Docs section: `docs/api.md:2637`.
- Patch applied:
  - `docs/api.md:92`, `docs/api.md:101`.

#### D5. Tauri dev port docs mismatched config
- Discrepancy:
  - Docs referenced `5174`, config uses `5173`.
- Source of truth:
  - `frontend/src-tauri/tauri.conf.json:9`.
- Patch applied:
  - `docs/desktop-app.md:310`, `docs/desktop-app.md:739`, `docs/desktop-app.md:755`.

#### D6. Go version requirements drift across docs
- Discrepancy:
  - README/development/architecture referenced Go 1.24.
  - Code and CI are Go 1.25.
- Source of truth:
  - `backend/go.mod:3`, `.github/workflows/ci.yml:20`, `.github/workflows/quality.yml:36`.
- Patch applied:
  - `README.md:75`, `docs/development.md:36`, `docs/development.md:55`, `docs/development.md:73`, `docs/architecture.md:54`.

#### D7. Missing due-date and telemetry endpoint docs
- Discrepancy:
  - `docs/api.md` previously did not document due-date and telemetry endpoints used by frontend and backend routes.
- Source of truth:
  - Routes: `backend/internal/api/routes_users_misc.go:77`, `backend/internal/api/routes_telemetry.go:6`, `backend/internal/api/routes_telemetry.go:8`.
  - Frontend usage: `frontend/src/lib/api/due-dates.ts:4`, `frontend/src/lib/stores/perf-metrics.svelte.ts:61`, `frontend/src/lib/stores/pwa.svelte.ts:82`.
- Patch applied:
  - `docs/api.md:99`, `docs/api.md:112` (ToC entries)
  - `docs/api.md:2609` (`GET /api/due-dates`)
  - `docs/api.md:3132` (`Telemetry` section with `/api/perf-metrics` and `/api/analytics/events`)

## Quick Wins (< 1 Day)
- Completed now:
  - `docs/api.md`: auth/public endpoint rules corrected; config response corrected; changelog/error-report endpoints documented; WebAuthn delete/touch corrected; ToC fixes.
  - `docs/api.md`: added missing docs for `GET /api/due-dates`, `POST /api/perf-metrics`, and `POST /api/analytics/events`.
  - `docs/desktop-app.md`: Tauri dev port corrected (`5173`).
  - `README.md`, `docs/development.md`, `docs/architecture.md`: Go/runtime version alignment and migration-count updates.
- Additional quick wins (not yet applied):
  - Add small shared frontend query helper (`withQuery`) and migrate 4-6 low-risk call-sites.
  - Centralize template/snippet size constants into one module.

## Medium/Large Refactors (With Migration Plan)

### M1. Consolidate sharing handlers and service ownership checks
- Scope:
  - `backend/internal/api/sharing_notes.go`
  - `backend/internal/api/sharing_folders.go`
  - `backend/internal/api/recipes_collection_shares.go`
  - `backend/internal/service/sharing.go`
  - `backend/internal/service/recipes_collections.go`
- Migration plan:
  1. Add helper functions with no call-site changes.
  2. Migrate one handler family (notes) and run tests.
  3. Migrate folder + collection handlers.
  4. Keep HTTP status/messages byte-for-byte stable.
- Rollback strategy:
  - Keep commits per family; revert specific commit if regression appears.
  - Use endpoint-level tests as canary.

### M2. Introduce shared CRUD helper substrate for template/snippet stacks
- Scope:
  - API/service/db/frontend modules for templates/snippets.
- Migration plan:
  1. Extract helper abstractions first, keep existing functions as wrappers.
  2. Migrate read paths (list/get), then write paths (create/update/delete).
  3. Validate responses/errors remain unchanged via existing tests + added golden tests.
- Rollback strategy:
  - Keep wrapper functions intact; switch wrappers back to old implementations by revert.

### M3. Incremental API boilerplate reduction (`requireUserID`/`decodeBody` helpers)
- Scope:
  - High-duplication handlers first (templates/snippets/sharing).
- Migration plan:
  1. Add helper utilities.
  2. Replace only 3-5 handlers per PR.
  3. Add regression tests around auth and bad JSON branches.
- Rollback strategy:
  - Helpers are additive; selective reverts on converted handlers only.

## Guardrails To Prevent Regression
- Add API-doc parity check script.
  - Tool: custom `scripts/check-api-doc-coverage.sh`.
  - Behavior: parse route literals from `backend/internal/api/routes*.go` and assert each public/protected route appears in `docs/api.md` headings.
  - CI wiring: add to `.github/workflows/quality.yml` under `docs` or `policy` job; add Lefthook pre-push command.
- Add doc contract tests for endpoint examples.
  - Tool: lightweight shell tests (`curl` + `jq`) against dev server for key endpoints (`/api/config`, `/api/users/webauthn/credentials*`, `/api/due-dates`).
  - CI wiring: new job in `.github/workflows/ci.yml` after backend build.
- Add version drift check script.
  - Tool: `scripts/check-doc-version-sync.sh`.
  - Behavior: compare `backend/go.mod` Go version and key dependency versions with claims in `README.md` and `docs/architecture.md`.
  - CI wiring: `.github/workflows/quality.yml` and Lefthook pre-commit for changed docs.
- Enforce Markdown link and anchor integrity for API docs.
  - Existing: `markdownlint` + `lychee` are already active.
  - Add: anchor check for local section links in `docs/api.md` (custom script using `markdown-toc`/`remark` parser).
- Keep docs examples executable where possible.
  - Add `docs/examples/` with runnable request fixtures and a smoke script in CI.
