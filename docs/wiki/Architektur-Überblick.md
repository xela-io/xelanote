# Architektur-Überblick

## Monorepo-Struktur

xelanote ist ein Monorepo mit zwei Hauptteilen:

```
xelanote/
├── backend/          # Go HTTP-Server (Single Binary)
│   ├── cmd/server/   # Einstiegspunkt (main.go)
│   └── internal/     # Gesamte Business-Logik
├── frontend/         # SvelteKit SPA
│   ├── src/          # Svelte-Quellcode
│   └── src-tauri/    # Desktop-App (Rust/Tauri)
├── docs/             # Dokumentation
├── scripts/          # Build-Skripte
└── Makefile          # Build-Targets
```

## Wie alles zusammenspielt

```
┌─────────────────────────────────────────────────┐
│                   Browser / Tauri                │
│  ┌─────────────────────────────────────────────┐ │
│  │            SvelteKit Frontend               │ │
│  │  ┌──────┐ ┌────────┐ ┌──────┐ ┌─────────┐  │ │
│  │  │Editor│ │Sidebar │ │Graph │ │Encrypted│  │ │
│  │  │(CM6) │ │(Tree)  │ │(D3)  │ │(sodium) │  │ │
│  │  └──┬───┘ └───┬────┘ └──┬───┘ └────┬────┘  │ │
│  │     │         │         │           │       │ │
│  │  ┌──┴─────────┴─────────┴───────────┴────┐  │ │
│  │  │        API Client (fetch + auth)      │  │ │
│  │  │   + Offline Queue (IndexedDB)         │  │ │
│  │  └──────────────┬────────────────────────┘  │ │
│  └─────────────────┼───────────────────────────┘ │
└────────────────────┼─────────────────────────────┘
                     │ HTTP/WS
┌────────────────────┼─────────────────────────────┐
│  Go Backend        │                             │
│  ┌─────────────────┴────────────────────────┐    │
│  │         Chi Router + Middleware           │    │
│  │  (Auth, CORS, CSRF, Rate-Limit, Gzip)    │    │
│  └──────────┬───────────────────┬───────────┘    │
│             │                   │                │
│  ┌──────────┴──────┐  ┌────────┴─────────┐      │
│  │   API Handlers  │  │   WebSocket Hub  │      │
│  │  (routes.go)    │  │  (real-time sync)│      │
│  └──────────┬──────┘  └────────┬─────────┘      │
│             │                  │                 │
│  ┌──────────┴──────────────────┴─────────┐       │
│  │          Service Layer                │       │
│  │  NoteService, AuthService, etc.       │       │
│  └──────────────────┬───────────────────┘       │
│                     │                            │
│  ┌──────────────────┴───────────────────┐       │
│  │           SQLite (FTS5)              │       │
│  │    + In-Memory Cache (5min TTL)      │       │
│  └──────────────────────────────────────┘       │
└──────────────────────────────────────────────────┘
```

## Server-Start (main.go)

Wenn der Server startet, passiert folgendes in `backend/cmd/server/main.go`:

1. **Config laden** — JWT-Secret, DB-Pfad, SQLCipher-Key aus Umgebungsvariablen
2. **SQLite öffnen** — `db.Open()` mit WAL-Journal-Mode (für parallele Reads)
3. **Migrationen** — Alle SQL-Dateien in `db/migrations/` werden sequentiell angewandt
4. **Maintenance** — Alte Activity-Logs bereinigen, abgelaufene Refresh-Tokens löschen, PRAGMA optimize
5. **Services initialisieren** — Jeder Service bekommt die DB-Connection injiziert
6. **HTTP-Server starten** — Chi-Router auf Port 8080 mit Graceful Shutdown

## Schichtenarchitektur (Backend)

Das Backend folgt strikt einer 3-Schichten-Architektur:

```
HTTP Handler (api/)  →  Service (service/)  →  Database (db/)
     ↓                       ↓                      ↓
  Request parsen        Business-Logik          SQL ausführen
  Response senden       Validierung             Daten lesen/schreiben
  Auth prüfen           Cache verwalten
```

**Wichtige Regel:** Handler dürfen nie direkt auf die DB zugreifen — immer über den Service.

## Request-Lifecycle

Ein typischer API-Request durchläuft:

```
Client Request
    ↓
Chi Router (Middleware-Stack)
    ↓
1. Logger Middleware        → Request-ID + Logging
2. Panic Recovery           → Fängt Panics, filed Forgejo-Issue
3. Gzip Compression         → Response komprimieren
4. CORS                     → Origin prüfen
5. Security Headers         → CSP, X-Frame-Options, etc.
6. Auth Middleware           → JWT aus Cookie/Header validieren
7. CSRF Middleware           → Double-Submit Cookie prüfen
8. Rate Limiter              → Request-Limit pro Endpoint
9. Timeout (60s)             → Maximale Bearbeitungszeit
    ↓
Handler-Funktion
    ↓
Service-Aufruf
    ↓
DB-Query
    ↓
JSON Response
```

## Datenfluss: Notiz speichern

Ein konkretes Beispiel — was passiert, wenn du eine Notiz speicherst:

```
1. [Frontend] CodeMirror Editor löst Auto-Save aus (Debounce)
2. [Frontend] Falls verschlüsselt: Content mit DEK verschlüsseln (XChaCha20)
3. [Frontend] PUT /api/notes/{id} mit Content + Version
4. [Backend]  Auth-Middleware prüft JWT
5. [Backend]  NoteHandler.UpdateNote() parst Request-Body
6. [Backend]  NoteService.UpdateNote():
              - Version-Check (Optimistic Locking)
              - Wikilinks parsen: [[Titel]] → Note-IDs auflösen
              - Links-Tabelle aktualisieren
              - Due-Dates extrahieren (@due(2024-01-15))
              - FTS5-Index aktualisieren
              - Version-Snapshot erstellen (falls genug Zeit vergangen)
              - Cache invalidieren
7. [Backend]  WebSocket: Broadcast an alle anderen Tabs/Sessions des Users
8. [Frontend] Andere Tabs empfangen WS-Event → aktualisieren ihre Ansicht
```

## Nächste Seiten

- [Backend](Backend.md) — Details zum Go-Server
- [Frontend](Frontend.md) — Details zum SvelteKit-Frontend
- [Datenbank](Datenbank.md) — SQLite-Schema und Migrationen
