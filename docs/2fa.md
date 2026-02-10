# Two-Factor Authentication (2FA)

## Übersicht

xelanote unterstützt TOTP-basierte Zwei-Faktor-Authentifizierung (2FA) als optionalen zusätzlichen Sicherheitslayer. User können 2FA aktivieren, um ihr Konto mit einem Authenticator-App (Google Authenticator, Authy, 1Password, etc.) zu schützen.

## Features

- **TOTP-basiert**: Standardisiertes Time-based One-Time Password (RFC 6238)
- **Authenticator-App-Support**: Kompatibel mit allen TOTP-Apps
- **Backup-Codes**: 10 einmalige Recovery-Codes für den Notfall
- **Opt-in**: User entscheiden selbst, ob sie 2FA aktivieren
- **Admin-Transparenz**: Admins sehen im Admin-Panel, welche User 2FA aktiviert haben

## Frontend UI

Die 2FA-Funktionalität ist in drei Bereiche der Anwendung integriert:

### 1. Login-Flow (`/login`)
- **Zwei-Schritt-Login**: Nach korrekten Credentials wird bei aktiviertem 2FA ein zusätzlicher Code-Input angezeigt
- **TOTP oder Backup-Code**: User können zwischen 6-stelligem TOTP-Code oder Backup-Code (XXXX-XXXX Format) wechseln
- **CAPTCHA-Integration**: Beide Login-Schritte werden durch Turnstile CAPTCHA geschützt

### 2. Settings-Bereich (`/settings`)
- **2FA-Status-Anzeige**: Zeigt ob 2FA aktiviert ist und wie viele ungenutzte Backup-Codes vorhanden sind
- **Setup-Dialog**: Modaler 4-Schritt-Wizard (Intro → QR-Code → Verifizierung → Backup-Codes)
- **Disable-Dialog**: Passwort + TOTP/Backup-Code-Verifizierung zum Deaktivieren
- **Backup-Code-Regenerierung**: Erstellt neue Backup-Codes (alte werden ungültig)

### 3. Admin-Panel (`/admin`)
- **2FA-Spalte**: Zeigt ✓ (grün) bei aktivierten Accounts
- **Tooltip**: Hover über Status zeigt Aktivierungsdatum

### UI-Komponenten

**TwoFactorSetup.svelte**:
- Multi-Step-Dialog mit 4 Phasen
- QR-Code als Data URL vom Backend (`github.com/pquerna/otp` Go-Library)
- Automatischer Input-Focus und Keyboard-Navigation
- Dark-Mode-Support

**TwoFactorDisable.svelte**:
- Passwort + Code-Verifizierung
- Toggle zwischen TOTP und Backup-Code
- Warnhinweis vor Deaktivierung

**BackupCodesDisplay.svelte**:
- 2-Spalten-Grid mit allen 10 Codes
- "Alle kopieren" und "Als Datei speichern" Funktionen
- Checkbox-Bestätigung vor Abschluss

## Für Benutzer

### 2FA Einrichten

1. **Einstellungen öffnen**: Navigiere zu Settings → Account
2. **"2FA einrichten" klicken**: Öffnet Setup-Dialog mit 4 Schritten
3. **QR-Code scannen**:
   - Scanne den QR-Code mit deiner Authenticator-App
   - Alternativ: Kopiere den Secret-Key und gib ihn manuell in der App ein
4. **Code verifizieren**: Gib den 6-stelligen Code aus deiner App ein
5. **Backup-Codes sichern**:
   - Klicke "Alle kopieren" oder "Als Datei speichern"
   - Speichere die 10 Backup-Codes an einem sicheren Ort
   - Bestätige mit Checkbox, dass du sie gespeichert hast
   - ⚠️ **WICHTIG**: Diese Codes werden nur einmal angezeigt!

### Login mit 2FA

1. **Credentials eingeben**: Username/Email + Passwort + CAPTCHA
2. **2FA-Code eingeben**:
   - Standardmäßig: 6-stelliger TOTP-Code aus Authenticator-App
   - Alternativ: Klicke "Backup-Code verwenden" und gib Code im Format XXXX-XXXX ein
   - Beide Optionen benötigen CAPTCHA-Verifizierung
3. **Einloggen**: Nach erfolgreicher Verifizierung erhältst du Zugriff

### 2FA Deaktivieren

1. **Einstellungen öffnen**: Settings → Account → 2FA-Bereich
2. **"2FA deaktivieren" klicken**: Öffnet Bestätigungs-Dialog
3. **Passwort eingeben**: Zur Bestätigung deiner Identität
4. **TOTP-Code oder Backup-Code**:
   - Standardmäßig: 6-stelliger TOTP-Code
   - Alternativ: Klicke "Backup-Code verwenden"
5. **Bestätigen**: Alle 2FA-Daten (Secret, Backup-Codes) werden gelöscht

### Backup-Codes neu generieren

1. **Einstellungen öffnen**: Settings → Account → 2FA-Bereich
2. **"Backup-Codes neu generieren" klicken**
3. **Neue Codes speichern**: Die alten Codes werden ungültig, speichere die neuen!

### Gerät verloren?

Falls du dein Authenticator-Gerät verloren hast:
1. **Mit Backup-Code einloggen**: Verwende einen deiner gespeicherten Backup-Codes beim Login
2. **Neue Backup-Codes generieren**: Nach dem Login in Settings neue Codes erstellen
3. **Backup-Codes alle aufgebraucht**: Kontaktiere den Administrator
   - ⚠️ Es gibt **keinen automatischen Reset** aus Sicherheitsgründen

## Für Entwickler

### Architektur

```
┌─────────────┐
│  Frontend   │
│  (Svelte)   │
└─────┬───────┘
      │ POST /api/2fa/setup
      │ POST /api/2fa/verify
      │ DELETE /api/2fa
      ▼
┌─────────────┐
│ API Layer   │
│ (api.go)    │
└─────┬───────┘
      │
      ▼
┌─────────────┐
│  Service    │
│ (twofa.go)  │
└─────┬───────┘
      │
      ▼
┌─────────────┐
│   Database  │
│ (SQLite)    │
└─────────────┘
```

### Datenbank-Schema

**users table** (neue Felder):
```sql
totp_secret TEXT DEFAULT NULL               -- TOTP Secret (base32)
totp_enabled INTEGER DEFAULT 0              -- 2FA aktiviert?
totp_verified_at TEXT DEFAULT NULL          -- Aktivierungs-Timestamp
totp_disabled_at TEXT DEFAULT NULL          -- Deaktivierungs-Timestamp (Audit)
totp_setup_started_at TEXT DEFAULT NULL     -- Setup-Start (für Expiry)
last_totp_step INTEGER DEFAULT NULL         -- Replay-Protection
```

**backup_codes table** (neu):
```sql
id INTEGER PRIMARY KEY
user_id INTEGER NOT NULL                    -- FK zu users
code_hash TEXT NOT NULL                     -- bcrypt-Hash (Cost 12)
used INTEGER DEFAULT 0                      -- Verwendet?
used_at TEXT DEFAULT NULL                   -- Verwendungs-Timestamp
created_at TEXT DEFAULT (strftime(...))     -- Erstellungs-Timestamp
```

### API Endpoints

#### Setup starten
```http
POST /api/2fa/setup
Authorization: Bearer <access_token>

Response 200 OK:
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code_url": "otpauth://totp/xelanote:user@example.com?secret=...",
  "backup_codes": [
    "ABCD-1234",
    "EFGH-5678",
    ...
  ]
}
```

#### Setup verifizieren
```http
POST /api/2fa/verify
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "code": "123456"
}

Response 200 OK:
{
  "message": "2FA enabled successfully"
}
```

#### Status abrufen
```http
GET /api/2fa/status
Authorization: Bearer <access_token>

Response 200 OK:
{
  "enabled": true,
  "verified_at": "2026-01-20T12:34:56Z",
  "unused_backup_codes": 8
}
```

#### 2FA deaktivieren
```http
DELETE /api/2fa
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "password": "user_password",
  "totp_code": "123456",        // Option 1: TOTP-Code
  "backup_code": "ABCD-1234"    // Option 2: Backup-Code (Recovery)
}

Response 200 OK:
{
  "message": "2FA disabled successfully"
}
```

#### Backup-Codes regenerieren
```http
POST /api/2fa/backup-codes/regenerate
Authorization: Bearer <access_token>

Response 200 OK:
{
  "backup_codes": [
    "WXYZ-9876",
    "QRST-5432",
    ...
  ]
}
```

#### Login mit 2FA
```http
POST /api/auth/login
Content-Type: application/json

{
  "username_or_email": "user@example.com",
  "password": "password",
  "captcha_token": "...",
  "totp_code": "123456",        // Optional: TOTP-Code
  "backup_code": "ABCD-1234"    // Optional: Backup-Code
}

// Response 1: 2FA required
{
  "requires_two_factor": true
}

// Response 2: Success (after TOTP/Backup)
{
  "access_token": "...",
  "refresh_token": "...",
  "user": { ... }
}
```

### Security-Features

#### 1. TOTP-Replay-Protection

Verhindert Wiederverwendung von TOTP-Codes im selben Zeit-Fenster:

```go
// Time-Step-basiert (30-Sekunden-Intervalle)
currentStep := time.Now().Unix() / 30

// 60s-Window (current + previous step)
if currentStep <= lastStep+1 {
    return errors.New("TOTP code already used")
}

// Atomic Update (Race-Protection)
rowsAffected, _ := db.UpdateLastTOTPStep(userID, currentStep)
if rowsAffected == 0 {
    return errors.New("TOTP code already used (race detected)")
}
```

#### 2. Constant-Time Backup-Code-Verifikation

Verhindert Timing-Angriffe auf Backup-Codes:

```go
// Input normalisieren BEVOR bcrypt
code = strings.ToUpper(strings.ReplaceAll(code, "-", ""))

// Exakte Länge validieren (8 Zeichen)
if len(code) != 8 {
    // Dummy-bcrypt für konstantes Timing
    for range codes {
        bcrypt.CompareHashAndPassword(dummyHash, []byte("DUMMY"))
    }
    return errors.New("invalid backup code")
}

// ALLE Hashes prüfen (auch used=1)
for _, c := range codes {
    err := bcrypt.CompareHashAndPassword([]byte(c.CodeHash), []byte(code))
    // Kein break, kein continue, kein early return!
}
```

#### 3. Setup-Expiry

TOTP-Secrets verfallen nach 15 Minuten ohne Verifizierung:

```go
if tfa.TOTPSetupStartedAt != "" {
    setupTime, _ := time.Parse(time.RFC3339, tfa.TOTPSetupStartedAt)
    if time.Since(setupTime) > 15*time.Minute {
        db.ClearTOTPSetup(userID)
        return errors.New("setup expired")
    }
}
```

#### 4. Atomic Backup-Code-Updates

Race-Condition-Schutz bei parallelen Login-Versuchen:

```sql
-- Nur updaten wenn noch nicht verwendet
UPDATE backup_codes
SET used = 1, used_at = ?
WHERE id = ? AND used = 0

-- RowsAffected prüfen
-- 0 = bereits verwendet oder paralleler Request
```

#### 5. Keine Tokens vor 2FA

Cookie-Leak-Prävention:

```go
// Bei 2FA-Required: KEINE Tokens generieren
if tfa.TOTPEnabled {
    return "", "", true, nil  // requiresTwoFactor=true
}

// API-Layer: KEINE Cookies setzen
if requiresTwoFactor {
    respondJSON(w, http.StatusOK, AuthResponse{
        RequiresTwoFactor: true,
    })
    return  // Kein setAccessTokenCookie!
}
```

#### 6. Rate-Limiting (mehrschichtig)

```go
// Login (allgemein): 10 Versuche / 15 Min
loginLimiter: NewRateLimiter(10, 15*time.Minute, 10)

// 2FA-Verify: 5 Versuche / 15 Min (stricter)
tfaVerifyLimiter: NewRateLimiter(5, 15*time.Minute, 5)

// Backup-Code: 3 Versuche / 15 Min (bcrypt-DOS-Schutz)
backupCodeLimiter: NewRateLimiter(3, 15*time.Minute, 3)
```

### Code-Organisation

```
backend/
├── internal/
│   ├── db/
│   │   ├── twofa.go                    # Repository Layer
│   │   └── migrations/
│   │       └── 019_add_two_factor_auth.sql
│   ├── service/
│   │   ├── twofa.go                    # Business Logic
│   │   └── auth.go                     # Login-Flow angepasst
│   └── api/
│       ├── twofa.go                    # HTTP Handlers
│       ├── auth.go                     # Login-Handler angepasst
│       └── api.go                      # Routes registriert
└── cmd/server/main.go                  # Service-Initialisierung

frontend/
├── src/
│   ├── lib/
│   │   ├── api.ts                      # 2FA API Client (setup, verify, disable, regenerate)
│   │   ├── stores/
│   │   │   └── auth.svelte.ts          # Auth Store (keine 2FA-spezifischen Änderungen)
│   │   └── components/
│   │       ├── TwoFactorSetup.svelte   # Setup-Dialog (4 Schritte)
│   │       ├── TwoFactorDisable.svelte # Deaktivierungs-Dialog
│   │       └── BackupCodesDisplay.svelte # Backup-Code-Anzeige
│   └── routes/
│       ├── login/+page.svelte          # 2FA Login-Flow
│       ├── settings/+page.svelte       # 2FA Management
│       └── admin/+page.svelte          # 2FA Status-Spalte
└── package.json
```

### Frontend Dependencies

Keine speziellen Dependencies für 2FA. QR-Code wird vom Backend als Data URL geliefert.

### Frontend-Backend-Integration

**Auth Flow mit 2FA**:
```typescript
// 1. Initial Login (credentials only)
POST /api/auth/login
{
  username_or_email: "user@example.com",
  password: "password",
  captcha_token: "..."
}
→ Response: { requires_two_factor: true }

// 2. Frontend zeigt 2FA-Input an

// 3. Login mit TOTP/Backup-Code
POST /api/auth/login
{
  username_or_email: "user@example.com",
  password: "password",
  totp_code: "123456",      // oder backup_code
  captcha_token: "..."
}
→ Response: { access_token, refresh_token, user }
→ Frontend: setAuth() in auth.svelte.ts
```

**2FA Setup Flow**:
```typescript
// 1. Setup starten
const data = await api.setup2FA()  // POST /api/2fa/setup
// → { secret, qr_code_url, backup_codes }
// qr_code_url ist bereits ein Base64-Data-URL vom Backend

// 2. QR-Code direkt anzeigen
// <img src={data.qr_code_url} alt="QR Code" />

// 3. User scannt QR-Code in Authenticator-App

// 4. Code verifizieren
await api.verify2FA(totpCode)  // POST /api/2fa/verify

// 5. Backup-Codes anzeigen und speichern lassen
```

**Auth Store Changes**:
- Keine 2FA-spezifischen State-Änderungen
- Login-Flow bleibt unverändert: `setAuth(accessToken, refreshToken, user)`
- 2FA-Logik vollständig in Login-Page (`/login/+page.svelte`)

## Testing

### Manueller Test-Flow

1. **Setup**:
   ```bash
   curl -X POST http://localhost:8080/api/2fa/setup \
     -H "Authorization: Bearer $ACCESS_TOKEN"
   ```

2. **QR-Code scannen** mit Authenticator-App

3. **Verify**:
   ```bash
   curl -X POST http://localhost:8080/api/2fa/verify \
     -H "Authorization: Bearer $ACCESS_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"code": "123456"}'
   ```

4. **Login mit 2FA**:
   ```bash
   # Step 1: Credentials
   curl -X POST http://localhost:8080/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"username_or_email": "user@example.com", "password": "pwd", "captcha_token": "..."}'

   # Response: {"requires_two_factor": true}

   # Step 2: TOTP
   curl -X POST http://localhost:8080/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"username_or_email": "user@example.com", "password": "pwd", "totp_code": "123456", "captcha_token": "..."}'
   ```

5. **Status prüfen**:
   ```bash
   curl http://localhost:8080/api/2fa/status \
     -H "Authorization: Bearer $ACCESS_TOKEN"
   ```

### Unit-Tests (TODO)

Empfohlene Tests:
- `TestGenerateTOTPSetup`: Secret-Format, 10 Codes
- `TestVerifyTOTP`: Gültiger/ungültiger Code
- `TestTOTPReplay`: Code-Wiederverwendung blockiert
- `TestBackupCodeConstantTime`: Timing-Analyse
- `TestSetupExpiry`: 16 Min → Verify → Fehler
- `TestBackupCodeRace`: Parallele Verwendung

## Admin-Panel

### User-Übersicht

Neue Spalte "2FA" in der User-Tabelle zeigt Status:

**Aktiviert** (✓ in grünem Badge):
- Anzeige: Grüner Badge mit Checkmark-Symbol
- Tooltip: "Aktiviert seit [Datum/Uhrzeit]" (beim Hover)
- Implementierung: `totp_enabled: true`

**Nicht aktiviert** (–):
- Anzeige: Grauer Strich-Symbol
- Keine zusätzliche Info
- Implementierung: `totp_enabled: false`

### Frontend-Anzeige

```svelte
<td class="px-6 py-4 whitespace-nowrap text-sm">
  {#if user.totp_enabled}
    <span
      class="px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800
             dark:bg-green-900 dark:text-green-200"
      title={user.totp_verified_at
        ? `Aktiviert seit ${formatDate(user.totp_verified_at)}`
        : 'Aktiviert'}
    >
      &#x2713;
    </span>
  {:else}
    <span class="text-gray-400 dark:text-gray-600">&ndash;</span>
  {/if}
</td>
```

### API-Response

```typescript
interface AdminUser {
  id: number;
  username: string;
  email: string;
  is_admin: boolean;
  note_count: number;
  storage_mb: number;
  created_at: string;
  totp_enabled: boolean                // 2FA aktiviert?
  totp_verified_at?: string            // Aktivierungs-Timestamp (ISO 8601)
  totp_disabled_at?: string            // Deaktivierungs-Timestamp (Audit)
  totp_setup_started_at?: string       // Setup-Start (wird nicht angezeigt)
}
```

**Hinweis**: `totp_setup_started_at` wird im Admin-Panel nicht angezeigt, da unvollständige Setups nach 15 Minuten automatisch verfallen.

## Bekannte Einschränkungen

### Aktuelle Version
- ✅ TOTP-Secrets werden im Klartext in der DB gespeichert
  - ⚠️ Für Produktion wird AES-256-GCM-Verschlüsselung empfohlen
  - Fallback: SQLCipher für gesamte DB (via `XELANOTE_DB_KEY`)

### Zukünftige Verbesserungen
- [ ] TOTP-Secret-Verschlüsselung mit AES-256-GCM
- [ ] SMS/Email als 2FA-Alternative
- [ ] WebAuthn/FIDO2-Support
- [ ] Trusted-Device-Management
- [ ] 2FA-Pflicht für Admin-Accounts

## Troubleshooting

### Setup schlägt fehl

**"Setup expired"** (nach 15 Minuten):
- **Ursache**: Setup-Dialog zu lange offen gelassen, ohne Code zu verifizieren
- **Lösung**: Dialog schließen und "2FA einrichten" erneut klicken
- **Technisch**: Backend löscht unverifizierten Secret automatisch

**QR-Code wird nicht angezeigt**:
- **Browser-Problem**: Prüfe Browser-Konsole auf JavaScript-Fehler
- **Backend-Problem**: Prüfe ob Backend QR-Code-URL korrekt generiert (github.com/pquerna/otp)
- **Workaround**: Kopiere den Secret-Key manuell und gib ihn in der Authenticator-App ein

**Verifizierung schlägt fehl trotz korrektem Code**:
- **Zeit nicht synchronisiert**: Prüfe Systemzeit auf Gerät und Server
- **Code bereits verwendet**: Warte 30 Sekunden auf neuen Code
- **Rate-Limit**: 5 Versuche / 15 Min → Danach warten

### Login mit 2FA funktioniert nicht

**"Invalid TOTP code"**:
- **Zeit-Synchronisation**: Stelle sicher, dass Gerätezeit korrekt ist
- **Code-Wiederverwendung**: Jeder Code kann nur einmal in einem 60s-Fenster verwendet werden
- **Falscher Account**: Prüfe, ob du den richtigen xelanote-Account in der Authenticator-App ausgewählt hast
- **Lösung**: Warte auf neuen Code (30 Sekunden), stelle Zeit ein, oder verwende Backup-Code

**"Too many attempts"**:
- **Rate-Limit**: 5 Versuche / 15 Min für TOTP
- **Lösung**: 15 Minuten warten oder Backup-Code verwenden (3 Versuche / 15 Min)

**2FA-Input erscheint nicht nach Login**:
- **Browser-Cache**: Hard-Reload (Strg+Shift+R)
- **Session-Problem**: Komplett ausloggen, Browser neu starten
- **Backend-Problem**: Prüfe ob 2FA wirklich aktiviert ist (Admin-Panel)

### Backup-Codes funktionieren nicht

**Format-Fehler**:
- **Korrekt**: `ABCD-1234` (Bindestriche optional, case-insensitive)
- **Frontend**: Input normalisiert automatisch auf Uppercase ohne Bindestriche

**"Invalid backup code"**:
- **Bereits verwendet**: Jeder Code funktioniert nur einmal
- **Alte Codes**: Nach Regenerierung sind alte Codes ungültig
- **Tippfehler**: Kopiere Code aus gespeicherter Datei

**Rate-Limit**:
- **3 Versuche / 15 Min** (strenger als TOTP wegen bcrypt-DOS-Schutz)
- **Lösung**: Warten oder TOTP-Code verwenden

### Frontend-spezifische Probleme

**Setup-Dialog friert ein**:
- **Netzwerk-Timeout**: Prüfe Backend-Erreichbarkeit
- **Lösung**: Dialog schließen (ESC), erneut öffnen

**Backup-Codes können nicht kopiert/gespeichert werden**:
- **Clipboard-API**: Browser benötigt HTTPS oder localhost
- **Lösung**: Codes manuell abtippen oder Screenshot machen

**Dark-Mode-Probleme mit QR-Code**:
- **QR-Code immer weiß/schwarz**: Dies ist gewollt (Kompatibilität mit Apps)
- **Workaround**: Manual Secret-Key verwenden

### Gerät und Backup-Codes verloren

**Keine automatische Lösung** aus Sicherheitsgründen!

**Admin muss 2FA manuell deaktivieren**:
```sql
-- In SQLite-DB (auf Server):
UPDATE users SET
  totp_secret = NULL,
  totp_enabled = 0,
  totp_verified_at = NULL,
  totp_disabled_at = strftime('%Y-%m-%dT%H:%M:%SZ','now'),
  last_totp_step = NULL
WHERE id = <user_id>;

DELETE FROM backup_codes WHERE user_id = <user_id>;
```

**User-Verifizierung notwendig**:
- Admin sollte Identität des Users über anderen Kanal verifizieren (Email, persönlich, etc.)
- **Empfehlung**: User muss 2FA danach sofort neu einrichten

## Referenzen

### Spezifikationen
- **TOTP RFC**: [RFC 6238](https://tools.ietf.org/html/rfc6238) - Time-based One-Time Password Algorithm

### Backend-Libraries (Go)
- **TOTP-Generierung & QR-Code**: [github.com/pquerna/otp](https://github.com/pquerna/otp) - Generiert TOTP-Secret und QR-Code-URL (Data URL)
- **Passwort-Hashing**: [golang.org/x/crypto/bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)

### Frontend-Libraries (npm)
- **Icons**: [lucide-svelte](https://lucide.dev/) - Shield, ShieldCheck, ShieldOff Icons
- **Keine QR-Code-Library benötigt** - Backend liefert QR-Code als Data URL

### Authenticator-Apps (empfohlen)
- **Google Authenticator**: iOS, Android
- **Authy**: iOS, Android, Desktop (Multi-Device-Support)
- **1Password**: Cross-Platform (integrierte Password-Manager-Lösung)
- **Bitwarden**: Open-Source, Cross-Platform
- **Microsoft Authenticator**: iOS, Android
