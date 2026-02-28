# KI-Integration

## BYOK — Bring Your Own Key

xelanote hat **keine eigenen KI-Kosten**. Jeder User bringt seine eigenen API-Keys mit:

- **Claude** (Anthropic)
- **Gemini** (Google)
- **ChatGPT** (OpenAI)

Die Keys werden **AES-256-GCM verschlüsselt** in der Datenbank gespeichert.

## Architektur

```
frontend/src/lib/api/ai.ts          # API-Client für KI-Endpoints
    ↓
backend/internal/api/*_handler.go   # HTTP-Handler
    ↓
backend/internal/service/
  ├── summarize_service.go          # Zusammenfassungen, Tags, Links, Transform
  └── recipe_suggestion_service.go  # Rezept-KI
    ↓
backend/internal/llm/
  ├── provider.go                   # Provider-Interface
  ├── provider_router.go            # Provider-Routing + Client-Cache
  ├── claude.go                     # Anthropic Claude
  ├── gemini.go                     # Google Gemini
  └── chatgpt.go                    # OpenAI ChatGPT
```

### Provider-Router

```go
type ProviderRouter struct {
    clients map[string]*Client  // Pro User ein cached Client
    mu      sync.RWMutex
}
```

Der Router wählt den Provider basierend auf User-Einstellung:
1. Bevorzugten Provider des Users prüfen
2. Fallback-Reihenfolge: Claude → Gemini → ChatGPT
3. Client wird pro User gecacht (Mutex-geschützt)

### Provider-Interface

```go
type Provider interface {
    Complete(ctx context.Context, prompt string) (string, error)
    Stream(ctx context.Context, prompt string) (<-chan string, error)
}

type VisionProvider interface {
    Provider
    CompleteWithImage(ctx context.Context, prompt string, image []byte) (string, error)
}
```

Claude und Gemini implementieren auch `VisionProvider` (für Bild-Verarbeitung).

## KI-Features

### 1. Zusammenfassung

```
POST /api/notes/{id}/summarize
    ↓
Notiz-Inhalt laden
    ↓
LLM-Prompt: "Fasse folgende Notiz zusammen: ..."
    ↓
Ergebnis in note.summary speichern
    ↓
Content-Hash speichern (für Cache-Invalidierung)
```

**SSE-Streaming:**
```
POST /api/notes/{id}/summarize (Accept: text/event-stream)
    ↓
Streaming via Server-Sent Events:
  data: {"chunk": "Die Notiz behandelt..."}
  data: {"chunk": " das Thema..."}
  data: {"done": true}
```

### 2. Tag-Vorschläge

```
POST /api/notes/{id}/suggest-tags
    ↓
LLM-Prompt: "Schlage 5 passende Tags vor für: ..."
    ↓
Response: ["architektur", "backend", "go", "api", "rest"]
    ↓
User kann Tags annehmen oder ablehnen
```

### 3. Link-Vorschläge

```
POST /api/notes/{id}/suggest-links
    ↓
1. Aktuelle Notiz laden
2. Alle Notiz-Titel des Users laden (/api/notes/titles/ai-enabled)
3. LLM-Prompt: "Welche dieser Notizen sind thematisch verwandt?"
    ↓
Response: [{title: "Go Patterns", reason: "Ähnliches Backend-Thema"}, ...]
    ↓
User kann Links mit einem Klick einfügen
```

### 4. Rechtschreibprüfung

```
POST /api/llm/spell-check
    ↓
LLM prüft Text auf Fehler
    ↓
Response: [{original: "Arhcitektur", corrected: "Architektur", position: 42}]
```

### 5. AI Transform

```
POST /api/notes/{id}/ai-transform
Body: {action: "expand" | "condense" | "rewrite" | "formalize" | "simplify"}
    ↓
LLM transformiert den markierten Text
    ↓
User sieht Vorschau und kann annehmen/ablehnen
```

### 6. Markdown-Formatierung

```
POST /api/notes/{id}/format-markdown
    ↓
LLM korrigiert und formatiert den Markdown-Text
```

### 7. Versions-Vergleich

```
POST /api/notes/{id}/versions/delta-summary
Body: {from_version: 3, to_version: 5}
    ↓
LLM: "Was hat sich zwischen Version 3 und 5 geändert?"
    ↓
"Der Abschnitt 'Architektur' wurde um Details zum..."
```

## Rezept-KI

### Rezept von URL importieren

```
POST /api/recipes/import-from-url
Body: {url: "https://chefkoch.de/rezept/..."}
    ↓
1. URL fetchen, HTML parsen
2. LLM: "Extrahiere Titel, Zutaten, Anleitung aus diesem HTML"
    ↓
Rezept mit Metadaten + strukturierten Zutaten erstellen
```

### Rezept von Foto importieren

```
POST /api/recipes/import-from-image
Body: {image: <base64>}
    ↓
VisionProvider.CompleteWithImage():
  "Erkenne das Rezept auf diesem Foto"
    ↓
Strukturiertes Rezept erstellen
```

### Ähnliche Rezepte vorschlagen

```
POST /api/recipes/suggestions/similar
Body: {recipe_id: "..."}
    ↓
LLM analysiert Zutaten + Kategorie
    ↓
Vorschläge für ähnliche Gerichte
```

### Rezepte nach Zutaten

```
POST /api/recipes/suggestions/by-ingredients
Body: {ingredients: ["Tomate", "Mozzarella", "Basilikum"]}
    ↓
LLM: "Was kann man daraus kochen?"
    ↓
Rezeptvorschläge
```

## Einkaufslisten-KI

### Intelligente Sortierung

```
POST /api/shopping/lists/{id}/sort
    ↓
Alle Items der Liste laden
    ↓
LLM-Prompt: "Sortiere diese Einkaufsliste nach
             typischer Supermarkt-Anordnung:
             Obst/Gemüse → Brot → Milch → ..."
    ↓
Items bekommen neue category + category_order
    ↓
Antwort: Umsortierte Liste
```

## Audio-Transkription (Whisper)

```
POST /api/llm/transcribe
Body: FormData mit Audio-Datei
Timeout: 120 Sekunden (erhöht wegen langer Audiodateien)
    ↓
OpenAI Whisper API
    ↓
Transkribierter Text
    ↓
User kann Text in Notiz einfügen
```

## Per-Notiz KI-Toggle

Jede Notiz hat ein `ai_enabled`-Flag:

- **Aktiviert:** Alle KI-Features verfügbar
- **Deaktiviert:** Keine KI-Anfragen für diese Notiz
- **Ordner-Default:** Neue Notizen erben `ai_enabled` vom Ordner

Das gibt dem User volle Kontrolle darüber, welche Inhalte an LLM-Provider gesendet werden.

## Modell-Einstellungen

User können pro Provider ein bevorzugtes Modell wählen (z.B. Claude Sonnet statt Opus). Gespeichert in `ai_model_preferences`.

## Nächste Seiten

- [Verschlüsselung](Verschlüsselung.md) — Wie KI mit verschlüsselten Notizen zusammenspielt
- [Features-im-Detail](Features-im-Detail.md) — Weitere App-Features
