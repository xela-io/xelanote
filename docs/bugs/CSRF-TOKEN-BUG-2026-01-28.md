# Bug Report: CSRF Token Validation Failed After SEC-L04

**Datum:** 2026-01-28
**Schweregrad:** Critical (Anwendung nicht nutzbar)
**Status:** In Bearbeitung

## Symptome

Nach dem SEC-L04 Security-Fix können Benutzer keine Notizen mehr speichern, keine Themes wechseln und keine neuen Notizen anlegen. Alle state-changing Requests (PUT, POST, DELETE) schlagen mit HTTP 403 fehl:

```
Failed to save: ApiError: CSRF token validation failed
```

## Analyse

### Chronologie der Fehler

1. **Erste Analyse:** `CSRF token header missing`
   - JavaScript konnte das CSRF-Cookie nicht lesen
   - Ursache: Cookie hatte `Path=/api`, Seite war `/note/...`
   - Fix: Cookie-Path auf `/` geändert

2. **Nach Fix:** `CSRF token mismatch`
   - JavaScript liest jetzt ein Cookie
   - Aber Cookie-Wert ≠ Header-Wert
   - Ursache: Es existieren ZWEI Cookies mit demselben Namen

### Root Cause

Es gibt zwei `csrf_token` Cookies mit unterschiedlichen Pfaden:
- **Altes Cookie:** `csrf_token` mit `Path=/api` (vom alten Code)
- **Neues Cookie:** `csrf_token` mit `Path=/` (vom neuen Code)

**Problem:**
1. `document.cookie` auf `/note/...` sieht nur das Cookie mit `Path=/` (neuer Token)
2. Der Browser sendet bei Requests an `/api/...` BEIDE Cookies
3. Der Server liest das ERSTE Cookie (möglicherweise das alte mit `Path=/api`)
4. → Mismatch zwischen Header (neuer Token) und Cookie (alter Token)

### Betroffene Dateien

- `backend/internal/api/csrf.go` - Cookie-Einstellungen
- `backend/internal/api/cookies.go` - Auth-Cookie-Einstellungen
- `frontend/src/lib/stores/auth.svelte.ts` - Session-Wiederherstellung
- `frontend/src/lib/api.ts` - CSRF-Token-Lesen

## Lösungsvorschläge

### Option 1: Altes Cookie explizit löschen (Backend)

Bei jedem Token-Refresh beide Cookie-Pfade löschen:

```go
// In csrf.go - vor dem Setzen des neuen Cookies
func setCSRFTokenCookie(w http.ResponseWriter, token string) {
    // Lösche altes Cookie mit Path=/api
    http.SetCookie(w, &http.Cookie{
        Name:   csrfCookieName,
        Value:  "",
        Path:   "/api",
        MaxAge: -1,
    })

    // Setze neues Cookie mit Path=/
    cookie := &http.Cookie{
        Name:     csrfCookieName,
        Value:    token,
        Path:     "/",
        // ... rest
    }
    http.SetCookie(w, cookie)
}
```

**Vorteile:** Einmaliger Fix, räumt alte Cookies auf
**Nachteile:** Temporär zwei Set-Cookie Headers

### Option 2: Cookie-Namen ändern

Neuen Cookie-Namen verwenden (z.B. `xelanote_csrf`):

```go
const csrfCookieName = "xelanote_csrf" // statt "csrf_token"
```

**Vorteile:** Keine Konflikte mit alten Cookies
**Nachteile:** Erfordert koordinierte Änderung in Frontend und Backend

### Option 3: Logout erzwingen (User-seitig)

Benutzer müssen sich ausloggen und neu einloggen, um alte Cookies zu löschen.

**Vorteile:** Kein Code-Change nötig
**Nachteile:** Schlechte UX, nicht automatisierbar

### Option 4: Frontend - Cookie vor Refresh löschen

Im Frontend vor dem Token-Refresh das alte Cookie löschen:

```typescript
// In auth.svelte.ts vor refreshTokenViaCookie()
document.cookie = 'csrf_token=; Path=/api; Max-Age=0';
document.cookie = 'csrf_token=; Path=/; Max-Age=0';
```

**Vorteile:** Keine Backend-Änderung
**Nachteile:** Frontend kann nur Cookies löschen, die es lesen kann (Path=/)

### Option 5: Server - Spezifischeren Cookie zuerst lesen

Im Backend den Cookie mit dem spezifischeren Pfad ignorieren:

```go
// In validateCSRF - alle Cookies mit dem Namen lesen und den mit Path=/ bevorzugen
// HTTP-Standard: Cookies werden nach Spezifität sortiert gesendet
```

**Vorteile:** Robuster gegen Cookie-Konflikte
**Nachteile:** Komplexere Logik, HTTP-Cookie-Verhalten ist browser-abhängig

## Empfehlung

**Option 1 (Backend: Altes Cookie explizit löschen)** ist die sauberste Lösung:
- Einmaliger Fix
- Räumt die Legacy-Cookies automatisch auf
- Erfordert keine User-Aktion
- Kein Breaking Change

## Zusätzliche Maßnahmen

1. **Postmortem erstellen** für SEC-L04 Follow-up
2. **Test hinzufügen** für Cookie-Path-Handling
3. **Dokumentation** der Cookie-Einstellungen erweitern

## Reproduktion

1. User war vor dem Fix eingeloggt (hat altes Cookie mit Path=/api)
2. Fix wird deployed (neues Cookie hat Path=/)
3. User macht Page Reload
4. Token Refresh setzt neues Cookie mit Path=/
5. Altes Cookie mit Path=/api existiert noch
6. PUT Request → Server sieht altes Cookie → Mismatch → 403

## Workaround (für Benutzer)

Bis zum Fix: Browser-Cookies für xelanote.com manuell löschen und neu einloggen.
