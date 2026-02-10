# Authentication System

Xelanote verwendet JWT-basierte Authentication mit Refresh Token Rotation für maximale Sicherheit.

## Übersicht

### User Interface

#### Security Badges (Login Page)

Die Login-Seite zeigt drei Security Badges im Shields.io-Stil, um Nutzern die Sicherheitsfeatures transparent zu kommunizieren:

1. **End-to-End Encrypted** - AES-256-GCM (mit Lock-Icon)
2. **Zero-Knowledge** - Keys stay local (mit Key-Icon)
3. **Open Source** - MIT License (mit Code-Icon)

Diese Badges sind vollständig internationalisiert (Deutsch/Englisch) und dienen dazu, Vertrauen aufzubauen, indem die wichtigsten Sicherheitsmerkmale bereits vor dem Login sichtbar sind.

**Datei**: `frontend/src/routes/login/+page.svelte`

### Token Types

1. **Access Token (JWT)**
   - Lebensdauer: 15 Minuten
   - Format: JWT mit HMAC-SHA256 Signatur
   - Enthält: user_id, username, issuer, expiration
   - Verwendung: Authorization Header bei jedem API Request

2. **Refresh Token**
   - Lebensdauer: 30 Tage
   - Format: Kryptographisch sicherer Random String (32 bytes, base64)
   - Speicherort: Database (refresh_tokens table)
   - Verwendung: Erneuert abgelaufene Access Tokens

### Sicherheits-Features

- ✅ **Token Rotation**: Bei Refresh werden alte Refresh Tokens ungültig
- ✅ **Bcrypt Password Hashing**: Cost Factor 12
- ✅ **Short-lived Access Tokens**: Reduziert Risiko bei Kompromittierung
- ✅ **Database-backed Refresh Tokens**: Ermöglicht Revocation (Logout)
- ✅ **Multi-User Data Isolation**: user_id Filter in allen Queries
- ✅ **HttpOnly Cookies**: Schutz vor XSS-Angriffen
- ✅ **SameSite Protection**: CSRF-Schutz durch SameSite Cookie-Attribut
- ✅ **CSRF Token**: Double-Submit Cookie Pattern mit robuster Multi-Cookie-Validierung
- ✅ **Dual Authentication**: Unterstützt Header-Auth UND Cookie-Auth

### Cookie-basierte Authentifizierung

Zusätzlich zur Header-basierten Authentifizierung unterstützt Xelanote jetzt HttpOnly-Cookies für verbesserte Sicherheit und bessere Browser-Kompatibilität.

#### Vorteile

1. **XSS-Schutz**: HttpOnly-Cookies sind für JavaScript nicht zugänglich
2. **Image-Loading**: Browser senden Cookies automatisch bei `<img src="/api/uploads/...">` Requests
3. **Rückwärtskompatibilität**: Alte Clients mit Header-Auth funktionieren weiterhin
4. **CSRF-Schutz**: SameSite-Attribut verhindert Cross-Site Request Forgery

#### Cookie-Konfiguration

Cookies werden automatisch bei Login/Register gesetzt und bei Refresh aktualisiert:

**Development Mode** (`XELANOTE_ENV=development`):
```
HttpOnly: true
Secure: false        // HTTP für localhost
SameSite: Strict    // Maximale CSRF-Sicherheit (seit SEC-L04)
Path: /api
Max-Age: 900s (access), 2592000s (refresh)
```

**Production Mode** (`XELANOTE_ENV=production`):
```
HttpOnly: true
Secure: true        // HTTPS only
SameSite: Strict   // Maximale Sicherheit
Path: /api
Max-Age: 900s (access), 2592000s (refresh)
```

**Hinweis:** Seit SEC-L04 (2026-01-28) wird `SameSite=Strict` für alle Auth-Cookies in beiden Modi verwendet.

#### Cookie-Namen

- `access_token` - JWT Access Token (15 Minuten)
- `refresh_token` - Refresh Token (30 Tage)

#### Middleware-Verhalten

Die Auth-Middleware akzeptiert beide Authentifizierungsmethoden:

1. **Priorität 1**: Authorization Header `Bearer <token>`
2. **Fallback**: Cookie `access_token`

Wenn beide vorhanden sind, wird der Header verwendet.

#### Frontend-Standard (aktueller Client)

**Seit SEC-006 (2026-01):** Der SvelteKit-Client speichert Tokens **ausschließlich** in HttpOnly-Cookies. `localStorage` wird für Auth-Tokens nicht mehr verwendet (XSS-Schutz). Desktop-Apps nutzen den OS Keyring.

Cookies werden automatisch bei allen `/api/*` Requests gesendet, inkl. `<img src="/api/uploads/...">` für Bilder.

## Backend Implementation

### Architektur

```
Client Request
    ↓
Auth Middleware (validates JWT)
    ↓
Context enrichment (adds user_id)
    ↓
Handler (extracts user_id from context)
    ↓
Service Layer (passes user_id)
    ↓
Database Layer (filters by user_id)
```

### Wichtige Dateien

#### 1. JWT Module (`internal/auth/jwt.go`)

```go
// Token Generation
GenerateAccessToken(userID int, username string, secret []byte) (string, error)
GenerateRefreshToken() (string, error)

// Token Validation
ValidateAccessToken(tokenString string, secret []byte) (*Claims, error)
```

#### 2. Auth Service (`internal/service/auth.go`)

```go
// User Registration
Register(ctx context.Context, username, email, password string) (*db.User, error)

// User Login
Login(ctx context.Context, usernameOrEmail, password string) (accessToken, refreshToken string, err error)

// Token Refresh
RefreshAccessToken(ctx context.Context, refreshToken string) (newAccessToken, newRefreshToken string, err error)

// Logout
Logout(ctx context.Context, refreshToken string) error
```

#### 3. Auth Middleware (`internal/api/middleware.go`)

```go
// Validates JWT and adds user_id to context
func (s *Server) authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Extract Authorization header
        // 2. Validate JWT
        // 3. Add user_id to context
        // 4. Call next handler
    })
}

// Helper to extract user_id from context
func getUserID(r *http.Request) (int, bool)
```

### API Endpoints

#### Register User
```http
POST /api/auth/register
Content-Type: application/json

{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "secure_password_123"
}

Response 201:
Set-Cookie: access_token=eyJhbGc...; Path=/api; Max-Age=900; HttpOnly; SameSite=Lax
Set-Cookie: refresh_token=wJYNqUI...; Path=/api; Max-Age=2592000; HttpOnly; SameSite=Lax

{
  "access_token": "eyJhbGc...",
  "refresh_token": "wJYNqUI...",
  "user": {
    "id": 1,
    "username": "johndoe",
    "email": "john@example.com"
  }
}
```

#### Login
```http
POST /api/auth/login
Content-Type: application/json

{
  "username_or_email": "johndoe",
  "password": "secure_password_123"
}

Response 200:
Set-Cookie: access_token=eyJhbGc...; Path=/api; Max-Age=900; HttpOnly; SameSite=Lax
Set-Cookie: refresh_token=wJYNqUI...; Path=/api; Max-Age=2592000; HttpOnly; SameSite=Lax

{
  "access_token": "eyJhbGc...",
  "refresh_token": "wJYNqUI...",
  "user": {
    "id": 1,
    "username": "johndoe",
    "email": "john@example.com"
  }
}
```

#### Refresh Token
```http
POST /api/auth/refresh
Content-Type: application/json
Cookie: refresh_token=wJYNqUI...  // Optional - Cookie hat Priorität

{
  "refresh_token": "wJYNqUI..."  // Optional - Fallback wenn kein Cookie
}

Response 200:
Set-Cookie: access_token=eyJhbGc...; Path=/api; Max-Age=900; HttpOnly; SameSite=Lax
Set-Cookie: refresh_token=JhjQcVc...; Path=/api; Max-Age=2592000; HttpOnly; SameSite=Lax

{
  "access_token": "eyJhbGc...",
  "refresh_token": "JhjQcVc..."  // NEW token (old one invalidated)
}
```

**SEC-006 Kompatibilität (Cookie-First Refresh):**

Nach dem SEC-006 Security Audit werden Tokens ausschließlich in HttpOnly Cookies gespeichert, nicht mehr in localStorage/sessionStorage. Der Refresh-Endpoint liest den Refresh Token primär aus dem Cookie (`getRefreshTokenFromCookie`) und akzeptiert Body-Parameter nur als Fallback für Legacy-Clients.

Das proaktive Token-Refresh-System (`frontend/src/lib/stores/token-refresh.svelte.ts`) nutzt die neue `refreshTokenViaCookie()` Funktion:

```typescript
// SEC-006: Cookie-basierter Refresh ohne Body-Parameter
const tokens = await api.refreshTokenViaCookie();
// credentials: 'include' sendet refresh_token Cookie automatisch
```

**Wichtig nach Page Reload:**
- Access Token wird aus localStorage gelesen (für Authorization Header)
- Refresh Token ist NICHT mehr in localStorage → nur im HttpOnly Cookie
- Proaktiver Refresh funktioniert trotzdem, da Cookie automatisch mitgesendet wird
- Backend priorisiert Cookie-Extraktion (Zeile 354 in `backend/internal/api/auth.go`)

#### Logout
```http
POST /api/auth/logout
Content-Type: application/json
Cookie: refresh_token=wJYNqUI...  // Optional - Cookie hat Priorität

{
  "refresh_token": "wJYNqUI..."  // Optional - Fallback wenn kein Cookie
}

Response 204 No Content
Set-Cookie: access_token=; Path=/api; Max-Age=0; HttpOnly; SameSite=Lax
Set-Cookie: refresh_token=; Path=/api; Max-Age=0; HttpOnly; SameSite=Lax
```

#### Get Current User
```http
GET /api/auth/me
Authorization: Bearer eyJhbGc...  // Option 1: Header
Cookie: access_token=eyJhbGc...   // Option 2: Cookie (Fallback)

Response 200:
{
  "id": 1,
  "username": "johndoe",
  "email": "john@example.com"
}
```

### Protected Endpoints

Alle anderen Endpoints erfordern JWT Authentication via Header ODER Cookie:

**Option 1: Authorization Header (bevorzugt)**
```http
GET /api/notes
Authorization: Bearer eyJhbGc...
```

**Option 2: Cookie (Fallback)**
```http
GET /api/notes
Cookie: access_token=eyJhbGc...
```

Bei fehlendem oder ungültigem Token:
```http
Response 401:
{
  "error": "invalid or expired token"
}
```

## Frontend Implementation

### Store Architecture

```typescript
// lib/stores/auth.svelte.ts
export function login(usernameOrEmail: string, password: string): Promise<void>
export function register(username: string, email: string, password: string): Promise<void>
export function logoutAsync(): Promise<void>
export function initAuth(): void  // Called on app startup
export function isAuthenticated(): boolean
export function getCurrentUser(): User | null
```

### API Client Integration

```typescript
// lib/api.ts
async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
    // 1. Add Authorization header if authenticated
    const accessToken = getAccessToken?.();
    if (accessToken) {
        headers['Authorization'] = `Bearer ${accessToken}`;
    }

    const response = await fetch(`${API_BASE}${path}`, { ...options, headers });

    // 2. Handle 401 with automatic token refresh
    if (response.status === 401 && accessToken) {
        const refreshToken = getRefreshToken?.();
        if (refreshToken) {
            // Refresh token
            const tokens = await refreshTokens(refreshToken);
            updateTokens(tokens.access_token, tokens.refresh_token);

            // Retry original request with new token
            return request<T>(path, options, false);  // prevent infinite loop
        }
    }

    return response.json();
}
```

### Protected Routes

```typescript
// routes/+layout.svelte
$effect(() => {
    const currentPath = $page.url.pathname;
    const isPublicRoute = ['/login', '/register'].some(r => currentPath.startsWith(r));
    const isAuth = auth.isAuthenticated();

    // Redirect to login if not authenticated and not on public route
    if (!isAuth && !isPublicRoute) {
        goto('/login');
    }

    // Redirect to home if authenticated and on login/register page
    if (isAuth && isPublicRoute) {
        goto('/');
    }
});
```

### LocalStorage Keys

```typescript
const ACCESS_TOKEN_KEY = 'xelanote_access_token';
const REFRESH_TOKEN_KEY = 'xelanote_refresh_token';
```

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);
```

### Refresh Tokens Table
```sql
CREATE TABLE refresh_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token TEXT UNIQUE NOT NULL,
    user_id INTEGER NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### User Ownership (Migration 005)
```sql
ALTER TABLE notes ADD COLUMN user_id INTEGER;
ALTER TABLE folders ADD COLUMN user_id INTEGER;

CREATE INDEX idx_notes_user_id ON notes(user_id);
CREATE INDEX idx_folders_user_id ON folders(user_id);
```

## Konfiguration

### Environment Variables

```bash
# Required - Must be at least 32 characters
JWT_SECRET=your-super-secret-jwt-key-min-32-characters

# Environment mode (affects cookie security)
XELANOTE_ENV=production  # or 'development'

# Generate secure secret:
openssl rand -base64 32
```

### .env.example

```bash
# JWT Secret (min 32 characters, cryptographically secure)
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production

# Environment mode (development or production)
# Affects cookie security: Secure flag, SameSite policy
XELANOTE_ENV=development

# Database
XELANOTE_DB=./data/xelanote.db
XELANOTE_DB_KEY_FILE=./secrets/db_key  # optional SQLCipher key

# Server
PORT=8080
```

## Testing

### Manual Testing

#### 1. Register User
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }'
```

#### 2. Login
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username_or_email": "testuser",
    "password": "password123"
  }'
```

Save the `access_token` from response.

#### 3. Access Protected Endpoint
```bash
curl http://localhost:8080/api/notes \
  -H "Authorization: Bearer <access_token>"
```

#### 4. Refresh Token
```bash
curl -X POST http://localhost:8080/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "<refresh_token>"
  }'
```

#### 5. Logout
```bash
curl -X POST http://localhost:8080/api/auth/logout \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "<refresh_token>"
  }'
```

### Multi-User Isolation Test

```bash
# 1. Register Alice
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "email": "alice@test.com", "password": "password123"}'

# Save Alice's access_token

# 2. Create note as Alice
curl -X POST http://localhost:8080/api/notes \
  -H "Authorization: Bearer <alice_token>" \
  -H "Content-Type: application/json" \
  -d '{"title": "Alice Secret", "content": "Only Alice can see this"}'

# 3. Register Bob
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "bob", "email": "bob@test.com", "password": "password123"}'

# Save Bob's access_token

# 4. List notes as Bob (should be empty)
curl http://localhost:8080/api/notes \
  -H "Authorization: Bearer <bob_token>"

# Expected: Empty list (Bob cannot see Alice's notes)
```

## Troubleshooting

### "JWT_SECRET environment variable is required"
**Problem:** Server startet nicht
**Lösung:** `export JWT_SECRET="$(openssl rand -base64 32)"` vor Server-Start

### "invalid or expired token"
**Problem:** 401 bei API Requests
**Lösung:**
1. Check if access_token is in Authorization header
2. Token might be expired - try refresh
3. Check JWT_SECRET is consistent across restarts

### "not enough args to execute query"
**Problem:** SQL-Fehler bei Operations
**Lösung:** Bug in userID parameter - bereits gefixt in v0.4.0

### "Note not found" bei Drag & Drop
**Problem:** Frontend store synchronization
**Lösung:**
1. Check notes store limit (should be 1000, not 100)
2. Reload page (F5)
3. Bereits gefixt in v0.4.0

## Security Best Practices

### Production Deployment

1. **JWT Secret**
   - Generate with `openssl rand -base64 32`
   - Store in environment variable (NOT in code)
   - Rotate periodically (invalidates all tokens)

2. **CORS Configuration**
   - Set specific allowed origins (not "*")
   - Credentials enabled for cookie-based auth (✅ implementiert)

3. **HTTPS**
   - Always use HTTPS in production
   - Tokens transmitted in plaintext over HTTP are vulnerable

4. **Password Policy**
   - Current: Minimum 8 characters
   - Consider: Password complexity checker
   - Consider: Rate limiting on login attempts

5. **Token Expiration**
   - Access: 15 minutes (short window if compromised)
   - Refresh: 30 days (balance between security and UX)
   - Consider: Configurable per deployment

6. **Session Management**
   - Implement session list UI
   - Allow users to revoke individual sessions
   - Log session activity

## Proaktiver Token-Refresh

### Übersicht

Access Tokens laufen nach 15 Minuten ab. Um 401-Fehler bei aktiver Nutzung zu vermeiden, implementiert xelanote einen **proaktiven Token-Refresh-Mechanismus**, der Tokens automatisch vor Ablauf erneuert.

**Architektur:** Hybrid-Ansatz (proaktiv + reaktiv als Fallback)

### Funktionsweise

```
┌─────────────────────────────────────────────────────────────────┐
│                    Token Timeline (15 Min)                       │
├─────────────────────────────────────────────────────────────────┤
│ 0 Min        │ 12 Min (80%)        │ 15 Min                     │
│ Token issued │ Proaktiver Refresh  │ Token expired              │
│              │ ↓                    │                            │
│              │ Neuer Token         │                            │
└─────────────────────────────────────────────────────────────────┘
```

### Komponenten

**1. JWT Expiry Tracking (`auth.svelte.ts`)**
- `parseJWTClaims()` - Extrahiert `exp` und `iat` aus Access Token
- `addTokenUpdateListener()` - Event-basiertes Subscriber-Pattern
- `getTokenExpiry()` / `getTokenIssuedAt()` - Getter für Timer-Berechnung
- SessionStorage Persistence für Page-Reload (`xelanote_token_exp`, `xelanote_token_iat`)

**2. Token-Refresh Store (`token-refresh.svelte.ts`)**
- Timer-Management nach `auto-lock.svelte.ts` Pattern
- Refresh bei 80% Token-Lebensdauer (default: 12 Min bei 15 Min Token)
- Idle-Detection: Pausiert nach 10 Min Inaktivität
- Visibility-Detection: Pausiert bei hidden Tab
- Sofortiger Refresh bei Tab-Rückkehr wenn überfällig

**3. Layout-Integration (`+layout.svelte`)**
- Listener für Token-Updates (Login + Refresh)
- Activity-Tracking für Idle-Detection
- Cleanup bei Logout/Unmount

### SEC-006 Kompatibilität

Nach der SEC-006 Migration (HttpOnly Cookies) ist der Refresh Token nicht mehr in JavaScript lesbar. Der proaktive Refresh nutzt daher:

```typescript
// Kein Body nötig - Cookie wird automatisch mitgesendet
const tokens = await api.refreshTokenViaCookie();
```

**Token-Expiry Persistence:**
- JWT-Timestamps (`exp`, `iat`) werden in sessionStorage gespeichert
- Nicht sensitiv - nur Zeitstempel, keine Tokens
- Ermöglicht Timer-Wiederherstellung nach Page Reload

### Verhalten nach Szenarien

| Szenario | Verhalten |
|----------|-----------|
| Aktive Nutzung | Token wird alle ~12 Min automatisch erneuert |
| Tab offen, idle >10 Min | Refresh pausiert. Bei Aktivität: sofortiger Refresh |
| Tab im Hintergrund | Refresh pausiert. Bei Rückkehr: sofortiger Refresh falls nötig |
| Page Reload (F5) | Timer startet neu aus sessionStorage-Timestamps |
| Logout | Timer gestoppt, Timestamps gelöscht |

### Logs (Console)

```
[Auth] Token timestamps updated
[Layout] Token updated, re-initializing token-refresh
[TokenRefresh] Initialized. Token expires at 14:15:00, refresh at 14:03:00
[TokenRefresh] Timer scheduled in 720s
...
[TokenRefresh] Token refreshed proactively
```

### Tests

Unit-Tests in `frontend/src/lib/stores/token-refresh.test.ts`:
- Timer-Initialisierung mit gültiger/ungültiger Expiry
- RefreshAt-Berechnung bei 80% Lebensdauer
- Fallback-Buffer wenn `iat` fehlt
- Idle-Pause-Detection
- Stop cleared alle State

## CSRF-Schutz

### Uebersicht

Xelanote implementiert CSRF-Schutz durch eine Kombination aus **SameSite=Strict Cookies** und dem **Double-Submit Cookie Pattern**.

### Schutzmechanismen

1. **SameSite=Strict**: Alle Auth-Cookies (access_token, refresh_token, csrf_token) werden nur bei Same-Origin-Requests gesendet
2. **Double-Submit Cookie**: CSRF-Token wird als Cookie UND Header gesendet, Server validiert Match

### Cookie-Konfiguration (CSRF)

```
Name: csrf_token
Path: /                  // Wichtig: "/" statt "/api" fuer JS-Zugriff
HttpOnly: false          // JavaScript muss Token lesen koennen
Secure: true (prod)      // HTTPS-only in Production
SameSite: Strict         // Maximaler CSRF-Schutz
Max-Age: 86400           // 24 Stunden
```

**Wichtig:** Der Cookie-Path muss `/` sein, damit JavaScript von allen Seiten (`/note/...`, `/settings/...`, etc.) das Token lesen kann. Ein Path von `/api` wuerde den Zugriff auf API-Seiten beschraenken.

### Request-Flow

```
1. Login/Register → Server setzt csrf_token Cookie
2. Client liest Cookie, speichert Token
3. Bei State-Changing Request:
   - Cookie wird automatisch mitgesendet (SameSite=Strict)
   - Client sendet Token auch als X-CSRF-Token Header
4. Server validiert: Cookie-Token == Header-Token
```

### Robuste Multi-Cookie-Validierung

Nach der SEC-L04 Migration (SameSite=Strict) koennen temporaer mehrere Cookies mit Namen `csrf_token` existieren (alter mit Path=/api, neuer mit Path=/). Der Server behandelt dies robust:

```go
// Server prueft ALLE csrf_token Cookies
for _, cookie := range parsedCookies {
    if cookie.Name == "csrf_token" {
        if subtle.ConstantTimeCompare(cookie.Value, headerToken) == 1 {
            // Match gefunden → Request valide
            return true
        }
    }
}
```

**Telemetrie:** Bei mehreren Cookies wird dies geloggt fuer Debugging:
```json
{"level":"INFO","msg":"Multiple CSRF cookies detected","count":2}
```

### Legacy-Cookie-Cleanup

Alte CSRF-Cookies mit `Path=/api` werden automatisch geloescht:

```go
// Bei jedem CSRF-Token-Refresh
http.SetCookie(w, &http.Cookie{
    Name:   "csrf_token",
    Value:  "",
    Path:   "/api",      // Alten Path matchen
    MaxAge: -1,          // Cookie loeschen
})
```

### Frontend-Integration

**Token-Refresh bei Session-Restore:**

Nach einem Browser-Neustart wird das CSRF-Token explizit refreshed:

```typescript
// auth.svelte.ts - initAuth()
if (authMethod === 'cookie') {
    // Session via Cookie wiederhergestellt
    await api.refreshCsrfToken();
}
```

**Token im Request:**

```typescript
// api.ts - request()
if (csrfToken && isStateChangingMethod(method)) {
    headers['X-CSRF-Token'] = csrfToken;
}
```

### Troubleshooting

#### "CSRF token mismatch"

**Symptome:** 403-Fehler bei POST/PUT/DELETE Requests

**Moegliche Ursachen:**
1. Cookie mit falschem Path (sollte `/` sein, nicht `/api`)
2. Mehrere Cookies mit unterschiedlichen Tokens
3. Token nach Page-Reload nicht refreshed

**Loesung:**
1. Browser-Cookies fuer Domain loeschen
2. Neu einloggen
3. Pruefen ob `csrf_token` Cookie Path=`/` hat (DevTools → Application → Cookies)

#### Cookie nicht in JavaScript lesbar

**Symptome:** `document.cookie` zeigt csrf_token nicht

**Ursache:** Cookie hat falschen Path oder ist HttpOnly

**Loesung:** Server-Konfiguration pruefen - CSRF-Cookie muss `HttpOnly: false` und `Path: /` haben

### Security-Ueberlegungen

| Aspekt | Status | Erklaerung |
|--------|--------|------------|
| XSS-Schutz | ✅ | CSRF-Token ist nur Cookie-Wert, kein sensitives Credential |
| CSRF-Schutz | ✅ | Double-Submit + SameSite=Strict |
| Token-Leakage | ✅ | Token hat keine Bedeutung ohne zugehoerige Session |
| Timing-Attacks | ✅ | Constant-Time-Vergleich bei Validierung |

## Future Enhancements

- [x] ~~HttpOnly Cookie Support~~ (✅ implementiert)
- [x] ~~CORS Credentials Support~~ (✅ implementiert)
- [x] ~~Proaktiver Token-Refresh~~ (✅ implementiert)
- [ ] Password Reset Flow (email-based)
- [ ] Email Verification
- [ ] Two-Factor Authentication (TOTP/SMS)
- [ ] OAuth Integration (Google, GitHub)
- [ ] Session Management UI
- [ ] Password Complexity Checker
- [ ] Rate Limiting (login attempts)
- [ ] Audit Log (security events)
- [ ] Account Deletion
- [ ] Export User Data (GDPR compliance)

## Weitere Dokumentation

- [CHANGELOG.md](../CHANGELOG.md) - Detailed release notes for v0.4.0
- [README.md](../README.md) - Project overview with authentication features
- [docs/api.md](api.md) - Complete API documentation
