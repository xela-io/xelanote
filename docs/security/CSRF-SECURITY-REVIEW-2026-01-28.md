# CSRF Security Review - 2026-01-28

**Reviewer:** Claude Opus 4.5
**Scope:** `backend/internal/api/csrf.go` und zugehörige Komponenten
**Status:** ✅ Sicher mit kleineren Empfehlungen

---

## Executive Summary

Die CSRF-Implementierung ist **solide** und folgt dem Double-Submit Cookie Pattern korrekt. Die kürzlichen Fixes haben die Sicherheit **nicht** verschlechtert. Es gibt einige kleinere Verbesserungsmöglichkeiten.

---

## Analyse

### ✅ Positiv

| Aspekt | Bewertung | Details |
|--------|-----------|---------|
| **Token-Generierung** | ✅ Sicher | 32 Bytes aus `crypto/rand` = 256 Bit Entropie |
| **Token-Vergleich** | ✅ Sicher | `subtle.ConstantTimeCompare` verhindert Timing-Angriffe |
| **SameSite=Strict** | ✅ Stark | Verhindert Cross-Site-Requests komplett |
| **Secure Flag** | ✅ Korrekt | Nur HTTPS in Production |
| **Token-Rotation** | ✅ Gut | Neuer Token bei jedem Login/Refresh |
| **Bearer-Token Bypass** | ✅ Korrekt | CSRF nur relevant bei Cookie-Auth |

### ⚠️ Zu prüfen

#### 1. Token-Länge nach Base64-Encoding

```go
csrfTokenLength = 32  // 32 Bytes = 256 Bit
base64.URLEncoding.EncodeToString(bytes)  // → 44 Zeichen
```

**Bewertung:** ✅ Ausreichend. 256 Bit Entropie ist mehr als genug.

#### 2. Cookie ohne Domain-Attribut

```go
cookie := &http.Cookie{
    Name:     csrfCookieName,
    // Domain fehlt → implizit aktuelle Domain
}
```

**Bewertung:** ✅ Korrekt. Ohne explizites `Domain` gilt das Cookie nur für die exakte Domain (kein Subdomain-Sharing), was sicherer ist.

#### 3. Multiple-Cookie-Handling

```go
for _, cookie := range csrfCookies {
    if subtle.ConstantTimeCompare(...) == 1 {
        return result  // Akzeptiert ersten Match
    }
}
```

**Bewertung:** ✅ Sicher. Ein Angreifer müsste trotzdem einen gültigen Token kennen, um einen passenden Header zu senden.

#### 4. Bearer-Token CSRF-Bypass

```go
if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " && !hasCookie {
    // Skip CSRF
}
```

**Bewertung:** ✅ Korrekt. Die Bedingung `!hasCookie` ist wichtig - ein Angreifer kann keine Custom-Header bei Cross-Origin-Requests setzen (CORS verhindert das).

**Aber:** Was wenn ein Angreifer einen Request mit `Authorization: Bearer X` Header UND ohne Cookies sendet?
- Der Request würde CSRF bypassen ✓
- Aber `authMiddleware` würde den ungültigen Token ablehnen ✓
- **Kein Risiko**

---

## Potenzielle Schwachstellen (alle bewertet)

### 1. ❌ CSRF-Token Leakage via Referer

**Szenario:** CSRF-Token ist in Cookie, könnte theoretisch via Referer-Header leaken.

**Bewertung:** Nicht anwendbar. Das Token ist im Cookie, nicht in der URL. `Referrer-Policy: strict-origin-when-cross-origin` ist gesetzt.

### 2. ❌ XSS → CSRF-Token-Diebstahl

**Szenario:** Bei XSS kann ein Angreifer `document.cookie` lesen und das CSRF-Token stehlen.

**Bewertung:** Ja, aber bei XSS hat der Angreifer ohnehin vollen Zugriff. Das CSRF-Token zu stehlen bringt keinen zusätzlichen Vorteil - er kann direkt Requests im Namen des Users machen.

**Mitigation:** XSS verhindern (CSP, Input-Sanitization). CSRF-Schutz ist nicht gegen XSS gedacht.

### 3. ❌ Subdomain-Takeover

**Szenario:** Angreifer übernimmt `evil.xelanote.com` und kann Cookies für `.xelanote.com` setzen.

**Bewertung:**
- Das CSRF-Cookie hat kein `Domain`-Attribut → gilt nur für exakte Domain
- Ein Angreifer auf einer Subdomain kann kein Cookie für `xelanote.com` setzen
- **Kein Risiko** (solange keine Subdomains existieren)

### 4. ⚠️ Token nicht an Session gebunden

**Beobachtung:** Das CSRF-Token ist nicht kryptographisch an die User-Session gebunden.

**Szenario:**
1. User A loggt sich ein, bekommt CSRF-Token X
2. User A loggt sich aus
3. User B loggt sich ein (gleiches Gerät), Cookie X existiert noch
4. User B's Requests verwenden Token X

**Bewertung:**
- **Geringes Risiko** in der Praxis (gleicher Browser, gleicher User-Kontext)
- Bei Logout werden Cookies gelöscht (`clearCSRFTokenCookie`)
- Bei Login wird neues Token generiert

**Empfehlung (optional):** Token könnte `HMAC(userID, sessionID, secret)` sein für stärkere Bindung.

---

## Empfehlungen

### Priorität: Niedrig (Nice-to-have)

1. **Token-Rotation bei kritischen Aktionen**
   ```go
   // Nach Passwort-Änderung, 2FA-Aktivierung etc.
   setCSRFTokenCookie(w, newToken)
   ```
   Bereits implementiert bei `/auth/refresh`.

2. **Legacy-Cookie-Cleanup entfernen** (nach 30 Tagen)
   ```go
   // TODO 2026-02-28: Remove legacy cookie cleanup
   // clearLegacyCSRFCookie can be removed after all users have migrated
   ```

3. **Rate-Limiting bei CSRF-Fehlern**
   Bereits vorhanden via `rateLimitMiddleware`.

---

## Fazit

| Kategorie | Status |
|-----------|--------|
| **Double-Submit Pattern** | ✅ Korrekt implementiert |
| **Token-Entropie** | ✅ 256 Bit (ausreichend) |
| **Timing-Angriffe** | ✅ Constant-time Vergleich |
| **SameSite Policy** | ✅ Strict (beste Option) |
| **Cookie Flags** | ✅ Secure, korrekter Path |
| **Session-Binding** | ⚠️ Nicht explizit, aber durch Login/Refresh implizit |

**Gesamtbewertung:** Die CSRF-Implementierung ist **produktionsreif** und **sicher**. Die kürzlichen Fixes haben die Sicherheit nicht verschlechtert - `SameSite=Strict` ist sogar eine Verbesserung.

---

## Appendix: OWASP CSRF Cheat Sheet Compliance

| OWASP Empfehlung | Status |
|------------------|--------|
| Synchronizer Token Pattern oder Double-Submit | ✅ Double-Submit |
| Token pro Session | ✅ Bei Login/Refresh |
| Token in Custom Header | ✅ X-CSRF-Token |
| SameSite Cookie Attribut | ✅ Strict |
| Verify Origin Header | ❌ Nicht implementiert (optional bei SameSite=Strict) |

**Hinweis:** Origin-Header-Prüfung ist bei `SameSite=Strict` redundant, da der Browser Cross-Site-Requests komplett blockiert.
