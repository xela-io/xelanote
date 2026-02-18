# Audit TODO Checklist

- [x] Owner: docs | Effort: S | Risk: low | Task: Fix `docs/api.md` auth/public endpoint rules and content-type exceptions.
- [x] Owner: docs | Effort: S | Risk: low | Task: Update `GET /api/config` docs to include `version`, `error_reporting_enabled`, and `captcha_iframe_url`.
- [x] Owner: docs | Effort: S | Risk: low | Task: Correct WebAuthn delete/touch request/response contracts in `docs/api.md`.
- [x] Owner: docs | Effort: S | Risk: low | Task: Fix API ToC drift (`/api/uploads/{user_id}/{filename}` and broken shared-folder anchor).
- [x] Owner: docs | Effort: S | Risk: low | Task: Align Tauri dev port docs (`docs/desktop-app.md`) with `frontend/src-tauri/tauri.conf.json`.
- [x] Owner: docs | Effort: S | Risk: low | Task: Align Go version docs in `README.md`, `docs/development.md`, and `docs/architecture.md` with `backend/go.mod`.
- [x] Owner: docs+backend | Effort: S | Risk: low | Task: Add missing API docs for `GET /api/due-dates`, `POST /api/perf-metrics`, and `POST /api/analytics/events`.
- [x] Owner: platform | Effort: M | Risk: low | Task: Add `scripts/check-api-doc-coverage.sh` and wire it into `.github/workflows/quality.yml` + `lefthook.yml`.
- [x] Owner: backend | Effort: S | Risk: low | Task: Centralize template/snippet size constants into one shared constraints module.
- [x] Owner: frontend | Effort: S | Risk: low | Task: Introduce shared query helper in `frontend/src/lib/api/query.ts` and migrate `notes.ts`, `trash.ts`, `graph.ts`, `admin.ts`, `versions.ts`.
- [x] Owner: backend | Effort: M | Risk: medium | Task: Extract shared share-input validation/error-mapping helper for notes/folders/collection handlers.
- [x] Owner: backend | Effort: M | Risk: medium | Task: Add ownership guard helpers in `sharing.go` and `recipes_collections.go`; migrate methods incrementally.
- [x] Owner: qa+backend | Effort: M | Risk: medium | Task: Add endpoint contract smoke tests for documented examples in CI.
