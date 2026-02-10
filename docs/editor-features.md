# Editor Features

Übersicht über die erweiterten Editor-Funktionen in xelanote.

## Wikilink Autocomplete

### Beschreibung
Die Wikilink-Autocomplete ermöglicht es, schnell Verknüpfungen zu anderen Notizen zu erstellen, ähnlich wie in Obsidian oder Roam Research.

### Verwendung

1. **Autocomplete starten**: Tippe `[[` im Editor
2. **Notizen durchsuchen**:
   - Bei leerem Query: Zeigt die zuletzt bearbeiteten Notizen
   - Bei Eingabe: Filtert Notizen nach Titel (z.B. `[[meet` → "Meeting Notes 2026")
3. **Notiz auswählen**:
   - Mit Pfeiltasten navigieren oder mit Maus klicken
   - Enter drücken zum Einfügen
   - Der Wikilink wird automatisch geschlossen: `[[Notiz-Titel]]`

### Technische Details

**Frontend** (`frontend/src/lib/editor/wikilink-autocomplete.ts`):
- CodeMirror Extension mit `@codemirror/autocomplete`
- Trigger-Pattern: `/\[\[([^\]|]*)$/` (matcht `[[` gefolgt von beliebigem Text außer `]` und `|`)
- API-Integration mit `/api/quick-search`
- Caching für bessere Performance
- Validierung mit: `/^[^\]|]*$/` (erlaubt keine `]` oder `|` Zeichen)

**Backend** (`backend/internal/api/search.go`, `backend/internal/db/search.go`):
- Leere Query → Gibt zuletzt bearbeitete Notizen zurück (sortiert nach `updated_at DESC`)
- Mit Query → Filtert nach `title_norm LIKE %query%`
- Prefix-Matching bevorzugt (Notizen, die mit Query beginnen, werden zuerst angezeigt)
- Limit: 10 Notizen (konfigurierbar bis max. 50)

### Debug-Logging

Console-Logs zum Troubleshooting:
```
[Wikilink Autocomplete] Extension created
[Wikilink Autocomplete] Function called at pos: X explicit: false
[Wikilink Autocomplete] textBefore: [[ match: Array [ "[[", "" ]
[Wikilink Autocomplete] query: (empty) from: 82 pos: 82
[Wikilink Autocomplete] Searching for: (empty)
[Wikilink Autocomplete] Results: 7
[Wikilink Autocomplete] Showing 7 results
```

### Bekannte Einschränkungen

- Funktioniert nur mit nicht-gelöschten Notizen (`is_deleted = 0`)
- Sucht nur in Titeln, nicht im Content
- Keine Fuzzy-Search (nur LIKE-Pattern-Matching)
- Berücksichtigt keine verschlüsselten Titel

---

## Preview Theme Selector

### Beschreibung
Wechsle zwischen verschiedenen Markdown-Preview-Themes für eine bessere Lesbarkeit.

### Verfügbare Themes

xelanote bietet **23 Themes** (11 Light, 12 Dark) mit OKLch-Farbsystem:

**Light Themes:** Default, Solarized Light, Ayu Light, Rosé Pine Dawn, Everforest Light, One Light, u.a.

**Dark Themes:** Nord, Dracula, Catppuccin, Gruvbox, Tokyo Night, Monokai, One Dark, Ayu Mirage, Rosé Pine Moon, Kanagawa, Everforest Dark, Dark Pastels

Vollständige Liste mit CSS-Variablen: **[Design System & Themes](./design-system.md)**

### Verwendung

1. **Theme wechseln**: Klicke auf das **Augen-Icon** (Eye) in der Editor-Toolbar
2. **Theme auswählen**: Wähle ein Theme aus dem Dropdown-Menü
3. **Einstellung wird gespeichert**: Das gewählte Theme bleibt über Sessions hinweg aktiv (localStorage)

### Technische Details

**Komponente**: `frontend/src/lib/components/PreviewThemeSelector.svelte`
- Dropdown mit Theme-Optionen
- Visueller Indikator (●) für aktuell gewähltes Theme
- Click-Outside-Detection zum automatischen Schließen

**Theme-Definitionen**: `frontend/src/lib/themes/index.ts`
- Zentrale Theme-Registry
- Jedes Theme hat ID, Name und CSS-Klasse
- Theme-Klassen definiert in `frontend/src/app.css`

**State Management**: `frontend/src/lib/stores/ui.svelte.ts`
- `previewThemeId`: Aktuelles Theme (reactive)
- `setPreviewTheme(id)`: Theme wechseln
- `getPreviewThemeId()`: Aktuelles Theme abrufen
- Persistierung in localStorage unter `previewTheme`

**CSS**: `frontend/src/app.css`
- Theme-spezifische Styles mit Selektoren wie `.preview-theme-github`
- Überschreibt Standard-Markdown-Styles (Headings, Links, Code, etc.)

---

## Focus Mode Extensions

### Typewriter Mode

**Beschreibung**: Hält die aktuelle Zeile vertikal zentriert im Editor-Viewport, sodass du immer in der Mitte des Bildschirms schreibst.

**Verwendung**:
- Aktivierung über UI-Store: `ui.setTypewriterMode(true)`
- Deaktivierung: `ui.setTypewriterMode(false)`

**Technische Details**: `frontend/src/lib/editor/focus-mode-extensions.ts`
- ViewPlugin mit `scrollIntoView` für aktive Zeile
- Nur aktiv bei Cursor-Bewegung (nicht bei Scroll)

### Dim Inactive Lines

**Beschreibung**: Blendet alle Zeilen außer der aktiven Zeile leicht aus, um Ablenkungen zu minimieren.

**Verwendung**:
- Aktivierung: `ui.setDimInactiveLines(true)`
- Deaktivierung: `ui.setDimInactiveLines(false)`

**Technische Details**:
- Fügt CSS-Klasse `.cm-dim` zu inaktiven Zeilen hinzu
- Opacity: 0.5 für nicht-aktive Zeilen
- Kombinierbar mit Typewriter Mode

---

## Table of Contents (TOC)

### Beschreibung
Automatisch generiertes Inhaltsverzeichnis basierend auf Markdown-Überschriften.

### Features

- **Auto-Update**: TOC aktualisiert sich automatisch beim Editieren
- **Hierarchische Darstellung**: H1, H2, H3, etc. werden eingerückt dargestellt
- **Sprungmarken**: Klick auf Heading scrollt zur entsprechenden Zeile im Editor
- **Visueller Indikator**: Aktiver Heading wird hervorgehoben

### Verwendung

Das TOC wird automatisch im Editor-Sidebar angezeigt (wenn aktiviert).

### Technische Details

**Komponente**: `frontend/src/lib/components/TableOfContents.svelte`

**Markdown-Parsing**: `frontend/src/lib/editor/markdown.ts`
```typescript
export function extractHeadings(markdown: string): Heading[]
```
- Parst Markdown mit Regex: `/^(#{1,6})\s+(.+)$/gm`
- Extrahiert Level (1-6) und Text
- Gibt Array von `{ level, text, line }` zurück

**Integration**:
```svelte
<TableOfContents
  headings={$derived(extractHeadings(content))}
  currentLine={editorCurrentLine}
  on:jumpToLine={(e) => editor.jumpToLine(e.detail)}
/>
```

---

## Breadcrumb Navigation

### Beschreibung
Zeigt den Pfad der aktuellen Notiz in der Ordnerstruktur an.

### Features

- **Folder-Pfad**: Zeigt vollständigen Pfad (z.B. `/docs/api`)
- **Klickbare Segmente**: Jedes Folder-Segment ist klickbar
- **Responsive**: Passt sich an verfügbare Breite an

### Verwendung

Die Breadcrumb wird automatisch oben im Editor angezeigt.

### Technische Details

**Komponente**: `frontend/src/lib/components/Breadcrumb.svelte`

**Props**:
- `path: string` - Folder-Pfad (z.B. `/docs/api`)
- Event: `navigate` - Wird beim Klick auf ein Folder ausgelöst

**Path-Parsing**: `frontend/src/lib/stores/tree.svelte.ts`
```typescript
export function parseFolderPath(path: string): string[]
```
- Teilt Pfad in Segmente auf
- Entfernt leere Segmente
- Root (`/`) wird als separates Segment behandelt

**Styling**:
- Separator: `/` zwischen Segmenten
- Hover-Effekt für klickbare Segmente
- Last Segment (aktueller Folder) nicht klickbar

---

## Farbmarkierungen für Ordner und Notizen

### Beschreibung
Markiere Ordner und Notizen in der Sidebar mit benutzerdefinierten Farben. Ähnlich wie VS Code zeigt xelanote einen farbigen Balken links am Element an, um wichtige Items visuell hervorzuheben.

### Features

- **VS Code-Style**: Vertikaler Farbbalken am linken Rand des Items
- **Palette & Custom Colors**: Wähle aus vordefinierten Design-Token-Farben oder nutze eigene Hex-Farben
- **Hover-Interaktion**: Palette-Icon wird beim Hover über Sidebar-Items angezeigt
- **Persistenz**: Farben werden in der Datenbank gespeichert (Spalte `color` in `folders` und `notes`)
- **Null-Wert**: Farbe kann jederzeit entfernt werden

### Verwendung

1. **Farbmarkierung hinzufügen**:
   - Bewege die Maus über einen Ordner oder eine Notiz in der Sidebar
   - Klicke auf das **Palette-Icon** (erscheint beim Hover)
   - Wähle eine Farbe aus:
     - **Palette-Tab**: Vordefinierte Farben (Primär, Warnung, Akzent, etc.)
     - **Eigene-Tab**: Hex-Farbcode eingeben (z.B. `#ff0000`) oder System-Farbwähler nutzen

2. **Farbmarkierung ändern**:
   - Klicke erneut auf das Palette-Icon beim Item
   - Wähle eine neue Farbe

3. **Farbmarkierung entfernen**:
   - Im Color Picker gibt es eine "Entfernen"-Option (oder setze Farbe auf `null` via API)

### Technische Details

**Migration**: `backend/internal/db/migrations/023_add_color_field.sql`
- Fügt `color TEXT DEFAULT NULL` Spalte zu `folders` und `notes` Tabellen hinzu
- Farben werden als Hex-String (`#RRGGBB`) oder `NULL` gespeichert

**API-Endpunkte**:
- `PUT /api/folders/{id}/color` - Ordner-Farbe setzen/ändern
- `PUT /api/notes/{id}/color` - Notizen-Farbe setzen/ändern

**Frontend-Komponenten**:
- `ColorPickerPopover.svelte`: Popover mit Palette- und Custom-Tabs
- `UnifiedTree.svelte`: Integration des Color Pickers in Sidebar-Items
- `api.ts`: `updateFolderColor()` und `updateNoteColor()` Funktionen

**State Management**: `frontend/src/lib/stores/tree.svelte.ts`
- Farbe wird in `TreeFolder` und `TreeNote` Typen gespeichert
- Automatische UI-Updates bei Farbänderungen

**Styling**:
- Farbbalken: 3px breit, links am Sidebar-Item
- Farben: Named Colors (Design Tokens) oder Custom Hex
- Sanitization: `sanitizeColor()` validiert Hex-Eingaben

### Verfügbare Palette-Farben

| Name | Label | CSS Variable |
|------|-------|--------------|
| `primary` | Primär | `var(--color-primary)` |
| `destructive` | Warnung | `var(--color-destructive)` |
| `accent` | Akzent | `var(--color-accent-foreground)` |
| `muted` | Gedämpft | `var(--color-muted-foreground)` |
| `secondary` | Sekundär | `var(--color-secondary-foreground)` |

**Hinweis**: Named Colors passen sich automatisch an das aktuelle Theme an (hell/dunkel).

### Custom Hex-Farben

- **Format**: `#RGB` oder `#RRGGBB` (case-insensitive)
- **Validierung**: `sanitizeColor()` prüft Hex-Format und normalisiert auf `#RRGGBB`
- **System-Farbwähler**: Native `<input type="color">` für einfache Farbauswahl
- **Live-Vorschau**: Zeigt gewählte Farbe als Text-Beispiel

### Bekannte Einschränkungen

- Color Picker ist nur auf Desktop optimiert (Mobile: Bottom Sheet)
- Keine Farbverlaufs- oder Transparenz-Unterstützung
- Farben werden nicht über E2E-Verschlüsselung synchronisiert (Metadaten bleiben unverschlüsselt)

---

## Auto-Sort Task Lists

### Beschreibung
Task Lists werden automatisch sortiert wenn Checkboxen aktiviert/deaktiviert werden. Erledigte Tasks wandern ans Ende der Liste, während unerledigte Tasks am Anfang bleiben.

### Features

- **Automatisches Sortieren**: Beim Abhaken einer Checkbox wandert der Task automatisch ans Ende der Liste (nach allen unchecked Tasks)
- **Reverse Sorting**: Beim Entfernen des Hakens wandert der Task ans Ende der unchecked Section (vor den checked Tasks)
- **Listen-Grenzen-Erkennung**: Detektiert automatisch Listengrenzen durch Leerzeilen oder Überschriften
- **Atomare Aktion**: Toggle + Verschiebung in einer einzigen Editor-Transaktion
- **Undo-Support**: `Ctrl+Z` macht sowohl Toggle als auch Verschiebung rückgängig
- **Multi-Level-Support**: Funktioniert mit verschachtelten Tasks und gemischten Indent-Levels

### Verwendung

1. **Task abhaken**: Klicke auf die Checkbox eines unerledigten Tasks → Task wandert automatisch nach unten
2. **Task wiederherstellen**: Klicke auf die Checkbox eines erledigten Tasks → Task wandert zurück nach oben
3. **Undo**: `Ctrl+Z` macht beide Änderungen rückgängig (Toggle + Verschiebung)

### Beispiel

**Vorher:**
```markdown
- [ ] Task A
- [ ] Task B
- [ ] Task C
```

**Nach Abhaken von Task A:**
```markdown
- [ ] Task B
- [ ] Task C
- [x] Task A
```

### Technische Details

**Implementierung**: `frontend/src/lib/components/Editor.svelte`

**Kern-Funktionen**:
- `toggleTaskByIndex(checkboxIndex, checked)` - Hauptfunktion für Toggle + Sortierung
- `findTaskListBoundary(doc, taskLineNum)` - Findet Start/Ende der Task-Liste
- `calculateTargetPosition(tasksInList, currentTask, isNowChecked)` - Berechnet Zielposition

**Algorithmus**:
1. Finde alle Tasks in der Markdown-Notiz via Regex
2. Bestimme die Grenzen der aktuellen Task-Liste (getrennt durch Leerzeilen/Überschriften)
3. Sortiere alle Tasks in der Liste: Unchecked zuerst, Checked danach
4. Berechne die Zielposition für den aktuellen Task
5. Falls nötig: Verschiebe die Zeile + Toggle Checkbox in einer atomaren Transaktion

**Liste-Grenzen-Regeln**:
- **Ende**: Leerzeile, Überschrift (z.B. `# Heading`), oder Dokumentende
- **Anfang**: Leerzeile, Überschrift, oder Dokumentanfang
- **Kontinuität**: Alle Zeilen mit Task-Syntax (`- [ ]` oder `- [x]`) gehören zur selben Liste, unabhängig vom Indent-Level

**Task-Erkennung (Regex)**:
```javascript
/^(\s*)- \[([ xX])\] (.+)$/gm
// Matches: "- [ ] Task" oder "  - [x] Task" (mit optionalem Indent)
```

**Atomare Transaktion**:
```javascript
editorView.dispatch({
  changes: [
    // 1. Toggle checkbox
    { from: checkboxStart, to: checkboxEnd, insert: newCheckboxText },
    // 2. Delete original line
    { from: taskLineStart, to: taskLineEnd + 1, insert: '' },
    // 3. Insert at target position
    { from: insertPos, insert: taskLineText + '\n' }
  ]
});
```

### Bekannte Einschränkungen

- Funktioniert nur innerhalb einer zusammenhängenden Task-Liste (Leerzeilen trennen Listen)
- Respektiert keine benutzerdefinierten Sortierkriterien (nur Checked/Unchecked)
- Verschachtelte Tasks werden als flache Liste behandelt (Indent-Level wird beibehalten, aber nicht für Sortierung genutzt)

### Zukünftige Erweiterungen

- **Custom Sorting**: Benutzerdefinierte Sortierkriterien (Priorität, Datum, Alphabetisch)
- **Sub-Task-Hierarchie**: Intelligentes Sortieren von Parent/Child-Tasks
- **Persistierte Reihenfolge**: Option zum Deaktivieren von Auto-Sort

---

## Drag & Drop Task Lists

### Beschreibung
Manuelle Neuordnung von To-Do Items per Drag & Drop innerhalb der Preview-Ansicht. Ermöglicht flexible, benutzerdefinierte Sortierung ohne Abhängigkeit vom Check-Status.

### Features

- **Mouse & Touch Support**: Funktioniert auf Desktop (Maus) und mobilen Geräten (Touch)
- **Visual Drag Handle**: ⠿ Icon zeigt an, wo der Task gegriffen werden kann
  - Desktop: Handle erscheint beim Hover über Task-Items
  - Mobile: Handle ist immer sichtbar
- **Long-Press Protection**: 200ms Delay auf Touch-Geräten verhindert versehentliches Ziehen beim Scrollen
- **Ghost Element**: Semi-transparente Vorschau des gezogenen Tasks während der Bewegung
- **List Isolation**: Drag & Drop funktioniert nur innerhalb derselben Task-Liste (keine Cross-List-Verschiebung)
- **Atomare Aktion**: Verschiebung wird als eine einzige Editor-Transaktion ausgeführt
- **Undo-Support**: `Ctrl+Z` macht die Verschiebung vollständig rückgängig
- **Auto-Save**: Nach jeder Verschiebung wird Auto-Save ausgelöst (falls aktiviert)

### Verwendung

1. **Desktop (Maus)**:
   - Bewege die Maus über einen Task → Drag Handle (⠿) erscheint links
   - Klicke und halte das Handle
   - Ziehe den Task an die gewünschte Position
   - Lasse die Maustaste los zum Ablegen

2. **Mobile (Touch)**:
   - Drag Handle ist immer sichtbar (links neben Checkbox)
   - Drücke und halte das Handle für 200ms
   - Ziehe den Task an die gewünschte Position
   - Loslassen zum Ablegen

3. **Undo**:
   - `Ctrl+Z` macht die letzte Verschiebung rückgängig

### Beispiel

**Vorher:**
```markdown
- [ ] Task A
- [ ] Task B
- [ ] Task C
```

**Nach Drag von Task C zwischen A und B:**
```markdown
- [ ] Task A
- [ ] Task C
- [ ] Task B
```

### Technische Details

**SortableJS Integration**: `frontend/src/lib/editor/task-sortable.ts`
- Svelte Action `use:taskSortable` aktiviert Drag & Drop auf Preview-Container
- Konfiguration:
  - `handle: '.task-drag-handle'` - Nur Handle ist draggable
  - `animation: 150` - Smooth Sortier-Animation
  - `delay: 200`, `delayOnTouchOnly: true` - Long-Press-Protection für Touch
  - `ghostClass: 'sortable-ghost'` - CSS-Klasse für Ghost-Element
  - `filter: '.task-checkbox-wrapper, .task-text'` - Checkboxen und Text nicht draggable

**Task-Identifikation**: `data-task-index` Attribute
- Jedes Task-Item erhält eindeutigen Index (0-basiert)
- Format: `data-task-index="0"`
- Ermöglicht Zuordnung zwischen DOM und Markdown-Quelltext

**Reorder-Algorithmus**: `frontend/src/lib/utils/task-reorder.ts`
```typescript
export function reorderTaskInMarkdown(
  markdown: string,
  oldIndex: number,
  newIndex: number
): string
```
- Findet alle Task-Listen im Markdown via Regex
- Identifiziert betroffene Liste (enthält oldIndex und newIndex)
- Verschiebt Zeile atomisch: Extract → Remove → Insert
- Erhält Whitespace und Formatierung
- Gibt gesamten Markdown mit aktualisierter Liste zurück

**Task-Listen-Grenzen**:
- **Start**: Erste Task-Zeile nach Leerzeile oder Überschrift
- **Ende**: Leerzeile, Überschrift, oder Dokumentende
- **Kontinuität**: Alle Zeilen mit `- [ ]` oder `- [x]` Syntax gehören zur selben Liste

**Atomare Editor-Transaktion**:
```typescript
editorView.dispatch({
  changes: {
    from: 0,
    to: editorView.state.doc.length,
    insert: newMarkdown
  }
});
```

**Auto-Save Integration**:
- `onEnd` Event triggert `editor.scheduleAutoSave()`
- Debounced Save (2s Delay, wie bei normalen Edits)
- Nur wenn Auto-Save aktiviert ist

### CSS-Styling

**Drag Handle**: `.task-drag-handle`
- Icon: `⠿` (U+2847 Braille Pattern Dots-1234)
- Opacity: 0 auf Desktop (hover → 1), 0.3 auf Mobile (immer sichtbar)
- Cursor: `grab` (Desktop), `move` (aktiv)
- Touch-Action: `none` (verhindert Scroll-Interferenz)

**Ghost Element**: `.sortable-ghost`
- Opacity: 0.4 (semi-transparent)
- Background: Leichte Farbe für visuelles Feedback

**Responsive Behavior**:
- Desktop: Handle nur bei Hover sichtbar (cleaner Look)
- Mobile: Handle immer sichtbar (Touch braucht visuellen Hinweis)

### Bekannte Einschränkungen

- **Nur innerhalb derselben Liste**: Drag & Drop zwischen verschiedenen Task-Listen ist nicht möglich
- **Keine verschachtelte Unterstützung**: Sub-Tasks (eingerückt) werden als flache Liste behandelt
- **Keine Cross-Note-Verschiebung**: Funktioniert nur innerhalb einer Notiz
- **Mobile Performance**: Bei sehr langen Listen (>100 Tasks) kann es zu Verzögerungen kommen

### Integration mit Auto-Sort

**Kombination möglich**: Drag & Drop und Auto-Sort können zusammen verwendet werden
- **Manuelle Sortierung**: Drag & Drop für benutzerdefinierte Reihenfolge
- **Auto-Sort beim Toggle**: Checkbox aktivieren → Task wandert automatisch nach unten (überschreibt manuelle Sortierung)

**Empfohlener Workflow**:
1. Verwende Drag & Drop für initiale Priorisierung
2. Erledigte Tasks wandern automatisch ans Ende (Auto-Sort)
3. Neue Reihenfolge manuell anpassen falls gewünscht

### Zukünftige Erweiterungen

- **Cross-List Drag & Drop**: Verschieben zwischen verschiedenen Task-Listen
- **Sub-Task-Hierarchie**: Intelligentes Handling von eingerückten Tasks
- **Drag-Indicator**: Visuelle Linie zeigt Zielposition während des Draggings
- **Keyboard-Support**: Arrow-Keys für Neuordnung (Accessibility)

---

## Interaktive Bildgroessenanpassung

### Beschreibung
Bilder in der Markdown-Preview koennen durch Ziehen an einem Resize-Handle (unten rechts) vergroessert oder verkleinert werden. Die neue Groesse wird automatisch im Markdown-Quelltext gespeichert.

### Features

- **Visual Resize Handle**: Kleines Dreieck unten rechts am Bild (erscheint beim Hover)
- **Drag & Drop**: Ziehen aendert die Bildbreite in Echtzeit
- **Auto-Save**: Neue Groesse wird automatisch im Markdown gespeichert
- **Touch-Support**: Funktioniert auch auf mobilen Geraeten
- **Flexible Einheiten**: Pixel (`width=300`) oder Prozent (`width=50%`)
- **Minimum-Breite**: 50px (verhindert versehentliches Ausblenden)

### Verwendung

1. **Bild einfuegen**:
   - Drag & Drop in den Editor
   - Paste aus Zwischenablage (Ctrl/Cmd+V)
   - Upload-Button in der Toolbar
   - Manuell: `![Alt-Text](url)`

2. **Groesse anpassen**:
   - Wechsle in den Split- oder Preview-Modus
   - Bewege die Maus ueber das Bild → Resize-Handle erscheint
   - Klicke und ziehe am Handle (unten rechts)
   - Loslassen speichert die neue Groesse

3. **Mobile (Touch)**:
   - Touch und halte das Handle
   - Ziehe zum Vergroessern/Verkleinern
   - Loslassen speichert

### Syntax

**Ohne Groessenangabe (Originalgroesse):**
```markdown
![Mein Bild](https://example.com/bild.png)
```

**Mit Pixel-Breite:**
```markdown
![Mein Bild](https://example.com/bild.png){width=300}
```

**Mit Prozent-Breite:**
```markdown
![Mein Bild](https://example.com/bild.png){width=50%}
```

**Mit Alt-Text und Titel:**
```markdown
![Screenshot der App "xelanote"](url){width=400}
```

### Technische Details

**Parser**: `frontend/src/lib/editor/markdown.ts`
- Regex: `!\[([^\]]*)\]\(([^)]+)\)(?:\{width=(\d+%?)\})?`
- Rendert `<img>` mit `style="width: Xpx"` oder `style="width: X%"`
- Wrapper `.image-container` fuer Handle-Positionierung

**Svelte Action**: `frontend/src/lib/editor/image-resize.ts`
- `use:imageResize` Directive auf Preview-Container
- Callback: `onResize(src: string, width: number)`
- Event-Handling: mousedown → mousemove → mouseup
- Touch-Events: touchstart → touchmove → touchend

**Editor-Integration**: `frontend/src/lib/components/Editor.svelte`
```typescript
function handleImageResize(src: string, width: number) {
  // Findet Bild im Markdown und aktualisiert/fuegt {width=...} hinzu
  const newContent = updateImageWidth(content, src, width);
  updateContent(newContent);
  scheduleAutoSave();
}
```

**CSS**: `frontend/src/app.css`
```css
.image-container {
  position: relative;
  display: inline-block;
}

.image-resize-handle {
  position: absolute;
  bottom: 4px;
  right: 4px;
  width: 12px;
  height: 12px;
  cursor: nwse-resize;
  opacity: 0;
  transition: opacity 0.2s;
}

.image-container:hover .image-resize-handle {
  opacity: 0.7;
}
```

### Konfiguration

Das Feature ist standardmaessig aktiviert. Deaktivierung in `frontend/src/lib/config.ts`:

```typescript
export const config = {
  features: {
    imageResize: false  // Deaktiviert Resize-Handles
  }
};
```

### Bekannte Einschraenkungen

- Nur Breite kann angepasst werden (keine Hoehe-Kontrolle)
- Prozent-Werte beziehen sich auf die Container-Breite
- Bei sehr kleinen Bildern kann das Handle schwer zu treffen sein
- Bilder in Code-Bloecken werden nicht processed

### Zukuenftige Erweiterungen

- Aspect-Ratio-Lock (Shift gehalten = proportionales Skalieren)
- Preset-Groessen (klein/mittel/gross)
- Kontextmenue fuer praezise Eingabe
- Hoehen-Kontrolle

---

## Mobile Editor Verbesserungen

### Hanging Indent fuer Aufzaehlungen

**Beschreibung**: Wenn Listentext auf kleinen Bildschirmen umbricht, wird der Folgetext korrekt eingerueckt (alignment mit dem Text nach dem Bullet-Marker, nicht mit dem Bullet selbst).

**Technische Details**: `frontend/src/lib/editor/codemirror.ts`
- Neues `listIndentPlugin` als CodeMirror ViewPlugin
- Nutzt `padding-left` + negativen `text-indent` CSS-Trick
- Erkennt Listenzeilen (unordered `"- "`, ordered `"1. "`, Task `"- [ ] "`) und berechnet die Marker-Breite
- Setzt dynamisch Inline-Styles auf die betroffenen Zeilen-Elemente

**Vorher (ohne Hanging Indent):**
```
- Dies ist ein langer
Listentext der
umbricht
```

**Nachher (mit Hanging Indent):**
```
- Dies ist ein langer
  Listentext der
  umbricht
```

### Dark Mode Farbkorrekturen

**Beschreibung**: Mehrere Syntaxfarben waren im Dark Mode schlecht lesbar. Task-Marker `[ ]`/`[x]` erschienen in einem dunklen Blau (#221199), das auf dunklem Hintergrund nahezu unsichtbar war.

**Korrekturen**: `frontend/src/lib/editor/codemirror.ts`
- **Task-Marker**: `tags.atom` Override auf `var(--color-muted-foreground)` (theme-aware)
- **Bracket-Matching**: Farben fuer passende Klammern sind jetzt theme-aware
- **Links/URLs**: Farbgebung passt sich an das aktuelle Theme an
- Ersetzt hardcoded Farbwerte aus `defaultHighlightStyle` durch CSS-Variablen

### Scrollbare Toolbar auf Mobile

**Beschreibung**: Die Editor-Toolbar-Buttons liefen auf schmalen Bildschirmen (< 768px) ueber den sichtbaren Bereich hinaus. Der Button-Container ist jetzt horizontal scrollbar.

**Technische Details**:
- `frontend/src/lib/components/Editor.svelte`: Layout-Aenderung am Toolbar-Container
- `frontend/src/app.css`: Mobile-spezifische Styles fuer horizontales Scrolling
- Unsichtbare Scrollbar (kein visuelles Scrollbar-Element)
- Desktop-Verhalten bleibt unveraendert (kein Scrolling noetig)

**CSS-Technik**:
```css
/* Mobile: horizontales Scrolling */
@media (max-width: 767px) {
  .toolbar-buttons {
    overflow-x: auto;
    scrollbar-width: none; /* Firefox */
  }
  .toolbar-buttons::-webkit-scrollbar {
    display: none; /* Chrome/Safari */
  }
}
```

---

## Keyboard Shortcuts

| Shortcut | Aktion |
|----------|--------|
| `Ctrl+Space` | Autocomplete manuell triggern (auch für Wikilinks) |
| `Mod+S` | Notiz speichern |
| `Tab` | Einrücken |
| `Shift+Tab` | Ausrücken |

---

## API Endpoints

### Quick Search

**Endpoint**: `GET /api/quick-search`

**Query Parameters**:
- `q` (string): Suchbegriff (optional, leer = zuletzt bearbeitete Notizen)
- `limit` (integer): Max. Anzahl Ergebnisse (default: 10, max: 50)
- `folders` (string): Komma-getrennte Folder-Pfade (optional)
- `tags` (string): Komma-getrennte Tags (optional)
- `created_after`, `created_before` (RFC3339): Datum-Filter (optional)
- `updated_after`, `updated_before` (RFC3339): Datum-Filter (optional)

**Response**:
```json
{
  "notes": [
    {
      "id": "note-001",
      "title": "Meeting Notes 2026",
      "folder_path": "/",
      "version": 1,
      "created_at": "2026-01-23T10:00:00Z",
      "updated_at": "2026-01-23T10:30:00Z"
    }
  ]
}
```

**Authentifizierung**: JWT Bearer Token erforderlich

---

## Konfiguration

### Preview Theme Persistierung

**Storage Key**: `previewTheme`
**Location**: `localStorage`
**Default**: `"default"`

### Focus Mode Persistierung

**Storage Keys**:
- `typewriterMode` (boolean)
- `dimInactiveLines` (boolean)

**Location**: UI-Store (nicht persistiert über Sessions)

---

## Troubleshooting

### Wikilink Autocomplete zeigt keine Ergebnisse

1. **Console öffnen** (F12) und nach `[Wikilink Autocomplete]` Logs suchen
2. **Network Tab** prüfen: Gibt es einen Request zu `/api/quick-search`?
3. **Status Code prüfen**:
   - `401`: JWT Token fehlt oder ungültig → Neu einloggen
   - `200` mit leerem Array: Keine Notizen in DB oder alle gelöscht
4. **Backend Logs prüfen**: Gibt es Fehler beim Query?

### Preview Theme wird nicht angewendet

1. **LocalStorage prüfen**: `localStorage.getItem('previewTheme')`
2. **CSS-Klasse prüfen**: Im Inspector sollte `.preview-theme-{id}` auf `.markdown-preview` sein
3. **Theme-Definition prüfen**: Existiert das Theme in `frontend/src/lib/themes/index.ts`?

### Focus Mode funktioniert nicht

1. **UI-Store prüfen**: `ui.typewriterMode` bzw. `ui.dimInactiveLines` sollten `true` sein
2. **Extension geladen?**: In CodeMirror sollte die Extension aktiv sein
3. **Browser Console**: Gibt es Fehler bei der Extension-Initialisierung?

---

## Zukünftige Erweiterungen

### Wikilink Autocomplete
- Fuzzy Search für besseres Matching
- Preview der verlinkten Notiz beim Hover
- Autocomplete für Aliases: `[[Notiz|Alias]]`
- Backlink-Integration (zeige Anzahl der Backlinks)

### Preview Themes
- Custom Theme Editor
- Theme-Import/Export
- Pro-Theme mit mehr Optionen (Font, Spacing, etc.)

### Focus Mode
- Zenmode: Fullscreen + Dim Lines + Typewriter
- Sentence-by-Sentence Mode
- Paragraph Focus Mode

### Table of Contents
- Collapsible Sections
- Drag & Drop für Heading-Reordering
- Export TOC als Markdown-Liste
