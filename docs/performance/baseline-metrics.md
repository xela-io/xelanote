# Performance Baseline Metrics

> **See also:** [P0 Results](./p0-results.md) | [Performance Analysis](../performance-analysis.md) | [Performance Baseline](../performance-baseline.md)

**Date:** 2026-01-25
**Commit:** Pre-P0-optimizations
**Measurement Context:** Before CodeMirror code-splitting and pprof implementation

---

## Bundle Analysis (Before)

### Top 25 Largest Chunks

| Chunk | Size (KB) | Gzipped (KB) | Description |
|-------|-----------|--------------|-------------|
| `CrdORME5.js` | 920 | 303 | **CodeMirror** (editor+extensions) |
| `BSCagXaN.js` | 332 | 124 | Vendor chunk (bits-ui, lucide) |
| `nodes/8.B3I9E3Tp.js` | 228 | 69 | Note page node |
| `CWt980ZJ.js` | 220 | 73 | Large vendor chunk |
| `bJVRF3Om.js` | 176 | 60 | Vendor chunk |
| `nodes/0.iYlRpfJB.js` | 132 | 27 | Root layout |
| `BjvLlZj3.js` | 100 | 28 | Vendor chunk |
| `nodes/11.BzS38aTB.js` | 88 | 28 | Settings/Graph node |
| `CDqgqkn6.js` | 52 | 15 | Vendor chunk |
| `BQlOMCHj.js` | 44 | 11 | Vendor chunk |
| `B49-uv7S.js` | 36 | 13 | Vendor chunk |
| `nodes/5.Cy6wwXSL.js` | 24 | 6 | Page node |
| `xDDhwnUV.js` | 24 | 8 | Vendor chunk |
| `BO33FbZd.js` | 20 | 6 | Vendor chunk |
| `BExGUTKk.js` | 20 | 7 | Vendor chunk |
| `kdf.worker-B0ubrTI1.js` | 16 | N/A | KDF Web Worker |
| `BSDCl2o4.js` | 16 | 5 | Vendor chunk |
| `BN5iycpe.js` | 16 | 4 | Vendor chunk |
| `nodes/7.wFUmd9IA.js` | 12 | 3 | Page node |
| `nodes/6.DVHotgks.js` | 12 | 4 | Page node |
| `nodes/13.CUtiEa1S.js` | 12 | 3 | Page node |
| `nodes/12.CXUididf.js` | 12 | 4 | Page node |
| `app.1FYHKevZ.js` | 12 | 3 | App entry |
| `DrPReto5.js` | 12 | 3 | Vendor chunk |
| `C912jgwF.js` | 12 | 3 | Vendor chunk |

### Total Bundle Size

- **Total JS (uncompressed):** ~2.5 MB
- **Total JS (gzipped):** ~830 KB
- **Largest single chunk:** 920 KB (303 KB gzipped) - **CodeMirror**

---

## Critical Issue

**Problem:** CodeMirror (920 KB / 303 KB gzipped) is loaded on **ALL pages** including login/register, where it's not needed.

**Current Manual Chunks Strategy (vite.config.ts):**
- `editor` chunk: @codemirror/view, @codemirror/state
- `editor-extensions` chunk: @codemirror/commands, @codemirror/lang-markdown, @codemirror/language
- `graph` chunk: force-graph
- `vendor` chunk: bits-ui, lucide-svelte

**Hypothesis:**
- Login/register pages load the editor chunks unnecessarily
- Note page loads them correctly (needed there)

---

## Performance Targets (Expected After P0)

### Bundle Size Reduction
- **Login/Register pages:** -300 KB (~42% reduction by removing CodeMirror)
- **Note page:** Similar size (CodeMirror needed here)

### Load Performance (Estimated)
- **Login FCP:** 2.5s → < 1.8s (-28%)
- **Login TTI:** 3.5s → < 2.5s (-29%)

Note: Lighthouse audit not run due to lack of local lighthouse CLI. Manual testing will verify improvements.

---

## Manual Chunks Changes Planned

**Current:** Split into `editor` + `editor-extensions` (2 chunks)
**Planned:** Merge into single `codemirror` chunk (1 chunk)

**Rationale:**
- Fewer HTTP requests (1 vs 2)
- Better gzip compression on larger files
- Simpler dynamic import (one chunk to load)
- Trade-off: Cache granularity (acceptable for initial load optimization)

**Additional chunks planned:**
- `crypto`: libsodium-wrappers, @noble/hashes
- `markdown`: markdown-it
- `icons`: lucide-svelte
- `force-graph`: force-graph (already exists)

---

## Next Steps

1. ✅ Baseline captured
2. ⏭️ Phase 2: Add pprof endpoint
3. ⏭️ Phase 3: Update manual chunks config
4. ⏭️ Phase 4: Implement dynamic import for Editor
5. ⏭️ Phase 5: E2E tests
6. ⏭️ Phase 6: After-metrics + comparison
7. ⏭️ Phase 7: Deployment

---

**Note:** This baseline will be compared against after-metrics in `p0-results.md` after implementation.
