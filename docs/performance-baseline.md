# Performance-Baseline

> **See also:** [Performance Analysis](./performance-analysis.md) | [P0 Results](./performance/p0-results.md) | [Graph Optimization](./graph-optimization-p3.md)

**Datum:** 2026-01-26
**Branch:** main (Commit: a31d7b5)

## Testumgebung

### Hardware (Lokaler Laptop)
- **CPU:** AMD Ryzen 9 9950X3D 16-Core Processor (32 Threads)
- **RAM:** 60GB
- **OS:** Linux 6.18.7-2-cachyos

### Software
- **Node:** v25.4.0
- **Browser:** Chrome (Headless via Lighthouse 13.0.1)

---

## Frontend Bundle-Analyse

### Gesamtgröße
- **Build-Ordner:** 3.0MB
- **JS Chunks total:** 2.2MB (unkomprimiert)

### Top 10 Chunks (nach Transfer-Größe, gzip)

| Chunk | Raw | Gzip | Inhalt |
|-------|-----|------|--------|
| HlGMkLCw.js | 952KB | 308KB | Crypto (noble-hashes, sha256, etc.) |
| DD2PZCcq.js | 561KB | 196KB | CodeMirror (Sprach-Grammars) |
| bJVRF3Om.js | 179KB | 60KB | D3.js (Force-Graph) |
| DjYic1EU.js | 105KB | 47KB | UI/Framework |
| B0Ry3Jdp.js | 101KB | 28KB | App-Logik |
| PAesxZLW.js | 79KB | 18KB | Utilities |
| DvSO7SXt.js | 50KB | 15KB | Components |
| BBP1I6Nv.js | 33KB | 13KB | Components |
| rzbp2_4k.js | 23KB | 8KB | Helpers |
| Co6JIzM6.js | 21KB | 6KB | Stores |

### Unused JavaScript (Lighthouse)

| Chunk | Ungenutzter Code |
|-------|-----------------|
| HlGMkLCw.js | 205KB |
| DjYic1EU.js | 28KB |

**Hinweis:** HlGMkLCw.js (Crypto) hat ~205KB ungenutzten Code weil nicht alle Hash-Algorithmen genutzt werden. Dies ist ein Trade-off für Tree-Shaking-Komplexität vs. Runtime-Performance.

---

## Lighthouse Scores (localhost:4173, Production Build)

### Desktop (--preset=desktop)

| Kategorie | Score |
|-----------|-------|
| **Performance** | **98%** |
| Accessibility | 96% |
| Best Practices | 96% |
| SEO | 90% |

#### Performance-Metriken (Desktop)

| Metrik | Wert | Ziel | Status |
|--------|------|------|--------|
| First Contentful Paint (FCP) | 0.8s | <1.0s | ✅ |
| Largest Contentful Paint (LCP) | 0.9s | <1.5s | ✅ |
| Total Blocking Time (TBT) | 0ms | <100ms | ✅ |
| Time to Interactive (TTI) | 0.9s | <2.5s | ✅ |
| Cumulative Layout Shift (CLS) | 0 | <0.1 | ✅ |
| Speed Index (SI) | 0.8s | <1.5s | ✅ |

### Mobile (--form-factor=mobile, 4x CPU throttle)

| Kategorie | Score |
|-----------|-------|
| **Performance** | **78%** |
| Accessibility | 96% |
| Best Practices | 96% |
| SEO | 90% |

#### Performance-Metriken (Mobile)

| Metrik | Wert | Ziel | Status |
|--------|------|------|--------|
| First Contentful Paint (FCP) | 1.7s | <2.0s | ✅ |
| Largest Contentful Paint (LCP) | 4.4s | <3.0s | ❌ |
| Total Blocking Time (TBT) | 280ms | <200ms | ❌ |
| Time to Interactive (TTI) | 4.4s | <4.0s | ❌ |
| Cumulative Layout Shift (CLS) | 0 | <0.1 | ✅ |
| Speed Index (SI) | 1.7s | <2.0s | ✅ |

**Hinweis:** Mobile Performance wird durch Lighthouse mit 4x CPU-Throttling simuliert. Die LCP/TTI-Werte von 4.4s zeigen, dass die JS-Bundle-Größe auf mobilen Geräten kritisch ist.

---

## Backend

### Fixture DB-Statistiken (Seed 42)

| Entity | Count |
|--------|-------|
| Notes | 500 |
| Folders | 50 |
| Links (resolved) | 733 |
| Links (unresolved) | 300 |
| Tags | 100 |
| Tag Assignments | 495 |

**Test User:** `testuser` / `testpassword123`
**First Note ID:** `3ff8270f-930b-7661-1b69-2ab1d2b6186f`

### Index-Analyse

**Existierende Indizes:**
- `idx_notes_user_id` (user_id)
- `idx_notes_deleted` (is_deleted, deleted_at)
- `idx_notes_user_title_norm` (user_id, title_norm) WHERE is_deleted = 0
- `idx_notes_folder` (folder_path)
- `idx_notes_id` (id)
- `idx_folders_user_path` (user_id, path) UNIQUE
- `idx_links_target` (target_id)
- `idx_unresolved_norm` (target_ref_norm)

**Query Plan Analyse:**

| Query | Index Used | Problem |
|-------|-----------|---------|
| Notes list (ORDER BY updated_at) | idx_notes_user_id | TEMP B-TREE for sort |
| Single note by ID | autoindex_notes_1 (id) | OK |
| Backlinks | idx_notes_user_id | Joins notes first, not links |
| Graph (resolved nodes) | idx_notes_user_id | TEMP B-TREE for ORDER BY title |
| Graph (resolved edges) | idx_notes_user_id x2 | Scans notes twice before links |

**Fehlender Index (Empfohlen):**
```sql
-- Für ORDER BY updated_at DESC
CREATE INDEX idx_notes_user_updated ON notes(user_id, is_deleted, updated_at DESC)
WHERE is_deleted = 0;
```

### API Response Times (Fixture, Local)

_Run benchmark script to populate:_
```bash
# Start backend with fixture
JWT_SECRET=$(openssl rand -hex 32) XELANOTE_DB=./data/xelanote_fixture.db make run-backend

# In another terminal (after login to get cookie)
./backend/scripts/benchmark_api.sh http://localhost:8080 "access_token=..." "3ff8270f-930b-7661-1b69-2ab1d2b6186f"
```

| Endpoint | Median (Local) | Ziel (Staging) |
|----------|----------------|----------------|
| GET /api/notes | - | <100ms |
| GET /api/notes/:id | - | <50ms |
| GET /api/notes/:id/backlinks | - | <200ms |
| GET /api/graph | - | <300ms |
| GET /api/search?q=test | - | <200ms |

### Graph Query Complexity (backend/internal/db/graph.go)

**✅ OPTIMIZED (2026-01-26 - Migration 027):**
- Reduced from 4 queries to 2 queries (CTE + UNION ALL)
- Query 1: Combined nodes (resolved + unresolved via INNER JOIN on CTE)
- Query 2: Combined edges (resolved + unresolved via UNION ALL)
- In-memory filtering via `nodeIDSet` retained (needed for truncation)
- Added index: `idx_unresolved_links_source` (source_id)

**Benchmark Results (AMD Ryzen 9 9950X3D, In-Memory SQLite):**

| Dataset | Before (4Q) | After (2Q) | Change | Notes |
|---------|-------------|------------|--------|-------|
| 100 notes, 99 links | 0.89 ms | 0.95 ms | +7.2% | CTE overhead on small datasets |
| 500 notes, 499 links | 21.9 ms | 21.2 ms | **-3.0%** ✅ | Improvement starts here |
| 1000 notes, 999 links | 92.1 ms | 86.5 ms | **-6.2%** ✅ | Clear win |
| 2000→1000 (truncated) | 374.5 ms | 353.3 ms | **-5.7%** ✅ | INNER JOIN reduces unresolved node load |

**Trade-off:** +5% memory, +23% allocations (CTE materialization), but faster execution for realistic workloads.

**Decision:** Optimization retained. Wins on 500-2000 note datasets outweigh 7% penalty on tiny datasets.

---

## Identifizierte Bottlenecks

### Frontend

1. **Bundle-Größe (Crypto)** - 952KB (308KB gzip)
   - noble-hashes enthält mehr Algorithmen als benötigt
   - Impact: Mobile LCP/TTI erhöht
   - Priorität: MITTEL (schwer zu optimieren ohne Breaking Changes)

2. **Bundle-Größe (CodeMirror)** - 561KB (196KB gzip)
   - Sprach-Grammars werden komplett geladen
   - Impact: Mobile LCP/TTI erhöht
   - Priorität: NIEDRIG (schwer zu reduzieren, Core-Feature)

3. **~~titleToIdMap Neuberechnung~~ (KEIN Problem)**
   - `Editor.svelte:48-55` - $derived.by() basiert auf `notes.getNotes()`
   - `updateCurrentNoteContent()` aktualisiert nur `currentNote`, nicht `notes`
   - **Rebuild nur bei**: Note save/create/delete/rename - NICHT bei jedem Keystroke
   - Priorität: NIEDRIG (kein Handlungsbedarf)

4. **filteredEdges O(n²)** - BESTÄTIGT
   - `graph.svelte.ts:23-29` - zwei `.some()` Aufrufe pro Edge
   - Code: `filteredNodes.some(n => n.id === e.source_id) && filteredNodes.some(n => n.id === e.target_id)`
   - Mit 500 Nodes, 1000 Edges: ~1.000.000 Vergleiche
   - **Lösung**: Set für O(1) Lookups:
     ```typescript
     const nodeIds = new Set(filteredNodes.map(n => n.id));
     return edges.filter(e => nodeIds.has(e.source_id) && nodeIds.has(e.target_id));
     ```
   - Impact: Graph-Filter-Performance
   - Priorität: MITTEL (einfach)

5. **~~Sidebar nicht virtualisiert~~** - ✅ GELÖST (Commits 0ccb80b + 08ee054)
   - Implementiert: `VirtualizedTree.svelte` mit @tanstack/svelte-virtual
   - Nur ~20 sichtbare Items werden gerendert (bei 500+ Notizen)
   - Status: Experimental, opt-in via Settings (standardmäßig deaktiviert)
   - Trade-off: Drag-and-Drop auf sichtbare Items beschränkt
   - Impact: Verbesserte Scroll-Performance und Memory-Nutzung bei 500+ Notizen
   - Refs: frontend/src/lib/components/VirtualizedTree.svelte, frontend/src/routes/settings/+page.svelte:705-716

### Backend

1. **Fehlender Index für ORDER BY** - Notes-Listing
   - Query: `SELECT * FROM notes WHERE user_id=? AND is_deleted=0 ORDER BY updated_at DESC`
   - Problem: TEMP B-TREE für Sortierung
   - Lösung: Composite Index `(user_id, is_deleted, updated_at DESC)`
   - Impact: Verbessert List-Performance
   - Priorität: NIEDRIG (einfach)

2. **Graph 4-Query Split** - backend/internal/db/graph.go
   - 4 separate Queries für Nodes + Edges
   - In-Memory Filterung mit nodeIDSet
   - Impact: Mehr DB-Roundtrips als nötig
   - Priorität: HOCH (Effort: HOCH)

3. **Backlinks Query Reihenfolge**
   - JOIN scannt Notes zuerst statt Links
   - Sollte von links.target_id ausgehen (hat Index)
   - Impact: Suboptimale Query-Reihenfolge
   - Priorität: MITTEL

---

## Nächste Schritte

1. ✅ Baseline dokumentiert (Phase 0)
2. ✅ Fixture-Generator erstellt (Phase 1)
3. ✅ API-Benchmark-Script erstellt (Phase 2)
4. ✅ Backend-Profiling durchgeführt (Phase 2)
5. ✅ Frontend-Profiling durchgeführt (Phase 3)

---

## Priorisierte Optimierungen

| # | Problem | Impact | Effort | Datei | Status |
|---|---------|--------|--------|-------|--------|
| ✅ P1 | filteredEdges O(n²) | MITTEL | NIEDRIG | graph.svelte.ts:23-29 | **DONE** |
| ✅ P2 | Index für ORDER BY | NIEDRIG | NIEDRIG | Migration 026 | **DONE** |
| ✅ P3 | Graph 4-Query Split | MITTEL | HOCH | graph.go, Migration 027 | **DONE** (-3% to -6%) |
| ✅ P4 | Tree Virtualisierung | MITTEL | HOCH | VirtualizedTree.svelte, Sidebar.svelte | **DONE** (Opt-in via Settings) |
| P5 | Backlinks Query | NIEDRIG | MITTEL | db/*.go | TODO |
| ~~P6~~ | ~~titleToIdMap~~ | - | - | - | _Kein Problem_ |

**Empfohlene Reihenfolge:** ~~P1~~ → ~~P2~~ → ~~P3~~ → ~~P4~~ → P5

**P4 Details:**
- Implementiert mit @tanstack/svelte-virtual (Commits 0ccb80b + 08ee054)
- Rendert nur ~20 sichtbare Items statt 500+ (bei 500 Notizen)
- Auto-Scroll, Keyboard-Navigation, Scroll-Restauration
- **Trade-off:** Drag-and-Drop auf sichtbare Items beschränkt
- **Status:** Experimental, standardmäßig deaktiviert (Settings → Performance → Virtual Tree Scrolling)
- Refs: frontend/src/lib/components/VirtualizedTree.svelte, frontend/src/routes/settings/+page.svelte:705-716

---

## Messpunkte für Verifikation

Nach jeder Optimierung wiederholen:

```bash
# Bundle-Size
cd frontend && npm run build
ls -la build/_app/immutable/chunks/*.js | sort -k5 -rn | head -10

# Lighthouse Desktop
npx lighthouse http://localhost:4173 --output=json --preset=desktop \
  --chrome-flags="--headless=new --no-sandbox" \
  | jq '.categories.performance.score * 100'

# Lighthouse Mobile
npx lighthouse http://localhost:4173 --output=json --form-factor=mobile \
  --chrome-flags="--headless=new --no-sandbox" \
  | jq '.categories.performance.score * 100'
```

**Revert-Kriterium:** >10% Verschlechterung in einer Metrik auf einer Plattform.

---

## Lighthouse-Reports

- Desktop: `docs/baseline-lighthouse-desktop.report.html`
- Mobile: `docs/baseline-lighthouse-mobile.report.html`
