# Notizen und Wikilinks

## Notiz-Typen

Jede Notiz hat einen `note_type`:

| Typ | Beschreibung | Editor |
|-----|-------------|--------|
| `note` | Standard-Markdown-Notiz | CodeMirror 6 |
| `journal` | Tagesbuch-Eintrag (1 pro Tag) | CodeMirror 6 |
| `recipe` | Rezept mit Metadaten/Zutaten | RecipeEditor |
| `canvas` | Infinite Canvas (Whiteboard) | CanvasEditor |

## Markdown-Editor

Der Editor basiert auf **CodeMirror 6** und unterstützt:

### Formatierung
- Standard-Markdown (Headings, Bold, Italic, Strikethrough, Code)
- Tabellen (mit Insert-Dialog)
- Checklisten / Task-Listen (`- [ ]` Syntax)
- Code-Blöcke mit Syntax-Highlighting
- Inline-Bilder (Upload per Drag & Drop oder Paste)

### Ansichtsmodi
```
┌──────────────────────────────┐
│  Edit  │  Preview  │  Split  │
└──────────────────────────────┘

Edit:    Nur Markdown-Quelltext
Preview: Nur gerendertes HTML
Split:   Links Markdown, rechts Preview
```

### Besondere Features
- **Inline-Titel:** Der Notiz-Titel wird oben im Editor angezeigt und kann dort bearbeitet werden
- **Focus-Mode:** Blendet Sidebar, TabBar und Toolbar aus
- **Find & Replace:** `Ctrl+F` öffnet Suchleiste im Editor
- **Auto-Save:** Änderungen werden nach kurzem Debounce automatisch gespeichert

## Wikilinks

### Syntax

```markdown
Siehe [[Meine andere Notiz]] für Details.
Auch [[Notiz in Unterordner]] funktioniert.
```

### Wie werden Links aufgelöst?

```
User tippt [[Meine Notiz]]
    ↓
Frontend: Auto-Complete Dropdown zeigt passende Notiz-Titel
    ↓
User speichert Notiz
    ↓
Backend: NoteService.UpdateNote()
    ↓
Parser extrahiert alle [[...]] aus dem Markdown
    ↓
Für jeden Link-Titel:
    SELECT id FROM notes
    WHERE title = 'Meine Notiz'
    AND user_id = ?
    ↓
    Gefunden? → INSERT INTO links (source, target)
    Nicht gefunden? → In unresolved_link_refs speichern
```

### Backlinks

Jede Notiz zeigt ihre Backlinks — also alle Notizen, die auf sie verlinken:

```
GET /api/notes/{id}/backlinks
    ↓
SELECT source_note_id FROM links
WHERE target_note_id = ?
    ↓
[{id: "abc", title: "Meeting-Protokoll"}, ...]
```

### Umbenennung und Link-Updates

Wenn eine Notiz umbenannt wird, müssen alle Links aktualisiert werden:

```
POST /api/notes/{id}/rename   (Body: {title: "Neuer Titel"})
    ↓
Async-Job wird erstellt (weil es langsam sein kann)
    ↓
Worker durchsucht ALLE Notizen des Users:
  - Finde alle mit [[Alter Titel]]
  - Ersetze durch [[Neuer Titel]]
  - Speichere jede betroffene Notiz
    ↓
Frontend pollt Job-Status: GET /api/jobs/{id}
    → "running" → Spinner zeigen
    → "completed" → Notiz-Liste neu laden
```

### Verschlüsselte Notizen und Links

Bei verschlüsselten Notizen kennt der Server den Inhalt nicht, daher:

1. Client entschlüsselt die Notiz lokal
2. Client extrahiert `[[...]]`-Links
3. Client sendet Link-Titel an Server: `POST /api/notes/{id}/resolve-links`
4. Server löst Titel → IDs auf
5. Server speichert in `links`-Tabelle

## Wissensgraph

### Datenquelle

```
GET /api/graph
    ↓
GraphService.GetGraph(userID):
  1. Alle Notizen laden (ID, Titel, Ordner, Link-Count)
  2. Alle Links laden (Source → Target)
  3. Nodes + Edges als JSON zurückgeben
  4. Im Cache halten (5min TTL)
```

### Visualisierung

Das Frontend nutzt die `force-graph` Bibliothek (D3-basierte Kraft-Simulation):

```
Nodes = Notizen (Kreise)
  - Größe proportional zur Anzahl der Links
  - Farbe nach Ordner

Edges = Links (Linien zwischen Kreisen)
  - [[Source]] → [[Target]]

Interaktion:
  - Drag: Nodes verschieben
  - Click: Notiz öffnen
  - Zoom: Rein-/Rauszoomen
  - Filter: Nach Ordner filtern
```

### Performance-Limit

Um den Browser nicht zu überlasten, gibt es ein `MaxGraphNodes`-Limit. Bei sehr vielen Notizen wird der Graph gefiltert.

## Ordner-System

```
/                          # Wurzel (unfiled Notizen)
/Projekte/                 # Ordner
/Projekte/Web/             # Unter-Ordner
/Projekte/Web/React/       # Beliebig tief verschachtelbar
```

Ordner-Features:
- **Reihenfolge:** `display_order` für manuelle Sortierung (Drag & Drop)
- **Farbe:** Individuelle Ordnerfarbe
- **KI-Default:** `ai_enabled`-Flag wird an neue Notizen vererbt
- **Verschlüsselungs-Default:** `encryption_default` wird an neue Notizen vererbt
- **Teilen:** Ganzen Ordner mit anderen Usern teilen

## Tags

```markdown
Notiz hat Tags: #wichtig #projekt-alpha
```

Tags werden in einer separaten `tags`-Tabelle gespeichert (M:N über `note_tags`). KI kann Tags basierend auf Notiz-Inhalt vorschlagen.

## Due-Dates

In Markdown können Deadlines gesetzt werden:

```markdown
- [ ] Report fertigstellen @due(2024-03-15)
- [x] Meeting vorbereiten @due(2024-03-10)  ✓ erledigt
```

Der Server-Parser extrahiert alle `@due(YYYY-MM-DD)` und speichert sie in `note_due_dates`. Die `/due-dates`-Seite zeigt alle fälligen/überfälligen Aufgaben.

## Versionshistorie

```
GET /api/notes/{id}/versions
    ↓
Paginierte Liste aller Snapshots:
  [{version: 5, title: "...", created_at: "..."}, ...]
    ↓
Vergleich: GET /api/notes/{id}/versions/compare?from=3&to=5
    ↓
Diff zwischen zwei Versionen (Zeile für Zeile)
    ↓
Wiederherstellen: POST /api/notes/{id}/versions/3/restore
    ↓
KI-Delta-Summary: POST /api/notes/{id}/versions/delta-summary
  → "In Version 5 wurde der Abschnitt 'Architektur' erweitert und ..."
```

## Nächste Seiten

- [Features-im-Detail](Features-im-Detail.md) — Journal, Rezepte, Einkaufslisten
- [Verschlüsselung](Verschlüsselung.md) — Wie verschlüsselte Notizen funktionieren
