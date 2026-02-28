# Entwicklung Setup

## Voraussetzungen

| Tool | Version | Zweck |
|------|---------|-------|
| **Go** | 1.21+ | Backend-Entwicklung |
| **Node.js** | 20+ | Frontend-Entwicklung |
| **pnpm** | 8+ | Package Manager (Frontend) |
| **Make** | - | Build-Automation |
| **Docker** | - | Container-Builds (optional) |
| **Rust** | - | Tauri Desktop-App (optional) |

## Erster Start

```bash
# Repository klonen
git clone git@github.com:xela-io/xelanote.git
cd xelanote

# Alles initialisieren (Go-Dependencies + npm install)
make init

# Backend starten (Port 8080)
make dev

# In einem zweiten Terminal: Frontend starten
make run-frontend
```

Das Frontend (SvelteKit Dev-Server) läuft dann auf `http://localhost:5173` und proxied API-Requests an das Backend auf Port 8080.

## Makefile-Targets

| Target | Beschreibung |
|--------|-------------|
| `make init` | Go + Node Dependencies installieren |
| `make dev` | Backend im Dev-Mode starten |
| `make run-frontend` | Frontend Dev-Server starten |
| `make build` | Production-Build (Backend + Frontend) |
| `make test` | Backend-Tests |
| `make test-frontend` | Frontend-Tests |
| `make test-e2e` | Playwright E2E-Tests |
| `make docker` | Docker-Image bauen |

## Build-Tags

```bash
# Lokale Builds (mit SQLCipher-Support)
go build -tags "fts5 sqlite_crypt" ./cmd/server/

# Docker-Builds (ohne SQLCipher)
go build -tags "fts5" ./cmd/server/
```

**Warum der Unterschied?** `sqlite_crypt` aktiviert SQLCipher für verschlüsselte DB-Dateien. Das Alpine-Docker-Image hat kein SQLCipher-Package, daher wird es dort weggelassen.

## Umgebungsvariablen

| Variable | Pflicht | Beschreibung |
|----------|---------|-------------|
| `JWT_SECRET` | Ja | Mind. 64 Zeichen, für JWT-Signierung |
| `CORS_ALLOWED_ORIGINS` | Prod | Erlaubte Origins (kommasepariert) |
| `XELANOTE_DB` | Nein | DB-Pfad (Default: `./data/xelanote.db`) |
| `SQLITE_CIPHER_KEY` | Nein | SQLCipher-Passwort (nur lokale Builds) |

## Projektstruktur verstehen

### Backend: Wo anfangen?

1. **`cmd/server/main.go`** — Server-Start lesen (was wird initialisiert?)
2. **`internal/api/routes.go`** — Alle Routen auf einen Blick
3. **`internal/api/note_handler.go`** — Typischer Handler als Beispiel
4. **`internal/service/note_service.go`** — Business-Logik
5. **`internal/db/migrations/`** — DB-Schema verstehen

### Frontend: Wo anfangen?

1. **`src/routes/+layout.svelte`** — Root-Layout (App-Initialisierung)
2. **`src/routes/+page.svelte`** — Home-Dashboard
3. **`src/routes/note/[id]/+page.svelte`** — Notiz-Editor
4. **`src/lib/stores/notes.svelte.ts`** — State-Management-Beispiel
5. **`src/lib/api/client.ts`** — API-Client Basis

### Code-Konventionen

Vor neuen Features unbedingt lesen: `docs/conventions.md`

**Backend-Regeln:**
- Schichtenarchitektur: API → Service → DB (nie überspringen)
- Fehler nie an den Client durchreichen (generische Fehlermeldungen)
- Neue DB-Felder immer als Migration

**Frontend-Regeln:**
- Nur Svelte 5 Runes (kein `writable()`, `readable()`, etc.)
- Kein `localStorage` für Auth-Tokens
- TypeScript-Typen in `lib/types/`

## Git-Workflow

```bash
# Entwickeln
make dev          # Backend
make run-frontend # Frontend

# Testen
make test
make test-frontend

# CHANGELOG.md pflegen (lefthook prüft das!)
# [Unreleased] Abschnitt aktualisieren

# Committen
git add .
git commit -m "feat: neue Feature-Beschreibung"

# Pushen (geht an Forgejo + GitHub gleichzeitig)
git push forgejo main
```

### Remotes

```
forgejo  → Forgejo (git.over-cloud.de) + GitHub (Push an beide)
origin   → GitHub nur (wenn explizit gewünscht)
```

### CI/CD

- **Forgejo:** Staging-Deploy bei Push auf `main`, Production bei Tags
- **GitHub:** CI-Checks (`ci.yml`, `quality.yml`, `security.yml`)
- **Staging:** https://notes.over-cloud.de (Homelab)
- **Production:** https://xelanote.com (Hetzner)

## Docker

```bash
# Image bauen
make docker

# Container starten
docker compose up -d
```

Das `Dockerfile` ist ein Multi-Stage-Build:
1. **Build-Stage:** Go + Node Builds
2. **Final-Stage:** Alpine mit dem Single Binary

## Pre-Commit Hooks (lefthook)

`lefthook.yml` definiert Hooks die vor jedem Commit laufen:
- CHANGELOG.md wurde aktualisiert?
- Go Linting
- Frontend Linting

Hooks temporär überspringen: `LEFTHOOK=0 git commit ...`

## E2E-Tests (Playwright)

```bash
make test-e2e
```

Tests liegen in `frontend/tests/e2e/`. Sie starten einen echten Browser und testen die komplette App.

## Nächste Seiten

- [Architektur-Überblick](Architektur-Überblick.md) — Wie alles zusammenspielt
- [Backend](Backend.md) — Backend-Details
- [Frontend](Frontend.md) — Frontend-Details
