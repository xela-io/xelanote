# Authentifizierung und Sicherheit

## Auth-Flow Überblick

```
┌─────────────────────────────────────────────────────────┐
│                    LOGIN-FLOW                           │
│                                                         │
│  Username + Passwort                                    │
│       ↓                                                 │
│  POST /api/auth/login                                   │
│       ↓                                                 │
│  Server: bcrypt.Compare(password, hash)                 │
│       ↓                                                 │
│  2FA aktiv? ──── Ja → {requiresTwoFactor: true,        │
│       │                 methods: ["totp","fido2"]}       │
│       │                     ↓                           │
│       │              User gibt TOTP-Code ein            │
│       │              ODER Passkey-Challenge              │
│       │                     ↓                           │
│       │              POST /api/auth/verify-2fa           │
│       ↓                     ↓                           │
│  Tokens ausstellen:                                     │
│   - Access Token (JWT, 15min) → HttpOnly Cookie         │
│   - Refresh Token (random, 30d) → HttpOnly Cookie       │
│   - CSRF Token → Normales Cookie (JS-lesbar)            │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Token-Architektur

### Access Token (JWT)

```
Header:  {"alg": "HS256"}
Payload: {"sub": "user-uuid", "exp": 1234567890, "iat": ...}
Signed:  HMAC-SHA256 mit JWT_SECRET (mind. 64 Zeichen)
```

- **TTL:** 15 Minuten
- **Speicherort:** HttpOnly Cookie (nie in localStorage!)
- **Desktop/Tauri:** OS Keyring

### Refresh Token

```
Token:    30 zufällige Bytes → Base64
Speicher: SHA-256-Hash in DB (Klartext wird nie gespeichert)
TTL:      30 Tage
Family:   Jedes Token gehört zu einer "Familie"
```

### Token-Rotation

```
1. Client: POST /api/auth/refresh (mit Refresh-Cookie)
2. Server:
   a. Hash des Tokens berechnen
   b. In DB nachschlagen
   c. Altes Token löschen
   d. Neues Access + Refresh Token generieren
   e. Neues Refresh Token gehört zur gleichen Familie
3. Client erhält neue Cookies
```

### Replay-Detection (Token-Reuse)

```
Angreifer stiehlt Refresh Token und nutzt es:

Legitimate User:   Token A (Familie X)
                        ↓ refresh
                   Token B (Familie X)   ← Token A ungültig

Angreifer:         Token A (Familie X)   ← Versucht zu nutzen
                        ↓
Server erkennt: Token A war schon rotiert!
                        ↓
ALLE Tokens der Familie X werden widerrufen
                        ↓
Legitimate User muss sich neu einloggen
(aber Angreifer hat auch keinen Zugriff mehr)
```

## Zweifaktor-Authentifizierung (2FA)

### TOTP (Time-based One-Time Password)

```
Setup:
1. POST /api/2fa/setup → Server generiert Secret + QR-Code
2. User scannt QR-Code mit Authenticator-App
3. POST /api/2fa/verify (mit 6-stelligem Code) → 2FA aktiviert
4. Server gibt Backup-Codes zurück (10 Stück, einmalig verwendbar)
```

### FIDO2 / WebAuthn (Passkeys)

```
Registrierung:
1. POST /api/2fa/fido2/register/begin
   → Server erstellt Challenge
2. Browser: navigator.credentials.create(challenge)
   → User berührt Security Key / nutzt Fingerabdruck
3. POST /api/2fa/fido2/register/finish
   → Server speichert Public Key

Login:
1. POST /api/auth/fido2/begin
   → Server sendet Challenge
2. Browser: navigator.credentials.get(challenge)
   → User authentifiziert sich
3. POST /api/auth/fido2/finish
   → Server verifiziert Signatur → Tokens ausstellen
```

### Backup-Codes

- 10 einmalig verwendbare Codes
- Gehasht in DB gespeichert
- Können regeneriert werden (alte werden ungültig)
- Letzte Rettung wenn Authenticator-App verloren

## CSRF-Schutz

**Double-Submit Cookie Pattern:**

```
1. Server setzt Cookie: csrf_token=<random> (HttpOnly=false)
2. JavaScript liest Cookie-Wert
3. Bei jedem POST/PUT/DELETE:
   Header: X-CSRF-Token: <gleicher Wert>
4. Server vergleicht Cookie-Wert mit Header-Wert
```

**Warum funktioniert das?**
- Ein Angreifer auf einer anderen Domain kann den Cookie-Wert nicht lesen
- Er kann also den X-CSRF-Token Header nicht setzen
- Same-Origin Policy verhindert das Lesen fremder Cookies

**Ausnahme:** Requests mit `Authorization: Bearer` Header (Desktop-App, API-Clients) brauchen kein CSRF, weil Bearer-Tokens nicht automatisch vom Browser gesendet werden.

## Rate-Limiting

Jeder sensible Endpoint hat ein eigenes Rate-Limit:

| Endpoint | Limit | Zweck |
|----------|-------|-------|
| `/auth/login` | Strikt | Brute-Force-Schutz |
| `/auth/register` | Sehr strikt | Spam-Registrierungen |
| `/api/search` | Moderat | DB-Last begrenzen |
| `/api/llm/*` | Moderat | API-Kosten begrenzen |
| `/api/notes/*/summarize` | Moderat | LLM-Kosten |
| `/api/error-reports` | Moderat | Spam verhindern |

Rate-Limiter sind **per-Endpoint** implementiert (nicht global), damit ein API-Heavy-Feature nicht andere blockiert.

## Account-Lockout

```
Falsches Passwort
    ↓
failed_attempts++ in account_lockouts
    ↓
>= N Versuche? → Account gesperrt bis locked_until
    ↓
Warten oder Admin-Intervention nötig
```

Persistenter Lockout (in SQLite, überlebt Server-Neustart).

## Registrierung

- **Erster User** wird automatisch Admin
- Weitere Registrierungen nur wenn Admin `registration_enabled` auf `true` setzt
- Optional: Cloudflare Turnstile CAPTCHA

## Recovery (Passwort vergessen)

Kein E-Mail-basierter Reset (Privacy-first). Stattdessen:

1. Bei Account-Erstellung generiert der Client einen **Recovery Key**
2. Server speichert nur den **Salt** für die Key-Derivation
3. Wenn User Passwort vergisst:
   - Recovery Key eingeben
   - Server verifiziert
   - Neues Passwort setzen
   - Verschlüsselungs-KEK wird mit neuem Passwort neu verschlüsselt

## Security Headers

Jede Response enthält:

```
Content-Security-Policy: ...     (XSS-Schutz)
X-Frame-Options: DENY            (Clickjacking-Schutz)
X-Content-Type-Options: nosniff
Strict-Transport-Security: ...   (HSTS)
Referrer-Policy: strict-origin-when-cross-origin
```

## Weitere Sicherheitsmaßnahmen

| Maßnahme | Umsetzung |
|----------|-----------|
| Kein localStorage für Auth | Tokens nur in HttpOnly Cookies |
| Fehler-Obfuskation | Interne Fehlerdetails nie an Client ("Login failed" statt "User not found") |
| Upload-Schutz | Uploads brauchen Auth + Owner-Check |
| API-Key-Verschlüsselung | LLM-Keys AES-256-GCM verschlüsselt in DB |
| Trusted Proxy | Explizite CIDR-Whitelist für X-Forwarded-For |
| Panic Recovery | Panics werden gefangen, als Forgejo-Issue gemeldet |
| bcrypt Timing | Constant-time Vergleich (verhindert Timing-Attacks) |

## Nächste Seiten

- [Verschlüsselung](Verschlüsselung.md) — E2E-Encryption im Detail
- [Backend](Backend.md) — Middleware-Stack
