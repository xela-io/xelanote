# Audit TODO Checklist

- [x] Owner: docs | Effort: S | Risk: low | Task: Update `docs/environment-variables.md` wording for `CORS_ALLOWED_ORIGINS` to clarify "strongly recommended" vs "required."
- [x] Owner: backend | Effort: S | Risk: low | Task: Replace duplicated note field validation with a shared helper (keep error strings stable).
- [x] Owner: backend | Effort: S | Risk: low | Task: Consolidate journal feature checks via a shared helper or middleware.
- [x] Owner: backend | Effort: S | Risk: low | Task: Use `parseETag` helper or remove it if unused, and add a unit test for ETag parsing.
- [x] Owner: backend | Effort: M | Risk: medium | Task: Standardize JSON error responses across API handlers (`respondError` everywhere) and validate client compatibility.
- [x] Owner: devops | Effort: M | Risk: low | Task: Add CI workflow to run `make quality`, `make test`, `npm run test`, and `npm run test:e2e`.
- [x] Owner: devops | Effort: S | Risk: low | Task: Add markdown lint + link check in CI for `README.md` and `docs/**`.
- [x] Owner: docs | Effort: S | Risk: low | Task: Add a short README link to `docs/offline-mode.md` for offline details.
