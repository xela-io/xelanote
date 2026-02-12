# Full Repository Audit Report

**Project:** xelanote (Self-hosted encrypted note-taking)
**Codebase:** ~45k LOC Go backend, ~49k LOC Svelte/TS frontend, 54 documentation files

### Revision History

| Pass | Date | Scope | Auditor |
|------|------|-------|---------|
| 1 | 2026-02-10 | Initial: architecture, redundancies R-01..R-10, doc inconsistencies D-01..D-08 | Claude |
| 2 | 2026-02-10 | Fixes applied, guardrails added, items resolved | Claude |
| 3 | 2026-02-12 | Re-audit: new redundancies R-11..R-14, doc findings D-09..D-11, report QA | Claude |
| 4 | 2026-02-12 | Quick Wins implemented: parseIntParam (#14), env.go (#15), validationError move (#16), graph TTL fix (#18), CLAUDE.md (#19), Makefile (#20) | Claude |

---

## 0. Methodology & Validation

### Approach
- **Codebase mapping:** Parallel agents explored backend (`internal/api/`, `internal/service/`, `internal/db/`, `cmd/server/`), frontend (`src/routes/`, `src/lib/`), docs, and CI/CD configs independently.
- **Redundancy detection:** `grep -c` for repeated patterns (getUserID: 145 hits/57 files, respondError+invalid id: 31 hits/13 files, strconv.Atoi+chi.URLParam: 16 hits), manual review of structurally similar files (snippets.go vs templates.go, dialog components).
- **Doc verification:** Claims in README, CLAUDE.md, conventions.md, environment-variables.md, deployment.md cross-checked against Makefile targets, Dockerfile, docker-compose.yml, CI workflows, and actual Go/Svelte source.

### What Was NOT Done
- No runtime testing (no `make test` or `make test-e2e` executed).
- No load/performance testing.
- No security penetration testing (separate `docs/security_audit_findings.md` exists).
- Frontend component duplication assessed by structure comparison, not AST diffing.

---

## 1. Architecture Map (20 Bullets)

1. **Entry point:** `backend/cmd/server/main.go` -- loads env/flags, opens SQLite, wires 15+ services, starts jobs/websocket, boots Chi router. Evidence: `main.go:32-371`.
2. **HTTP layer:** `backend/internal/api/` -- `Server` struct with 30+ fields, `setupRoutes()` registers 100+ routes, JSON helpers (`respondJSON`, `respondError`, `decodeJSON`). Evidence: `api.go:1-640`.
3. **Business logic:** `backend/internal/service/` -- 15 service types (NoteService, AuthService, RecipeService, SharingService, etc.) injected into API layer. Evidence: `main.go:101-239`.
4. **Persistence:** `backend/internal/db/` -- SQLite with FTS5, 45 migrations in `db/migrations/`, optional SQLCipher encryption. Evidence: `main.go:84-93`, `db/migrations/`.
5. **Auth stack:** JWT HS256 access+refresh tokens, bcrypt passwords, TOTP 2FA, FIDO2/WebAuthn, account lockout, rate limiting. Evidence: `api/auth_login.go`, `service/auth.go`, `fido2/`.
6. **Background jobs:** `backend/internal/jobs/` -- worker pool (4 workers), currently handles note rename jobs. Evidence: `main.go:137-140`.
7. **WebSocket:** `backend/internal/websocket/` -- real-time note/recipe updates via Manager pattern. Evidence: `main.go:199-202`.
8. **LLM integration:** `backend/internal/llm/` -- provider router (Claude/Gemini), text transformations, recipe suggestions. Evidence: `main.go:133-134`.
9. **Frontend framework:** SvelteKit with adapter-static, Svelte 5 runes ($state, $derived, $effect). Evidence: `frontend/svelte.config.js`.
10. **Editor:** CodeMirror 6 with custom extensions (spell-check, task toggle, due dates, wikilinks). Evidence: `frontend/src/lib/editor/` (25+ modules).
11. **Offline:** PWA Service Worker + IndexedDB queue with background sync and conflict resolution. Evidence: `frontend/src/lib/offline/`.
12. **E2E encryption:** XChaCha20-Poly1305 + Argon2id key derivation, vault lock/unlock pattern. Evidence: `frontend/src/lib/stores/encryption.svelte.ts`.
13. **i18n:** ICU MessageFormat, ~604 keys per locale (en/de), reactive via svelte-i18n. Evidence: `frontend/src/lib/locales/`.
14. **Desktop targets:** Electron (`frontend/src-electron/`) and Tauri (`frontend/src-tauri/`) -- both present but maturity unclear. Evidence: `frontend/electron-builder.yml`, `frontend/src-tauri/`.
15. **Build system:** Root `Makefile` with 20+ targets (build, dev, test, lint, quality, docker, demo-db). Evidence: `Makefile:1-142`.
16. **Docker:** Multi-stage `Dockerfile` (node:22 -> Go 1.24 -> alpine:3.20), embeds SvelteKit into Go binary. Evidence: `Dockerfile:1-76`.
17. **CI/CD:** GitHub Actions (`ci.yml`) + Forgejo Actions (`deploy-staging.yml`). Evidence: `.github/workflows/`, `.forgejo/workflows/`.
18. **Pre-commit:** lefthook with gofmt, go vet, ESLint, Prettier, markdownlint. Evidence: `lefthook.yml`.
19. **Config:** 15+ env vars loaded in main.go, documented in `docs/environment-variables.md`. Evidence: `main.go:38-238`, `docs/environment-variables.md`.
20. **Data model:** Notes (with note_type: note/journal/recipe), Folders, Tags, Links, Versions, Shares, Collections, Users, Activity. Evidence: `db/migrations/`.

---

## 2. Redundancies (R-01 through R-14)

### Pass 1 Findings (R-01..R-10)

#### R-01: Auth Boilerplate (getUserID pattern) -- HIGH IMPACT
- **Where:** 145 occurrences across 57 handler files in `api/*.go`
- **Pattern:** Every handler starts with `userID, ok := getUserID(r); if !ok { respondError(...); return }`
- **Consolidation:** Middleware wrapper that injects userID, handlers receive it as parameter
- **Risk:** Medium (touches 57 files) | **Effort:** L
- **Status:** SKIPPED -- idiomatic Go, marginal gain vs. migration effort

#### R-02: Snippet/Template Near-Identical CRUD -- MEDIUM IMPACT
- **Where:** `api/templates.go`, `api/snippets.go`, `service/template.go`, `service/snippet.go`, `db/templates.go`, `db/snippets.go`
- **Pattern:** Create/List/Get/Update/Delete handlers, service methods, and DB queries are structurally identical across all 3 layers
- **Consolidation:** Generic CRUD helper or shared interface for simple key-value entities
- **Risk:** Low | **Effort:** M
- **Status:** SKIPPED -- only 2 resources, break-even at 12+

#### R-03: JSON Decode Pattern -- MEDIUM IMPACT
- **Where:** 50+ handlers across `api/*.go`
- **Pattern:** `if err := decodeJSON(r, &req); err != nil { respondError(w, http.StatusBadRequest, "Invalid JSON"); return }`
- **Consolidation:** Already have `decodeJSON` helper, but error handling is still repeated. Could use a middleware or functional wrapper.
- **Risk:** Low | **Effort:** S

#### R-04: Note Field Validation Duplication -- RESOLVED
- **Pattern:** Title/content/folder validation repeated in createNote, updateNote, and decrypt flows
- **Resolution:** `validateNoteFields()` helper already exists at `api/notes_helpers_validate.go:13`

#### R-05: Journal Feature Gate Repeated -- RESOLVED
- **Pattern:** Same `isFeatureEnabled("journal")` check in every handler
- **Resolution:** `requireJournalFeature` helper already exists

#### R-06: ETag Parsing Logic Duplication -- RESOLVED
- **Pattern:** Helper exists but was not used everywhere
- **Resolution:** Fixed in Pass 2 -- `parseETag` now used in all ETag-checking paths including `versions.go`

#### R-07: Error Response Inconsistency -- RESOLVED
- **Pattern:** Some handlers used `http.Error()` (plaintext), others `respondError()` (JSON)
- **Resolution:** Verified no `http.Error` calls remain; all use `respondError()`

#### R-08: Frontend Store Getter Functions -- LOW IMPACT
- **Where:** 163+ trivial getter functions across `frontend/src/lib/stores/*.svelte.ts`
- **Pattern:** `function getX() { return stateX; }` wrapping direct state access
- **Consolidation:** Export $state directly, or use Svelte 5 `$derived` where appropriate
- **Risk:** Low (but touches many files) | **Effort:** M/L

#### R-09: Sharing Dialog Duplication -- RESOLVED
- **Pattern:** 3 nearly identical dialogs with user search, role selection, share management
- **Resolution:** Consolidated into generic `ShareDialog.svelte` (-400 lines)

#### R-10: Encryption Payload Creation -- LOW IMPACT
- **Where:** `frontend/src/lib/stores/notes.svelte.ts` -- repeated 4x
- **Pattern:** Same encrypt-and-build-payload logic for create, update, save, batch operations
- **Consolidation:** Extract `buildEncryptedPayload(note, content, vaultKey)` helper
- **Risk:** Low | **Effort:** S

### Pass 3 Findings (R-11..R-14)

#### R-11: URL Parameter ID Parsing Boilerplate -- HIGH IMPACT

**Evidence:** 31 occurrences of `respondError(w, http.StatusBadRequest, "invalid ... id")` across 13 files, 16 of which are in recipe handlers with identical `strconv.Atoi(chi.URLParam(...))` blocks.

**Affected files:** `recipes_shared.go`, `recipes_collections.go`, `recipes_collection_shares.go`, `recipes_images.go`, `folders_crud.go`, `folders_encryption.go`, `folders_color.go`, `folders_ai.go`, `snippets.go`, `templates.go`, `tags.go`, `uploads.go`, `fido2.go`

**Repeated pattern (5 lines each):**
```go
collID, err := strconv.Atoi(chi.URLParam(r, "id"))
if err != nil {
    respondError(w, http.StatusBadRequest, "invalid collection id")
    return
}
```

**Consolidation:** Add to `response_helpers.go`:
```go
func parseIntParam(w http.ResponseWriter, r *http.Request, param string) (int, bool) {
    v, err := strconv.Atoi(chi.URLParam(r, param))
    if err != nil {
        respondError(w, http.StatusBadRequest, "invalid "+param)
        return 0, false
    }
    return v, true
}
```

| Risk | Effort | Lines saved |
|------|--------|-------------|
| Low | S (~30min) | ~120 lines |

#### R-12: isDevelopment() vs. isTestEnv() -- Overlapping Env Checks -- MEDIUM IMPACT

**Evidence:** Two separate functions reading `XELANOTE_ENV` with overlapping but different semantics:

```go
// cookies.go:15 -- includes "test"/"testing" in "development"
func isDevelopment() bool {
    env := os.Getenv("XELANOTE_ENV")
    return env == "development" || env == "test" || env == "testing"
}

// rate_limits.go:27 -- test-only check
func isTestEnv() bool {
    env := os.Getenv("XELANOTE_ENV")
    return env == "test" || env == "testing"
}
```

**Problem:** `isDevelopment()` returns true for test environments, which is semantically misleading.

**Consolidation:** Create `env.go`:
```go
func getEnvMode() string { return os.Getenv("XELANOTE_ENV") }
func isDevMode() bool    { return getEnvMode() == "development" }
func isTestMode() bool   { m := getEnvMode(); return m == "test" || m == "testing" }
func isNonProd() bool    { return isDevMode() || isTestMode() }
```

| Risk | Effort | Lines saved |
|------|--------|-------------|
| Low | S (~20min) | ~10 lines, clearer semantics |

#### R-13: validationError Type Misplaced -- LOW IMPACT

**Evidence:** `validationError` struct defined in `templates.go:51` but also used by `snippets.go:37`. Same package so it compiles, but ownership is misleading.

**Fix:** Move to `validation.go` (which already exists for encryption validation).

| Risk | Effort |
|------|--------|
| Low | S (~10min) |

#### R-14: Frontend Input Dialog Component Duplication -- HIGH IMPACT

**Evidence:** 5 dialog components with ~80% code overlap:

| Component | Lines | Key Pattern |
|-----------|-------|-------------|
| `RenameNoteDialog.svelte` | 110 | $state + validation + keydown + error display |
| `RenameFolderDialog.svelte` | 119 | identical structure |
| `CreateFolderDialog.svelte` | ~100 | identical structure |
| `CreateNoteDialog.svelte` | ~100 | identical structure |
| `RecipeCollectionDialog.svelte` | ~90 | identical structure |

**Duplicated sub-patterns:**
1. State management triplet (`$state('')`, `$state(false)`, `$state<string|null>(null)`) in all 5
2. Error display markup (`text-red-600 bg-red-50`) in 5 files
3. Input field classes in 8 files (11 total occurrences of identical `w-full px-3 py-2 bg-background border...`)

**Consolidation:** Extract `<InputDialog>` component with configurable title, label, onSubmit, onClose.

| Risk | Effort | Lines saved |
|------|--------|-------------|
| Low | M (~2-3h) | ~300 lines |

---

## 3. Documentation Inconsistencies (D-01 through D-11)

### Pass 1 Findings (D-01..D-08)

| # | Issue | Severity | Status |
|---|-------|----------|--------|
| D-01 | `.env.example` JWT_SECRET said "min 32 chars", code requires 64 | CRITICAL | RESOLVED (Pass 1) |
| D-02 | `.env.example` referenced non-existent PORT env var | MEDIUM | RESOLVED (Pass 1) |
| D-03 | `.env.example` only had 4 of 15+ env vars | MEDIUM | RESOLVED (Pass 1) |
| D-04 | Two conflicting Dockerfiles (root vs. `backend/`) | HIGH | RESOLVED (Pass 2): `backend/Dockerfile` deleted |
| D-05 | CHANGELOG duplicate `### Added/Changed/Fixed` headers | MEDIUM | RESOLVED (Pass 1) |
| D-06 | CORS_ALLOWED_ORIGINS docs said "required" but code only warned | LOW | RESOLVED (Pass 2): clarified as log.Fatal in prod |
| D-07 | README theme count mismatch | LOW | RESOLVED (pre-audit) |
| D-08 | TODO.md contained operational test instructions | LOW | RESOLVED (Pass 2): moved to `docs/runbooks/` |

### Pass 3 Findings (D-09..D-11)

#### D-09: CLAUDE.md Build Tags Description -- LOW

**CLAUDE.md line 17** says: `Backend immer mit FTS5: -tags "fts5" (SQLCipher optional: sqlite_crypt)`

**Reality:**
- `Makefile:18` always uses `-tags "fts5 sqlite_crypt"` for local builds
- `Dockerfile:38` uses `-tags "fts5"` only for Docker builds

**Fix:** Clarify that Makefile always includes both, Docker omits sqlite_crypt by default.

#### D-10: GraphService Comment Claims Wrong TTL -- LOW

**`backend/internal/service/graph.go:18`**: "NewGraphService creates a new GraphService with its own cache (2 min TTL)."
**Line 21**: "Note: Using shared cache from NoteService" (contradicts "its own cache")
**Line 33**: "GetGlobalGraph returns the global graph for a user with caching (2 min TTL)."

**Reality:** Uses shared NoteService cache with 5 min TTL.

**Fix:** Update comments to "shared cache" and remove "2 min TTL" references.

#### D-11: `make dev-full` Is Informational Only -- LOW

`Makefile:77-80` target only prints instructions, doesn't actually launch anything.

**Fix:** Add comment above target clarifying it's informational.

---

## 4. Consolidated Status Tracker (Single Source of Truth)

This is the **only** status table. Items from Quick Wins, Medium Refactors, and Guardrails are merged here.

| # | Item | Ref | Pass | Effort | Priority | Status |
|---|------|-----|------|--------|----------|--------|
| 1 | Fix `.env.example` JWT_SECRET (32->64) | D-01 | 1 | 5 min | P0 | DONE |
| 2 | Remove non-existent PORT from `.env.example` | D-02 | 1 | 2 min | P0 | DONE |
| 3 | Expand `.env.example` with all env vars | D-03 | 1 | 15 min | P1 | DONE |
| 4 | Consolidate CHANGELOG duplicate section headers | D-05 | 1 | 30 min | P1 | DONE |
| 5 | Use `parseETag` helper in all ETag paths | R-06 | 2 | 30 min | P2 | DONE |
| 6 | Resolve Dockerfile conflict (delete `backend/Dockerfile`) | D-04 | 2 | 15 min | P1 | DONE |
| 7 | Clarify CORS docs wording | D-06 | 2 | 5 min | P2 | DONE |
| 8 | Move salt test from TODO.md to `docs/runbooks/` | D-08 | 2 | 10 min | P3 | DONE |
| 9 | Unify sharing dialogs -> generic `ShareDialog.svelte` | R-09 | 2 | 1h | P3 | DONE |
| 10 | Add env var sync CI check | Guard. | 2 | 30 min | P2 | DONE |
| 11 | Add bundle size tracking to CI | Guard. | 2 | 20 min | P3 | DONE |
| 12 | Auth context middleware | R-01 | 1 | L | P3 | SKIPPED (idiomatic Go) |
| 13 | Generic CRUD for templates/snippets | R-02 | 1 | M | P3 | SKIPPED (only 2 resources) |
| 14 | Add `parseIntParam()` helper | R-11 | 3 | S (30min) | P1 | DONE (Pass 4) |
| 15 | Consolidate env check functions into `env.go` | R-12 | 3 | S (20min) | P2 | DONE (Pass 4) |
| 16 | Move `validationError` to `validation.go` | R-13 | 3 | S (10min) | P3 | DONE (Pass 4) |
| 17 | Extract `<InputDialog>` component | R-14 | 3 | M (2-3h) | P2 | TODO |
| 18 | Fix GraphService stale TTL comments | D-10 | 3 | S (5min) | P3 | DONE (Pass 4) |
| 19 | Clarify CLAUDE.md build tags text | D-09 | 3 | S (5min) | P3 | DONE (Pass 4) |
| 20 | Add comment to `make dev-full` target | D-11 | 3 | S (2min) | P3 | DONE (Pass 4) |
| 21 | Add `deadcode` analyzer to CI | Guard. | 3 | S (15min) | P2 | TODO |
| 22 | Add `dupl` linter to CI | Guard. | 3 | S (15min) | P3 | TODO |

**Summary:** 22 total items. 17 done, 2 skipped, 3 open.

---

## 5. Medium/Large Refactors

### MR-1: Auth Middleware (R-01) -- SKIPPED
- **Rationale:** 145 handlers across 57 files. Go convention is explicit error handling. Migration cost high, readability gain marginal.

### MR-2: Generic CRUD for Templates/Snippets (R-02) -- SKIPPED
- **Rationale:** Only 2 resources share this pattern. Generic CRUD abstractions pay off at ~5+ resources. Not worth the indirection cost.

### MR-3: Unified Share Dialog (R-09) -- DONE
- **Result:** 3 dialogs -> 1 generic `ShareDialog.svelte`, ~400 lines removed.

### MR-4: Resolve Dockerfile Conflict (D-04) -- DONE
- **Result:** `backend/Dockerfile` deleted. SQLCipher instructions as comments in root `Dockerfile`.

### MR-5: Input Dialog Consolidation (R-14) -- TODO
- **Scope:** 5 dialog components -> 1 `<InputDialog>` + per-dialog thin wrappers
- **Migration:** Create base component, migrate RenameNoteDialog first as reference, then remaining 4 one at a time. Run E2E tests after each.
- **Rollback:** Each dialog can be reverted independently.
- **Effort:** M (2-3h) | **Risk:** Low

---

## 6. Guardrails (Existing + Recommended)

### Existing (Already in Place)
- lefthook pre-commit hooks (gofmt, go vet, ESLint, Prettier, markdownlint)
- GitHub Actions ci.yml (frontend format/lint/typecheck, backend vet/typecheck, docs lychee, changelog check)
- Forgejo Actions staging deploy (backend tests + frontend lint before deploy)
- markdownlint-cli2 config (`.markdownlint.yaml`)
- lychee link checker config (`.lychee.toml`)
- Env var sync check (`scripts/check-env-sync.sh`) -- added in Pass 2
- Bundle size tracking (`scripts/check-bundle-size.sh`, budget: 3600 KB) -- added in Pass 2

### Recommended Additions (Pass 3)
1. **Dead code detection:** `deadcode` analyzer in Go CI (see item #21)
2. **Duplicate code detection:** `dupl` linter with 50-token threshold (see item #22)
3. **CHANGELOG structure lint:** Script to verify each version has at most one of each section type

### Concrete CI Addition:
```yaml
# Add to .github/workflows/ci.yml
dead-code:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.24'
    - run: go install golang.org/x/tools/cmd/deadcode@latest
    - run: cd backend && deadcode -tags "fts5 sqlite_crypt" ./...
```

---

## 7. Overall Assessment

**Strengths:**
- Clean layered architecture (API -> Service -> DB) consistently followed
- Comprehensive security: JWT rotation, bcrypt, 2FA (TOTP + FIDO2), rate limiting, account lockout, CSRF, CSP headers, Docker hardening
- Excellent documentation coverage: 54 doc files, detailed env var reference, deployment guide, planning docs
- Strong CI/CD: dual pipeline (GitHub + Forgejo), pre-commit hooks, quality gates
- Feature-rich: offline mode, E2E encryption, i18n, graph view, recipes, sharing -- all well-integrated

**Areas for Improvement:**
- `.env.example` was dangerously out of sync with code (fixed in Pass 1+2)
- CHANGELOG accumulated format drift (fixed in Pass 1)
- Backend has typical Go boilerplate duplication (auth check, ID parsing) -- R-01 accepted as idiomatic, R-11 fixed in Pass 4
- Frontend has dialog component duplication (~300 lines across 5 components) -- R-14 open
- Overlapping env-check functions -- fixed in Pass 4 (consolidated into `env.go`)

**Risk Assessment:** No critical runtime issues found. All findings are maintainability and documentation concerns. The codebase is well-structured and production-ready.

### Per-Pass Summary

**Pass 1+2 (2026-02-10):** 16 items identified. 11 actively fixed, 2 skipped as not cost-effective, 3 found already resolved in codebase. Guardrails added: env var sync check, bundle size tracking.

**Pass 3 (2026-02-12):** 9 new items identified (4 redundancies, 3 doc findings, 2 guardrails). All previous resolutions verified. Pass 3 doc findings are all cosmetic (stale comments, clarification text) -- unlike Pass 1 which had the critical D-01 JWT_SECRET mismatch.

**Pass 4 (2026-02-12):** 6 quick wins implemented. `parseIntParam()` helper added to `response_helpers.go`, ~35 call sites refactored across 13 files (~105 lines reduced). `isDevelopment()` and `isTestEnv()` consolidated into `env.go` with shared `xelanoteEnv()` helper. `validationError` moved from `templates.go` to `validation.go`. GraphService stale TTL comments corrected. CLAUDE.md build tags clarified. Makefile `dev-full` comment added. Remaining open: R-14 (InputDialog component, M effort), #21 (deadcode CI), #22 (dupl CI).
