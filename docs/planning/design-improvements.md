# Design-Verbesserungen: Implementierungsplan

## Kontext

Die xelanote Web-App hat ein funktionales Gruvbox-basiertes Design-System mit Tailwind v4, OKLch-Farben und definierten Design-Tokens (`frontend/src/lib/design/tokens.ts`). Die Codebase-Analyse hat aufgedeckt:

- **Fehlende Transitions**: Mobile Backdrop erscheint/verschwindet ohne Fade — harter visueller Bruch
- **Aktiver Bug**: Sidebar Drop-Zone nutzt undefinierte CSS-Variablen, rendert immer Blau statt Theme-Farbe
- **Inkonsistente Easings**: 3 verschiedene Easing-Kurven wild gemischt (`ease`, `cubic-bezier(0.4,0,0.2,1)`, Token-Werte)
- **Performance**: `transition-all` auf Sidebar-Collapse loest unnoetige Layout-Reflows aus
- 59 hardcodierte `border-radius`-Werte in 16 Dateien trotz vorhandener Tokens
- Sidebar-Kontrast nur 2% Lightness-Differenz zum Hauptbereich

**Priorisierung**: Erst was der User spuert (Fluidity, Bugs, Konsistenz), dann technische Hygiene (Token-Vereinheitlichung).

> **Entscheidung**: Primary-Farbton bleibt theme-spezifisch (Blau im Light, Cyan/Gruen im Dark) — authentischer Gruvbox-Look hat Vorrang vor Brand-Einheitlichkeit.

---

## Phase 1: Fluidity & Bug-Fixes

Hoechster User-Impact. Drei unabhaengige Fixes, zusammen ein Commit.

### 1a: Mobile Backdrop Fade-Transition

**Problem**: Das Sidebar-Overlay (`+layout.svelte:471-486`) erscheint und verschwindet **instantan** — `bg-black/50` ohne Transition. Die Sidebar selbst hat `transition-transform duration-200`, aber der Backdrop springt. Das ist der sichtbarste UX-Bruch auf Mobile.

**Datei**: `frontend/src/routes/+layout.svelte` (Z.471-486)

**Ist-Zustand**:
```svelte
{#if ui.getIsMobile() && ui.getSidebarOpen()}
  <div class="fixed inset-0 bg-black/50 z-40" ...></div>
{/if}
```

**Loesung**: Svelte `transition:fade` mit 200ms (passend zur Sidebar-Slide-Duration):

```svelte
{#if ui.getIsMobile() && ui.getSidebarOpen()}
  <div
    class="fixed inset-0 bg-black/50 z-40"
    transition:fade={{ duration: 200 }}
    ...
  ></div>
{/if}
```

Import `import { fade } from 'svelte/transition';` ergaenzen (falls nicht vorhanden).

> **Warum `transition:fade` statt CSS?** Das Overlay wird per `{#if}`-Block gerendert — CSS-Transitions greifen nicht bei DOM-Insert/Remove. Sveltes `transition:fade` handled genau diesen Fall: opacity 0→1 bei Mount, 1→0 bei Unmount, mit automatischem `outroend`-Event.

### 1b: Sidebar Drop-Zone Bug fixen

**Problem**: `Sidebar.svelte:995-1007` verwendet CSS-Variablen `--text-accent` und `--bg-accent-subtle`, die **nirgends definiert** sind. Die Fallback-Werte (`#3b82f6` = hardcoded Blau) greifen immer, unabhaengig vom aktiven Theme.

Zeile 1023 nutzt korrekt `var(--color-primary)` — die Drop-Zone-Styles darueber nicht.

**Datei**: `frontend/src/lib/components/Sidebar.svelte` (Z.991-1008)

**Ist-Zustand**:
```css
.drop-zone.active {
  border-color: var(--text-accent, #3b82f6);
  background: var(--bg-accent-subtle, rgba(59, 130, 246, 0.05));
}
.drop-zone.active .drop-zone-hint {
  color: var(--text-accent, #3b82f6);
}
```

**Soll-Zustand** (konsistent mit Z.1023):
```css
.drop-zone.active {
  border-color: var(--color-primary);
  background: color-mix(in oklch, var(--color-primary), transparent 90%);
}
.drop-zone.active .drop-zone-hint {
  color: var(--color-primary);
}
```

Ebenso `.drop-zone-hint` (Z.1001): `var(--text-muted, #6b7280)` → `var(--color-muted-foreground)` (korrekte Theme-Variable).

### 1c: `transition-all` durch spezifische Properties ersetzen

**Problem**: `Sidebar.svelte:319` nutzt `transition-all duration-200` fuer den Desktop-Collapsed-Zustand. `transition-all` transitioned **jede CSS-Property** und loest bei Breiten-Animationen Layout-Reflows fuer alle Properties aus — messbar langsamer als gezielte Transitions.

**Datei**: `frontend/src/lib/components/Sidebar.svelte` (Z.319)

**Ist-Zustand**:
```
'relative w-12 shrink-0 transition-all duration-200'
```

**Soll-Zustand**:
```
'relative w-12 shrink-0 transition-[width,opacity] duration-200'
```

Nur `width` und `opacity` aendern sich tatsaechlich beim Collapse. Tailwinds Arbitrary-Value-Syntax fuer `transition-property` ist hier korrekt, da Tailwind v4 kein Built-in fuer `transition-[width,opacity]` als Klasse hat.

> **Alternativ** (falls keine Arbitrary Values gewuenscht — siehe Phase 4 Guardrails): Inline-Style `style="transition-property: width, opacity; transition-duration: 200ms;"` verwenden. Konsistenz-Entscheidung beim Implementieren treffen.

**Aufwand Phase 1 gesamt**: ~15 Minuten

---

## Phase 2: Transition- und Easing-Konsistenz

**Problem**: Die App mixt drei verschiedene Easing-Kurven in Scoped CSS:

| Easing | Kurve | Verwendet in |
|--------|-------|-------------|
| `ease` (CSS keyword) | `cubic-bezier(0.25, 0.1, 0.25, 1)` | Logo.svelte, Sidebar.svelte, TableOfContents.svelte, WebAuthnDeviceManager.svelte, RecipeEditor.svelte, RecipeButton.svelte, JournalButton.svelte |
| `cubic-bezier(0.4, 0, 0.2, 1)` | Material Standard | DesktopTitleBar.svelte (einzige Stelle) |
| Kein Easing (implizit `ease`) | — | register/+page.svelte, login/+page.svelte, FolderTree.svelte, NoteItem.svelte, UnifiedTree.svelte |

Dazu inkonsistente Durations: `0.1s`, `0.15s`, `0.2s`, `0.3s`, `150ms` — ohne erkennbares System.

`tokens.ts` definiert `easing.default = cubic-bezier(0.4, 0, 0.2, 1)` und `animationDurations.fast = 150`, aber nur 4 Komponenten (Button, SidebarItem, SidebarButton, Section) importieren diese Werte.

**Loesung**: CSS Custom Properties fuer Easing und Duration in `app.css` definieren, damit Scoped CSS sie ohne JS-Import nutzen kann:

**Datei**: `frontend/src/app.css` (in `@theme`-Block oder als `:root`-Variablen)

```css
:root {
  --ease-default: cubic-bezier(0.4, 0, 0.2, 1);
  --ease-in: cubic-bezier(0.4, 0, 1, 1);
  --ease-out: cubic-bezier(0, 0, 0.2, 1);
  --ease-entrance: cubic-bezier(0.2, 0, 0, 1);
  --ease-exit: cubic-bezier(0.3, 0, 0.8, 0.15);
  --duration-fast: 150ms;
  --duration-base: 200ms;
  --duration-slow: 300ms;
}
```

> **Warum `:root` statt `@theme`?** Tailwind v4 `@theme` ist fuer Tailwind-Klassen-Generierung. Custom Easings und Durations werden hier in Scoped CSS (`<style>`-Bloecke) gebraucht, nicht als Tailwind-Utility-Klassen. `:root` macht sie ueberall verfuegbar ohne Side-Effects.

**Migration** (28 Stellen in Scoped CSS):

Alle `transition`-Deklarationen in Scoped `<style>`-Bloecken auf die neuen Custom Properties umstellen. Beispiele:

| Datei | Ist | Soll |
|-------|-----|------|
| `Sidebar.svelte:985` | `transition: all 0.15s ease` | `transition: all var(--duration-fast) var(--ease-default)` |
| `Sidebar.svelte:1036` | `transition: background-color 0.15s ease` | `transition: background-color var(--duration-fast) var(--ease-default)` |
| `Logo.svelte:49` | `transition: transform 0.3s ease` | `transition: transform var(--duration-slow) var(--ease-default)` |
| `UnifiedTree.svelte:836` | `transition: background 0.1s` | `transition: background var(--duration-fast) var(--ease-default)` |
| `UnifiedTree.svelte:873` | `transition: opacity 0.15s` | `transition: opacity var(--duration-fast) var(--ease-default)` |
| `TableOfContents.svelte:135` | `transition: all 0.15s ease` | `transition: all var(--duration-fast) var(--ease-default)` |
| `DesktopTitleBar.svelte:128` | `transition: background-color 150ms cubic-bezier(...)` | `transition: background-color var(--duration-fast) var(--ease-default)` |
| `login/+page.svelte:673` | `transition: background 0.2s` | `transition: background var(--duration-base) var(--ease-default)` |
| `register/+page.svelte:436` | `transition: background 0.2s` | `transition: background var(--duration-base) var(--ease-default)` |
| `FolderTree.svelte:163` | `transition: background-color 0.15s` | `transition: background-color var(--duration-fast) var(--ease-default)` |
| `NoteItem.svelte:65` | `transition: background-color 0.15s` | `transition: background-color var(--duration-fast) var(--ease-default)` |
| `WebAuthnDeviceManager.svelte` (4x) | `transition: all 0.15s ease` | `transition: all var(--duration-fast) var(--ease-default)` |
| `RecipeEditor.svelte:380` | `transition: all 0.15s ease` | `transition: all var(--duration-fast) var(--ease-default)` |
| `RecipeButton.svelte:50` | `transition: all 0.15s ease` | `transition: all var(--duration-fast) var(--ease-default)` |
| `JournalButton.svelte:50` | `transition: all 0.15s ease` | `transition: all var(--duration-fast) var(--ease-default)` |
| `about/+page.svelte:420` | `transition: background 0.2s` | `transition: background var(--duration-base) var(--ease-default)` |
| `RecipeImageGallery.svelte` (2x) | `transition: opacity 0.15s` | `transition: opacity var(--duration-fast) var(--ease-default)` |
| `SummaryPanel.svelte:188` | `transition: all 150ms ease` | `transition: all var(--duration-fast) var(--ease-default)` |
| `Editor.svelte:1047` | `transition: background-color 0.15s ease` | `transition: background-color var(--duration-fast) var(--ease-default)` |

> **Vorteil**: Ein einziger Ort zum Tunen aller Transitions. Wenn die App sich "traeger" anfuehlt, `--duration-fast` von 150ms auf 100ms aendern — wirkt sofort ueberall.

**Hinweis zu `transition: all`**: Bei dieser Migration `all` beibehalten wo es steht — das Ersetzen durch spezifische Properties ist ein eigener Schritt (aehnlich Phase 1c), der mehr Kontext pro Komponente erfordert. Hier geht es nur um Easing/Duration-Konsistenz.

**Aufwand**: ~30 Minuten mechanische Ersetzungen + ~15 Minuten visueller QA

> **QA-Hinweis**: Die Easing-Kurve aendert sich von `ease` (`cubic-bezier(0.25, 0.1, 0.25, 1)`) auf `cubic-bezier(0.4, 0, 0.2, 1)` (schnellerer Start, langsamerer Auslauf). Bei 150ms-Transitions ist der Unterschied kaum wahrnehmbar, bei 300ms (Logo.svelte:49) kann er spuerbar sein. Deshalb: Alle Komponenten mit `--duration-slow` (300ms) einzeln im Browser pruefen — Hover-Feeling und Focus-Transitions muessen sich natuerlich anfuehlen. Bei den 150ms-Stellen reicht ein Stichproben-Check (2-3 Buttons/Tree-Items).

---

## Phase 3: Sidebar-Kontrast erhoehen

**Problem**: Sidebar-Hintergrund nur 2% Lightness-Differenz zum Hauptbereich (kaum wahrnehmbar).

**Loesung**: Differenz auf ~5% erhoehen.

**Datei**: `frontend/src/app.css`

| Theme | Variable | Aktuell | Neu |
|-------|----------|---------|-----|
| Light (`@theme` Z.190 + `.theme-gruvbox-light` Z.234) | `--color-sidebar-background` | `oklch(95% 0.03 85)` | `oklch(92% 0.03 85)` |
| Dark (`.theme-gruvbox-dark` Z.271) | `--color-sidebar-background` | `oklch(20% 0.02 60)` | `oklch(17% 0.02 60)` |

> **WCAG-Pflichtpruefung VOR Commit**: Kontrast zwischen neuem Sidebar-Background und Sidebar-Foreground berechnen:
> - Light: `oklch(92% 0.03 85)` (bg) gegen `oklch(38% 0.08 60)` (fg) → muss >= 4.5:1 sein
> - Dark: `oklch(17% 0.02 60)` (bg) gegen `oklch(72% 0.12 155)` (fg) → muss >= 4.5:1 sein
>
> Falls Kontrast unzureichend: Lightness-Werte iterativ anpassen (z.B. Dark nur auf 18% statt 17%).

**Aufwand**: ~5 Minuten Aenderung + 10 Minuten WCAG-Kontrast-Validierung, 3 Werte aendern (1x @theme, 1x light class, 1x dark class)

---

## Phase 4: Border-Radius vereinheitlichen

**Problem**: 59 hardcodierte Pixel-Werte (2px, 3px, 4px, 6px, 8px, 10px, 12px) verstreut ueber 16 Dateien, obwohl Tailwind v4 ein vollstaendiges Radius-Token-System mitbringt.

**Mapping-Standard** (Tailwind v4 Built-in-Defaults, **keine `@theme`-Aenderung noetig**):

| px-Wert | CSS Variable (Tailwind) | Tailwind-Klasse | Bemerkung |
|---------|------------------------|-----------------|-----------|
| 2px | `var(--radius-xs)` (0.125rem) | `rounded-xs` | Exakt |
| 3px | `var(--radius-sm)` (0.25rem=4px) | `rounded-sm` | +1px, bewusste Anpassung |
| 4px | `var(--radius-sm)` (0.25rem) | `rounded-sm` | Exakt |
| 6px | `var(--radius-md)` (0.375rem) | `rounded-md` | Exakt |
| 8px | `var(--radius-lg)` (0.5rem) | `rounded-lg` | Exakt |
| 10px | `var(--radius-xl)` (0.75rem=12px) | `rounded-xl` | +2px, bewusste Anpassung |
| 12px | `var(--radius-xl)` (0.75rem) | `rounded-xl` | Exakt |
| 16px | `var(--radius-2xl)` (1rem) | `rounded-2xl` | Exakt |
| 24px | `var(--radius-3xl)` (1.5rem) | `rounded-3xl` | Exakt |
| 9999px | `calc(infinity * 1px)` | `rounded-full` | Tailwind v4 Konvention |

> **Warum Tailwind-Defaults nutzen statt eigene `@theme`-Variablen?**
>
> Die Codebase hat bereits **381 `rounded-*` Klassen in 71 Dateien** (204x `rounded-lg`, 174x `rounded-md`), die alle auf Tailwinds Default-Skala basieren. Eigene `--radius-*` Werte in `@theme` zu definieren wuerde diese Defaults ueberschreiben und eine **massive visuelle Regression** ausloesen (z.B. alle `rounded-lg` von 8px auf 16px).
>
> **Single Source of Truth** ist damit Tailwinds eingebaute Radius-Skala. In CSS-Styles `var(--radius-*)` verwenden, in Templates Tailwind-Klassen.

**Batch 4a — Globale Styles** (`frontend/src/app.css`, 6 Stellen):
- `.cm-due-date` (Z.637): `2px` → `var(--radius-xs)`
- `.search-highlight` (Z.657): `2px` → `var(--radius-xs)`
- `.resize-handle` (Z.870): `2px` → `var(--radius-xs)`
- `.drag-handle` (Z.1039): `2px` → `var(--radius-xs)`
- `.task-drag-ghost` (Z.1068): `4px` → `var(--radius-sm)`
- `.task-drag-chosen` (Z.1073): `4px` → `var(--radius-sm)`

Hinweis: Z.117 und Z.136 (`border-radius: 4px` in Scrollbar-Thumb-Styles) gehoeren zu **Phase 6**, nicht Phase 4.

**Batch 4b — Login/Register-Seiten**:
- `frontend/src/routes/login/+page.svelte` (8 Stellen):
  - `.login-card`: `8px` → `var(--radius-lg)` (8px, exakt)
  - `input`, `.login-button`, `.error-message`, `.back-button`: `4px` → `var(--radius-sm)` (4px, exakt)
  - `.info-message`: `6px` → `var(--radius-md)` (6px, exakt)
  - `.method-tabs`, `.method-tab`: `4px` → `var(--radius-sm)` (4px, exakt)
  - `.login-card` Shadow: `box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1)` → `box-shadow: var(--shadow-md, 0 4px 6px rgba(0,0,0,0.1))` (Theme-aware mit Fallback)
- `frontend/src/routes/register/+page.svelte` (5 Stellen): Selbes Muster

> **Warum Shadows mitfixen?** Wenn wir die Login-Seite ohnehin anfassen, sollten die hardcoded `rgba()`-Shadows zumindest einen CSS-Variable-Fallback bekommen. Im Dark Mode ist `rgba(0,0,0,0.1)` auf dunklem Hintergrund praktisch unsichtbar.

**Batch 4c — Komponenten** (alle Werte explizit mit Ziel-Token):
- `Sidebar.svelte` (Z.983): `4px` → `var(--radius-sm)`
- `Logo.svelte` (Z.111): `6px` → `var(--radius-md)`
- `LanguageSelector.svelte` (Z.33): `4px` → `var(--radius-sm)`
- `FolderTree.svelte` (3 Stellen):
  - Z.162: `4px` → `var(--radius-sm)`
  - Z.193: `2px` → `var(--radius-xs)`
  - Z.219: `10px` → `var(--radius-xl)` (+2px, visuelle Anpassung)
- `NoteItem.svelte` (Z.64): `4px` → `var(--radius-sm)`
- `UnifiedTree.svelte` (4+1 Stellen):
  - Z.772: `1.5px` → `var(--radius-xs)` (2px, +0.5px — dekorative Color-Bar, visuell unmerklich)
  - Z.809: `3px` → `var(--radius-sm)` (+1px)
  - Z.833: `4px` → `var(--radius-sm)`
  - Z.870: `10px` → `var(--radius-xl)` (+2px)
  - Z.939: `3px` → `var(--radius-sm)` (+1px)
- `VirtualizedTree.svelte` (Z.293): `4px` → `var(--radius-sm)`
- `TableOfContents.svelte` (4 Stellen):
  - Z.129: `6px` → `var(--radius-md)`
  - Z.152: `9999px` → `calc(infinity * 1px)` oder `rounded-full`
  - Z.166: `8px` → `var(--radius-lg)`
  - Z.200: `4px` → `var(--radius-sm)`
- `ColorPickerDialog.svelte` (Z.158): `4px` → `var(--radius-sm)`
- `DesktopTitleBar.svelte` (Z.159): `4px` → `var(--radius-sm)`
- `WebAuthnDeviceManager.svelte` (6 Stellen): Werte bei Implementierung pruefen, Mapping gemaess Tabelle
- `SecurityKeyManager.svelte` (9 Stellen): Werte bei Implementierung pruefen, Mapping gemaess Tabelle
- `about/+page.svelte` (5 Stellen):
  - Z.206: `12px` → `var(--radius-xl)`
  - Z.274, 344, 385: `8px` → `var(--radius-lg)`
  - Z.417: `6px` → `var(--radius-md)`

**Aufwand**: ~45-60 Minuten, ueberwiegend mechanische Ersetzungen

---

## Phase 5: UX-Polish (Loading, Logo, Backdrop-Blur)

### 5a: Branded Loading-State

**Datei**: `frontend/src/routes/+layout.svelte` (Z.425-437)

Generischen Border-Spinner ersetzen durch Logo mit Pulse-Animation.

**Ist-Zustand**:
```svelte
<div class="animate-spin rounded-full h-12 w-12 border-b-2 mx-auto mb-4"
     style="border-color: var(--color-muted-foreground, #666);"></div>
<p style="color: var(--color-muted-foreground, #888);">{$_('common.loading')}</p>
```

**Loesung**: Leichtgewichtige statische Variante direkt im Layout:
- Inline-SVG des Logos (kein Komponenten-Import noetig)
- Reine CSS `pulse`-Animation (kein `gradient-shift`, kein Canvas/GPU)

> **Kein Import von Logo.svelte in +layout.svelte**: `+layout.svelte` ist das globale Layout und wird auf **jeder** Seite geladen. `Logo.svelte` hat eine permanent laufende 8s `gradient-shift`-Animation, die im Loading-Kontext ueberdimensioniert ist. Ein leichtgewichtiges Inline-SVG mit `pulse` reicht voellig aus.
>
> **Fallback-Farbe beachten**: Der aktuelle `background-color: var(--color-background, #111)` flasht dunkel bei Light-Theme-Usern. Besser: `#111` durch `oklch(97% 0.03 85)` ersetzen (Light-Default) oder `color-scheme: light dark` setzen, damit der Browser einen passenden Default waehlt.

### 5b: Logo-Animation nur on-hover (Desktop) / reduziert (Touch)

**Datei**: `frontend/src/lib/components/Logo.svelte` (Z.57-86)

**Loesung**:

```css
/* Desktop: nur bei Hover animieren */
@media (hover: hover) {
  .logo-text {
    animation-play-state: paused;
  }
  .logo-wrapper:hover .logo-text {
    animation-play-state: running;
    animation-duration: 3s; /* bestehendes Hover-Speedup beibehalten */
  }
}

/* Touch: langsame Endlos-Animation (dezent, nicht ablenkend) */
@media (hover: none) {
  .logo-text {
    animation-duration: 16s; /* doppelt so lang wie Default, sehr subtil */
  }
}
```

`prefers-reduced-motion: reduce` greift weiterhin als Override (Z.148: `animation: none;` — bereits korrekt implementiert).

> **Begruendung Touch-Strategie**: Reines `:hover` wuerde Touch-Usern die Animation komplett entziehen. Eine langsame 16s-Animation ist ein guter Kompromiss: visuell lebendig, aber so dezent dass sie nicht ablenkt. Die bestehende `prefers-reduced-motion`-Regel respektiert User, die explizit keine Animationen wollen.

### 5c: Mobile Backdrop-Blur

**Datei**: `frontend/src/routes/+layout.svelte` (Z.473)

Von `bg-black/50` zu `bg-black/40 backdrop-blur-md`:

> **Bewusste Entscheidung: `backdrop-blur-md` (8px) statt `backdrop-blur-sm` (4px)**: Die einzige bestehende Backdrop-Blur-Nutzung in der App (`TableOfContents.svelte:131`) verwendet `blur(8px)`. Um Konsistenz zu wahren, denselben Wert nutzen. Dazu Opacity von 50% auf 40% reduzieren — Blur uebernimmt einen Teil der visuellen Abgrenzung, weniger Abdunkelung noetig.

> **Performance-Hinweis**: `backdrop-blur` ist GPU-intensiv auf Mobile, besonders auf aelteren Android-Geraeten (< 2020). Auf Low-End-Geraeten testen. Falls Ruckler auftreten: `@media (prefers-reduced-motion: reduce)` Fallback:
> ```css
> @media (prefers-reduced-motion: reduce) {
>   .backdrop-overlay { backdrop-filter: none; background: rgba(0,0,0,0.5); }
> }
> ```

**Aufwand Phase 5 gesamt**: ~25 Minuten (inkl. SVG-Erstellung fuer 5a)

---

## Phase 6: Scrollbar-Scoping und Cleanup

**Problem**: `html, body, * { scrollbar-width: thin !important; }` erzwingt duenne Scrollbars auf *jedem* DOM-Element inkl. Third-Party-Widgets (CodeMirror-Tooltips, Autocomplete-Popups, Modals).

**Datei**: `frontend/src/app.css` (Z.93-161)

**Ist-Zustand** — globales Scoping betrifft **beide** Engines:

| Engine | Selektor | Zeilen | Global? |
|--------|----------|--------|---------|
| Firefox/Edge | `html, body, * { scrollbar-width: thin !important }` | Z.93-97 | Ja (`*`) |
| WebKit (Light) | `::-webkit-scrollbar`, `::-webkit-scrollbar-track`, `::-webkit-scrollbar-thumb` | Z.125-141 | **Ja** (kein Scope) |
| WebKit (Dark) | `.theme-gruvbox-dark ::-webkit-scrollbar*` | Z.106-122 | Nein (gescoped) |

**Loesung**: Explizite Opt-in-Klasse `.thin-scrollbar` fuer **beide** Engines:

Firefox:
- `html, body, .thin-scrollbar { scrollbar-width: thin; }` (ohne `!important`)

WebKit Light (Z.125-141):
- `::-webkit-scrollbar` → `.thin-scrollbar::-webkit-scrollbar` (analog fuer `-track`, `-thumb`, `-thumb:hover`)

WebKit Dark (Z.106-122):
- `.theme-gruvbox-dark ::-webkit-scrollbar` → `.theme-gruvbox-dark .thin-scrollbar::-webkit-scrollbar`

`.thin-scrollbar` manuell an Scroll-Containern setzen:
- `html, body` (bewahrt aktuelles Verhalten: Root-Viewport hat immer duenne Scrollbars)
- `.cm-scroller` (CodeMirror)
- Sidebar-Container, Main-Content-Area

**Zusaetzlich: Doppelte Scrollbar-Hide-Utilities konsolidieren**

`app.css` definiert zwei semantisch identische Utilities:
- `.scrollbar-none` (Z.143-160): `scrollbar-width: none !important` + WebKit-Hiding
- `.scrollbar-hide` (Z.1155-1167): `scrollbar-width: none` + WebKit-Hiding + iOS-Smooth-Scrolling

Konsolidieren auf **eine** Utility-Klasse (`.scrollbar-none` beibehalten, da Tailwind-Konvention). iOS-Smooth-Scrolling (`-webkit-overflow-scrolling: touch`) aus `.scrollbar-hide` uebernehmen. Alle Stellen, die `.scrollbar-hide` verwenden, auf `.scrollbar-none` umstellen.

> Hinweis: Z.1169-1172 (`[style*='scrollbar-width: none']::-webkit-scrollbar`) ist ein Inline-Style-Workaround fuer die Settings-Seite. Bei der Konsolidierung pruefen, ob die betroffene Stelle auf `.scrollbar-none` umgestellt werden kann — dann kann dieser Selektor entfallen.

**Aufwand**: ~25 Minuten (inkl. manuellem Hinzufuegen der Klasse an 4-5 Container-Elementen + Konsolidierung)

---

## Commit-Reihenfolge

1. `fix(frontend): add backdrop transition, fix drop-zone theme vars, optimize sidebar transition` (Phase 1)
2. `refactor(frontend): unify transition easings and durations via CSS custom properties` (Phase 2)
3. `style(frontend): increase sidebar background contrast` (Phase 3)
4. `refactor(frontend): replace hardcoded border-radius with design tokens` (Phase 4a + 4b + 4c)
5. `style(frontend): branded loading state, logo animation strategy, backdrop blur` (Phase 5)
6. `refactor(frontend): scope scrollbar styles and consolidate utilities` (Phase 6)

Phase 1 ist ein einzelner Commit, weil alle drei Fixes unabhaengig und klein sind. Phase 4 kann bei Bedarf in 2 Commits gesplittet werden (4a+4b und 4c), aber die Aenderungen sind rein mechanisch und risikoarm.

---

## Verifizierung

**Automatisierte Tests:**
- `make test-frontend` nach jeder Phase
- `make test-e2e` nach Phase 1 (Sidebar-Transition), Phase 4b (Login-Seite), Phase 5a (Loading-State)

**Manuelle Checks:**
- **Phase 1**: Sidebar auf Mobile oeffnen/schliessen — Backdrop muss sanft ein-/ausfaden. Drop-Zone muss Theme-Farbe zeigen (nicht hardcoded Blau). Sidebar-Collapse auf Desktop darf nicht ruckeln.
- **Phase 2**: Alle interaktiven Elemente (Buttons, Tree-Items, Links) muessen sich gleich "anfuehlen" — selbe Easing-Kurve, konsistente Speed.
- **Phase 3**: Sidebar visuell klar vom Hauptbereich abgesetzt, Dark/Light Toggle pruefen. Vorher/Nachher-Screenshots.
- **Phase 4**: Visueller Check Login-Seite, Sidebar, Editor, Settings — keine sichtbaren Radius-Sprünge.
- **Phase 5**: Loading-State zeigt Logo, nicht Spinner. Logo-Animation auf Desktop nur bei Hover, auf Touch langsam laufend. `prefers-reduced-motion: reduce` deaktiviert Animation. Backdrop-Blur auf Low-End-Android testen.
- **Phase 6**: Scrollbars in Editor, Sidebar, Modals, Dropdowns pruefen. Third-Party-Widgets (CodeMirror-Tooltips) duerfen nicht mehr pauschal duenne Scrollbars haben.

**Pflicht vor Commit:**
- WCAG-Kontrast-Check Phase 3: **vorberechnen**, nicht nur visuell pruefen
- Vorher/Nachher-Screenshots fuer Phase 3 und Phase 4 (Login/Register)

---

## Guardrails gegen Regressionen

**Border-Radius Lint-Regel** (nach Phase 4):
- **Jede** harte `border-radius`-Literalform verhindern — `px`, `rem` und Tailwind Arbitrary Values (`rounded-[...]`).
- Erlaubt: `var(--radius*)`, Tailwind-Klassen (`rounded-xs` bis `rounded-full`, `rounded-none`), explizite Whitelist fuer Legacy/Third-Party.
- Regex: `border-radius:\s*[\d.]+(px|rem|em)` (CSS) und `rounded-\[` (Tailwind)

**Transition Lint-Regel** (nach Phase 2):
- Hardcoded Easing-Werte (`ease`, `ease-in-out`, `cubic-bezier(...)`) und Durations (`0.15s`, `150ms`, `0.2s`) in Scoped CSS verhindern.
- Erlaubt: `var(--ease-*)`, `var(--duration-*)`, Tailwind-Klassen (`duration-150`, `ease-out`, etc.)
- **Stylelint statt Regex**: Eine reine Regex (`transition:.*\b(ease|...)\b`) erzeugt False Positives bei komplexen Shorthands, Tailwind-generierten Styles und Template-Inline-Styles. Besser: Stylelint mit `stylelint-declaration-strict-value` Plugin, das `transition-timing-function` und `transition-duration` auf Custom Properties einschraenkt. Alternativ: Regex nur auf `<style>`-Bloecke scopen (z.B. via `svelte-check` Plugin oder Pre-Commit-Hook mit `awk`-Extraktion der Style-Sektionen).

---

## Voraussetzungen (Blocking, vor Implementierungsbeginn)

1. **Tailwind v4 Radius-Variablen verifizieren**: Sicherstellen, dass `--radius-xs` bis `--radius-3xl` im Build vorhanden sind. **Build-Zeit-Check** (nicht manuell im Browser): `npx tailwindcss --content '' --output /dev/stdout 2>/dev/null | grep 'radius-xs'` — wenn die Variable im generierten CSS auftaucht, ist sie verfuegbar. Alternativ: Tailwind-Config auf Custom-`borderRadius`-Overrides pruefen. Wenn `var(--radius-sm)` ohne Fallback nicht aufloest, rendert der Browser `border-radius: unset` (= 0) — das waere sofort sichtbar, aber trotzdem sollte der Check **vor** Phase 4 erfolgen, nicht erst beim visuellen QA.
2. **WCAG-Kontrast fuer Phase 3 vorberechnen** (siehe Phase 3 Abschnitt).
3. **Svelte `transition:fade` Import pruefen**: Sicherstellen dass `+layout.svelte` den Import unterstuetzt ohne Circular-Dependencies.

---

## Bewusst zurueckgestellt

- **Sidebar-Code-Duplizierung** (~600 Zeilen Mobile/Desktop): Eigener PR, hohes Testrisiko
- **Login-Seite komplett auf Tailwind migrieren**: Eigener PR nach Phase 4 (nutzt dann die Token-Werte)
- **Elevation-System einfuehren**: Wuerde Shadow-Tokens konsistent anwenden, aber breiter Scope. Login-Shadows in Phase 4b bekommen vorerst CSS-Variable-Fallbacks.
- **`tokens.ts` konsolidieren**: 8 von 10 Token-Kategorien sind ungenutzt (`typography`, `spacing`, `shadows`, `borderRadius`, `zIndex`, `buttonSizes`, `focusRing`, `transitions`). Optionen: (A) Schrittweise Adoption erhoehen, (B) Ungenutzte Exports entfernen und auf CSS Custom Properties als Single Source of Truth setzen. Entscheidung separat treffen — betrifft die gesamte Component-Architektur.
- **`transition: all` durch spezifische Properties ersetzen** (ausser Phase 1c): Erfordert pro Komponente eine Analyse, welche Properties sich tatsaechlich aendern. Lohnt sich als eigener Refactoring-Pass nach Phase 2.
- **`UnifiedTree.svelte`: Doppelte `@media (pointer: coarse)`-Bloecke konsolidieren** (Z.970-984 und Z.987-1016): Funktional korrekt aber redundant. Bei naechster Tree-Aenderung mitfixen.
