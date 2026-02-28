# Backend (Go)

## Überblick

Das Backend ist ein einzelner Go-Binary, der sowohl die REST-API als auch das kompilierte Frontend ausliefert. Er nutzt den [Chi-Router](https://github.com/go-chi/chi) (v5) und SQLite als einzige Datenbank.

## Projektstruktur

```
backend/
├── cmd/server/main.go              # Einstiegspunkt
└── internal/
    ├── api/                         # HTTP-Schicht
    │   ├── server.go                # Server-Struct mit allen Services
    │   ├── routes.go                # Alle Route-Definitionen
    │   ├── middleware.go            # Auth, CSRF, Rate-Limit, etc.
    │   ├── note_handler.go          # Notiz-Endpoints
    │   ├── auth_handler.go          # Login/Register/Refresh
    │   ├── folder_handler.go        # Ordner-CRUD
    │   ├── recipe_handler.go        # Rezepte
    │   ├── shopping_handler.go      # Einkaufslisten
    │   ├── admin_handler.go         # Admin-Panel
    │   ├── sharing_handler.go       # Teilen-Funktionen
    │   ├── graph_handler.go         # Wissensgraph
    │   ├── search_handler.go        # Suche
    │   ├── upload_handler.go        # Datei-Uploads
    │   ├── canvas_handler.go        # Canvas-Notizen
    │   ├── journal_handler.go       # Journal
    │   └── ...
    ├── service/                     # Business-Logik
    │   ├── note_service.go          # Notiz-Logik + Cache
    │   ├── auth_service.go          # Auth + Token-Management
    │   ├── two_factor_service.go    # TOTP + Backup-Codes
    │   ├── fido2_service.go         # WebAuthn/Passkeys
    │   ├── graph_service.go         # Graph-Berechnung
    │   ├── summarize_service.go     # KI-Zusammenfassungen
    │   ├── recipe_service.go        # Rezept-Logik
    │   ├── recipe_suggestion_service.go  # KI-Rezeptvorschläge
    │   ├── shopping_service.go      # Einkaufslisten-Logik
    │   ├── sharing_service.go       # Teilen-Logik
    │   ├── admin_service.go         # Admin-Funktionen
    │   └── ...
    ├── db/
    │   ├── database.go              # DB-Öffnung + Migration
    │   └── migrations/              # 59 SQL-Migrationen
    ├── auth/                        # JWT-Generierung
    ├── cache/                       # In-Memory Cache (sync.Map)
    ├── websocket/                   # WebSocket Hub
    ├── jobs/                        # Async Job Queue
    ├── llm/                         # LLM-Provider (Claude, Gemini, ChatGPT)
    ├── parser/                      # Markdown-Parser (Due-Dates, Links)
    ├── crypto/                      # AES-256-GCM Verschlüsselung
    └── fido2/                       # WebAuthn-Konfiguration
```

## Server-Struct (`api/server.go`)

Der `Server` ist das zentrale Struct, das alle Services hält:

```go
type Server struct {
    Router            *chi.Mux
    NoteService       *service.NoteService
    AuthService       *service.AuthService
    FolderService     *service.FolderService
    GraphService      *service.GraphService
    SummarizeService  *service.SummarizeService
    RecipeService     *service.RecipeService
    ShoppingService   *service.ShoppingService
    SharingService    *service.SharingService
    AdminService      *service.AdminService
    // ... weitere Services
    WSManager         *websocket.Manager
    JobManager        *jobs.Manager
}
```

Jeder Service wird in `main.go` erstellt und dem Server übergeben. Die Handler-Methoden sind Methoden auf dem `Server`-Struct, damit sie Zugriff auf alle Services haben.

## Routing (`api/routes.go`)

Die Routen sind in Gruppen organisiert:

```go
func (s *Server) SetupRoutes() {
    r := s.Router

    // Globale Middleware
    r.Use(middleware.Logger)
    r.Use(panicRecoveryMiddleware)
    r.Use(middleware.RequestID)
    r.Use(gzipMiddleware)
    r.Use(corsMiddleware)
    r.Use(securityHeadersMiddleware)

    // Öffentliche Routen (kein Auth nötig)
    r.Group(func(r chi.Router) {
        r.Post("/api/auth/login", s.Login)
        r.Post("/api/auth/register", s.Register)
        r.Post("/api/auth/refresh", s.RefreshToken)
        // ...
    })

    // Geschützte Routen (Auth + CSRF)
    r.Group(func(r chi.Router) {
        r.Use(s.authMiddleware)
        r.Use(s.csrfMiddleware)
        r.Use(middleware.Timeout(60 * time.Second))

        // Notizen
        r.Get("/api/notes", s.GetNotes)
        r.Post("/api/notes", s.CreateNote)
        r.Get("/api/notes/{id}", s.GetNote)
        r.Put("/api/notes/{id}", s.UpdateNote)
        r.Delete("/api/notes/{id}", s.DeleteNote)
        // ...

        // Admin-Bereich (zusätzliche Admin-Middleware)
        r.Group(func(r chi.Router) {
            r.Use(s.adminMiddleware)
            r.Get("/api/admin/stats", s.GetStats)
            // ...
        })
    })
}
```

## Middleware im Detail

### Auth-Middleware (`middleware.go`)

```
Request kommt rein
    ↓
Prüfe "Authorization: Bearer <token>" Header
    ↓ (falls leer)
Prüfe "access_token" HttpOnly Cookie
    ↓
JWT validieren (HS256, Expiry prüfen)
    ↓
User-ID aus Claims extrahieren
    ↓
In Request-Context speichern: ctx.Value("userID")
    ↓
Nächster Handler
```

### CSRF-Middleware

Schützt vor Cross-Site Request Forgery:
- Setzt ein `csrf_token` Cookie (random, HttpOnly=false damit JS es lesen kann)
- Erwartet `X-CSRF-Token` Header bei jedem Schreib-Request
- Vergleicht Cookie-Wert mit Header-Wert
- **Ausnahme:** Requests mit Bearer-Token (API-Clients/Desktop) brauchen kein CSRF

### Rate-Limiting

Jeder sensible Endpoint hat sein eigenes Rate-Limit:
- Login: strikt (Brute-Force-Schutz)
- Register: sehr strikt
- Suche: moderat
- KI-Endpoints: moderat (LLM-Kosten begrenzen)

## Handler-Pattern

Alle Handler folgen dem gleichen Muster:

```go
func (s *Server) UpdateNote(w http.ResponseWriter, r *http.Request) {
    // 1. User-ID aus Context
    userID := r.Context().Value("userID").(string)

    // 2. Parameter aus URL
    noteID := chi.URLParam(r, "id")

    // 3. Request-Body parsen
    var req UpdateNoteRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // 4. Service aufrufen (Business-Logik)
    note, err := s.NoteService.UpdateNote(userID, noteID, req.Title, req.Content)
    if err != nil {
        // Fehler-Handling (nie interne Details an Client!)
        http.Error(w, "Failed to update note", http.StatusInternalServerError)
        return
    }

    // 5. WebSocket-Broadcast an andere Tabs
    s.WSManager.Broadcast(userID, websocket.Event{
        Type: "note_updated",
        Data: note,
    })

    // 6. JSON-Response
    json.NewEncoder(w).Encode(note)
}
```

## Service-Layer

### NoteService — Das Herzstück

Der `NoteService` verwaltet Notizen und ist der komplexeste Service:

```go
type NoteService struct {
    db    *sql.DB
    cache *cache.Cache  // In-Memory Cache (5min TTL)
}
```

**Wichtige Methoden:**

| Methode | Was sie tut |
|---------|-------------|
| `CreateNote()` | Notiz erstellen, Wikilinks parsen, FTS-Index aktualisieren |
| `UpdateNote()` | Notiz aktualisieren, Version-Snapshot erstellen, Links aktualisieren |
| `DeleteNote()` | Soft-Delete (setzt `deleted_at`), verschiebt in Papierkorb |
| `GetNote()` | Notiz laden (mit Cache), Sharing-Berechtigungen prüfen |
| `SearchNotes()` | FTS5-Volltextsuche mit Snippet-Highlighting |
| `RenameNote()` | Async-Job: Titel ändern + alle `[[OldTitle]]` Links in anderen Notizen aktualisieren |
| `ParseWikilinks()` | `[[Titel]]` aus Markdown extrahieren, gegen Note-Tabelle auflösen |

**Wikilink-Auflösung:**
```
Markdown: "Siehe [[Meine Notiz]] für Details"
    ↓
Parser findet: "Meine Notiz"
    ↓
DB-Query: SELECT id FROM notes WHERE title = 'Meine Notiz' AND user_id = ?
    ↓
Links-Tabelle: INSERT INTO links (source_note_id, target_note_id)
    ↓
Falls Titel nicht gefunden: unresolved_link_refs speichern
```

### AuthService

Verwaltet Login, Registration und Token-Lifecycle:

- **Passwort:** bcrypt-Hash mit konstantem Zeitvergleich
- **Access Token:** JWT (HS256), 15 Minuten TTL
- **Refresh Token:** Zufällige 30 Bytes, SHA-256-gehasht in DB, 30 Tage TTL
- **Token-Rotation:** Bei jedem Refresh wird ein neues Token-Paar ausgegeben
- **Replay-Detection:** Wenn ein bereits rotiertes Refresh-Token wiederverwendet wird, werden **alle** Tokens der Familie widerrufen

### GraphService

Berechnet den Wissensgraph:

```
Alle Notizen des Users laden
    ↓
Alle Links laden (links-Tabelle)
    ↓
Nodes bauen: [{id, title, folder, linkCount}]
Edges bauen: [{source, target}]
    ↓
Pro User im Cache halten (5min TTL)
    ↓
Frontend rendert mit force-graph (D3)
```

## WebSocket (`websocket/`)

Der WebSocket-Manager hält pro User eine Liste offener Connections:

```go
type Manager struct {
    connections map[string][]*Connection  // userID → Connections
    register    chan *Connection
    unregister  chan *Connection
    broadcast   chan BroadcastMessage
}
```

Events die über WebSocket gepusht werden:
- `note_updated` — Notiz wurde in einem anderen Tab gespeichert
- `note_deleted` — Notiz wurde gelöscht
- `folder_updated` — Ordner-Änderung
- Shared-Note-Updates (wenn ein anderer User eine geteilte Notiz bearbeitet)

Ping/Pong Keepalive alle 50 Sekunden.

## Job-System (`jobs/`)

Für CPU-intensive Operationen gibt es eine einfache In-Process Job-Queue:

```
Client: POST /api/notes/{id}/rename
    ↓
Handler erstellt Job: JobTypeRenameNote
    ↓
Job-ID sofort zurückgeben (202 Accepted)
    ↓
Worker-Goroutine:
  - Notiz umbenennen
  - ALLE Notizen des Users durchsuchen
  - [[Alter Titel]] → [[Neuer Titel]] ersetzen
    ↓
Client pollt: GET /api/jobs/{id}
  → Status: "running" / "completed" / "failed"
```

Jobs verfallen nach 24 Stunden aus dem Speicher.

## Nächste Seiten

- [Datenbank](Datenbank.md) — SQLite-Schema und Migrationen
- [API-Referenz](API-Referenz.md) — Alle REST-Endpunkte
- [Authentifizierung-und-Sicherheit](Authentifizierung-und-Sicherheit.md) — Auth-Flow im Detail
