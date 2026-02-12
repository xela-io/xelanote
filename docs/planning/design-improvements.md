# Design-Verbesserungen: Implementierungsplan

## Review-Status

> **Letzter Review**: 2026-02-12 — Codebase-Verifikation und State-of-the-Art-Abgleich durchgefuehrt. Alle Behauptungen gegen den tatsaechlichen Code geprueft. Technologie-Empfehlungen gegen 2026-Best-Practices validiert. Ergebnisse und Korrekturen sind im Plan eingearbeitet (markiert mit `[REVIEW]` fuer Code-Verifikation, `[2026]` fuer State-of-the-Art-Korrekturen).

## Kontext

Die xelanote Web-App hat ein funktionales Gruvbox-basiertes Design-System mit Tailwind v4, OKLch-Farben und definierten Design-Tokens (`frontend/src/lib/design/tokens.ts`). Die Codebase-Analyse hat aufgedeckt:

- **Fehlende Transitions**: Mobile Backdrop erscheint/verschwindet ohne Fade — harter visueller Bruch
- **Aktiver Bug**: Sidebar Drop-Zone nutzt undefinierte CSS-Variablen, rendert immer Blau statt Theme-Farbe
- **Inkonsistente Easings**: 3-4 verschiedene Easing-Kurven wild gemischt (`ease`, `cubic-bezier(0.4,0,0.2,1)`, `linear`, implizit/kein Easing)
- **Performance**: `transition-all` auf Sidebar-Collapse loest unnoetige Layout-Reflows aus (auch in `UnlockEncryptionModal.svelte:213`)
- ~56 hardcodierte `border-radius`-Werte in ~20 Dateien trotz vorhandener Tokens `[REVIEW: korrigiert von 59/16]`
- Sidebar-Kontrast nur 2% Lightness-Differenz zum Hauptbereich

**Priorisierung**: Erst was der User spuert (Fluidity, Bugs, Konsistenz), dann technische Hygiene (Token-Vereinheitlichung).

> **Entscheidung**: Primary-Farbton bleibt theme-spezifisch (Blau im Light, Cyan/Gruen im Dark) — authentischer Gruvbox-Look hat Vorrang vor Brand-Einheitlichkeit.

### Bekannte Blocker

> **BLOCKER Phase 4**: Die gestaffelten Tailwind-v4-Radius-Variablen (`--radius-xs`, `--radius-sm`, etc.) existieren **nicht** im aktuellen Build. In `app.css` `@theme` ist nur `--radius: 0.5rem;` definiert (Zeile 200). Phase 4 kann nicht wie geplant umgesetzt werden — siehe Details im Phase-4-Abschnitt.

---

## PR-Strategie

Drei PRs mit steigendem Regressionsrisiko. Jeder PR ist eigenstaendig mergebar — kein PR haengt von einem anderen ab.

| PR | Phasen | Thema | Risiko | Commits |
|----|--------|-------|--------|---------|
| **A: UX-Fixes** | 1a, 1b, 3, 5a, 5b, 5c | Sofort spuerbarer User-Impact: Transitions, Bug-Fixes, Polish | Niedrig | 5 |
| **B: Motion/Token-System** | 1c, 2 + Transition-Guardrails | Technische Konsistenz: Easing/Duration als CSS Custom Properties | Mittel | 3 |
| **C: Radius + Scrollbar-Refactor** | 4, 6 + Radius-Guardrails | Style-Hygiene: 56 hardcoded Radii + Scrollbar-Scoping | Hoch | 5 |

**Reihenfolge**: A → B → C (empfohlen, aber nicht zwingend).

> **Warum Phase 1c in PR B?** `transition-all` durch spezifische Properties ersetzen ist thematisch ein Transition-Refactor, nicht ein UX-Fix. Ausserdem beruehrt Phase 2 (PR B) dieselben Zeilen — in einem PR vermeidet das doppeltes Anfassen.

---

## Phase 1: Fluidity & Bug-Fixes

Hoechster User-Impact. Drei unabhaengige Fixes. 1a und 1b gehoeren zu **PR A** (UX-Fixes), 1c zu **PR B** (Motion/Token-System — beruehrt dieselben Zeilen wie Phase 2).

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
>
> `[2026]` **Bestaetigt**: `transition:fade` ist in Svelte 5 (2026) weiterhin der empfohlene Ansatz fuer Komponenten-Ein/Austrittsanimationen. Svelte 5 nutzt intern die Web Animation API, was Performance und Kompatibilitaet verbessert. Die View Transitions API (seit Oktober 2025 cross-browser stabil) ist fuer **Seiten-Navigation** gedacht, nicht fuer Komponenten-Animationen innerhalb einer Seite.

### 1b: Sidebar Drop-Zone Bug fixen

**Problem**: `Sidebar.svelte:995-1007` verwendet CSS-Variablen `--text-accent` und `--bg-accent-subtle`, die **nirgends definiert** sind. Die Fallback-Werte (`#3b82f6` = hardcoded Blau) greifen immer, unabhaengig vom aktiven Theme.

Zeile ~1030 nutzt korrekt `var(--color-primary)` (in `.drop-zone.touch-drop-into`) — die Drop-Zone-Styles darueber nicht. `[REVIEW: Zeilennummer korrigiert von 1023 auf ~1030]`

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

**Soll-Zustand** (konsistent mit Z.~1030):
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

> `[REVIEW]` **Browser-Kompatibilitaet**: `color-mix(in oklch, ...)` hat ~93% Browser-Support (caniuse.com). Fuer die restlichen ~7% einen Fallback mitliefern:
> ```css
> background: rgba(59, 130, 246, 0.05); /* Fallback */
> background: color-mix(in oklch, var(--color-primary), transparent 90%);
> ```

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

**Aufwand Phase 1 gesamt**: ~30-45 Minuten `[REVIEW: korrigiert von 15 Min — inkl. Build-Verifikation, manueller Mobile-Check, Browser-Fallback-Test]`

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

**Datei**: `frontend/src/app.css`

`[2026]` **`@theme` statt `:root` verwenden**: Tailwind v4 generiert aus `@theme`-Eintraegen automatisch sowohl Utility-Klassen als auch CSS Custom Properties. Damit stehen die Variablen in Scoped CSS (`<style>`-Bloecke) **und** als Tailwind-Klassen zur Verfuegung. Es gibt keinen Grund mehr fuer `:root` — `@theme` ist die Single Source of Truth in Tailwind v4.

Zusaetzlich: `@property`-Deklarationen fuer strukturierte Defaults und Typ-Sicherheit (100% Browser-Support seit Juli 2024):

```css
/* Typ-Deklarationen (optional, aber Best Practice 2026) */
/* Hinweis: --ease-default nutzt syntax: '*' weil CSS keinen <easing-function>-Typ
   fuer @property kennt. Damit gibt es hier KEINE echte Typ-Validierung — der Nutzen
   liegt nur im initial-value als eingebautem Fallback. Die Duration-Properties
   profitieren hingegen voll von der <time>-Validierung. */
@property --ease-default {
  syntax: '*';
  inherits: true;
  initial-value: cubic-bezier(0.4, 0, 0.2, 1);
}
@property --duration-fast {
  syntax: '<time>';
  inherits: true;
  initial-value: 150ms;
}
@property --duration-base {
  syntax: '<time>';
  inherits: true;
  initial-value: 200ms;
}
@property --duration-slow {
  syntax: '<time>';
  inherits: true;
  initial-value: 300ms;
}

/* Token-Definitionen */
@theme {
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

> `[2026]` **Warum `@property`?** Seit Juli 2024 100% Browser-Support. Vorteile: (1) **Echte Typ-Validierung fuer Duration-Properties** (`<time>`) — der Browser warnt bei falschen Werten und ermoeglicht Transitions auf den Custom Properties selbst (z.B. dynamisches Verlangsamen), (2) `initial-value` als eingebauter Fallback fuer alle Properties. **Einschraenkung**: Fuer Easing-Properties (`--ease-default` etc.) existiert kein passender CSS-Typ — `syntax: '*'` validiert effektiv nicht, liefert aber immerhin den Fallback-Wert. **Kein Muss fuer Phase 2**, aber Best Practice 2026 und zukunftssicher. Kann als Follow-up ergaenzt werden.
>
> `[2026]` **`linear()` Easing-Funktion**: Seit 2024 in Chrome/Firefox verfuegbar, wird im Juni 2026 Baseline Widely Available. Ermoeglicht komplexe Easing-Kurven (Bounce, Spring), die mit `cubic-bezier()` nicht moeglich sind. Fuer den aktuellen Anwendungsfall (Standard-UI-Transitions) ist `cubic-bezier(0.4, 0, 0.2, 1)` weiterhin korrekt. `linear()` vormerken fuer zukuenftige Animationen (z.B. Drag-and-Drop-Feedback, Notification-Bounce).

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

**Hinweis zu `transition: all`**: `[2026]` ~~Bei dieser Migration `all` beibehalten wo es steht.~~ **Korrektur**: `transition: all` wird in 2026 ausdruecklich als Anti-Pattern betrachtet — es verursacht unbeabsichtigte Transitions bei Code-Aenderungen und Performance-Probleme. **Empfehlung**: Wenn eine Stelle mit `transition: all` ohnehin angefasst wird, die tatsaechlich animierten Properties identifizieren und explizit angeben. Das ist minimal mehr Aufwand pro Stelle (~1 Min Analyse), aber verhindert zukuenftige Regressions. Falls der Aufwand fuer alle `all`-Stellen zu gross ist: Mindestens die haeufig gerenderten Komponenten (Sidebar, Tree-Items, Buttons) fixen, Rest als Follow-up.

> `[REVIEW]` **`tokens.ts`-Konflikt aufloesen**: `tokens.ts` definiert `easing.default` und `animationDurations.fast` — dieselben Werte, die jetzt als CSS Custom Properties existieren werden. 4 Komponenten importieren diese Werte (Button.svelte, SidebarItem.svelte, SidebarButton.svelte, Section.svelte) plus `animations.ts`. Diese Imports muessen im Rahmen von Phase 2 auf die neuen CSS Custom Properties umgestellt werden, sonst gibt es **zwei konkurrierende Quellen** fuer dieselben Werte. Konkret:
> - In den 4 Komponenten: JS-Import `tokens.easing.default` durch CSS `var(--ease-default)` ersetzen
> - In `animations.ts`: Pruefen ob die Werte dort programmatisch benoetigt werden (Svelte `animate:`/`transition:` Direktiven mit JS-Parametern) — falls ja, koennen die CSS-Variablen per `getComputedStyle` gelesen werden, oder `tokens.ts` verweist auf dieselben Literal-Werte wie die CSS-Properties
>
> **Entscheidung (fixiert)**: Nach Abschluss von Phase 2 wird `frontend/src/lib/design/tokens.ts` entfernt. Akzeptanzkriterium: `grep -R "lib/design/tokens" frontend/src` liefert 0 Treffer. Falls `animations.ts` weiterhin programmatische Werte braucht, dort `getComputedStyle(document.documentElement)` als Quelle verwenden statt einer zweiten Token-Datei.

**Aufwand**: ~60-90 Minuten `[REVIEW: korrigiert von 45 Min — 28 Stellen sind nicht rein mechanisch, jede Duration-Kategorie (fast/base/slow) muss zur Intent passen. Dazu kommen die 4+1 tokens.ts-Migrationen und visueller QA]`

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

> `[REVIEW]` **Kontrastwerte MUESSEN vor Implementierung berechnet werden** — nicht "bei Implementierung pruefen". Die oklch-Werte sind bekannt, also den WCAG-Kontrast jetzt berechnen (z.B. via oklch.com oder einem APCA-Tool). Wenn die Werte erst beim visuellen QA auffallen, ist der Feedback-Loop unnoetig lang. Berechnete Werte hier eintragen:
> - Light-Kontrast: **8.09:1** (`oklch(92% 0.03 85)` vs `oklch(38% 0.08 60)`)
> - Dark-Kontrast: **8.13:1** (`oklch(17% 0.02 60)` vs `oklch(72% 0.12 155)`)
>
> Beide Werte liegen ueber WCAG AA (4.5:1) fuer normalen Text.

> `[REVIEW]` **Rollback-Plan**: Kontrastaenderungen sind rein subjektiv. Falls auf Staging das Feedback negativ ist, muessen die 3 Werte sofort revertierbar sein. Empfehlung: Eigener Commit (nicht mit anderen Aenderungen mischen), damit `git revert` sauber funktioniert.

**Aufwand**: ~5 Minuten Aenderung + 10 Minuten WCAG-Kontrast-Validierung, 3 Werte aendern (1x @theme, 1x light class, 1x dark class)

---

## Phase 4: Border-Radius vereinheitlichen

> `[REVIEW]` **BLOCKER**: Die Annahme "Tailwind v4 Built-in-Defaults, keine `@theme`-Aenderung noetig" ist **falsch**. Im aktuellen Build existiert nur `--radius: 0.5rem;` (app.css Z.200). Die gestaffelten Variablen `--radius-xs` bis `--radius-3xl` werden von Tailwind v4 **nicht automatisch generiert** — sie muessen explizit im `@theme`-Block definiert werden.
>
> **Dilemma**: Der Plan warnt gleichzeitig (korrekt), dass eigene `--radius-*` Werte in `@theme` die 381 bestehenden `rounded-*` Klassen beeinflussen koennten. Das muss **vor** Implementierung geklaert werden:
>
> **Option A**: Gestaffelte Variablen in `@theme` definieren, die **exakt** den Tailwind-v4-Defaults entsprechen (0.125rem, 0.25rem, 0.375rem, 0.5rem, 0.75rem, 1rem, 1.5rem). Dann aendern sich die bestehenden `rounded-*` Klassen nicht, und die Variablen stehen in Scoped CSS zur Verfuegung.
>
> **Option B**: Variablen ausserhalb von `@theme` in `:root` definieren (analog Phase 2). Dann beeinflussen sie Tailwind-Klassen nicht, aber es gibt zwei parallele Systeme.
>
> **Option C**: Hardcoded px-Werte direkt durch Tailwind-Klassen ersetzen (statt CSS-Variablen). Erfordert aber bei Scoped-CSS-Stellen Template-Refactoring.
>
> **Empfehlung**: Option A mit exakten Tailwind-Default-Werten ist am sichersten. Aber ein Build-Test (Vorher/Nachher-Vergleich aller `rounded-*` Klassen) ist **Pflicht** vor dem Merge.
>
> `[2026]` **Bestaetigt**: Tailwind v4 generiert aus `@theme`-Eintraegen automatisch sowohl Utility-Klassen als auch CSS Custom Properties. Option A ist damit der idiomatische Tailwind-v4-Weg — die Variablen werden in `@theme` definiert, stehen dann sowohl als `var(--radius-sm)` in Scoped CSS als auch als `rounded-sm` Klasse zur Verfuegung. Kein Parallelsystem noetig.

**Problem**: ~56 hardcodierte Pixel-Werte (1.5px, 2px, 3px, 4px, 6px, 8px, 10px, 12px) verstreut ueber ~20 Dateien trotz vorhandener Tailwind-Klassen. `[REVIEW: korrigiert von 59/16]`

**Mapping-Standard** (Tailwind v4 Defaults — **muessen erst in `@theme` definiert werden, siehe Blocker oben**):

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
| 9999px | `9999px` (oder Tailwind-Klasse `rounded-full`) | `rounded-full` | Etablierter grosser Radius, `calc(infinity * 1px)` vermeiden — erhoehte Komplexitaet ohne klaren Nutzen |

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
  - Z.152: `9999px` → Tailwind-Klasse `rounded-full` (bevorzugt) oder `9999px` beibehalten wenn Scoped CSS noetig
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

**Aufwand**: ~2-3 Stunden `[REVIEW: korrigiert von 45-60 Min — inkl. @theme-Setup, Build-Verifikation, 56 Stellen (davon WebAuthn/SecurityKey noch unspezifiziert mit "bei Implementierung pruefen"), visueller QA]`

> `[REVIEW]` **Commit-Strategie**: 56 Aenderungen in ~20 Dateien als ein Commit ist zu grobkoernig. Wenn eine Stelle falsch ist, muss der gesamte Commit zurueckgerollt werden. Empfehlung: Mindestens 3 Commits:
> 1. `@theme`-Setup + globale Styles (Batch 4a)
> 2. Login/Register-Seiten (Batch 4b)
> 3. Komponenten (Batch 4c)

---

## Phase 5: UX-Polish (Logo, Loading, Backdrop-Blur)

> `[REVIEW]` **Reihenfolge geaendert**: 5a (Logo-Animation) vor 5b (Loading-State), weil 5a das Problem loest, das 5b umgehen wollte. Nach 5a ist die permanent laufende `gradient-shift`-Animation on-desktop paused — damit kann 5b einfach `<Logo />` als Komponente importieren statt ein Inline-SVG zu duplizieren.

### 5a: Logo-Animation nur on-hover (Desktop) / reduziert (Touch)

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

### 5b: Branded Loading-State

**Datei**: `frontend/src/routes/+layout.svelte` (Z.425-437)

Generischen Border-Spinner ersetzen durch Logo mit Pulse-Animation.

**Ist-Zustand**:
```svelte
<div class="animate-spin rounded-full h-12 w-12 border-b-2 mx-auto mb-4"
     style="border-color: var(--color-muted-foreground, #666);"></div>
<p style="color: var(--color-muted-foreground, #888);">{$_('common.loading')}</p>
```

**Loesung**: `[REVIEW: geaendert — nach 5a kann Logo.svelte importiert werden]`
- `<Logo />` Komponente importieren (nach 5a ist die Animation on-desktop paused, also kein Performance-Problem)
- Reine CSS `pulse`-Animation um die Komponente
- Falls Svelte-Lifecycle-Probleme auftreten (Layout noch nicht gemountet): Fallback auf leichtgewichtiges Inline-SVG

> **Fallback-Farbe beachten**: Der aktuelle `background-color: var(--color-background, #111)` flasht dunkel bei Light-Theme-Usern. Besser: `#111` durch `oklch(97% 0.03 85)` ersetzen (Light-Default) oder `color-scheme: light dark` setzen, damit der Browser einen passenden Default waehlt.

### 5c: Mobile Backdrop-Blur

**Datei**: `frontend/src/routes/+layout.svelte` (Z.473)

Von `bg-black/50` zu `bg-black/40 backdrop-blur-md`:

> **Bewusste Entscheidung: `backdrop-blur-md` (8px) statt `backdrop-blur-sm` (4px)**: Die einzige bestehende Backdrop-Blur-Nutzung in der App (`TableOfContents.svelte:131`) verwendet `blur(8px)`. Um Konsistenz zu wahren, denselben Wert nutzen. Dazu Opacity von 50% auf 40% reduzieren — Blur uebernimmt einen Teil der visuellen Abgrenzung, weniger Abdunkelung noetig.

> `[2026]` **Performance-Update**: `backdrop-filter: blur()` hat 2026 ~92% Browser-Support und ist auf modernen Geraeten gut optimiert. Best Practices: Blur-Radius unter 20px halten (hier 8px — OK), Anzahl gleichzeitig sichtbarer backdrop-filtered Elemente minimieren, `will-change` sparsam einsetzen. Das Feature ist produktionsreif, aber auf aelteren Android-Geraeten (< 2020) immer noch GPU-intensiv.
>
> **Performance-Hinweis (aktualisiert)**: Falls Ruckler auf aelteren Geraeten auftreten: `@media (prefers-reduced-motion: reduce)` Fallback:
> ```css
> @media (prefers-reduced-motion: reduce) {
>   .backdrop-overlay { backdrop-filter: none; background: rgba(0,0,0,0.5); }
> }
> ```

> `[REVIEW]` **`prefers-reduced-motion` ist kein ausreichender Guard**: Die meisten Mobile-User haben diese Einstellung nicht aktiv. `backdrop-blur` ist ein rein kosmetisches Feature — es sollte nicht pauschal ausgerollt werden, wenn Low-End-Performance nicht getestet werden kann. Besser: `@supports` als zusaetzlichen Guard verwenden und `prefers-reduced-motion` als Override:
> ```css
> /* Blur nur wenn der Browser es unterstuetzt UND User keine reduced-motion will */
> /* Safari nutzt teils weiterhin -webkit-backdrop-filter, daher beide Varianten pruefen */
> @supports (backdrop-filter: blur(8px)) or (-webkit-backdrop-filter: blur(8px)) {
>   .backdrop-overlay {
>     -webkit-backdrop-filter: blur(8px);
>     backdrop-filter: blur(8px);
>     background: rgba(0,0,0,0.4);
>   }
> }
> @media (prefers-reduced-motion: reduce) {
>   .backdrop-overlay { backdrop-filter: none; -webkit-backdrop-filter: none; background: rgba(0,0,0,0.5); }
> }
> ```
> Falls Low-End-Android-Tests nicht moeglich sind: Feature zurueckstellen oder hinter einer User-Preference (Settings) verstecken.

> **Rollout-Entscheidung (fixiert)**: Phase 5c wird standardmaessig **deaktiviert** ausgeliefert und hinter ein Feature-Flag gestellt (z.B. `enableBackdropBlur`). Aktivierung erst nach dokumentiertem Test auf mindestens einem Low-End-Android-Geraet. Ohne dieses Ergebnis bleibt das Flag aus.

**Aufwand Phase 5 gesamt**: ~30-40 Minuten `[REVIEW: korrigiert von 25 Min]`

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

> `[2026]` **Scrollbar-Standards 2026**: `scrollbar-width` und `scrollbar-color` werden seit Chrome/Edge 121+ (2024) unterstuetzt. Firefox hat seit Jahren Support. **Safari/WebKit unterstuetzt die Standard-Properties jedoch immer noch NICHT** — dort sind weiterhin `::-webkit-scrollbar` Pseudo-Elemente noetig. Die Dual-Implementierung (Standard + WebKit) bleibt also fuer Cross-Browser-Support Pflicht. Empfehlung: `@supports` verwenden, um Standard-Properties zu bevorzugen und WebKit-Fallbacks nur bei Bedarf zu laden:
> ```css
> .thin-scrollbar { scrollbar-width: thin; scrollbar-color: auto; }
> @supports not (scrollbar-width: thin) {
>   .thin-scrollbar::-webkit-scrollbar { width: 6px; height: 6px; }
>   /* ... WebKit styles ... */
> }
> ```

Firefox/Chrome/Edge (Standard):
- `html, body, .thin-scrollbar { scrollbar-width: thin; }` (ohne `!important`)

WebKit/Safari Fallback (Z.125-141):
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

> `[REVIEW]` **Versteckte Komplexitaet bei der Konsolidierung**: `.scrollbar-none` verwendet `!important`, `.scrollbar-hide` nicht. Stellen, die `.scrollbar-hide` nutzen, funktionieren moeglicherweise genau deshalb, weil kein `!important` die Spezifitaet uebertrumpft (z.B. bei dynamischem `overflow`-Wechsel). Einfaches Umbenennen kann Seiteneffekte haben. **Empfehlung**: Scrollbar-Scoping (globaler `*`-Selektor entfernen) und Scrollbar-Konsolidierung (`.scrollbar-none`/`.scrollbar-hide` mergen) als **zwei separate Commits** behandeln. Der erste ist risikoarm und liefert sofort Wert (Third-Party-Widgets). Der zweite erfordert mehr Testing.

**Aufwand**: ~25 Minuten Scoping + ~20 Minuten Konsolidierung (separater Commit)

---

## Commit-Reihenfolge (nach PR gruppiert)

### PR A: UX-Fixes (5 Commits)

1. `fix(frontend): add backdrop fade transition and fix drop-zone theme vars` (Phase 1a + 1b)
2. `style(frontend): increase sidebar background contrast` (Phase 3 — eigener Commit fuer einfaches Revert)
3. `style(frontend): logo animation hover-only on desktop` (Phase 5a)
4. `style(frontend): branded loading state with logo component` (Phase 5b)
5. `style(frontend): add backdrop blur to mobile sidebar overlay` (Phase 5c)

### PR B: Motion/Token-System (3 Commits)

6. `perf(frontend): replace transition-all with specific properties` (Phase 1c)
7. `refactor(frontend): unify transition easings and durations via CSS custom properties` (Phase 2, inkl. tokens.ts-Migration)
8. `chore(frontend): enforce transition guardrails in lint and CI` (separater Guardrail-Commit)

### PR C: Radius + Scrollbar-Refactor (5 Commits)

9. `refactor(frontend): define radius token scale in @theme` (Phase 4a — Setup + globale Styles)
10. `refactor(frontend): replace hardcoded border-radius in login/register` (Phase 4b)
11. `refactor(frontend): replace hardcoded border-radius in components` (Phase 4c + Radius-Guardrails)
12. `refactor(frontend): scope thin-scrollbar styles to explicit containers` (Phase 6a — Scoping)
13. `refactor(frontend): consolidate scrollbar-none and scrollbar-hide utilities` (Phase 6b — Konsolidierung)

---

## Verifizierung

**Automatisierte Tests:** `make test-frontend` nach jeder Phase, `make test-e2e` nach Phase 1a/1b, 4b und 5b.

### PR A: UX-Fixes

- **Phase 1a/1b**: Sidebar auf Mobile oeffnen/schliessen — Backdrop muss sanft ein-/ausfaden. Drop-Zone muss Theme-Farbe zeigen (nicht hardcoded Blau). `make test-e2e` (Sidebar-Tests).
- **Phase 3**: Sidebar visuell klar vom Hauptbereich abgesetzt, Dark/Light Toggle pruefen. Vorher/Nachher-Screenshots. **Pflicht**: WCAG-Kontrast vorberechnen.
- **Phase 5a**: Logo-Animation auf Desktop nur bei Hover, auf Touch langsam laufend. `prefers-reduced-motion: reduce` deaktiviert Animation.
- **Phase 5b**: Loading-State zeigt Logo, nicht Spinner. `make test-e2e`.
- **Phase 5c**: Backdrop-Blur auf Low-End-Android testen (falls Geraet verfuegbar).

### PR B: Motion/Token-System

- **Phase 1c**: Sidebar-Collapse auf Desktop darf nicht ruckeln.
- **Phase 2**: Alle interaktiven Elemente (Buttons, Tree-Items, Links) muessen sich gleich "anfuehlen" — selbe Easing-Kurve, konsistente Speed. Besonders 300ms-Stellen (Logo) einzeln pruefen.

### PR C: Radius + Scrollbar-Refactor

- **Phase 4**: Visueller Check Login-Seite, Sidebar, Editor, Settings — keine sichtbaren Radius-Sprünge. Vorher/Nachher-Screenshots. `make test-e2e` nach 4b (Login).
- **Phase 6**: Scrollbars in Editor, Sidebar, Modals, Dropdowns pruefen. Third-Party-Widgets (CodeMirror-Tooltips) duerfen nicht mehr pauschal duenne Scrollbars haben.

> `[REVIEW]` **Fehlend: Visuelle Regressionstests**. `make test-e2e` testet Funktionalitaet, nicht Visuelles. Bei einer rein visuellen Ueberarbeitung waeren Screenshot-Vergleiche (z.B. Playwright Visual Comparisons / `toHaveScreenshot()`) deutlich wertvoller. Empfehlung: Zumindest fuer Login-Seite, Sidebar (open/closed, dark/light) und Editor-View Baseline-Screenshots erstellen, gegen die nach jeder Phase verglichen werden kann. Muss nicht sofort als CI-Pipeline laufen — manuelle Playwright-Screenshots mit `npx playwright test --update-snapshots` reichen als Startpunkt.

---

## Guardrails gegen Regressionen

**Transition Lint-Regel** (PR B, nach Phase 2):
- Hardcoded Easing-Werte (`ease`, `ease-in-out`, `cubic-bezier(...)`) und Durations (`0.15s`, `150ms`, `0.2s`) in Scoped CSS verhindern.
- Erlaubt: `var(--ease-*)`, `var(--duration-*)`, Tailwind-Klassen (`duration-150`, `ease-out`, etc.)
- **Stylelint statt Regex**: Eine reine Regex (`transition:.*\b(ease|...)\b`) erzeugt False Positives bei komplexen Shorthands, Tailwind-generierten Styles und Template-Inline-Styles. Besser: Stylelint mit `stylelint-declaration-strict-value` Plugin, das `transition-timing-function` und `transition-duration` auf Custom Properties einschraenkt. Alternativ: Regex nur auf `<style>`-Bloecke scopen (z.B. via `svelte-check` Plugin oder Pre-Commit-Hook mit `awk`-Extraktion der Style-Sektionen).

**Border-Radius Lint-Regel** (PR C, nach Phase 4):
- **Jede** harte `border-radius`-Literalform verhindern — `px`, `rem` und Tailwind Arbitrary Values (`rounded-[...]`).
- Erlaubt: `var(--radius*)`, Tailwind-Klassen (`rounded-xs` bis `rounded-full`, `rounded-none`), explizite Whitelist fuer Legacy/Third-Party.
- Regex: `border-radius:\s*[\d.]+(px|rem|em)` (CSS) und `rounded-\[` (Tailwind)

**Durchsetzung (lokal + CI, verbindlich):**
- Lint-Regeln werden in den bestehenden Qualitaetslauf integriert (`make quality`).
- Merge-Condition fuer PR B/PR C: `make quality`, `make test-frontend` und betroffene `make test-e2e` Jobs muessen gruen sein.
- Optional lokal: Pre-Commit Hook auf `make quality`, damit Verstoesse vor Push sichtbar sind.

---

## Voraussetzungen (Blocking, vor Implementierungsbeginn)

**Vor PR A:**
1. **WCAG-Kontrast fuer Phase 3 vorberechnen** (siehe Phase 3 Abschnitt). `[REVIEW: konkrete Werte muessen VOR Implementierung berechnet und hier eingetragen werden]`
2. `[REVIEW]` **`color-mix(in oklch, ...)` Browser-Kompatibilitaet**: ~93% Support (Phase 1b). Entscheidung treffen: Fallback-Deklaration (empfohlen) oder Minimum-Browser-Version dokumentieren.
3. `[REVIEW]` **Low-End-Android-Testgeraet verfuegbar?** Phase 5c (`backdrop-blur`) ist GPU-intensiv. Ohne Testmoeglichkeit: Feature hinter `enableBackdropBlur=false` deaktiviert ausliefern.

**Vor PR C:**
4. **`[REVIEW: STATUS = VERIFIZIERT — NICHT VORHANDEN]`** ~~Tailwind v4 Radius-Variablen verifizieren~~: Die gestaffelten Variablen `--radius-xs` bis `--radius-3xl` sind im aktuellen Build **nicht vorhanden**. Nur `--radius: 0.5rem;` existiert (app.css Z.200). **Aktion erforderlich**: Gestaffelte Werte im `@theme`-Block definieren, die exakt den Tailwind-v4-Defaults entsprechen. Danach Build laufen lassen und pruefen, dass bestehende `rounded-*` Klassen (381 Stellen) sich nicht veraendern. Siehe Phase-4-Blocker fuer Details.

**Entfernt:**
- ~~Svelte `transition:fade` Import pruefen~~ — `svelte/transition` ist ein Framework-Modul ohne realistische Circular-Dependency-Gefahr.

**Operationalisierung (Owner, Artefakt, Deadline):**

| ID | Blocker | Owner | Artefakt | Deadline |
|----|---------|-------|----------|----------|
| A1 | WCAG-Kontrast fuer Phase 3 | Frontend | Eingetragene Kontrastwerte im Abschnitt Phase 3 | Vor Commit 2 |
| A2 | `color-mix()` Kompatibilitaetsentscheidung | Frontend | Dokumentierte Entscheidung + ggf. Fallback-Deklaration in PR A | Vor Commit 1 |
| A3 | Low-End-Android fuer Phase 5c | Frontend/QA | Testnotiz (Geraet + Ergebnis) oder explizit gesetztes Feature-Flag `enableBackdropBlur=false` | Vor Commit 5 |
| C1 | Radius-Scale in `@theme` verifizieren | Frontend | Build-Protokoll + Spot-Check `rounded-*` unveraendert | Vor Commit 9 |

---

## Bewusst zurueckgestellt

- **Sidebar-Code-Duplizierung** (~600 Zeilen Mobile/Desktop): Eigener PR, hohes Testrisiko
- **Login-Seite komplett auf Tailwind migrieren**: Eigener PR nach Phase 4 (nutzt dann die Token-Werte)
- **Elevation-System einfuehren**: Wuerde Shadow-Tokens konsistent anwenden, aber breiter Scope. Login-Shadows in Phase 4b bekommen vorerst CSS-Variable-Fallbacks.
- **`transition: all` durch spezifische Properties ersetzen** (ausser Phase 1c): Erfordert pro Komponente eine Analyse, welche Properties sich tatsaechlich aendern. Lohnt sich als eigener Refactoring-Pass nach Phase 2.
- **`UnifiedTree.svelte`: Doppelte `@media (pointer: coarse)`-Bloecke konsolidieren** (Z.970-984 und Z.987-1016): Funktional korrekt aber redundant. Bei naechster Tree-Aenderung mitfixen.
- `[2026]` **View Transitions API fuer Seitennavigation**: Seit Oktober 2025 cross-browser stabil (Chrome, Firefox 144+, Safari). SvelteKit unterstuetzt Integration via `onNavigate`-Lifecycle-Hook. Koennte die User Experience bei Seitenwechseln (z.B. Login → Dashboard, Note → Note) signifikant verbessern. Eigener PR, da orthogonal zu den Design-Token-Verbesserungen in diesem Plan.

---

## PR Ready Checklist

### PR A Ready

- [ ] A1 erledigt: WCAG-Kontrastwerte in Phase 3 eingetragen und >= 4.5:1 bestaetigt
- [ ] A2 erledigt: `color-mix()`-Entscheidung dokumentiert (mit Fallback oder Mindestbrowser)
- [ ] A3 entschieden: Low-End-Android-Testnote vorhanden **oder** `enableBackdropBlur=false`
- [ ] `make test-frontend` erfolgreich
- [ ] `make test-e2e` erfolgreich fuer Phase 1a/1b und 5b

### PR B Ready

- [ ] `tokens.ts`-Imports entfernt (`grep -R "lib/design/tokens" frontend/src` = 0 Treffer)
- [ ] Transition-Guardrails als separater Commit enthalten
- [ ] `make quality` erfolgreich
- [ ] `make test-frontend` erfolgreich

### PR C Ready

- [ ] C1 erledigt: Radius-Scale in `@theme` angelegt und `rounded-*` Spot-Check dokumentiert
- [ ] Radius-Guardrails aktiviert
- [ ] Scrollbar-Scoping und Scrollbar-Konsolidierung als getrennte Commits umgesetzt
- [ ] `make quality` erfolgreich
- [ ] `make test-frontend` erfolgreich
- [ ] `make test-e2e` erfolgreich fuer Phase 4b

---

## Review-Zusammenfassung (2026-02-12)

### Codebase-Verifikation

| # | Behauptung | Status | Anmerkung |
|---|-----------|--------|-----------|
| 1 | Sidebar backdrop ohne transition | Bestaetigt | Z.471-486 in `+layout.svelte` |
| 2 | Undefinierte CSS-Variablen in Drop-Zone | Bestaetigt | `--text-accent` / `--bg-accent-subtle` nicht definiert, Fallbacks greifen |
| 3 | Referenz auf Z.1023 `var(--color-primary)` | Korrigiert | Tatsaechlich Z.~1030 (`.drop-zone.touch-drop-into`) |
| 4 | `transition-all duration-200` Z.319 | Bestaetigt | Auch in `UnlockEncryptionModal.svelte:213` gefunden |
| 5 | Easing-Inkonsistenzen | Bestaetigt | 3-4 verschiedene Easing-Werte + `linear` in 19 Dateien |
| 6 | Sidebar Kontrast oklch-Werte | Bestaetigt | Light `oklch(95% ...)`, Dark `oklch(20% ...)` |
| 7 | Hardcoded `border-radius` | Korrigiert | 56 Instanzen in 20 Dateien (nicht 59/16) |
| 8 | Scrollbar-Duplikate + globaler Selector | Bestaetigt | `.scrollbar-none` (mit !important) und `.scrollbar-hide` (ohne) |
| 9 | `tokens.ts` Kategorien/Nutzung | Bestaetigt | 10 Kategorien, nur 2 verwendet (animationDurations, easing) |
| 10 | Tailwind v4 Radius-Variablen vorhanden | **WIDERLEGT** | Nur `--radius: 0.5rem`, keine gestaffelten Variablen |

### Eingearbeitete Aenderungen (Code-Verifikation)

- **Phase 1**: Browser-Kompatibilitaets-Fallback fuer `color-mix()`, Aufwand korrigiert (30-45 Min)
- **Phase 2**: `tokens.ts`-Migration der 4+1 bestehenden Imports eingeplant, Aufwand korrigiert (60-90 Min)
- **Phase 3**: Rollback-Plan ergaenzt, WCAG-Kontrastwerte vorab berechnet und eingetragen (Light 8.09:1, Dark 8.13:1)
- **Phase 4**: **BLOCKER** dokumentiert (Radius-Variablen existieren nicht), 3 Loesungsoptionen, Commit-Splitting, Aufwand korrigiert (2-3h)
- **Phase 5**: Reihenfolge geaendert (5a Logo-Animation → 5b Loading-State), Loading-State nutzt jetzt `<Logo />` statt Inline-SVG, `@supports`-Guard fuer backdrop-blur
- **Phase 6**: Aufgesplittet in Scoping (risikoarm) und Konsolidierung (erfordert Testing)
- **Commit-Reihenfolge**: Auf 13 Commits verfeinert (Guardrails separiert) fuer granulares Rollback
- **Verifizierung**: Visuelle Regressionstests (Playwright Screenshots) empfohlen
- **Voraussetzungen**: 2 neue Blocker ergaenzt (Browser-Kompatibilitaet, Low-End-Android)

### State-of-the-Art-Abgleich (2026)

| Thema | Plan-Annahme | 2026-Realitaet | Aenderung |
|-------|-------------|----------------|-----------|
| Easing/Duration Tokens | `:root` Custom Properties | `@theme` generiert CSS Vars + Utility-Klassen automatisch | Phase 2 auf `@theme` umgestellt |
| `@property` Typ-Deklarationen | Nicht erwaehnt | 100% Browser-Support seit Juli 2024 | Als Best-Practice-Option ergaenzt |
| `linear()` Easing-Funktion | Nicht erwaehnt | Baseline Widely Available ab Juni 2026 | Als Future-Option fuer komplexe Animationen vorgemerkt |
| `transition: all` | "Beibehalten wo es steht" | Anti-Pattern in 2026 — explizite Properties empfohlen | Empfehlung geaendert: bei jeder angefassten Stelle fixen |
| Svelte 5 `transition:fade` | Korrekt | Weiterhin empfohlen, nutzt intern Web Animation API | Bestaetigt |
| View Transitions API | Nicht erwaehnt | Cross-browser stabil seit Oktober 2025 | Als zurueckgestelltes Feature ergaenzt |
| `color-mix()` Support | ~93% (Fallback noetig) | ~92% — Fallback weiterhin empfohlen | Bestaetigt, Fallback-Code ergaenzt |
| `backdrop-filter: blur()` | "GPU-intensiv, vorsichtig" | 92% Support, produktionsreif mit Best Practices | Performance-Hinweis aktualisiert, weniger restriktiv |
| `scrollbar-width` Standard | Dual-Implementierung | Safari unterstuetzt Standard-Properties IMMER NOCH NICHT | Bestaetigt, `@supports`-Strategie ergaenzt |
| `oklch()` Support | Implizit angenommen | 93% Support — stabil | Bestaetigt |
| `tokens.ts` Zukunft | Nicht erwaehnt | Nach Phase 2 keine aktiven Consumer mehr sinnvoll | Entscheidung fixiert: Datei entfernen, CSS-Variablen als einzige Quelle |
