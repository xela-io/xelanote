# Design-Verbesserungen: Implementierungsplan

## Kontext

Die xelanote Web-App hat ein funktionales Gruvbox-basiertes Design-System mit Tailwind v4, OKLch-Farben und definierten Design-Tokens (`frontend/src/lib/design/tokens.ts`). Die Analyse hat Inkonsistenzen aufgedeckt: 51+ hardcodierte `border-radius`-Werte trotz vorhandener Tokens, zu geringer Sidebar-Kontrast, und UX-Polishing-Punkte.

Ziel: Visuell konsistenteres, professionelleres Erscheinungsbild bei minimalem Risiko.

> **Entscheidung**: Primary-Farbton bleibt theme-spezifisch (Blau im Light, Cyan/Gruen im Dark) - authentischer Gruvbox-Look hat Vorrang vor Brand-Einheitlichkeit.

---

## Phase 1: Border-Radius vereinheitlichen

**Problem**: 51+ hardcodierte Pixel-Werte (2px, 4px, 6px, 8px, 10px) verstreut ueber Komponenten, obwohl Token-System existiert.

**Mapping-Standard** (basierend auf `tokens.ts` borderRadius):

| Verwendung | Token | Wert | Tailwind |
|-----------|-------|------|----------|
| Mikro (Tags, Editor-Deko) | sm | 0.125rem (2px) | `rounded-sm` |
| Standard (Inputs, Buttons, Tabs) | base | 0.5rem (8px) | `rounded` |
| Mittel (Cards, Container) | md | 0.75rem (12px) | `rounded-md` |
| Gross (Login-Card, Modals) | lg | 1rem (16px) | `rounded-lg` |
| Pill (Badges) | full | 9999px | `rounded-full` |

**Batch 1a - Globale Styles** (`frontend/src/app.css`):
- `.cm-due-date` (Z.637): `2px` → `0.125rem`
- `.search-highlight` (Z.657): `2px` → `0.125rem`
- `.resize-handle` (Z.870): `2px` → `0.125rem`
- `.drag-handle` (Z.1039): `2px` → `0.125rem`
- `.task-drag-ghost` (Z.1068): `4px` → `0.5rem`
- `.task-drag-chosen` (Z.1073): `4px` → `0.5rem`

**Batch 1b - Login/Register-Seiten**:
- `frontend/src/routes/login/+page.svelte`:
  - `.login-card`: `8px` → `1rem`
  - `input`, `.login-button`, `.error-message`, `.back-button`: `4px` → `0.5rem`
  - `.info-message`, `.method-tabs`, `.method-tab`: `6px`/`4px` → `0.5rem`
- `frontend/src/routes/register/+page.svelte`: Selbes Muster

**Batch 1c - Komponenten**:
- `frontend/src/lib/components/Sidebar.svelte` (Z.987: drop-zone 4px → 0.5rem)
- `frontend/src/lib/components/Logo.svelte` (Z.111: badge 6px → 0.5rem)
- `frontend/src/lib/components/LanguageSelector.svelte` (Z.33)
- `frontend/src/lib/components/FolderTree.svelte` (Z.139, 170, 196)
- `frontend/src/lib/components/NoteItem.svelte` (Z.64)
- `frontend/src/lib/components/UnifiedTree.svelte` (Z.769, 793, 830, 899)
- `frontend/src/lib/components/VirtualizedTree.svelte` (Z.293)
- `frontend/src/lib/components/TableOfContents.svelte` (Z.129, 152, 166, 200)
- `frontend/src/lib/components/ColorPickerDialog.svelte` (Z.158)
- `frontend/src/lib/components/WebAuthnDeviceManager.svelte` (8 Stellen)
- `frontend/src/lib/components/SecurityKeyManager.svelte` (10 Stellen)
- `frontend/src/routes/about/+page.svelte` (5 Stellen)

**Aufwand**: ~45 Minuten, mechanische Ersetzungen

---

## Phase 2: Sidebar-Kontrast erhoehen

**Problem**: Sidebar-Hintergrund nur 2% Lightness-Differenz zum Hauptbereich (kaum wahrnehmbar).

**Loesung**: Differenz auf ~5% erhoehen.

**Datei**: `frontend/src/app.css`

| Theme | Variable | Aktuell | Neu |
|-------|----------|---------|-----|
| Light (`@theme` + `.theme-gruvbox-light`) | `--color-sidebar-background` | `oklch(95% 0.03 85)` | `oklch(92% 0.03 85)` |
| Dark (`.theme-gruvbox-dark`) | `--color-sidebar-background` | `oklch(20% 0.02 60)` | `oklch(17% 0.02 60)` |

**Aufwand**: ~5 Minuten, 3 Werte aendern (1x @theme, 1x light class, 1x dark class)

---

## Phase 3: UX-Polish (Loading, Logo, Backdrop)

### 3a: Branded Loading-State

**Datei**: `frontend/src/routes/+layout.svelte` (Z.424-437)

Generischen Border-Spinner ersetzen durch Logo-Komponente mit Pulse-Animation. Import `Logo` hinzufuegen (existiert bereits im Projekt, wird aber nicht in +layout.svelte importiert).

### 3b: Logo-Animation nur on-hover

**Datei**: `frontend/src/lib/components/Logo.svelte` (Z.67-76)

`animation-play-state: paused` als Default setzen, `running` nur bei `:hover`. Stoppt die permanent laufende 8s GPU-Animation.

### 3c: Mobile Backdrop-Blur

**Datei**: `frontend/src/routes/+layout.svelte` (Z.472)

`backdrop-blur-sm` zur Sidebar-Overlay-Klasse hinzufuegen (von `bg-black/50` zu `bg-black/50 backdrop-blur-sm`).

**Aufwand**: ~15 Minuten

---

## Phase 4: Scrollbar-Selector praezisieren

**Problem**: `html, body, * { scrollbar-width: thin !important; }` erzwingt duenne Scrollbars auf *jedem* DOM-Element inkl. Third-Party-Widgets.

**Loesung**: `*`-Wildcard entfernen, gezielt auf Scroll-Container anwenden. `!important` entfernen.

**Datei**: `frontend/src/app.css` (Z.93-103, 106-141)

Ersetze `*` durch spezifische Selektoren: `html, body`, plus `.overflow-y-auto`, `.overflow-x-auto`, `.cm-scroller` fuer CodeMirror. Ggf. eine Utility-Klasse `.thin-scrollbar` wo noetig.

**Aufwand**: ~15 Minuten

---

## Commit-Reihenfolge

1. `refactor(frontend): consolidate border-radius to design tokens` (Phase 1)
2. `refactor(frontend): increase sidebar background contrast` (Phase 2)
3. `style(frontend): improve loading state, logo animation, backdrop` (Phase 3)
4. `refactor(frontend): scope scrollbar styles to containers` (Phase 4)

Phase 1 zuerst (groesster Diff, wichtigster Impact), dann Phase 2 (schnell, gleiche Datei), dann 3+4 als Polish.

---

## Verifizierung

- `make test-frontend` nach jeder Phase
- Visueller Check: Login-Seite, Sidebar (auf/zu/collapsed), Editor, Settings, Dark/Light Toggle
- Mobile-Viewport testen (Sidebar-Drawer, Backdrop, Touch-Targets)
- `prefers-reduced-motion: reduce` testen (Logo-Animation muss deaktiviert bleiben)
- Kontrast-Check: Sidebar-Text auf neuem dunkleren Sidebar-Background gegen WCAG AA

---

## Bewusst zurueckgestellt

- **Sidebar-Code-Duplizierung** (~400 Zeilen Mobile/Desktop): Eigener PR, hohes Testrisiko
- **Login-Seite auf Tailwind migrieren**: Eigener PR nach Phase 1 (nutzt dann die Token-Werte)
- **Elevation-System einfuehren**: Wuerde Shadow-Tokens aus tokens.ts konsistent anwenden, aber breiter Scope
