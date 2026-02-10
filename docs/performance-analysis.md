# Performance-Analyse xelanote

> **See also:** [Performance Baseline](./performance-baseline.md) | [P0 Optimization Plan](./planning/p0-optimization.md) | [P0 Results](./performance/p0-results.md) | [Graph Optimization](./graph-optimization-p3.md)

**Datum:** 2026-01-25
**Status:** Completed
**Analyseumfang:** Backend, Datenbank, Frontend Bundle, Client Runtime

---

## Executive Summary

Xelanote ist grundsätzlich gut architektiert mit Cache, Pagination und modernen Patterns. Die größten Optimierungspotenziale liegen im **Frontend Bundle-Splitting** (CodeMirror 939KB nicht lazy-loaded) und fehlenden **Backend-Profiling-Tools**.

### Priorität-Matrix

| Issue | Impact | Effort | Prio |
|-------|--------|--------|------|
| CodeMirror Code-Splitting | 🔴 Hoch | 🟡 Mittel | **P0** |
| Backend pprof-Endpoint | 🟡 Mittel | 🟢 Niedrig | **P1** |
| force-graph Lazy-Loading | 🟡 Mittel | 🟢 Niedrig | **P1** |
| Wikilink-Plugin Debouncing | 🟡 Mittel | 🟢 Niedrig | **P2** |
| Virtual Scrolling (Sidebar) | 🟢 Niedrig | 🔴 Hoch | **P3** |

---

## 1. Backend Performance

### ✅ Positive Findings

1. **Cache-Strategie**
   - `cache.Cache` mit 5-Minuten TTL für Notes
   - Prefix-basierte Invalidierung (`DeleteByPrefix`)
   - Shared Cache zwischen NoteService und GraphService
   - **Fundstelle:** `backend/internal/service/notes.go:42`

2. **N+1 Query Prevention**
   - `GetNotesByIDs()` für Bulk-Fetching
   - Single-query für mehrere Notes vermeidet N+1
   - **Fundstelle:** `backend/internal/db/notes.go:445`

3. **Cursor-Pagination**
   - Stabile Pagination mit `updated_at|id` Cursor
   - Limit-Caps (max 500 Notes pro Request)
   - **Fundstelle:** `backend/internal/db/notes.go:210`

4. **Async Job-System**
   - Rename-Operations können async laufen (z.B. Backlink-Updates)
   - 4 Worker-Goroutines
   - **Fundstelle:** `backend/cmd/server/main.go:123`

5. **WebSocket Real-time Updates**
   - Broadcast für `note.created`, `note.updated`, `note.deleted`
   - Vermeidet unnötige Polling
   - **Fundstelle:** `backend/internal/api/notes.go:144`

### ❌ Bottlenecks & Fehlende Features

#### KRITISCH: Kein Profiling-Endpoint

**Problem:**
- Kein pprof-Endpoint implementiert
- Keine Möglichkeit, CPU/Memory-Profile zu erstellen
- Keine Runtime-Metrics (Prometheus, etc.)

**Empfehlung:**
```go
import _ "net/http/pprof"

// In main.go:
go func() {
    log.Println("pprof server listening on :6060")
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

**Impact:** 🔴 Hoch - ohne Profiling können Performance-Probleme nicht diagnostiziert werden

**Effort:** 🟢 Niedrig - 10 Zeilen Code

---

#### MEDIUM: Graph-Queries ohne Optimization

**Problem:**
- `GetGlobalGraph` lädt bis zu 1000 Nodes + 5000 Edges
- Zwei separate Queries (resolved + unresolved nodes)
- Keine Query-Result-Caching
- **Fundstelle:** `backend/internal/db/graph.go:45`

**Query-Analyse:**
```sql
-- Query 1: Resolved nodes (kann teuer werden bei 1000+ notes)
SELECT id, title, folder_path
FROM notes
WHERE user_id = ? AND is_deleted = 0
ORDER BY title ASC
LIMIT 1000

-- Query 2: Unresolved nodes (JOIN über alle Notes)
SELECT DISTINCT 'unresolved:' || target_ref_norm as id, target_ref as title
FROM unresolved_links ul
JOIN notes n ON n.id = ul.source_id
WHERE n.user_id = ? AND n.is_deleted = 0
```

**Empfehlung:**
1. Graph-Daten in Service-Layer cachen (15min TTL)
2. Delta-Updates statt Full-Refresh via WebSocket
3. Lazy-Loading für große Graphs (nur visible viewport)

**Impact:** 🟡 Mittel - wird bei 500+ notes langsam

**Effort:** 🟡 Mittel - Cache + WebSocket-Integration

---

#### LOW: Potenzielle N+1 bei Backlinks

**Problem:**
- `GetBacklinks()` läuft pro Note einzeln
- Bei Batch-Operations (z.B. Folder-Display) potenzielle N+1
- **Fundstelle:** `backend/internal/db/links.go` (nicht vollständig gelesen)

**Empfehlung:**
- `GetBacklinksForNotes(noteIDs []string)` Bulk-Funktion
- Ähnlich wie `GetNotesByIDs`

**Impact:** 🟢 Niedrig - betrifft nur spezielle Views

**Effort:** 🟢 Niedrig - 20 Zeilen Code

---

## 2. Datenbank-Performance

### ✅ Positive Findings

1. **Umfassende Indizes**
   - Primary Keys, Foreign Keys, Composite Indices vorhanden
   - FTS5-Indizes für Fulltext-Search
   - Partial Indices für Trash (`WHERE is_deleted = 1`)
   - **Fundstelle:** SQLite Schema-Analyse

```sql
-- Wichtigste Indizes:
CREATE INDEX idx_notes_user_id ON notes(user_id);
CREATE INDEX idx_notes_folder_order ON notes(folder_path, display_order) WHERE is_deleted = 0;
CREATE INDEX idx_note_versions_note_user ON note_versions(note_id, user_id, version DESC);
CREATE INDEX idx_links_target ON links(target_id);
CREATE INDEX idx_unresolved_norm ON unresolved_links(target_ref_norm);
```

2. **FTS5 Fulltext-Search**
   - `notes_fts` mit unicode61 tokenizer
   - BM25-Ranking für Relevanz
   - Snippet-Generierung mit `snippet()` Funktion
   - **Fundstelle:** `backend/internal/db/search.go:62`

3. **Title-Normalisierung**
   - `title_norm` für case-insensitive Suche
   - Verhindert COLLATE NOCASE (ist langsamer)

### ⚠️ Potenzielle Optimierungen

#### QuickSearch LIKE-Pattern

**Problem:**
```sql
WHERE title_norm LIKE ? AND user_id = ?
ORDER BY
  CASE WHEN title_norm LIKE ? THEN 0 ELSE 1 END,
  updated_at DESC
```

- LIKE mit `%query%` kann nicht INDEX nutzen (kein Leading Wildcard)
- Bei 1000+ notes potentiell langsam

**Empfehlung:**
- Für Prefix-Match: `LIKE 'query%'` → nutzt Index
- Für Fuzzy-Match: FTS5 auch für Title-Search nutzen

**Impact:** 🟢 Niedrig - QuickSearch ist meist unter 100ms

**Effort:** 🟢 Niedrig - SQL-Query anpassen

---

## 3. Frontend Bundle-Size

### 📊 Bundle-Analyse (nach Gzip)

| Chunk | Ungzipped | Gzipped | Library | Verwendung |
|-------|-----------|---------|---------|------------|
| `CrdORME5.js` | 939 KB | 303 KB | **CodeMirror 6** | Editor (jede Page) |
| `BSCagXaN.js` | 340 KB | 124 KB | **force-graph** | Graph-Visualisierung |
| `CWt980ZJ.js` | 222 KB | 73 KB | **libsodium-wrappers** | Encryption |
| `bJVRF3Om.js` | 179 KB | 60 KB | **markdown-it** | Preview-Rendering |
| `node_0` | 133 KB | 27 KB | Layout/Routing | - |
| `node_8` | 232 KB | 69 KB | Graph-Page | - |

**Gesamt Initial Bundle:** ~500-600 KB (gzipped)

### ❌ KRITISCH: CodeMirror nicht Code-Split

**Problem:**
- CodeMirror (303 KB gzipped) wird IMMER geladen
- Auch auf `/login`, `/register`, `/graph` (wo kein Editor ist!)
- Vite warnt: "Some chunks are larger than 500 kB"

**Fundstelle:** Jede Route importiert `Editor.svelte` → `codemirror.ts`

**Empfehlung:**

1. **Dynamic Import für Editor:**
```typescript
// In routes/note/[id]/+page.svelte
let EditorComponent = $state<Component<any> | null>(null);

$effect(() => {
  import('$lib/components/Editor.svelte').then(m => {
    EditorComponent = m.default;
  });
});
```

2. **Vite Manual Chunks:**
```typescript
// vite.config.ts
export default defineConfig({
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'codemirror': [
            '@codemirror/state',
            '@codemirror/view',
            '@codemirror/lang-markdown',
            '@codemirror/autocomplete',
            '@codemirror/commands'
          ],
          'force-graph': ['force-graph'],
          'crypto': ['libsodium-wrappers', '@noble/hashes']
        }
      }
    }
  }
});
```

**Impact:** 🔴 Hoch - 303 KB weniger auf Login/Register = ~50% schneller FCP

**Effort:** 🟡 Mittel - 2-3 Stunden Testing

**Expected Gain:**
- Login-Page: 600 KB → 300 KB (-50%)
- Time to Interactive: 2.5s → 1.5s (-40%)

---

### ⚠️ MEDIUM: force-graph auf Graph-Page beschränken

**Problem:**
- force-graph (124 KB) wird global geladen
- Sollte nur auf `/graph` lazy-loaded werden

**Empfehlung:**
```typescript
// routes/graph/+page.svelte
import GraphCanvas from '$lib/components/GraphCanvas.svelte'; // dynamic import inside
```

**Impact:** 🟡 Mittel - 124 KB weniger auf anderen Pages

**Effort:** 🟢 Niedrig - 30 Min

---

### ⚠️ LOW: libsodium Lazy-Loading

**Problem:**
- libsodium-wrappers (73 KB gzipped) immer geladen
- Wird nur gebraucht bei:
  - Encryption-Setup
  - Encrypted Note Create/Edit
  - Password-Change

**Empfehlung:**
```typescript
// lib/crypto/sodium.ts
let sodiumInstance: typeof sodium | null = null;

export async function getSodium() {
  if (!sodiumInstance) {
    const mod = await import('libsodium-wrappers');
    await mod.default.ready;
    sodiumInstance = mod.default;
  }
  return sodiumInstance;
}
```

**Impact:** 🟢 Niedrig - betrifft nur Power-User mit Encryption

**Effort:** 🟢 Niedrig - 1 Stunde

---

## 4. Client-Runtime Performance

### ✅ Positive Findings

1. **Svelte 5 Runes**
   - `$state`, `$derived`, `$effect` optimal genutzt
   - Fine-grained Reactivity (besser als Svelte 4 Stores)
   - **Fundstelle:** `frontend/src/lib/components/Editor.svelte`

2. **Derived Computations**
   - `titleToIdMap` via `$derived.by()` → cached
   - `autoSaveStatusText` ebenfalls derived
   - Keine unnötigen Re-Computations

3. **Lazy Component Loading**
   - MoveToFolderDialog, VersionHistory lazy geladen
   - Gutes Pattern bereits vorhanden

### ⚠️ Optimierungspotenzial

#### MEDIUM: Markdown Re-Rendering bei jedem Keystroke

**Problem:**
```typescript
$effect(() => {
  const note = notes.getCurrentNote();
  if (note) {
    renderedContent = renderMarkdown(note.content, { titleToIdMap });
  }
});
```

- `renderMarkdown()` läuft bei JEDER Content-Änderung
- Bei großen Dokumenten (>5000 Zeilen) kann das laggen

**Empfehlung:**
1. **Debouncing:**
```typescript
let renderTimeout: number;
$effect(() => {
  const note = notes.getCurrentNote();
  if (note) {
    clearTimeout(renderTimeout);
    renderTimeout = setTimeout(() => {
      renderedContent = renderMarkdown(note.content, { titleToIdMap });
    }, 300); // 300ms debounce
  }
});
```

2. **Incremental Rendering:**
- Nur geänderte Absätze re-rendern
- markdown-it unterstützt das nicht nativ → eigene Lösung nötig

**Impact:** 🟡 Mittel - spürbar bei >2000 Zeilen Markdown

**Effort:** 🟢 Niedrig (Debouncing) / 🔴 Hoch (Incremental)

---

#### MEDIUM: Wikilink-Plugin bei jedem docChanged

**Problem:**
```typescript
update(update: ViewUpdate) {
  if (update.docChanged || update.viewportChanged) {
    this.decorations = getWikilinkDecorations(update.view);
  }
}
```

- Regex-Matching über GESAMTES Dokument bei jedem Tastendruck
- Bei 10.000 Zeilen → 10.000 Zeilen Regex-Scan

**Fundstelle:** `frontend/src/lib/editor/codemirror.ts:48`

**Empfehlung:**
1. **Viewport-Only Decoration:**
```typescript
function getWikilinkDecorations(view: EditorView): DecorationSet {
  const decorations: { from: number; to: number }[] = [];

  // NUR sichtbaren Viewport scannen
  for (const { from, to } of view.visibleRanges) {
    const text = view.state.doc.sliceString(from, to);
    let match;
    while ((match = wikilinkMatcher.exec(text)) !== null) {
      decorations.push({
        from: from + match.index,
        to: from + match.index + match[0].length
      });
    }
  }

  return Decoration.set(
    decorations.map((d) => wikilinkDecoration.range(d.from, d.to)),
    true
  );
}
```

**Impact:** 🟡 Mittel - bei großen Dokumenten merkbar

**Effort:** 🟢 Niedrig - 20 Zeilen Code-Änderung

**Expected Gain:**
- 10.000 Zeilen Dokument: 200ms → 10ms Decoration-Update

---

#### LOW: Keine Virtual Scrolling in Sidebar

**Problem:**
- UnifiedTree rendert ALLE Notes/Folders auf einmal
- Bei 1000+ Notes → 1000+ DOM-Elemente
- Scrolling kann ruckeln

**Empfehlung:**
- `@tanstack/svelte-virtual` bereits in `package.json`!
- Implementierung in UnifiedTree.svelte

**Impact:** 🟢 Niedrig - betrifft nur Power-User mit >500 Notes

**Effort:** 🔴 Hoch - Tree-Struktur + Virtual-Scrolling komplex

---

## 5. Prioritized Recommendations

### 🔴 P0 - Sofort umsetzen (Quick Wins)

#### 1. CodeMirror Code-Splitting
- **Impact:** -300 KB auf Login/Register
- **Effort:** 2-3 Stunden
- **ROI:** 🟢🟢🟢 Sehr hoch

**Implementation:**
```typescript
// vite.config.ts - Manual Chunks hinzufügen
manualChunks: {
  'codemirror': [
    '@codemirror/state',
    '@codemirror/view',
    '@codemirror/lang-markdown',
    '@codemirror/autocomplete',
    '@codemirror/commands',
    '@codemirror/language'
  ]
}

// routes/note/[id]/+page.svelte - Dynamic Import
const EditorComponent = import('$lib/components/Editor.svelte');
```

---

### 🟡 P1 - Nächste Sprint

#### 2. Backend pprof-Endpoint
- **Impact:** Observability für zukünftige Optimierungen
- **Effort:** 10 Zeilen Code
- **ROI:** 🟢🟢 Hoch

**Implementation:**
```go
// backend/cmd/server/main.go
import _ "net/http/pprof"

// Nach srv.ListenAndServe():
if os.Getenv("XELANOTE_ENV") == "production" {
    // Nur localhost, nicht public
    go func() {
        log.Println("pprof: http://localhost:6060/debug/pprof/")
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}
```

**Usage:**
```bash
# CPU-Profile (30 Sekunden)
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -http=:8081 cpu.prof

# Memory-Profile
curl http://localhost:6060/debug/pprof/heap > mem.prof
go tool pprof -http=:8081 mem.prof

# Goroutines
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
```

---

#### 3. force-graph Lazy-Loading
- **Impact:** -124 KB auf nicht-Graph-Pages
- **Effort:** 30 Minuten
- **ROI:** 🟢🟢 Mittel-Hoch

---

### 🟢 P2 - Langfristige Optimierungen

#### 4. Wikilink-Plugin Viewport-Only
- **Impact:** Bessere Performance bei >5000 Zeilen
- **Effort:** 1 Stunde
- **ROI:** 🟢 Mittel

#### 5. Markdown Debouncing
- **Impact:** Weniger CPU-Last beim Tippen
- **Effort:** 30 Minuten
- **ROI:** 🟢 Mittel

#### 6. Graph-Data Caching
- **Impact:** Schnellerer Graph-Load
- **Effort:** 2-3 Stunden (Cache + Invalidation)
- **ROI:** 🟡 Niedrig-Mittel

---

### 🔵 P3 - Nice-to-Have

#### 7. Virtual Scrolling in Sidebar
- **Impact:** Besseres Scrolling bei >500 Notes
- **Effort:** 1-2 Tage (komplex)
- **ROI:** 🟡 Niedrig

#### 8. libsodium Lazy-Loading
- **Impact:** -73 KB für Nicht-Encryption-User
- **Effort:** 1 Stunde
- **ROI:** 🟡 Niedrig

---

## 6. Performance-Testing Checklist

Für zukünftige Performance-Validierung:

### Backend Load-Testing
```bash
# Install hey
go install github.com/rakyll/hey@latest

# Test: List Notes (authenticated)
hey -n 1000 -c 10 -H "Cookie: token=YOUR_JWT" \
  http://localhost:8080/api/notes

# Test: Search
hey -n 500 -c 5 -H "Cookie: token=YOUR_JWT" \
  "http://localhost:8080/api/search?q=test"

# Test: Graph
hey -n 100 -c 2 -H "Cookie: token=YOUR_JWT" \
  http://localhost:8080/api/graph
```

### Frontend Bundle-Size Monitoring
```bash
# Nach Build
cd frontend
npm run build

# Größte Chunks finden
find build/_app/immutable -name "*.js" -exec du -h {} \; | sort -rh | head -20

# Gzip-Größen
find build/_app/immutable -name "*.js" -exec sh -c 'gzip -c {} | wc -c' \; | \
  awk '{s+=$1} END {print "Total gzipped:", s/1024/1024, "MB"}'
```

### Lighthouse CI
```bash
# Install
npm install -g @lhci/cli

# Run
lhci autorun --collect.url=http://localhost:8080/login
lhci autorun --collect.url=http://localhost:8080/note/test-id
```

**Targets:**
- Performance Score: >90
- First Contentful Paint: <1.5s
- Time to Interactive: <3s
- Total Blocking Time: <300ms

---

## 7. Monitoring & Observability (Missing!)

### ❌ Aktuell nicht vorhanden

1. **Backend Metrics:**
   - Keine Prometheus-Metrics
   - Keine Request-Duration-Histogramme
   - Keine Error-Rates

2. **Frontend Monitoring:**
   - Kein Sentry / Error-Tracking
   - Keine Real-User-Monitoring (RUM)
   - Keine Web Vitals Collection

### Empfohlene Tools

#### Backend: Prometheus + Grafana
```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

// Metrics-Endpoint
http.Handle("/metrics", promhttp.Handler())
```

**Wichtige Metriken:**
- `http_request_duration_seconds{endpoint, method}`
- `db_query_duration_seconds{query_name}`
- `cache_hit_rate{cache_name}`
- `active_websocket_connections`

#### Frontend: Sentry + Web Vitals
```typescript
// lib/monitoring.ts
import * as Sentry from '@sentry/svelte';
import { onCLS, onFID, onLCP, onFCP, onTTFB } from 'web-vitals';

Sentry.init({ dsn: '...' });

onCLS(metric => Sentry.captureMessage(`CLS: ${metric.value}`));
onLCP(metric => Sentry.captureMessage(`LCP: ${metric.value}`));
```

---

## 8. Benchmark-Ergebnisse (Simulation)

### Backend (geschätzt, ohne echte Benchmarks)

| Operation | Aktuell | Nach Optimierung | Gain |
|-----------|---------|------------------|------|
| List Notes (100) | ~15ms | ~10ms | -33% |
| Search FTS5 | ~20ms | ~20ms | 0% |
| Get Graph (500 nodes) | ~150ms | ~50ms | -67% (mit Cache) |
| Create Note | ~5ms | ~5ms | 0% |

### Frontend (geschätzt)

| Metrik | Aktuell | Nach Code-Split | Gain |
|--------|---------|-----------------|------|
| Login FCP | 2.5s | 1.5s | -40% |
| Login TTI | 3.5s | 2.0s | -43% |
| Note-Page FCP | 2.0s | 1.8s | -10% |
| Bundle-Size (gzipped) | 600 KB | 350 KB | -42% |

---

## 9. Zusammenfassung

### Stärken ✅

1. **Backend:**
   - Solide Cache-Strategie
   - N+1 Prevention
   - Cursor-Pagination
   - Async Jobs
   - WebSocket Real-time

2. **Datenbank:**
   - Umfassende Indizes
   - FTS5 Fulltext-Search
   - Normalisierung

3. **Frontend:**
   - Svelte 5 Runes (optimal)
   - Lazy Component Loading
   - Tree-Shaking funktioniert

### Schwächen ❌

1. **Backend:**
   - Kein Profiling-Endpoint
   - Keine Metrics/Observability
   - Graph-Queries nicht gecacht

2. **Frontend:**
   - CodeMirror 939KB nicht code-split
   - force-graph nicht lazy-loaded
   - Markdown re-rendert bei jedem Keystroke
   - Wikilink-Plugin scannt ganzes Dokument

### Next Steps

1. **Diese Woche:**
   - ✅ pprof-Endpoint hinzufügen (10 Min)
   - ✅ CodeMirror Manual Chunks (2h)

2. **Nächster Sprint:**
   - force-graph Lazy-Loading
   - Wikilink Viewport-Optimization
   - Markdown Debouncing

3. **Q1 2026:**
   - Prometheus-Metriken
   - Sentry-Integration
   - Virtual Scrolling

---

**Erstellt mit:** Claude Opus 4.5
**Nächste Review:** Nach P0/P1-Implementierung
