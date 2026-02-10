# Cloudflare Turnstile CAPTCHA Integration

xelanote unterstützt optionalen CAPTCHA-Schutz für Login und Registrierung mittels [Cloudflare Turnstile](https://www.cloudflare.com/products/turnstile/).

## Übersicht

- **Zweck**: Schutz gegen Bot-Angriffe bei Login und Registrierung
- **Anbieter**: Cloudflare Turnstile (kostenlos, datenschutzfreundlich)
- **Aktivierung**: Optional via Umgebungsvariablen
- **Ohne Konfiguration**: CAPTCHA ist deaktiviert (ideal für lokale Entwicklung)

## Konfiguration

### Umgebungsvariablen

| Variable | Beschreibung | Erforderlich |
|----------|--------------|--------------|
| `TURNSTILE_SECRET_KEY` | Cloudflare Secret Key (nur Backend) | Ja* |
| `TURNSTILE_SITE_KEY` | Cloudflare Site Key (wird ans Frontend gesendet) | Ja* |
| `TRUSTED_PROXIES` | Komma-getrennte CIDRs für Proxy-Validierung | Nein |

*Beide Keys müssen gesetzt sein, damit CAPTCHA aktiv ist.

### Trusted Proxies

Standardmäßig werden folgende Netzwerke als vertrauenswürdig behandelt:
- `127.0.0.1/32` (localhost IPv4)
- `::1/128` (localhost IPv6)
- `10.0.0.0/8` (Private Klasse A)
- `172.16.0.0/12` (Private Klasse B)
- `192.168.0.0/16` (Private Klasse C)

Für Cloudflare-Proxy zusätzlich setzen:
```bash
TRUSTED_PROXIES=173.245.48.0/20,103.21.244.0/22,103.22.200.0/22,103.31.4.0/22,141.101.64.0/18,108.162.192.0/18,190.93.240.0/20,188.114.96.0/20,197.234.240.0/22,198.41.128.0/17,162.158.0.0/15,104.16.0.0/13,104.24.0.0/14,172.64.0.0/13,131.0.72.0/22
```

## Cloudflare Turnstile einrichten

### 1. Widget erstellen

1. Gehe zu [Cloudflare Dashboard](https://dash.cloudflare.com/) → Turnstile
2. Klicke "Add widget"
3. Konfiguration:
   - **Widget name**: z.B. "xelanote"
   - **Hostname**: Alle Domains hinzufügen, auf denen xelanote läuft
     - Beispiel: `xelanote.com`, `notes.example.de`, `localhost`
   - **Widget Mode**: "Managed" (empfohlen)
   - **Pre-Clearance**: Nein

### 2. Keys kopieren

Nach dem Erstellen erhältst du:
- **Site Key**: Beginnt mit `0x4...` (öffentlich, für Frontend)
- **Secret Key**: Beginnt mit `0x4...` (geheim, nur für Backend)

### 3. Docker-Container starten

```bash
docker run -d --name xelanote \
  -p 8080:8080 \
  -v xelanote-data:/app/data \
  -e JWT_SECRET=dein-jwt-secret \
  -e XELANOTE_DB=/app/data/xelanote.db \
  -e TURNSTILE_SECRET_KEY=0x4AAAA...dein-secret-key \
  -e TURNSTILE_SITE_KEY=0x4AAAA...dein-site-key \
  xelanote:latest
```

## API

### GET /api/config

Gibt die öffentliche Konfiguration zurück (kein Auth erforderlich).

**Response wenn CAPTCHA aktiviert:**
```json
{
  "captcha_enabled": true,
  "captcha_site_key": "0x4AAAAAACNggULpFEdcIsrE",
  "captcha_iframe_url": "/captcha?sitekey=0x4AAAAAACNggULpFEdcIsrE"
}
```

**Response wenn CAPTCHA deaktiviert:**
```json
{
  "captcha_enabled": false
}
```

### POST /api/auth/login

```json
{
  "username_or_email": "user@example.com",
  "password": "geheim123",
  "captcha_token": "0.token-von-turnstile..."
}
```

### POST /api/auth/register

```json
{
  "username": "neuuser",
  "email": "neu@example.com",
  "password": "geheim123",
  "captcha_token": "0.token-von-turnstile..."
}
```

**Fehler bei fehlendem/ungültigem Token:**
```json
{
  "error": "captcha token required"
}
```

## Architektur

### Backend

```
backend/
├── internal/
│   ├── service/
│   │   └── captcha.go          # TurnstileService
│   └── api/
│       ├── config.go           # /api/config Endpoint (inkl. captcha_iframe_url)
│       ├── auth.go             # Login/Register mit CAPTCHA
│       ├── captcha_page.go     # GET /captcha Handler (iframe-Seite)
│       ├── security.go         # Globale Security-Headers
│       ├── middleware.go       # Trusted Proxy Validierung
│       └── static/
│           └── captcha.html    # Eingebettete CAPTCHA-Seite für iframe
└── cmd/server/
    └── main.go                 # ENV-Variablen laden
```

### Frontend

```
frontend/src/
├── lib/
│   ├── api.ts                  # getConfig(), login(), register()
│   ├── config.ts               # getServerUrl(), isDesktop()
│   ├── components/
│   │   └── CaptchaIframe.svelte # iframe-basierte CAPTCHA-Komponente (Desktop)
│   └── stores/auth.svelte.ts   # Auth-Store mit CAPTCHA-Token
└── routes/
    ├── login/+page.svelte      # Login mit Turnstile Widget / CaptchaIframe
    └── register/+page.svelte   # Register mit Turnstile Widget / CaptchaIframe
```

### Ablauf

```
┌─────────┐     ┌─────────┐     ┌──────────┐     ┌────────────┐
│ Browser │     │ Backend │     │Cloudflare│     │  Turnstile │
└────┬────┘     └────┬────┘     │  CDN     │     │    API     │
     │               │          └────┬─────┘     └─────┬──────┘
     │ GET /api/config              │                  │
     │──────────────>│               │                  │
     │ {captcha_enabled, site_key}  │                  │
     │<──────────────│               │                  │
     │               │               │                  │
     │ Load turnstile.js            │                  │
     │──────────────────────────────>│                  │
     │ <script>                      │                  │
     │<──────────────────────────────│                  │
     │               │               │                  │
     │ User löst Challenge          │                  │
     │ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ >│                  │
     │ Token                         │                  │
     │<─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│                  │
     │               │               │                  │
     │ POST /api/auth/login          │                  │
     │ {credentials, captcha_token}  │                  │
     │──────────────>│               │                  │
     │               │ POST /siteverify                 │
     │               │ {secret, token}                  │
     │               │─────────────────────────────────>│
     │               │ {success: true}                  │
     │               │<─────────────────────────────────│
     │               │               │                  │
     │ Login erfolgreich             │                  │
     │<──────────────│               │                  │
```

## Fehlerbehebung

### CAPTCHA Widget wird nicht angezeigt

1. **Domain prüfen**: Ist die aktuelle Domain im Turnstile Widget konfiguriert?
   - Cloudflare Dashboard → Turnstile → Widget bearbeiten → Hostnames
   - Auch Subdomains müssen explizit hinzugefügt werden

2. **Browser-Konsole prüfen**: Gibt es Fehler beim Laden von `challenges.cloudflare.com`?

3. **Config-Endpoint prüfen**:
   ```bash
   curl https://deine-domain.de/api/config
   ```
   Sollte `captcha_enabled: true` und den Site Key zeigen.

4. **Adblocker**: Manche Adblocker blockieren Turnstile. Zum Testen deaktivieren.

### "captcha token required" Fehler

- CAPTCHA ist aktiviert, aber kein Token wurde gesendet
- Widget wurde nicht gerendert oder User hat Challenge nicht gelöst

### "captcha verification failed: invalid token"

- Token ist ungültig oder abgelaufen
- Falscher Secret Key konfiguriert

### "captcha verification failed: server configuration error"

- `TURNSTILE_SECRET_KEY` ist falsch oder nicht gesetzt

## Lokale Entwicklung

Für lokale Entwicklung ohne CAPTCHA einfach die Umgebungsvariablen weglassen:

```bash
# Ohne CAPTCHA (Entwicklung)
JWT_SECRET=$(openssl rand -hex 32) ./xelanote

# Mit CAPTCHA (Test)
TURNSTILE_SECRET_KEY=xxx TURNSTILE_SITE_KEY=yyy JWT_SECRET=$(openssl rand -hex 32) ./xelanote
```

Cloudflare bietet auch Test-Keys für die Entwicklung:
- **Site Key (always passes)**: `1x00000000000000000000AA`
- **Secret Key (always passes)**: `1x0000000000000000000000000000000AA`

## Content Security Policy (CSP)

Die CSP in `backend/internal/api/security.go` muss Cloudflare Turnstile erlauben:

```go
// Erforderliche CSP-Direktiven für Turnstile:
"script-src 'self' 'unsafe-inline' https://challenges.cloudflare.com; " +
"connect-src 'self' ws: wss: https://challenges.cloudflare.com; " +
"frame-src https://challenges.cloudflare.com; "
```

Ohne diese Einträge wird das Turnstile-Script vom Browser blockiert und das Widget erscheint nicht.

## Desktop App Integration (Electron/Tauri)

### Problem

Cloudflare Turnstile akzeptiert nur `http://` oder `https://` Origins. Desktop-Apps (Electron mit `app://`, Tauri mit `tauri://`) werden bei der serverseitigen Token-Verifikation abgelehnt.

### Lösung: iframe-basierter Ansatz

Die Desktop-App lädt die CAPTCHA-Challenge in einem iframe, der auf den Backend-Server zeigt:

1. **Backend** stellt `GET /captcha?sitekey=...` bereit -- eine minimale HTML-Seite mit dem Turnstile-Widget
2. **Desktop-App** bettet diese Seite als iframe ein (`CaptchaIframe.svelte`)
3. **Token-Austausch** erfolgt via `postMessage` vom iframe zum Parent
4. **Origin** des iframes ist `https://xelanote.com` (oder der konfigurierte Server) -- Turnstile akzeptiert dies

```
┌──────────────────────────────────────────────┐
│ Desktop App (Electron/Tauri)                 │
│                                              │
│  ┌────────────────────────────────────────┐  │
│  │ Login-Seite                            │  │
│  │                                        │  │
│  │  ┌──────────────────────────────────┐  │  │
│  │  │ iframe: /captcha?sitekey=...     │  │  │
│  │  │ (Origin: https://xelanote.com)   │  │  │
│  │  │                                  │  │  │
│  │  │  [Turnstile Widget]              │  │  │
│  │  │                                  │  │  │
│  │  └──────────────────────────────────┘  │  │
│  │         │ postMessage({token})         │  │
│  │         ▼                              │  │
│  │  captchaToken = token                  │  │
│  └────────────────────────────────────────┘  │
└──────────────────────────────────────────────┘
```

### CSP-Anforderungen

**Electron** (`frontend/src-electron/main.ts`):
```
frame-src https: http://localhost:*
```

**Tauri** (`frontend/src-tauri/tauri.conf.json`):
```
frame-src https: http://localhost:*;
```

**Backend** (`/captcha` Endpunkt hat eigene CSP):
```
frame-ancestors *
```
(Der Rest der App behält `frame-ancestors 'none'`)

### Fallback-Verhalten

Desktop-Clients ohne CAPTCHA-Token (z.B. offline oder iframe-Fehler) erhalten einen Fallback-Bypass. Das Backend erkennt Desktop-Clients via `X-Client-Type: desktop` Header + localhost-Check.

### Multi-Server-Support

Jeder Server stellt seinen eigenen `/captcha`-Endpunkt bereit. Die Desktop-App kombiniert die Server-URL mit der relativen `captcha_iframe_url` aus `/api/config`:

```
getServerUrl() + config.captcha_iframe_url
// z.B. "https://xelanote.com" + "/captcha?sitekey=0x4AAAA..."
```

### Troubleshooting

**iframe rendert nicht:**
- CSP der Desktop-App prüfen (`frame-src` muss HTTPS erlauben)
- Server-URL in der Desktop-App korrekt konfiguriert?
- Backend erreichbar? `curl {server}/captcha?sitekey=test`

**Token wird nicht empfangen:**
- Browser-Konsole auf postMessage-Fehler prüfen
- Origin-Validierung in `CaptchaIframe.svelte` prüft gegen Server-URL

**Turnstile Widget zeigt Fehler:**
- Server-Domain in Cloudflare Turnstile Widget konfiguriert?
- Adblocker deaktivieren

## Sicherheitshinweise

1. **Secret Key geheim halten**: Niemals im Frontend oder in Git committen
2. **HTTPS verwenden**: Turnstile funktioniert auch über HTTP, aber HTTPS ist empfohlen
3. **Rate Limiting**: Turnstile ersetzt kein serverseitiges Rate Limiting
4. **Trusted Proxies konfigurieren**: Ohne korrekte Konfiguration wird die falsche Client-IP verifiziert
