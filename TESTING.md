# Testing Guide

Overview of the xelanote test infrastructure and how to run tests.

## Quick Reference

```bash
make test            # Backend unit tests (Go)
make test-frontend   # Frontend unit tests (Vitest)
make test-e2e        # E2E tests (Playwright)
make test-parser     # Parser tests only
make bench-parser    # Parser benchmarks
make lint            # Linting (go vet + ESLint)
make fmt             # Format code (gofmt + prettier)
make fmt-check       # Check formatting (gofmt + prettier --check)
make typecheck       # Type checking (Go compile + svelte-check)
make quality         # Format check + lint + typecheck
```

## Backend Tests (Go)

**Framework:** Go standard `testing` package + [Testify](https://github.com/stretchr/testify) for assertions

**Build tags:** All backend test commands require `-tags "fts5 sqlite_crypt"` for SQLite FTS5 and SQLCipher support.

### Run all backend tests

```bash
make test
# or manually:
cd backend && go test -tags "fts5 sqlite_crypt" -v ./...
```

### Run tests for a specific package

```bash
cd backend && go test -tags "fts5" -v ./internal/db/...
cd backend && go test -tags "fts5" -v ./internal/service/...
cd backend && go test -tags "fts5" -v ./internal/api/...
cd backend && go test -tags "fts5" -v ./internal/parser/...
cd backend && go test -tags "fts5" -v ./internal/auth/...
```

### Run a specific test

```bash
cd backend && go test -tags "fts5" -v ./internal/service -run "TestChangePassword"
```

### Test with coverage

```bash
cd backend && go test -tags "fts5" -cover ./...
```

### Test files

| Package | Tests | What they cover |
|---------|-------|-----------------|
| `internal/db/` | notes, folders, search, graph, versions, preferences, features, journal | Database layer CRUD, queries, migrations |
| `internal/service/` | auth, user, notes_cache, notes_links | Business logic, caching, link processing |
| `internal/api/` | security, ratelimit, lockout, uploads, twofa, auth_salt | HTTP handlers, middleware, rate limiting |
| `internal/parser/` | wikilink, clean | Markdown/wikilink parsing |
| `internal/auth/` | upload_signature | Signed URL generation |

**Test data:** JSON test vectors for the wikilink parser in `testdata/parser/` (basic, aliases, edge cases, unicode, code blocks, stress tests).

## Frontend Unit Tests (Vitest)

**Framework:** Vitest with `@testing-library/svelte`
**Environment:** jsdom
**Config:** `frontend/vite.config.ts`
**Setup:** `frontend/src/test/setup.ts` (mocks localStorage, auto-cleanup)

### Run all frontend tests

```bash
make test-frontend
# or manually:
cd frontend && npm run test
```

### Run in watch mode

```bash
cd frontend && npx vitest --watch
```

### Run a specific test file

```bash
cd frontend && npx vitest src/lib/stores/tree.test.ts
```

### Test files

| File | What it covers |
|------|----------------|
| `src/lib/stores/tree.test.ts` | Folder tree state management |
| `src/lib/stores/folders.test.ts` | Folder store operations |
| `src/lib/stores/toast.test.ts` | Toast notification system |
| `src/lib/stores/token-refresh.test.ts` | Token refresh logic with retry/backoff |
| `src/lib/crypto/sodium.test.ts` | Cryptographic operations |
| `src/lib/crypto/kek-persistence.test.ts` | KEK storage/retrieval |
| `src/lib/editor/markdown.test.ts` | Markdown rendering and wikilinks |
| `src/lib/utils/task-reorder.test.ts` | Task list drag & drop reordering |
| `src/lib/themes/index.test.ts` | Theme system validation |
| `src/lib/components/NoteItem.test.ts` | Note list item component |
| `src/test/e2e-feature.test.ts` | Feature-level integration tests |

## E2E Tests (Playwright)

**Framework:** Playwright
**Browser:** Chromium
**Config:** `frontend/playwright.config.ts`
**Timeout:** 60s per test

Playwright automatically starts both backend (in-memory SQLite on port 8080) and frontend (Vite on port 4173).

### Run E2E tests

```bash
make test-e2e
# or manually:
cd frontend && npm run test:e2e
```

### E2E test specs

| File | What it covers |
|------|----------------|
| `tests/e2e/login.spec.ts` | Authentication flows |
| `tests/e2e/2fa.spec.ts` | Two-factor authentication |
| `tests/e2e/notes.spec.ts` | Note CRUD, editing, deletion |
| `tests/e2e/folders.spec.ts` | Folder hierarchy and management |
| `tests/e2e/encryption-security.spec.ts` | End-to-end encryption |
| `tests/e2e/code-splitting.spec.ts` | Bundle code-splitting verification |

**Known issue:** `e2e-feature.test.ts > Scenario 2: Session is persisted` is a pre-existing failure.

## Type Checking

```bash
cd frontend && npm run check    # svelte-check (TypeScript validation)
```

## Parser Benchmarks

```bash
make bench-parser
# or manually:
cd backend && go test -tags "fts5" -bench=. -benchmem ./internal/parser/...
```

## CI/CD (GitHub Actions)

Two workflows run on push to `main` and on pull requests:

**`ci.yml`** — Main pipeline:
1. **backend-tests** — Go tests + `go vet` + format check
2. **security-scan** — `govulncheck` for Go vulnerabilities
3. **frontend-tests** — Vitest + ESLint + svelte-check + Playwright E2E
4. **build** — Full `make build` verification
5. **docker** — Docker image build verification

**`security.yml`** — Security scanning (also weekly):
- govulncheck, npm audit, dependency review, Trivy container scan, CodeQL analysis

## Writing New Tests

### Backend

Place test files next to the code they test with the `_test.go` suffix:

```
backend/internal/db/notes.go       # implementation
backend/internal/db/notes_test.go  # tests
```

Use the standard Go testing patterns with Testify assertions. Tests for the database layer use in-memory SQLite databases via `setupTestDB(t)`.

### Frontend

Place test files next to the code with the `.test.ts` suffix:

```
frontend/src/lib/stores/tree.ts       # implementation
frontend/src/lib/stores/tree.test.ts  # tests
```

Use Vitest with `@testing-library/svelte` for component tests. Mock API calls with `vi.mock('$lib/api')`.

### E2E

Place spec files in `frontend/tests/e2e/` with the `.spec.ts` suffix. Use Playwright's Locator API (`getByRole`, `getByText`, `locator`).
