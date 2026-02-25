# Design-Konsistenz Audit – Februar 2026

> Automatisiert erstellt am 2026-02-25 mittels Playwright Design-Tests und Code-Analyse.

## Gesamtbewertung: 8/10 – Gut, mit Verbesserungspotenzial

Das Projekt hat ein **solides, einheitliches Design-System**. Farbpalette, Typografie, Spacing und Komponenten folgen einem klaren, durchgaengigen Muster. Die Hauptschwaeche sind **hardcoded Tailwind-Farben** in ca. 13 Komponenten, die das Token-System umgehen.

---

## 1. Was gut funktioniert

### Konsistentes Theme-System
- Gruvbox Light + Dark Theme mit warmen Retro-Farben
- OKLCH-Farbraum fuer perzeptuelle Gleichmaessigkeit
- Vollstaendiges Token-System: `--color-*`, `--radius-*`, `--duration-*`, `--ease-*`
- Theme-Wechsel funktioniert auf allen Seiten korrekt (0 Dark-Mode Violations)

### Einheitliche Komponenten
- `.ui-*` CSS-Klassen-Hierarchie (page, panel, button, form, tab, list, empty-state)
- Button-Varianten (primary/secondary/ghost/outline/destructive) konsistent
- Panel/Card-Styling ueberall gleich (Rundungen, Shadows, Padding)
- Empty-States einheitlich gestaltet (Icon + Text + CTA)

### Gutes Responsive Design
- Sidebar → Mobile BottomNav Transition
- Page-Header-Pattern konsistent auf allen Seiten
- iOS PWA: Safe Areas, Dynamic Viewport, Standalone-Modus

### Typografie
- Inter Font durchgaengig (400/500/600/700)
- Klare Hierarchie: Titel → Subtitle → Body → Caption
- 16px auf Mobile verhindert iOS Safari Auto-Zoom

---

## 2. Playwright Design-Audit Ergebnisse

### Test-Zusammenfassung

| Test                          | Status     | Details                                     |
|-------------------------------|------------|---------------------------------------------|
| Desktop Layout (6 Seiten)     | **PASS**   | Kein horizontaler Overflow                  |
| Mobile Layout (4 Seiten)      | **PASS**   | Kein horizontaler Overflow                  |
| Mobile Touch-Targets          | **WARN**   | 58 Elemente < 44px                          |
| Kontrast (WCAG AA)            | **PASS**   | Keine kritischen Verletzungen               |
| Heading-Hierarchie            | **WARN**   | 2 Seiten mit h1→h3 Sprung                  |
| Dark Mode                     | **PASS**   | 0 Violations                                |

### Mobile Touch-Target Warnings

| Seite       | Anzahl | Beispiele                                        |
|-------------|--------|--------------------------------------------------|
| `/`         | 14     | Sidebar-Buttons 32x32px, BottomNav-Items 126x32px |
| `/settings` | 11     | Tab-Icons 44x40px (knapp unter Minimum)          |
| `/recipes`  | 8      | Filter-Chips 168x36px, Links 151x20px            |
| `/journal`  | 6      | Datum-Button 97x26px                             |

### Schriftgroesse-Warnings (Mobile < 12px)

| Element          | Groesse  | Vorkommen              |
|------------------|----------|------------------------|
| Sidebar-Icons    | 10px     | Alle Seiten            |
| Kicker-Text      | 11.52px  | `/recipes`             |
| BottomNav-Label  | 11px     | `/` (Home)             |

### Heading-Hierarchie-Probleme

| Seite                  | Problem            |
|------------------------|--------------------|
| `/settings`            | h1 → h3 (h2 fehlt)|
| `/settings/encryption` | h1 → h3 (h2 fehlt)|

---

## 3. Hardcoded Farben (Token-Bypasses)

### Kritisch – Sollten Design-Tokens nutzen

| Datei                       | Zeile | Ist                              | Soll                          |
|-----------------------------|-------|----------------------------------|-------------------------------|
| `Toast.svelte`              | 24    | `bg-yellow-500 text-white`       | `bg-warning text-warning-foreground` |
| `AlertDialog.svelte`        | 24    | `bg-yellow-500`                  | `bg-warning`                  |
| `AlertDialog.svelte`        | 46    | `text-yellow-500`                | `text-warning`                |
| `AITransformDialog.svelte`  | 275   | `text-green-500`                 | `text-success`                |
| `EditorToolbar.svelte`      | 188   | `bg-blue-500 text-white`         | `bg-primary text-primary-foreground` |
| `EditorToolbar.svelte`      | 198   | `text-blue-500 bg-blue-500/12`   | `text-primary bg-primary/12`  |
| `CanvasEditorToolbar.svelte`| 134   | `bg-blue-500 text-white`         | `bg-primary text-primary-foreground` |
| `CanvasLinkDialog.svelte`   | 39    | `text-red-600`                   | `text-destructive`            |
| `FilterChip.svelte`         | 21    | `text-purple-600`                | Neuer Token oder `text-accent` |
| `FilterMenu.svelte`         | 103   | `text-purple-600 dark:text-purple-400` | Konsistenter Token      |
| `SessionRestoreBanner.svelte`| 9    | `bg-blue-600/95 text-white`      | `bg-primary/95 text-primary-foreground` |
| `OfflineBanner.svelte`      | 33    | `bg-blue-500`, `bg-amber-500/600`| `bg-primary`, `bg-warning`    |

### Akzeptabel – Semantisch korrekt im Kontext

| Datei                           | Kontext                | Anmerkung                             |
|---------------------------------|------------------------|---------------------------------------|
| `ConflictDialog.svelte`         | Git-artige Diff-Ansicht| Gruen/Rot fuer Added/Removed sinnvoll |
| `VersionHistoryDialog.svelte`   | Versions-Diff          | Gleiche Konvention                    |
| `settings/encryption/+page.svelte` | Sicherheitswarnungen| Hat korrekte `dark:` Varianten        |
| `settings/migration/+page.svelte`  | Datenmigration      | Hat korrekte `dark:` Varianten        |
| `note/[id]/+page.svelte`       | Fehlermeldung          | Einfacher Fehlertext                  |

---

## 4. Visuelle Konsistenz (Screenshot-Analyse)

### Seiten-uebergreifende Konsistenz

| Aspekt              | Bewertung | Anmerkung                                          |
|---------------------|-----------|-----------------------------------------------------|
| Sidebar             | **10/10** | Identisch auf allen Seiten, gleiche Icons/Spacing   |
| Page-Header         | **9/10**  | Titel + Action-Buttons konsistent, Subtitles variieren |
| Panel/Cards         | **9/10**  | Gleiche Rundungen und Schatten ueberall             |
| Buttons             | **9/10**  | Primary-CTA immer oben rechts, konsistente Groesse  |
| Empty-States        | **9/10**  | Icon + Text + Link, gleicher Stil                   |
| Mobile BottomNav    | **8/10**  | Konsistent, aber Labels etwas klein (11px)          |
| Settings Tabs       | **7/10**  | Desktop: Text + Icon, Mobile: nur Icons (schwer erkennbar) |
| Color-Consistency   | **7/10**  | Theme-Tokens gut, aber Hardcoded-Farben problematisch |

### Theme-Konsistenz (Light vs. Dark)

| Seite       | Light→Dark Transition | Anmerkung                     |
|-------------|:---:|---------------------------------------------------|
| Home        | ok  | Farben korrekt invertiert                        |
| Recipes     | ok  | Panels und Empty-State passen sich an            |
| Journal     | ok  | Lock-Icon und CTA-Button korrekt                |
| Settings    | ok  | Tabs und Panels konsistent                       |
| Graph       | ok  | Knoten-Farben passen sich an                     |

---

## 5. Empfehlungen nach Prioritaet

### P1 – Schnelle Fixes (je < 5 Min.)

1. `Toast.svelte:24` → `bg-warning text-warning-foreground`
2. `AlertDialog.svelte:24,46` → `bg-warning`, `text-warning`
3. `AITransformDialog.svelte:275` → `text-success`
4. `EditorToolbar.svelte:188,198` → `bg-primary`, `text-primary`
5. `CanvasLinkDialog.svelte:39` → `text-destructive`

### P2 – Mittlerer Aufwand

6. Settings-Seiten: h2-Ebene zwischen h1 und h3 einfuegen
7. Mobile BottomNav: Label-Groesse von 11px auf 12px erhoehen
8. `FilterChip`/`FilterMenu`: Purple-Farbe als `--color-filter-active` Token definieren

### P3 – Groessere Aufgaben

9. Wiederverwendbare Alert/Banner-Komponente mit Token-Unterstuetzung
10. Settings Mobile: Tab-Labels als Tooltip oder kleine Beschriftung ergaenzen
11. Backend `validThemes` in `user_types.go` aufraeumen: 22 Legacy-Theme-IDs auf 2 aktive reduzieren

---

## 6. Test-Kommandos

```bash
# Design-Audit (Layout, Touch-Targets, Kontrast, Headings, Dark Mode)
TMPDIR=/workspace/.tmp CI=1 npx playwright test --project=design tests/design/design-audit.spec.ts --workers=1

# Visual Regression Screenshots generieren (Light)
TMPDIR=/workspace/.tmp CI=1 npx playwright test --project=e2e tests/e2e/ui-visual-regression.spec.ts --update-snapshots --workers=1

# Visual Regression Screenshots generieren (Dark)
TMPDIR=/workspace/.tmp CI=1 VISUAL_THEME=gruvbox-dark npx playwright test --project=e2e tests/e2e/ui-visual-regression.spec.ts --update-snapshots --workers=1

# Accessibility
TMPDIR=/workspace/.tmp CI=1 npx playwright test --project=accessibility --workers=1

# Alle Design-Tests
TMPDIR=/workspace/.tmp CI=1 npx playwright test --project=design --project=accessibility --workers=1
```

> **Hinweis:** `TMPDIR=/workspace/.tmp` ist noetig wenn `/tmp` < 256MB hat (typisch in Container-Umgebungen).

---

## 7. Snapshot-Inventar

### Vorhandene Baselines (26 Dateien)

**Desktop (1440x900):**
- `desktop-home-de-gruvbox-{light,dark}`
- `desktop-recipes-de-gruvbox-{light,dark}`
- `desktop-journal-de-gruvbox-{light,dark}`
- `desktop-graph-de-gruvbox-{light,dark}`
- `desktop-settings-de-gruvbox-{light,dark}`
- `desktop-settings-encryption-de-gruvbox-{light,dark}`
- `desktop-settings-migration-de-gruvbox-{light,dark}`

**Mobile (393x852):**
- `mobile-home-de-gruvbox-{light,dark}` + `en-gruvbox-light`
- `mobile-recipes-de-gruvbox-{light,dark}` + `en-gruvbox-light`
- `mobile-journal-de-gruvbox-{light,dark}` + `en-gruvbox-light`
- `mobile-settings-de-gruvbox-{light,dark}` + `en-gruvbox-light`

Speicherort: `frontend/tests/e2e/ui-visual-regression.spec.ts-snapshots/`
