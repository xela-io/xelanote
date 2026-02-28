# Features im Detail

## Feature-Flags

Viele Features sind pro User ein-/ausschaltbar:

| Feature | Default | Beschreibung |
|---------|---------|-------------|
| `journal` | Ein | Tagesbuch-Funktion |
| `graph` | Ein | Wissensgraph |
| `recipes` | Ein | Rezeptverwaltung |
| `shopping` | Ein | Einkaufslisten |
| `canvas` | **Aus** | Infinite Canvas |

API: `GET/PUT /api/features/{feature}`

Gespeichert in `user_features`-Tabelle. Frontend liest Features beim Start und zeigt/versteckt entsprechende UI-Elemente.

---

## Journal

### Konzept

Ein Tagesbuch mit einer Notiz pro Tag. Journal-Einträge sind spezielle Notizen mit `note_type = 'journal'` und einem `journal_date`.

### Ansichten

```
┌──────────────────────────────────────────────────────┐
│  Journal                                              │
│                                                       │
│  ┌─────────────┐  ┌──────────────────────────────┐   │
│  │  Kalender    │  │  Heutiger Eintrag            │   │
│  │              │  │                              │   │
│  │  Mo Di Mi .. │  │  Markdown-Editor             │   │
│  │   ● ●  ●    │  │  (gleicher wie Notiz-Editor)  │   │
│  │     ●       │  │                              │   │
│  │  ●    ● ●   │  │                              │   │
│  └─────────────┘  └──────────────────────────────┘   │
│                                                       │
│  ┌─────────────────────────────────────────────────┐  │
│  │  Jahres-Heatmap (GitHub-Style)                  │  │
│  │  ░░▓▓░░▓░░▓▓▓░░░▓▓░░▓░░▓░░░▓▓▓░░▓░░           │  │
│  └─────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────┘
```

### API

```
GET  /api/journal/                     # Heutigen Eintrag laden
GET  /api/journal/entries              # Alle Einträge (paginiert)
GET  /api/journal/lookup?date=2024-03-15  # Eintrag für Datum
GET  /api/journal/calendar?year=2024&month=3  # Tage mit Einträgen
GET  /api/journal/calendar/year?year=2024     # Jahres-Heatmap
```

### Besonderheiten

- **Unique Constraint:** Maximal ein Eintrag pro Tag pro User
- **Auto-Delete:** Leere Journal-Einträge werden beim Weg-Navigieren gelöscht
- **Kalender:** Punkte zeigen Tage mit Einträgen an
- **Heatmap:** GitHub-Style Jahresansicht mit Intensität basierend auf Wortanzahl

---

## Rezepte

### Datenmodell

Ein Rezept besteht aus:

```
Rezept-Notiz (note_type = 'recipe')
    ├── Markdown-Inhalt (Anleitung)
    ├── recipe_metadata
    │   ├── servings (Portionen)
    │   ├── prep_time (Vorbereitungszeit in Min.)
    │   ├── cook_time (Kochzeit in Min.)
    │   ├── difficulty (easy/medium/hard)
    │   └── source_url (Original-Quelle)
    ├── recipe_ingredients[]
    │   ├── name, amount, unit
    │   ├── group_name ("Sauce", "Teig")
    │   └── display_order
    └── recipe_images[]
        ├── filename (signed URL)
        └── display_order
```

### Rezept-Editor

```
┌──────────────────────────────────────────────────────┐
│  Pasta Carbonara                              [📷+]  │
│                                                       │
│  ┌────────────┐  ┌───────────────────────────────┐   │
│  │ Metadaten   │  │  Zutaten                      │   │
│  │ 4 Portionen │  │  ┌─────────────────────────┐  │   │
│  │ 15min Vorb. │  │  │ 400g  Spaghetti         │  │   │
│  │ 20min Koch  │  │  │ 200g  Guanciale         │  │   │
│  │ Mittel      │  │  │ 4     Eigelb            │  │   │
│  └────────────┘  │  │ 100g  Pecorino          │  │   │
│                   │  └─────────────────────────┘  │   │
│  ┌───────────────────────────────────────────────┐   │
│  │  Anleitung (Markdown)                         │   │
│  │                                                │   │
│  │  1. Guanciale in Streifen schneiden...        │   │
│  │  2. Eigelb mit Pecorino verrühren...          │   │
│  └───────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────┘
```

### Zutatenskalierung

```
Original: 4 Portionen → 400g Spaghetti
User will: 2 Portionen → Server berechnet: 200g Spaghetti
```

### Rezept-Collections (Kochbücher)

Rezepte können in Collections organisiert werden (viele-zu-viele):

```
Collection "Italienisch"
  ├── Pasta Carbonara
  ├── Pizza Margherita
  └── Tiramisu

Collection "Schnelle Gerichte"
  ├── Pasta Carbonara  (kann in mehreren Collections sein)
  └── Rührei
```

Collections können mit anderen Usern geteilt werden (3-stufige Berechtigungskette: Collection → Rezept → User).

### KI-Features für Rezepte

Siehe [KI-Integration > Rezept-KI](KI-Integration.md#Rezept-KI)

---

## Einkaufslisten

### Datenmodell

```
Shopping List "Wocheneinkauf"
  ├── Item: 500g Spaghetti (Kategorie: Nudeln)
  │   └── Sub-Item: Alternativ: Penne
  ├── Item: 3 Tomaten (Kategorie: Obst & Gemüse)
  ├── Item: 200g Parmesan (Kategorie: Milchprodukte)
  │   └── source_recipe: "Pasta Carbonara" (von Rezept importiert)
  └── Item: Basilikum (Kategorie: Kräuter)
```

### Features

| Feature | Beschreibung |
|---------|-------------|
| **Kategorien** | Items nach Supermarkt-Abteilung gruppiert |
| **Abhaken** | Checkboxen für erledigte Items |
| **Drag & Drop** | Reihenfolge anpassen |
| **Favoriten** | Häufig gekaufte Items merken |
| **Rezept-Import** | Zutaten aus Rezept in Liste übernehmen |
| **KI-Sortierung** | LLM sortiert nach Supermarkt-Layout |
| **Teilen** | Liste mit anderen Usern teilen |
| **Archivieren** | Erledigte Listen archivieren |

### Favoriten-System

```
User kauft oft "Milch 1L":
  → Nach 3x hinzufügen erscheint es als Favorit
  → usage_count wird hochgezählt
  → Favoriten werden nach Häufigkeit sortiert
  → Schnelles Hinzufügen mit einem Tap
```

### Teilen von Einkaufslisten

```
POST /api/shopping/lists/{id}/shares
Body: {username: "partner", role: "editor"}
    ↓
Partner sieht die Liste in seiner App
    ↓
Echtzeit-Sync via WebSocket:
  Partner hakt "Milch" ab → sofort bei dir sichtbar
```

---

## Canvas

### Konzept

Infinite Canvas — ein endloses Whiteboard für visuelle Notizen. Kompatibel mit dem [JSON Canvas](https://jsoncanvas.org/) Format (auch von Obsidian verwendet).

### Node-Typen

```
┌──────────────────────────────────────────────┐
│  Canvas: "Projekt-Planung"                   │
│                                               │
│  ┌──────────┐     ┌──────────┐               │
│  │ Textcard │────→│ Textcard │               │
│  │ "Idee A" │     │ "Idee B" │               │
│  └──────────┘     └──────────┘               │
│       │                                       │
│       ↓                                       │
│  ┌──────────────┐    ┌─────────────────┐     │
│  │ Embedded Note│    │ Link            │     │
│  │ [[Details]]  │    │ https://...     │     │
│  └──────────────┘    └─────────────────┘     │
│                                               │
│  ┌─ Group "Phase 1" ─────────────────────┐   │
│  │  ┌────────┐  ┌────────┐               │   │
│  │  │ Card 1 │  │ Card 2 │               │   │
│  │  └────────┘  └────────┘               │   │
│  └───────────────────────────────────────┘   │
└──────────────────────────────────────────────┘
```

**Node-Typen:**
- **Text Cards:** Freitext-Karten
- **Embedded Notes:** Verweis auf eine existierende Notiz
- **Links:** URL-Karten
- **Groups:** Container für andere Nodes

**Standardmäßig deaktiviert** — muss in Feature-Flags aktiviert werden.

---

## Suche

### Volltextsuche

```
GET /api/search?q=architektur
    ↓
FTS5-Query:
  SELECT *, snippet(notes_fts, ...) FROM notes_fts
  WHERE notes_fts MATCH 'architektur'
  AND user_id = ?
    ↓
Response: [{
  id: "...",
  title: "Backend-Architektur",
  snippet: "Die <mark>Architektur</mark> basiert auf..."
}]
```

### Quick-Switcher (Ctrl+P)

```
Ctrl+P → Modal öffnet sich
    ↓
Tippen: "arc"
    ↓
GET /api/quick-search?q=arc
    ↓
Leichtgewichtige Titel-Suche (kein FTS5)
    ↓
Dropdown: "Architektur-Überblick", "Backend-Architektur"
    ↓
Enter → Notiz öffnen
```

### Verschlüsselte Notizen

Für verschlüsselte Notizen: Client-seitige Fuse.js-Suche über entschlüsselte Inhalte.

---

## Teilen (Sharing)

### Was kann geteilt werden?

| Objekt | Rollen |
|--------|--------|
| Einzelne Notiz | Viewer, Editor |
| Ordner | Viewer, Editor |
| Rezept-Collection | Viewer, Editor |
| Einkaufsliste | Viewer, Editor |

### Berechtigungs-Modell

```
Viewer:  Lesen
Editor:  Lesen + Schreiben
Owner:   Alles (Teilen, Löschen, Berechtigungen ändern)
```

### Echtzeit-Kollaboration

Über WebSocket werden Änderungen an geteilten Notizen in Echtzeit gepusht:

```
User A bearbeitet geteilte Notiz
    ↓
Server: Broadcast an alle User mit Zugriff
    ↓
User B sieht Änderungen (nach Reload / Auto-Refresh)
```

---

## Papierkorb

```
DELETE /api/notes/{id}
    ↓
Soft-Delete: is_deleted = true, deleted_at = NOW()
    ↓
Notiz erscheint im Papierkorb (/trash)
    ↓
Wiederherstellen: POST /api/notes/{id}/restore
Endgültig löschen: DELETE /api/notes/{id}/permanent
Alles leeren: DELETE /api/trash
```

---

## Admin-Panel

Nur für Admin-User (erster registrierter User ist automatisch Admin):

```
/admin
  ├── System-Stats (User, Notizen, Ordner, Tags, Speicher)
  ├── Detaillierte Stats mit Wachstums-Charts
  ├── User-Verwaltung
  │   ├── User-Liste (Notiz-Anzahl, Speicherverbrauch, 2FA-Status)
  │   ├── Admin-Status toggl
  │   ├── Storage-Limit pro User setzen
  │   └── User löschen
  ├── Aktivitätsprotokoll
  └── System-Einstellungen
      └── Registrierung aktivieren/deaktivieren
```

---

## PWA & Desktop

### PWA (Progressive Web App)

- Vite PWA Plugin mit Workbox (Service Worker)
- Installierbar auf Mobilgeräten
- Web Share Target API: URLs/Text an xelanote teilen, um Notizen zu erstellen
- iOS-spezifisch: Standalone-Erkennung, Safe-Area-Insets, Keyboard-Detection

### Tauri (Desktop)

- Native Desktop-App für Windows/macOS/Linux
- OS Keyring für Token-Speicherung (statt Cookies)
- Datei-System-Zugriff für Exports

## Nächste Seiten

- [Notizen-und-Wikilinks](Notizen-und-Wikilinks.md) — Editor und Links im Detail
- [API-Referenz](API-Referenz.md) — Alle REST-Endpunkte
- [Entwicklung-Setup](Entwicklung-Setup.md) — Lokale Entwicklungsumgebung
