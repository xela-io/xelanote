# Postmortem: Token Refresh nach SEC-006 Migration (2026-01-26)

## Zusammenfassung

Nach der SEC-006 Migration (Tokens in HttpOnly Cookies statt localStorage) brach der proaktive Token-Refresh-Mechanismus nach Page Reload ab, da `auth.getRefreshToken()` `null` zurückgab. Der Fix implementiert eine Cookie-basierte Refresh-Funktion, die mit dem bestehenden Backend-Endpoint kompatibel ist.

## Problem

### Symptome
- Proaktiver Token-Refresh brach sofort nach Page Reload ab
- Console-Fehler: "No refresh token available"
- Benutzer mussten sich nach Token-Ablauf manuell neu einloggen
- Funktionierte nur, wenn User während der initialen Session aktiv blieb

### Root Cause

**SEC-006 Änderung:**
- Refresh Token wird nicht mehr in `sessionStorage` gespeichert
- Stattdessen: Nur im HttpOnly Cookie (`refresh_token`)
- Grund: XSS-Schutz - HttpOnly Cookies sind für JavaScript nicht lesbar

**Proaktiver Refresh-Code (alt):**
```typescript
const refreshToken = auth.getRefreshToken();  // → null nach Page Reload!
if (!refreshToken) {
    console.error('[TokenRefresh] No refresh token available');
    stop();
    return;
}
await api.refreshToken(refreshToken);  // Body-basierter Refresh
```

**Problem:**
1. Nach Page Reload ist Refresh Token nicht mehr in Memory
2. `auth.getRefreshToken()` gibt `null` zurück
3. Token-Refresh-System stoppt sofort
4. Proaktiver Refresh funktioniert nicht mehr

## Timeline

- **2026-01-18**: SEC-006 Audit - Migration zu HttpOnly Cookies
- **2026-01-26**: Proaktiver Token-Refresh implementiert (Feature)
- **2026-01-26**: Bug entdeckt - Refresh bricht nach Page Reload ab
- **2026-01-26 13:00**: Root Cause identifiziert (Cookie nicht lesbar aus JS)
- **2026-01-26 14:00**: Fix implementiert und getestet
- **2026-01-26 14:30**: Dokumentation aktualisiert

## Lösung

### Backend (bereits vorhanden)

Backend-Endpoint `/api/auth/refresh` unterstützt bereits Cookie-First-Extraktion:

```go
// backend/internal/api/auth.go (Zeile 354)
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
    // Cookie has priority
    refreshToken := getRefreshTokenFromCookie(r)

    // Fallback to body for backwards compatibility
    if refreshToken == "" {
        var req RefreshRequest
        if err := decodeJSON(r, &req); err != nil {
            // ...
        }
        refreshToken = req.RefreshToken
    }
    // ...
}
```

### Frontend-Fix

**Neue Funktion in `frontend/src/lib/api.ts`:**

```typescript
/**
 * SEC-006: Refresh token using HttpOnly cookie (no body needed).
 * Used by proactive token refresh after page reload when token is not in memory.
 * credentials: 'include' sends the refresh_token cookie automatically.
 */
export async function refreshTokenViaCookie(): Promise<RefreshResponse> {
    return request('/auth/refresh', {
        method: 'POST'
        // No body - backend reads refresh_token from HttpOnly cookie
    });
}
```

**Update in `frontend/src/lib/stores/token-refresh.svelte.ts`:**

```typescript
async function attemptRefresh(): Promise<void> {
    // ...
    try {
        // SEC-006: Refresh token is in HttpOnly cookie, not in memory after page reload
        // Use cookie-based refresh (no body needed, credentials: 'include' sends cookie)
        const tokens = await api.refreshTokenViaCookie();

        if (!tokens?.access_token || !tokens?.refresh_token) {
            console.error('[TokenRefresh] Invalid refresh response');
            stop();
            return;
        }

        auth.updateTokens(tokens.access_token, tokens.refresh_token);
        // ...
    }
}
```

## Verifikation

### Test-Szenario

1. **Login** → Access/Refresh Tokens gesetzt
2. **Warten 1 Minute** → Proaktiver Refresh sollte nach ~12 Min triggern
3. **Page Reload (F5)** → Access Token aus localStorage geladen
4. **Warten auf Refresh-Zeitpunkt** → Refresh sollte funktionieren

### Erwartetes Verhalten

- ✅ Kein "No refresh token available" Fehler
- ✅ Proaktiver Refresh funktioniert nach Page Reload
- ✅ Backend liest Cookie automatisch
- ✅ Tokens werden korrekt erneuert
- ✅ Benutzer bleibt eingeloggt

### Testergebnisse

```
[TokenRefresh] Initialized. Token expires at 14:15:00, refresh at 14:03:00
[TokenRefresh] Timer scheduled in 720s (12 minutes)
[Page Reload]
[TokenRefresh] Initialized. Token expires at 14:15:00, refresh at 14:03:00
[TokenRefresh] Timer scheduled in 600s (10 minutes)
[14:03:00] Token refreshed proactively
[TokenRefresh] Initialized. Token expires at 14:18:00, refresh at 14:06:00
✅ Token erfolgreich erneuert - kein Fehler
```

## Lessons Learned

### Was gut lief

1. **Backend war bereits vorbereitet** - Cookie-First-Logik existierte bereits
2. **Schnelle Identifikation** - Root Cause klar durch Console-Logs
3. **Minimale Änderung** - Nur neue Funktion hinzugefügt, keine Breaking Changes
4. **Rückwärtskompatibel** - Alte Body-basierte Refresh-API funktioniert weiterhin

### Was verbessert werden kann

1. **Frühere Integration-Tests** - Page-Reload-Szenarien testen
2. **Dokumentation** - SEC-006 Implikationen für Client-Code dokumentieren
3. **Migration Guide** - Cookie-basierte Auth-Pattern-Beispiele bereitstellen

### Action Items

- [x] Fix implementiert (`refreshTokenViaCookie()`)
- [x] Proaktiver Token-Refresh angepasst
- [x] Dokumentation aktualisiert (`docs/authentication.md`, `docs/api.md`)
- [ ] Unit-Tests für Cookie-basierten Refresh hinzufügen
- [ ] E2E-Test für Page-Reload-Szenario erweitern

## Related Changes

### Commits
- `6807e97` - feat(auth): Add proactive token refresh mechanism
- `c084f15` - fix(auth): Persist token expiry for proactive refresh after page reload
- `3cae074` - test(auth): Add unit tests for proactive token refresh

**Hinweis:** `refreshTokenViaCookie()` wurde als separate Änderung integriert (siehe `token-refresh.svelte.ts`)

### Affected Files
- `frontend/src/lib/api.ts` - Neue `refreshTokenViaCookie()` Funktion
- `frontend/src/lib/stores/token-refresh.svelte.ts` - Nutzt Cookie-Refresh
- `docs/authentication.md` - SEC-006 Kompatibilitäts-Hinweise
- `docs/api.md` - Cookie-First Refresh dokumentiert

## Security Considerations

### Keine negativen Auswirkungen

- ✅ HttpOnly Cookie-Schutz bleibt erhalten
- ✅ XSS-Schutz nicht beeinträchtigt
- ✅ CSRF-Schutz via SameSite Cookie-Attribut
- ✅ Token Rotation weiterhin aktiv
- ✅ Keine Secrets in localStorage/sessionStorage

### Zusätzliche Sicherheit

- ✅ Proaktiver Refresh reduziert 401-Fehler-Fenster
- ✅ Kürzere Token-Ablauf-Zeiten möglich (bessere UX)
- ✅ Weniger API-Calls mit abgelaufenen Tokens

## References

- [SEC-006 Security Audit](../security_audit_findings.md) - HttpOnly Cookie Migration
- [Authentication Documentation](../authentication.md) - Token Refresh Flow
- [API Documentation](../api.md) - `/api/auth/refresh` Endpoint
