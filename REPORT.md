# Repository Audit Report (xelanote)

## Architecture Map

1. Backend entry point is `backend/cmd/server/main.go`, which loads env/flags, opens SQLite, wires services, starts jobs/websocket, and boots the HTTP router. Evidence: `backend/cmd/server/main.go:32-241`.
2. HTTP API layer lives in `backend/internal/api` with routing, middleware, handlers, and JSON helpers. Evidence: `backend/internal/api/api.go:560-633` and `backend/internal/api/*.go`.
3. Domain logic is organized in `backend/internal/service`, called by API handlers. Evidence: `backend/cmd/server/main.go:101-239`.
4. Persistence and migrations are in `backend/internal/db` and `backend/internal/db/migrations`. Evidence: `backend/cmd/server/main.go:84-93`, `backend/internal/db/migrations`.
5. Authentication is split across `backend/internal/auth` (JWT/session) and `backend/internal/fido2` (WebAuthn). Evidence: `backend/cmd/server/main.go:102-227`, `backend/internal/auth`, `backend/internal/fido2`.
6. Background jobs are handled by `backend/internal/jobs` with a job manager started in main. Evidence: `backend/cmd/server/main.go:129-133`.
7. WebSocket real-time updates run via `backend/internal/websocket`. Evidence: `backend/cmd/server/main.go:192-195`.
8. LLM features are in `backend/internal/llm` and wired via `service.NewSummarizeService`. Evidence: `backend/cmd/server/main.go:125-127`.
9. Frontend is a SvelteKit app under `frontend/src` with routes in `frontend/src/routes` and shared logic in `frontend/src/lib`. Evidence: `frontend/src`, `frontend/src/routes`, `frontend/src/lib`.
10. Offline capabilities are implemented in `frontend/src/lib/offline` with IndexedDB queueing. Evidence: `frontend/src/lib/offline/offline-queue.ts:1-148`.
11. Desktop builds target Electron (`frontend/src-electron`, `frontend/electron-builder.yml`) and Tauri (`frontend/src-tauri`). Evidence: `frontend/electron-builder.yml`, `frontend/src-electron`, `frontend/src-tauri`.
12. Frontend build scripts and tests are in `frontend/package.json` (Vite, Vitest, Playwright). Evidence: `frontend/package.json`.
13. Backend build/test commands are centralized in the repo `Makefile`. Evidence: `Makefile:1-120`.
14. Docker build uses a multi-stage root `Dockerfile` to build frontend then backend, embedding `frontend/build` into `backend/cmd/server/static`. Evidence: `Dockerfile:1-50`.
15. Environment configuration is documented in `docs/environment-variables.md` and enforced in main for critical values like `JWT_SECRET`. Evidence: `docs/environment-variables.md:1-79`, `backend/cmd/server/main.go:38-51`.

## Top Redundancies (Prioritized)

1. Repeated note field validation across `createNote`, `updateNote`, and decrypt flows. Evidence: `backend/internal/api/notes.go:158-172`, `backend/internal/api/notes.go:411-425`, `backend/internal/api/notes.go:622-630`. Consolidation target: new helper in `backend/internal/api/notes_validation.go` used by `createNote`, `updateNote`, and decrypt handler. Risk: low. Effort: S. Refactor sketch: `func validateNoteFields(title, content, folderPath string, allowFolder bool) error` and call it before encrypted/plaintext branching in each handler.
2. Journal feature gate repeated in every journal handler. Evidence: `backend/internal/api/journal.go:35-43`, `backend/internal/api/journal.go:70-78`, `backend/internal/api/journal.go:102-110`, `backend/internal/api/journal.go:147-155`. Consolidation target: helper `requireJournalFeature(userID string) error` or route-group middleware under `backend/internal/api`. Risk: low. Effort: S. Refactor sketch: `func (s *Server) requireFeature(userID, feature string) error` then call in each handler or attach middleware to `/journal` routes.
3. ETag parsing logic is duplicated while a helper exists but is unused. Evidence: helper `parseETag` defined in `backend/internal/api/notes.go:124-141`, manual parsing logic in `backend/internal/api/notes.go:372-403` and similar checks around encrypted/decrypt flows `backend/internal/api/notes.go:600-612`. Consolidation target: use `parseETag` in all ETag checks or remove the unused helper. Risk: low. Effort: S. Refactor sketch: replace manual parsing with `parseETag(ifMatch, existingNote.ID, existingNote.Version)` and unify error mapping.
4. Error responses are inconsistent between handlers (`http.Error` vs `respondError`), duplicating response formatting logic. Evidence: `backend/internal/api/graph.go:33-43` vs helpers in `backend/internal/api/api.go:607-616`. Consolidation target: standardize on `respondError` and `respondJSON` across handlers. Risk: medium (response shape change). Effort: S/M. Refactor sketch: replace `http.Error` usages with `respondError` and ensure clients tolerate JSON error bodies.

## Doc Inconsistencies (Open)

1. Docs say `CORS_ALLOWED_ORIGINS` is required in production, but code allows permissive mode and only logs a warning. Evidence: `docs/environment-variables.md:9-10` vs `backend/internal/api/api.go:560-571`. Proposed fix: clarify that it is “strongly recommended” rather than “required” unless you enforce a hard fail in production.

## Resolved Doc Inconsistencies (Fixed In This Audit)

1. README theme count now matches the two themes exposed in code. Evidence: `README.md:47` and `frontend/src/lib/themes/index.ts:3-28`.
2. README offline mode now reflects read + limited write (phase 1) consistent with docs and code. Evidence: `README.md:50`, `docs/offline-mode.md:3-43`, `frontend/src/lib/offline/offline-queue.ts:1-148`.
3. Environment variable reference now includes WebAuthn and Forgejo error reporting settings used by the server. Evidence: `docs/environment-variables.md:23-51`, `backend/cmd/server/main.go:176-232`.

## Quick Wins (Under 1 Day)

1. Update README theme count and offline mode description. Implemented in `README.md`. Evidence: `README.md:47-50`.
2. Add missing env vars (WebAuthn, Forgejo error reporting) to `docs/environment-variables.md`. Implemented in `docs/environment-variables.md:23-51`.
3. Deduplicate IndexedDB store creation in offline queue. Implemented in `frontend/src/lib/offline/offline-queue.ts:24-70`.

## Medium/Large Refactors (With Migration Plan)

1. Centralize request validation for note operations. Scope: `createNote`, `updateNote`, decrypt, and any bulk/rename flows that validate title/content/folder path. Evidence: `backend/internal/api/notes.go:158-172`, `backend/internal/api/notes.go:411-425`, `backend/internal/api/notes.go:622-630`. Migration plan: add helper `validateNoteFields(...)` with unit tests, update one handler at a time, keep existing error messages for compatibility. Rollback: revert handler-specific calls to inline validation if errors diverge.
2. Standardize error response format across all API handlers. Scope: replace `http.Error` with `respondError` and ensure all responses are JSON. Evidence: `backend/internal/api/graph.go:33-43`, `backend/internal/api/api.go:607-616`. Migration plan: introduce `respondError` in handlers behind a feature flag or only for endpoints with known JSON clients, then expand after client validation. Rollback: revert individual handlers to `http.Error` where clients break.

## Guardrails To Prevent Drift

1. ~~Add Markdown linting to CI (`markdownlint-cli2`) for `README.md` and `docs/**` to catch formatting drift.~~ **DONE**: `.markdownlint.yaml` config (structural rules only), `make lint-md` target, integrated in GitHub Actions quality.yml and lefthook pre-commit.
2. ~~Add link checking (`lychee` or `remark-validate-links`) to prevent stale URLs in docs.~~ **DONE**: `.lychee.toml` config, `docs` job in GitHub Actions quality.yml with `lychee-action@v2`.
3. Add "docs as tests" for command snippets using `mdbook` test harness or a lightweight script that extracts fenced `bash` blocks and runs them in a dry-run mode. **(DEFERRED - low priority)**
4. ~~Add pre-commit hooks for `gofmt`, `go vet`, `eslint`, and `prettier` to keep docs/code aligned with formatting rules.~~ **DONE**: `lefthook.yml` with parallel pre-commit hooks (gofmt, go-vet, eslint, prettier, markdownlint). Auto-installed via `make init`.
5. ~~Wire CI to run `make quality`, `make test`, `npm run test`, and `npm run test:e2e` on PRs.~~ **DONE**: Forgejo staging deploy now runs backend tests + frontend lint before deploying. GitHub Actions quality.yml has frontend (format, lint, markdownlint, typecheck), backend (gofmt, vet, typecheck), docs (lychee), and changelog check jobs.
6. **NEW**: CHANGELOG update check in PRs - warns if CHANGELOG.md wasn't modified (skips for docs-only and test-only changes).

## Notes

- Tests were not run as part of this audit.
