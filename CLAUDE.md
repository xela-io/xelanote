# CLAUDE.md

Kurzueberblick fuer KI-Agenten. Hinterfrage alles kritisch.

## Projekt

- **Backend**: Go/Chi/SQLite in `backend/` | **Frontend**: SvelteKit + Tauri in `frontend/`
- **Docs**: `docs/` (Index: `docs/index.md`) | **Planung**: `TODO.md`, `ROADMAP.md`, `docs/planning/`
- **Datenbank**: `./data/xelanote.db` (konfigurierbar via `XELANOTE_DB`)

### Projektstruktur

```
├── backend/
│   ├── cmd/server/              # HTTP-Server-Einstiegspunkt
│   └── internal/
│       ├── api/                 # HTTP-Handler, Middleware, Routen
│       ├── service/             # Business-Logik
│       ├── db/migrations/       # SQL-Migrationen (inkrementell)
│       ├── auth/                # Authentifizierung & Autorisierung
│       ├── cache/               # In-Memory-Caching
│       ├── websocket/           # WebSocket-Handler
│       ├── jobs/                # Async-Job-Verarbeitung
│       ├── llm/                 # LLM-Integration
│       ├── parser/              # Markdown/Text-Parsing
│       ├── crypto/              # Verschluesselung
│       └── fido2/               # FIDO2/WebAuthn
├── frontend/
│   ├── src/
│   │   ├── routes/              # SvelteKit-Seiten (file-based routing)
│   │   │   ├── note/            # Notiz-Editor
│   │   │   ├── recipes/         # Rezeptverwaltung
│   │   │   ├── search/          # Suche
│   │   │   ├── graph/           # Wissensgraph
│   │   │   ├── settings/        # Benutzereinstellungen
│   │   │   ├── admin/           # Admin-Panel
│   │   │   ├── journal/         # Journal-Ansicht
│   │   │   ├── due-dates/       # Aufgaben/Deadlines
│   │   │   └── shared/          # Geteilte Notizen
│   │   └── lib/
│   │       ├── api/             # API-Client-Methoden
│   │       ├── components/      # Wiederverwendbare UI-Komponenten
│   │       ├── editor/          # Editor-Utilities
│   │       ├── stores/          # Svelte 5 Runes (State)
│   │       ├── types/           # TypeScript-Typen
│   │       ├── utils/           # Hilfsfunktionen
│   │       ├── crypto/          # Client-seitige Verschluesselung
│   │       ├── offline/         # Offline-Modus
│   │       ├── themes/          # Theme-Definitionen
│   │       └── locales/         # i18n-Strings
│   ├── src-tauri/               # Desktop-App (Tauri/Rust)
│   ├── tests/e2e/               # Playwright E2E-Tests
│   └── static/                  # Statische Assets
├── docs/                        # Dokumentation (Architektur, Guides, Audits)
│   ├── planning/                # Feature-Planung
│   ├── security/                # Sicherheits-Reviews
│   ├── postmortems/             # Incident-Analysen
│   └── performance/             # Performance-Berichte
├── scripts/                     # Build- & Utility-Skripte
├── .github/workflows/           # CI/CD (ci.yml, quality.yml, security.yml)
├── Makefile                     # Build-Targets
├── docker-compose.yml           # Docker-Services
├── Dockerfile                   # Multi-Stage-Build
└── lefthook.yml                 # Pre-Commit-Hooks
```

## Regeln

- Backend-Tags lokal: `-tags "fts5 sqlite_crypt"` (Makefile-Default), Docker nur `-tags "fts5"`. Der Unterschied ist beabsichtigt: `sqlite_crypt` aktiviert SQLCipher-Unterstuetzung, die in Docker-Production nicht benoetigt wird (kein SQLCipher-Package im Alpine-Image). Lokale Builds koennen damit verschluesselte DBs oeffnen.
- `JWT_SECRET` min. 64 Zeichen, `CORS_ALLOWED_ORIGINS` Pflicht in Produktion
- Uploads owner-only, Refresh Tokens gehasht
- Migrationen: `backend/internal/db/migrations/` (inkrementell)
- **Vor neuen Features: [`docs/conventions.md`](docs/conventions.md) lesen**
- Backend: API -> Service -> DB (nie Schichten ueberspringen)
- Frontend: Nur Svelte 5 Runes, keine Svelte 4 Stores
- Security: Kein localStorage fuer Auth, keine internen Fehlerdetails an Client

## Kommandos

| Aufgabe | Befehl |
|---------|--------|
| Lokale Entwicklung | `make dev` + `make run-frontend` |
| Init / Build | `make init` / `make build` |
| Tests | `make test`, `make test-frontend`, `make test-e2e` |
| Docker | `make docker`, `docker compose up -d` |


## Workflow

1. Lokal entwickeln & testen (`make dev` + `make run-frontend`)
2. **CHANGELOG.md pflegen** (`[Unreleased]`, Keep a Changelog-Style) - lefthook prueft das
3. Commit mit `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>`
4. Push: `git push origin main` (GitHub)

## Git & Deployment

- GitHub: CI-Checks (`ci.yml`, `quality.yml`, `security.yml`)
- Production: https://xelanote.com (Static-only SPA, nginx:alpine)
- Deploy: Frontend bauen → `rsync` Build auf Server → `docker compose up -d --build`
- Server: `xela@46.224.208.7`, App in `~/xelanote.com/`
- Details: `docs/deployment.md`
