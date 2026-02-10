# API Dokumentation

Xelanote bietet eine RESTful HTTP API für alle Note-Operationen. Die API folgt standard REST-Konventionen und gibt JSON zurück.

## Inhaltsverzeichnis

- [Basis-Informationen](#basis-informationen)
- [Configuration](#configuration)
  - [GET /api/config](#get-apiconfig)
- [Authentication](#authentication)
  - [POST /api/auth/register](#post-apiauthregister)
  - [POST /api/auth/login](#post-apiauthlogin)
  - [POST /api/auth/refresh](#post-apiauthrefresh)
  - [POST /api/auth/logout](#post-apiauthlogout)
  - [GET /api/auth/me](#get-apiauthme)
  - [POST /api/auth/recovery/salt](#post-apiauthrecoverysalt)
  - [POST /api/auth/recovery/reset-password](#post-apiauthrecoveryreset-password)
- [Two-Factor Authentication](#two-factor-authentication)
  - [POST /api/2fa/setup](#post-api2fasetup)
  - [POST /api/2fa/verify](#post-api2faverify)
  - [GET /api/2fa/status](#get-api2fastatus)
  - [DELETE /api/2fa](#delete-api2fa)
  - [POST /api/2fa/backup-codes/regenerate](#post-api2fabackup-codesregenerate)
- [Notes](#notes)
  - [GET /api/notes](#get-apinotes)
  - [POST /api/notes](#post-apinotes)
  - [GET /api/notes/:id](#get-apinotesid)
  - [PUT /api/notes/:id](#put-apinotesid)
  - [DELETE /api/notes/:id](#delete-apinotesid)
  - [POST /api/notes/:id/rename](#post-apinotesid-rename)
  - [POST /api/notes/:id/restore](#post-apinotesidrestore)
  - [DELETE /api/notes/:id/permanent](#delete-apinotesidpermanent)
  - [GET /api/notes/:id/backlinks](#get-apinotesidbacklinks)
  - [PUT /api/notes/:id/color](#put-apinotesidcolor)
  - [POST /api/notes/:id/decrypt](#post-apinotesiddecrypt)
  - [POST /api/notes/batch-reencrypt-deks](#post-apinotesbatch-reencrypt-deks)
  - [GET /api/notes/:id/versions](#get-apinotesidversions)
  - [GET /api/notes/:id/versions/:version](#get-apinotesidversionsversion)
  - [GET /api/notes/:id/versions/compare](#get-apinotesidversionscompare)
  - [POST /api/notes/:id/versions/:version/restore](#post-apinotesidversionsversionrestore)
- [Jobs](#jobs)
  - [GET /api/jobs/:id](#get-apijobsid)
- [Trash](#trash)
  - [GET /api/trash](#get-apitrash)
  - [GET /api/trash/count](#get-apitrashcount)
  - [DELETE /api/trash](#delete-apitrash)
- [Folders](#folders)
  - [GET /api/folders](#get-apifolders)
  - [GET /api/folders-legacy](#get-apifolders-legacy)
  - [POST /api/folders](#post-apifolders)
  - [POST /api/folders/reorder](#post-apifoldersreorder)
  - [PUT /api/folders/:id/move](#put-apifoldersidmove)
  - [PUT /api/folders/:id/rename](#put-apifoldersidrename)
  - [PUT /api/folders/:id/color](#put-apifoldersidcolor)
  - [DELETE /api/folders/:id](#delete-apifoldersid)
  - [GET /api/folders/:id/encryption-default](#get-apifoldersidencryption-default)
  - [PUT /api/folders/:id/encryption-default](#put-apifoldersidencryption-default)
- [Tags](#tags)
  - [GET /api/tags](#get-apitags)
  - [GET /api/notes/:id/tags](#get-apinotesidtags)
  - [PUT /api/notes/:id/tags](#put-apinotesidtags)
  - [DELETE /api/tags/:tagId](#delete-apitagstagid)
- [Templates](#templates)
  - [GET /api/templates](#get-apitemplates)
  - [GET /api/templates/:id](#get-apitemplatesid)
  - [POST /api/templates](#post-apitemplates)
  - [PUT /api/templates/:id](#put-apitemplatesid)
  - [DELETE /api/templates/:id](#delete-apitemplatesid)
- [Snippets](#snippets)
  - [GET /api/snippets](#get-apisnippets)
  - [GET /api/snippets/:id](#get-apisnippetsid)
  - [POST /api/snippets](#post-apisnippets)
  - [PUT /api/snippets/:id](#put-apisnippetsid)
  - [DELETE /api/snippets/:id](#delete-apisnippetsid)
- [Note Sharing](#note-sharing)
  - [POST /api/notes/:id/shares](#post-apinotesidshares)
  - [GET /api/notes/:id/shares](#get-apinotesidshares)
  - [PUT /api/notes/:id/shares/:userId](#put-apinotesidsharesuserId)
  - [DELETE /api/notes/:id/shares/:userId](#delete-apinotesidsharesuserId)
  - [GET /api/shared](#get-apishared)
  - [GET /api/shared/:id](#get-apisharedid)
  - [PUT /api/shared/:id](#put-apisharedid)
  - [GET /api/users/search](#get-apiuserssearch)
- [Folder Sharing](#folder-sharing)
  - [POST /api/folders/:id/shares](#post-apifoldersidshares)
  - [GET /api/folders/:id/shares](#get-apifoldersidshares)
  - [PUT /api/folders/:id/shares/:userId](#put-apifoldersidsharesuserId)
  - [DELETE /api/folders/:id/shares/:userId](#delete-apifoldersidsharesuserId)
  - [GET /api/shared/folders](#get-apisharedfolders)
  - [GET /api/shared/folders/:id/notes](#get-apisharedfoldersiднotes)
- [Shared Note Placements](#shared-note-placements)
  - [POST /api/shared/:id/placement](#post-apisharedidplacement)
  - [DELETE /api/shared/:id/placement](#delete-apisharedidplacement)
- [Search](#search)
  - [GET /api/search](#get-apisearch)
  - [GET /api/quick-search](#get-apiquick-search)
- [Uploads](#uploads)
  - [POST /api/uploads](#post-apiuploads)
  - [GET /api/uploads/:filename](#get-apiuploadsfilename)
- [Import](#import)
  - [POST /api/import/markdown](#post-apiimportmarkdown)
- [Export](#export)
  - [GET /api/export/markdown](#get-apiexportmarkdown)
- [Graph](#graph)
  - [GET /api/graph](#get-apigraph)
- [WebSocket](#websocket)
  - [GET /api/ws](#get-apiws)
- [Users](#users)
  - [GET /api/users/preferences](#get-apiuserspreferences)
  - [PUT /api/users/preferences](#put-apiuserspreferences)
  - [PUT /api/users/email](#put-apiusersemail)
  - [PUT /api/users/password](#put-apiuserspassword)
  - [PUT /api/users/preferences/encryption](#put-apiuserspreferencesencryption)
  - [PUT /api/users/preferences/security](#put-apiuserspreferencessecurity)
  - [POST /api/users/recovery-key](#post-apiusersrecovery-key)
  - [GET /api/users/recovery-key/salt](#get-apiusersrecovery-keysalt)
  - [POST /api/users/webauthn/credentials](#post-apiuserswebauthncredentials)
  - [DELETE /api/users/webauthn/credentials](#delete-apiuserswebauthncredentials)
  - [PATCH /api/users/webauthn/credentials/touch](#patch-apiuserswebauthncredentialstouch)
- [Admin](#admin)
  - [GET /api/admin/stats](#get-apiadminstats)
  - [GET /api/admin/stats/detailed](#get-apiadminstatsdetailed)
  - [GET /api/admin/users](#get-apiadminusers)
  - [GET /api/admin/users/:id](#get-apiadminusersid)
  - [PUT /api/admin/users/:id/admin](#put-apiadminusersidadmin)
  - [DELETE /api/admin/users/:id](#delete-apiadminusersid)
  - [GET /api/admin/activity](#get-apiadminactivity)
  - [GET /api/admin/settings](#get-apiadminsettings)
  - [PUT /api/admin/settings](#put-apiadminsettings)
- [Health](#health)
  - [GET /health](#get-health)
- [CSRF Protection](#csrf-protection)
  - [GET /api/csrf-token](#get-apicsrf-token)
- [Error Codes](#error-codes)

---

## Basis-Informationen

### Base URL

```
http://localhost:8080
```

Bei Docker Deployment ggf. anderen Port verwenden.

### Content-Type

Alle Requests und Responses verwenden JSON:

```http
Content-Type: application/json
```

### Authentication

Alle API-Requests (außer `/health` und `/api/auth/*`) erfordern JWT Authentication.

**Header (primär)**:

```
Authorization: Bearer <access_token>
```

Access Tokens sind kurzlebig (15 Minuten). Refresh Tokens werden zum Erneuern genutzt.

**Cookie (Fallback)**:

- `access_token` Cookie wird akzeptiert, wenn kein Authorization Header gesetzt ist.
- Auth-Endpunkte setzen Cookies automatisch (siehe `docs/authentication.md`).

### CORS

CORS ist standardmäßig aktiviert für Development:

```http
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, If-Match, Authorization, Cookie
Access-Control-Allow-Credentials: true
Access-Control-Expose-Headers: ETag
```

Für Production: Setze `CORS_ALLOWED_ORIGINS` (CSV), z.B.:

```
CORS_ALLOWED_ORIGINS="https://notes.example.com,https://admin.example.com"
```

### Versioning

API verwendet **Optimistic Locking** mit ETags:

- `GET` Requests geben `ETag` Header zurück
- `PUT` Requests erfordern `If-Match` Header mit aktuellem ETag

---

## Configuration

### GET /api/config

Öffentlicher Endpunkt für Client-Konfiguration (z.B. CAPTCHA-Einstellungen).

#### Request

```http
GET /api/config
```

#### Response

```http
200 OK
```

```json
{
  "captcha_enabled": true,
  "captcha_site_key": "0x4AAAAAACNggULpFEdcIsrE"
}
```

**Hinweis**: `captcha_site_key` ist nur vorhanden wenn `captcha_enabled: true`.

---

## Authentication

### POST /api/auth/register

Registriert einen neuen Benutzer und gibt Tokens zurück.

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "secret123"
}
```

### POST /api/auth/login

Login mit Username oder Email.

```json
{
  "username_or_email": "alice",
  "password": "secret123"
}
```

### POST /api/auth/refresh

Erneuert Access/Refresh Tokens.

```json
{
  "refresh_token": "<refresh_token>"
}
```

**Hinweis**: `refresh_token` kann alternativ aus dem HttpOnly Cookie gelesen werden (Body ist Fallback).

**SEC-006 Cookie-First Refresh:**
- Backend liest primär aus Cookie (`getRefreshTokenFromCookie`)
- Body-Parameter nur für Legacy-Kompatibilität
- Frontend nutzt `refreshTokenViaCookie()` für proaktiven Token-Refresh nach Page Reload
- Cookie wird automatisch via `credentials: 'include'` mitgesendet

### POST /api/auth/logout

Revoziert einen Refresh Token.

```json
{
  "refresh_token": "<refresh_token>"
}
```

**Hinweis**: `refresh_token` kann alternativ aus dem HttpOnly Cookie gelesen werden (Body ist Fallback).

### GET /api/auth/me

Gibt das aktuell authentifizierte User-Objekt zurück.

---

### POST /api/auth/recovery/salt

Öffentlicher Endpunkt zum Abrufen des Recovery Key Salt per E-Mail (für Passwort-Wiederherstellung).

#### Request

```http
POST /api/auth/recovery/salt
Content-Type: application/json
```

```json
{
  "email": "user@example.com"
}
```

#### Response

```http
200 OK
```

```json
{
  "salt": "base64-encoded-salt"
}
```

#### Errors

```http
400 Bad Request - "email is required"
404 Not Found - "no recovery key found for this email"
```

**Hinweis**: Gibt generische Fehlermeldung zurück um User-Enumeration zu verhindern.

---

### POST /api/auth/recovery/reset-password

Öffentlicher Endpunkt zum Zurücksetzen des Passworts mit Recovery Key.

#### Request

```http
POST /api/auth/recovery/reset-password
Content-Type: application/json
```

```json
{
  "email": "user@example.com",
  "recovery_key": "der-recovery-key",
  "new_password": "neues-sicheres-passwort"
}
```

#### Response

```http
200 OK
```

```json
{
  "message": "password reset successfully, please login with your new password"
}
```

#### Errors

```http
400 Bad Request - "email is required", "recovery_key is required", "new_password is required"
401 Unauthorized - "invalid email, recovery key, or password requirements not met"
```

**Hinweis**: Gibt generische Fehlermeldung zurück um Informationslecks zu verhindern.

---

## Two-Factor Authentication

Alle 2FA-Endpoints erfordern Authentication (Bearer Token).

### POST /api/2fa/setup

Startet 2FA-Setup und generiert TOTP-Secret + Backup-Codes.

**Request**:
```http
POST /api/2fa/setup
Authorization: Bearer <access_token>
```

**Response** (200 OK):
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code_url": "otpauth://totp/xelanote:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=xelanote",
  "backup_codes": [
    "ABCD-1234",
    "EFGH-5678",
    "IJKL-9012",
    "MNOP-3456",
    "QRST-7890",
    "UVWX-1234",
    "YZAB-5678",
    "CDEF-9012",
    "GHIJ-3456",
    "KLMN-7890"
  ]
}
```

**Hinweise**:
- Setup-Daten verfallen nach 15 Minuten ohne Verifizierung
- Wiederholtes Aufrufen löscht altes Setup und generiert neue Codes
- QR-Code sollte frontendseitig generiert werden (sicherer)

### POST /api/2fa/verify

Verifiziert TOTP-Code und aktiviert 2FA.

**Request**:
```http
POST /api/2fa/verify
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "code": "123456"
}
```

**Response** (200 OK):
```json
{
  "message": "2FA enabled successfully"
}
```

**Errors**:
- `400 Bad Request`: Code fehlt oder ungültig
- `400 Bad Request`: "setup expired, please restart" (> 15 Min)
- `429 Too Many Requests`: Rate-Limit überschritten (5 Versuche / 15 Min)

### GET /api/2fa/status

Gibt 2FA-Status für aktuellen User zurück.

**Request**:
```http
GET /api/2fa/status
Authorization: Bearer <access_token>
```

**Response** (200 OK):
```json
{
  "enabled": true,
  "verified_at": "2026-01-20T12:34:56Z",
  "unused_backup_codes": 8
}
```

### DELETE /api/2fa

Deaktiviert 2FA für aktuellen User.

**Request**:
```http
DELETE /api/2fa
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "password": "user_password",
  "totp_code": "123456",        // Option 1: TOTP-Code
  "backup_code": "ABCD-1234"    // Option 2: Backup-Code (Recovery)
}
```

**Response** (200 OK):
```json
{
  "message": "2FA disabled successfully"
}
```

**Hinweise**:
- `password` ist immer erforderlich
- Wenn 2FA aktiv: `totp_code` ODER `backup_code` erforderlich
- Backup-Code-Versuche: 3 / 15 Min (strenger als TOTP)
- Löscht alle 2FA-Daten (Secret + Backup-Codes)

### POST /api/2fa/backup-codes/regenerate

Generiert neue Backup-Codes (alte werden gelöscht).

**Request**:
```http
POST /api/2fa/backup-codes/regenerate
Authorization: Bearer <access_token>
```

**Response** (200 OK):
```json
{
  "backup_codes": [
    "WXYZ-9876",
    "QRST-5432",
    ...
  ]
}
```

**Errors**:
- `400 Bad Request`: "2FA is not enabled"

---

### Login mit 2FA

Der Login-Endpoint wurde erweitert:

**Request Step 1** (Credentials):
```http
POST /api/auth/login
Content-Type: application/json

{
  "username_or_email": "user@example.com",
  "password": "password",
  "captcha_token": "..."
}
```

**Response Step 1** (2FA Required):
```json
{
  "requires_two_factor": true
}
```

**Request Step 2** (TOTP/Backup):
```http
POST /api/auth/login
Content-Type: application/json

{
  "username_or_email": "user@example.com",
  "password": "password",
  "captcha_token": "...",
  "totp_code": "123456",        // Option 1: TOTP
  "backup_code": "ABCD-1234"    // Option 2: Backup (Recovery)
}
```

**Response Step 2** (Success):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "...",
  "user": {
    "id": 1,
    "username": "alice",
    "email": "alice@example.com",
    "is_admin": false
  }
}
```

**Hinweise**:
- Beide Schritte erfordern CAPTCHA
- CAPTCHA muss zwischen den Schritten neu gelöst werden
- Keine Tokens/Cookies vor erfolgreicher 2FA-Verifikation
- Rate-Limiting: 10 Versuche / 15 Min (beide Schritte zusammen)
- Backup-Code zusätzlich: 3 Versuche / 15 Min

---

## Notes

### GET /api/notes

Liste aller Notizen mit Pagination.

#### Query Parameters

| Parameter | Typ | Default | Beschreibung |
|-----------|-----|---------|--------------|
| `limit` | integer | 50 | Anzahl Notizen pro Page (max 100) |
| `cursor` | string | - | Pagination Cursor (aus `next_cursor` Response) |
| `folder` | string | - | Filter nach Ordner-Path (z.B. `/Projects`) |

#### Response

```http
GET /api/notes?limit=10
```

```json
{
  "notes": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Meeting Notes",
      "content": "Discussed [[Project A]] and [[Project B]]...",
      "folder_path": "/Work",
      "version": 3,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-16T14:22:00Z"
    },
    {
      "id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "title": "Project A",
      "content": "# Project A\n\nSee [[Meeting Notes]]",
      "folder_path": "/Projects",
      "version": 1,
      "created_at": "2024-01-10T09:00:00Z",
      "updated_at": "2024-01-10T09:00:00Z"
    }
  ],
  "next_cursor": "2024-01-10T09:00:00Z|6ba7b810-9dad-11d1-80b4-00c04fd430c8"
}
```

#### Cursor-based Pagination

```http
GET /api/notes?limit=10&cursor=2024-01-10T09:00:00Z|6ba7b810...
```

**Cursor Format**: `<timestamp>|<id>`

- Sortierung: `updated_at DESC, id DESC`
- Stabile Pagination auch bei Updates
- Letzter Cursor ist `""` (keine weitere Page)

#### Folder Filter

```http
GET /api/notes?folder=/Projects
```

Gibt nur Notizen in `/Projects` zurück (ohne Subfolders).

---

### POST /api/notes

Erstelle neue Notiz.

#### Request Body

```json
{
  "title": "New Note",
  "content": "Content with [[Wikilinks]]",
  "folder_path": "/Projects"  // optional, default: "/"
}
```

#### Validation

- `title`: Required, nicht-leer
- `content`: Optional, default ""
- `folder_path`: Optional, default "/"; Trailing Slash wird entfernt; Ordner werden nicht implizit erstellt

#### Response

```http
201 Created
ETag: "1"
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "New Note",
  "content": "Content with [[Wikilinks]]",
  "folder_path": "/Projects",
  "version": 1,
  "created_at": "2024-01-16T15:30:00Z",
  "updated_at": "2024-01-16T15:30:00Z"
}
```

#### Link Processing

Nach Creation werden automatisch Links verarbeitet:

**Standard-Notizen (unverschluesselt):**
1. Server-Parser extrahiert `[[Wikilinks]]` aus Content
2. Existierende Targets → `links` Table
3. Nicht-existierende Targets → `unresolved_links` Table
4. Falls neue Note unresolved Links aufloest → Backlinks werden aktualisiert

**E2E-verschluesselte Notizen:**
Bei verschluesselten Notizen kann der Server den Content nicht lesen. Daher:
1. Frontend extrahiert `[[Wikilinks]]` **vor** der Verschluesselung
2. Links werden im Request-Body als `links`-Array mitgesendet
3. Server validiert (max 500 Links, max 200 Zeichen pro Link) und speichert sie
4. Backlinks und Graph-Visualisierung funktionieren wie bei Standard-Notizen

**Request mit Client-seitigen Links:**
```json
{
  "title": "Encrypted Note",
  "content": "",
  "encrypted_content": "base64...",
  "links": ["Project A", "Meeting Notes"]
}
```

**Validierung:**
- Max 500 Links pro Notiz
- Max 200 Zeichen pro Link-Titel
- Links werden case-insensitive dedupliziert

#### Duplicate Title

Falls Titel bereits existiert:

```http
500 Internal Server Error
```

```json
{
  "error": "UNIQUE constraint failed: notes.title_norm"
}
```

**Note**: Case-Insensitive Check via `title_norm` (user-scoped Index).

---

### GET /api/notes/:id

Einzelne Notiz abrufen.

#### Request

```http
GET /api/notes/550e8400-e29b-41d4-a716-446655440000
```

#### Response

```http
200 OK
ETag: "3"
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Meeting Notes",
  "content": "Discussed [[Project A]]...",
  "folder_path": "/Work",
  "version": 3,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-16T14:22:00Z"
}
```

#### Not Found

```http
404 Not Found
```

```json
{
  "error": "note not found"
}
```

---

### PUT /api/notes/:id

Notiz aktualisieren (Titel, Content, oder Ordner-Pfad).

#### Headers

**Required**: `If-Match` mit aktuellem Version (ETag)

```http
If-Match: 3
```

#### Request Body

```json
{
  "title": "Updated Title",
  "content": "Updated content with [[New Link]]",
  "folder_path": "/Projekte"
}
```

**Felder:**

| Feld | Typ | Required | Beschreibung |
|------|-----|----------|--------------|
| `title` | string | Ja | Titel der Notiz |
| `content` | string | Ja | Markdown-Content |
| `folder_path` | string | Nein | Ordner-Pfad (Default: `/`) |

**`folder_path` Validierung:**

- Muss mit `/` beginnen (wenn gesetzt)
- Beispiel: `/Projekte/xelanote`
- Trailing Slash wird entfernt
- Ordner werden nicht implizit erstellt (siehe `/api/folders`)

#### Response

```http
200 OK
ETag: "4"
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Updated Title",
  "content": "Updated content with [[New Link]]",
  "folder_path": "/Work",
  "version": 4,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-16T15:45:00Z"
}
```

#### Optimistic Locking

**Version Mismatch**:

```http
409 Conflict
```

```json
{
  "error": "version mismatch - note was modified"
}
```

**Lösung**: Client muss Note neu laden (`GET /api/notes/:id`) und Update erneut versuchen.

#### Missing If-Match Header

```http
400 Bad Request
```

```json
{
  "error": "If-Match header required"
}
```

#### Link Reprocessing

Nach Update:

1. Alte Links werden gelöscht (`DELETE FROM links WHERE source_id = ?`)
2. Content wird neu geparsed
3. Neue Links werden eingefügt
4. Unresolved Links werden ggf. resolved

---

### DELETE /api/notes/:id

Notiz löschen (Soft Delete).

#### Request

```http
DELETE /api/notes/550e8400-e29b-41d4-a716-446655440000
```

#### Response

```http
204 No Content
```

#### Soft Delete

- Note wird nicht physisch gelöscht
- `is_deleted` Flag wird auf `1` gesetzt
- Note erscheint nicht mehr in Listen/Suchen
- Links bleiben erhalten (für Audit-Trail)

#### CASCADE Behavior

- `links` Table: Bleibt erhalten (Foreign Key ON DELETE CASCADE nicht aktiv bei Soft Delete)
- FTS5: Note wird aus Suchindex entfernt (via Trigger)

#### Not Found

```http
404 Not Found
```

```json
{
  "error": "note not found"
}
```

---

### POST /api/notes/:id/rename

Notiz umbenennen mit automatischer Refactoring-Unterstützung für alle Referenzen.

**Unterstützt Sync und Async Mode:**
- Sync Mode (default): Sofortige Ausführung, Response nach Completion
- Async Mode (`?async=true`): Background Job, sofortige Response mit Job ID

#### Query Parameters

| Parameter | Typ | Default | Beschreibung |
|-----------|-----|---------|--------------|
| `async` | boolean | false | Aktiviert Async Mode für lange Operationen |

#### Request Body

```json
{
  "newTitle": "Renamed Note"
}
```

#### Response (Sync Mode)

```http
200 OK
```

```json
{
  "note": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Renamed Note",
    "content": "...",
    "folder_path": "/Work",
    "version": 5,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-16T16:00:00Z"
  },
  "updated_note_count": 3
}
```

#### Response (Async Mode)

```http
202 Accepted
```

```json
{
  "job_id": "job_1_1705420800000000000",
  "status": "pending"
}
```

**Async Mode Workflow:**
1. Client sendet `POST /api/notes/:id/rename?async=true`
2. Server gibt sofort `202 Accepted` mit Job ID zurück
3. Client pollt `GET /api/jobs/{id}` bis Status `completed` oder `failed`
4. Bei `completed`: Result enthält Note und `updated_note_count`

**Empfehlung**: Async Mode nutzen bei >100 Backlinks (automatisch im Frontend)

#### Refactoring-Logik

**Beispiel**: Rename "Project A" → "Project Alpha"

1. **Update Target Note**:
   ```sql
   UPDATE notes SET title = 'Project Alpha', version = version + 1
   ```

2. **Finde alle referenzierenden Notizen**:
   ```sql
   SELECT source_id FROM links WHERE target_id = ?
   UNION
   SELECT source_id FROM unresolved_links WHERE target_ref_norm = 'project a'
   ```

3. **Für jede Referenz**:
   - Lade Content
   - Rewrite `[[Project A]]` → `[[Project Alpha]]`
   - Rewrite `[[Project A|Alias]]` → `[[Project Alpha|Alias]]` (Alias bleibt!)
   - Update Content + Version

4. **Commit Transaktion**

5. **Reprocess Links** (außerhalb Transaktion)

#### Case Change Only

Falls nur Case ändert (z.B. "project a" → "Project A"):

- Keine Referenz-Updates
- Nur Display-Title wird geändert
- `updated_note_count = 0`

#### Alias Preservation

**Input**:

```markdown
See [[Project A|my project]] and [[Project A]]
```

**After Rename to "Project Alpha"**:

```markdown
See [[Project Alpha|my project]] and [[Project Alpha]]
```

Aliase werden **nicht** geändert (User-Intent bleibt erhalten).

#### Performance

- Transaktional: Alle Updates in einer DB-Transaktion
- Bei 100 referenzierenden Notizen: ~500ms (abhängig von Content-Größe)
- Bei >1000 Referenzen: Erwäge Background Job (TODO)

---

### GET /api/notes/:id/backlinks

Liste aller Notizen, die auf diese Notiz verweisen.

#### Request

```http
GET /api/notes/550e8400-e29b-41d4-a716-446655440000/backlinks
```

#### Response

```http
200 OK
```

```json
{
  "backlinks": [
    {
      "id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "title": "Daily Notes"
    },
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "title": "Project Overview"
    }
  ]
}
```

#### Query

```sql
SELECT n.id, n.title
FROM notes n
JOIN links l ON l.source_id = n.id
WHERE l.target_id = ? AND n.is_deleted = 0
ORDER BY n.title ASC
```

#### Empty Result

```json
{
  "backlinks": []
}
```

---

### PUT /api/notes/:id/color

Setzt oder ändert die Farbmarkierung einer Notiz. Die Farbe wird als Hex-String (`#RRGGBB`) oder `null` (zum Entfernen) gespeichert und in der Sidebar als vertikaler Balken angezeigt (VS Code-Style).

#### Request

```http
PUT /api/notes/550e8400-e29b-41d4-a716-446655440000/color
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Body**:
```json
{
  "color": "#ff6b6b"
}
```

**Farbe entfernen**:
```json
{
  "color": null
}
```

#### Unterstützte Farbformate

1. **Hex-Farben**: `#RGB` oder `#RRGGBB` (case-insensitive)
   - Beispiele: `#f00`, `#ff0000`, `#3B82F6`
   - Werden normalisiert auf `#RRGGBB` Format

2. **Named Colors** (Design Tokens):
   - `primary`, `destructive`, `accent`, `muted`, `secondary`
   - Passen sich automatisch an Theme an (hell/dunkel)

3. **Null**: Entfernt die Farbmarkierung

#### Response

```http
200 OK
```

```json
{
  "status": "ok"
}
```

#### Fehler

**401 Unauthorized**: Kein gültiger JWT-Token
```json
{
  "error": "user not authenticated"
}
```

**404 Not Found**: Notiz existiert nicht oder gehört anderem User
```json
{
  "error": "note not found"
}
```

**400 Bad Request**: Ungültiges Farbformat
```json
{
  "error": "invalid color format"
}
```

#### Hinweise

- Farben sind **nicht verschlüsselt** (Metadaten, kein Content)
- Farbe wird in `notes.color` Spalte gespeichert (Migration 023)
- Frontend validiert mit `sanitizeColor()` Funktion
- Bei Named Colors wird der Name gespeichert, nicht der CSS-Wert

---

### POST /api/notes/:id/decrypt

Entschluesselt eine verschluesselte Notiz und speichert sie als Klartext. Erfordert den `If-Match` Header mit der aktuellen Version der Notiz (Optimistic Locking). Der entschluesselte Inhalt wird im Request-Body mitgesendet, da der Server keinen Zugriff auf den Verschluesselungsschluessel hat.

#### Request

```http
POST /api/notes/abc123/decrypt
Content-Type: application/json
If-Match: "v5"
```

```json
{
  "title": "Meine Notiz",
  "content": "Der entschluesselte Inhalt der Notiz..."
}
```

| Feld | Typ | Pflicht | Beschreibung |
|------|-----|---------|--------------|
| `title` | string | Ja | Der Klartext-Titel der Notiz |
| `content` | string | Ja | Der entschluesselte Inhalt der Notiz |

| Header | Pflicht | Beschreibung |
|--------|---------|--------------|
| `If-Match` | Ja | Aktuelle Version der Notiz (Optimistic Locking) |

#### Response

```http
200 OK
```

Die aktualisierte Notiz als JSON (mit `content` statt `encrypted_content`, ohne `wrapped_dek`).

#### Fehler

| Code | Bedeutung |
|------|-----------|
| 400 | Notiz ist bereits Klartext (nicht verschluesselt) |
| 404 | Notiz nicht gefunden |
| 409 | Version stimmt nicht ueberein (If-Match Header) |
| 412 | If-Match Header fehlt |

---

### POST /api/notes/batch-reencrypt-deks

Batch-Update für verschlüsselte DEKs (Data Encryption Keys). Wird nach Passwortänderungen verwendet, um alle DEKs mit dem neuen KEK (Key Encryption Key) neu zu verschlüsseln.

#### Request

```http
POST /api/notes/batch-reencrypt-deks
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "updates": [
    {
      "note_id": "550e8400-e29b-41d4-a716-446655440000",
      "wrapped_dek": "base64-encoded-new-wrapped-dek"
    },
    {
      "note_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "wrapped_dek": "base64-encoded-new-wrapped-dek"
    }
  ]
}
```

#### Response

```http
200 OK
```

```json
{
  "updated_count": 2,
  "message": "DEKs successfully re-encrypted"
}
```

#### Errors

```http
400 Bad Request - "no updates provided", "note_id required for update X", "invalid wrapped_dek for update X"
401 Unauthorized - "user not authenticated"
404 Not Found - "one or more notes not found"
500 Internal Server Error - "failed to update DEKs"
```

**Verwendung**: Dieser Endpunkt wird vom Client nach einer Passwortänderung aufgerufen. Der Client:
1. Entschlüsselt alle lokalen DEKs mit dem alten KEK
2. Verschlüsselt sie mit dem neuen KEK (abgeleitet vom neuen Passwort)
3. Sendet alle aktualisierten wrapped_deks in einem Batch-Request

---

### GET /api/notes/:id/versions

Liefert eine paginierte Liste aller gespeicherten Versionen einer Notiz.

**Version History Feature**: Das System erstellt automatisch Snapshots, wenn eine Notiz geändert wird UND mehr als 5 Minuten seit dem letzten Snapshot vergangen sind. Dies ermöglicht es, die Entwicklung einer Notiz über die Zeit nachzuvollziehen.

#### Request

```http
GET /api/notes/550e8400-e29b-41d4-a716-446655440000/versions?limit=50&cursor=...
Authorization: Bearer <token>
```

#### Query Parameters

| Parameter | Typ | Default | Beschreibung |
|-----------|-----|---------|--------------|
| `limit` | integer | 50 | Anzahl Versionen pro Page (max 100) |
| `cursor` | string | - | Pagination Cursor für weitere Pages |

#### Response

```http
200 OK
```

```json
{
  "versions": [
    {
      "id": 1,
      "note_id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": 1,
      "version": 5,
      "title": "Meeting Notes",
      "content": "Updated content from version 5...",
      "snapshot_at": "2026-01-17T14:30:00Z"
    },
    {
      "id": 2,
      "note_id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": 1,
      "version": 4,
      "title": "Meeting Notes",
      "content": "Content from version 4...",
      "snapshot_at": "2026-01-17T10:15:00Z"
    }
  ],
  "next_cursor": "2026-01-17T10:15:00Z|2",
  "total": 12
}
```

**Sortierung**: Neueste Version zuerst (`version DESC`)

**Hinweise**:
- Versionen werden automatisch erstellt, wenn Content/Title ändert UND >5 Minuten seit letztem Snapshot
- Reine Umbenennungen (via `/api/notes/:id/rename`) erzeugen KEINE Snapshots
- Maximale Retention: 30 Versionen pro Notiz (ältere werden täglich gelöscht)
- Cursor-basierte Pagination wie bei Notes

#### Error Responses

```http
404 Not Found
```

```json
{
  "error": "note not found"
}
```

---

### GET /api/notes/:id/versions/:version

Liefert eine spezifische Version einer Notiz zurück.

#### Request

```http
GET /api/notes/550e8400-e29b-41d4-a716-446655440000/versions/3
Authorization: Bearer <token>
```

#### Response

```http
200 OK
```

```json
{
  "id": 5,
  "note_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 1,
  "version": 3,
  "title": "Meeting Notes",
  "content": "Content from version 3...",
  "snapshot_at": "2026-01-16T15:00:00Z"
}
```

#### Error Responses

**Version nicht gefunden**:

```http
404 Not Found
```

```json
{
  "error": "version not found"
}
```

**Ungültige Versionsnummer**:

```http
400 Bad Request
```

```json
{
  "error": "invalid version number"
}
```

---

### GET /api/notes/:id/versions/compare

Liefert zwei Versionen einer Notiz zum Vergleichen. Der Diff wird client-seitig berechnet.

#### Request

```http
GET /api/notes/550e8400-e29b-41d4-a716-446655440000/versions/compare?v1=3&v2=5
Authorization: Bearer <token>
```

#### Query Parameters

| Parameter | Typ | Required | Beschreibung |
|-----------|-----|----------|--------------|
| `v1` | integer | Ja | Erste Versionsnummer |
| `v2` | integer | Ja | Zweite Versionsnummer |

**Hinweis**: Um die aktuelle Version mit einer älteren Version zu vergleichen, nutze die Versionsnummer der aktuellen Notiz für `v1` oder `v2`.

#### Response

```http
200 OK
```

```json
{
  "version1": {
    "id": 5,
    "note_id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": 1,
    "version": 3,
    "title": "Meeting Notes",
    "content": "Old content from version 3...",
    "snapshot_at": "2026-01-16T15:00:00Z"
  },
  "version2": {
    "id": 1,
    "note_id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": 1,
    "version": 5,
    "title": "Meeting Notes",
    "content": "New content from version 5...",
    "snapshot_at": "2026-01-17T14:30:00Z"
  }
}
```

**Client-seitige Diff-Berechnung**: Das Frontend nutzt die `diff` npm Library, um einen Line-Diff zwischen den beiden Versionen zu erzeugen.

#### Error Responses

**Fehlende Parameter**:

```http
400 Bad Request
```

```json
{
  "error": "v1 and v2 query parameters are required"
}
```

**Version nicht gefunden**:

```http
404 Not Found
```

```json
{
  "error": "version v1 not found"
}
```

---

### POST /api/notes/:id/versions/:version/restore

Stellt eine Notiz auf eine frühere Version zurück. Die aktuelle Version wird VOR dem Restore als Snapshot gespeichert (non-destructive).

#### Headers

**Required**: `If-Match` mit aktueller Version (Optimistic Locking)

```http
If-Match: 5
```

#### Request

```http
POST /api/notes/550e8400-e29b-41d4-a716-446655440000/versions/3/restore
Authorization: Bearer <token>
If-Match: 5
```

#### Response

```http
200 OK
ETag: "6"
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Meeting Notes",
  "content": "Restored content from version 3...",
  "folder_path": "/Work",
  "version": 6,
  "created_at": "2026-01-15T10:30:00Z",
  "updated_at": "2026-01-18T09:45:00Z"
}
```

**Ablauf**:
1. Aktuelle Version wird als Snapshot gespeichert
2. Note wird mit Content/Title der Target-Version aktualisiert
3. Version wird inkrementiert (neue Version 6)
4. Links werden neu verarbeitet

**Non-Destructive**: Durch das Speichern der aktuellen Version vor dem Restore können auch "versehentliche" Restores rückgängig gemacht werden.

#### Error Responses

**If-Match Header fehlt**:

```http
400 Bad Request
```

```json
{
  "error": "If-Match header required"
}
```

**Version Mismatch**:

```http
409 Conflict
```

```json
{
  "error": "version mismatch - note was modified"
}
```

**Version nicht gefunden**:

```http
404 Not Found
```

```json
{
  "error": "note or version not found"
}
```

---

## Jobs

Job Management für asynchrone Background-Operationen.

### GET /api/jobs/:id

Liefert Status und Ergebnis eines Background Jobs.

#### Request

```http
GET /api/jobs/job_1_1705420800000000000
Authorization: Bearer <token>
```

#### Response

**Job Running:**
```http
200 OK
```

```json
{
  "id": "job_1_1705420800000000000",
  "type": "rename_note",
  "user_id": 1,
  "status": "running",
  "progress": 0.5,
  "result": null,
  "error": "",
  "created_at": "2026-01-17T15:00:00Z",
  "updated_at": "2026-01-17T15:00:02Z",
  "metadata": {
    "noteID": "550e8400-e29b-41d4-a716-446655440000",
    "newTitle": "Renamed Note"
  }
}
```

**Job Completed:**
```http
200 OK
```

```json
{
  "id": "job_1_1705420800000000000",
  "type": "rename_note",
  "user_id": 1,
  "status": "completed",
  "progress": 1.0,
  "result": {
    "note": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Renamed Note",
      "content": "...",
      "folder_path": "/Work",
      "version": 5,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2026-01-17T15:00:05Z"
    },
    "updated_note_count": 15
  },
  "error": "",
  "created_at": "2026-01-17T15:00:00Z",
  "updated_at": "2026-01-17T15:00:05Z",
  "metadata": {
    "noteID": "550e8400-e29b-41d4-a716-446655440000",
    "newTitle": "Renamed Note"
  }
}
```

**Job Failed:**
```http
200 OK
```

```json
{
  "id": "job_1_1705420800000000000",
  "type": "rename_note",
  "user_id": 1,
  "status": "failed",
  "progress": 0.1,
  "result": null,
  "error": "note not found",
  "created_at": "2026-01-17T15:00:00Z",
  "updated_at": "2026-01-17T15:00:01Z",
  "metadata": {
    "noteID": "550e8400-e29b-41d4-a716-446655440000",
    "newTitle": "Renamed Note"
  }
}
```

#### Status Values

| Status | Beschreibung |
|--------|--------------|
| `pending` | Job wartet auf Ausführung in Queue |
| `running` | Job wird aktuell von Worker ausgeführt |
| `completed` | Job erfolgreich abgeschlossen |
| `failed` | Job fehlgeschlagen (siehe `error` Feld) |

#### Error Responses

**Job Not Found:**
```http
404 Not Found
```

```json
{
  "error": "job not found"
}
```

**Access Denied** (Job gehört anderem User):
```http
403 Forbidden
```

```json
{
  "error": "access denied"
}
```

#### Polling Strategy

**Empfohlenes Polling-Interval:**
- Initialer Poll: Sofort nach Job-Submission
- Folge-Polls: Alle 1 Sekunde
- Timeout: 60 Sekunden (60 Polls)

**Beispiel-Client:**

```typescript
async function pollJobCompletion(jobId: string): Promise<Job> {
  const maxAttempts = 60;

  for (let i = 0; i < maxAttempts; i++) {
    const job = await fetch(`/api/jobs/${jobId}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    }).then(r => r.json());

    if (job.status === 'completed') {
      return job;
    }

    if (job.status === 'failed') {
      throw new Error(job.error || 'Job failed');
    }

    // Wait 1 second before next poll
    await new Promise(resolve => setTimeout(resolve, 1000));
  }

  throw new Error('Job timeout');
}
```

#### Use Cases

**Rename mit vielen Backlinks:**
```typescript
// 1. Submit async rename
const { job_id } = await fetch('/api/notes/abc-123/rename?async=true', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify({ newTitle: 'New Title' })
}).then(r => r.json());

// 2. Show toast
toast.info('Renaming note in background...');

// 3. Poll für completion
const job = await pollJobCompletion(job_id);

// 4. Show result
if (job.result) {
  toast.success(`Renamed! Updated ${job.result.updated_note_count} references.`);
  currentNote = job.result.note;
}
```

#### Performance Considerations

- Jobs werden in Worker Pool ausgeführt (4 Workers parallel)
- Queue Capacity: 1000 Jobs
- Job-States werden in Memory gehalten (kein Persistence)
- Bei Server-Restart gehen pending/running Jobs verloren

---

## Trash

Trash-Management für gelöschte Notizen. Notizen werden soft-deleted (`is_deleted=1`) und können wiederhergestellt oder permanent gelöscht werden.

### GET /api/trash

Liefert paginierte Liste gelöschter Notizen.

#### Request

```http
GET /api/trash?limit=50&cursor=<timestamp>|<id>
Authorization: Bearer <token>
```

**Query Parameters**:
- `limit` (optional): Anzahl der Notizen (default: 50, max: 100)
- `cursor` (optional): Pagination Cursor im Format `timestamp|id`

#### Response

```json
{
  "notes": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Gelöschte Notiz",
      "content": "Inhalt der gelöschten Notiz",
      "folder_path": "/Archive",
      "version": 5,
      "created_at": "2026-01-15T10:00:00Z",
      "updated_at": "2026-01-16T14:30:00Z",
      "deleted_at": "2026-01-17T09:15:00Z"
    }
  ],
  "next_cursor": "2026-01-17T09:15:00Z|550e8400-e29b-41d4-a716-446655440000"
}
```

**Sortierung**: `deleted_at DESC` (neueste zuerst)

---

### GET /api/trash/count

Liefert Anzahl gelöschter Notizen (für Badge).

#### Request

```http
GET /api/trash/count
Authorization: Bearer <token>
```

#### Response

```json
{
  "count": 5
}
```

---

### POST /api/notes/:id/restore

Stellt eine gelöschte Notiz wieder her.

#### Request

```http
POST /api/notes/550e8400-e29b-41d4-a716-446655440000/restore
Authorization: Bearer <token>
```

#### Response

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Wiederhergestellte Notiz",
  "content": "...",
  "folder_path": "/Archive",
  "version": 6,
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-01-17T10:00:00Z",
  "deleted_at": null
}
```

**Hinweis**:
- Setzt `is_deleted=0`, `deleted_at=NULL`
- Verarbeitet Links neu (Wikilink Resolution)
- Version wird inkrementiert

**Error Responses**:
- `404 Not Found` - Notiz existiert nicht oder ist nicht gelöscht

---

### DELETE /api/notes/:id/permanent

Löscht eine Notiz permanent (Hard Delete).

#### Request

```http
DELETE /api/notes/550e8400-e29b-41d4-a716-446655440000/permanent
Authorization: Bearer <token>
```

#### Response

```http
HTTP/1.1 204 No Content
```

**Safety**: Nur Notizen mit `is_deleted=1` können permanent gelöscht werden.

**Error Responses**:
- `404 Not Found` - Notiz existiert nicht oder ist nicht soft-deleted

---

### DELETE /api/trash

Leert den Papierkorb (löscht alle soft-deleted Notizen permanent).

#### Request

```http
DELETE /api/trash
Authorization: Bearer <token>
```

#### Response

```json
{
  "deleted_count": 5
}
```

**Warnung**: Diese Operation kann nicht rückgängig gemacht werden!

---

## Folders

### GET /api/folders

Liefert alle Ordner mit Note-Counts.

```http
GET /api/folders
```

```json
{
  "folders": [
    {
      "id": 1,
      "path": "/",
      "parent_id": null,
      "name": "Root",
      "note_count": 12,
      "display_order": 0,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-16T14:22:00Z"
    }
  ]
}
```

### GET /api/folders-legacy

Legacy-Endpoint (kompatibel mit aelteren Clients). Liefert flache Ordnerpfade aus der Notes-Tabelle.

```http
GET /api/folders-legacy
```

```json
{
  "folders": [
    {
      "path": "/",
      "note_count": 5
    },
    {
      "path": "/Projects",
      "note_count": 2
    }
  ]
}
```

**Hinweis**: Wenn keine Ordner existieren, wird `"/"` mit `note_count: 0` geliefert.

### POST /api/folders

Erstellt einen Ordner.

```json
{
  "path": "/Projects"
}
```

**Validierung**:
- Muss mit `/` beginnen
- Kein `..` und keine doppelten Slashes `//`
- Kein Trailing Slash (ausser Root `/`)

### POST /api/folders/reorder

Sortierung innerhalb eines Parents aktualisieren.

```json
{
  "parent_id": 1,
  "items": [3, 5, 2]
}
```

### PUT /api/folders/:id/move

```json
{
  "new_parent_path": "/Archive"
}
```

### PUT /api/folders/:id/rename

```json
{
  "new_name": "Work"
}
```

### PUT /api/folders/:id/color

Setzt oder ändert die Farbmarkierung eines Ordners. Die Farbe wird als Hex-String (`#RRGGBB`) oder `null` (zum Entfernen) gespeichert und in der Sidebar als vertikaler Balken angezeigt (VS Code-Style).

#### Request

```http
PUT /api/folders/42/color
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Body**:
```json
{
  "color": "#10b981"
}
```

**Farbe entfernen**:
```json
{
  "color": null
}
```

#### Unterstützte Farbformate

1. **Hex-Farben**: `#RGB` oder `#RRGGBB` (case-insensitive)
   - Beispiele: `#0f0`, `#00ff00`, `#10B981`
   - Werden normalisiert auf `#RRGGBB` Format

2. **Named Colors** (Design Tokens):
   - `primary`, `destructive`, `accent`, `muted`, `secondary`
   - Passen sich automatisch an Theme an (hell/dunkel)

3. **Null**: Entfernt die Farbmarkierung

#### Response

```http
200 OK
```

```json
{
  "status": "ok"
}
```

#### Fehler

**401 Unauthorized**: Kein gültiger JWT-Token
```json
{
  "error": "user not authenticated"
}
```

**400 Bad Request**: Ungültige Folder-ID oder Farbformat
```json
{
  "error": "invalid folder id"
}
```

```json
{
  "error": "invalid color format"
}
```

#### Hinweise

- Farben sind **nicht verschlüsselt** (Metadaten)
- Farbe wird in `folders.color` Spalte gespeichert (Migration 023)
- Frontend validiert mit `sanitizeColor()` Funktion
- Ordner-Farben werden oft für Kategorisierung genutzt (z.B. Rot = Urgent, Grün = Done)

### DELETE /api/folders/:id

Löscht Ordner (fehlschlägt, wenn Subfolders existieren).

---

### GET /api/folders/:id/encryption-default

Gibt den Encryption-Default-Wert eines Ordners zurueck. Bestimmt, ob neue Notizen in diesem Ordner verschluesselt oder unverschluesselt erstellt werden.

#### Request

```http
GET /api/folders/5/encryption-default
```

#### Response

```http
200 OK
```

```json
{
  "encryption_default": true
}
```

| Feld | Typ | Beschreibung |
|------|-----|--------------|
| `encryption_default` | boolean | `true` = neue Notizen werden verschluesselt (Standard), `false` = neue Notizen werden unverschluesselt erstellt |

---

### PUT /api/folders/:id/encryption-default

Setzt den Encryption-Default-Wert eines Ordners. Beeinflusst nur neue Notizen -- bestehende Notizen bleiben unveraendert.

#### Request

```http
PUT /api/folders/5/encryption-default
Content-Type: application/json
```

```json
{
  "encryption_default": false
}
```

| Feld | Typ | Pflicht | Beschreibung |
|------|-----|---------|--------------|
| `encryption_default` | boolean | Ja | `true` fuer verschluesselt, `false` fuer unverschluesselt |

#### Response

```http
200 OK
```

```json
{
  "message": "encryption default updated"
}
```

#### Fehler

| Code | Bedeutung |
|------|-----------|
| 400 | Ungueltiger Request Body |
| 403 | Benutzer ist nicht Besitzer des Ordners |
| 404 | Ordner nicht gefunden |

---

## Tags

Tags ermöglichen die Kategorisierung von Notizen mit Schlagwörtern. Jeder User hat eigene Tags (user-scoped).

### GET /api/tags

Gibt alle Tags des eingeloggten Users zurück.

#### Request

```http
GET /api/tags
Authorization: Bearer <access_token>
```

#### Response

```http
200 OK
```

```json
[
  {
    "id": 1,
    "name": "work",
    "user_id": 1
  },
  {
    "id": 2,
    "name": "important",
    "user_id": 1
  }
]
```

---

### GET /api/notes/:id/tags

Gibt alle Tags einer Notiz zurück.

#### Request

```http
GET /api/notes/550e8400-e29b-41d4-a716-446655440000/tags
Authorization: Bearer <access_token>
```

#### Response

```http
200 OK
```

```json
[
  {
    "id": 1,
    "name": "work",
    "user_id": 1
  }
]
```

---

### PUT /api/notes/:id/tags

Setzt die Tags einer Notiz (überschreibt bestehende Tags).

#### Request

```http
PUT /api/notes/550e8400-e29b-41d4-a716-446655440000/tags
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "tags": ["work", "important", "project"]
}
```

#### Response

```http
200 OK
```

```json
[
  {
    "id": 1,
    "name": "work",
    "user_id": 1
  },
  {
    "id": 2,
    "name": "important",
    "user_id": 1
  },
  {
    "id": 3,
    "name": "project",
    "user_id": 1
  }
]
```

**Hinweise**:
- Tags werden automatisch erstellt, wenn sie nicht existieren
- Leere Tags werden ignoriert
- Case-insensitive: "Work" und "work" sind dasselbe Tag

---

### DELETE /api/tags/:tagId

Löscht ein Tag und entfernt es von allen Notizen.

#### Request

```http
DELETE /api/tags/1
Authorization: Bearer <access_token>
```

#### Response

```http
204 No Content
```

**Fehler**:
- `404 Not Found` - Tag existiert nicht oder gehört nicht dem User

---

## Search

### GET /api/search

Volltextsuche mit SQLite FTS5.

#### Query Parameters

| Parameter | Typ | Default | Beschreibung |
|-----------|-----|---------|--------------|
| `q` | string | - | **Required** - Suchquery |
| `limit` | integer | 20 | Max Anzahl Ergebnisse (max 100) |

#### Request

```http
GET /api/search?q=project&limit=10
```

#### Response

```http
200 OK
```

```json
{
  "query": "project",
  "results": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Project A",
      "snippet": "...discussing the <mark>project</mark> timeline...",
      "rank": 0.85
    },
    {
      "id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "title": "Project Updates",
      "snippet": "New <mark>project</mark> requirements...",
      "rank": 0.72
    }
  ]
}
```

#### Query Semantik (aktuell)

Der Server zerlegt `q` in Whitespace-Tokens und baut eine FTS5 Query der Form:

```
"term1"* "term2"* ...
```

- Prefix-Match ist immer aktiv.
- Keine Phrase-/Boolean-/Column-Queries in der API (werden serverseitig nicht durchgereicht).

Beispiel:

```http
GET /api/search?q=project timeline
```

Sucht nach Notes, die beide Terme enthalten (AND-Logik).

#### Ranking

FTS5 BM25 Ranking-Algorithmus (Standard, keine expliziten Column-Weights im Code):

- Implementiert als `bm25(notes_fts)` in `backend/internal/db/search.go`

#### Snippet Highlighting

Snippets enthalten `<mark>...</mark>` Tags um Matches:

```json
"snippet": "...the <mark>project</mark> was completed..."
```

**Snippet-Länge**: FTS5 `snippet(..., 32)` (token-basiert) in `backend/internal/db/search.go`.

#### Sicherheit (XSS-Schutz)

Snippets werden server-seitig mit `html.EscapeString()` sanitized, wobei nur `<mark>` Tags
für das Highlighting erhalten bleiben. Das Frontend verwendet kein `@html` Directive,
sondern parst die `<mark>` Tags explizit und rendert sie sicher.

- Backend: `backend/internal/db/search.go` → `escapeSnippetHTML()`
- Frontend: `frontend/src/routes/search/+page.svelte` → `parseSnippet()`
- Tests: `backend/internal/db/search_test.go`

#### Empty Query

```http
400 Bad Request
```

```json
{
  "error": "query parameter 'q' is required"
}
```

---

### GET /api/quick-search

Schnelle Titel-basierte Suche für Quick Switcher (Ctrl+P) mit optionalen Filtern.

#### Query Parameters

| Parameter | Typ | Default | Beschreibung |
|-----------|-----|---------|--------------|
| `q` | string | - | Optional - Suchquery (Titel-Suche) |
| `limit` | integer | 10 | Max Anzahl Ergebnisse (max 50) |
| `folders` | string | - | Optional - Komma-separierte Ordner-Pfade (OR-Logik) |
| `tags` | string | - | Optional - Komma-separierte Tag-Namen (AND-Logik) |
| `created_after` | string | - | Optional - ISO8601 Timestamp |
| `created_before` | string | - | Optional - ISO8601 Timestamp |
| `updated_after` | string | - | Optional - ISO8601 Timestamp |
| `updated_before` | string | - | Optional - ISO8601 Timestamp |

#### Request (Basic)

```http
GET /api/quick-search?q=proj
```

#### Request (Mit Filtern)

```http
GET /api/quick-search?q=meeting&tags=work,important&folders=/Projects&created_after=2026-01-01T00:00:00Z
```

#### Response

```http
200 OK
```

```json
{
  "notes": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Project A",
      "content": "...",
      "folder_path": "/Projects",
      "version": 2,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

#### Query-Methode (ohne Filter)

```sql
SELECT id, title, content, folder_path, version, created_at, updated_at
FROM notes
WHERE title_norm LIKE ? AND user_id = ? AND is_deleted = 0
ORDER BY
  CASE WHEN title_norm LIKE ? THEN 0 ELSE 1 END,
  updated_at DESC
LIMIT ?
```

- Case-Insensitive via `title_norm`
- Substring-Match

**Mit Filtern**: `backend/internal/db/search.go` baut eine dynamische Query (Tags/Folders/Datum).
- Keine FTS5 (schneller für Titel-Only)

#### Filter-Logik

**Tags (AND-Logik)**: Notiz muss ALLE angegebenen Tags haben

**Ordner (OR-Logik)**: Notiz kann in EINEM der angegebenen Ordner sein

**Datum (Bereich)**: ISO8601 Format (z.B. `2026-01-18T10:30:00Z`)

#### Backward-Kompatibilität

Bestehende Aufrufe ohne Filter funktionieren weiter.

#### Empty Query

```http
GET /api/quick-search
```

Gibt leere Liste zurück:

```json
{
  "notes": []
}
```

---

## Uploads

### POST /api/uploads

Image Upload (multipart/form-data, Feldname `file`).

- Max 10MB per file
- Allowed Types: PNG, JPEG, GIF, WebP
- **Storage Quota Enforcement**: Pre-write validation blocks uploads that would exceed user's storage limit
  - Returns `403 Forbidden` if upload would exceed `max_storage_mb_per_user` setting
  - Quota check performed BEFORE writing to disk to prevent race conditions

**Security Note (SEC-L04):**
- Upload responses include cryptographically signed URLs valid for 7 days
- Signed URLs allow image rendering without cookies (prevents SameSite=Strict blocking)
- Signature format: HMAC-SHA256 of `userID|filename|expires` with JWT_SECRET
- Fallback to cookie-based authentication for backward compatibility

Response:

```json
{
  "url": "/api/uploads/1/abc123.png?signature=XYZ&expires=1234567890",
  "filename": "screenshot.png"
}
```

Error Responses:

- `400 Bad Request`: File too large (>10MB), invalid file type, or no file provided
- `403 Forbidden`: Storage limit exceeded or would be exceeded by this upload
- `500 Internal Server Error`: Server-side error during upload

### GET /api/uploads/{user_id}/{filename}

Serves uploaded images with two authentication methods:

1. **Signed URLs** (primary): Query parameters `?signature=...&expires=...`
   - No cookies required
   - Valid for 7 days from upload
   - Signature validated via HMAC-SHA256
   - Example: `/api/uploads/1/abc123.png?signature=XYZ&expires=1234567890`

2. **Cookie Authentication** (fallback): JWT access token cookie
   - Used when signature is missing, invalid, or expired
   - Requires user to be authenticated
   - Ownership verification: user can only access their own uploads

**Security:**
- Path traversal prevention via `filepath.Base()` sanitization
- User isolation: uploads stored in `/uploads/{user_id}/` directories
- Ownership verification on cookie fallback

---

## Import

### POST /api/import/markdown

Bulk-Import von Markdown-Dateien.

```json
{
  "preserve_structure": true,
  "files": [
    {
      "path": "Projects/alpha.md",
      "filename": "alpha.md",
      "content": "# Alpha\\n\\n[[Notes]]"
    }
  ]
}
```

Response:

```json
{
  "imported": 1,
  "skipped": 0,
  "failed": 0,
  "folders_created": 1,
  "errors": []
}
```

---

## Export

### GET /api/export/markdown

Exportiere alle Notizen als ZIP-Archiv mit Markdown-Dateien.

#### Request

```http
GET /api/export/markdown
```

#### Response

```http
200 OK
Content-Type: application/zip
Content-Disposition: attachment; filename="xelanote-export-2024-01-16_153022.zip"
```

ZIP-File enthält:

```
xelanote-export-2024-01-16_153022.zip
├── Meeting Notes.md
├── Projects/
│   ├── Project A.md
│   └── Project B.md
└── Work/
    └── Daily Notes.md
```

#### File Format

Jede Markdown-Datei enthält YAML Frontmatter (Obsidian-kompatibel):

```markdown
---
title: "Meeting Notes"
---

# Meeting Notes

Content with [[Wikilinks]]...
```

#### Filename Sanitization

Invalide Zeichen werden ersetzt:

- `/`, `\`, `:`, `*`, `?`, `"`, `<`, `>`, `|` → `_`
- Max Länge: 200 Zeichen
- Fallback für leere Namen: `untitled.md`

#### Duplicate Names

Bei Duplikaten wird Counter angehängt:

```
Project.md
Project_2.md
Project_3.md
```

(Case-Insensitive Check)

#### Verwendung

**Obsidian Import**:

1. Download ZIP
2. Entpacke in Obsidian Vault
3. Wikilinks funktionieren direkt

**Backup**:

```bash
curl -o backup.zip http://localhost:8080/api/export/markdown
```

---

## Graph

Graph Visualization API für die Visualisierung von Notizen-Netzwerken.

### GET /api/graph

Gibt den globalen Notizen-Graph zurück mit allen Notizen und ihren Verbindungen.

#### Request

```http
GET /api/graph
GET /api/graph?folder=/Work
Authorization: Bearer <access_token>
```

#### Query Parameter

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `folder` | string | (Optional) Filtert Graph nach Ordner-Pfad. Verwendet Prefix-Match für Subordner. |

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "nodes": [
    {
      "id": "abc123",
      "title": "Projekt A",
      "folder_path": "/Work",
      "is_resolved": true
    },
    {
      "id": "def456",
      "title": "Meeting Notes",
      "folder_path": "/Work",
      "is_resolved": true
    },
    {
      "id": "unresolved:todo-liste",
      "title": "Todo Liste",
      "folder_path": "",
      "is_resolved": false
    }
  ],
  "edges": [
    {
      "source_id": "abc123",
      "target_id": "def456",
      "type": "resolved"
    },
    {
      "source_id": "abc123",
      "target_id": "unresolved:todo-liste",
      "type": "unresolved"
    }
  ],
  "metadata": {
    "node_count": 3,
    "edge_count": 2,
    "truncated": false
  }
}
```

#### Response Fields

**GraphNode**:
- `id`: Note ID (oder `unresolved:<title>` für ungelöste Links)
- `title`: Notiz-Titel
- `folder_path`: Ordner-Pfad (leer für ungelöste Knoten)
- `is_resolved`: `true` = existierende Notiz, `false` = ungelöster Link

**GraphEdge**:
- `source_id`: Quell-Notiz ID
- `target_id`: Ziel-Notiz ID (oder unresolved ID)
- `type`: `"resolved"` oder `"unresolved"`

**GraphMetadata**:
- `node_count`: Anzahl Knoten
- `edge_count`: Anzahl Verbindungen
- `truncated`: `true` wenn Graph gekürzt wurde (Max: 1000 Nodes, 5000 Edges)

#### Wie entstehen Verbindungen?

Verbindungen entstehen durch **Wikilinks** in Notizen:

```markdown
# Meine Notiz
Siehe [[Andere Notiz]] und [[Noch nicht erstellt]].
```

**Verarbeitung**:
1. Parser extrahiert alle `[[Links]]` aus dem Content
2. System prüft, ob Ziel-Notizen existieren:
   - **Existiert** → Resolved Link (blauer Pfeil im Graph)
   - **Existiert nicht** → Unresolved Link (roter Pfeil + roter Knoten)
3. Bei Note-Erstellung werden ungelöste Links automatisch aufgelöst

**Tabellen**:
- `links`: Resolved links (source_id → target_id)
- `unresolved_links`: Unresolved links (source_id → target_title)

#### Performance & Limits

**Capping**:
- Max Nodes: 1000
- Max Edges: 5000
- Bei Überschreitung: `metadata.truncated = true`

**Caching**:
- Global Graph: 5 Minuten TTL
- Invalidation bei Note Create/Update/Delete/Rename

---

## WebSocket

Realtime Updates fuer Notes (Create/Update/Delete).

### GET /api/ws

Upgradet die Verbindung zu WebSocket. Der Access Token wird als Query-Param uebergeben.

#### Request

```http
GET /api/ws?token=<access_token>
```

#### Message Format

```json
{
  "type": "note.updated",
  "payload": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Updated Title",
    "content": "...",
    "folder_path": "/Work",
    "version": 4,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-16T15:45:00Z"
  }
}
```

#### Events

- `note.created` (Payload: komplette Note)
- `note.updated` (Payload: komplette Note)
- `note.deleted` (Payload: `{ "id": "<note-id>" }`)

#### Hinweise

- Der Server erwartet aktuell keine Client-Nachrichten.
- Folder Filter: Kein Cache (dynamisch)
- Cache wird invalidiert bei: Note create/update/delete/rename

**User Isolation**:
- Alle Queries sind user-scoped
- Keine Cross-User Links möglich

#### Beispiele

**Global Graph abrufen**:

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/graph
```

**Nach Ordner filtern**:

```bash
# Nur Notizen im /Work Ordner (inkl. Subordner)
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/graph?folder=/Work"
```

**TypeScript/JavaScript**:

```typescript
interface GraphNode {
  id: string;
  title: string;
  folder_path: string;
  is_resolved: boolean;
}

interface GraphEdge {
  source_id: string;
  target_id: string;
  type: 'resolved' | 'unresolved';
}

interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
  metadata: {
    node_count: number;
    edge_count: number;
    truncated: boolean;
  };
}

async function getGraph(folder?: string): Promise<GraphData> {
  const params = new URLSearchParams();
  if (folder) params.set('folder', folder);

  const response = await fetch(`/api/graph?${params}`, {
    headers: { 'Authorization': `Bearer ${accessToken}` }
  });

  return response.json();
}
```

**Python**:

```python
def get_graph(self, folder: str = None) -> Dict:
    params = {}
    if folder:
        params['folder'] = folder

    response = requests.get(
        f"{self.base_url}/api/graph",
        params=params,
        headers={"Authorization": f"Bearer {self.token}"}
    )
    response.raise_for_status()
    return response.json()
```

#### Frontend Integration

**force-graph Library** (Canvas-basiert):

```typescript
import ForceGraph from 'force-graph';

const graphData = await getGraph();

const graph = ForceGraph()(container)
  .graphData({
    nodes: graphData.nodes,
    links: graphData.edges.map(e => ({
      source: e.source_id,
      target: e.target_id,
      type: e.type
    }))
  })
  .nodeId('id')
  .nodeLabel('title')
  .nodeColor(node => node.is_resolved ? '#3b82f6' : '#ef4444')
  .linkColor(link => link.type === 'resolved' ? '#6366f1' : '#f87171')
  .onNodeClick(node => {
    if (node.is_resolved) {
      window.location.href = `/note/${node.id}`;
    }
  });
```

#### Error Responses

**404 Not Found** (Feature Flag deaktiviert):
```json
{
  "error": "Not Found"
}
```

**401 Unauthorized**:
```json
{
  "error": "user not authenticated"
}
```

**500 Internal Server Error**:
```json
{
  "error": "failed to get global graph: <details>"
}
```

---

## Templates

Templates sind wiederverwendbare Notizvorlagen mit Platzhaltern für dynamische Inhalte.

**Platzhalter**:
- `{{date}}` - Wird zu YYYY-MM-DD (z.B. 2026-01-18)
- `{{time}}` - Wird zu HH:MM (z.B. 14:30)
- `{{cursor}}` - Markiert Cursor-Position nach dem Einfügen

**Limits**:
- Name: max. 100 Zeichen
- Title: max. 200 Zeichen
- Content: max. 100KB (102400 bytes)

### GET /api/templates

Gibt alle Templates des authentifizierten Users zurück.

#### Request

```http
GET /api/templates
Authorization: Bearer <access_token>
```

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "templates": [
    {
      "id": 1,
      "user_id": 1,
      "name": "Meeting Notes",
      "description": "Template for meeting protocols",
      "title": "Meeting {{date}}",
      "content": "# Meeting Notes\n\nDate: {{date}}\nTime: {{time}}\n\n## Agenda\n{{cursor}}\n\n## Action Items\n-",
      "created_at": "2026-01-18T10:00:00Z",
      "updated_at": "2026-01-18T10:00:00Z"
    }
  ]
}
```

#### Errors

```http
401 Unauthorized - User nicht authentifiziert
500 Internal Server Error - Fehler beim Abrufen der Templates
```

---

### GET /api/templates/:id

Gibt ein einzelnes Template zurück.

#### Request

```http
GET /api/templates/1
Authorization: Bearer <access_token>
```

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "id": 1,
  "user_id": 1,
  "name": "Meeting Notes",
  "description": "Template for meeting protocols",
  "title": "Meeting {{date}}",
  "content": "# Meeting Notes\n\nDate: {{date}}\nTime: {{time}}\n\n## Agenda\n{{cursor}}",
  "created_at": "2026-01-18T10:00:00Z",
  "updated_at": "2026-01-18T10:00:00Z"
}
```

#### Errors

```http
401 Unauthorized - User nicht authentifiziert
404 Not Found - Template existiert nicht oder gehört anderem User
500 Internal Server Error - Fehler beim Abrufen
```

---

### POST /api/templates

Erstellt ein neues Template.

#### Request

```http
POST /api/templates
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "name": "Meeting Notes",
  "description": "Template for meeting protocols",
  "title": "Meeting {{date}}",
  "content": "# Meeting Notes\n\nDate: {{date}}\nTime: {{time}}\n\n## Agenda\n{{cursor}}"
}
```

#### Request Fields

| Feld | Typ | Beschreibung |
|------|-----|--------------|
| `name` | string | **Erforderlich**. Name des Templates (1-100 Zeichen) |
| `description` | string | Optionale Beschreibung |
| `title` | string | **Erforderlich**. Titel-Template für neue Notizen (1-200 Zeichen) |
| `content` | string | **Erforderlich**. Inhalt-Template (max. 100KB) |

#### Response

```http
201 Created
Content-Type: application/json
```

```json
{
  "id": 1,
  "user_id": 1,
  "name": "Meeting Notes",
  "description": "Template for meeting protocols",
  "title": "Meeting {{date}}",
  "content": "# Meeting Notes\n\nDate: {{date}}\nTime: {{time}}\n\n## Agenda\n{{cursor}}",
  "created_at": "2026-01-18T10:00:00Z",
  "updated_at": "2026-01-18T10:00:00Z"
}
```

#### Errors

```http
400 Bad Request - Ungültige Request-Daten (z.B. Content zu groß)
401 Unauthorized - User nicht authentifiziert
500 Internal Server Error - Fehler beim Erstellen
```

---

### PUT /api/templates/:id

Aktualisiert ein bestehendes Template.

#### Request

```http
PUT /api/templates/1
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "name": "Meeting Notes Updated",
  "description": "Updated description",
  "title": "Meeting {{date}} - Updated",
  "content": "# New Content\n{{cursor}}"
}
```

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "id": 1,
  "user_id": 1,
  "name": "Meeting Notes Updated",
  "description": "Updated description",
  "title": "Meeting {{date}} - Updated",
  "content": "# New Content\n{{cursor}}",
  "created_at": "2026-01-18T10:00:00Z",
  "updated_at": "2026-01-18T10:15:00Z"
}
```

#### Errors

```http
400 Bad Request - Ungültige Request-Daten
401 Unauthorized - User nicht authentifiziert
404 Not Found - Template existiert nicht oder gehört anderem User
500 Internal Server Error - Fehler beim Aktualisieren
```

---

### DELETE /api/templates/:id

Löscht ein Template.

#### Request

```http
DELETE /api/templates/1
Authorization: Bearer <access_token>
```

#### Response

```http
204 No Content
```

#### Errors

```http
401 Unauthorized - User nicht authentifiziert
404 Not Found - Template existiert nicht oder gehört anderem User
500 Internal Server Error - Fehler beim Löschen
```

---

## Snippets

Snippets sind wiederverwendbare Textbausteine für schnelles Einfügen im Editor.

**Unterschied zu Templates**:
- Snippets haben kein `title`-Feld (nur `content`)
- Snippets werden via Slash-Command (`/`) im Editor eingefügt
- Templates erstellen neue Notizen, Snippets fügen Text an Cursor-Position ein

**Platzhalter**: Identisch zu Templates (`{{date}}`, `{{time}}`, `{{cursor}}`)

**Limits**:
- Name: max. 100 Zeichen
- Content: max. 100KB (102400 bytes)

### GET /api/snippets

Gibt alle Snippets des authentifizierten Users zurück.

#### Request

```http
GET /api/snippets
Authorization: Bearer <access_token>
```

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "snippets": [
    {
      "id": 1,
      "user_id": 1,
      "name": "Signature",
      "description": "Email signature",
      "content": "Best regards,\nJohn Doe\n{{cursor}}",
      "shortcut": "",
      "created_at": "2026-01-18T10:00:00Z",
      "updated_at": "2026-01-18T10:00:00Z"
    }
  ]
}
```

#### Errors

```http
401 Unauthorized - User nicht authentifiziert
500 Internal Server Error - Fehler beim Abrufen
```

---

### GET /api/snippets/:id

Gibt ein einzelnes Snippet zurück.

#### Request

```http
GET /api/snippets/1
Authorization: Bearer <access_token>
```

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "id": 1,
  "user_id": 1,
  "name": "Signature",
  "description": "Email signature",
  "content": "Best regards,\nJohn Doe\n{{cursor}}",
  "shortcut": "",
  "created_at": "2026-01-18T10:00:00Z",
  "updated_at": "2026-01-18T10:00:00Z"
}
```

#### Errors

```http
401 Unauthorized - User nicht authentifiziert
404 Not Found - Snippet existiert nicht oder gehört anderem User
500 Internal Server Error - Fehler beim Abrufen
```

---

### POST /api/snippets

Erstellt ein neues Snippet.

#### Request

```http
POST /api/snippets
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "name": "Signature",
  "description": "Email signature",
  "content": "Best regards,\nJohn Doe\n{{cursor}}",
  "shortcut": ""
}
```

#### Request Fields

| Feld | Typ | Beschreibung |
|------|-----|--------------|
| `name` | string | **Erforderlich**. Name des Snippets (1-100 Zeichen) |
| `description` | string | Optionale Beschreibung |
| `content` | string | **Erforderlich**. Snippet-Inhalt (max. 100KB) |
| `shortcut` | string | Optional. Text-Trigger für zukünftige Auto-Expansion |

#### Response

```http
201 Created
Content-Type: application/json
```

```json
{
  "id": 1,
  "user_id": 1,
  "name": "Signature",
  "description": "Email signature",
  "content": "Best regards,\nJohn Doe\n{{cursor}}",
  "shortcut": "",
  "created_at": "2026-01-18T10:00:00Z",
  "updated_at": "2026-01-18T10:00:00Z"
}
```

#### Errors

```http
400 Bad Request - Ungültige Request-Daten
401 Unauthorized - User nicht authentifiziert
500 Internal Server Error - Fehler beim Erstellen
```

---

### PUT /api/snippets/:id

Aktualisiert ein bestehendes Snippet.

#### Request

```http
PUT /api/snippets/1
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "name": "Signature Updated",
  "description": "New description",
  "content": "Updated content\n{{cursor}}",
  "shortcut": "sig"
}
```

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "id": 1,
  "user_id": 1,
  "name": "Signature Updated",
  "description": "New description",
  "content": "Updated content\n{{cursor}}",
  "shortcut": "sig",
  "created_at": "2026-01-18T10:00:00Z",
  "updated_at": "2026-01-18T10:15:00Z"
}
```

#### Errors

```http
400 Bad Request - Ungültige Request-Daten
401 Unauthorized - User nicht authentifiziert
404 Not Found - Snippet existiert nicht oder gehört anderem User
500 Internal Server Error - Fehler beim Aktualisieren
```

---

### DELETE /api/snippets/:id

Löscht ein Snippet.

#### Request

```http
DELETE /api/snippets/1
Authorization: Bearer <access_token>
```

#### Response

```http
204 No Content
```

#### Errors

```http
401 Unauthorized - User nicht authentifiziert
404 Not Found - Snippet existiert nicht oder gehört anderem User
500 Internal Server Error - Fehler beim Löschen
```

---

## Users

User-Einstellungen und Profilverwaltung.

### GET /api/users/preferences

Gibt die Benutzereinstellungen zurück. Erstellt Default-Werte, falls noch keine existieren.

#### Request

```http
GET /api/users/preferences
Authorization: Bearer <access_token>
```

#### Response

```http
200 OK
```

```json
{
  "theme": "default-dark",
  "editor_mode": "split",
  "created": false
}
```

| Feld | Beschreibung |
|------|--------------|
| `theme` | Aktuelles Theme (z.B. "default-dark", "default-light", "nord-dark", etc.) |
| `editor_mode` | Editor-Modus ("edit", "preview", "split") |
| `created` | `true` wenn Default-Werte neu erstellt wurden |

---

### PUT /api/users/preferences

Aktualisiert die Benutzereinstellungen.

#### Request

```http
PUT /api/users/preferences
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "theme": "nord-dark",
  "editor_mode": "split"
}
```

| Parameter | Typ | Pflicht | Beschreibung |
|-----------|-----|---------|--------------|
| `theme` | string | Ja | Theme-Name (siehe verfügbare Themes) |
| `editor_mode` | string | Ja | "edit", "preview" oder "split" |

**Verfügbare Themes:**
- `default-dark`, `default-light`
- `nord-dark`, `nord-light`
- `solarized-dark`, `solarized-light`
- `dracula`
- `catppuccin-latte`, `catppuccin-mocha`

#### Response

```http
200 OK
```

```json
{
  "theme": "nord-dark",
  "editor_mode": "split",
  "created": false
}
```

#### Errors

```http
400 Bad Request - "invalid theme" oder "invalid editor mode"
401 Unauthorized - User nicht authentifiziert
500 Internal Server Error - Fehler beim Speichern
```

---

### PUT /api/users/email

Ändert die E-Mail-Adresse des Benutzers. Erfordert Passwort-Verifikation. Invalidiert alle anderen Sessions.

#### Request

```http
PUT /api/users/email
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "new_email": "newemail@example.com",
  "current_password": "mein-passwort"
}
```

#### Response

```http
200 OK
```

```json
{
  "message": "email changed successfully"
}
```

#### Errors

```http
400 Bad Request - "new email is required", "current password is required", "invalid email format"
401 Unauthorized - "incorrect password"
409 Conflict - "email already in use"
500 Internal Server Error - Fehler beim Ändern
```

---

### PUT /api/users/password

Ändert das Passwort des Benutzers. Erfordert aktuelles Passwort. Invalidiert alle anderen Sessions.

#### Request

```http
PUT /api/users/password
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "current_password": "altes-passwort",
  "new_password": "neues-sicheres-passwort"
}
```

#### Response

```http
200 OK
```

```json
{
  "message": "password changed successfully"
}
```

#### Errors

```http
400 Bad Request - "current password is required", "new password is required", "password must be at least 8 characters"
401 Unauthorized - "incorrect password"
500 Internal Server Error - Fehler beim Ändern
```

---

### PUT /api/users/preferences/encryption

Aktualisiert Verschlüsselungs-spezifische Einstellungen.

#### Request

```http
PUT /api/users/preferences/encryption
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "keywords_enabled": true,
  "encrypt_titles": false
}
```

#### Response

```http
200 OK
```

```json
{
  "message": "encryption preferences updated successfully"
}
```

#### Errors

```http
400 Bad Request - "invalid request body"
401 Unauthorized - Keine gültige Authentifizierung
500 Internal Server Error - Fehler beim Aktualisieren
```

---

### PUT /api/users/preferences/security

Aktualisiert Sicherheits-spezifische Einstellungen (Security Level, Auto-Lock).

#### Request

```http
PUT /api/users/preferences/security
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "security_level": "balanced",
  "auto_lock_timeout": 15
}
```

**Gültige Werte:**
- `security_level`: `"paranoid"`, `"balanced"`, `"convenient"`
- `auto_lock_timeout`: `0` (deaktiviert), `5`, `15`, `30`, `60` (Minuten)

#### Response

```http
200 OK
```

```json
{
  "theme": "dark",
  "editor_mode": "vim",
  "keywords_enabled": true,
  "encrypt_titles": false,
  "security_level": "balanced",
  "auto_lock_timeout": 15,
  "credentials": [],
  "created": false
}
```

#### Errors

```http
400 Bad Request - "invalid security level", "invalid auto-lock timeout"
401 Unauthorized - Keine gültige Authentifizierung
500 Internal Server Error - Fehler beim Aktualisieren
```

---

### POST /api/users/recovery-key

Setzt einen Recovery Key für den authentifizierten Benutzer. Ermöglicht Passwort-Wiederherstellung.

#### Request

```http
POST /api/users/recovery-key
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "recovery_key_hash": "argon2id-hash-des-recovery-keys",
  "salt": "base64-encoded-salt"
}
```

#### Response

```http
200 OK
```

```json
{
  "message": "recovery key set successfully"
}
```

#### Errors

```http
400 Bad Request - "recovery_key_hash is required", "salt is required", "invalid base64 salt"
401 Unauthorized - Keine gültige Authentifizierung
500 Internal Server Error - Fehler beim Setzen
```

---

### GET /api/users/recovery-key/salt

Ruft das Recovery Key Salt für den authentifizierten Benutzer ab.

#### Request

```http
GET /api/users/recovery-key/salt
Authorization: Bearer <access_token>
```

#### Response

```http
200 OK
```

```json
{
  "salt": "base64-encoded-salt"
}
```

#### Errors

```http
401 Unauthorized - Keine gültige Authentifizierung
404 Not Found - "no recovery key set"
500 Internal Server Error - Fehler beim Abrufen
```

---

### POST /api/users/webauthn/credentials

Fügt eine neue WebAuthn-Credential (z.B. YubiKey) hinzu.

#### Request

```http
POST /api/users/webauthn/credentials
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "credential_id": "base64-credential-id",
  "device_name": "YubiKey 5"
}
```

#### Response

```http
201 Created
```

```json
{
  "id": 1,
  "credential_id": "base64-credential-id",
  "device_name": "YubiKey 5",
  "created_at": "2026-01-22T10:00:00Z",
  "last_used_at": null
}
```

#### Errors

```http
400 Bad Request - "credential_id required"
401 Unauthorized - Keine gültige Authentifizierung
500 Internal Server Error - Fehler beim Hinzufügen
```

---

### DELETE /api/users/webauthn/credentials

Entfernt eine WebAuthn-Credential.

#### Request

```http
DELETE /api/users/webauthn/credentials
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "credential_id": "base64-credential-id"
}
```

#### Response

```http
200 OK
```

```json
{
  "message": "credential deleted successfully"
}
```

#### Errors

```http
400 Bad Request - "credential_id required"
401 Unauthorized - Keine gültige Authentifizierung
404 Not Found - "credential not found"
500 Internal Server Error - Fehler beim Löschen
```

---

### PATCH /api/users/webauthn/credentials/touch

Aktualisiert den "last_used_at" Zeitstempel einer WebAuthn-Credential.

#### Request

```http
PATCH /api/users/webauthn/credentials/touch
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "credential_id": "base64-credential-id"
}
```

#### Response

```http
200 OK
```

```json
{
  "message": "credential touched successfully"
}
```

#### Errors

```http
400 Bad Request - "credential_id required"
401 Unauthorized - Keine gültige Authentifizierung
404 Not Found - "credential not found"
500 Internal Server Error - Fehler beim Aktualisieren
```

---

## Admin

**Authentifizierung**: Alle Admin-Endpunkte erfordern einen gültigen JWT/Cookie **und** Admin-Status (`is_admin = 1`).

**Zugriff**: Nur Administratoren haben Zugriff. Nicht-Admins erhalten `403 Forbidden`.

**Detaillierte Dokumentation**: Siehe [Admin Panel Dokumentation](admin-panel.md) für ausführliche Erklärungen aller Features, Sicherheitsmaßnahmen und Best Practices.

---

### GET /api/admin/stats

Basis-Systemstatistiken abrufen.

#### Request

```http
GET /api/admin/stats
Authorization: Bearer <jwt>
```

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "total_users": 10,
  "total_notes": 250,
  "total_folders": 35,
  "total_tags": 42,
  "storage_used_mb": 125.5
}
```

#### Errors

```http
401 Unauthorized - Keine gültige Authentifizierung
403 Forbidden - Kein Admin-Zugriff
500 Internal Server Error - Fehler beim Abrufen der Statistiken
```

---

### GET /api/admin/stats/detailed

Detaillierte Systemstatistiken mit Zeitreihen (letzte 30 Tage).

#### Request

```http
GET /api/admin/stats/detailed
Authorization: Bearer <jwt>
```

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "stats": {
    "total_users": 10,
    "total_notes": 250,
    "total_folders": 35,
    "total_tags": 42,
    "storage_used_mb": 125.5
  },
  "user_growth": [
    { "date": "2026-01-15", "count": 2 },
    { "date": "2026-01-16", "count": 1 }
  ],
  "note_growth": [
    { "date": "2026-01-15", "count": 15 },
    { "date": "2026-01-16", "count": 22 }
  ],
  "storage_trend": [
    { "date": "2026-01-15", "value": 120.0 },
    { "date": "2026-01-16", "value": 125.5 }
  ]
}
```

#### Errors

```http
401 Unauthorized - Keine gültige Authentifizierung
403 Forbidden - Kein Admin-Zugriff
500 Internal Server Error - Fehler beim Abrufen der Statistiken
```

---

### GET /api/admin/users

Alle Benutzer mit Statistiken abrufen.

#### Request

```http
GET /api/admin/users
Authorization: Bearer <jwt>
```

#### Response

```http
200 OK
Content-Type: application/json
```

```json
[
  {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "is_admin": true,
    "note_count": 50,
    "storage_mb": 25.5,
    "created_at": "2026-01-01T10:00:00Z"
  },
  {
    "id": 2,
    "username": "user1",
    "email": "user1@example.com",
    "is_admin": false,
    "note_count": 10,
    "storage_mb": 5.2,
    "created_at": "2026-01-02T14:30:00Z"
  }
]
```

#### Errors

```http
401 Unauthorized - Keine gültige Authentifizierung
403 Forbidden - Kein Admin-Zugriff
500 Internal Server Error - Fehler beim Abrufen der Benutzer
```

---

### GET /api/admin/users/:id

Details zu einem einzelnen Benutzer abrufen.

#### Request

```http
GET /api/admin/users/2
Authorization: Bearer <jwt>
```

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "id": 2,
  "username": "user1",
  "email": "user1@example.com",
  "is_admin": false,
  "note_count": 10,
  "storage_mb": 5.2,
  "created_at": "2026-01-02T14:30:00Z"
}
```

#### Errors

```http
400 Bad Request - Ungültige User-ID
401 Unauthorized - Keine gültige Authentifizierung
403 Forbidden - Kein Admin-Zugriff
404 Not Found - Benutzer nicht gefunden
500 Internal Server Error - Fehler beim Abrufen der Details
```

---

### PUT /api/admin/users/:id/admin

Admin-Status eines Benutzers ändern.

#### Request

```http
PUT /api/admin/users/2/admin
Authorization: Bearer <jwt>
Content-Type: application/json
```

```json
{
  "is_admin": true
}
```

#### Response

```http
204 No Content
```

#### Errors

```http
400 Bad Request - Ungültige Request-Daten
401 Unauthorized - Keine gültige Authentifizierung
403 Forbidden - "cannot demote yourself" oder kein Admin-Zugriff
404 Not Found - Benutzer nicht gefunden
500 Internal Server Error - Fehler beim Aktualisieren
```

**Sicherheit**: Administratoren können ihren eigenen Admin-Status nicht entziehen.

---

### DELETE /api/admin/users/:id

Benutzer vollständig löschen (inklusive aller Notizen, Ordner, Uploads).

#### Request

```http
DELETE /api/admin/users/2
Authorization: Bearer <jwt>
```

#### Response

```http
204 No Content
```

#### Errors

```http
400 Bad Request - Ungültige User-ID
401 Unauthorized - Keine gültige Authentifizierung
403 Forbidden - "cannot delete yourself" oder kein Admin-Zugriff
404 Not Found - Benutzer nicht gefunden
500 Internal Server Error - Fehler beim Löschen
```

**Sicherheit**: Administratoren können sich nicht selbst löschen. Alle Benutzer-Daten werden permanent gelöscht (keine Wiederherstellung möglich).

---

### GET /api/admin/activity

Activity Logs mit Filterung und Pagination abrufen.

#### Request

```http
GET /api/admin/activity?limit=20&page=1&action=login&date_from=2026-01-15
Authorization: Bearer <jwt>
```

#### Query Parameters

| Parameter | Typ | Default | Beschreibung |
|-----------|-----|---------|--------------|
| `limit` | int | 50 | Einträge pro Seite (1-100) |
| `page` | int | 1 | Seitennummer |
| `action` | string | - | Action-Typ filtern (z.B. "login", "note_create") |
| `user_id` | int | - | Nach User-ID filtern |
| `target_type` | string | - | Target-Typ filtern (z.B. "note", "user") |
| `date_from` | string | - | Startdatum (ISO 8601) |
| `date_to` | string | - | Enddatum (ISO 8601) |

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "logs": [
    {
      "id": 123,
      "user_id": 2,
      "username": "user1",
      "action": "login",
      "target_type": null,
      "target_id": null,
      "details": "",
      "ip_address": "192.168.1.10",
      "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)...",
      "created_at": "2026-01-16T10:30:00Z"
    },
    {
      "id": 122,
      "user_id": 1,
      "username": "admin",
      "action": "user_admin_set",
      "target_type": "user",
      "target_id": "2",
      "details": "{\"target_username\":\"user1\",\"is_admin\":true}",
      "ip_address": "192.168.1.5",
      "user_agent": "Mozilla/5.0...",
      "created_at": "2026-01-16T10:00:00Z"
    }
  ],
  "total": 450
}
```

#### Errors

```http
401 Unauthorized - Keine gültige Authentifizierung
403 Forbidden - Kein Admin-Zugriff
500 Internal Server Error - Fehler beim Abrufen der Logs
```

---

### GET /api/admin/settings

Alle Systemeinstellungen abrufen.

#### Request

```http
GET /api/admin/settings
Authorization: Bearer <jwt>
```

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "registration_enabled": "true",
  "max_notes_per_user": "0",
  "max_storage_mb_per_user": "0",
  "maintenance_mode": "false",
  "activity_retention_days": "90"
}
```

#### Errors

```http
401 Unauthorized - Keine gültige Authentifizierung
403 Forbidden - Kein Admin-Zugriff
500 Internal Server Error - Fehler beim Abrufen der Einstellungen
```

---

### PUT /api/admin/settings

Systemeinstellungen aktualisieren. Es können beliebig viele Settings gleichzeitig aktualisiert werden.

#### Request

```http
PUT /api/admin/settings
Authorization: Bearer <jwt>
Content-Type: application/json
```

```json
{
  "registration_enabled": "false",
  "max_notes_per_user": "100"
}
```

#### Response

```http
200 OK
Content-Type: application/json
```

```json
{
  "registration_enabled": "false",
  "max_notes_per_user": "100",
  "max_storage_mb_per_user": "0",
  "maintenance_mode": "false",
  "activity_retention_days": "90"
}
```

#### Settings-Referenz

| Key | Typ | Default | Beschreibung |
|-----|-----|---------|--------------|
| `registration_enabled` | bool | `true` | Öffentliche Registrierung erlauben |
| `max_notes_per_user` | int | `0` | Maximale Anzahl Notizen pro User (0 = unbegrenzt) |
| `max_storage_mb_per_user` | int | `0` | Maximaler Speicher in MB pro User (0 = unbegrenzt) |
| `maintenance_mode` | bool | `false` | Wartungsmodus aktivieren |
| `activity_retention_days` | int | `90` | Aufbewahrungszeit für Activity Logs in Tagen |

#### Errors

```http
400 Bad Request - Ungültige Einstellungen, unbekannter Key oder leere Request
401 Unauthorized - Keine gültige Authentifizierung
403 Forbidden - Kein Admin-Zugriff
500 Internal Server Error - Fehler beim Speichern der Einstellungen
```

**Validierung**:
- Boolean-Werte müssen "true" oder "false" sein (als String)
- Numerische Werte müssen >= 0 sein
- Unbekannte Keys führen zu 400 Bad Request

---

## Note Sharing

Notizen koennen mit anderen Benutzern geteilt werden. Unterstuetzte Rollen: `viewer` (nur lesen) und `editor` (lesen + bearbeiten). E2E-verschluesselte Notizen koennen nicht geteilt werden -- muessen vorher ueber `POST /api/notes/:id/decrypt` entschluesselt werden. Beim erneuten Verschluesseln einer geteilten Notiz werden alle Shares automatisch entfernt.

**Hinweis:** `GET /api/shared` gibt sowohl direkt geteilte Notizen (via `note_shares`) als auch Notizen aus geteilten Ordnern (via `folder_shares`) zurueck. Deduplizierung: Wenn eine Notiz sowohl direkt als auch ueber einen Ordner geteilt ist, wird sie nur einmal gelistet (mit der `note_share`-Rolle). Siehe auch [Folder Sharing](#folder-sharing).

### POST /api/notes/:id/shares

Teilt eine Notiz mit einem anderen Benutzer. Nur der Besitzer der Notiz kann Shares erstellen.

#### Request

```http
POST /api/notes/abc123/shares
Content-Type: application/json
```

```json
{
  "user_id": 42,
  "role": "viewer"
}
```

| Feld | Typ | Pflicht | Beschreibung |
|------|-----|---------|--------------|
| `user_id` | integer | Ja | ID des Benutzers, mit dem geteilt wird |
| `role` | string | Ja | Rolle: `viewer` oder `editor` |

#### Response

```http
201 Created
```

```json
{
  "id": 1,
  "note_id": "abc123",
  "user_id": 42,
  "role": "viewer",
  "shared_by": 1,
  "created_at": "2026-02-07T12:00:00Z"
}
```

#### Fehler

| Code | Bedeutung |
|------|-----------|
| 400 | Ungueltige Rolle, Selbst-Teilen, oder Notiz ist verschluesselt |
| 403 | Benutzer ist nicht Besitzer der Notiz |
| 404 | Notiz oder Ziel-Benutzer nicht gefunden |
| 409 | Share existiert bereits |

---

### GET /api/notes/:id/shares

Listet alle Shares einer Notiz auf. Nur fuer den Besitzer der Notiz.

#### Request

```http
GET /api/notes/abc123/shares
```

#### Response

```http
200 OK
```

```json
[
  {
    "id": 1,
    "note_id": "abc123",
    "user_id": 42,
    "username": "jane",
    "email": "jane@example.com",
    "role": "viewer",
    "shared_by": 1,
    "created_at": "2026-02-07T12:00:00Z"
  }
]
```

---

### PUT /api/notes/:id/shares/:userId

Aendert die Rolle eines bestehenden Shares. Nur fuer den Besitzer der Notiz.

#### Request

```http
PUT /api/notes/abc123/shares/42
Content-Type: application/json
```

```json
{
  "role": "editor"
}
```

#### Response

```http
200 OK
```

```json
{
  "message": "role updated"
}
```

#### Fehler

| Code | Bedeutung |
|------|-----------|
| 400 | Ungueltige Rolle |
| 403 | Benutzer ist nicht Besitzer der Notiz |
| 404 | Share nicht gefunden |

---

### DELETE /api/notes/:id/shares/:userId

Entfernt einen Share. Nur fuer den Besitzer der Notiz.

#### Request

```http
DELETE /api/notes/abc123/shares/42
```

#### Response

```http
204 No Content
```

---

### GET /api/shared

Listet alle Notizen auf, die mit dem aktuellen Benutzer geteilt wurden.

#### Request

```http
GET /api/shared
```

#### Response

```http
200 OK
```

```json
[
  {
    "id": "abc123",
    "title": "Shared Note",
    "content": "...",
    "role": "viewer",
    "owner_username": "john",
    "owner_id": 1,
    "shared_at": "2026-02-07T12:00:00Z"
  }
]
```

---

### GET /api/shared/:id

Ruft eine einzelne geteilte Notiz ab.

#### Request

```http
GET /api/shared/abc123
```

#### Response

```http
200 OK
```

```json
{
  "id": "abc123",
  "title": "Shared Note",
  "content": "...",
  "role": "editor",
  "owner_username": "john",
  "owner_id": 1,
  "shared_at": "2026-02-07T12:00:00Z"
}
```

#### Fehler

| Code | Bedeutung |
|------|-----------|
| 403 | Benutzer hat keinen Zugriff auf diese Notiz |
| 404 | Notiz nicht gefunden |

---

### PUT /api/shared/:id

Bearbeitet eine geteilte Notiz. Nur fuer Benutzer mit `editor`-Rolle.

#### Request

```http
PUT /api/shared/abc123
Content-Type: application/json
```

```json
{
  "title": "Updated Title",
  "content": "Updated content..."
}
```

#### Response

```http
200 OK
```

```json
{
  "id": "abc123",
  "title": "Updated Title",
  "content": "Updated content...",
  "updated_at": "2026-02-07T12:30:00Z"
}
```

#### Fehler

| Code | Bedeutung |
|------|-----------|
| 403 | Benutzer hat keine Editor-Rolle fuer diese Notiz |
| 404 | Notiz nicht gefunden |

---

### GET /api/users/search

Sucht Benutzer nach Username oder E-Mail. Fuer die User-Auswahl beim Teilen.

#### Request

```http
GET /api/users/search?q=jane
```

| Parameter | Typ | Pflicht | Beschreibung |
|-----------|-----|---------|--------------|
| `q` | string | Ja | Suchbegriff (min. 2 Zeichen) |

#### Response

```http
200 OK
```

```json
[
  {
    "id": 42,
    "username": "jane",
    "email": "jane@example.com"
  }
]
```

**Hinweis:** Der aktuelle Benutzer wird aus den Ergebnissen ausgeschlossen.

---

## Folder Sharing

Ganze Ordner koennen mit anderen Benutzern geteilt werden. Alle Notizen im Ordner sind implizit geteilt (Permission-Vererbung). Unterstuetzte Rollen: `viewer` (nur lesen) und `editor` (lesen + bearbeiten).

**Einschraenkungen:**
- Ordner mit `encryption_default = true` koennen nicht geteilt werden
- Ordner mit verschluesselten Notizen koennen nicht geteilt werden
- Verschluesselte Notizen in geteilten Ordnern werden aus der Anzeige gefiltert
- Verschluesselung kann nicht aktiviert werden solange ein Ordner geteilt ist

**Permission-Chain:** Bei Zugriffspruefung gilt: `note_shares` (expliziter Share) hat Vorrang vor `folder_shares` (impliziter Share). Wenn beides existiert, gewinnt der note_share.

### POST /api/folders/:id/shares

Teilt einen Ordner mit einem anderen Benutzer. Nur der Besitzer des Ordners kann Shares erstellen.

#### Request

```http
POST /api/folders/5/shares
Content-Type: application/json
```

```json
{
  "identifier": "jane",
  "role": "viewer"
}
```

| Feld | Typ | Pflicht | Beschreibung |
|------|-----|---------|--------------|
| `identifier` | string | Ja | Username oder E-Mail des Empfaengers |
| `role` | string | Ja | Rolle: `viewer` oder `editor` |

#### Response

```http
201 Created
```

```json
{
  "id": 1,
  "folder_id": 5,
  "folder_path": "/Rezepte",
  "folder_name": "Rezepte",
  "owner_user_id": 1,
  "owner_username": "john",
  "shared_with_user_id": 42,
  "shared_with_username": "jane",
  "role": "viewer",
  "created_at": "2026-02-07T12:00:00Z",
  "updated_at": "2026-02-07T12:00:00Z"
}
```

#### Fehler

| Code | Bedeutung |
|------|-----------|
| 400 | Ungueltige Rolle, Selbst-Teilen, Ordner ist verschluesselt, oder Ordner enthaelt verschluesselte Notizen |
| 403 | Benutzer ist nicht Besitzer des Ordners |
| 404 | Ordner oder Ziel-Benutzer nicht gefunden |
| 409 | Share existiert bereits |

---

### GET /api/folders/:id/shares

Listet alle Shares eines Ordners auf. Nur fuer den Besitzer des Ordners.

#### Request

```http
GET /api/folders/5/shares
```

#### Response

```http
200 OK
```

```json
{
  "shares": [
    {
      "id": 1,
      "folder_id": 5,
      "folder_path": "/Rezepte",
      "folder_name": "Rezepte",
      "owner_user_id": 1,
      "owner_username": "john",
      "shared_with_user_id": 42,
      "shared_with_username": "jane",
      "role": "viewer",
      "created_at": "2026-02-07T12:00:00Z",
      "updated_at": "2026-02-07T12:00:00Z"
    }
  ]
}
```

---

### PUT /api/folders/:id/shares/:userId

Aendert die Rolle eines bestehenden Folder-Shares. Nur fuer den Besitzer des Ordners.

#### Request

```http
PUT /api/folders/5/shares/42
Content-Type: application/json
```

```json
{
  "role": "editor"
}
```

#### Response

```http
204 No Content
```

#### Fehler

| Code | Bedeutung |
|------|-----------|
| 400 | Ungueltige Rolle |
| 403 | Benutzer ist nicht Besitzer des Ordners |
| 404 | Share nicht gefunden |

---

### DELETE /api/folders/:id/shares/:userId

Entfernt einen Folder-Share. Entfernt auch alle zugehoerigen Shared Note Placements fuer Notizen in diesem Ordner. Nur fuer den Besitzer des Ordners.

#### Request

```http
DELETE /api/folders/5/shares/42
```

#### Response

```http
204 No Content
```

#### Fehler

| Code | Bedeutung |
|------|-----------|
| 403 | Benutzer ist nicht Besitzer des Ordners |
| 404 | Share nicht gefunden |

---

### GET /api/shared/folders

Listet alle Ordner auf, die mit dem aktuellen Benutzer geteilt wurden.

#### Request

```http
GET /api/shared/folders
```

#### Response

```http
200 OK
```

```json
{
  "folders": [
    {
      "id": 5,
      "path": "/Rezepte",
      "name": "Rezepte",
      "note_count": 12,
      "shared_by": "john",
      "share_role": "viewer",
      "share_id": 1,
      "created_at": "2026-02-07T12:00:00Z",
      "updated_at": "2026-02-07T12:00:00Z"
    }
  ]
}
```

| Feld | Typ | Beschreibung |
|------|-----|--------------|
| `note_count` | integer | Anzahl sichtbarer (unverschluesselter, nicht-geloeschter) Notizen im Ordner |

---

### GET /api/shared/folders/:id/notes

Listet alle Notizen in einem geteilten Ordner auf. Verschluesselte und geloeschte Notizen werden ausgeschlossen.

#### Request

```http
GET /api/shared/folders/5/notes
```

#### Response

```http
200 OK
```

```json
{
  "notes": [
    {
      "id": "abc123",
      "title": "Pasta Rezept",
      "content": "...",
      "folder_path": "/Rezepte",
      "version": 3,
      "shared_by": "john",
      "role": "viewer",
      "share_id": 1,
      "created_at": "2026-02-07T12:00:00Z",
      "updated_at": "2026-02-07T14:30:00Z"
    }
  ]
}
```

---

## Shared Note Placements

Empfaenger koennen geteilte Notizen in ihre eigenen Ordner einordnen. Die Notiz bleibt beim Besitzer, erscheint aber auch im Ordner des Empfaengers (mit "Geteilt"-Badge). Placements werden automatisch entfernt wenn der Share entzogen wird.

### POST /api/shared/:id/placement

Ordnet eine geteilte Notiz in einen eigenen Ordner ein. Nur fuer Notizen, auf die der Benutzer ueber einen aktiven Share Zugriff hat.

#### Request

```http
POST /api/shared/abc123/placement
Content-Type: application/json
```

```json
{
  "folder_id": 10
}
```

| Feld | Typ | Pflicht | Beschreibung |
|------|-----|---------|--------------|
| `folder_id` | integer | Ja | ID des eigenen Ziel-Ordners |

#### Response

```http
204 No Content
```

#### Fehler

| Code | Bedeutung |
|------|-----------|
| 400 | `folder_id` fehlt, eigene Notiz (nicht geteilte), oder Ziel-Ordner gehoert nicht dem Benutzer |
| 403 | Benutzer hat keinen aktiven Share-Zugriff auf diese Notiz |

---

### DELETE /api/shared/:id/placement

Entfernt die Einordnung einer geteilten Notiz aus dem eigenen Ordner. Die Notiz bleibt weiterhin unter "Geteilt mit mir" sichtbar.

#### Request

```http
DELETE /api/shared/abc123/placement
```

#### Response

```http
204 No Content
```

#### Fehler

| Code | Bedeutung |
|------|-----------|
| 404 | Keine Einordnung gefunden |

---

## Health

### GET /health

Health Check Endpoint für Monitoring.

#### Request

```http
GET /health
```

#### Response

```http
200 OK
```

```
ok
```

#### Verwendung

**Docker Health Check**:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s \
  CMD curl -f http://localhost:8080/health || exit 1
```

**Monitoring**:

```bash
# Uptime Kuma, Prometheus, etc.
curl http://localhost:8080/health
```

---

## Error Codes

### HTTP Status Codes

| Code | Bedeutung | Beispiel |
|------|-----------|----------|
| 200 | OK | Erfolgreiche GET/PUT Request |
| 201 | Created | Note erfolgreich erstellt |
| 204 | No Content | Note erfolgreich gelöscht |
| 400 | Bad Request | Validation Error, fehlender Parameter |
| 404 | Not Found | Note nicht gefunden |
| 409 | Conflict | Version Mismatch bei PUT |
| 500 | Internal Server Error | Datenbank-Fehler, unerwarteter Error |

### Error Response Format

Alle Errors geben JSON zurück:

```json
{
  "error": "Human-readable error message"
}
```

**Beispiele**:

```json
// 400 Bad Request
{
  "error": "title is required"
}

// 404 Not Found
{
  "error": "note not found"
}

// 409 Conflict
{
  "error": "version mismatch - note was modified"
}

// 500 Internal Server Error
{
  "error": "failed to update note: database is locked"
}
```

### Client Error Handling

**Empfohlene Strategie**:

```typescript
async function updateNote(id: string, data: NoteRequest, version: number) {
  try {
    const response = await fetch(`/api/notes/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'If-Match': version.toString()
      },
      body: JSON.stringify(data)
    });

    if (response.status === 409) {
      // Conflict: Note wurde zwischenzeitlich geändert
      alert('Note was modified. Please reload and try again.');
      return;
    }

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Unknown error');
    }

    return await response.json();
  } catch (error) {
    console.error('Update failed:', error);
    throw error;
  }
}
```

---

## Rate Limiting

**Aktuell**: Kein Rate Limiting implementiert.

**Empfehlung**: Reverse Proxy (nginx, Caddy) für Rate Limiting konfigurieren:

```nginx
# nginx.conf
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;

location /api/ {
    limit_req zone=api burst=20;
    proxy_pass http://localhost:8080;
}
```

---

## Pagination Best Practices

### Cursor-based Pagination

**Vorteile** gegenüber Offset-based:

- Stabil bei Updates (neue Notes ändern nicht Page-Offsets)
- Performant (Index-Scan, kein OFFSET Skip)

**Verwendung**:

```typescript
let cursor = '';
let allNotes = [];

while (true) {
  const response = await fetch(`/api/notes?limit=50&cursor=${cursor}`);
  const data = await response.json();

  allNotes.push(...data.notes);

  if (!data.next_cursor) break;  // Letzte Page
  cursor = data.next_cursor;
}
```

**Hinweis**: Cursor enthält Timestamp + ID, nicht manipulieren!

---

## Concurrency & Locking

### Optimistic Locking Flow

```
Client A: GET /api/notes/123 → version: 5
Client B: GET /api/notes/123 → version: 5

Client A: PUT /api/notes/123 (If-Match: 5) → SUCCESS → version: 6
Client B: PUT /api/notes/123 (If-Match: 5) → 409 Conflict

Client B: GET /api/notes/123 → version: 6 (neue Content)
Client B: Merge changes + PUT (If-Match: 6) → SUCCESS
```

### Read-Uncommitted Risk

SQLite WAL-Modus ermöglicht parallele Reads:

- **GET** während **PUT**: Liest alte Version (vor Commit)
- **PUT** während **GET**: Kein Lock (GET sieht alte Version)

**Für xelanote OK**: Single-User App, keine Critical Consistency Anforderungen.

---

## API Client Beispiel

### TypeScript/JavaScript

```typescript
// api.ts
const BASE_URL = 'http://localhost:8080';

export interface Note {
  id: string;
  title: string;
  content: string;
  folder_path: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export async function getNotes(limit = 50, cursor = ''): Promise<{
  notes: Note[];
  next_cursor: string;
}> {
  const url = `${BASE_URL}/api/notes?limit=${limit}&cursor=${cursor}`;
  const response = await fetch(url);
  if (!response.ok) throw new Error('Failed to fetch notes');
  return response.json();
}

export async function getNote(id: string): Promise<Note> {
  const response = await fetch(`${BASE_URL}/api/notes/${id}`);
  if (!response.ok) throw new Error('Note not found');
  return response.json();
}

export async function createNote(data: {
  title: string;
  content: string;
  folder_path?: string;
}): Promise<Note> {
  const response = await fetch(`${BASE_URL}/api/notes`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  });
  if (!response.ok) throw new Error('Failed to create note');
  return response.json();
}

export async function updateNote(
  id: string,
  data: { title: string; content: string },
  version: number
): Promise<Note> {
  const response = await fetch(`${BASE_URL}/api/notes/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      'If-Match': version.toString()
    },
    body: JSON.stringify(data)
  });

  if (response.status === 409) {
    throw new Error('Note was modified by another client');
  }

  if (!response.ok) throw new Error('Failed to update note');
  return response.json();
}

export async function deleteNote(id: string): Promise<void> {
  const response = await fetch(`${BASE_URL}/api/notes/${id}`, {
    method: 'DELETE'
  });
  if (!response.ok) throw new Error('Failed to delete note');
}

export async function renameNote(
  id: string,
  newTitle: string
): Promise<{ note: Note; updated_note_count: number }> {
  const response = await fetch(`${BASE_URL}/api/notes/${id}/rename`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ newTitle })
  });
  if (!response.ok) throw new Error('Failed to rename note');
  return response.json();
}

export async function search(query: string, limit = 20): Promise<{
  query: string;
  results: Array<{
    id: string;
    title: string;
    snippet: string;
    rank: number;
  }>;
}> {
  const url = `${BASE_URL}/api/search?q=${encodeURIComponent(query)}&limit=${limit}`;
  const response = await fetch(url);
  if (!response.ok) throw new Error('Search failed');
  return response.json();
}
```

### Python

```python
import requests
from typing import List, Dict, Optional

BASE_URL = "http://localhost:8080"

class XelanoteClient:
    def __init__(self, base_url: str = BASE_URL):
        self.base_url = base_url

    def get_notes(self, limit: int = 50, cursor: str = "") -> Dict:
        response = requests.get(
            f"{self.base_url}/api/notes",
            params={"limit": limit, "cursor": cursor}
        )
        response.raise_for_status()
        return response.json()

    def get_note(self, note_id: str) -> Dict:
        response = requests.get(f"{self.base_url}/api/notes/{note_id}")
        response.raise_for_status()
        return response.json()

    def create_note(self, title: str, content: str, folder_path: str = "/") -> Dict:
        response = requests.post(
            f"{self.base_url}/api/notes",
            json={"title": title, "content": content, "folder_path": folder_path}
        )
        response.raise_for_status()
        return response.json()

    def update_note(self, note_id: str, title: str, content: str, version: int) -> Dict:
        response = requests.put(
            f"{self.base_url}/api/notes/{note_id}",
            headers={"If-Match": str(version)},
            json={"title": title, "content": content}
        )
        response.raise_for_status()
        return response.json()

    def delete_note(self, note_id: str) -> None:
        response = requests.delete(f"{self.base_url}/api/notes/{note_id}")
        response.raise_for_status()

    def search(self, query: str, limit: int = 20) -> Dict:
        response = requests.get(
            f"{self.base_url}/api/search",
            params={"q": query, "limit": limit}
        )
        response.raise_for_status()
        return response.json()

# Verwendung
client = XelanoteClient()
notes = client.get_notes(limit=10)
```

---

## CSRF Protection

### GET /api/csrf-token

Öffentlicher Endpunkt zum Abrufen eines CSRF-Tokens.

**Response:**
```json
{
  "token": "base64-encoded-token"
}
```

Alle state-changing Requests (POST/PUT/DELETE) erfordern:
- Header: `X-CSRF-Token: <token>`
- Oder: Cookie `xelanote_csrf` (automatisch gesetzt)

Siehe: `docs/security/CSRF-SECURITY-REVIEW-2026-01-28.md` für Details zur Implementierung.

---

## Weitere Ressourcen

- [Architektur-Dokumentation](architecture.md) - System-Design und Datenbank-Schema
- [Development Guide](development.md) - API Testing, Debugging
- [SQLite FTS5 Syntax](https://www.sqlite.org/fts5.html#full_text_query_syntax) - Erweiterte Suchsyntax
