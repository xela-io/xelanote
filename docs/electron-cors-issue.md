# Electron Desktop App - CORS Issue & Lösung

**Datum:** 25. Januar 2026
**Status:** ✅ Backend gefixt, ⚠️ Cloudflare Cache muss gepurgt werden
**Server:** xelanote.com (Production - Hetzner)

## Problem

Die Electron Desktop App (`app://.` origin) konnte sich nicht mit `https://xelanote.com` verbinden.

### Symptome

**Browser Console Error:**
```
Access to fetch at 'https://xelanote.com/api/auth/login' from origin 'app://.'
has been blocked by CORS policy: Response to preflight request doesn't pass
access control check: The value of the 'Access-Control-Allow-Origin' header
in the response must not be the wildcard '*' when the request's credentials
mode is 'include'.
```

**Electron Renderer Log:**
```
[Renderer error] Failed to fetch
[Renderer error] Login failed: TypeError: Failed to fetch
```

### Root Cause

Der Production Server auf xelanote.com hatte `app://.` nicht in den erlaubten CORS Origins:

```bash
# VORHER
CORS_ALLOWED_ORIGINS=https://xelanote.com,https://www.xelanote.com

# Origin 'app://.' wurde abgelehnt
```

## Lösung

### 1. Backend CORS-Konfiguration aktualisiert

**Server:** `xelanote-prod`
**Datei:** `~/.xelanote.env`

```bash
# NACHHER
CORS_ALLOWED_ORIGINS=https://xelanote.com,https://www.xelanote.com,app://.
```

**Implementierung:**
```bash
# SSH auf Production Server
ssh xelanote-prod

# Env-Datei aktualisieren
nano ~/.xelanote.env
# Füge 'app://.' hinzu

# Docker Container mit neuer Config neustarten
sudo docker stop xelanote && sudo docker rm xelanote
sudo docker run -d --name xelanote --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v ~/xelanote-data:/app/data \
  --memory=512m \
  --cpus=1 \
  --security-opt no-new-privileges \
  --pids-limit=200 \
  --env-file ~/.xelanote.env \
  xelanote:latest
```

**Verifikation:**
```bash
# Prüfe dass Container die neue Config hat
sudo docker exec xelanote env | grep CORS
# Output: CORS_ALLOWED_ORIGINS=https://xelanote.com,https://www.xelanote.com,app://.

# Teste CORS Preflight
curl -v -X OPTIONS https://xelanote.com/api/config -H "Origin: app://." 2>&1 | grep access-control-allow-origin
# Output: access-control-allow-origin: app://.
```

### 2. Backend Code (bereits korrekt implementiert)

**Datei:** `backend/internal/api/api.go`

Die CORS-Middleware war bereits korrekt implementiert:

```go
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        allowAll := len(s.allowedOrigins) == 0

        if allowAll {
            if origin != "" {
                // Development: echo origin
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Vary", "Origin")
            }
        } else if origin != "" && originAllowed(origin, s.allowedOrigins) {
            // Production: nur erlaubte Origins
            w.Header().Set("Access-Control-Allow-Origin", origin)
            w.Header().Set("Vary", "Origin")
        } else if r.Method == "OPTIONS" && origin != "" {
            http.Error(w, "origin not allowed", http.StatusForbidden)
            return
        }

        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, If-Match, Authorization, Cookie")
        w.Header().Set("Access-Control-Allow-Credentials", "true")
        w.Header().Set("Access-Control-Expose-Headers", "ETag")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusNoContent)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

**Wichtig:**
- SEC-003 konform: Kein Wildcard `*` mit `credentials: include`
- Origin wird aus erlaubter Liste geprüft
- Antwort echoed den requesting origin (nicht `*`)

## Cloudflare Cache Problem ⚠️

### Problem

**Cloudflare cached die alten CORS-Header!**

Auch nachdem der Backend-Server korrekte Header sendet, liefert Cloudflare noch gecachte Antworten mit `*` aus.

**Symptom:**
- `curl` direkt zum Server zeigt: `access-control-allow-origin: app://.` ✅
- Browser/Electron sieht: `access-control-allow-origin: *` ❌
- Fehler bleibt: "wildcard '*' not allowed with credentials"

### Lösungen

#### Option 1: Cache Purge (empfohlen, sofort wirksam)

1. Gehe zu: https://dash.cloudflare.com
2. Wähle Domain: `xelanote.com`
3. Linke Sidebar: **Caching** → **Configuration**
4. Klicke: **Purge Everything**
5. Bestätige mit "Purge Everything"

**Alternativ - Selective Purge:**
- Nur API-Pfade purgen: Custom Purge mit URL-Pattern `/api/*`

**Dauer:** Sofort wirksam (1-2 Sekunden)

#### Option 2: Development Mode (empfohlen für Testing)

1. Cloudflare Dashboard → `xelanote.com`
2. **Quick Actions** → Toggle **Development Mode** ON
3. Cache ist für 3 Stunden deaktiviert
4. Testen ob Login funktioniert
5. Development Mode wieder ausschalten

**Dauer:** Sofort wirksam, automatisch nach 3h wieder aus

#### Option 3: Warten (keine Aktion erforderlich)

Cloudflare Cache läuft automatisch ab.

**Dauer:** 5-10 Minuten (abhängig von Cache-TTL)

**Verifikation:**
```bash
# Prüfe ob Cloudflare noch cached
curl -I https://xelanote.com/api/config | grep cf-cache-status
# DYNAMIC = nicht gecached (gut)
# HIT = gecached (warte noch)
```

## Electron App neu starten

Nach Cache-Purge die App mit gelöschtem Cache neu starten:

```bash
# 1. App beenden
pkill -f xelanote-frontend

# 2. Electron Cache löschen
rm -rf ~/.config/xelanote-frontend/Cache
rm -rf ~/.config/xelanote-frontend/Code\ Cache

# 3. App neu starten
cd ~/Nextcloud/Documents/projects/xelanote/frontend
./release/xelanote-0.1.0-x86_64.AppImage
```

## Verifikation

### 1. Server sendet korrekte Header

```bash
curl -v -X OPTIONS https://xelanote.com/api/config \
  -H "Origin: app://." 2>&1 | grep access-control

# Erwartung:
# access-control-allow-origin: app://.
# access-control-allow-credentials: true
```

### 2. Cloudflare Cache Status

```bash
curl -I https://xelanote.com/api/config | grep cf-cache-status

# Erwartung:
# cf-cache-status: DYNAMIC  (nicht gecached)
```

### 3. Login funktioniert

1. Starte Electron App
2. Öffne DevTools (Strg+Shift+I)
3. Gehe zu Console Tab
4. Versuche Login
5. **Kein CORS-Fehler** mehr sichtbar
6. Login erfolgreich

## Weitere erlaubte Origins

Falls du weitere Origins hinzufügen möchtest:

```bash
ssh xelanote-prod
nano ~/.xelanote.env

# Füge neue Origins hinzu (komma-getrennt, KEINE Leerzeichen!)
CORS_ALLOWED_ORIGINS=https://xelanote.com,https://www.xelanote.com,app://.,tauri://localhost

# Container neustarten
sudo docker restart xelanote
```

**Wichtige Origins:**
- `https://xelanote.com` - Web-App Production
- `https://www.xelanote.com` - Web-App mit www
- `app://.` - Electron Desktop App
- `tauri://localhost` - Tauri Desktop App (falls verwendet)

## Security Hinweise

### ✅ Sicher (aktuelle Implementierung)

```
CORS_ALLOWED_ORIGINS=https://xelanote.com,https://www.xelanote.com,app://.
```

- Whitelist-basiert
- Keine Wildcards mit Credentials
- Origin wird geprüft und geechoed

### ❌ Unsicher (NIEMALS verwenden)

```bash
# FALSCH - erlaubt ALLE Origins
CORS_ALLOWED_ORIGINS=*

# FALSCH - leer = Development-Mode (echoed alle Origins)
CORS_ALLOWED_ORIGINS=
```

## Debugging

### Container-Logs prüfen

```bash
ssh xelanote-prod
sudo docker logs xelanote 2>&1 | grep -i cors
```

**Erwartung:** Keine Warnungen

**Wenn Warnung erscheint:**
```
CORS in permissive mode - echoing origin
```
→ CORS_ALLOWED_ORIGINS ist leer, setze in .env

### Electron App Logs

```bash
# App mit Logs starten
./release/xelanote-0.1.0-x86_64.AppImage &> /tmp/xelanote.log &

# Logs live verfolgen
tail -f /tmp/xelanote.log | grep -i cors
```

**Bei Erfolg:** Keine CORS-Fehler mehr

### Browser DevTools (Electron)

1. Starte App
2. Drücke `Strg+Shift+I` (DevTools öffnen)
3. Network Tab → Filtere "Fetch/XHR"
4. Login versuchen
5. Prüfe Response Headers bei API-Calls:
   - `access-control-allow-origin: app://.` ✅
   - `access-control-allow-credentials: true` ✅

## Timeline

**25. Januar 2026, 22:45 UTC:**
- Problem identifiziert (CORS blocked app://.)
- Backend .env aktualisiert
- Docker Container neugestartet
- Server sendet korrekte Header (verifiziert)

**25. Januar 2026, 22:50 UTC:**
- Cloudflare Cache-Problem entdeckt
- Lösung dokumentiert (Cache Purge erforderlich)

**Status:**
- ✅ Backend: Gefixt und deployed
- ⚠️ Cloudflare: Cache muss manuell gepurgt werden
- ⏳ Electron App: Wartet auf Cache-Purge

## Related

- [TitleBar Modernization](titlebar-modernization.md) - Desktop App UI Updates
- [Desktop App Documentation](desktop-app.md) - Vollständige Desktop App Docs
- Backend CORS-Middleware: `backend/internal/api/api.go` (Zeile 345-384)
- Frontend Config: `frontend/src/lib/config.ts` (getServerUrl, getApiBaseUrl)
