# Xelanote Design Review

> Automatisierte Analyse basierend auf 36 Playwright-Screenshots (Desktop + Mobile, DE/EN, Gruvbox Light/Dark)
> Datum: 2026-02-24
> Rev 2: Ueberarbeitet nach kritischer Gegenpruefung (Advocatus Diaboli)

---

## Inhaltsverzeichnis

1. [Methodik](#methodik)
2. [Zusammenfassung](#zusammenfassung)
3. [Kritische Bugs](#1-kritische-bugs)
4. [Internationalisierung (i18n)](#2-internationalisierung-i18n)
5. [Layout & Spacing](#3-layout--spacing)
6. [Typografie](#4-typografie)
7. [Button- & Interaktions-Design](#5-button---interaktions-design)
8. [Dark Mode (Gruvbox Dark)](#6-dark-mode-gruvbox-dark)
9. [Mobile UX](#7-mobile-ux)
10. [Empty States](#8-empty-states)
11. [Navigation & Wayfinding](#9-navigation--wayfinding)
12. [Gezielte Verbesserungen](#10-gezielte-verbesserungen)
13. [Priorisierte Roadmap](#priorisierte-roadmap)

---

## Methodik

- **Screenshots**: `npm run test:e2e:ui-review` (Playwright, Chromium headless)
- **Seiten**: Home Dashboard, Note Editor, Recipes, Journal, Graph, Settings (3 Tabs), Settings/Encryption, Settings/Migration
- **Varianten**: Default (EN), DE Gruvbox Light, DE Gruvbox Dark, EN Gruvbox Light
- **Viewports**: Desktop 1440x900, Mobile 393x852 (iPhone 14)
- **Analyse-Methode**: Pixel-genaue Sichtpruefung aller 36 Screenshots, CSS-Quellcode-Review (`app.css`, Theme-Tokens, Komponenten), Design-System-Abgleich
- **Gegenpruefung**: Kritische Review aller Findings gegen den Quellcode (Advocatus Diaboli), Korrektur von Fehlanalysen

**Bekannte Einschraenkungen des Test-Setups:**
- Recipe-Feature ist standardmaessig deaktiviert → `/recipes` redirected zu `/` (kein Bug, Feature-Flag)
- Graph-Zoom bei 2 Test-Nodes ist kein realistisches Szenario
- Alle Screenshots zeigen einen frisch registrierten Nutzer mit minimalem Content

---

## Zusammenfassung

Das Gruvbox-Farbschema gibt Xelanote eine warme, eigenstaendige Identitaet, die sich positiv von generischen SaaS-Designs abhebt. Das Grundlayout (Icon-Sidebar + Content) ist solide. Es gibt aber **systematische Inkonsistenzen** in Spacing, Typografie, Button-Hierarchie und i18n, die den Gesamteindruck einer professionellen Anwendung schmaelern.

**Staerken:**
- Eigenstaendiges, wiedererkennbares Farbschema
- Gute Mobile-Grundstruktur (Bottom-Nav, Safe-Area-Support)
- Durchdachtes Dashboard mit Drag-Reorder und Collapse
- Solide Accessibility-Basis (ARIA-Attribute, semantisches HTML)
- Glassmorphism-Effekte auf dem Dashboard (zeitgemaess)

**Schwaechen:**
- ~20 hardcoded deutsche Strings auf dem Dashboard (kaputte EN-Locale)
- Button-Hierarchie nicht klar definiert (unterschiedliche Radii, Styles)
- Dark Mode hat Kontrast-Probleme (nicht WCAG-geprueft, aber visuell auffaellig)
- Mobile Settings nur mit Icons (ohne Labels)
- `confirm()` vs `dialog.confirm()` Inkonsistenz
- Drag & Drop auf Mobile funktioniert nicht (HTML5 DnD, kein Touch-Support)

---

## 1. Kritische Bugs

### 1.1 `taskSortable` Import fehlt (GEFIXT)

**Datei:** `frontend/src/lib/components/Editor.svelte:87`

**Problem:** Nur `type TaskSortableOptions` wurde importiert, nicht die `taskSortable`-Funktion selbst. Da die Funktion in Zeile 1038 und 1103 als Svelte-Action (`use:taskSortable`) genutzt wird, crashte der gesamte Editor mit dem Fehler "taskSortable is not defined".

**Screenshot-Evidenz:** Crash-Screen "This section has crashed - taskSortable is not defined" mit Try-again/Reload-Buttons.

**Fix:**
```diff
- import type { TaskSortableOptions } from '$lib/editor/task-sortable';
+ import { taskSortable, type TaskSortableOptions } from '$lib/editor/task-sortable';
```

**Status:** Gefixt waehrend dieser Review-Session.

### 1.2 ~~Recipes-Seite zeigt Home-Dashboard statt Recipes~~ (KEIN BUG)

**Urspruengliche Annahme:** Die Recipes-Screenshots sind pixelidentisch zum Home-Dashboard → vermuteter Redirect-Bug.

**Tatsaechliche Ursache (nach Code-Review):** Das Recipe-Feature ist standardmaessig deaktiviert (`featureEnabled` Flag). `routes/recipes/+page.svelte` Zeile 71-75 redirected zu `/` wenn das Feature aus ist. Die Recipes-Seite hat bereits einen vollstaendigen Empty State mit ChefHat-Icon, Beschreibung und CTA (Zeile 361-371).

**Aktion:** Kein Code-Fix noetig. Das Test-Setup sollte das Recipe-Feature aktivieren, damit Screenshots die tatsaechliche Recipes-Seite zeigen. ➜ Test-Fixture anpassen.

---

## 2. Internationalisierung (i18n)

### 2.1 Hardcoded deutsche Strings im Dashboard

**Betroffene Seiten:** Home Dashboard (`routes/+page.svelte`), `DashboardSection.svelte`

**Evidenz (Desktop, Default EN):**
- Section-Header Englisch: "RECENTLY EDITED", "AKTIVITAT" (Deutsch!)
- Quick-Stats mischen: "1 notes available" (EN) neben "1 heute bearbeitet" (DE) neben "Recently edited" (EN) neben "1 diese Woche" (DE) neben "1 im Root-Ordner" (DE)
- "Weiterarbeiten" (DE) als Section-Title, aber "just now" (EN) als Timestamp
- "Collapsed" (EN) unter "ALLE NOTIZEN" (DE)

**Root Cause (nach Code-Review):** Die Locale-Dateien (`en.json`, `de.json`) sind vollstaendig. Das Problem sind **~20 hardcoded deutsche Strings direkt im Template** von `+page.svelte`, die nie durch `$_()` i18n-Aufrufe ersetzt wurden:

| Zeile | Hardcoded String | Sollte sein |
|-------|-----------------|-------------|
| ~455 | `{count} heute bearbeitet` | `$_('page.home.edited_today', ...)` |
| ~461 | `{count} diese Woche` | `$_('page.home.this_week', ...)` |
| ~467 | `{count} im Root-Ordner` | `$_('page.home.in_root', ...)` |
| ~474 | `Weiterarbeiten` | `$_('page.home.continue_working')` |
| ~555 | `Noch keine Notizen vorhanden...` | `$_('page.home.no_notes')` |
| ~562 | `title="Aktivitaet"` | `$_('page.home.activity')` |
| ~577 | `Heute ({count})` | `$_('page.home.today', ...)` |
| ~601 | `Diese Woche ({count})` | `$_('page.home.this_week_count', ...)` |
| ~628 | `title="Zuletzt erstellt"` | `$_('page.home.recently_created')` |
| ~674 | `title="Alle Notizen"` | `$_('page.home.all_notes')` |
| ~698 | `"Notizen oder Ordner filtern..."` | `$_('page.home.filter_placeholder')` |
| ~713-733 | `"Zuletzt bearbeitet"`, `"Neu erstellt"`, `"A-Z"` | `$_('page.home.sort_*')` |

Zusaetzlich: `DashboardSection.svelte` hat hardcoded deutsche `aria-label`-Attribute ("Bereich ausklappen", "Reihenfolge aendern").

**Empfehlung:**
1. Alle ~20 hardcoded Strings in `+page.svelte` durch `$_()` ersetzen (S, 1-3h)
2. Hardcoded `aria-label` in `DashboardSection.svelte` durch `$_()` ersetzen (XS)
3. Optional: CI-Lint-Regel die deutsche Strings in `.svelte`-Dateien erkennt

### 2.2 Fehlende Pluralisierung

**Problem:** `{count} notes available` ist grammatisch falsch bei count=1 ("1 notes available"). Der ICU MessageFormat-String nutzt keine Pluralisierung.

**Empfehlung:**
```
// en.json
"page.home.notes_available": "{count, plural, one {# note available} other {# notes available}}"

// de.json
"page.home.notes_available": "{count, plural, one {# Notiz vorhanden} other {# Notizen vorhanden}}"
```

### 2.2 Inkonsistente Datumsformate

**Problem:** "just now" (EN-Format) erscheint auch in deutschen Contexten. Deutsche Seiten zeigen "gerade eben" - das ist korrekt. Aber im gemischten Zustand irritiert es.

**Empfehlung:** `date-fns` oder `Intl.RelativeTimeFormat` mit korrekter Locale nutzen, verknuepft an die globale Spracheinstellung.

---

## 3. Layout & Spacing

### 3.1 Dashboard: Card-Spacing ist technisch konsistent, visuell unruhig

**Technischer Befund:** Die `gap-5` zwischen Cards und `p-4 sm:p-5` innerhalb sind konsistent. Der visuelle Eindruck unterschiedlicher Abstande entsteht durch unterschiedliche Content-Mengen, nicht durch inkonsistente CSS-Werte.

**Trotzdem verbesserbar:**
- "ALLE NOTIZEN" Card mit nur "1 insgesamt / Collapsed" wirkt im Verhaeltnis leer
- Optional: `min-h` fuer Cards mit wenig Content, damit der visuelle Rhythmus gleichmaessiger wird
- **Hinweis:** `min-height` erzwingt bei leeren Cards viel Whitespace - abwaegen ob das besser ist

### 3.2 Dashboard: Grid-Ratio (subjektiv)

**Ist-Zustand:** `grid-cols-[1.08fr_0.92fr]` (54:46 Verhaeltnis, nahezu gleich).

**Anmerkung:** Ob die linke oder rechte Spalte mehr Platz verdient, ist ohne A/B-Testing oder Nutzerfeedback nicht objektiv beantwortbar. Die aktuelle Verteilung ist vertretbar. Kein Handlungsbedarf.

### 3.3 Editor: Below-the-Fold Bereich ueberladen

**Problem:** Der Bereich unterhalb des Editors ("Hide Summary & Tags") ist standardmaessig aufgeklappt und zeigt:
- Summary (leer: "No summary available" + "Regenerate" Button)
- Tag Suggestions (collapsed)
- Link Suggestions (collapsed)
- Tags

Bei kurzen Notizen nimmt dieser Bereich ~40% des sichtbaren Viewports ein und lenkt vom eigentlichen Content ab.

**Empfehlung:**
- Summary-Section standardmaessig collapsed wenn leer
- "No summary available" nicht als eigene aufgeklappte Section zeigen
- Gesamten Below-Editor-Bereich in eine einzige collapsible Section mit Tabs umwandeln
- Alternativ: In eine rechte Sidebar verschieben (Desktop)

### 3.4 Settings: Zu viel Leerraum

**Problem:** Die Settings-Seite (Appearance-Tab) zeigt Language-Dropdown + 2 Theme-Cards und darunter ~50% leeren Raum. Bei nur 2 Theme-Optionen wirkt die Seite unausgefuellt.

**Empfehlung:**
- Theme-Cards breiter machen oder in einem Grid anordnen
- Zusaetzliche Appearance-Optionen gruppieren (Font-Groesse, Sidebar-Breite, etc.)
- Oder: Kompakteres Layout das den verfuegbaren Platz besser nutzt

### 3.5 Encryption-Settings: Inkonsistente Section-Abstande

**Problem:** Die Sections "Encrypt titles", "Extract searchable keywords" und "What is encrypted?" haben unterschiedliche vertikale Abstande zueinander. Zwischen "Encrypt titles" und "Extract searchable keywords" ist deutlich mehr Platz als zwischen den anderen Sections.

**Empfehlung:** Einheitliche `space-y-6` oder `gap-6` zwischen allen Settings-Sections.

---

## 4. Typografie

### 4.1 Heading-Hierarchie nicht einheitlich

**Ist-Zustand:**

| Seite | Element | Style |
|-------|---------|-------|
| Dashboard | "Overview" / "Uebersicht" | `ui-page-title` (H1) |
| Dashboard | Section-Headers | `text-[11px] uppercase tracking-[0.12em]` |
| Dashboard | "Weiterarbeiten" | Normaler Text, kein Uppercase |
| Settings | "Settings" | `ui-page-title` (H1) |
| Settings | Tab-Labels | Regular weight, kein Uppercase |
| Settings | "LANGUAGE" / "SPRACHE" | Uppercase, aehnlich Dashboard |
| Editor | Note-Titel | Gross, eigenes Styling |
| Journal | "Journal" | `ui-page-title` (H1) |
| Encryption | "Encryption Settings" | `ui-page-title` (H1) - aber OHNE Back-Arrow-Konsistenz |

**Problem:** Es gibt mindestens 4 verschiedene Heading-Styles die nicht klar hierarchisch sortiert sind:
1. Page Title (H1): Gross, bold
2. Section Header (Uppercase, 11px, tracked): Dashboard-Sections
3. Inline Section Header (Regular, 14-16px): "Weiterarbeiten", Settings-Labels
4. Card Title (Medium weight, 14px): Note-Titel in Listen

**Empfehlung - Einheitliche Type-Scale:**

```
H1 (Page Title):     text-2xl font-semibold  (24px)
H2 (Section Title):  text-sm font-semibold uppercase tracking-wider  (14px)
H3 (Card Title):     text-base font-medium  (16px)
H4 (Subsection):     text-sm font-medium  (14px)
Body:                 text-sm  (14px)
Caption:              text-xs text-muted-foreground  (12px)
Overline:             text-[11px] uppercase tracking-[0.12em] text-muted-foreground
```

### 4.2 Dashboard Section-Titel: "Weiterarbeiten" bricht Pattern

**Problem:** Alle Dashboard-Sections nutzen UPPERCASE Overline-Titel ("RECENTLY EDITED", "AKTIVITAT", "ZULETZT ERSTELLT", "ALLE NOTIZEN"), aber "Weiterarbeiten" im Hero-Bereich nutzt normalen Title-Case. Da es sich visuell in der gleichen Hierarchie befindet, irritiert das.

**Empfehlung:** Entweder alle Section-Titel als Overline ODER alle als Title-Case. Nicht mischen.

---

## 5. Button- & Interaktions-Design

### 5.1 Button-Hierarchie unklar

**Ist-Zustand - gefundene Button-Varianten:**

| Button | Style | Verwendung |
|--------|-------|------------|
| "Create new note" | Filled Primary, rounded-xl, shadow | Dashboard CTA |
| "Open today's journal" | Filled Primary, rounded | Journal Page Action |
| "Unlock Encryption" | Filled Primary, rounded | Journal/Encryption |
| "Reset layout" | Outlined/Ghost, rounded | Dashboard |
| "Regenerate" | Text + Icon, kein Background | Editor Summary |
| "Hide Summary & Tags" | Text Link | Editor |
| Sidebar Icons | Ghost, kein Label | Navigation |
| Settings Tabs | Outlined Pill, Icon + Label | Settings |
| Quick Stats Badges | Subtle Border, Background/40 | Dashboard |

**Probleme:**
1. **Primary Buttons** haben unterschiedliche `border-radius`: `rounded-xl` (Dashboard), `rounded` (Journal), Standard (Encryption)
2. **Secondary Actions** haben keinen einheitlichen Style: "Reset layout" ist outlined, "Regenerate" ist text-only, "Hide Summary" ist ein Link
3. **Ghost Buttons** (Sidebar, Editor-Toolbar) haben unterschiedliche Groessen und Padding-Werte

**Empfehlung - Button-System:**

```
Primary:     bg-primary text-primary-foreground rounded-lg px-4 py-2.5 font-medium
Secondary:   bg-secondary text-secondary-foreground rounded-lg px-4 py-2.5 font-medium
Ghost:       bg-transparent hover:bg-accent rounded-lg px-3 py-2
Destructive: bg-destructive text-destructive-foreground rounded-lg px-4 py-2.5
Link:        text-primary underline-offset-4 hover:underline
Icon:        p-2 rounded-lg hover:bg-accent
```

Konsequent `rounded-lg` fuer alle interaktiven Elemente. Keine Mischung von `rounded-xl`, `rounded`, `rounded-full`.

### 5.2 Editor-Toolbar: Inkonsistente Icon-Gruppierung

**Problem:** Die Editor-Toolbar zeigt Icon-Buttons in visuellen Gruppen (getrennt durch Divider), aber die Gruppierung ist nicht intuitiv:
- Gruppe 1: Live-Preview Toggle, Edit, Preview, Split (Modi)
- Gruppe 2: Insert
- Gruppe 3: Save, Timer-Icon
- Gruppe 4: Fullscreen
- Separat: More-Menu (drei Punkte)

**Empfehlung:**
- Klare semantische Gruppen mit sichtbaren Separatoren
- Tooltips fuer alle Icons (sind vorhanden, gut)
- Active-State klarer hervorheben (aktuell subtiler Hintergrund)

---

## 6. Dark Mode (Gruvbox Dark)

### 6.1 Kontrast-Probleme

**Betroffene Elemente:**

| Element | Vordergrund | Hintergrund | Problem |
|---------|-------------|-------------|---------|
| "Neue Notiz erstellen" Button | Helles Gruen | Dunkelgruener BG | Grenzwertiger Kontrast |
| Card-Borders | Dunkles Gruen/Braun | Dunkelbrauner BG | Kaum sichtbar |
| Quick-Stats Badges | Heller Text | Semi-transparenter BG | Text verschmilzt |
| Muted Foreground Text | Gedaempftes Gold | Dunkler BG | Lesebarkeit bei 11px-Text |

**Empfehlung:**
- WCAG AA Kontrast-Ratio (4.5:1 fuer normalen Text, 3:1 fuer grossen Text) als Minimum
- Tool: `oklch`-Werte in der Theme-Definition systematisch pruefen
- Besonders `--color-muted-foreground` im Dark Mode muss heller werden
- Card-Borders: `border-opacity` von aktuell ~60% auf mindestens 80% erhoehen

### 6.2 Sidebar-Abgrenzung

**Problem:** Die Sidebar im Dark Mode hat nahezu denselben Hintergrund wie der Main-Content. Es gibt keinen visuellen Separator.

**Ist-Zustand:**
```css
--color-sidebar-background: oklch(92% 0.03 85);  /* Light - ok, deutlich anders */
/* Dark Mode sidebar values not checked but visually near-identical to content */
```

**Empfehlung:**
- Subtilen `border-right` oder 1-2% Helligkeitsunterschied zum Main-Content
- Alternativ: Semi-transparenter Gradient am rechten Rand der Sidebar

### 6.3 Active-States schwer erkennbar

**Problem:** Hover- und Active-States der Sidebar-Icons und Dashboard-Cards sind im Dark Mode kaum von Inactive-States unterscheidbar.

**Empfehlung:**
- Active-State: `bg-primary/20` statt `bg-primary/12`
- Hover-State: sichtbarer Ring oder Border-Aenderung
- Focus-State: `ring-2 ring-primary/50` (fuer Keyboard-Navigation)

---

## 7. Mobile UX

### 7.1 Settings-Tabs: Nur Icons, keine Labels

**Problem:** Die mobile Settings-Seite zeigt 6 Tab-Icons ohne jegliche Beschriftung:
- Appearance (Sonnen-Icon?)
- Editor (Stift-Icon)
- Security (Schild-Icon)
- Account (Person-Icon)
- AI (Sparkles-Icon)
- Data (Datenbank-Icon)

Fuer Erstnutzer sind diese Icons nicht selbsterklaerend. Besonders "AI" und "Data" sind schwer zu erraten.

**Empfehlung - Option A (bevorzugt):**
Scrollbare Tab-Leiste mit Icon + Label:
```
[Appearance] [Editor] [Security] [Account] [AI] [Data]
     ↔ horizontales Scrollen
```

**Empfehlung - Option B:**
Segmented Control mit Kurzlabels:
```
Darst. | Editor | Sich. | Konto | KI | Daten
```

### 7.2 Theme-Cards: Schlechter Umbruch auf Mobile

**Problem:** Die Theme-Card-Beschreibung "Warme Retro-Farben mit hohem Kontrast" wird auf 5-6 Zeilen umbrochen, wodurch die Card unverhältnismaessig hoch wird (~120px fuer einen einzeiligen Inhalt).

**Empfehlung:**
- Beschreibung auf Mobile kuerzen oder verbergen
- Horizontales Card-Layout: Icon/Name links, Checkmark rechts
- Oder: Grid mit 2 Spalten, kompaktere Cards

### 7.3 Mobile Editor: Touch-Target-Groessen

**Problem:** Der "Live preview" Dropdown, Undo/Redo Buttons und der "+"-Insert Button haben auf Mobile (393px Viewport) kleine Touch-Targets. Die Toolbar ist funktional dicht gepackt.

**Ist-Zustand (app.css):**
```css
@media (pointer: coarse) {
  .toolbar-btn { min-height: 48px; min-width: 48px; }
  .ui-button, .ui-tab { min-height: 44px; }
}
```

Die WCAG AAA-Empfehlung (48px) ist in CSS definiert, aber visuell wirken die Buttons trotzdem eng beieinander, da kein ausreichendes `gap` zwischen ihnen ist.

**Empfehlung:**
- `gap-1` zwischen Toolbar-Buttons auf `gap-2` erhoehen
- "Live preview" Dropdown als dedizierte Zeile ueber der Toolbar (nicht inline)
- Insert-Button prominenter gestalten (groesser, farblich hervorgehoben)

### 7.4 Mobile Bottom-Nav: "More" nicht selbsterklaerend

**Problem:** Das "More" Tab (drei Punkte + Label) oeffnet ein Bottom-Sheet mit allen weiteren Navigationszielen. Erstnutzer wissen nicht, was sich dahinter verbirgt.

**Empfehlung:**
- Die 3 sichtbaren Tabs sollten die haeufigsten Ziele sein (aktuell: Notes, Search, More - gut)
- "More" Label durch ein konkretes Icon/Label ersetzen wenn moeglich, oder Badge-Count zeigen wenn Aktionen anstehen
- Bottom-Sheet sollte Kategorien haben (Navigation, Features, Einstellungen)

### 7.5 Keyboard-Shortcut auf Mobile

**Problem:** "Open quick search Ctrl+P" wird auch auf Mobile angezeigt. Keyboard-Shortcuts sind auf Touch-Geraeten irrelevant.

**Empfehlung:** `Ctrl+P` Badge auf Touch-Geraeten (`pointer: coarse`) ausblenden.

---

## 8. Empty States

### 8.1 Journal: Minimaler Empty State

**Problem:** Die Journal-Seite zeigt bei gesperrter Verschluesselung nur ein Lock-Icon, einen Satz Text und einen "Unlock Encryption" Button. Der gesamte restliche Viewport ist leer (~80% Whitespace).

**Empfehlung:**
```
[Lock-Icon]
Dein Journal ist verschluesselt

Journal-Eintraege sind Ende-zu-Ende verschluesselt.
Entsperre die Verschluesselung um auf dein Journal zuzugreifen.

[Unlock Encryption]  (Primary CTA)

---
Was ist das Journal?
Dein privater Raum fuer taegliche Gedanken und Reflexionen.
Eintraege werden automatisch nach Datum organisiert.
```

### 8.2 Graph: Zoom-Cap als defensive Massnahme

**Befund:** Bei nur 2 Test-Nodes zeigt der Graph 317% Zoom. Das ist primaer ein Test-Artefakt (unrealistisch wenig Daten), aber ein `max-zoom` Cap waere trotzdem sinnvoll als defensive Massnahme.

**Empfehlung:** `max-zoom: 200%` Cap einbauen (XS-Aufwand). Kein aufwendiger Auto-fit-Algorithmus noetig.

### 8.3 ~~Recipes: Kein eigener Empty State~~ (EXISTIERT BEREITS)

Der Recipes Empty State existiert bereits (`ChefHat`-Icon + CTA, Zeile 361-371 in `+page.svelte`). Siehe Abschnitt 1.2 - das Test-Setup aktiviert das Feature-Flag nicht.

---

## 9. Navigation & Wayfinding

### 9.1 Desktop-Sidebar: Icon-Only ohne Kontext

**Problem:** Die Desktop-Sidebar zeigt nur Icons ohne Labels. Bei 9+ Icons (Home, Checklist, Sharing, Recipes, Calendar, Graph, Trash, Theme-Toggle, Settings) faellt es schwer, spezifische Features zu finden.

**Vergleich mit State-of-the-Art:**
- **Notion**: Expanded Sidebar mit Labels (default), collapsible zu Icons
- **Obsidian**: Icon-Ribbon links, aber mit Tooltips + customizable
- **Linear**: Icon-Only Sidebar, aber nur 5-6 sehr distinkte Icons
- **Apple Notes**: Sidebar mit Labels

**Empfehlung:**
- Expanded/Collapsed Toggle fuer die Sidebar
- Im expanded State: Icon + Label + optional Badge
- Im collapsed State: Icon + Tooltip (aktueller Zustand)
- Default: Expanded auf Desktop (>1200px), Collapsed auf Tablet

### 9.2 Fehlende Breadcrumbs

**Problem:** Im Editor fehlt eine Pfad-Anzeige (z.B. "Root / Subfolder / Note Title"). Der Nutzer sieht nur den Note-Titel, aber nicht wo die Notiz in der Ordnerstruktur liegt.

**Empfehlung:**
- Kompakte Breadcrumb-Zeile ueber dem Editor-Titel:
  `Root > Subfolder > [Note Title]`
- Klickbare Pfad-Segmente zur schnellen Navigation

### 9.3 Settings: Kein Zurueck-Button auf Encryption/Migration

**Problem:** Die Encryption- und Migration-Seiten haben einen Zurueck-Pfeil zum Page-Header, aber es ist nicht klar ob dieser zu Settings oder zur vorherigen Seite navigiert. Der Pfeil ist visuell nicht als "Zurueck zu Settings" erkennbar.

**Empfehlung:**
- Breadcrumb: `Settings > Encryption` statt nur Back-Arrow
- Oder: "← Back to Settings" als expliziter Link-Text

---

## 10. Gezielte Verbesserungen

### 10.1 `confirm()` vs `dialog.confirm()` Inkonsistenz

**Problem (nach Code-Review entdeckt):** In `routes/recipes/+page.svelte` werden sowohl Browser-native `confirm()` Dialoge (Zeile ~139, ~145) als auch die eigene `dialog.confirm()` Loesung (Zeile ~157) verwendet. Der native `confirm()` passt nicht zum Xelanote-Design.

**Empfehlung:** Alle `confirm()` durch `dialog.confirm()` ersetzen. (XS-Aufwand)

### 10.2 HTML5 Drag & Drop funktioniert nicht auf Mobile

**Problem (nach Code-Review entdeckt):** Dashboard-Sections nutzen HTML5 Drag & Drop (`draggable="true"`, `ondragstart`, etc.) fuer die Reihenfolge-Aenderung. HTML5 DnD hat **keinen nativen Touch-Support**. Der "GripVertical" Drag-Handle ist auf Mobile sichtbar, aber funktionslos.

**Empfehlung:**
- Option A: Touch-faehige Library integrieren (z.B. SortableJS, dnd-kit-Aequivalent fuer Svelte)
- Option B: Drag-Handle auf Mobile ausblenden, stattdessen Up/Down-Buttons anbieten
- Option C: `@media (pointer: coarse)` nutzen um den Handle zu verbergen (minimal)

### 10.3 Micro-Interactions (gezielt, nicht global)

**Aktuell:** Hover-States auf Cards sind vorhanden (`hover:border-border/60 hover:bg-accent/30`), aber sehr subtil.

**Empfehlung (nur fuer interaktive Cards, NICHT fuer alle `.ui-panel`):**
```css
/* Nur fuer klickbare Dashboard-Cards und Listeneintraege */
.note-list-item,
.dashboard-card-interactive {
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}
.note-list-item:hover,
.dashboard-card-interactive:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px -4px rgba(0,0,0,0.08);
}

/* Button Press Feedback */
.ui-button:active {
  transform: scale(0.98);
}
```

**Wichtig:** NICHT auf `.ui-panel` global anwenden - das wuerde auch Editor, Settings-Container, Sidebar betreffen.

### 10.4 Border-Radius: Inkonsistenz bei Primary Buttons

**Problem:** Primary Buttons nutzen unterschiedliche Radii je nach Seite:
- Dashboard CTA: `rounded-xl`
- Journal CTA: `rounded`
- Encryption CTA: Default

**Empfehlung:** Einen einheitlichen Radius fuer alle Primary Buttons waehlen (z.B. `rounded-lg`) und als `ui-button-primary` Klasse definieren. Kein komplettes Radius-System-Refactoring noetig - nur die offensichtlichen Inkonsistenzen bei Buttons gleicher Kategorie beheben.

---

## Priorisierte Roadmap

### P0 - Sofort (kaputte Funktionalitaet)

| # | Issue | Aufwand | Dateien |
|---|-------|---------|---------|
| 1 | ~~taskSortable Import~~ | ✅ Done | `Editor.svelte` |
| 2 | Hardcoded DE-Strings im Dashboard durch `$_()` ersetzen (~20 Strings) | S (1-3h) | `routes/+page.svelte` |
| 3 | Hardcoded DE `aria-label` in DashboardSection | XS (<1h) | `DashboardSection.svelte` |
| 4 | Pluralisierung fixen ("1 notes available") | XS (<1h) | `locales/en.json`, `locales/de.json` |

### P1 - Naechster Sprint (Usability + Accessibility)

| # | Issue | Aufwand | Dateien |
|---|-------|---------|---------|
| 5 | Dark Mode Kontrast pruefen + fixen (oklch-Werte messen!) | M-L (6-12h) | `app.css` (Theme-Tokens) |
| 6 | Mobile Settings: Labels zu Tabs hinzufuegen | S (1-4h) | `routes/settings/+page.svelte` |
| 7 | `confirm()` → `dialog.confirm()` in Recipes | XS (<1h) | `routes/recipes/+page.svelte` |
| 8 | Mobile Theme-Cards Layout kompakter | S (1-4h) | `AppearanceTab.svelte` |
| 9 | Keyboard-Shortcut-Badge auf Mobile ausblenden | XS (<1h) | `+page.svelte` |
| 10 | Test-Fixture: Recipe-Feature-Flag aktivieren | XS (<1h) | `tests/fixtures/auth.fixture.ts` |

### P2 - Visual Polish (folgende Sprints)

| # | Issue | Aufwand | Dateien |
|---|-------|---------|---------|
| 11 | Primary Button Radius vereinheitlichen | S (1-4h) | Diverse Komponenten |
| 12 | Heading-Hierarchie vereinheitlichen | M (4-8h) | Globale Styles + Komponenten |
| 13 | Editor Summary collapsed-by-default wenn leer | S (1-4h) | `Editor.svelte` |
| 14 | Dark Mode Sidebar-Abgrenzung | XS (<1h) | `app.css` |
| 15 | Graph max-zoom Cap (200%) | XS (<1h) | `GraphCanvas.svelte` |
| 16 | Dashboard Drag & Drop auf Mobile (Touch-Support oder ausblenden) | M (4-8h) | `DashboardSection.svelte` |

### P3 - Enhancement (Backlog)

| # | Issue | Aufwand | Dateien |
|---|-------|---------|---------|
| 17 | Micro-Interactions (Card Hover Lift, gezielt) | S (1-4h) | `app.css` |
| 18 | Journal Empty State informativer gestalten | S (1-4h) | `routes/journal/+page.svelte` |
| 19 | Editor Breadcrumbs | M (4-8h) | `Editor.svelte` |
| 20 | Settings-Encryption Section-Spacing vereinheitlichen | XS (<1h) | Encryption-Settings-Komponente |

**Legende:** XS = <1h, S = 1-4h, M = 4-8h, L = 1-2 Tage

**Bewusst gestrichen (nach kritischer Pruefung):**
- ~~Recipes Empty State~~ → existiert bereits, Test-Setup-Problem
- ~~Opacity-Scale konsolidieren~~ → hoher Aufwand, kein sichtbarer Nutzen, Regressionsrisiko
- ~~Container Queries Migration~~ → kein konkretes Problem das damit geloest wird
- ~~Skeleton Loading States~~ → SPA mit lokalem State, Seitenwechsel laden keine Server-Daten
- ~~Desktop Sidebar Expand/Collapse~~ → Feature-Request, kein Design-Problem; Obsidian nutzt denselben Ansatz

---

## Anhang: Screenshot-Referenz

Alle Screenshots unter: `frontend/tests/e2e/test-results/ui-review/`

### Desktop (1440x900)
| Datei | Seite | Variante |
|-------|-------|----------|
| `desktop/home-dashboard.png` | Home | Default (EN) |
| `desktop/home-dashboard--en-gruvbox-light.png` | Home | EN Light |
| `desktop/home-dashboard--de-gruvbox-light.png` | Home | DE Light |
| `desktop/home-dashboard--de-gruvbox-dark.png` | Home | DE Dark |
| `desktop/note-editor.png` | Editor | Default |
| `desktop/note-editor--en-gruvbox-light.png` | Editor | EN Light |
| `desktop/recipes.png` | Recipes | Default |
| `desktop/recipes--en-gruvbox-light.png` | Recipes | EN Light |
| `desktop/recipes--de-gruvbox-light.png` | Recipes | DE Light |
| `desktop/recipes--de-gruvbox-dark.png` | Recipes | DE Dark |
| `desktop/journal.png` | Journal | Default |
| `desktop/journal--en-gruvbox-light.png` | Journal | EN Light |
| `desktop/graph.png` | Graph | Default |
| `desktop/graph--en-gruvbox-light.png` | Graph | EN Light |
| `desktop/settings.png` | Settings | Default (EN) |
| `desktop/settings--de-gruvbox-light.png` | Settings | DE Light |
| `desktop/settings--de-gruvbox-dark.png` | Settings | DE Dark |
| `desktop/settings--en-gruvbox-light.png` | Settings | EN Light |
| `desktop/settings-encryption.png` | Encryption | Default |
| `desktop/settings-encryption--en-gruvbox-light.png` | Encryption | EN Light |
| `desktop/settings-migration.png` | Migration | Default |
| `desktop/settings-migration--en-gruvbox-light.png` | Migration | EN Light |

### Mobile (393x852)
| Datei | Seite | Variante |
|-------|-------|----------|
| `mobile/home-dashboard-mobile.png` | Home | Default (EN) |
| `mobile/home-dashboard-mobile--en-gruvbox-light.png` | Home | EN Light |
| `mobile/home-dashboard-mobile--de-gruvbox-light.png` | Home | DE Light |
| `mobile/home-dashboard-mobile--de-gruvbox-dark.png` | Home | DE Dark |
| `mobile/note-editor-mobile.png` | Editor | Default |
| `mobile/note-editor-mobile--en-gruvbox-light.png` | Editor | EN Light |
| `mobile/recipes-mobile.png` | Recipes | Default |
| `mobile/recipes-mobile--en-gruvbox-light.png` | Recipes | EN Light |
| `mobile/journal-mobile.png` | Journal | Default |
| `mobile/journal-mobile--en-gruvbox-light.png` | Journal | EN Light |
| `mobile/settings-mobile.png` | Settings | Default (EN) |
| `mobile/settings-mobile--en-gruvbox-light.png` | Settings | EN Light |
| `mobile/settings-mobile--de-gruvbox-light.png` | Settings | DE Light |
| `mobile/settings-mobile--de-gruvbox-dark.png` | Settings | DE Dark |
