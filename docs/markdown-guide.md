# Markdown Formatting Guide für xelanote

xelanote verwendet **markdown-it** für Markdown-Rendering mit zusätzlicher Unterstützung für Wikilinks.

## Grundlegende Formatierung

### Überschriften

```markdown
# Überschrift 1
## Überschrift 2
### Überschrift 3
```

**Ergebnis:**
- H1: Große Überschrift mit Unterstrich (2rem)
- H2: Mittlere Überschrift (1.5rem)
- H3: Kleinere Überschrift (1.25rem)

### Text-Formatierung

```markdown
**fett** oder __fett__
*kursiv* oder _kursiv_
***fett und kursiv***
~~durchgestrichen~~
```

### Listen

**Ungeordnete Liste:**
```markdown
- Punkt 1
- Punkt 2
  - Unterpunkt 2.1
  - Unterpunkt 2.2
- Punkt 3
```

**Geordnete Liste:**
```markdown
1. Erster Punkt
2. Zweiter Punkt
3. Dritter Punkt
```

### Links

**Standard-Links:**
```markdown
[Link-Text](https://example.com)
```

**Auto-Linking:**
```markdown
https://example.com wird automatisch zu einem Link
```

### Code

**Inline-Code:**
```markdown
Das ist `inline code` im Text.
```

**Code-Block:**
````markdown
```
const x = 10;
console.log(x);
```
````

**Code-Block mit Sprache:**
````markdown
```javascript
const x = 10;
console.log(x);
```
````

### Zitate

```markdown
> Dies ist ein Zitat.
> Es kann mehrere Zeilen haben.
```

### Bilder

```markdown
![Alternativer Text](https://example.com/bild.png)
![Bild aus Upload](/api/uploads/filename.png)
```

**Bild hochladen:**
- Drag & Drop in den Editor
- Paste aus Zwischenablage (Ctrl/Cmd+V)
- Upload-Button in der Toolbar

**Bildgroesse anpassen:**

xelanote unterstuetzt eine erweiterte Syntax fuer Bildgroessen:

```markdown
# Originalgroesse
![Bild](url)

# Feste Breite in Pixel
![Bild](url){width=300}

# Relative Breite in Prozent
![Bild](url){width=50%}
```

**Interaktive Groessenanpassung:**
- Im Preview-Modus erscheint beim Hover ein Resize-Handle (unten rechts)
- Ziehe am Handle um die Bildgroesse visuell anzupassen
- Die neue Groesse wird automatisch im Markdown gespeichert
- Minimum-Breite: 50px
- Funktioniert auch auf Touch-Geraeten

## Wikilinks (Spezial-Feature)

Wikilinks sind eine Besonderheit von xelanote für interne Verknüpfungen zwischen Notizen.

### Einfacher Wikilink

```markdown
[[Notiz-Titel]]
```

- Erstellt einen Link zur Notiz mit dem angegebenen Titel
- Grüne/blaue Farbe = Notiz existiert
- Rote gestrichelte Linie = Notiz existiert noch nicht

### Wikilink mit Alias

```markdown
[[Notiz-Titel|Angezeigter Text]]
```

Zeigt "Angezeigter Text" an, verlinkt aber auf "Notiz-Titel".

### Wikilink-Verhalten

- **Existierende Notiz**: Klick öffnet die Notiz
- **Nicht-existierende Notiz**: Klick fragt, ob neue Notiz erstellt werden soll
- **Automatische Erkennung**: Wikilinks werden im Editor farbig markiert
- **Backlinks**: Alle Notizen, die auf die aktuelle Notiz verweisen, werden unten angezeigt

## Farbige Text-Hervorhebungen

Mit farbigen Text-Hervorhebungen kannst du wichtige Textstellen visuell hervorheben.

### Syntax

```markdown
{color:FARBE}Text{/color}
```

**Theme-Farben:**
- `{color:primary}`, `{color:destructive}`, `{color:accent}`, `{color:muted}`, `{color:secondary}`
- Hex: `{color:#ff6600}`, RGB: `{color:rgb(0, 128, 255)}`

**Features:**
- ColorPicker UI über den Palette-Button in der Toolbar
- Keyboard-Shortcut: `Ctrl/Cmd + Shift + C`
- Visuelle Tag-Markierung im Editor

## Typografische Verbesserungen

xelanote aktiviert **typographer** für automatische Ersetzungen:

- `"Zitat"` → typografische Anführungszeichen
- `--` → En-Dash (–)
- `---` → Em-Dash (—)
- `...` → Ellipse (…)
- `(c)` → Copyright (©)
- `(tm)` → Trademark (™)

## Task-Listen / Checklisten

```markdown
- [ ] Unerledigte Aufgabe
- [x] Erledigte Aufgabe
- [ ] Noch eine Aufgabe
```

Im Preview-Modus kannst du Checkboxen direkt anklicken - der Markdown-Quelltext wird automatisch aktualisiert.

**Features:**
- **Interaktive Checkboxen**: Klicken zum Abhaken/Wiederherstellen
- **Auto-Sort**: Erledigte Tasks wandern automatisch ans Ende der Liste
- **Drag & Drop**: Manuelle Neuordnung per Maus oder Touch
  - Desktop: Ziehe am ⠿ Handle (erscheint beim Hover)
  - Mobile: ⠿ Handle ist immer sichtbar, Long-Press (200ms) zum Ziehen
  - Nur innerhalb derselben Liste möglich
- **Undo-Support**: `Ctrl+Z` macht Toggle und Verschiebung rückgängig
- **Automatische Synchronisation**: Änderungen im Preview werden sofort im Editor übernommen
- **Visuelle Darstellung**: Erledigte Aufgaben werden durchgestrichen angezeigt

## Was NICHT unterstützt wird

- ❌ HTML-Tags (werden escaped für Sicherheit)
- ❌ Markierung `==...==` (kein Plugin konfiguriert)
- ❌ Fußnoten (kein Plugin konfiguriert)
- ❌ Mathe-Formeln (kein Plugin konfiguriert)

## Editor-Shortcuts

- **Ctrl/Cmd + S**: Notiz manuell speichern
- **Auto-Save**: Speichert automatisch nach 2 Sekunden Inaktivität (wenn aktiviert)

## Split-View Modi

xelanote bietet drei Editor-Modi:

1. **Nur Editor** (Edit): Markdown-Quelltext bearbeiten
2. **Split** (Editor + Preview): Gleichzeitig schreiben und Vorschau sehen
3. **Nur Preview**: Gerenderte Markdown-Ansicht

Modi umschalten über die Buttons in der Toolbar.

## Tipps

- Verwende **Wikilinks** statt Standard-Links für interne Verknüpfungen
- **Backlinks** zeigen automatisch, welche Notizen auf die aktuelle verweisen
- Nutze **Split-View**, um das Ergebnis während des Schreibens zu sehen
- **Auto-Save** aktivieren, um nie Änderungen zu verlieren
- Bilder per **Drag & Drop** hochladen statt manuell einzufügen

## Beispiel-Notiz

```markdown
# Projekt-Planung

## Ziele
- **Sprint 1**: [[Backend API]] implementieren
- *Sprint 2*: [[Frontend Components]] erstellen
- Sprint 3: [[Testing Strategy|Tests]] schreiben

## Status
**WICHTIG**: Deadline am 31. Januar!

## Notizen
Siehe auch [[Team Meeting 2026-01-18]] für Details.

**Wichtig**: Code muss `production-ready` sein!

> "Premature optimization is the root of all evil"
> - Donald Knuth

```javascript
function hello() {
  console.log("Hello, xelanote!");
}
```

## Links
- Offizielle Docs: https://docs.example.com
- GitHub Repo: [[GitHub Repository]]
```

---

*Stand: 2026-01-30*
