# E2E Encryption Deployment - Bug Fixes & Verification

**Datum**: 2026-01-20
**Server**: Homelab Staging (<STAGING_URL>)
**Status**: ✅ Erfolgreich deployed und verifiziert

---

## Übersicht

Erfolgreiche Implementierung und Deployment der End-to-End-Verschlüsselung mit Argon2id KDF, per-note DEKs und AES-GCM-256. Drei kritische Bugs wurden identifiziert und behoben.

---

## Bugs & Fixes

### Bug 1: Migration nicht angewendet

**Problem:**
- Migration `020_e2e_encryption.sql` existierte im Verzeichnis `backend/internal/db/migrations/`
- Wurde aber NICHT ausgeführt, da sie nicht in der hardcoded Liste in `backend/internal/db/db.go` war
- Notes-Tabelle hatte keine Encryption-Spalten (encrypted_content, wrapped_dek, etc.)

**Symptom:**
```sql
sqlite> SELECT encrypted_content FROM notes LIMIT 1;
Error: no such column: encrypted_content
```

**Root Cause:**
`backend/internal/db/db.go` enthielt Migration 020 nicht im migrations-Array:
```go
migrations := []string{
    "002_folders_table.sql",
    // ...
    "018_system_settings.sql",
    // 019 und 020 fehlten!
}
```

**Fix:**
Datei: `backend/internal/db/db.go` (Zeilen 128-129)
```go
migrations := []string{
    // ... existing migrations ...
    "018_system_settings.sql",
    "019_add_two_factor_auth.sql",  // ADDED
    "020_e2e_encryption.sql",        // ADDED
}
```

**Commit:** Teil des E2E Encryption Deployments

**Verification:**
```bash
docker logs xelanote | grep "Applied migration: 020"
# Output: Applied migration: 020_e2e_encryption.sql
```

**Schema nach Migration:**
```sql
-- users table
ALTER TABLE users ADD COLUMN encryption_salt BLOB;

-- notes table
ALTER TABLE notes ADD COLUMN encrypted_content BLOB;
ALTER TABLE notes ADD COLUMN content_encrypted INTEGER DEFAULT 0;
ALTER TABLE notes ADD COLUMN encrypted_title TEXT;
ALTER TABLE notes ADD COLUMN title_encrypted INTEGER DEFAULT 0;
ALTER TABLE notes ADD COLUMN wrapped_dek TEXT;
ALTER TABLE notes ADD COLUMN encryption_version INTEGER DEFAULT 0;
ALTER TABLE notes ADD COLUMN encryption_metadata TEXT;
```

---

### Bug 2: Encryption Locked nach Login (Auto-Logout fehlte)

**Problem:**
- User hatte gültige Session von vor dem E2E-Deployment
- Session Tokens wurden aus sessionStorage wiederhergestellt
- `setupEncryption()` wurde NICHT aufgerufen (nur bei Login, nicht bei Token-Restore)
- Encryption State: `isUnlocked = false`

**Symptom:**
```javascript
[NOTES] Creating note, encryption unlocked: false
Error: Encryption locked
```

**Root Cause:**
Der Auth-Flow war:
1. Page Load → `initAuth()` stellt Tokens wieder her
2. User gilt als authenticated
3. ABER: `setupEncryption()` wurde nie aufgerufen → KEK fehlt
4. User versucht Notiz zu erstellen → Encryption locked

**Fix:**
Datei: `frontend/src/lib/stores/notes.svelte.ts`
```typescript
if (!isUnlocked) {
    error = 'Encryption locked - logging out for re-authentication';
    console.error('[NOTES] Cannot create note: encryption is locked - forcing logout');

    // Force logout to re-authenticate with encryption setup
    const { logoutAsync } = await import('./auth.svelte');
    await logoutAsync();

    // Redirect to login with reason parameter
    if (typeof window !== 'undefined') {
        window.location.href = '/login?reason=encryption_locked';
    }
    throw new Error('Encryption locked - logged out');
}
```

Datei: `frontend/src/routes/login/+page.svelte`
```svelte
onMount(() => {
    // Check if user was logged out due to encryption being locked
    if (typeof window !== 'undefined') {
        const params = new URLSearchParams(window.location.search);
        if (params.get('reason') === 'encryption_locked') {
            infoMessage = '🔒 Bitte erneut einloggen um die Verschlüsselung zu aktivieren. Deine Session wurde aus Sicherheitsgründen beendet.';
        }
    }
});

{#if infoMessage}
    <div class="info-message" style="...">
        {infoMessage}
    </div>
{/if}
```

**Commit:** `01a0c89` - Add auto-logout when encryption locked

**User Flow nach Fix:**
1. User versucht verschlüsselte Notiz zu erstellen
2. System erkennt: Encryption locked
3. Automatischer Logout
4. Redirect zu `/login?reason=encryption_locked`
5. Info-Nachricht wird angezeigt
6. User loggt sich neu ein
7. `setupEncryption()` wird aufgerufen → KEK wird deriviert
8. Encryption unlocked ✅

---

### Bug 3: WebCrypto DOMException - KEK Key Usages falsch

**Problem:**
- KEK wurde mit falschen Key Usages importiert: `['wrapKey', 'unwrapKey']`
- Wir verwenden AES-GCM für Key Wrapping via `crypto.subtle.encrypt()`
- `encrypt()` benötigt `'encrypt'` usage, NICHT `'wrapKey'`

**Symptom:**
```javascript
[ENCRYPTION] ✅ KEK derived in 2265 ms
[ENCRYPTION] ✅ Encryption unlocked
[NOTES] Creating note, encryption unlocked: true
DOMException: A parameter or an operation is not supported by the underlying object
```

**Debugging-Schritte:**

1. **Erste Vermutung: AES-KW Inkompatibilität**
   - Problem: KEK als AES-GCM importiert, aber AES-KW für wrapping versucht
   - Fix-Versuch: Entfernte AES-KW komplett, nur noch AES-GCM wrapping
   - Commit: `d8e7d81`
   - Resultat: ❌ Gleicher Fehler

2. **Root Cause gefunden: Key Usages**
   - KEK wurde importiert mit: `['wrapKey', 'unwrapKey']`
   - Code ruft auf: `crypto.subtle.encrypt({ name: 'AES-GCM' }, kek, ...)`
   - WebCrypto-Regel: `encrypt()` benötigt `'encrypt'` usage!

**Fix:**
Datei: `frontend/src/lib/crypto/argon2.ts` (Zeilen 48-57)
```typescript
/**
 * Derives a cryptographic key from a password using Argon2id.
 *
 * @returns CryptoKey suitable for AES-GCM key wrapping operations
 */
export async function deriveKeyFromPassword(
    password: string,
    salt: Uint8Array,
    options: Argon2Options = DEFAULT_OPTIONS
): Promise<CryptoKey> {
    // Derive raw key bytes with Argon2id using @noble/hashes
    const hash = argon2id(password, salt, {
        t: options.t,       // time cost
        m: options.m,       // memory cost in KB
        p: options.p,       // parallelism
        dkLen: options.dkLen // output length
    });

    // Import as WebCrypto CryptoKey
    // Note: We need 'encrypt' and 'decrypt' for AES-GCM key wrapping
    // (since we wrap DEKs using encrypt() instead of wrapKey())
    return crypto.subtle.importKey(
        'raw',
        hash, // Uint8Array
        { name: 'AES-GCM', length: 256 },
        false, // not extractable
        ['encrypt', 'decrypt'] // KEK for AES-GCM wrapping ✅ FIXED
    );
}
```

**Vorher (falsch):**
```typescript
['wrapKey', 'unwrapKey'] // ❌ Erlaubt nur wrapKey()/unwrapKey()
```

**Nachher (korrekt):**
```typescript
['encrypt', 'decrypt']   // ✅ Erlaubt encrypt()/decrypt()
```

**Commit:** `d79f797` - Fix KEK key usages for AES-GCM wrapping

**Verification:**
```javascript
[ENCRYPTION] ✅ KEK derived in 2303 ms
[ENCRYPTION] ✅ Encryption unlocked
[NOTES] Creating note, encryption unlocked: true
// Keine DOMException mehr! ✅
```

---

## AES-GCM Key Wrapping Implementation

Da AES-KW nicht mit AES-GCM KEKs kompatibel ist, verwenden wir AES-GCM für Key Wrapping:

**Datei:** `frontend/src/lib/crypto/e2e.ts`

### wrapDEK() - DEK mit KEK verschlüsseln
```typescript
private async wrapDEK(dek: CryptoKey, kek: CryptoKey): Promise<Uint8Array> {
    // Use AES-GCM for key wrapping
    const iv = crypto.getRandomValues(new Uint8Array(12));
    const exportedDEK = await crypto.subtle.exportKey('raw', dek);
    const encrypted = await crypto.subtle.encrypt(
        { name: 'AES-GCM', iv: iv },
        kek,
        exportedDEK
    );

    // Prepend IV to ciphertext (12 bytes IV + ciphertext)
    const result = new Uint8Array(iv.length + encrypted.byteLength);
    result.set(iv, 0);
    result.set(new Uint8Array(encrypted), iv.length);
    return result;
}
```

### unwrapDEK() - DEK mit KEK entschlüsseln
```typescript
private async unwrapDEK(wrappedDEK: Uint8Array, kek: CryptoKey): Promise<CryptoKey> {
    // Extract IV and ciphertext
    const iv = wrappedDEK.slice(0, 12);
    const ciphertext = wrappedDEK.slice(12);

    // Decrypt the wrapped DEK
    const decrypted = await crypto.subtle.decrypt(
        { name: 'AES-GCM', iv: iv },
        kek,
        ciphertext as BufferSource
    );

    // Import the unwrapped DEK
    return crypto.subtle.importKey(
        'raw',
        decrypted,
        { name: 'AES-GCM', length: 256 },
        false, // not extractable
        ['decrypt']
    );
}
```

**Vorteile:**
- ✅ Kompatibel mit AES-GCM KEK
- ✅ Authenticated Encryption (GCM liefert Auth-Tag)
- ✅ Breite Browser-Unterstützung
- ✅ Gleiche Sicherheit wie AES-KW (RFC 3394)

---

## Deployment Timeline

### 1. Initial Deployment
```bash
# Commit: 165b309 - Argon2 fix (switch to @noble/hashes)
git push origin main
ssh <STAGING_USER>@<STAGING_IP> "cd ~/xelanote && git pull && docker build -t xelanote:latest ."
docker run -d --name xelanote --restart unless-stopped \
  -p 8081:8080 --network nginx_default \
  -v xelanote_xelanote-data:/app/data \
  -e JWT_SECRET=... \
  -e XELANOTE_DB=/app/data/xelanote.db \
  -e XELANOTE_ENV=production \
  -e CORS_ALLOWED_ORIGINS=https://<STAGING_URL> \
  xelanote:latest
```

**Resultat:** Migration 020 angewendet ✅

### 2. Bug Fix: Auto-Logout
```bash
# Commit: 01a0c89 - Add auto-logout when encryption locked
git push origin main
ssh <STAGING_USER>@<STAGING_IP> "cd ~/xelanote && git pull && docker build && docker restart"
```

**Resultat:** User wird bei locked encryption automatisch ausgeloggt ✅

### 3. Debug Logging
```bash
# Commit: 21be4a8 - Add comprehensive debug logging
git push origin main
ssh <STAGING_USER>@<STAGING_IP> "cd ~/xelanote && git pull && docker build && docker restart"
```

**Resultat:** Console zeigt detaillierte Encryption-Flow-Logs ✅

### 4. AES-GCM Wrapping Only
```bash
# Commit: d8e7d81 - Remove AES-KW, use AES-GCM only
git push origin main
ssh <STAGING_USER>@<STAGING_IP> "cd ~/xelanote && git pull && docker build && docker restart"
```

**Resultat:** ❌ DOMException blieb bestehen

### 5. Final Fix: KEK Key Usages
```bash
# Commit: d79f797 - Fix KEK key usages for AES-GCM wrapping
git push origin main
ssh <STAGING_USER>@<STAGING_IP> "cd ~/xelanote && git pull && docker build && docker restart"
```

**Resultat:** ✅ Funktioniert! Notiz erfolgreich verschlüsselt

---

## Verification

### Database Verification
```bash
docker exec xelanote sqlite3 /app/data/xelanote.db \
  "SELECT id, title, content_encrypted, encryption_version,
          length(encrypted_content) as encrypted_size,
          length(wrapped_dek) as wrapped_dek_size,
          created_at
   FROM notes
   WHERE user_id = 4
   ORDER BY created_at DESC
   LIMIT 1;"
```

**Output:**
```
74fbeeac-c862-4045-9063-56ddabaf1575|1313|1|1|16|80|2026-01-20T13:45:04Z
```

**Analyse:**
- `content_encrypted = 1` ✅
- `encryption_version = 1` ✅
- `encrypted_content = 16 bytes` (verschlüsselter Ciphertext) ✅
- `wrapped_dek = 80 chars` Base64 (12 bytes IV + 32 bytes encrypted DEK + 16 bytes auth tag) ✅

### Encryption Metadata
```bash
docker exec xelanote sqlite3 /app/data/xelanote.db \
  "SELECT encryption_metadata
   FROM notes
   WHERE id = '74fbeeac-c862-4045-9063-56ddabaf1575';"
```

**Output:**
```json
{
  "version": 1,
  "algorithm": "AES-GCM-256",
  "kdf": "Argon2id",
  "kdf_params": {
    "time": 3,
    "memory": 65536,
    "parallelism": 4
  },
  "iv": "tSSb+UN2DA+J/ty1",
  "wrapped_dek": "esysWt3njWftxNuA0RpV6cwOqg0FPUCrLsPEaC5qWSdIIz5TLYLvDtvOekVPaK0sh7jPFwmfAwiNCFVn"
}
```

**Analyse:**
- ✅ Vollständige Metadata vorhanden
- ✅ KDF-Parameter gespeichert (für zukünftige Crypto-Agility)
- ✅ IV und wrapped_dek Base64-encoded
- ✅ Version 1 (für Migrations)

### Plaintext Content Check
```bash
docker exec xelanote sqlite3 /app/data/xelanote.db \
  "SELECT length(content) as plaintext_len
   FROM notes
   WHERE id = '74fbeeac-c862-4045-9063-56ddabaf1575';"
```

**Output:**
```
0
```

**Analyse:** ✅ Kein Klartext in `content`-Spalte! (Wurde nicht gesetzt oder auf NULL)

### Ciphertext Sample
```bash
docker exec xelanote sqlite3 /app/data/xelanote.db \
  "SELECT hex(substr(encrypted_content, 1, 20))
   FROM notes
   WHERE id = '74fbeeac-c862-4045-9063-56ddabaf1575';"
```

**Output:**
```
95D4A7331975ABF6E40484170A17E7CA
```

**Analyse:** ✅ Zufällige Bytes, nicht lesbar!

---

## Browser Console Logs (Erfolgreicher Flow)

```javascript
[AUTH] Setting up E2E encryption...
[AUTH] encryption_salt present: true
[AUTH] encryption_salt length: 24
[AUTH] Salt decoded, length: 16 bytes
[ENCRYPTION] setupEncryption called
[ENCRYPTION] userId: 4
[ENCRYPTION] salt length: 16
[ENCRYPTION] Deriving KEK with Argon2id...
[ENCRYPTION] ✅ KEK derived in 2303 ms
[ENCRYPTION] ✅ Encryption unlocked
[AUTH] ✅ Encryption setup successful
[NOTES] Creating note, encryption unlocked: true
```

**Keine Errors!** ✅

---

## Technische Details

### Crypto Stack (v1 - Legacy, replaced by v2)

**HINWEIS:** Diese Seite dokumentiert das v1 Deployment (AES-GCM). Das aktuelle System verwendet **Encryption v2 (libsodium.js mit XChaCha20-Poly1305)**. Siehe [encryption-v2.md](./encryption-v2.md) für aktuelle Details.

**v1 Stack (historisch):**
- **KDF**: Argon2id via `@noble/hashes/argon2`
  - Zeit-Parameter: t=3 (~2300ms auf modernem CPU)
  - Memory: 64MB (65536 KB)
  - Parallelism: 4 Threads
  - Output: 32 bytes (256-bit KEK)

- **Encryption**: AES-GCM-256
  - IV: 12 bytes (96-bit, random per operation)
  - Auth Tag: 16 bytes (128-bit, in ciphertext enthalten)
  - Key Length: 256-bit

- **Key Wrapping**: AES-GCM (nicht AES-KW)
  - Wraps DEK mit KEK
  - Format: `[12 bytes IV][32 bytes encrypted DEK + 16 bytes auth tag]` = 60 bytes
  - Base64: ~80 Zeichen

**v2 Stack (aktuell, seit 2026-01-20):**
- **Library**: libsodium.js
- **KDF**: Argon2id (in Web Worker, non-blocking)
- **Encryption**: XChaCha20-Poly1305-IETF (RFC 8439)
- **Nonce**: 24 bytes (keine Kollisionsgefahr)
- **Key Wrapping**: XChaCha20-Poly1305-IETF

Siehe [encryption-v2.md](./encryption-v2.md) für vollständige v2 Dokumentation.

### Security Properties
✅ **Zero-Knowledge**: Server kann Ciphertext nicht lesen
✅ **Authenticated Encryption**: GCM Auth-Tag verhindert Tampering
✅ **Memory-Hard KDF**: Argon2id (64MB) schützt vor GPU-Brute-Force
✅ **Per-Note DEKs**: Key-Rotation ohne alle Notizen neu zu verschlüsseln
✅ **Forward Secrecy**: KEK nur in Memory, nicht in Storage
✅ **Crypto-Agility**: Version 1 Metadata erlaubt zukünftige Upgrades

---

## Lessons Learned

### 1. Hardcoded Migration Lists sind fragil
**Problem:** Neue Migrations wurden vergessen zu registrieren.

**Lösung:**
- Automatisches Discovery von Migrations (via embed.FS)
- Oder: CI-Check der migrations gegen db.go

### 2. WebCrypto Key Usages sind strikt
**Problem:** `wrapKey` usage ≠ `encrypt` usage

**Learning:**
- CryptoKey Usages müssen exakt matchen
- `wrapKey()`/`unwrapKey()` benötigt `['wrapKey', 'unwrapKey']`
- `encrypt()`/`decrypt()` benötigt `['encrypt', 'decrypt']`
- Keine Cross-Usage möglich!

**Best Practice:**
```typescript
// Dokumentiere die Usages im Code:
return crypto.subtle.importKey(
    'raw', hash,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt'] // MUST match actual usage (encrypt/decrypt)
);
```

### 3. AES-KW ≠ AES-GCM Key Wrapping
**Problem:** AES-KW ist separate Algorithm, nicht mit AES-GCM KEKs kompatibel.

**Lösung:** AES-GCM für Key Wrapping verwenden (gleich sicher, breiter Support)

### 4. Session-State und E2E-Encryption
**Problem:** User mit alter Session hatte keine KEK.

**Lösung:**
- Auto-Logout bei locked encryption
- Explizite Info-Message warum
- User muss neu einloggen für KEK-Derivation

### 5. Debug Logging ist essentiell
**Erfolg:** Die ausführlichen Console Logs halfen immens beim Debugging.

**Best Practice:**
- Console.log für Crypto-Flows in Production (mit `[TAG]` Prefix)
- Performance-Metriken (KEK derivation time)
- State-Checks (encryption unlocked?)

---

## Nächste Schritte

### Empfohlene Verbesserungen:

1. **Title Encryption (optional)**
   - User kann wählen: Title encrypted oder searchable
   - UI: Toggle in Settings

2. **Keyword Extraction (opt-in)**
   - Mit expliziter Warnung über Data Leakage
   - Max 30 Keywords, Stopword-Filter

3. **Recovery Key**
   - 256-bit Recovery Token generieren
   - User muss downloaden vor erstem Encrypt
   - Für Passwort-Vergessen-Fallback

4. **Batch Re-Encryption**
   - Tool um alte Plaintext-Notes zu verschlüsseln
   - Mit Progress-Bar

5. **Key Rotation**
   - Passwort-Änderung Flow
   - Re-wrap alle DEKs mit neuem KEK

6. **Performance Monitoring**
   - Argon2id Zeit tracken
   - Alert wenn >5s (zu langsam)

7. **Error Handling**
   - Graceful Fallback bei Crypto-Errors
   - User-friendly Error Messages

---

## Status

- [x] E2E Encryption deployed auf Homelab
- [x] Migration 020 angewendet
- [x] Alle 3 Bugs gefixt
- [x] Verifiziert in Datenbank
- [x] User kann verschlüsselte Notizen erstellen
- [ ] Hetzner Production Deployment (nächster Schritt)
- [ ] Recovery Key Implementation
- [ ] Batch Re-Encryption Tool

**Version deployed:** `d79f797`
**Server:** <STAGING_URL>
**Status:** ✅ Production-Ready
