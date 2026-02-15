# Architecture & Conventions

Regeln fuer neuen Code. Bei Verstoessen Code anpassen, nicht die Regeln.

**Automatisierte Durchsetzung:** Viele dieser Regeln werden seit 2026-02-12 automatisch geprueft:
- **Layer Pattern**: `scripts/check-layer-violations.sh` (pre-commit + CI) prueft API->DB Violations gegen Baseline
- **Svelte 5 Only**: `scripts/check-svelte4-imports.sh` (pre-commit + CI) blockiert verbotene `svelte/store` Imports
- **Kein localStorage fuer Auth**: `scripts/check-security-patterns.sh` (pre-commit + CI) blockiert Auth-Token Persistenz in localStorage
- **Code Quality**: golangci-lint (CI) mit revive, misspell, bodyclose, gocritic, unused
- **Alle Checks**: `make quality` (lokal) oder `make check-policy` (nur Policy-Checks)

## Backend: 3-Layer Pattern

```
API Handler (internal/api/)  -->  Service (internal/service/)  -->  DB (internal/db/)
```

- **API Layer**: HTTP-Parsing, Validierung, Auth-Check (`getUserID(r)`), Response-Encoding, WebSocket-Events. Kein direkter DB-Zugriff.
- **Service Layer**: Business Logic, Caching, Orchestrierung. Keine HTTP-Typen (`http.Request`, `http.ResponseWriter`).
- **DB Layer**: SQL-Queries, Modelle, Migrationen. Nur `db.Err*`-Errors zurueckgeben (`ErrNotFound`, `ErrVersionMismatch`, `ErrDuplicate`).
- Neue Features: Handler in `api/`, Logik in `service/`, Queries in `db/`. Nie eine Schicht ueberspringen.

## Backend: Error Handling

- DB Layer: Custom Errors aus `db/errors.go` (`ErrNotFound`, `ErrVersionMismatch`, etc.)
- Service Layer: Faengt DB-Errors, fuegt Business-Errors hinzu, propagiert nach oben
- API Layer: Mapped Errors auf HTTP Status Codes via `respondError()` / `respondInternalErr()`
- Interne Details nie an Client leaken - `respondInternalErr()` loggt Details, Client sieht nur "internal server error"

## Backend: Route Registration

- Routes aufgeteilt in `routes_*.go` Dateien (z.B. `routes_notes.go`, `routes_folders.go`)
- Registry-Pattern: `registerProtectedResourceRoutes()` ruft Feature-spezifische Register-Funktionen auf
- Rate Limiting per Endpoint via `rateLimitMiddleware(s.xxxLimiter)`

## Backend: Tests

- Table-Driven Tests mit `t.Run()` Subtests
- In-Memory SQLite fuer DB/Service Tests (`db.Open(":memory:", "")`)
- Setup-Helper: `setupTestDB()`, `createTestUser()`
- DB Tests pruefen SQL + Constraints, Service Tests pruefen Business Logic, API Tests pruefen Validierung
- Keine HTTP-Server in Unit Tests - Service Layer direkt testen

## Backend: Migrationen

- Forward-only (kein Rollback), Dateiname: `NNN_beschreibung.sql`
- Immer `IF NOT EXISTS` / `IF EXISTS` verwenden
- Neue Migration: naechste freie Nummer, in `db.go` `runMigrations()` Array eintragen
- schema.sql = Basis fuer neue DBs, Migrationen laufen trotzdem alle (auch auf neuen DBs)

## Frontend: Svelte 5 Only

- **Nur Svelte 5 Runes** (`$state`, `$derived`, `$effect`, `$props`). Keine Svelte 4 Stores (`writable`, `readable`, `derived`).
- Props: `interface Props` + `const { x }: Props = $props()`
- Store-Dateien: `*.svelte.ts` (nicht `.ts`)

## Frontend: Store Pattern

- Module-Level State mit Getter/Setter Exports (kein Class-basiertes Pattern)
- Private `$state()` Variablen, oeffentliche `getX()` / `setX()` Funktionen
- Komplexe Stores modularisieren in Unterordner (siehe `stores/notes/` mit `saver.ts`, `loaders.ts`, etc.)
- Dependency Injection fuer testbare Helper: Store-Funktionen uebergeben Dependencies als Parameter-Objekt

## Frontend: API Layer

- Modular: Ein File pro Domain in `lib/api/` (notes.ts, auth.ts, folders.ts, ...)
- Barrel Export: `lib/api.ts` re-exportiert alle Module
- Zentraler Client: `lib/api/client.ts` handhabt Token-Refresh, Offline-Queue, Error-Handling
- `ApiError` Klasse mit `.status` fuer HTTP-Fehler - immer `instanceof ApiError` pruefen, nie String-Matching
- Offline: `_offlineAllowed: true` Flag an request() uebergeben wenn Operation offline queued werden darf

## Frontend: i18n

- Library: `svelte-i18n`, Locale-Dateien: `lib/locales/{de,en}.json`
- Template: `{$_('namespace.key')}`, TypeScript: `get(_)('namespace.key')`
- Namespaces: `page.*`, `component.*`, `dialog.*`, `nav.*`, `settings.*`
- Plurals: ICU MessageFormat `"{count, plural, one {# Notiz} other {# Notizen}}"`
- Neue UI-Strings: Immer in beide Locale-Dateien (de + en), nie hardcoded

## Frontend: Encryption

- E2E Encryption: XChaCha20-Poly1305, KEK via Argon2id, Per-Note DEKs
- Encryption-State lebt in `stores/encryption.svelte.ts` - immer `isEncryptionUnlocked()` pruefen vor Encrypt/Decrypt
- Verschluesselte Notizen: `encrypted_content` (BLOB) + `wrapped_dek`, nie Klartext in Logs/Errors
- Klartext-Titel nur wenn `encryptTitles: false` in User-Settings

## Telemetrie & Datenschutz

Verbindliche Regeln fuer alle Telemetrie-Features (Web Vitals, Analytics Events, etc.):

1. **URL-Sanitizing:** Nur Route-Pfade loggen (z.B. `/note/:id`), keine Query-Parameter. UUIDs und numerische IDs durch Platzhalter ersetzen (`:id`). Regex: `/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/g` → `:id`, `/\/\d+/g` → `/:id`.
2. **Kein PII:** Keine User-Agent-Strings, keine IP-Adressen in Telemetrie-Tabellen speichern. Fuer Session-Zuordnung nur anonymisierte Hashes verwenden.
3. **Sampling:** Max 10% der Seitenaufrufe senden Metriken (`Math.random() < 0.1` beim Init). Einmal pro Session entschieden, nicht pro Event.
4. **Retention:** Automatisches Loeschen nach 90 Tagen. Umsetzung via Backend-Cronjob oder bei Schreibzugriff (Cleanup-Query vor Insert).
5. **Payload-Limit:** Max 1 KB pro Event. Backend verwirft groessere Payloads mit 413 Status.
6. **DNT-Respektierung:** Do-Not-Track ist harte Untergrenze. Hierarchie:
   - `DNT=1` → Telemetrie immer deaktiviert, unabhaengig vom User-Setting
   - Kein DNT → User-Setting entscheidet (Default: an)
   - User-Setting kann Telemetrie nur zusaetzlich abschalten, nie gegen DNT einschalten

## Security (nicht verhandelbar)

- **Kein `localStorage` fuer Auth-Tokens** - nur HttpOnly Cookies
- **Constant-Time Auth**: Dummy-bcrypt fuer nicht-existente User (Timing-Attack-Prevention)
- **Generic Error Messages**: Login/Register-Fehler verraten nicht ob User existiert
- **PII Hashing**: User-IDs/IPs in Logs immer hashen (`hashIdentifier()`)
- **Optimistic Locking**: Notes haben `version` Feld, Updates pruefen Version, 409 bei Conflict
