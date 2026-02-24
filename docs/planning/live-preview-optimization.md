# Live-Preview Optimierungsplan (v2 — nach kritischer Prüfung)

## Context

Die Live-Preview in xelanote hat zwei Rendering-Pfade:
1. **Split/Preview Mode**: `renderMarkdown()` (markdown-it + DOMPurify) erzeugt HTML-String, der via `{@html renderedContent}` den gesamten DOM ersetzt (Editor.svelte:1194). 150ms Debounce im Split-Mode.
2. **Live Preview Mode**: CodeMirror ViewPlugin mit Decorations (`live-preview.ts`, 840 Zeilen). Baut `DecorationSet` bei jedem Update komplett neu auf.

Ziel: State-of-the-Art Preview-Performance und Feature-Parity mit Obsidian/Typora.

### Korrigierte Annahmen (aus kritischer Prüfung)

1. **Idiomorph + Svelte Actions**: `taskCollapse` erstellt `<details>`-Wrapper via DOM-Manipulation (Zeilen 112-141 in task-collapse.ts). Diese existieren NICHT im renderMarkdown()-Output. Idiomorph würde sie entfernen → Flash. `taskSortable` hat `instancesByContainer.has()` Check (Zeile 141 in task-sortable.ts) der recycelte Nodes nicht refresht. **→ Actions müssen VOR Idiomorph-Einführung kompatibel gemacht werden.**
2. **DOMPurify im Worker**: DOMPurify funktioniert NICHT in Web Workern (braucht `document.createElement()`, nicht nur `DOMParser`). **→ Sanitisierung bleibt auf Main Thread.**
3. **Block-Level Caching**: Markdown-it ist kontextsensitiv (Reference-Links, Lazy Continuations). Block-Splitting an Leerzeilen bricht Semantik. **→ Gestrichen.**

---

## Phase 0: Performance-Baseline erstellen

**Warum**: Ohne Messung ist Priorisierung Raten. Wir brauchen konkrete Zahlen.

### Zu erstellende Datei

**`frontend/src/lib/editor/__benchmarks__/preview-baseline.bench.ts`** (~80 Zeilen)
- Benchmark mit Vitest bench: `renderMarkdown()` für 100, 500, 2000, 10000 Zeilen
- Messung: markdown-it Parse-Zeit, DOMPurify Sanitize-Zeit, Total
- Messung: DOM-Update-Zeit (innerHTML vs morphdom theoretical)
- Baseline für `buildDecorations()` (Live Preview) — existierender Test `live-preview-update-spike.test.ts` als Basis

### Erfolgskriterien für den gesamten Plan
- Split-Mode Input-Latenz: < 16ms (60fps) für Dokumente bis 5000 Zeilen
- Preview First Paint nach Notiz-Wechsel: < 100ms
- Live Preview `buildDecorations`: < 2ms (bestehender Guard)

---

## Phase 1: Image-Optimierung

**Warum**: Geringstes Risiko, sofortiger CLS-Gewinn, 30 Minuten Aufwand.

### Zu ändernde Dateien

**`frontend/src/lib/editor/markdown/image-plugin.ts`** (Zeile 66-76):
- `loading="lazy"` und `decoding="async"` zu `<img>` hinzufügen
- Bei gesetzter Width: `aspect-ratio: auto` Style für CLS-Prevention

```typescript
// Zeile 76, vor dem schließenden >:
html += ` loading="lazy" decoding="async"`;
```

**`frontend/src/lib/editor/markdown/html-sanitizer.ts`**: `loading`, `decoding` zu ALLOWED_ATTR

### Tests
- Unit: Gerendertes HTML enthält `loading="lazy"` und `decoding="async"`
- Bestehende `markdown.test.ts` müssen weiterhin bestehen

---

## Phase 2: CSS `content-visibility: auto`

**Warum**: Pure CSS, kein JS, 30-60% weniger Rendering-Arbeit für Off-Screen-Blöcke.

### Zu ändernde Dateien

**`frontend/src/app.css`** (nach Zeile ~1604, nach `.markdown-preview` Block):

```css
.markdown-preview > h1, .markdown-preview > h2, .markdown-preview > h3,
.markdown-preview > h4, .markdown-preview > h5, .markdown-preview > h6,
.markdown-preview > p, .markdown-preview > blockquote, .markdown-preview > hr {
  content-visibility: auto;
  contain-intrinsic-size: auto 2lh;
}
.markdown-preview > pre {
  content-visibility: auto;
  contain-intrinsic-size: auto 8lh;
}
.markdown-preview > table {
  content-visibility: auto;
  contain-intrinsic-size: auto 10lh;
}
.markdown-preview > ul, .markdown-preview > ol {
  content-visibility: auto;
  contain-intrinsic-size: auto 4lh;
}
```

**Hinweis**: `auto` Keyword bei `contain-intrinsic-size` merkt sich die echte Größe nach erstem Render. `lh` Units passen sich an line-height an (besser als `em`). Safari 17.0+ supportet dies, ältere Browser ignorieren die Property graceful.

### Tests
- Visuell: 500+-Heading-Dokument, kein Scrollbar-Jumping nach erstem vollständigem Scroll
- Ctrl+F: Browser-Suche findet Text in Off-Screen-Blöcken (spec-konform)

---

## Phase 3: Element-Level Scroll-Sync

**Warum**: Löst echtes UX-Problem. Ratio-basierter Sync driftet bei gemischtem Content. Unabhängig von allen anderen Phasen.

### Neue Dateien

**`frontend/src/lib/editor/scroll-sync-elements.ts`** (~180 Zeilen)
- `collectAnchors(previewContainer)`: Sammelt `[data-source-line]`-Elemente mit `offsetTop`
- `computePreviewScroll(editorTopLine, anchors)`: Interpoliert Position zwischen Ankern
- `setupElementScrollSync(editorView, previewScroller)`: Nutzt CodeMirror's `view.lineBlockAtHeight()` für exakte Editor-Zeile
- Anchor-Cache: Nur nach Render invalidieren, nicht bei jedem Scroll
- Bidirektional: Preview→Editor Sync ebenfalls über Anchor-Mapping

### Zu ändernde Dateien

**`frontend/src/lib/editor/markdown.ts`** — Heading Renderer (Zeilen 79-110):
- `data-source-line="${token.map ? token.map[0] + 1 : ''}"` zu `heading_open` hinzufügen
- Neuer `paragraph_open` Renderer: `data-source-line` Attribut
- DOMPurify: `data-source-line` ist bereits durch `data-*` Pattern erlaubt (html-sanitizer.ts:66)
- Koordination mit bestehendem `data-task-line` aus `task-processor.ts`

**`frontend/src/lib/editor/scroll-sync.ts`**: Bestehende Ratio-Logik als `setupRatioScrollSync()` behalten (Fallback). Neuer Export `setupElementScrollSync()`.

**`frontend/src/lib/components/Editor.svelte`** (Zeilen 933-938):
- Feature-Flag Gate: `FEATURE_FLAGS.elementScrollSync` → `setupElementScrollSync()` statt `setupScrollSync()`
- `editorView` an Sync-Funktion übergeben

**`frontend/src/lib/config.ts`**: Flag `elementScrollSync: true`

### Tests
- Unit: `computePreviewScroll()` mit Edge Cases (leer, single anchor, Zeile vor/nach allen Ankern)
- Integration: Editor+Preview, Scroll-Genauigkeit prüfen mit gemischtem Content (Bilder, Tabellen, Code)

---

## Phase 4: Web Worker für Markdown-Rendering

**Warum**: `renderMarkdown()` blockiert Main Thread. Aber: **DOMPurify bleibt auf Main Thread** (funktioniert nicht im Worker).

### Architektur-Entscheidung (korrigiert)

```
Worker:      markdown-it.render() + addDragHandlesToTasks()  → unsanitized HTML
Main Thread: DOMPurify.sanitize(html) → safe HTML → DOM Update
```

DOMPurify ist typischerweise 1-3ms — kein Bottleneck. markdown-it ist der teure Teil (5-50ms je nach Dokumentgröße).

### Neue Dateien

**`frontend/src/lib/editor/markdown-config.ts`** (~80 Zeilen)
- Extrahiert `configureMarkdownIt()` aus `markdown.ts` — Plugin-Registrierung als shared Code
- Wird sowohl von Main Thread als auch Worker importiert
- Enthält: taskLists, colorPlugin, wikilinkPlugin, duedatePlugin, imagePlugin Registration

**`frontend/src/lib/editor/markdown.worker.ts`** (~60 Zeilen)
- `/// <reference lib="webworker" />`
- Importiert `markdown-config.ts` (KEIN DOMPurify im Worker)
- Vite Worker-Support: `new Worker(new URL('./markdown.worker.ts', import.meta.url), { type: 'module' })` — bereits konfiguriert (`vite.config.ts:232-238`)
- Message-Protokoll: `{type: 'render', id, content, titleToIdMap, resolvedTitles}` → `{type: 'render-done', id, html}` (unsanitized!)
- Cancellation: `{type: 'cancel', id}`

**`frontend/src/lib/editor/markdown-worker-client.ts`** (~100 Zeilen)
- `renderMarkdownAsync(content, options): Promise<string>`
- Auto-Cancel vorheriger Requests
- Empfangenes HTML wird auf Main Thread durch `sanitizeRenderedHtml()` geschleust
- Timeout-Fallback: Nach 500ms → synchrones `renderMarkdown()` als Fallback

### Zu ändernde Dateien

**`frontend/src/lib/editor/markdown.ts`**: Plugin-Config nach `markdown-config.ts` auslagern

**`frontend/src/lib/config.ts`**: Flag `workerMarkdown: true`

**`frontend/src/lib/components/Editor.svelte`** (Zeilen 233-253):
- Split-Mode: `renderMarkdownAsync()` statt `setTimeout(() => renderMarkdown())`
- Worker-Ergebnis wird auf Main Thread sanitisiert, dann in `renderedContent` gesetzt

### Serialisierung
- `titleToIdMap` (SvelteMap): Konvertierung zu `[string, string][]` für `postMessage()`
- `widthMap`: Im Worker berechnen (imagePlugin.extractImageWidths auf Worker-Seite)

### Tests
- Mock-Worker-Test: Cancellation, Timeout-Fallback
- Korrektheit: Sync vs Async Output identisch (nach Sanitisierung)
- Performance: Main-Thread-Blocking-Zeit vorher/nachher messen

---

## Phase 5: Idiomorph DOM-Diffing (Split/Preview Mode)

**Warum**: `{@html}` ersetzt den kompletten DOM-Baum. Idiomorph morpht nur geänderte Nodes.

**Prerequisite**: Actions müssen ERST kompatibel gemacht werden.

**Dependency**: `npm install idiomorph` (~3.3KB min/gzipped, v0.7.4)

### Schritt 5a: Action-Kompatibilität herstellen

**`frontend/src/lib/editor/task-collapse.ts`** — Umbau:
- `taskCollapse` muss so geändert werden, dass `cleanup()` vor jedem Morph aufgerufen wird und `init()` danach. Das ist bereits der Fall bei `update()` — ABER: Idiomorph entfernt die `<details>`-Wrapper bevor `update()` feuert, was einen sichtbaren Flash erzeugt.
- **Lösung**: `taskCollapse` als Post-Processing in `renderMarkdown()` integrieren (analog zu `addDragHandlesToTasks()`). Das heißt: Die `<details>`-Wrapper werden bereits im HTML-String erzeugt, nicht via DOM-Manipulation. Dann enthält sowohl der alte als auch der neue HTML-String die Wrapper, und Idiomorph morpht korrekt.
- Alternativ: `taskCollapse` beibehalten wie es ist, Idiomorph überspringt den Morph wenn der Flash akzeptabel ist (Actions machen cleanup+init in <16ms, kaum sichtbar)

**`frontend/src/lib/editor/task-sortable.ts`** (Zeile 140-141) — Bugfix:
- `if (instancesByContainer.has(sortableContainer)) return;` → muss geändert werden zu: Bei `update()` ALLE Instanzen destroyen und neu erstellen. Der aktuelle Code überspringt recycelte Container (auch ohne Idiomorph ein Bug-in-Waiting).

### Schritt 5b: Idiomorph Integration

**Neue Dateien:**

**`frontend/src/lib/types/idiomorph.d.ts`** (~30 Zeilen)
- TypeScript-Deklaration (idiomorph liefert keine eigenen Types)

**`frontend/src/lib/editor/preview-morph.ts`** (~60 Zeilen)
- Wrapper um `Idiomorph.morph()` mit `morphStyle: 'innerHTML'`
- `ignoreActiveValue: true`, `restoreFocus: true`
- `callbacks.beforeNodeMorphed`: `<details>` open-State preserven (falls taskCollapse nicht in renderMarkdown integriert)
- Export: `morphPreview(container, newHtml)`

**`frontend/src/lib/editor/preview-renderer.ts`** (~40 Zeilen)
- Svelte Action `previewRenderer` mit `ActionReturn<PreviewRendererOptions>`
- Erster Render: `innerHTML`
- Folge-Updates: `morphPreview()` (wenn Feature-Flag an)

### Zu ändernde Dateien

**`frontend/src/lib/config.ts`**: Flag `morphPreview: true`

**`frontend/src/lib/components/Editor.svelte`** (Zeilen 1181-1195):
- `{@html renderedContent}` ersetzen durch `use:previewRenderer={{ html: renderedContent }}`
- `use:previewRenderer` VOR den anderen `use:`-Direktiven

### Tests
- Unit-Test für `morphPreview()`: DOM-Struktur, `<details>` open-State
- Integrations-Test: Alle 4 Actions nach Morph funktionsfähig
- **A/B-Benchmark**: Messen ob Idiomorph für typische Notizen messbar schneller ist als innerHTML + Action-Reinitialisierung

---

## Phase 6: Syntax-Highlighting mit Shiki

**Warum**: Code-Blöcke haben kein Syntax-Highlighting. Standard-Erwartung bei modernen Note-Apps.

**Dependency**: `npm install shiki` (lazy-loaded via `shiki/bundle/web`)

### Architektur-Entscheidung: Shiki statt highlight.js

Shiki ist größer (~695KB Bundle/web vs ~50KB highlight.js), aber bietet:
- Exakte VS Code Grammatiken (TextMate)
- CSS-Variable-basiertes Theming via `createCssVariablesTheme()` (perfekt für Gruvbox)
- WASM-basiert (schneller als Regex-Highlighter für große Code-Blöcke)
- Lazy Language Loading

Trade-off: Wenn die Bundle-Größe für PWA kritisch ist, highlight.js als leichtere Alternative evaluieren.

### Zweistufiges Rendering (kein Flash of Unstyled Code)

1. markdown-it `fence` Renderer erzeugt `<pre class="shiki-pending" data-lang="typescript"><code>...</code></pre>`
2. CSS sorgt für Base-Styling (monospace, Hintergrund, Padding) — sofort sichtbar
3. Svelte Action `shikiHighlighter` auf `.markdown-preview` Container:
   - Scannt `pre.shiki-pending` Elemente
   - Lazy-load Shiki + Sprache on demand
   - Ersetzt `<pre>` In-Place mit highlighted Version
   - Entfernt `.shiki-pending` Klasse
4. Bei Idiomorph (Phase 5): Shiki-Output bleibt im DOM (Idiomorph morpht `<pre>` effizient)

### Neue Dateien

**`frontend/src/lib/editor/shiki-loader.ts`** (~80 Zeilen)
- Import von `shiki/bundle/web`
- `createCssVariablesTheme()` mit Mapping auf Gruvbox CSS-Vars:
  ```css
  --shiki-foreground: var(--color-foreground);
  --shiki-background: var(--surface-panel-bg);
  --shiki-token-constant: /* Gruvbox orange */;
  --shiki-token-string: /* Gruvbox green */;
  --shiki-token-keyword: /* Gruvbox red */;
  /* etc. */
  ```
- `getHighlighter()`: Singleton
- `highlightCode(code, lang)`: Lädt Sprache on-demand

**`frontend/src/lib/editor/markdown/code-highlight-plugin.ts`** (~50 Zeilen)
- markdown-it Plugin: Override `fence` Renderer mit `data-lang` und `shiki-pending` Klasse

**`frontend/src/lib/editor/shiki-action.ts`** (~60 Zeilen)
- Svelte Action auf Preview-Container: scannt/highlighted `pre.shiki-pending`

### Zu ändernde Dateien

**`frontend/src/lib/editor/markdown.ts`**: Code-Highlight Plugin registrieren
**`frontend/src/lib/editor/markdown/html-sanitizer.ts`**: Shiki `<span style="color:...">` erlauben
**`frontend/src/app.css`**: Shiki CSS-Variable-Mapping, `.shiki-pending` Base-Styles
**`frontend/src/lib/config.ts`**: Flag `shikiHighlight: true`
**`frontend/src/lib/components/Editor.svelte`**: `use:shikiHighlighter` Action hinzufügen
**`frontend/vite.config.ts`**: Shiki in separatem Chunk

### Tests
- Unit: `highlightCode('const x = 1;', 'typescript')` → HTML mit Color-Spans
- Lazy-Load: Shiki-Bundle wird erst bei Code-Block geladen
- Theme: Gruvbox Light + Dark korrekt

---

## Phase 7: Live Preview Performance-Optimierung

**Warum (korrigiert)**: Statt komplettem Refactor zu inkrementellen Decorations, **erst profilen, dann gezielt optimieren**. Cross-Line State (Heading Sections, Table Blocks, Task Groups) macht `RangeSet.update()` extrem komplex.

### Ansatz: Profiling-gesteuertes Tuning

**Schritt 7a**: Profiling-Ergebnisse auswerten
- Bestehende `setLivePreviewProfilerSink()` nutzen
- Messen: Was dauert länger — `collectTreeFeatures()`, `collectStructuredLines()`, `collectHeadingInfo()`, oder `RangeSetBuilder.finish()`?
- Typisch: Tree-Walking und Regex-Matching sind der Bottleneck, nicht der Builder

**Schritt 7b**: Viewport-Pruning verschärfen
- `collectHeadingInfo()` und `collectCompletedTaskGroups()` iterieren aktuell über das GESAMTE Dokument
- Optimierung: Nur für sichtbaren Viewport + 50-Zeilen-Kontext berechnen
- Heading-Collapse-State per Page-Cache statt Document-Scan

**Schritt 7c**: Dirty-Range-Tracking (optional, nur wenn Profiling es rechtfertigt)
- Statt vollem `buildDecorations()`: Nur Zeilen im sichtbaren Viewport + geänderte Bereiche neu berechnen
- Eigene Dirty-Range-Logik statt `RangeSet.update()` (Cross-Line State macht letzteres impraktikabel)
- Bestehende `shouldRecomputeStructuredLines()` Optimierung als Vorbild

### Zu ändernde Dateien

**`frontend/src/lib/editor/live-preview/heading-manager.ts`**: Viewport-aware machen
**`frontend/src/lib/editor/live-preview/task-group-manager.ts`**: Viewport-aware machen
**`frontend/src/lib/editor/live-preview.ts`**: Conditional Rebuild nur für geänderte Viewport-Bereiche

### Bestehende Performance-Guards
- `live-preview-update-spike.test.ts`: `selectionSet:build < 2.0ms`, `docChanged:build < 1.5ms`

---

## Phase 8: KaTeX Math-Rendering

**Warum**: Kein Math-Support. Essentiell für akademische/technische Nutzer.

**Dependency**: `npm install katex` (~90KB, lazy-loaded)

### Neue Dateien

**`frontend/src/lib/editor/math-loader.ts`** (~50 Zeilen)
- Lazy-Load KaTeX JS + CSS nur wenn `$`/`$$` Syntax erkannt
- `renderMath(tex, displayMode)`: nutzt `katex.renderToString(tex, { throwOnError: false, displayMode, output: 'htmlAndMathml' })`
- `renderToString` braucht kein DOM → auch im Web Worker nutzbar

**`frontend/src/lib/editor/markdown/math-plugin.ts`** (~80 Zeilen)
- markdown-it Plugin: `$$block$$` (display) und `$inline$` Erkennung
- `ignoredTags: ["code", "pre", "script", "style", "textarea"]`
- Escape: `\$` wird nicht als Math interpretiert

### Zu ändernde Dateien

**`frontend/src/lib/editor/markdown.ts`**: Math Plugin registrieren (bedingt durch Flag)
**`frontend/src/lib/editor/markdown/html-sanitizer.ts`**: KaTeX/MathML-Tags erlauben: `math`, `mrow`, `mi`, `mo`, `mn`, `msup`, `msub`, `mfrac`, `msqrt`, `mtext`, `annotation`, `semantics`
**`frontend/src/app.css`**: KaTeX CSS import (lazy, nur wenn math-plugin aktiv)
**`frontend/src/lib/config.ts`**: Flag `mathRendering: false` (opt-in, `$`-Konflikte mit Währungen)

---

## Phase 9: Mermaid Diagramme

**Warum**: Feature-Parity mit Obsidian, Notion, GitHub.

**Dependency**: `npm install mermaid` (~500KB, lazy-loaded)

### Neue Dateien

**`frontend/src/lib/editor/mermaid-loader.ts`** (~60 Zeilen)
- Lazy-Load Mermaid.js bei erstem ` ```mermaid` Block
- Content-Hash Cache (FNV-1a): Gleicher Diagram-Code → cached SVG

**`frontend/src/lib/editor/mermaid-action.ts`** (~100 Zeilen)
- Svelte Action auf `.markdown-preview`: scannt `<code class="language-mermaid">` Blöcke
- Ersetzt mit gerendertem SVG in Shadow DOM (CSS-Isolation)
- DOMPurify auf SVG Output (Mermaid rendert arbitrary User-Input!)
- 500ms Debounce (Mermaid-Rendering ist teuer auf Mobile)
- Theme-Integration: Gruvbox-kompatible Farben

### Zu ändernde Dateien

**`frontend/src/lib/editor/markdown/html-sanitizer.ts`**: Erweiterte SVG-Tags für Mermaid-Output
**`frontend/src/lib/components/Editor.svelte`**: `use:mermaidRenderer` Action
**`frontend/src/lib/config.ts`**: Flag `mermaidDiagrams: false` (opt-in)

---

## Feature Flags Übersicht

```typescript
// frontend/src/lib/config.ts
export const FEATURE_FLAGS = {
  // Bestehend
  colorSyntax: true,
  taskLists: true,
  livePreview: true,
  imageResize: true,
  dueDateSyntax: true,
  tagSuggestions: true,
  linkSuggestions: true,
  spellCheck: true,
  // Neue Optimierungen
  lazyImages: true,            // Phase 1
  elementScrollSync: true,     // Phase 3
  workerMarkdown: true,        // Phase 4
  morphPreview: true,          // Phase 5
  shikiHighlight: true,        // Phase 6
  mathRendering: false,        // Phase 8 (opt-in)
  mermaidDiagrams: false,      // Phase 9 (opt-in)
};
```

Phase 2 (CSS content-visibility) braucht kein Flag — pure CSS.
Phase 7 (Live Preview Tuning) nutzt bestehende Profiler-Infrastruktur.

---

## Implementierungsreihenfolge

| # | Phase | Aufwand | Risiko | Abhängigkeiten |
|---|-------|---------|--------|----------------|
| 0 | Performance-Baseline | 0.5 Tage | Keins | — |
| 1 | Image-Optimierung | 0.5 Tage | Sehr niedrig | — |
| 2 | CSS content-visibility | 0.5 Tage | Niedrig | — |
| 3 | Element-Level Scroll-Sync | 2 Tage | Mittel | — |
| 4 | Web Worker (ohne DOMPurify) | 2 Tage | Mittel | — |
| 5 | Idiomorph (nach Action-Umbau) | 3 Tage | Mittel-Hoch | Phase 4 optional |
| 6 | Shiki Syntax-Highlighting | 3 Tage | Mittel | Phase 5 optional |
| 7 | Live Preview Tuning | 2 Tage | Mittel | — (profiling-gesteuert) |
| 8 | KaTeX Math | 2 Tage | Niedrig | — |
| 9 | Mermaid Diagrams | 2 Tage | Mittel | Phase 5 optional |

**Total: ~18 Tage**

---

## Verifizierte Quellen (Context7 + Web)

- **Idiomorph**: [GitHub](https://github.com/bigskysoftware/idiomorph) | [npm](https://www.npmjs.com/package/idiomorph) — v0.7.4, 3.3KB, `morphStyle: 'innerHTML'`, Callbacks, `ignoreActiveValue`, `restoreFocus`. Keine TS-Types.
- **DOMPurify in Workers**: [GitHub Issue #577](https://github.com/cure53/DOMPurify/issues/577) — Nicht unterstützt, braucht `document.createElement()`.
- **Shiki**: [Context7 /shikijs/shiki](https://github.com/shikijs/shiki) — `createCssVariablesTheme()`, `shiki/bundle/web`, lazy Languages.
- **CodeMirror RangeSet**: [Context7 /websites/codemirror_net](https://codemirror.net/docs/ref) — `RangeSet.update({ add, filter })`, `.map(changes)`. Cross-Line State macht inkrementelle Updates komplex.
- **KaTeX**: [Context7 /katex/katex](https://github.com/katex/katex) — `renderToString()` ohne DOM (Worker-kompatibel), `renderMathInElement()` mit `ignoredTags`.
- **Svelte 5 Actions**: [Context7 /websites/svelte_dev_svelte](https://svelte.dev/docs/svelte/use) — `ActionReturn<Parameter>` mit `update`/`destroy` funktioniert.

## Verifikation

Nach jeder Phase:
1. `make test-frontend` — alle bestehenden Tests bestehen
2. `make run-frontend` — manuelle Verifikation im Browser (Split + Preview + Live Mode)
3. Performance-Profiling gegen Baseline (Phase 0)
4. Mobile: iOS Safari Test (touch targets, momentum scrolling, content-visibility)
5. `live-preview-update-spike.test.ts` Guards bestehen weiterhin
