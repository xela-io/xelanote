# Plan: Mobile-Verbesserungen für xelanote Frontend (Revidiert)

## Zusammenfassung

Dieser Plan wurde nach kritischem Review überarbeitet. Änderungen:
- **Phase 0 hinzugefügt**: Research & Audit vor Implementierung
- **Task 1.3 gestrichen**: Checkbox Touch-Sizing bereits in app.css vorhanden
- **Task 1.2 als Spike**: Visual Viewport API hat iOS-Bugs, erst Prototype
- **iOS-Probleme adressiert**: Pull-to-Refresh und Long-Press mit Platform-Detection
- **Apple PWA Tags korrigiert**: Veraltete Meta-Tags durch moderne ersetzt

---

## Phase 0: Research & Audit (Voraussetzung)

> **WICHTIG:** Diese Phase muss VOR jeder Implementierung abgeschlossen werden!

### Task 0.1: Code-Audit bestehender Mobile-Styles
**Komplexität:** S (30 min)

**Beschreibung:** Dokumentiere alle bereits vorhandenen Mobile-Optimierungen

**Bereits gefunden:**
- [x] `app.css:6-15` - Dynamic Viewport Height (100dvh)
- [x] `app.css:18-21` - Safe Area Insets (.pb-safe, .pt-safe, etc.)
- [x] `app.css:23-29` - Touch-friendly Toolbar Buttons (44px)
- [x] `app.css:32-43` - Touch-action manipulation
- [x] `app.css:826-836` - **Task-Checkboxen bereits 20px auf Touch!**

**Akzeptanzkriterien:**
- [ ] Vollständige Liste aller `@media (pointer: coarse)` Regeln
- [ ] Liste aller Touch-Event-Listener im Frontend
- [ ] Dokumentation in `docs/mobile.md` (optional)

---

### Task 0.2: User Research - Echte Mobile-Probleme
**Komplexität:** M (1-2h)

**Beschreibung:** Identifiziere die Top 3 echten Mobile-UX-Probleme

**Methoden:**
- [ ] GitHub Issues nach "mobile", "touch", "iOS", "Android" durchsuchen
- [ ] Eigene Tests auf echtem Device (nicht Simulator!)
- [ ] Optional: 2-3 Testnutzer beobachten (Guerilla Testing)

**Akzeptanzkriterien:**
- [ ] Liste der Top 3 bestätigten Mobile-Probleme
- [ ] Priorisierung nach: `Severity × Frequency`
- [ ] Go/No-Go Entscheidung für jeden geplanten Task

**Offene Fragen:**
- Wurde das Keyboard-verdeckt-Editor-Problem jemals gemeldet?
- Wie viele aktive Mobile-Nutzer gibt es?

---

### Task 0.3: Testing-Setup
**Komplexität:** S (30 min)

**Beschreibung:** Echtes iOS-Device-Testing ermöglichen

**Optionen:**
- [ ] Eigenes iPhone/iPad verfügbar?
- [ ] BrowserStack Free Trial (3h/Monat)
- [ ] Freunde/Familie mit iOS-Device

**Akzeptanzkriterien:**
- [ ] Mindestens 1 echtes iOS-Device für Testing verfügbar
- [ ] Safari Web Inspector Remote Debug funktioniert
- [ ] Android-Device für Vergleichstests

**Warum wichtig:** iOS Simulator bildet Touch-Events und Visual Viewport API nicht korrekt ab!

---

### Task 0.4: Performance-Baseline
**Komplexität:** S (15 min)

**Beschreibung:** Lighthouse Mobile Score vor Änderungen aufzeichnen

**Akzeptanzkriterien:**
- [ ] Lighthouse Mobile Score dokumentiert (Performance, Accessibility, Best Practices)
- [ ] Memory-Profil mit geöffneter Sidebar (viele Notes)
- [ ] Baseline für Vorher/Nachher-Vergleich

---

## Phase 1: Quick Wins (Hohe Priorität)

### Task 1.1: Touch-Sizing für Login/Register Inputs
**Komplexität:** S (30 min)

**Beschreibung:** Input-Fields auf min 48px Höhe für bessere Touch-Targets

**Dateien:**
- `frontend/src/routes/login/+page.svelte`
- `frontend/src/routes/register/+page.svelte`

**Akzeptanzkriterien:**
- [ ] Input-Fields haben `min-height: 48px` auf Touch (`@media (pointer: coarse)`)
- [ ] Font-size bleibt 16px (verhindert iOS Auto-Zoom) ✓ bereits vorhanden
- [ ] Layout passt noch in Card auf iPhone SE (375px)
- [ ] Desktop-Layout unverändert
- [ ] **Getestet auf echtem iOS-Device**

**Hinweis:** Bestehende Styles prüfen - `padding: 0.75rem` könnte mit `min-height` kollidieren.

---

### ~~Task 1.2: Visual Viewport Handling~~ → **SPIKE**
### Task 1.2: Visual Viewport API - Spike/Prototype
**Komplexität:** M (2h Spike, dann Go/No-Go)

**Beschreibung:** Prototype bauen und auf echtem iOS testen, BEVOR Implementierung

**Warum Spike:**
- Visual Viewport API hat bekannte iOS Safari Bugs (iOS 13-15)
- `100dvh` + `safe-area-inset` bereits implementiert - könnte kollidieren
- Debounce-Timing kritisch (Keyboard öffnet in ~300ms)

**Spike-Ziele:**
- [ ] Minimaler Prototype in separatem Branch
- [ ] Test auf iOS Safari 15+ (echtes Device!)
- [ ] Test auf iOS Safari 14 (falls User-Base relevant)
- [ ] Dokumentiere: Funktioniert es zuverlässig? Flackert es?

**Go/No-Go Kriterien:**
- **GO:** Funktioniert auf iOS 15+ ohne Flackern, graceful degradation auf älteren
- **NO-GO:** Bugs, Flackern, oder nur marginale Verbesserung → Skip, `100dvh` reicht

**Fallback bei NO-GO:** Behalte aktuelle `100dvh` + `h-screen-safe` Lösung

---

### ~~Task 1.3: Task-Checkboxen Touch-Sizing~~ → **GESTRICHEN**

**Grund:** Bereits implementiert in `app.css:826-836`:
```css
@media (pointer: coarse) {
    .task-list-item-checkbox {
        width: 1.25rem;  /* 20px */
        height: 1.25rem;
    }
}
```

**Optional:** Falls 20px nicht ausreicht (User-Feedback!), auf 24px erhöhen.

---

### ~~Task 1.4: Resize-Handler Debounce~~ → **NEU DEFINIERT**
### Task 1.4: Audit Resize-Handler
**Komplexität:** S (20 min)

**Beschreibung:** Prüfe ob Resize-Handler überhaupt existieren und Debounce brauchen

**Audit durchführen:**
```bash
grep -r "addEventListener.*resize" frontend/src/
grep -r "window.resize" frontend/src/
```

**Akzeptanzkriterien:**
- [ ] Liste aller Resize-Listener mit Datei:Zeile
- [ ] Für jeden: Braucht er Debounce? (Window-Resize vs User-Drag)
- [ ] Falls ja: `debounce.ts` Utility erstellen
- [ ] Falls nein: Task als "nicht nötig" schließen

---

## Phase 2: Haptic Feedback (Mittlere Priorität)

### Task 2.1: Haptic Feedback Utility
**Komplexität:** S (25 min)

**Beschreibung:** Vibration API Wrapper mit Feature-Detection

**Dateien:**
- `frontend/src/lib/utils/haptics.ts` (neu)

**Akzeptanzkriterien:**
- [ ] `hapticFeedback(type)` exportiert
- [ ] Types: `'light' | 'medium' | 'heavy' | 'success' | 'error'`
- [ ] Patterns: light=10ms, medium=20ms, heavy=40ms
- [ ] Feature-Detection: Return `false` wenn nicht verfügbar
- [ ] TypeScript-Typen

**Code-Beispiel:**
```typescript
export function hapticFeedback(type: 'light' | 'medium' | 'heavy'): boolean {
  if (!navigator.vibrate) return false;
  const patterns = { light: 10, medium: 20, heavy: 40 };
  return navigator.vibrate(patterns[type]);
}
```

---

### Task 2.2: Haptic Feedback in Editor
**Komplexität:** S (20 min)

**Beschreibung:** Feedback bei wichtigen Actions (optional, nice-to-have)

**Nur implementieren bei:**
- [ ] Task-Toggle: `light`
- [ ] Note-Save (Erfolg): `success`

**NICHT implementieren:** Upload, Delete, Undo/Redo (Overkill)

**Abhängigkeit:** Task 2.1

---

### ~~Task 2.3: Pull-to-Refresh~~ → **BEDINGT**
### Task 2.3: Pull-to-Refresh (nur wenn iOS 16.4+ Zielgruppe)
**Komplexität:** M (50 min)

**Beschreibung:** Pull-Down Gesture zum Neuladen

**KRITISCHES RISIKO:**
- `overscroll-behavior-y: contain` funktioniert NICHT auf iOS Safari <16.4
- Natives Browser-PTR + Custom PTR = doppelte Spinner

**Implementieren NUR WENN:**
- [ ] User-Base ist iOS 16.4+ (Check Analytics!)
- [ ] ODER: Expliziter "Reload"-Button als Alternative akzeptabel

**Alternative (empfohlen):**
Statt PTR → Sichtbarer Refresh-Button in Sidebar-Header

**Falls implementiert - Akzeptanzkriterien:**
- [ ] Feature-Detection: `CSS.supports('overscroll-behavior-y', 'contain')`
- [ ] Deaktiviert auf iOS <16.4 (User-Agent oder Feature-Detection)
- [ ] Visueller Spinner bei Pull > 80px
- [ ] Haptic Feedback bei Trigger

---

### ~~Task 2.4: Long-Press~~ → **PLATFORM-SPEZIFISCH**
### Task 2.4: Long-Press Kontext-Menü (nur Android)
**Komplexität:** M (40 min)

**Beschreibung:** 500ms Long-Press für Kontext-Menü

**iOS-PROBLEM:**
- `preventDefault()` blockiert natives Context-Menu auf iOS NICHT
- Long-Press triggert BEIDE Menüs = schlechte UX

**Lösung:**
- **Android:** Long-Press Action implementieren
- **iOS:** Expliziter "..." Menu-Button (bereits vorhanden?)

**Akzeptanzkriterien:**
- [ ] Platform-Detection: `'ontouchstart' in window && !navigator.userAgent.includes('iPhone')`
- [ ] Nur auf Android/Chrome aktiv
- [ ] iOS-User sehen "..." Button statt Long-Press
- [ ] Haptic Feedback bei Trigger (Android)

---

## Phase 3: PWA Polish (Niedrige Priorität)

### Task 3.1: PWA Manifest Shortcuts
**Komplexität:** S (25 min)

**Beschreibung:** Quick-Actions für Android Home-Screen

**Dateien:**
- `frontend/vite.config.ts`

**Akzeptanzkriterien:**
- [ ] Shortcuts: "Neue Notiz" (/), "Graph" (/graph)
- [ ] Funktioniert auf Android Chrome
- [ ] Graceful: Ignoriert auf iOS (nicht unterstützt)

---

### ~~Task 3.2: Apple PWA Meta-Tags~~ → **KORRIGIERT**
### Task 3.2: PWA Meta-Tags modernisieren
**Komplexität:** S (15 min)

**Beschreibung:** Veraltete Apple-Tags durch moderne Standards ersetzen

**Dateien:**
- `frontend/src/app.html`

**NICHT verwenden (deprecated seit iOS 13):**
```html
<!-- VERALTET - NICHT HINZUFÜGEN -->
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
```

**Stattdessen (bereits vorhanden prüfen):**
```html
<meta name="theme-color" content="#1a1a2e">
<meta name="theme-color" media="(prefers-color-scheme: dark)" content="#1a1a2e">
<link rel="manifest" href="/manifest.webmanifest">
```

**Akzeptanzkriterien:**
- [ ] `theme-color` Meta-Tag vorhanden (für iOS 15+)
- [ ] Manifest hat `display: "standalone"`
- [ ] Keine veralteten `apple-mobile-web-app-*` Tags

---

### Task 3.3: Settings-Page Touch-Sizing
**Komplexität:** S (30 min)

**Beschreibung:** Touch-optimierte Inputs auf Settings

**Akzeptanzkriterien:**
- [ ] Alle Inputs min-height 48px auf Touch
- [ ] Spacing mindestens 8px zwischen Elementen
- [ ] Getestet auf echtem Device

---

## Gestrichene/Verschobene Tasks

| Task | Status | Grund |
|------|--------|-------|
| 1.3 Checkbox-Sizing | GESTRICHEN | Bereits in app.css implementiert (20px) |
| 1.4 Resize Debounce | NEU DEFINIERT | Erst Audit, dann entscheiden |
| 2.3 Pull-to-Refresh | BEDINGT | iOS <16.4 Bug, Alternative: Reload-Button |
| 2.4 Long-Press | EINGESCHRÄNKT | Nur Android, iOS bekommt Menu-Button |
| 3.2 Apple Meta-Tags | KORRIGIERT | Veraltete Tags, moderne Alternative |

---

## Revidierter Zeitplan

| Phase | Tasks | Aufwand | Voraussetzung |
|-------|-------|---------|---------------|
| **0** | Research & Audit | 2-3h | - |
| **1** | Quick Wins | 1-2h | Phase 0 abgeschlossen |
| **2** | Haptic (optional) | 1h | User-Feedback positiv |
| **3** | PWA Polish | 1h | Zeit übrig |

**Gesamt: 5-7h** (reduziert von 6h, aber realistischer)

---

## Checkliste vor Implementierung

- [ ] Phase 0 abgeschlossen
- [ ] Echtes iOS-Device für Testing verfügbar
- [ ] Top 3 Mobile-Probleme identifiziert und validiert
- [ ] Visual Viewport Spike: Go oder No-Go?
- [ ] Performance-Baseline aufgezeichnet

---

## Offene Entscheidungen

1. **Visual Viewport:** Spike durchführen → Go/No-Go
2. **Pull-to-Refresh:** Implementieren oder Reload-Button?
3. **Haptic Feedback:** Wirklich gewünscht oder Overkill?
4. **iOS <16.4 Support:** Relevant für User-Base?
