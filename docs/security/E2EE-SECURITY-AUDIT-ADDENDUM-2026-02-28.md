# E2EE Security Audit Addendum (2026-02-28)

## Scope

This addendum reflects the **current remediation status** of the previously reported E2EE findings in this repository (code-backed snapshot as of 2026-02-28).

## Remediation Status (Code-Verified)

### Closed Findings

1. **AAD binding for note payloads and wrapped DEKs is implemented (v3).**
   - `frontend/src/lib/crypto/e2e.ts:77-83` (AAD construction)
   - `frontend/src/lib/crypto/e2e.ts:226-231` (encrypt with AAD)
   - `frontend/src/lib/crypto/e2e.ts:269-281` (decrypt with AAD)
   - `frontend/src/lib/crypto/sodium.ts:226-231` and `frontend/src/lib/crypto/sodium.ts:280-285` (AAD passed into libsodium AEAD API)

2. **Server-side AI processing is blocked for encrypted notes (summary, tags, links, format, transform).**
   - `backend/internal/service/summarize_service.go:70-73`
   - `backend/internal/api/notes_ai_summary.go:93-95`
   - `backend/internal/api/notes_ai_summary.go:142-144`
   - `backend/internal/api/notes_ai_suggest.go`
   - `backend/internal/api/notes_ai_format.go`
   - `backend/internal/api/notes_ai_transform.go`
   - Test coverage: `backend/internal/service/summarize_test.go:62-85`, `backend/internal/api/notes_ai_privacy_test.go`

3. **Server-side export no longer silently emits empty encrypted-note files.**
   - `backend/internal/api/export.go:14-17` (explicit placeholder)
   - `backend/internal/api/export.go:80-87` (encrypted marker + placeholder content)
   - Test coverage: `backend/internal/api/export_test.go:78-135`

4. **`encrypted_title` wire format mismatch is resolved (`ciphertext` + `metadata`).**
   - Validator expects and validates `ciphertext` + `metadata`: `backend/internal/api/validation.go:139-195`
   - Create/update paths enforce validation: `backend/internal/api/notes_crud_create_helpers.go:89-92`, `backend/internal/api/notes_crud_update.go:72-75`
   - API tests for encrypted title create/update: `backend/internal/api/encryption_handlers_test.go:128-151`, `backend/internal/api/encryption_handlers_test.go:256-296`

5. **DEK re-wrap hardening is implemented (full validation + strict server checks).**
   - Full deterministic client-side validation (no sampling): `frontend/src/lib/crypto/e2e.ts:592-665`
   - Server rejects missing/extra IDs and malformed wrapped DEKs: `backend/internal/service/user_account.go:137-171`, `backend/internal/service/user_account.go:228-245`

6. **Recovery reset is blocked for encrypted users until secure DEK recovery re-wrap exists.**
   - `backend/internal/service/user_recovery.go:61-68`
   - `backend/internal/service/user_recovery.go:138-145`
   - `backend/internal/service/user_types.go:55`
   - Tests: `backend/internal/service/user_account_recovery_test.go:182-211`, `backend/internal/service/user_account_recovery_test.go:287-317`

7. **Encrypted-note attachments are uploaded as encrypted blobs (`.xenc`).**
   - Client encrypts attachment bytes before upload: `frontend/src/lib/components/Editor.svelte:906-918`
   - Dedicated encrypted upload endpoint: `backend/internal/api/uploads.go:191-222`
   - Endpoint behavior tests: `backend/internal/api/uploads_test.go:498-559`

8. **API-key crypto hardening is implemented (HKDF + key separation).**
   - HKDF key derivation: `backend/internal/crypto/apikey.go:68-75`
   - Dedicated secret required and separated from JWT secret: `backend/internal/crypto/apikey.go:51-66`
   - Startup enforcement for env policy: `backend/cmd/server/server_config.go:30-41`

9. **Client KEK setup now prefers worker-based KDF (non-blocking path) with tested fallback.**
   - `frontend/src/lib/crypto/e2e.ts:65-74`
   - `frontend/src/lib/crypto/e2e.ts:170-172`
   - Unit tests: `frontend/src/lib/crypto/e2e.test.ts:34-64`

10. **Plaintext keyword persistence for encrypted notes is disabled.**

- Encrypted create/update API paths drop `keywords` payloads: `backend/internal/api/notes_crud_create_helpers.go`, `backend/internal/api/notes_crud_update.go`
- Encrypted updates clear legacy keywords: `backend/internal/service/notes_encryption_update.go`
- Migration removes existing keyword rows for encrypted notes: `backend/internal/db/migrations/060_delete_keywords_for_encrypted_notes.sql`

11. **`keywords_enabled` preference is now fully deprecated/enforced-off.**

- Encryption settings UI no longer exposes keyword extraction toggle: `frontend/src/routes/settings/encryption/+page.svelte`
- Client no longer sets runtime keyword opt-in state from preferences: `frontend/src/lib/stores/encryption.svelte.ts`
- Server clamps `keywords_enabled` to `false` on updates: `backend/internal/service/user_preferences.go`, `backend/internal/db/preferences_encryption.go`
- Migration clears legacy enabled preference flags: `backend/internal/db/migrations/061_disable_keywords_encryption_preference.sql`

12. **Legacy link and due-date metadata for encrypted notes is purged and service-enforced.**

- Encrypted updates now clear links + unresolved links + due dates in service layer: `backend/internal/service/notes_encryption_update.go`
- API encrypted update path now only ignores client metadata (no duplicated clearing logic): `backend/internal/api/notes_crud_update.go`
- Migration removes existing encrypted-note metadata rows: `backend/internal/db/migrations/062_delete_links_due_dates_for_encrypted_notes.sql`
- Defense-in-depth guards reject persistence even if future callers send metadata for encrypted notes: `backend/internal/service/notes_links.go`

13. **Recovery-key setup is blocked for encrypted accounts to avoid false recovery guarantees.**

- Service rejects `SetRecoveryKey` when encrypted notes/versions exist: `backend/internal/service/user_recovery.go`
- API returns `409 Conflict` for blocked setup: `backend/internal/api/users_encryption.go`
- Coverage: `backend/internal/service/user_account_recovery_test.go`, `backend/internal/api/users_isolation_test.go`

14. **Recovery-key material is auto-invalidated when encrypted content exists.**

- Encrypted create/update flows invalidate stored recovery key state: `backend/internal/service/notes_encryption_create.go`, `backend/internal/service/notes_encryption_update.go`, `backend/internal/service/recipes_notes.go`, `backend/internal/service/canvas.go`
- Migration clears legacy recovery-key rows for users with encrypted notes/versions: `backend/internal/db/migrations/063_invalidate_recovery_keys_for_encrypted_users.sql`
- Coverage: `backend/internal/service/notes_encryption_test.go`

15. **Encrypted create paths no longer contain residual keyword-persistence hooks.**

- Encrypted note/journal/recipe/canvas create services now ignore keyword inputs entirely: `backend/internal/service/notes_encryption_create.go`, `backend/internal/service/recipes_notes.go`, `backend/internal/service/canvas.go`
- Canvas DB encrypted create path no longer accepts/stores keywords: `backend/internal/db/canvas.go`

16. **Recovery-key salt retrieval is blocked for encrypted accounts (including legacy seeded keys).**

- Service blocks direct and email-based salt lookup when encrypted notes/versions exist: `backend/internal/service/user_recovery.go`
- API maps the block to `404` on `GET /api/users/recovery-key/salt`: `backend/internal/api/users_encryption.go`
- Coverage includes service + API isolation tests for legacy-seeded recovery rows: `backend/internal/service/user_account_recovery_test.go`, `backend/internal/api/users_isolation_test.go`

17. **Server-side note tags are disabled for encrypted notes; legacy encrypted-tag rows are purged.**

- Service blocks `SetNoteTags` on encrypted notes with dedicated error guard: `backend/internal/service/notes_tags.go`
- API returns `409` on `PUT /api/notes/:id/tags` for encrypted notes and returns `[]` on `GET`, with best-effort legacy cleanup: `backend/internal/api/tags.go`
- Encrypted update flow now clears existing note-tag mappings: `backend/internal/service/notes_encryption_update.go`
- Migration removes legacy encrypted-note tag rows and orphaned tag entries: `backend/internal/db/migrations/064_delete_tags_for_encrypted_notes.sql`
- Coverage: `backend/internal/api/tags_versions_handlers_test.go`, `backend/internal/service/notes_encryption_test.go`

18. **Encrypted note folder paths are normalized to root (`/`) to reduce metadata leakage.**

- Encrypted create/update service flows no longer persist client folder paths: `backend/internal/service/notes_encryption_create.go`, `backend/internal/service/notes_encryption_update.go`, `backend/internal/service/recipes_notes.go`, `backend/internal/service/canvas.go`
- Shared helper enforces a single normalized encrypted folder path: `backend/internal/service/notes_encryption_paths.go`
- Migration rewrites existing encrypted-note folder paths to `/`: `backend/internal/db/migrations/065_set_encrypted_notes_folder_root.sql`
- Coverage: `backend/internal/api/encryption_handlers_test.go`, `backend/internal/service/notes_encryption_test.go`

19. **Encrypted write downgrade guard enforces modern metadata version (v3+).**

- Encryption request validation now rejects encrypted writes with `encryption_metadata.version < 3`: `backend/internal/api/validation.go`
- API create/update regression coverage includes explicit rejection for v2 metadata: `backend/internal/api/encryption_handlers_test.go`
- Existing legacy v2 data remains readable (no decrypt-path regression in this change).

## Remaining Limitations / Open Product Decisions

1. **Recovery still cannot decrypt existing encrypted notes after password loss** (intentional block, not an implementation bug in current state).
   - Doc claim aligned: `docs/e2e-encryption.md:66-77`, `docs/e2e-encryption.md:107-108`
   - README claim aligned: `README.md:95`

2. **E2EE does not cover all metadata** (explicitly documented scope limit).
   - `docs/e2e-encryption.md:51-58`

## Verification Run (2026-02-28)

Targeted verification command executed:

```bash
cd backend && CGO_ENABLED=1 go test -tags "fts5 sqlite_crypt" ./internal/api ./internal/service \
  -run 'TestCreateEncryptedNote_WithEncryptedTitle_Success|TestUpdateEncryptedNote_WithEncryptedTitle_Success|TestUserService_RecoverPasswordWithRecoveryKey|TestUserService_RecoverPasswordWithRecoveryKeyByEmail|TestExportMarkdown_EncryptedNotesAreMarkedInExport|TestSummarizeNote_EncryptedServerProcessingDisabled'
```

Result: `ok` for both `internal/api` and `internal/service`.
