# Graph Query Optimization (PERF P3)

> **See also:** [Performance Analysis](./performance-analysis.md) | [Performance Baseline](./performance-baseline.md) | [P0 Results](./performance/p0-results.md)

**Status:** Implemented (2026-01-26)
**Migration:** 027_graph_indexes.sql
**Commit:** be0b7aa

## Zusammenfassung

Optimierung der Graph-Queries von 4 separaten SQL-Queries auf 2 kombinierte Queries mit CTE (Common Table Expressions) und UNION ALL. Reduziert Datenbank-Roundtrips und minimiert In-Memory-Filterung.

**Performance-Verbesserung:**
- 500 Notes: **-3.0%** (21.9ms → 21.2ms)
- 1000 Notes: **-6.2%** (92.1ms → 86.5ms)
- 2000 Notes (truncated): **-5.7%** (374.5ms → 353.3ms)

**Trade-off:**
- +5% Speicher, +23% Allokationen (CTE-Materialisierung)
- +7% langsamer bei sehr kleinen Datasets (<100 Notes)

---

## Problem

### Alte Implementierung (4 Queries)

```go
// GetGlobalGraph - OLD (4 separate queries)

// Query 1: Resolved nodes (notes)
SELECT id, title, folder_path FROM notes
WHERE user_id = ? AND is_deleted = 0
ORDER BY title ASC LIMIT ?

// Query 2: Unresolved nodes (ALL unresolved_links from user)
SELECT DISTINCT 'unresolved:' || target_ref_norm, target_ref
FROM unresolved_links ul
JOIN notes n ON n.id = ul.source_id
WHERE n.user_id = ? AND n.is_deleted = 0

// Query 3: Resolved edges (links)
SELECT l.source_id, l.target_id FROM links l
JOIN notes src ON src.id = l.source_id
JOIN notes tgt ON tgt.id = l.target_id
WHERE src.user_id = ? AND tgt.user_id = ?
  AND src.is_deleted = 0 AND tgt.is_deleted = 0

// Query 4: Unresolved edges
SELECT ul.source_id, 'unresolved:' || ul.target_ref_norm
FROM unresolved_links ul
JOIN notes n ON n.id = ul.source_id
WHERE n.user_id = ? AND n.is_deleted = 0

// In-Memory: Filter edges where both endpoints in nodeIDSet
```

**Probleme:**
1. **4 DB-Roundtrips** - jede Query hat Latenz-Overhead (auch in-memory SQLite)
2. **Query 2 ineffizient** - lädt ALLE unresolved nodes vom User, nicht nur von truncated sources
3. **In-Memory Filterung dupliziert Arbeit** - SQL könnte Edges bereits filtern

---

## Lösung

### Neue Implementierung (2 Queries)

#### Query 1: Combined Nodes (CTE + UNION ALL)

```sql
WITH resolved AS (
    SELECT id, title, folder_path, 1 as is_resolved
    FROM notes
    WHERE user_id = ? AND is_deleted = 0
    ORDER BY title ASC
    LIMIT ?
)
SELECT id, title, folder_path, is_resolved FROM resolved
UNION ALL
SELECT
    'unresolved:' || ul.target_ref_norm as id,
    COALESCE(MIN(ul.target_ref), ul.target_ref_norm) as title,
    '' as folder_path,
    0 as is_resolved
FROM unresolved_links ul
INNER JOIN resolved r ON ul.source_id = r.id
GROUP BY ul.target_ref_norm
ORDER BY is_resolved DESC, title ASC
```

**Vorteile:**
- **INNER JOIN auf CTE** - unresolved nodes nur von geladenen (truncated) sources
- **GROUP BY statt DISTINCT** - dedupliziert korrekt (vorher Bug bei unterschiedlichen Schreibweisen)
- **Deterministische Reihenfolge** - resolved nodes zuerst, dann unresolved (alphabetisch)

#### Query 2: Combined Edges (UNION ALL)

```sql
SELECT l.source_id, l.target_id, 'resolved' as type
FROM links l
JOIN notes src ON src.id = l.source_id
JOIN notes tgt ON tgt.id = l.target_id
WHERE src.user_id = ? AND tgt.user_id = ?
  AND src.is_deleted = 0 AND tgt.is_deleted = 0
UNION ALL
SELECT ul.source_id, 'unresolved:' || ul.target_ref_norm, 'unresolved'
FROM unresolved_links ul
JOIN notes n ON n.id = ul.source_id
WHERE n.user_id = ? AND n.is_deleted = 0

-- In-Memory: nodeIDSet.has(source) && nodeIDSet.has(target)
```

**Vorteile:**
- **1 Roundtrip statt 2** für alle Edges
- **Type-Column** unterscheidet resolved/unresolved ohne separate Arrays

---

## Migration 027

```sql
-- Index für unresolved_links source_id JOIN
CREATE INDEX IF NOT EXISTS idx_unresolved_links_source
ON unresolved_links(source_id);
```

**Hinweis:** PRIMARY KEY `(source_id, target_ref)` existiert bereits. SQLite kann den PK für source_id-only Lookups nutzen. Dieser dedizierte Index KÖNNTE effizienter sein - Benchmark zeigt minimalen Unterschied.

**Bereits existierende relevante Indizes:**
- `idx_unresolved_norm` (target_ref_norm) - für GROUP BY
- `idx_notes_user_id` (user_id) - für WHERE Filter

---

## Behavior Changes

### 1. Unresolved Node Ordering

**Vorher:** Undefinierte Reihenfolge für unresolved nodes
**Nachher:** Alphabetisch sortiert nach `title`

**Impact:** Frontend-Graph zeigt unresolved nodes in konsistenter Reihenfolge

### 2. Unresolved Node Deduplication (Minor Bug-Fix)

**Vorher:** `SELECT DISTINCT (id, title)` - mehrere Nodes mit gleicher ID aber unterschiedlichem Titel möglich
**Nachher:** `GROUP BY target_ref_norm` mit `MIN(target_ref)` - garantiert einen Node pro ID

**Beispiel:**
- Link 1: `[[Note A]]` → `target_ref_norm = "note a"`
- Link 2: `[[note a]]` → `target_ref_norm = "note a"`

**Vorher:** 2 unresolved nodes mit IDs `unresolved:note a` (unterschiedliche Titel)
**Nachher:** 1 unresolved node mit Titel `MIN("Note A", "note a") = "Note A"`

### 3. Unresolved Node Visibility bei Truncation

**Vorher:** ALLE unresolved nodes vom User geladen, aber Edges korrekt gefiltert
**Nachher:** Nur unresolved nodes von geladenen (truncated) Sources

**Funktional identisch** - nur weniger Daten übertragen.

**Beispiel:**
- 1000 Notes, LIMIT 500 → nur erste 500 nach Alphabet geladen
- Note 501 hat unresolved link zu "Missing"
- **Vorher:** "Missing" node geladen, aber keine Edge (source gefiltert)
- **Nachher:** "Missing" node NICHT geladen (INNER JOIN auf CTE)

---

## Benchmark-Ergebnisse

**Testumgebung:**
- CPU: AMD Ryzen 9 9950X3D 16-Core
- OS: Linux 6.18.7-2-cachyos
- SQLite: In-Memory (`:memory:`)
- Go: 1.23

**Setup:**
- Small: 100 notes, 99 links, 20 unresolved
- Medium: 500 notes, 499 links, 100 unresolved
- Large: 1000 notes, 999 links, 200 unresolved
- Truncated: 2000 notes → LIMIT 1000

### Vergleich: 4 Queries vs. 2 Queries

| Dataset | Vorher (4Q) | Nachher (2Q) | Δ Time | Δ Memory | Δ Allocs |
|---------|-------------|--------------|--------|----------|----------|
| Small (100) | 0.89 ms | 0.95 ms | **+7.2%** ⚠️ | +3.9% | +21.7% |
| Medium (500) | 21.9 ms | 21.2 ms | **-3.0%** ✅ | +4.9% | +24.8% |
| Large (1000) | 92.1 ms | 86.5 ms | **-6.2%** ✅ | +4.7% | +25.3% |
| Truncated (2000→1000) | 374.5 ms | 353.3 ms | **-5.7%** ✅ | +4.1% | +22.9% |

**Interpretation:**
- ✅ **Schneller bei realistischen Workloads** (500+ Notes)
- ⚠️ **Minimal langsamer bei sehr kleinen Datasets** (<100 Notes) - CTE-Overhead
- ⚠️ **+23% Allokationen** - CTE-Materialisierung, aber vernachlässigbar (1.8k → 2.2k)
- ✅ **Rollback-Kriterium nicht erreicht** (<20% Slowdown)

---

## Tests

**9 neue Unit-Tests** in `backend/internal/db/graph_test.go`:

1. `TestGetGlobalGraph_EmptyDatabase` - leere DB
2. `TestGetGlobalGraph_ResolvedNodesOnly` - nur resolved nodes
3. `TestGetGlobalGraph_UnresolvedLinks` - mixed resolved/unresolved
4. `TestGetGlobalGraph_TruncationFiltersEdges` - LIMIT filtert Edges korrekt
5. `TestGetGlobalGraph_UserIsolation` - Multi-User Isolation
6. `TestGetGlobalGraph_TruncationWithMixedNodes` - **INNER JOIN Mechanismus**
7. `TestGetGlobalGraph_DeterministicOrder` - resolved vor unresolved
8. `TestGetFilteredGraph_FolderFilter` - Folder-Filterung
9. `TestGetFilteredGraph_TruncationWithMixedNodes` - **INNER JOIN mit Folder**

**Kritischer Test:** `TestGetGlobalGraph_TruncationWithMixedNodes`

```go
// Setup: Note A, B, C (alphabetisch)
// Note A → unresolved link zu "Missing"
// Note C → unresolved link zu "Also Missing"

// LIMIT 2 → nur A, B geladen

// Erwartung:
// - Nodes: A, B (resolved), "Missing" (unresolved von A)
// - NICHT: C (truncated), "Also Missing" (source C nicht geladen)
```

**Verifiziert:** INNER JOIN auf CTE funktioniert korrekt.

---

## Deployment Guide

### 1. Backup (empfohlen)

```bash
# Homelab Staging
ssh <STAGING_USER>@<STAGING_IP>
docker exec xelanote sqlite3 /app/data/xelanote.db ".backup /tmp/xelanote_pre_m027.db"
docker cp xelanote:/tmp/xelanote_pre_m027.db ~/backups/

# Hetzner Production
ssh xelanote-prod
sudo docker exec xelanote sqlite3 /app/data/xelanote.db ".backup /tmp/xelanote_pre_m027.db"
sudo docker cp xelanote:/tmp/xelanote_pre_m027.db ~/backups/
```

### 2. Deploy

**Standard Deployment** (siehe `CLAUDE.md`):

```bash
# Push
git push origin main

# Homelab Staging
ssh <STAGING_USER>@<STAGING_IP> "cd ~/xelanote && git pull && docker build -t xelanote:latest ."
ssh <STAGING_USER>@<STAGING_IP> "docker stop xelanote && docker rm xelanote"
ssh <STAGING_USER>@<STAGING_IP> 'docker run -d --name xelanote --restart unless-stopped \
  -p 8081:8080 --network nginx_default \
  -v xelanote_xelanote-data:/app/data \
  --env-file ~/.xelanote.env \
  xelanote:latest'

# Hetzner Production (analog)
```

### 3. Migration verifikation

```bash
# Check migration applied
ssh SERVER 'docker exec xelanote sqlite3 /app/data/xelanote.db "SELECT name FROM sqlite_master WHERE type=\"index\" AND name=\"idx_unresolved_links_source\";"'
# Erwartung: idx_unresolved_links_source

# Smoke test
curl -s https://<STAGING_URL>/api/graph -H "Cookie: access_token=..." | jq '.metadata'
# Erwartung: {"node_count": N, "edge_count": M, "truncated": false/true}
```

### 4. Rollback (falls nötig)

**Kriterium:** Performance >20% schlechter als vorher

```bash
# Stop container
ssh SERVER "docker stop xelanote && docker rm xelanote"

# Restore backup
ssh SERVER "sudo docker run --rm -v xelanote_xelanote-data:/data -v ~/backups:/backup alpine cp /backup/xelanote_pre_m027.db /data/xelanote.db"

# Checkout previous commit
ssh SERVER "cd ~/xelanote && git checkout 5708eb2"

# Rebuild & restart (Standard-Deployment)
```

**Note:** Rollback ist SAFE - Migration 027 erstellt nur einen Index, keine Schema-Änderungen.

---

## Query Plan Analyse (optional)

```bash
# Start Backend mit Fixture
JWT_SECRET=$(openssl rand -hex 32) XELANOTE_DB=./data/xelanote_fixture.db make run-backend

# In SQLite Shell
sqlite3 ./data/xelanote_fixture.db

# Query Plan für Node-Query (vereinfacht)
EXPLAIN QUERY PLAN
WITH resolved AS (
    SELECT id, title, folder_path FROM notes WHERE user_id = 1 AND is_deleted = 0 ORDER BY title LIMIT 1000
)
SELECT * FROM resolved
UNION ALL
SELECT 'unresolved:' || ul.target_ref_norm, MIN(ul.target_ref), ''
FROM unresolved_links ul
INNER JOIN resolved r ON ul.source_id = r.id
GROUP BY ul.target_ref_norm;
```

**Erwarteter Plan:**
- SCAN notes USING INDEX idx_notes_user_id
- USE TEMP B-TREE FOR ORDER BY (akzeptabel - alphabetische Sortierung)
- SCAN unresolved_links USING INDEX idx_unresolved_links_source
- SEARCH resolved USING AUTOMATIC COVERING INDEX (CTE)

---

## API-Kompatibilität

**Keine Breaking Changes:**
- `/api/graph` Response-Format unverändert
- `/api/graph?folder=...` Response-Format unverändert
- Frontend-Code benötigt keine Änderungen

**Behavior Changes** (siehe oben):
- Unresolved nodes alphabetisch sortiert (vorher undefiniert)
- Bei Truncation weniger unresolved nodes (funktional identisch)

---

## Lessons Learned

### Was hat funktioniert

1. **Benchmark VORHER laufen lassen** - ermöglichte objektiven Vergleich
2. **CTE + INNER JOIN** - elegante Lösung für Truncation-Problem
3. **Umfassende Tests** - fingen Edge-Cases früh ab
4. **Dokumentiertes Rollback-Kriterium** - klare Entscheidungsgrundlage

### Was zu beachten ist

1. **CTE-Overhead bei kleinen Datasets** - SQLite materialisiert CTEs komplett
2. **Memory-Anstieg** (+5%) - CTE braucht temporären Speicher
3. **In-Memory Filterung weiterhin nötig** - LIMIT auf Nodes kann Edges orphan machen

### Potenzielle Weiterentwicklung

1. **Streaming statt CTE** - für sehr große Datasets (>10k Notes)
2. **Covering Index für ORDER BY** - eliminiert TEMP B-TREE
3. **Edge-Filterung in SQL** - komplexer, aber würde In-Memory Check sparen

**Priorität:** NIEDRIG - aktuelle Lösung ist ausreichend für realistische Workloads.

---

## Referenzen

- **Commit:** be0b7aa
- **Issue/Plan:** `docs/performance-baseline.md` (P3)
- **Migration:** `backend/internal/db/migrations/027_graph_indexes.sql`
- **Code:** `backend/internal/db/graph.go`
- **Tests:** `backend/internal/db/graph_test.go`
- **Benchmark:** Output in commit message

---

## Kontakt / Fragen

Bei Fragen zur Optimierung oder unerwarteten Performance-Problemen nach Deployment:

1. Prüfe `docs/performance-baseline.md` für Baseline-Werte
2. Führe Benchmark aus: `cd backend && go test -bench=BenchmarkGetGlobalGraph -tags fts5`
3. Vergleiche mit dokumentierten Werten in diesem Doc
4. Falls >20% Abweichung: Rollback erwägen (siehe oben)
