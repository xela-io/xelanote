# E2EE Recovery DEK Rewrap - Implementation Plan (2026-02-28)

## Goal

Enable password recovery for accounts with encrypted notes **without exposing plaintext to the server** and without weakening the current E2EE model.

Current block is intentional and implemented in:

- `backend/internal/service/user_recovery.go` (`ErrRecoveryResetNeedsDEKRewrap`)
- `backend/internal/api/auth_user.go` (generic `401` for blocked encrypted recovery reset)

## Current Constraints (Code-Based)

1. Encrypted notes/versions store exactly one DEK wrapper (`wrapped_dek`):
   - `notes.wrapped_dek`
   - `note_versions.wrapped_dek`
2. Recovery key is currently only a verifier (`recovery_key_hash`) + KDF salt (`recovery_key_salt`) in `user_preferences`:
   - `backend/internal/db/preferences_encryption.go`
3. Recovery setup/salt for encrypted accounts is blocked:
   - `backend/internal/service/user_recovery.go`
   - `backend/internal/api/users_encryption.go`
4. Password-change rewrap path already exists and validates full note/version coverage:
   - `backend/internal/service/user_account.go`
   - `backend/internal/db/notes_rewrap.go`

## Security Requirements

1. No server-side plaintext access during recovery.
2. No recovery bypass that updates password without cryptographic control over encrypted DEKs.
3. Atomic reset: password update and DEK rewrap persist together or not at all.
4. Replay-safe recovery session (short-lived, single-use token).
5. Recovery flow must validate complete note/version coverage (same hardening level as password change).

## Proposed Design

## 1) Dual DEK Wrapping

Store two wrappers for each encrypted note/version:

1. `wrapped_dek` (password KEK wrapper, existing)
2. `wrapped_dek_recovery` (recovery KEK wrapper, new)

Proposed DB changes:

- `notes`: add nullable `wrapped_dek_recovery TEXT`
- `note_versions`: add nullable `wrapped_dek_recovery TEXT`

Rationale:

- Recovery flow can unwrap DEKs using recovery KEK (derived from recovery key + salt) even when password is lost.
- Server still only stores ciphertext and wrapped keys.

## 2) Recovery Session Handshake

Split current reset into verify + finalize:

1. `POST /api/auth/recovery/verify`
   - Input: `email`, `recovery_key`
   - Server verifies bcrypt hash (`recovery_key_hash`)
   - Returns short-lived one-time `recovery_reset_token`
2. `GET /api/auth/recovery/encrypted-deks`
   - Auth: `recovery_reset_token`
   - Returns `{notes[], versions[]}` with IDs + `wrapped_dek_recovery`
3. `POST /api/auth/recovery/reset-password`
   - Input: `recovery_reset_token`, `new_password`, `re_wrapped_note_deks`, `re_wrapped_version_deks`
   - Server validates full coverage, updates password hash, updates `wrapped_dek`, invalidates sessions, consumes token

Rationale:

- Keeps verifier logic server-side.
- Keeps cryptographic unwrap/rewrap client-side.

## 3) AAD Binding for Recovery Wrappers

Recovery wrappers must be AEAD-bound to note/version context similarly to existing v3 AAD strategy.

Frontend crypto (`frontend/src/lib/crypto/e2e.ts`):

- extend material purpose for recovery wrapper, e.g. `dek_recovery`
- use AAD format consistent with existing namespace, e.g.
  - `xelanote:e2ee:v3:note:dek_recovery:{noteID}`
  - equivalent for version IDs

Rationale:

- Prevents wrapper substitution across notes/versions.

## 4) Recovery Setup Flow (Encrypted Accounts)

Re-enable recovery setup for encrypted accounts, but only with recovery wrappers:

1. User is logged in and unlocked (password KEK available client-side).
2. Client derives recovery KEK from recovery key + recovery salt.
3. Client fetches encrypted notes/versions.
4. Client unwraps each DEK with password KEK, wraps with recovery KEK, uploads `wrapped_dek_recovery` maps.
5. Server stores:
   - `recovery_key_hash`
   - `recovery_key_salt`
   - all `wrapped_dek_recovery` values (atomic)

If coverage is incomplete, reject setup.

## 5) Recovery Key Rotation + Post-Recovery Policy

Recommended policy:

1. After successful recovery reset, invalidate recovery key material (`recovery_key_hash`, `recovery_key_salt`) and all `wrapped_dek_recovery`.
2. Require explicit re-setup after next login.

Rationale:

- Limits persistence window of compromised recovery keys.
- Mirrors current defensive invalidation posture already used in code.

## Implementation Steps

## Phase A: Schema + DB Layer

1. Migration `066_add_recovery_wrapped_dek_columns.sql`
   - add `wrapped_dek_recovery` to `notes`, `note_versions`
2. DB methods:
   - list encrypted notes/versions including recovery wrappers
   - bulk update recovery wrappers (note + version)
   - clear recovery wrappers

Files:

- `backend/internal/db/migrations/`
- `backend/internal/db/notes_encryption.go`
- `backend/internal/db/versions*.go`
- `backend/internal/db/notes_rewrap.go` (or dedicated recovery rewrap file)

## Phase B: Recovery Session Backend

1. Add recovery reset token storage (new table) or signed JWT with single-use server-side nonce.
2. Add endpoints:
   - verify
   - list encrypted recovery wrappers
   - finalize reset with rewrapped DEKs
3. Reuse existing completeness checks from `ChangePasswordWithDEKRewrap` logic.

Files:

- `backend/internal/api/auth_user.go`
- `backend/internal/service/user_recovery.go`
- `backend/internal/service/user_account.go` (extract reusable coverage validator)
- `backend/internal/db/*` for token persistence

## Phase C: Frontend Recovery Crypto Flow

1. Recovery UI flow:
   - verify recovery key
   - fetch recovery wrappers
   - derive recovery KEK + new password KEK
   - rewrap and submit
2. Add recovery wrapper methods in `E2EEncryption`.
3. Keep current unlock semantics and zeroization behavior.

Files:

- `frontend/src/lib/crypto/e2e.ts`
- `frontend/src/routes/...` recovery pages/forms
- `frontend/src/lib/api/...` recovery client calls

## Phase D: Re-enable Account Recovery Endpoints

1. Remove hard block for encrypted accounts only after phase B/C is complete.
2. Keep generic public errors (`invalid email or recovery key`) for enumeration resistance.

Files:

- `backend/internal/service/user_recovery.go`
- `backend/internal/api/users_encryption.go`
- `backend/internal/api/auth_user.go`

## Phase E: Rollout + Backfill

1. Existing encrypted users currently have recovery invalidated by migration `063`.
2. No risky automatic backfill is possible (missing recovery key material by design).
3. Rollout path:
   - ship new flow
   - prompt users to set recovery key while unlocked
   - create recovery wrappers at setup time

## Tests (Required)

## Backend

1. Recovery verify token issuance + expiry + one-time use.
2. Finalize reset rejects:
   - missing note/version IDs
   - extra IDs
   - malformed wrapped DEKs
3. Transactionality:
   - password hash and wrapped_dek update commit together
4. Cross-user isolation for recovery token + DEK lists.

## Frontend

1. Recovery rewrap roundtrip for note + version wrappers.
2. Wrong recovery key -> unwrap fails deterministically.
3. Tampered `wrapped_dek_recovery` -> decryption fails.
4. KEK/DEK zeroization after flow completion/failure.

## E2E

1. User with encrypted notes sets recovery key successfully.
2. Password loss simulation:
   - recovery verify
   - reset with new password
   - encrypted notes still decrypt with new password
3. Replay attempt with consumed recovery token fails.

## Non-Goals (for this phase)

1. Multi-device ratcheting / PCS.
2. Full metadata confidentiality.
3. Sharing encrypted notes with cross-user key exchange.

## Risks and Mitigations

1. Risk: Increased key-management complexity.
   - Mitigation: reuse existing rewrap validators and transaction boundaries.
2. Risk: Recovery token replay.
   - Mitigation: one-time token + TTL + server-side consume flag.
3. Risk: Partial wrapper state.
   - Mitigation: enforce complete coverage and atomic DB update.

## Acceptance Criteria

1. Encrypted account recovery reset succeeds without server plaintext.
2. Reset flow proves possession of valid recovery key and cryptographic DEK control.
3. Existing password-change rewrap guarantees remain intact.
4. Public endpoints do not increase account enumeration leakage.
