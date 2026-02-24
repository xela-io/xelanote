# Refactoring-Analyse

> Erstellt: 2026-02-24 | Analysiert: Backend (Go), Frontend (SvelteKit), Dependencies & Tooling
> Kritisch geprueft: 2026-02-24 (advocatus-diaboli-Review, 5 False Positives korrigiert)
> Technik-Check: 2026-02-24 (Abgleich mit Context7-Docs: Svelte 5, SvelteKit 2.12+, Vite 6+, golangci-lint v2)
>
> **Zusammenfassung:** 42 valide Issues identifiziert. 5 kritisch, 15 wichtig, 22 nice-to-have.
> Gesamtaufwand geschaetzt: 20-30 Personentage fuer alle Phasen.

---

## Kritisch 🔴 (sollte sofort angegangen werden)

### K-01: Swallowed Errors + globale errcheck-Exclusion als Root Cause
**Betroffene Dateien:**
- `backend/internal/db/auth.go:159` - Token-Revocation Error ignoriert
- `backend/internal/db/sharing_notes.go:56-65` - Placement-Cleanup Error ignoriert
- `.golangci.yml:43` - `(*database/sql.DB).Exec` global von errcheck excluded

**Problem:** Fehler bei sicherheitskritischen Operationen (Token-Revocation, Shared-Note-Cleanup) werden mit `_, _ = db.Exec(...)` still verschluckt. Die **eigentliche Root Cause** ist die pauschale errcheck-Exclusion in `.golangci.yml` Zeile 43, die `(*database/sql.DB).Exec` global suppressed. Dadurch werden *alle* unbehandelten `db.Exec`-Errors im gesamten Projekt vom Linter nicht erkannt.

Zusaetzlich ist `(*database/sql.Row).Scan` (Zeile 42) excluded - unchecked Scan-Errors fuehren zu Zero-Value-Daten, was Security-relevant sein kann.

**Loesungsvorschlag:**
1. `.golangci.yml`: Die globale `(*database/sql.DB).Exec`-Exclusion entfernen (Zeile 43). Auch `(*database/sql.Row).Scan` (Zeile 42) pruefen - laut golangci-lint v2 Best Practices sollten `exclude-functions` nur fuer tatsaechlich nicht-actionable Errors genutzt werden (Scan-Errors sind actionable!).
2. An den wenigen bewusst ignorierten Stellen gezielte `//nolint:errcheck`-Kommentare setzen
3. Die erkannten swallowed Errors in `auth.go:159` und `sharing_notes.go:56` fixen:
```go
// Statt: _, _ = db.Exec(...)
if _, err := db.Exec(`UPDATE refresh_tokens...`, tokenHash); err != nil {
    slog.Warn("failed to revoke refresh token", slog.String("error", err.Error()))
}
```

**Aufwand:** M

---

### K-02: God-Component `settings/+page.svelte` (1453 Zeilen)
**Betroffene Dateien:**
- `frontend/src/routes/settings/+page.svelte` (1453 Zeilen)

**Problem:** Grosse Settings-Seite mit 6+ Tabs. Tab-Komponenten existieren bereits teilweise (`AccountTab.svelte`, `AiTab.svelte`, `SecurityTab.svelte`) und werden ab Zeile 1142 eingesetzt - die Extraktion ist also **bereits im Gange**, aber noch nicht abgeschlossen. Der verbleibende Code mischt Layout, Tab-Navigation und Dialog-Koordination.

**Loesungsvorschlag:** Die begonnene Tab-Extraktion konsequent zu Ende fuehren. Verbleibende Tabs (Appearance, Import/Export, Migration etc.) in eigene Komponenten verschieben. Settings-Page wird zum reinen Container/Layout. Fuer wiederverwendbare UI-Fragmente innerhalb der Tabs bieten sich Svelte 5 `{#snippet}` / `{@render}` Blocks an (empfohlenes Composition-Pattern in Svelte 5, ersetzt Slots).

**Risiko:** Hoch - State-Sharing zwischen Tabs und Dialogen muss sauber ueber Context API oder Props geloest werden. Regression-Tests fuer alle Settings-Funktionen noetig.

**Aufwand:** L (5-8 Personentage inkl. Testing)

---

### K-03: God-Component `Editor.svelte` (1407 Zeilen)
**Betroffene Dateien:**
- `frontend/src/lib/components/Editor.svelte` (1407 Zeilen, 100+ State-Variablen)

**Problem:** Massiver Editor mit Formatting, Dialogen, Toolbar, Preview, Find/Replace, Task-Sortierung, Split-View. Alles in einer Datei. Zusaetzlich: Verwendet `import { page } from '$app/stores'` (Zeile 14), was in SvelteKit 2+ **deprecated** ist zugunsten von `$app/state`.

**Loesungsvorschlag:** Extract Component - Aufteilen in:
- `EditorCore.svelte` (CodeMirror-Integration)
- `EditorDialogs.svelte` (Table-Insert, Link-Insert etc.)
- `EditorFindReplace.svelte`
- `EditorPreview.svelte`
- Migration `$app/stores` -> `$app/state`

Fuer die Extraktion bieten sich Svelte 5 `{#snippet}` / `{@render}` Blocks als Zwischenschritt an: Markup-Fragmente lassen sich damit innerhalb der Datei strukturieren, bevor sie in separate Komponenten verschoben werden. Das reduziert das Risiko, weil sich die State-Abhaengigkeiten schrittweise entkoppeln lassen.

**Risiko:** Sehr hoch - CodeMirror-State-Sharing, Event-Propagation und CSS-Scope-Probleme bei der Extraktion. Ohne gute E2E-Tests koennen subtile State-Management-Bugs entstehen.

**Aufwand:** XL (5+ Personentage)

---

### K-04: God-Component `CanvasEditor.svelte` (1072 Zeilen)
**Betroffene Dateien:**
- `frontend/src/lib/components/CanvasEditor.svelte` (1072 Zeilen)

**Problem:** Canvas-Komponente mit Node-Manipulation, Drag & Drop, mehrere Dialog-Handler und komplexer State in einer Datei.

**Loesungsvorschlag:** Extract Module - Canvas-Logik in Sub-Module aufteilen:
- `canvas-state.ts` (State-Management)
- `canvas-drag.ts` (Drag & Drop)
- `CanvasToolbar.svelte`
- `CanvasNodeEditor.svelte`

**Aufwand:** L (3-5 Personentage)

---

### K-05: Fehlende Komponenten-Level Error-Boundaries im Frontend
**Betroffene Dateien:**
- `frontend/src/lib/components/Editor.svelte` - Preview-Rendering ohne try/catch
- `frontend/src/lib/components/GraphCanvas.svelte:82-94` - Force-Graph init ohne Error-Boundary
- `frontend/src/lib/components/CanvasEditor.svelte:83` - loadNote-Call ohne Error-Handling

**Hinweis:** Das Projekt hat bereits eine **Route-Level Error-Boundary** (`+error.svelte` mit Error-ID-Anzeige und Home/Reload-Buttons) sowie einen `handleError`-Hook in `hooks.client.ts` (Error-ID-Generierung + Console-Logging). Das Problem betrifft daher spezifisch **Komponenten-interne async-Operationen**, die bei Fehlern die Komponente still versagen lassen, ohne dass der Route-Level-Handler greift (weil kein Navigation-Error ausgeloest wird).

**Problem:** Bei Rendering-Fehlern (Markdown, Graph, Canvas) faellt die Komponente still aus oder zeigt nichts an. User bekommt keine Fehlermeldung. Besonders kritisch bei GraphCanvas, wo ein fehlgeschlagenes `import('force-graph')` den Graphen komplett verschwinden laesst.

**Loesungsvorschlag:** Try/catch-Wrapper um async Renderer-Aufrufe + standardisierter Error-Handler mit Fallback-UI:
```typescript
// lib/utils/component-error.ts
export function handleComponentError(err: unknown, context: string) {
  console.error(`[${context}]`, err);
  toast.error(getErrorMessage(err));
}
```
```svelte
{#if renderError}
  <div class="error-state">Rendering fehlgeschlagen: {renderError.message}</div>
{/if}
```

**Aufwand:** M

---

## Wichtig 🟡 (sollte bald angegangen werden)

### W-01: `UpdateNote` ist 90+ Zeilen mit komplexer Logik
**Betroffene Dateien:**
- `backend/internal/service/notes_crud.go:50-150`

**Problem:** Komplexe Logik fuer Title-Changes, Versioning, Backlink-Updates, Canvas-Links in einer Funktion.

**Loesungsvorschlag:** Extract Method:
- `updateTitleBacklinks(userID, existingNote, newTitle)` (Zeilen 85-121)
- `createSnapshotIfNeeded(userID, id, existingNote)` (Zeilen 67-83)
- `invalidateNoteCaches(userID, note)` (Zeilen 146-155)

**Aufwand:** M

---

### W-02: Duplizierter Note-Scanning-Code
**Betroffene Dateien:**
- `backend/internal/db/notes_list.go:34-80` (`scanNote`)
- `backend/internal/db/notes_list.go:83-130` (`scanNoteWithShared`)

**Problem:** 97% identischer Code - beide Funktionen scannen SQL-Rows, nur `scanNoteWithShared` hat 2 zusaetzliche Felder.

**Loesungsvorschlag:** Gemeinsamen `scanNoteBase()` extrahieren, `scanNoteWithShared()` als Wrapper:
```go
func scanNoteBase(rows *sql.Rows, userID int) (Note, error) { ... }
func scanNoteWithShared(rows *sql.Rows, userID int) (Note, error) {
    note, err := scanNoteBase(rows, userID)
    // ... zusaetzliche Felder scannen
}
```
(Vermeidet Bool-Parameter-Smell.)

**Aufwand:** S

---

### W-03: Synchrone Forgejo-HTTP-Calls im Error-Reporting
**Betroffene Dateien:**
- `backend/internal/service/errorreport.go:145-149`

**Problem:** HTTP-Calls an Forgejo-API blockieren den Request-Handler. Bei SQLite-Backend mit begrenzter Concurrency kann ein Forgejo-Timeout unter Last problematisch werden, da Request-Handler-Goroutinen blockiert bleiben.

**Loesungsvorschlag:** Error-Reports in async Job-Queue einreihen. HTTP-Operationen aus `ErrorReportService` in ein `ForgejoClient`-Interface extrahieren.

**Aufwand:** L

---

### W-04: Inkonsistente Error-Wrapping-Patterns im Backend
**Betroffene Dateien:**
- Diverse Dateien in `backend/internal/db/` und `backend/internal/service/`

**Problem:** Mix aus `fmt.Errorf("context: %w", err)`, `error.Error()` String-Matching und `errors.Is()` vs. direkter Equality (`== db.ErrNotFound`).

**Loesungsvorschlag:** Konsistentes Pattern standardisieren:
```go
// Standard: immer %w fuer Error-Wrapping
if err != nil {
    return nil, fmt.Errorf("notes.UpdateNote(id=%s): %w", id, err)
}
// Standard: immer errors.Is() fuer Error-Vergleiche
if errors.Is(err, db.ErrNotFound) { ... }
```

**Aufwand:** M

---

### W-05: `auth.svelte.ts` hat komplexen Store-Dependency-Graph
**Betroffene Dateien:**
- `frontend/src/lib/stores/auth.svelte.ts` (652 Zeilen)

**Problem:** Auth-Store importiert von encryption, features, journal, recipes, settings, ui. Aenderungen am Auth-State triggern kaskadenartige Updates ueber das gesamte Store-Netzwerk.

**Loesungsvorschlag:** Store-Interconnections in ein hoeher gelegenes Coordinator-Modul verschieben. Auth-Store sollte nur Auth-Logik enthalten.

**Aufwand:** L

---

### W-06: Inkonsistentes Error-Handling-Pattern im Frontend
**Betroffene Dateien:**
- Diverse Komponenten in `frontend/src/lib/components/`

**Problem:** Drei verschiedene Patterns im Einsatz:
1. Try/catch mit Toast-Notification (gut) - z.B. `+page.svelte:336`
2. Try/catch mit `console.error` (unzureichend) - z.B. `GraphCanvas.svelte:92`
3. Kein Error-Handling (schlecht) - z.B. `CanvasEditor.svelte:83`

**Loesungsvorschlag:** Pattern 1 als Standard etablieren. Siehe Error-Handler in K-05.

**Aufwand:** M

---

### W-07: Duplizierter Admin-User-Conversion-Loop
**Betroffene Dateien:**
- `backend/internal/api/admin.go:110-124` und `146-158`

**Problem:** Identischer Loop konvertiert `service.AdminUser` zu `AdminUserResponse` an zwei Stellen.

**Loesungsvorschlag:** Extract Helper:
```go
func toAdminUserResponse(u service.AdminUser) AdminUserResponse {
    return AdminUserResponse{...}
}
```

**Aufwand:** S

---

### W-08: `ErrorReportService` ist ein God-Object (552 Zeilen)
**Betroffene Dateien:**
- `backend/internal/service/errorreport.go` (552 Zeilen)

**Problem:** Vermischt Error-Reporting, Forgejo-Integration, HTTP-Kommunikation, Label-Management, Issue-/Comment-Erstellung.

**Loesungsvorschlag:** Extract Class - `ForgejoClient` Interface/Struct fuer HTTP-Operationen extrahieren.

**Aufwand:** M

---

### W-09: RecipeIngredientEditor Prop-Drilling (927 Zeilen)
**Betroffene Dateien:**
- `frontend/src/lib/components/RecipeIngredientEditor.svelte` (927 Zeilen, 14+ State-Variablen)

**Problem:** 10+ Callbacks werden durch 3 Ebenen durchgereicht (RecipeEditor > RecipeIngredientEditor > RecipeIngredientRow).

**Loesungsvorschlag:** Context-API fuer Ingredient-State nutzen oder Store-Modul extrahieren. Komponente aufteilen in:
- `IngredientGroupManager.svelte`
- `QuickAddIngredient.svelte`

**Aufwand:** L

---

### W-10: Dashboard-Page `+page.svelte` mit duplizierten Drag-Drop-Bloecken (953 Zeilen)
**Betroffene Dateien:**
- `frontend/src/routes/+page.svelte` (953 Zeilen)

**Problem:** Dashboard-Seite mit Layout, State-Management und Section-Reordering. Grosse Bloecke duplizierter Drag-Drop-Logik.

**Loesungsvorschlag:** Drag-Drop-Logik in wiederverwendbare Action extrahieren. Dashboard-Sections als eigene Komponenten.

**Aufwand:** M

---

### W-11: Schwache Typisierung in `api/types.ts`
**Betroffene Dateien:**
- `frontend/src/lib/api/types.ts` (878 Zeilen)

**Problem:** `SummarizeRequest` hat 3 optionale Felder, wo genau 1 required sein sollte. Keine Discriminated Union.

**Loesungsvorschlag:** Discriminated Union Types:
```typescript
type SummarizePayload =
  | { plaintext_content: string }
  | { plaintext_content_hash: string; encrypted_summary: string };
```

**Aufwand:** M

---

### W-12: Docker-Image-Digest veraltet
**Betroffene Dateien:**
- `/workspace/Dockerfile:6`

**Problem:** SHA256-Digest des `node:22-alpine` Images braucht regelmaessige Aktualisierung (Kommentar sagt "Update quarterly", aber kein Datum).

**Loesungsvorschlag:** Datum-Kommentar hinzufuegen und quartalmaessig aktualisieren:
```dockerfile
# Last updated: 2026-02-24
FROM node:22-alpine@sha256:...
```

**Aufwand:** S

---

### W-13: Fehlende Node-Engine-Constraint
**Betroffene Dateien:**
- `frontend/package.json`

**Problem:** CI nutzt Node 22, aber `package.json` hat kein `engines`-Feld. Entwickler koennten lokal andere Versionen nutzen.

**Loesungsvorschlag:**
```json
"engines": {
  "node": ">=22.0.0",
  "npm": ">=10.0.0"
}
```

**Aufwand:** S

---

### W-14: `npm audit` in CI prueft nur Production-Dependencies
**Betroffene Dateien:**
- `.github/workflows/security.yml:55`

**Problem:** `npm audit --omit=dev --audit-level=high` ignoriert dev-Dependencies mit Vulnerabilities.

**Loesungsvorschlag:** Auch dev-Dependencies pruefen:
```yaml
- name: Run npm audit (all)
  run: npm audit --audit-level=moderate
```

**Aufwand:** S

---

### W-15: `$app/stores` deprecated Imports
**Betroffene Dateien:**
- `frontend/src/lib/components/Editor.svelte:14`
- `frontend/src/routes/+error.svelte:5`

**Problem:** `import { page } from '$app/stores'` ist seit SvelteKit 2.12 deprecated zugunsten von `$app/state`. Projekt nutzt SvelteKit ^2.50.0 - die neue API ist also verfuegbar. Wird bei zukuenftigen Major-Updates brechen.

**Loesungsvorschlag:** Migration auf `$app/state`. Achtung: Die Syntax aendert sich leicht - `$page.status` (Store Auto-Subscription) wird zu `page.status` (direkter Property-Zugriff, kein `$`-Prefix mehr):
```typescript
// Alt:
import { page } from '$app/stores';
const status = $derived($page.status);
// Neu:
import { page } from '$app/state';
const status = $derived(page.status);
```

**Aufwand:** S

---

## Nice-to-have 🟢 (bei Gelegenheit)

### N-01: Magic Numbers in `recipe_generator.go`
**Betroffene Dateien:**
- `backend/internal/service/recipe_generator.go:101-126`

**Problem:** Hardcodierte Limits: 999 (max Servings), 2048 (max URL-Laenge), 200 (max Ingredient-Name).

**Loesungsvorschlag:** Named Constants extrahieren:
```go
const (
    MaxRecipeServings       = 999
    MaxRecipeSourceURLLen   = 2048
    MaxIngredientNameLen    = 200
)
```

**Aufwand:** S

---

### N-02: Magic Numbers in `search.go` und `htmlutil/fetch.go`
**Betroffene Dateien:**
- `backend/internal/db/search.go:18-23` (20, 100, 500, 5s)
- `backend/internal/htmlutil/fetch.go:19-23` (2 << 20, 50000, 3, 15s)

**Loesungsvorschlag:** Named Constants.

**Aufwand:** S

---

### N-03: Magic Numbers im Frontend
**Betroffene Dateien:**
- `frontend/src/lib/components/TableOfContents.svelte` - `'(max-width: 640px)'` hardcoded
- `frontend/src/lib/components/RecipeMetadataForm.svelte` - `val >= 1 && val <= 999` an 2+ Stellen
- `frontend/src/lib/components/RecipeScaleControl.svelte` - gleiches Range-Pattern

**Loesungsvorschlag:** Breakpoints in Config-Modul, MIN/MAX_SERVINGS Konstanten.

**Aufwand:** S

---

### N-04: Dupliziertes Resize-Handler-Pattern
**Betroffene Dateien:**
- `frontend/src/lib/components/Sidebar.svelte:82-137`
- `frontend/src/lib/editor/split-resize.ts`

**Loesungsvorschlag:** Wiederverwendbare `useResizable()` Action erstellen.

**Aufwand:** M

---

### N-05: Dupliziertes Drag-Indicator-Pattern
**Betroffene Dateien:**
- `frontend/src/lib/actions/touchdrag.ts:133-160`
- `frontend/src/lib/components/canvas/canvas-note-drop.ts`

**Loesungsvorschlag:** Drop-Indicator-Logik in shared Utility extrahieren.

**Aufwand:** S

---

### N-06: Encryption-Salt-Handling dupliziert
**Betroffene Dateien:**
- `backend/internal/db/auth.go:74-77` (`GetUserByID`)
- `backend/internal/db/auth.go:102-105` (`GetUserByUsernameOrEmail`)

**Loesungsvorschlag:** Helper-Funktion `decodeEncryptionSalt(raw)` extrahieren.

**Aufwand:** S

---

### N-07: Sidebar.svelte ist 909 Zeilen
**Betroffene Dateien:**
- `frontend/src/lib/components/Sidebar.svelte` (909 Zeilen)

**Loesungsvorschlag:** Resize-Logik und Tree-Rendering in separate Module extrahieren.

**Aufwand:** M

---

### N-08: `touchdrag.ts` State-Machine mit tiefer Verschachtelung (464 Zeilen)
**Betroffene Dateien:**
- `frontend/src/lib/actions/touchdrag.ts:189-300+`

**Loesungsvorschlag:** Explizite State-Machine-Methoden (`enterDragging()`, `exitDragging()`, `handleDragMove()`).

**Aufwand:** M

---

### N-09: `vite.config.ts` - Chunk-Rule fuer nicht-existierendes Package
**Betroffene Dateien:**
- `frontend/vite.config.ts:233`

**Problem:** Manual-Chunk-Rule fuer `bits-ui` definiert, aber Package scheint nicht in Dependencies.

**Loesungsvorschlag:** Pruefen und ggf. entfernen.

**Aufwand:** S

---

### N-10: `vite.config.ts` - `target: 'esnext'` weicht vom Vite-Default ab
**Betroffene Dateien:**
- `frontend/vite.config.ts:180`

**Problem:** Vite 6+ nutzt standardmaessig `baseline-widely-available` als Build-Target (Chrome 111+, Firefox 114+, Safari 16.4+). Das Projekt setzt explizit `'esnext'`, was aeltere Browser ausschliesst.

**Loesungsvorschlag:** Bewusste Entscheidung treffen:
- Falls das Projekt primaer PWA/Electron/Tauri-Targets hat: `'esnext'` kann bewusst beibehalten werden (als Kommentar dokumentieren: `// Intentional: PWA/Electron/Tauri only`)
- Falls breitere Browser-Kompatibilitaet gewuenscht: Target entfernen (Vite-Default `baseline-widely-available` nutzen) oder explizit auf `'baseline-widely-available'` setzen
- **Nicht** `'es2020'` nutzen - das ist veraltet und unnoetig restriktiv

**Aufwand:** S

---

### N-11: `chunkSizeWarningLimit` auf 1000 KB erhoeht
**Betroffene Dateien:**
- `frontend/vite.config.ts:182`

**Problem:** Limit erhoeht wegen CodeMirror (~931 KB). Versteckt ggf. echte Bloat-Probleme.

**Loesungsvorschlag:** Auf 800 KB reduzieren und CodeMirror-Chunks weiter optimieren.

**Aufwand:** S

---

### N-12: `gosec` in golangci-lint aktivieren
**Betroffene Dateien:**
- `/workspace/.golangci.yml`

**Problem:** Go-Security-Linter `gosec` ist nicht aktiviert. Projekt hat bereits 9 aktive Linter inkl. `errcheck` und `staticcheck`, die viele Security-Issues abdecken.

**Loesungsvorschlag:** In `.golangci.yml` unter `linters.enable` hinzufuegen mit Test-Exclusions.

**Aufwand:** S

---

### N-13: `otpauth` devDependency moeglicherweise ungenutzt
**Betroffene Dateien:**
- `frontend/package.json` (devDependencies)

**Problem:** `otpauth` ist als devDependency installiert, kein Import in Source-Code gefunden. Beeinflusst Production-Bundle **nicht**, ist aber unnoetige devDependency.

**Loesungsvorschlag:** Pruefen ob in Tests oder Build-Scripts referenziert. Falls nicht: `npm uninstall otpauth`.

**Aufwand:** S

---

### N-14: Fehlende `.prettierignore`
**Betroffene Dateien:**
- `frontend/` (Datei fehlt)

**Loesungsvorschlag:** Erstellen mit node_modules, dist, build, coverage, .svelte-kit, playwright-report.

**Aufwand:** S

---

### N-15: Go-Version an 3 Stellen in CI hardcoded
**Betroffene Dateien:**
- `.github/workflows/ci.yml:20,36,105`

**Loesungsvorschlag:** GitHub-Actions-Variable `GO_VERSION` nutzen.

**Aufwand:** S

---

### N-16: `docker-compose.yml` JWT_SECRET ohne Fehler bei fehlendem Wert
**Betroffene Dateien:**
- `/workspace/docker-compose.yml:14`

**Loesungsvorschlag:** `JWT_SECRET=${JWT_SECRET:?JWT_SECRET is required}`

**Aufwand:** S

---

### N-17: Naming-Inkonsistenzen im Backend
**Betroffene Dateien:**
- Diverse Backend-Dateien

**Problem:** Abkuerzungen gemischt: `tf`, `tfa`, `2fa`.

**Loesungsvorschlag:** Naming-Convention dokumentieren und schrittweise vereinheitlichen.

**Aufwand:** M

---

### N-18: Inline-Arrays die bei jedem Render neu erstellt werden
**Betroffene Dateien:**
- `frontend/src/lib/components/Sidebar.svelte:59` - `const sortOptions` inline

**Loesungsvorschlag:** Auf Modul-Ebene als Konstante definieren.

**Aufwand:** S

---

### N-19: Fehlende Precompression im Svelte-Build
**Betroffene Dateien:**
- `frontend/svelte.config.js:14`

**Loesungsvorschlag:** `precompress: process.env.NODE_ENV === 'production'`

**Aufwand:** S

---

### N-20: `cookie`-Override in `package.json`
**Betroffene Dateien:**
- `frontend/package.json:96-98`

**Loesungsvorschlag:** Upstream-Dependency identifizieren (`npm ls cookie`), Issue melden oder auf Update warten.

**Aufwand:** S

---

### N-21: SQLite-Dev-Package in CI nicht gecacht
**Betroffene Dateien:**
- `.github/workflows/ci.yml:25`

**Loesungsvorschlag:** Caching oder System-Image mit vorinstalliertem Package.

**Aufwand:** S

---

### N-22: Markdown-Lint-Excludes sollten in .gitignore
**Betroffene Dateien:**
- `/workspace/lefthook.yml:28`

**Loesungsvorschlag:** `SALT_BUG_FIX.md`, `ENCRYPTION_DEBUG_LOG.md` in `.gitignore` verschieben.

**Aufwand:** S

---

## Priorisierter Umsetzungsplan

### Phase 1: Quick Wins (1-2 Tage)
| ID | Aufwand | Beschreibung |
|----|---------|-------------|
| K-01 | M | errcheck-Exclusion fixen + swallowed Errors beheben |
| W-02 | S | Note-Scanning-Code konsolidieren |
| W-07 | S | Admin-User-Conversion-Helper extrahieren |
| W-12 | S | Docker-Digest-Datum dokumentieren |
| W-13 | S | Node-Engine-Constraint hinzufuegen |
| W-14 | S | npm audit in CI erweitern |
| W-15 | S | `$app/stores` -> `$app/state` migrieren |

### Phase 2: Architektur-Verbesserungen (3-5 Tage)
| ID | Aufwand | Beschreibung |
|----|---------|-------------|
| K-05 | M | Komponenten-Level Error-Boundaries + Fallback-UI |
| W-01 | M | `UpdateNote` aufteilen |
| W-04 | M | Error-Wrapping standardisieren |
| W-06 | M | Frontend Error-Handling vereinheitlichen |
| W-08 | M | `ErrorReportService` aufteilen |
| W-10 | M | Dashboard Drag-Drop-Duplikation reduzieren |
| W-11 | M | API-Types mit Discriminated Unions verbessern |

### Phase 3: Grosse Refactorings (15-25 Personentage)

> **Achtung:** Diese Schaetzungen sind realistischer als die urspruenglichen 5-8 Tage.
> Jedes grosse Refactoring erfordert Regression-Testing und kann subtile Bugs einfuehren.

| ID | Aufwand | Beschreibung |
|----|---------|-------------|
| K-02 | L (5-8 PT) | Settings-Page Tab-Extraktion abschliessen |
| K-03 | XL (5+ PT) | Editor-Component aufteilen (hohes Risiko: CodeMirror State-Sharing) |
| K-04 | L (3-5 PT) | Canvas-Editor aufteilen |
| W-03 | L | Forgejo-Calls async machen |
| W-05 | L | Auth-Store-Dependencies entkoppeln |
| W-09 | L | RecipeIngredientEditor aufteilen |

**Voraussetzung fuer Phase 3:** Ausreichende E2E-Tests vorhanden, um Regressions zu erkennen. Stufenweises Vorgehen - jeweils ein Refactoring abschliessen und deployen, bevor das naechste beginnt.

### Phase 4: Nice-to-have (fortlaufend)
Alle N-* Issues koennen bei Gelegenheit oder im Rahmen anderer Arbeiten umgesetzt werden.

---

## Korrekturprotokoll

Folgende Issues aus der Erstanalyse waren **False Positives** und wurden entfernt:

| Original-ID | Behauptung | Realitaet |
|-------------|-----------|-----------|
| K-01 (alt) | Service-Layer hat nur 2 Test-Dateien | **19 Test-Dateien** vorhanden (notes_test, auth_test, sharing_test, etc.) |
| K-02 (alt) | LLM/htmlutil: 0 Tests | LLM hat **6 Test-Dateien**, htmlutil hat **1** |
| K-06 (alt) | Makefile `go vet` fehlt `sqlite_crypt` Tag | Tag ist korrekt gesetzt (Zeile 118) |
| K-08 (alt) | Recipe-Erstellung ohne Transaction | `CreateRecipeNoteWithIngredients` nutzt **bereits `tx, err := db.Begin()`** |
| K-09 (alt) | `qrcode` ungenutzt | Wird per **dynamischem Import** in `TwoFactorSetup.svelte` verwendet |

Folgende Issues wurden **herabgestuft**:

| Original-ID | Original-Severity | Neue Severity | Grund |
|-------------|-------------------|---------------|-------|
| K-10 (alt) | Kritisch | Nice-to-have (N-12) | Projekt hat bereits 9 aktive Linter; gosec wuerde primaer Style-Warnings erzeugen |
| W-06 (alt) | Wichtig | Entfernt | `buildDecorations()` importiert bereits aus 8 separaten Modulen - ist eine Orchestrierungsfunktion |
| W-07 (alt) | Wichtig | Entfernt | Accessor-Pattern folgt **dokumentierter Konvention** in `docs/conventions.md`; Notes-Store ist bereits in 13+ Sub-Module aufgeteilt |
| W-03 (alt) | Wichtig | Entfernt | `RotateRefreshToken` hat primaer lineare `if/err`-Checks (Standard-Go-Idiom), maximal 2-3 echte Nesting-Ebenen |

Folgende Issues wurden **neu hinzugefuegt**:

| Neue ID | Severity | Beschreibung |
|---------|----------|-------------|
| K-01 (Root Cause) | Kritisch | `.golangci.yml` errcheck-Exclusion fuer `db.Exec` als eigentliche Ursache identifiziert |
| K-01 (Row.Scan) | Kritisch | `(*database/sql.Row).Scan`-Exclusion als zusaetzliches Risiko identifiziert |
| W-15 | Wichtig | `$app/stores` deprecated Imports (Editor.svelte + `+error.svelte`) |

Folgende Issues wurden durch **Context7-/Technik-Check korrigiert** (2026-02-24):

| ID | Korrektur |
|----|-----------|
| K-05 | Ergaenzt: Route-Level Error-Handling (`+error.svelte`, `handleError`-Hook) existiert bereits; Issue betrifft spezifisch Komponenten-Level async-Errors |
| K-02, K-03 | Svelte 5 `{#snippet}` / `{@render}` als empfohlenes Composition-Pattern fuer Component-Splitting ergaenzt |
| W-15 | `+error.svelte:5` als zusaetzlich betroffene Datei ergaenzt; Migration-Syntax-Hinweis (`$page.status` → `page.status`) hinzugefuegt |
| N-10 | Korrigiert: `es2020` war veralteter Vorschlag; Vite 6+ Default ist `baseline-widely-available`; `esnext` kann bei PWA/Electron/Tauri-Target bewusst sein |
