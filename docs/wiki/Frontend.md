# Frontend (SvelteKit)

## Überblick

Das Frontend ist eine SvelteKit Single-Page-Application (SPA) mit Svelte 5 und Tailwind v4. Es läuft im Browser oder als Desktop-App via Tauri.

## Projektstruktur

```
frontend/src/
├── routes/                    # Seiten (file-based routing)
│   ├── +layout.svelte         # Root-Layout (Auth, Sidebar, WebSocket)
│   ├── +page.svelte           # Home-Dashboard
│   ├── login/                 # Login-Seite
│   ├── register/              # Registrierung
│   ├── note/[id]/             # Notiz-Editor
│   ├── journal/               # Journal-Ansicht
│   ├── recipes/               # Rezepte
│   ├── shopping/              # Einkaufslisten
│   ├── graph/                 # Wissensgraph
│   ├── search/                # Suche
│   ├── due-dates/             # Fällige Aufgaben
│   ├── settings/              # Einstellungen
│   ├── admin/                 # Admin-Panel
│   ├── shared/                # Geteilte Notizen
│   └── trash/                 # Papierkorb
├── lib/
│   ├── api/                   # API-Client (ein Modul pro Feature)
│   ├── components/            # Wiederverwendbare UI-Komponenten
│   ├── editor/                # Editor-Utilities (CodeMirror)
│   ├── stores/                # Svelte 5 Runes (State Management)
│   ├── types/                 # TypeScript-Typen
│   ├── utils/                 # Hilfsfunktionen
│   ├── crypto/                # Client-seitige Verschlüsselung
│   ├── offline/               # Offline-Modus (IndexedDB)
│   ├── themes/                # Gruvbox Light/Dark
│   └── locales/               # i18n (de/en)
```

## Root-Layout (`+layout.svelte`)

Das Root-Layout ist das Herzstück der App. Es initialisiert beim Laden:

```
App startet
    ↓
1. Auth prüfen (Cookie-basiert)
   → Kein Token? → Redirect zu /login
   → Token abgelaufen? → Refresh versuchen
    ↓
2. Sidebar initialisieren (Notiz-Baum laden)
    ↓
3. WebSocket verbinden (für Echtzeit-Updates)
    ↓
4. Verschlüsselung initialisieren
   → KEK aus IndexedDB laden (falls vorhanden)
    ↓
5. Offline-Sync starten
   → Queued Operations abspielen
    ↓
6. PWA Service Worker registrieren
    ↓
7. Feature-Flags laden (Journal, Graph, Canvas, etc.)
```

### Layout-Komponenten

```
┌──────────────────────────────────────────────────┐
│  TabBar (wenn mehrere Notizen offen)             │
├──────────┬───────────────────────────────────────┤
│          │                                       │
│ Sidebar  │         Hauptinhalt                   │
│          │         (Route-Slot)                   │
│ - Ordner │                                       │
│ - Notizen│                                       │
│ - Quick  │                                       │
│   Actions│                                       │
│          │                                       │
├──────────┴───────────────────────────────────────┤
│  MobileBottomNav (nur auf Mobilgeräten)          │
└──────────────────────────────────────────────────┘
```

## State Management (Svelte 5 Runes)

**Wichtig:** xelanote verwendet ausschließlich **Svelte 5 Runes** — keine Svelte 4 Stores (`writable`, `readable`, etc.).

### Wie Runes funktionieren

Jedes State-Modul ist eine `.svelte.ts`-Datei, die `$state` und `$derived` verwendet:

```typescript
// stores/notes.svelte.ts

// Reaktiver State (wie ein Store, aber nativ in Svelte 5)
let notes = $state(new SvelteMap<string, Note>());
let currentNoteId = $state<string | null>(null);
let isDirty = $state(false);
let isSaving = $state(false);

// Abgeleiteter State (wird automatisch aktualisiert)
let currentNote = $derived(
    currentNoteId ? notes.get(currentNoteId) : null
);

// Getter-Funktionen exportieren (für Zugriff aus Komponenten)
export function getNotes() { return notes; }
export function getCurrentNote() { return currentNote; }
export function getIsDirty() { return isDirty; }

// Mutations-Funktionen
export function setCurrentNote(id: string) {
    currentNoteId = id;
}

export function updateNote(id: string, data: Partial<Note>) {
    const existing = notes.get(id);
    if (existing) {
        notes.set(id, { ...existing, ...data });
    }
}

export async function saveNote() {
    if (!currentNote || !isDirty) return;
    isSaving = true;
    try {
        await api.updateNote(currentNote.id, currentNote);
        isDirty = false;
    } finally {
        isSaving = false;
    }
}
```

### Alle State-Module

| Datei | Zweck |
|-------|-------|
| `auth.svelte.ts` | User-Session, Token-Expiry, Login/Logout |
| `notes.svelte.ts` | Notiz-Liste (SvelteMap), aktuelle Notiz, Auto-Save, Dirty-State |
| `folders.svelte.ts` | Ordner-Baum, Drag & Drop Reihenfolge |
| `tabs.svelte.ts` | Multi-Tab-State (offene Notiz-Tabs, persistiert via API) |
| `encryption.svelte.ts` | KEK-Status, Security-Level, Unlock-Status |
| `ui.svelte.ts` | Sidebar offen/zu, Editor-Modus, Mobile-Erkennung, Theme |
| `websocket.svelte.ts` | WS-Verbindung, Event-Handler |
| `shopping.svelte.ts` | Einkaufslisten-State |
| `features.svelte.ts` | Feature-Flags (Journal, Graph, Canvas, Shopping) |
| `tree.svelte.ts` | Virtueller Baum für Sidebar |
| `graph.svelte.ts` | Graph Nodes/Edges |
| `journal.svelte.ts` | Journal-Einträge + Kalender |
| `search.svelte.ts` | Suchergebnisse |
| `recipes.svelte.ts` | Rezept-Liste |

### SvelteMap statt Object/Array

Für die Notiz-Liste wird `SvelteMap` (Svelte 5) statt einem normalen Objekt verwendet:

```typescript
// O(1) Lookup statt O(n) Array-Suche
let notes = $state(new SvelteMap<string, Note>());

// Schneller Zugriff
const note = notes.get(noteId);  // O(1)

// Reaktive Updates (SvelteMap triggert automatisch Rerenders)
notes.set(noteId, updatedNote);
```

## Der Notiz-Editor

### Drei Editor-Modi

Je nach `note_type` wird ein anderer Editor lazy-geladen:

```
note_type === 'note'    → Editor.svelte       (Markdown, CodeMirror 6)
note_type === 'recipe'  → RecipeEditor.svelte  (Rezept-spezifisch)
note_type === 'canvas'  → CanvasEditor.svelte  (Infinite Canvas)
```

### Markdown-Editor (CodeMirror 6)

Der Haupteditor basiert auf CodeMirror 6 und bietet:

- **Live Preview:** Side-by-side oder Toggle (Edit / Preview / Split)
- **Wikilinks:** `[[Notiz-Titel]]` mit Autovervollständigung
- **Inline-Titel:** Titel wird direkt im Editor bearbeitet
- **Task-Checkboxen:** Klickbare Checkboxen mit Drag-Reorder
- **Bild-Upload:** Drag & Drop + Paste (wird als Upload zum Server geschickt)
- **Find & Replace:** Suchleiste im Editor
- **Focus Mode:** Alles außer dem Editor ausblenden
- **Dictation:** Spracheingabe via Whisper (OpenAI)

### Editor-Toolbar

```
┌─────────────────────────────────────────────────────────┐
│ B I S ~  H1 H2 H3  Link Img Code Table  │  AI  Share  │
│ Bold Italic Strike  Headings              │  Panel      │
└─────────────────────────────────────────────────────────┘
```

### AI-Sidebar

Rechts vom Editor kann ein KI-Panel geöffnet werden:

- **Zusammenfassung:** LLM fasst die Notiz zusammen (SSE-Streaming)
- **Tag-Vorschläge:** KI schlägt Tags basierend auf Inhalt vor
- **Link-Vorschläge:** KI empfiehlt verwandte Notizen zum Verlinken
- **Rechtschreibprüfung:** LLM-basierte Korrektur
- **Transform:** Text umschreiben/erweitern/kürzen lassen

## API-Client (`lib/api/`)

Der API-Client ist modular aufgebaut — ein Modul pro Feature:

```
api/
├── client.ts          # Base-Fetch mit Auth + Offline-Queue
├── notes.ts           # Notiz-CRUD
├── auth.ts            # Login/Register/Refresh
├── folders.ts         # Ordner-CRUD
├── search.ts          # Suche
├── recipes.ts         # Rezepte
├── shopping.ts        # Einkaufslisten
├── ai.ts              # KI-Endpoints
├── sharing.ts         # Teilen
├── encryption.ts      # Verschlüsselungs-Endpoints
└── ...
```

### Base-Client (`client.ts`)

```typescript
async function apiFetch(path: string, options?: RequestInit) {
    // 1. Base-URL bestimmen (Web vs. Desktop/Tauri)
    const url = getBaseUrl() + path;

    // 2. Auth-Token anhängen
    const headers = {
        ...options?.headers,
        'Authorization': `Bearer ${getAccessToken()}`,
        'X-CSRF-Token': getCsrfToken(),
    };

    // 3. Offline? → In IndexedDB-Queue speichern
    if (!navigator.onLine && isWriteRequest(options)) {
        await offlineQueue.enqueue(path, options);
        return;
    }

    // 4. Request ausführen
    const response = await fetch(url, { ...options, headers });

    // 5. 401? → Token refreshen und erneut versuchen
    if (response.status === 401) {
        await refreshToken();
        return apiFetch(path, options);  // Retry
    }

    return response;
}
```

## Offline-Modus

### Wie funktioniert's?

```
User bearbeitet Notiz
    ↓
Netzwerk verfügbar? ────────── Ja → Normal speichern via API
    ↓ Nein
In IndexedDB-Queue schreiben:
  {type: 'update', path: '/api/notes/123', body: {...}, timestamp: ...}
    ↓
Netzwerk kommt zurück (online Event)
    ↓
Sync-Manager spielt Queue ab:
  1. Älteste Operation zuerst
  2. Konflikte erkennen (Server-Version ≠ lokale Version)
  3. Konflikt-Dialog zeigen: "Server" vs. "Lokal" behalten
```

### IndexedDB-Stores

| Store | Inhalt |
|-------|--------|
| `operation_queue` | Ausstehende API-Operationen |
| `local_note_cache` | Lokale Kopien für Offline-Lesen |
| `temp_id_mappings` | Temporäre IDs → Server-IDs (für offline erstellte Notizen) |

## Theming

Zwei Themes, basierend auf CSS-Variablen (OKLch Farbraum, Tailwind v4):

- **Gruvbox Light** — warme, helle Farben
- **Gruvbox Dark** — warme, dunkle Farben

Theme wird in `localStorage` gespeichert und über CSS-Variablen angewandt.

## Internationalisierung (i18n)

- Bibliothek: `svelte-i18n`
- Sprachen: Deutsch (Standard) + Englisch
- Locale-Dateien: `lib/locales/de.json`, `lib/locales/en.json`
- Gespeichert in `localStorage`, Fallback auf Browser-Sprache

## Nächste Seiten

- [Notizen-und-Wikilinks](Notizen-und-Wikilinks.md) — Editor und Wikilink-System im Detail
- [Verschlüsselung](Verschlüsselung.md) — E2E-Encryption-Architektur
- [Features-im-Detail](Features-im-Detail.md) — Journal, Rezepte, Einkaufslisten
