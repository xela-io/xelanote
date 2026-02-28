# E2EE Remediation Plan (2026-02-28)

## Ziel

Strukturierte Abarbeitung der E2EE-Findings aus:
- `docs/security/E2EE-SECURITY-AUDIT-ADDENDUM-2026-02-28.md`

## PR-Plan

## PR1 (P0 Hotfixes)

Status: Abgeschlossen (implementiert am 2026-02-28, Tests grün)

Scope:
1. Recovery-Reset blockieren, wenn verschlüsselte Notizen existieren.
2. Server-seitige AI-Summarization für verschlüsselte Notizen deaktivieren.
3. Export von verschlüsselten Notizen serverseitig klar markieren (kein stilles Leeren).

Akzeptanzkriterien:
1. Recovery-Reset schlägt deterministisch fehl, wenn `content_encrypted=1` oder `title_encrypted=1` Notizen vorhanden sind.
2. AI-Summary-Endpoint verarbeitet keine plaintext payloads für verschlüsselte Notizen mehr.
3. Export-ZIP enthält keine stillschweigend leeren Dateien für verschlüsselte Notizen.
4. Relevante Unit/API-Tests grün.

Betroffene Bereiche:
- `backend/internal/service/user_recovery.go`
- `backend/internal/service/summarize_service.go`
- `backend/internal/api/notes_ai_summary.go`
- `backend/internal/api/export.go`
- zugehörige Tests in `backend/internal/service/*_test.go`, `backend/internal/api/*_test.go`

## PR2 (P0 Kryptoprotokoll: AAD Binding)

Status: Abgeschlossen (implementiert am 2026-02-28, Tests grün)

Scope:
1. AAD für Content + DEK-Wrapping einführen.
2. Ciphertext/Wrapped-DEK an Note-Kontext binden (z. B. `note_id`, `purpose`, `version`).
3. Migrationsstrategie v2 -> v3 definieren.

Akzeptanzkriterien:
1. Substitution zwischen zwei Notizen schlägt bei Decrypt fehl.
2. Alte v2-Daten bleiben lesbar oder werden migriert.
3. Negative Tests für tampering/substitution vorhanden.

## PR3 (P1 Schema-Konsistenz `encrypted_title`)

Status: Abgeschlossen (implementiert am 2026-02-28, Tests grün)

Scope:
1. Ein einheitliches Wire-Format für `encrypted_title` festlegen.
2. Backend-Validator und Frontend-Payload harmonisieren.
3. Integrationstest für create/update mit `title_encrypted=true`.

Akzeptanzkriterien:
1. Kein Format-Mismatch mehr zwischen FE und BE.
2. Titelverschlüsselung funktioniert in Create + Update.

## PR4 (P1 Rewrap-Härtung)

Status: Abgeschlossen (implementiert am 2026-02-28, Tests grün)

Scope:
1. Vollvalidierung aller rewrapped DEKs statt Stichprobe.
2. Striktere serverseitige Validierung von `wrapped_dek`-Inputs.

Akzeptanzkriterien:
1. Kein probabilistisches Validation-Sampling mehr.
2. Manipulierte/defekte Rewrap-Payloads werden abgelehnt.

## PR5 (P1 Vertraulichkeit Uploads/Metadaten)

Status: Abgeschlossen (implementiert am 2026-02-28, Tests grün)

Scope:
1. Client-seitige Attachment-Verschlüsselung konzipieren/implementieren.
2. Metadaten-Leakage reduzieren (links/due_dates/keywords) oder explizit hart dokumentieren + defaults anpassen.

Akzeptanzkriterien:
1. Upload-Dateien liegen serverseitig verschlüsselt vor.
2. Dokumentiertes, minimiertes Metadaten-Sichtbarkeitsmodell.

## PR6 (Dokumentation + UX Truthfulness)

Status: Abgeschlossen (implementiert am 2026-02-28)

Scope:
1. Recovery-Claims in README/UI/Docs auf Implementierungsrealität bringen.
2. E2EE-Grenzen (AI, Metadaten, Uploads) konsistent dokumentieren.

Akzeptanzkriterien:
1. Keine widersprüchlichen Security-Claims mehr.
2. Nutzerhinweise entsprechen dem tatsächlichen Sicherheitsmodell.

## PR7 (P2 Hardening: KDF + Key-Separation)

Status: Abgeschlossen (implementiert am 2026-02-28, Tests grün)

Scope:
1. API-Key-Verschlüsselungsschlüssel auf HKDF-SHA256 umstellen.
2. Harte Key-Separation erzwingen (`XELANOTE_API_KEY_SECRET` Pflicht, kein `JWT_SECRET`-Fallback).
3. Client-Login-KDF auf Worker-basiertes `deriveKeyAsync()` umstellen (Fallback nur bei Worker-Fehler).

Akzeptanzkriterien:
1. Backend startet nicht ohne dediziertes `XELANOTE_API_KEY_SECRET` (>=64 Zeichen, ungleich `JWT_SECRET`).
2. API-Key-Encryption nutzt keine direkte SHA-256-String-Ableitung mehr.
3. `setupKEK()` blockiert den Main Thread nicht im Standardpfad; Fallback ist getestet.
