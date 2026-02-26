# Optimierungsmoeglichkeiten – Februar 2026

> Erstellt am 2026-02-25 basierend auf Codebase-Analyse, Playwright Design-Tests und Backend-Review.

## Uebersicht

| Prioritaet | Kategorie | Anzahl | Aufwand |
|------------|-----------|--------|---------|
| P0 | Bugs (Datenintegritaet) | 7 Dateien | Niedrig |
| P1 | Performance + CSS + Design-Tokens | 14 Stellen | Niedrig-Mittel |
| P2 | Komponentenstruktur + Cleanup | 11 Stellen | Mittel |
| P3 | Kleine Verbesserungen | 5 Stellen | Niedrig |

---

## P0 – Bugs (Datenintegritaet)

### Stille Timestamp-Parsing-Fehler im Backend

In 7 Dateien werden `time.Parse`-Fehler mit `_, _ =` ignoriert. Wenn ein Datenbankwert malformed ist, wird stillschweigend ein Zero-Timestamp (`0001-01-01`) gesetzt.

| Datei | Zeilen | Pattern |
|-------|--------|---------|
| `backend/internal/db/folders_queries.go` | 45-46, 93-94, 132-133 | `f.CreatedAt, _ = time.Parse(...)` |
| `backend/internal/db/recipes_collections.go` | Diverse | Gleiche Pattern |
| `backend/internal/db/recipes_notes.go` | Diverse | Gleiche Pattern |
| `backend/internal/db/recipes_sharing.go` | Diverse | Gleiche Pattern |
| `backend/internal/db/search.go` | Diverse | Gleiche Pattern |
| `backend/internal/db/versions.go` | Diverse | Gleiche Pattern |
| `backend/internal/db/task_events.go` | Diverse | Gleiche Pattern |

**Fix:**
```go
// Vorher (fehlerhaft):
f.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

// Nachher:
f.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
if err != nil {
    return nil, fmt.Errorf("failed to parse created_at: %w", err)
}
```

---

## P1 – Performance (Backend)

### Recipe-Detail N+1 Query Pattern

**Datei:** `backend/internal/service/recipes_notes.go:128-147`

Jeder Rezept-Aufruf macht 5 separate Queries:

1. `GetNote` – Notiz laden
2. `GetRecipeMetadata` – Metadaten
3. `GetRecipeIngredients` – Zutaten
4. `GetRecipeImages` – Bilder
5. `GetCollectionsForRecipe` – Sammlungen

**Fix:** Zusammenfuehren zu 1-2 Queries mit JOINs (`GetRecipeDetailFull()`). Geschaetzter Impact: 80-400% weniger Queries pro Rezeptaufruf.

---

## P1 – CSS-Optimierung

### Duplizierte Theme-Variablen

**Datei:** `frontend/src/app.css`

Der `@theme`-Block (Zeilen 284-343) und `.theme-gruvbox-light` (Zeilen 346-394) definieren **identische** 25+ CSS-Variablen.

**Fix:** `.theme-gruvbox-light`-Block entfernen – `@theme` ist bereits der Default. Einsparung: ca. 3-4KB.

### Ungenutzte CSS-Animationen

**Datei:** `frontend/src/app.css`

| Animation | Definiert | Verwendet |
|-----------|-----------|-----------|
| `slideDown` | Ja | Nein |
| `slideRight` | Ja | Nein |
| `slideLeft` | Ja | Nein |
| `heightExpand` | Ja | Nein |
| `heightCollapse` | Ja | Nein |

**Fix:** Entfernen. Einsparung: ca. 2KB.

---

## P1 – Hardcoded Farben (Design-Token-Bypasses)

12 Komponenten verwenden direkte Tailwind-Farben statt Design-Tokens. Diese brechen die Theme-Konsistenz bei kuenftigen Theme-Aenderungen.

### Schnelle Fixes (je < 5 Min.)

| Datei | Zeile | Ist | Soll |
|-------|-------|-----|------|
| `components/Toast.svelte` | 24 | `bg-yellow-500 text-white` | `bg-warning text-warning-foreground` |
| `components/ui/AlertDialog.svelte` | 24 | `bg-yellow-500 text-white` | `bg-warning text-warning-foreground` |
| `components/ui/AlertDialog.svelte` | 46 | `text-yellow-500` | `text-warning` |
| `components/AITransformDialog.svelte` | 275 | `text-green-500` | `text-success` |
| `components/editor/EditorToolbar.svelte` | 188 | `bg-blue-500 text-white` | `bg-primary text-primary-foreground` |
| `components/editor/EditorToolbar.svelte` | 198 | `text-blue-500 bg-blue-500/12` | `text-primary bg-primary/12` |
| `components/canvas/CanvasEditorToolbar.svelte` | 134 | `bg-blue-500 text-white` | `bg-primary text-primary-foreground` |
| `components/canvas/CanvasLinkDialog.svelte` | 39 | `text-red-600` | `text-destructive` |
| `components/SessionRestoreBanner.svelte` | 9 | `bg-blue-600/95 text-white` | `bg-primary/95 text-primary-foreground` |
| `components/OfflineBanner.svelte` | 33 | `bg-blue-500`, `bg-amber-500/600` | `bg-primary`, `bg-warning` |

### Mittlerer Aufwand

| Datei | Zeile | Ist | Soll |
|-------|-------|-----|------|
| `components/FilterChip.svelte` | 21 | `text-purple-600` | Neuer Token `--color-filter-active` |
| `components/FilterMenu.svelte` | 103 | `text-purple-600 dark:text-purple-400` | Konsistenter Token |

### Akzeptabel (semantisch korrekt im Kontext)

| Datei | Kontext | Begruendung |
|-------|---------|-------------|
| `ConflictDialog.svelte` | Git-artige Diff-Ansicht | Gruen/Rot fuer Added/Removed ist universell |
| `VersionHistoryDialog.svelte` | Versions-Diff | Gleiche Konvention |
| `settings/encryption/+page.svelte` | Sicherheitswarnungen | Hat korrekte `dark:` Varianten |
| `settings/migration/+page.svelte` | Datenmigration | Hat korrekte `dark:` Varianten |

---

## P2 – Komponentenstruktur

### Grosse Komponenten aufteilen

| Datei | Zeilen | Empfehlung |
|-------|--------|------------|
| `components/Editor.svelte` | 1.269 | Toolbar, Find/Replace, Preview extrahieren |
| `components/CanvasEditor.svelte` | 955 | Kontextmenu, Toolbar, Node-Handlers extrahieren |
| `components/Sidebar.svelte` | 909 | Drag/Drop + Resize-Handler in Module auslagern |
| `routes/login/+page.svelte` | 870 | Auth-Methoden (Passwort, FIDO2, Captcha) trennen |
| `routes/settings/+page.svelte` | 896 | Tab-State-Management extrahieren |
| `components/VersionHistoryDialog.svelte` | 737 | Versionsliste + Diff-View trennen |
| `components/RecipeEditor.svelte` | 706 | Tab-Switching-Logik extrahieren |
| `components/GraphCanvas.svelte` | 695 | Rendering von Controls/Interaktionen trennen |
| `components/UnifiedTree.svelte` | 641 | Drag/Drop + Keyboard-Navigation in Module |
| `components/tree/TreeNodeRow.svelte` | 593 | 23 Props – Folder/Note-Rendering trennen |

### Wiederverwendbare Alert/Banner-Komponente

11+ Stellen nutzen dasselbe Pattern:

```svelte
<div class="ui-panel-soft p-4 bg-yellow-50 border-yellow-200 dark:bg-yellow-900/20 dark:border-yellow-800">
  <AlertTriangle class="w-5 h-5 text-yellow-600 dark:text-yellow-400" />
  <span class="font-semibold text-yellow-800 dark:text-yellow-300">Warnung</span>
</div>
```

Betroffene Dateien:
- `routes/settings/encryption/+page.svelte` (4x)
- `routes/settings/migration/+page.svelte` (3x)
- `components/AITransformDialog.svelte` (1x)
- `components/VersionHistoryDialog.svelte` (1x)

**Fix:** `AlertBox.svelte` mit Varianten (warning, error, info, success) erstellen. Einsparung: ca. 200 Zeilen Code.

---

## P2 – Backend-Cleanup

### validThemes aufraeumen

**Datei:** `backend/internal/service/user_types.go:47-71`

22 Theme-IDs validiert, nur 2 im Frontend aktiv (`gruvbox-light`, `gruvbox-dark`).

**Migrationsstrategie:**
1. Bestehende User-Praeferenzen mit alten Theme-IDs auf `gruvbox-light`/`gruvbox-dark` mappen
2. Alte IDs aus `validThemes` entfernen
3. Frontend und Backend synchron halten

### Token-Revocation ohne Logging

**Datei:** `backend/internal/service/auth.go:220,249`

```go
_ = s.db.RevokeRefreshTokenFamilyByToken(refreshToken)
```

Bei Token-Reuse-Detection wird das Cleanup-Ergebnis ignoriert. Mindestens Warn-Log hinzufuegen:

```go
if err := s.db.RevokeRefreshTokenFamilyByToken(refreshToken); err != nil {
    s.logger.Warn("failed to revoke token family", "error", err)
}
```

---

## P3 – Kleine Verbesserungen

### Mobile UX

| Bereich | Detail | Datei |
|---------|--------|-------|
| Touch-Targets | Sidebar-Buttons 32px → 44px (WCAG AA) | `components/Sidebar.svelte` |
| Font-Size | BottomNav-Labels 11px → 12px (Mobile-Minimum) | `components/MobileBottomNav.svelte` |
| Heading-Hierarchie | Settings: h1→h3 Sprung → h2 einfuegen | `routes/settings/+page.svelte` |
| Settings Mobile | Tab-Icons ohne Labels → Tooltip ergaenzen | `routes/settings/+page.svelte` |

### Inline-Styles → CSS-Klassen

| Pattern | Vorkommen | Fix |
|---------|-----------|-----|
| `-webkit-tap-highlight-color: transparent` | 3 Mobile-Komponenten | CSS-Klasse `.no-tap-highlight` |
| `min-width: 44px; min-height: 44px` | `IOSInstallCoach.svelte` | CSS-Klasse `.touch-target-min` |

---

## Kein Handlungsbedarf (bereits gut optimiert)

| Bereich | Status |
|---------|--------|
| Bundle-Splitting | CodeMirror, Graph, Crypto, Mermaid, Shiki lazy-loaded |
| Lazy Loading | Alle schweren Komponenten korrekt lazy-loaded |
| Event Cleanup | Konsistent – keine Memory Leaks gefunden |
| SQLite Pragmas | WAL, busy_timeout, synchronous=NORMAL, PRAGMA optimize() |
| Connection Pooling | MaxOpenConns(1) – absichtlich fuer SQLite-Serialisierung |
| Middleware | Intelligente Timeouts je Route-Typ |
| Font Loading | Alle 4 Gewichte (400/500/600/700) werden aktiv genutzt |
| Dependencies | Keine ungenutzten Abhaengigkeiten |
| $derived Reaktivitaet | Kein teures Computing in Derivations |
| Speicher-Allokationen | Alle Slices/Maps korrekt begrenzt |
| DB-Indizes | Alle relevanten Indizes vorhanden |
