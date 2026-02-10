# Error Reporting (Forgejo Integration)

Automatische Fehlerberichte und manuelles User-Feedback, das direkt als Issues im Forgejo-Repository erstellt wird.

## Uebersicht

Das Error-Reporting-System hat zwei Modi:

1. **Automatisch**: Unbehandelte JavaScript-Fehler (`window.onerror`, `unhandledrejection`) werden als Forgejo-Issues erfasst.
2. **Manuell**: Benutzer koennen ueber den "Issue melden"-Button in der Sidebar Feedback senden.

Der Forgejo-API-Token bleibt serverseitig — das Frontend kennt ihn nicht. Das Ziel-Repo kann privat sein.

## Voraussetzungen

Das Feature ist **nur aktiv**, wenn alle drei Umgebungsvariablen gesetzt sind:

| Variable | Beispiel | Beschreibung |
|----------|----------|--------------|
| `FORGEJO_URL` | `https://<FORGEJO_URL>` | Basis-URL der Forgejo-Instanz |
| `FORGEJO_REPO` | `xela/xelanote` | Repository im Format `owner/name` |
| `FORGEJO_API_TOKEN` | `gta_xxxx...` | API-Token mit Issue-Schreibrechten |

Wenn eine der Variablen fehlt, ist das Feature deaktiviert. Das Frontend erhaelt `error_reporting_enabled: false` von `/api/config` und zeigt den "Issue melden"-Button nicht an.

### Forgejo API-Token erstellen

1. Forgejo oeffnen → Profil → Settings → Applications
2. "Generate New Token" mit Scope **`write:issue`** (minimum)
3. Token in die `.env`-Datei des Servers eintragen

## Architektur

```
Browser                         Backend                          Forgejo
  |                               |                                |
  |-- POST /api/error-reports --> |                                |
  |   (JSON, max 16KB)           |-- POST /api/v1/repos/.../issues |
  |                               |   (mit API-Token)              |
  |                               |                                |
  | <-- { accepted: true } ----  |  <-- 201 Created ------------ |
```

### Ablauf

1. Frontend faengt JS-Fehler oder User klickt "Issue melden"
2. Frontend berechnet einen **Fingerprint** (SHA-256 von `errorType:message`, normalisiert, erste 16 Hex-Zeichen)
3. Frontend sendet Report an `POST /api/error-reports`
4. Backend prueft Rate-Limit und validiert Payload
5. Backend sucht auf Forgejo nach offenem Issue mit gleichem Fingerprint
   - **Gefunden**: Kommentar an bestehendes Issue anfuegen (Deduplizierung)
   - **Nicht gefunden**: Neues Issue erstellen mit Fingerprint-Label

## Deduplizierung

Gleiche Fehler erzeugen keine Flut von Issues. Die Deduplizierung arbeitet auf drei Ebenen:

| Ebene | Mechanismus | Beschreibung |
|-------|-------------|--------------|
| Frontend (Session) | `SvelteSet<fingerprint>` | Gleicher Fehler wird pro Browser-Session nur einmal gesendet |
| Frontend (Rate) | Max 3 Reports / 5 Minuten | Client-seitiges Rate-Limiting |
| Backend (Forgejo) | Fingerprint-Label (`fp:xxxx`) | Gleicher Fingerprint → Kommentar statt neues Issue |

### Fingerprint-Normalisierung

Vor dem Hashing werden dynamische Werte entfernt, damit z.B. `Error in note 42` und `Error in note 99` denselben Fingerprint ergeben:

- UUIDs → `UUID`
- ISO-Timestamps → `DATE`
- Zahlen → `N`

## Labels

Das Backend erstellt automatisch Labels im Forgejo-Repo beim Start:

| Label | Farbe | Beschreibung |
|-------|-------|--------------|
| `auto-report` | rot (#e11d48) | Automatisch erfasste JS-Fehler |
| `user-feedback` | blau (#2563eb) | Manuell vom Benutzer gesendet |
| `fp:xxxxxxxxxxxxxxxx` | grau (#6b7280) | Fingerprint-Label fuer Deduplizierung |

## Rate-Limiting

| Ebene | Limit | Beschreibung |
|-------|-------|--------------|
| Frontend | 3 Reports / 5 Min | Client-seitig, verhindert Spam bei Error-Loops |
| Backend | 5 Reports / Stunde (Burst: 3) | Server-seitig, IP-basiert |

## API

### `POST /api/error-reports`

Oeffentlicher Endpoint (keine Authentifizierung noetig), rate-limited.

**Request Body** (max 16KB):

```json
{
  "type": "automatic | manual",
  "error_type": "TypeError",
  "message": "Cannot read property 'x' of null",
  "stack": "TypeError: Cannot read...\n  at foo.js:42",
  "fingerprint": "a1b2c3d4e5f67890",
  "url": "/notes/123",
  "component": "Editor.svelte",
  "app_version": "v1.2.3",
  "description": "Nur bei manuellen Reports",
  "steps_to_reproduce": "Nur bei manuellen Reports"
}
```

**Validierung**:

| Feld | Regel |
|------|-------|
| `type` | Muss `automatic` oder `manual` sein |
| `message` | Min. 3 Zeichen (automatic) / 10 Zeichen (manual), max. 500 |
| `fingerprint` | Exakt 16 Hex-Zeichen (lowercase) |
| `stack` | Max. 4000 Zeichen |
| `description` | Max. 2000 Zeichen |
| `steps_to_reproduce` | Max. 2000 Zeichen |
| `url` | Max. 500 Zeichen, muss mit `/` beginnen |
| `error_type` | Max. 50 Zeichen |

**Response**:

```json
{ "accepted": true }
```

## User-Facing Features

### "Issue melden"-Button (Sidebar)

Sichtbar nur wenn `error_reporting_enabled: true` in `/api/config`. Oeffnet den `FeedbackDialog` mit:
- Beschreibungsfeld (min. 10 Zeichen)
- Optionales "Schritte zum Reproduzieren"-Feld
- Datenschutzhinweis

### Opt-out (Settings)

Benutzer koennen automatische Fehlerberichte in den Settings deaktivieren. Die Einstellung wird in `localStorage` gespeichert (`error-reporting-enabled`). Manuelles Feedback ueber den Button bleibt davon unabhaengig moeglich.

## Warum ist der Button in Dev nicht sichtbar?

Die `FORGEJO_*`-Variablen sind typischerweise nur in der Production-`.env` gesetzt. Lokal fehlen sie, daher liefert `/api/config` `error_reporting_enabled: false`.

Um den Button lokal zu sehen, die Variablen exportieren oder in eine lokale `.env` eintragen:

```bash
export FORGEJO_URL=https://<FORGEJO_URL>
export FORGEJO_REPO=xela/xelanote
export FORGEJO_API_TOKEN=gta_dein_token_hier
```

## Referenzen

- Backend Service: `backend/internal/service/errorreport.go`
- Backend Handler: `backend/internal/api/errorreport.go`
- Backend Config: `backend/internal/api/config.go` (`error_reporting_enabled`)
- Frontend Store: `frontend/src/lib/stores/error-reporter.svelte.ts`
- Frontend Dialog: `frontend/src/lib/components/FeedbackDialog.svelte`
- Sidebar-Button: `frontend/src/lib/components/Sidebar.svelte`
- Settings Opt-out: `frontend/src/routes/settings/+page.svelte`
- Env-Variablen: `docs/environment-variables.md`
