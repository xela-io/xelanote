# Development Guide

Dieses Dokument beschreibt das Setup, die Entwicklungs-Workflows und Best Practices für die Arbeit an xelanote.

## Inhaltsverzeichnis

- [Voraussetzungen](#voraussetzungen)
- [Projekt-Setup](#projekt-setup)
- [Entwicklungs-Workflow](#entwicklungs-workflow)
- [Projektstruktur](#projektstruktur)
- [Testing](#testing)
- [CI/CD](#cicd)
- [Security & Configuration](#security--configuration)
- [Building & Deployment](#building--deployment)
- [Code-Standards](#code-standards)
- [Debugging](#debugging)
- [Häufige Probleme](#häufige-probleme)
- [Contributing](#contributing)
- [Version History Feature](#version-history-feature)
- [Mobile Sidebar Feature](#mobile-sidebar-feature)
- [Graph View Mobile Touch Interaction](#graph-view-mobile-touch-interaction)
- [Performance-Optimierungen](#performance-optimierungen)
- [Editor TAB-Einrückung](#editor-tab-einrückung)
- [Markdown Preview Styling](#markdown-preview-styling)

**Note**: For accessibility features and guidelines, see [docs/accessibility.md](./accessibility.md)

---

## Voraussetzungen

### System Requirements

| Tool | Version | Zweck |
|------|---------|-------|
| **Go** | 1.25+ | Backend Development |
| **Node.js** | 20+ | Frontend Build |
| **npm** | 10+ | Frontend Package Manager |
| **Docker** | 24+ | Optional: Container Deployment |
| **Make** | - | Optional: Build Automation |
| **Git** | - | Version Control |

### Installation

**macOS (Homebrew)**:

```bash
brew install go node npm
```

**Ubuntu/Debian**:

```bash
# Go
wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Node.js
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs
```

**Windows**:

Verwende offizielle Installer:
- [Go Downloads](https://go.dev/dl/)
- [Node.js Downloads](https://nodejs.org/)

### Verifizierung

```bash
go version    # go1.25.0 oder höher
node --version  # v20.x.x oder höher
npm --version   # 10.x.x oder höher
```

---

## Projekt-Setup

### 1. Repository Klonen

```bash
git clone https://github.com/xela-io/xelanote.git
cd xelanote
```

### 2. Abhängigkeiten Installieren

**Mit Make** (empfohlen):

```bash
make init
```

**Manuell**:

```bash
# Backend Dependencies
cd backend
go mod download

# Frontend Dependencies
cd ../frontend
npm install
```

**Wichtig für Entwicklung**: Stelle sicher, dass folgende Frontend-Dependencies korrekt installiert sind:

```bash
cd frontend
npm install @tailwindcss/postcss@latest  # Für Tailwind v4
npm install @sveltejs/vite-plugin-svelte@^5.0.0  # Für Vite 6 Kompatibilität
```

### 3. Projekt-Initialisierung

Das Projekt ist jetzt bereit für Development. Fahre mit [Entwicklungs-Workflow](#entwicklungs-workflow) fort.

---

## Entwicklungs-Workflow

### Development Mode

Für aktive Entwicklung laufen Frontend und Backend **separat**:

1. **Backend starten** (Terminal 1):

   ```bash
   cd backend
   go run -tags "fts5 sqlite_crypt" ./cmd/server -addr :8080
   ```

   Oder mit Make:

   ```bash
   make run-backend
   ```

   Backend läuft auf `http://localhost:8080`.

   **Wichtig**: `fts5` ist erforderlich. Die Makefile-Targets nutzen standardmaessig `sqlite_crypt`.
   Wenn keine SQLCipher-Libs installiert sind, nutze `-tags "fts5"` oder passe das Makefile an.

2. **Frontend starten** (Terminal 2):

   ```bash
   cd frontend
   npm run dev
   ```

   Oder mit Make:

   ```bash
   make run-frontend
   ```

   Frontend läuft auf `http://localhost:5173` mit Proxy zu Backend.

3. **App öffnen**:

   Öffne `http://localhost:5173` im Browser.

### Warum separate Entwicklung?

- **Hot Reload**: SvelteKit recompiled automatisch bei Änderungen
- **Schnellere Builds**: Kein Rebuild des kompletten Go Binaries
- **Besseres Debugging**: Separate Logs für Frontend/Backend

### Vite Proxy Konfiguration

`frontend/vite.config.ts`:

```typescript
export default {
  server: {
    proxy: {
      '/api': 'http://localhost:8080'  // API Requests → Backend
    }
  }
}
```

**Effekt**: `fetch('/api/notes')` wird zu `http://localhost:8080/api/notes` geroutet.

---

## Projektstruktur

### Backend (`backend/`)

```
backend/
├── cmd/
│   └── server/
│       ├── main.go          # Einstiegspunkt
│       └── static/          # Embedded Frontend (nach Build)
├── internal/
│   ├── api/                 # HTTP Layer (auth, notes, folders, search, templates, snippets, versions, ws)
│   ├── auth/                # JWT + Refresh Token Logic
│   ├── cache/               # In-Memory Cache
│   ├── db/                  # Database Layer
│   │   ├── migrations/      # SQL Migrations
│   │   └── schema.sql       # Base Schema
│   ├── jobs/                # Background Jobs
│   ├── parser/              # Wikilink Parser
│   ├── service/             # Business Logic (auth, notes, graph, templates, snippets)
│   └── websocket/           # WebSocket Manager
├── Dockerfile               # SQLCipher Build (optional)
├── go.mod                   # Go Dependencies
└── go.sum
```

**Layer-Trennung**:

- `cmd/`: Executable Entry Points
- `internal/`: Nicht-exportierbare Packages (intern)
- `api/`: HTTP Request/Response Handling
- `service/`: Business Logic (unabhängig von HTTP)
- `db/`: Datenbank-Operationen
- `parser/`: Standalone Parser (keine Dependencies)

### Frontend (`frontend/`)

```
frontend/
├── src/
│   ├── lib/
│   │   ├── components/      # Svelte Components
│   │   │   ├── Editor.svelte
│   │   │   ├── Sidebar.svelte
│   │   │   ├── QuickSwitcher.svelte
│   │   │   └── ...
│   │   ├── stores/          # Svelte Stores
│   │   │   ├── auth.svelte.ts
│   │   │   ├── notes.svelte.ts
│   │   │   ├── search.svelte.ts
│   │   │   ├── tree.svelte.ts
│   │   │   ├── ui.svelte.ts
│   │   │   └── websocket.svelte.ts
│   │   ├── editor/          # Editor Utilities
│   │   │   ├── codemirror.ts
│   │   │   └── markdown.ts
│   │   ├── api/             # API Module (Client + Domains)
│   │   └── api.ts           # API Facade (Re-Exports)
│   ├── routes/              # SvelteKit Routes
│   │   ├── +page.svelte     # Main Page
│   │   ├── +layout.svelte   # Layout
│   │   ├── note/[id]/+page.svelte
│   │   ├── graph/+page.svelte
│   │   ├── search/+page.svelte
│   │   ├── trash/+page.svelte
│   │   ├── login/+page.svelte
│   │   └── register/+page.svelte
│   └── app.html             # HTML Template
├── static/                  # Static Assets
├── package.json
├── vite.config.ts
├── svelte.config.js
└── tsconfig.json
```

**SvelteKit Konventionen**:

- `routes/`: Filesystem-based Routing
- `+page.svelte`: Page Component
- `+layout.svelte`: Layout Wrapper
- `[id]/`: Dynamic Route Parameter

**API Usage**:
- Frontend Call-Sites nutzen `import * as api from '$lib/api'` (Facade).

---

## Testing

### Backend Tests

**Alle Tests ausführen**:

```bash
cd backend
CGO_ENABLED=1 go test -tags "fts5 sqlite_crypt" -v ./...
# Ohne SQLCipher: -tags "fts5"
```

Oder mit Make:

```bash
make test
```

**Spezifische Tests**:

```bash
# Parser Tests
make test-parser

# Oder manuell
cd backend
go test -v ./internal/parser/...
```

**Mit Coverage**:

```bash
cd backend
go test -cover ./...
```

### Parser Tests

Die wichtigsten Tests sind für den Wikilink-Parser:

```bash
cd backend
go test -v ./internal/parser/... -run TestParse
```

**Test-Struktur** (`backend/internal/parser/wikilink_test.go`):

```go
func TestParse(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []WikiLink
    }{
        {
            name:  "basic wikilink",
            input: "Text [[Link]] more text",
            expected: []WikiLink{
                {TargetTitle: "Link", SpanStart: 5, SpanEnd: 13},
            },
        },
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Parse(tt.input)
            // Assert expected == result
        })
    }
}
```

**Test Data**:

Test-Vektoren befinden sich in `testdata/parser/`:

```
testdata/parser/
├── basic.md              # Basis-Wikilinks
├── code_fences.md        # Code Blocks
├── inline_code.md        # Inline Code
├── edge_cases.md         # Edge Cases
└── nested.md             # Verschachtelte Strukturen
```

**Wichtig: UTF-8 Byte-Offsets**

Der Wikilink-Parser arbeitet mit **Byte-Offsets** (nicht Zeichen-Offsets), weil Go's String-Indexierung bytebasiert ist. Bei Tests mit Nicht-ASCII-Zeichen ist das kritisch:

- **ASCII-Zeichen**: 1 Byte pro Zeichen (z.B. `a`, `A`, `1`, `!`)
- **Umlaute**: 2 Bytes (z.B. `ä` = `0xC3 0xA4`, `ö` = `0xC3 0xB6`)
- **Emojis**: 4 Bytes (z.B. `📝` = `0xF0 0x9F 0x93 0x9D`)

**Beispiel Test mit Umlauten**:

```go
// "Täst [[Link]]" - Beachte: ä = 2 Bytes!
// Position 0: T
// Position 1-2: ä (2 Bytes!)
// Position 3: s

---

## CI/CD

Aktuell ist keine CI-Konfiguration im Repo versioniert (kein `.github/workflows/` im Repo-Root).
Lokale Entsprechungen:

- Backend: `make test`, `make lint`
- Frontend: `make test-frontend`, `make test-e2e`, `cd frontend && npm run check`
- Build: `make build`, `make docker`

### Setup

Alle Dependencies werden einmalig installiert:

```bash
make init
```

Dies lädt:
- Go-Module (Standard Go Cache)
- npm-Packages (`frontend/node_modules`)

Tests nutzen lokale Caches:
- `make test` und `make test-parser` setzen `GOCACHE`/`GOMODCACHE` auf `.cache/`.

---

## Security & Configuration

### CORS

Setze `CORS_ALLOWED_ORIGINS` (CSV) für produktive Deployments, z.B.:

```bash
export CORS_ALLOWED_ORIGINS="https://notes.example.com,https://admin.example.com"
```

Wenn die Variable leer ist, werden alle Origins zugelassen (nur für lokale Entwicklung gedacht).

### Refresh Tokens

Refresh Tokens werden gehasht gespeichert. Nach Migration `006_refresh_token_hash.sql` müssen sich alle Nutzer neu einloggen.

### Uploads

Upload-URLs sind nutzergebunden und nur mit gültigem JWT zugreifbar. Neue Upload-URLs sehen so aus:

```
/api/uploads/{filename}
```

Ältere URLs mit `/api/uploads/{user_id}/{filename}` werden weiterhin akzeptiert, aber nur für den eigenen `user_id`.
// Position 4: t
// Position 5: (space)
// Position 6-7: [[

input := "Täst [[Link]]"
expected := WikiLink{
    TargetTitle: "Link",
    SpanStart:   5,   // Nicht 4! (ä verbraucht 2 Bytes)
    SpanEnd:     13,  // Nicht 12!
}
```

**Tipp**:
- `len(string)` gibt Byte-Count zurück
- `utf8.RuneCountInString()` gibt Zeichen-Count zurück
- Bei Parser-Tests immer Byte-Offsets verwenden!

### Benchmarks

**Parser Benchmarks**:

```bash
make bench-parser

# Oder manuell
cd backend
go test -bench=. -benchmem ./internal/parser/...
```

**Beispiel Output**:

```
BenchmarkParse/small-8         50000    25432 ns/op    8192 B/op    12 allocs/op
BenchmarkParse/medium-8        10000   152341 ns/op   65536 B/op    89 allocs/op
BenchmarkParse/large-8          1000  1234567 ns/op  524288 B/op   456 allocs/op
```

### Frontend Tests

**Aktuell**: Vitest + Testing Library + Playwright sind eingerichtet.

```bash
cd frontend
npm run test         # Unit Tests
npm run test:e2e     # E2E Tests (Playwright)
```

---

## Building & Deployment

### Local Build

**Kompletter Build** (Frontend + Backend):

```bash
make build
```

**Output**: `bin/xelanote` (Single Binary)

**Schritte im Detail**:

1. **Frontend Build**:

   ```bash
   cd frontend
   npm run build
   # Output: frontend/build/
   ```

   Erzeugt optimiertes Static Bundle:
   - Minified JavaScript
   - CSS Bundle
   - HTML Templates
   - Asset Hashes

2. **Frontend kopieren**:

   ```bash
   cp -r frontend/build/* backend/cmd/server/static/
   ```

3. **Backend Build**:

   ```bash
   cd backend
   CGO_ENABLED=1 go build -tags "fts5 sqlite_crypt" -o ../bin/xelanote ./cmd/server
   ```

   **Wichtig**:
   - `CGO_ENABLED=1` ist erforderlich für SQLite (benötigt C-Compiler)
   - `-tags "fts5"` aktiviert SQLite FTS5 (erforderlich)
   - `sqlite_crypt` ist fuer SQLCipher (benoetigt `sqlcipher-dev`); Makefile nutzt es standardmaessig

### Binary ausführen

```bash
./bin/xelanote -addr :8080 -db ./data/xelanote.db
```

**Flags**:

- `-addr`: HTTP Listen Address (default: `:8080`)
- `-db`: Database Path (default: `./data/xelanote.db`)

**Umgebungsvariablen**:

```bash
export XELANOTE_DB=/path/to/db.sqlite
export XELANOTE_DB_KEY_FILE=/path/to/db.key  # optional
./bin/xelanote
```

### Docker Build

**Image bauen**:

```bash
make docker

# Oder manuell
docker build -t xelanote:latest .

# Optional: SQLCipher Build
docker build -t xelanote:sqlcipher -f backend/Dockerfile .
```

**Dockerfiles**:
- Root `Dockerfile`: FTS5 (ohne SQLCipher)
- `backend/Dockerfile`: FTS5 + SQLCipher

**Multi-Stage Dockerfile** (`backend/Dockerfile`):

```dockerfile
FROM node:20-alpine AS frontend-builder
WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.24-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev sqlcipher-dev
WORKDIR /app
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-builder /frontend/build ./cmd/server/static/
RUN CGO_ENABLED=1 go build -tags "fts5 sqlite_crypt" -ldflags="-s -w" -o /xelanote ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata sqlcipher
COPY --from=backend-builder /xelanote /app/xelanote
CMD ["/app/xelanote", "-addr", ":8080"]
```

### Docker Compose

**Starten**:

```bash
make docker-up

# Oder manuell
docker compose up -d
```

**docker-compose.yml**:

```yaml
services:
  xelanote:
    build:
      context: .
      dockerfile: Dockerfile
    image: xelanote:latest
    container_name: xelanote
    ports:
      - "8081:8080"
    volumes:
      - xelanote-data:/app/data
    environment:
      - XELANOTE_DB=/app/data/xelanote.db
      - JWT_SECRET=${JWT_SECRET}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - xelanote-net

volumes:
  xelanote-data:

networks:
  xelanote-net:
    driver: bridge
```

**Stoppen**:

```bash
make docker-down
```

### Cross-Compilation

**Für Linux (auf macOS)**:

```bash
cd backend
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags "fts5 sqlite_crypt" -o ../bin/xelanote-linux ./cmd/server
# Ohne SQLCipher: -tags "fts5"
```

**Wichtig**: Cross-Compilation mit CGO erfordert C-Compiler für Target-Plattform.

**Einfacher**: Verwende Docker Build für Linux Binaries.

---

## Code-Standards

### Go

**Formatierung**:

```bash
cd backend
go fmt ./...

# Oder mit Make
make fmt

# Nur prüfen (keine Änderungen)
make fmt-check
```

**Linting**:

```bash
cd backend
go vet ./...

# Oder mit Make
make lint
```

**Empfohlene Tools**:

- `golangci-lint`: [Install Guide](https://golangci-lint.run/docs/welcome/install/local/)

```bash
golangci-lint run ./...
```

**Code Style**:

- Folge [Effective Go](https://go.dev/doc/effective_go)
- Verwende `gofmt` für Formatierung
- Package-Kommentare für alle Packages
- Exported Functions brauchen Doc-Comments

**Beispiel**:

```go
// Package parser provides wikilink extraction from markdown content.
package parser

// Parse extracts all wikilinks from the given content.
// It properly handles code blocks, inline code, and escape sequences.
func Parse(content string) ParseResult {
    // ...
}
```

### TypeScript/Svelte

**Formatierung**:

```bash
cd frontend
npm run format

# Prettier
npx prettier --write .
```

**Linting**:

```bash
cd frontend
npm run lint

# ESLint
npx eslint .
```

**Type Checking**:

```bash
cd frontend
npm run check

# Svelte Check
npx svelte-check
```

**Qualitäts-Checks (alles zusammen)**:

```bash
make quality
```

**Code Style**:

- Verwende TypeScript für Type-Safety
- `interface` für Data Shapes
- `type` für Unions/Aliases
- Funktionale Components (keine Class Components)

**Beispiel**:

```typescript
// lib/stores/notes.ts
import { writable, type Writable } from 'svelte/store';

export interface Note {
  id: string;
  title: string;
  content: string;
  version: number;
}

export const notes: Writable<Note[]> = writable([]);

export async function loadNotes(): Promise<void> {
  const response = await fetch('/api/notes');
  const data = await response.json();
  notes.set(data.notes);
}
```

---

## Debugging

### Backend Debugging

**Mit Delve** (Go Debugger):

```bash
cd backend
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug ./cmd/server -- -addr :8080
```

**VSCode Launch Configuration** (`.vscode/launch.json`):

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Backend",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/backend/cmd/server",
      "args": ["-addr", ":8080"],
      "cwd": "${workspaceFolder}/backend"
    }
  ]
}
```

**Logging**:

Aktiviere Verbose Logging:

```go
// cmd/server/main.go
log.SetFlags(log.LstdFlags | log.Lshortfile)
```

**SQLite Debugging**:

```bash
# Öffne Datenbank
sqlite3 ./data/xelanote.db

# Inspect Tables
.tables
.schema notes
SELECT * FROM notes LIMIT 5;

# FTS5 Index
SELECT * FROM notes_fts WHERE notes_fts MATCH 'search query';
```

**Performance Profiling (pprof)**:

The backend includes an optional pprof server for performance profiling and diagnostics.

**Security:** Disabled by default. Must be explicitly enabled via `PPROF_ENABLED=true` environment variable.

**Enable pprof:**

```bash
cd backend
PPROF_ENABLED=true JWT_SECRET="your-secret" make run-backend
# Server logs: "pprof server available at http://127.0.0.1:6060/debug/pprof/"
```

**Access pprof (local development):**

```bash
# View available profiles
curl http://127.0.0.1:6060/debug/pprof/

# CPU profiling (30 seconds)
curl http://127.0.0.1:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -http=:8081 cpu.prof

# Heap profiling
curl http://127.0.0.1:6060/debug/pprof/heap > heap.prof
go tool pprof -http=:8081 heap.prof

# Goroutine dump
curl http://127.0.0.1:6060/debug/pprof/goroutine > goroutine.prof
go tool pprof -http=:8081 goroutine.prof
```

**Access pprof (remote server via SSH tunnel):**

```bash
# Create SSH tunnel (Hetzner example)
ssh -L 6060:127.0.0.1:6060 xelanote-prod

# In another terminal: access as if local
curl http://localhost:6060/debug/pprof/
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -http=:8081 cpu.prof
```

**Usage in production/staging:**

```bash
# Set environment variable
export PPROF_ENABLED=true

# Or via docker
docker run -e PPROF_ENABLED=true ...

# Note: pprof only listens on 127.0.0.1:6060 (localhost)
# Use SSH tunnel for remote access
```

**Common profiling tasks:**

```bash
# Find CPU hotspots
go tool pprof -http=:8081 cpu.prof
# Open browser → Flame Graph / Top functions

# Find memory leaks
go tool pprof -http=:8081 heap.prof
# Open browser → View → Inuse_space

# Find goroutine leaks
go tool pprof -http=:8081 goroutine.prof
# Look for unexpectedly high goroutine counts

# Compare profiles (before/after optimization)
go tool pprof -http=:8081 -base=before.prof after.prof
```

### Frontend Debugging

**Browser DevTools**:

- Chrome/Firefox DevTools (F12)
- Network Tab: API Requests inspizieren
- Console: JavaScript Errors

**Svelte DevTools**:

Installiere [Svelte DevTools Extension](https://github.com/sveltejs/svelte-devtools).

**Vite Debugging**:

```bash
cd frontend
DEBUG=vite:* npm run dev
```

**API Mocking**:

Für Frontend-Development ohne Backend:

```typescript
// lib/api.ts
const USE_MOCK = import.meta.env.DEV && false;

export async function getNotes() {
  if (USE_MOCK) {
    return { notes: mockNotes, next_cursor: '' };
  }
  return fetch('/api/notes').then(r => r.json());
}
```

---

## Häufige Probleme

### Problem: FTS5 Build-Tag fehlt

**Symptom**:

```
Error: no such table: notes_fts
```

Oder Volltextsuche funktioniert nicht.

**Ursache**: SQLite FTS5 Extension wurde nicht aktiviert.

**Lösung**:

Kompiliere mindestens mit dem `-tags "fts5"` Flag. `sqlite_crypt` ist optional fuer SQLCipher:

```bash
# Development
cd backend && go run -tags "fts5" ./cmd/server

# Build
CGO_ENABLED=1 go build -tags "fts5" -o xelanote ./cmd/server
```

**Warum?** Das `fts5` Build-Tag aktiviert die FTS5-Extension in der SQLite-Bibliothek (`github.com/mattn/go-sqlite3`). Ohne dieses Tag ist die Volltextsuche nicht verfügbar und die App kann nicht richtig funktionieren.

### Problem: `CGO_ENABLED=1` Build schlägt fehl

**Symptom**:

```
# github.com/mattn/go-sqlite3
exec: "gcc": executable file not found in $PATH
```

**Lösung**:

Installiere C Compiler:

```bash
# macOS
xcode-select --install

# Ubuntu/Debian
sudo apt-get install build-essential

# Alpine (Docker)
apk add gcc musl-dev
```

### Problem: Frontend Proxy funktioniert nicht

**Symptom**: API Requests gehen zu `http://localhost:5173/api/notes` statt Backend.

**Lösung**:

Prüfe `vite.config.ts`:

```typescript
export default {
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
}
```

**Restart** Vite Dev Server nach Config-Änderung.

### Problem: SQLite "database is locked"

**Symptom**:

```
Error: database is locked
```

**Ursachen**:

- Mehrere Writer gleichzeitig
- Lange Transaktionen
- WAL-Modus nicht aktiviert

**Lösung**:

1. Prüfe WAL-Modus:

   ```bash
   sqlite3 ./data/xelanote.db "PRAGMA journal_mode;"
   # Sollte "wal" ausgeben
   ```

2. Falls nicht WAL:

   ```bash
   sqlite3 ./data/xelanote.db "PRAGMA journal_mode=WAL;"
   ```

3. Code-Fix: Verwende kürzere Transaktionen

### Problem: Frontend Build-Fehler (Vite 6 / Tailwind v4)

**Symptom**:

```
Error: Cannot resolve @tailwindcss/postcss
```

Oder:

```
Error: Plugin version mismatch - @sveltejs/vite-plugin-svelte
```

**Ursache**: Tailwind CSS v4 hat neue Dependencies und `@sveltejs/vite-plugin-svelte` muss v5+ sein für Vite 6.

**Lösung**:

```bash
cd frontend
npm install @tailwindcss/postcss@latest @sveltejs/vite-plugin-svelte@^5.0.0
```

**Wichtige Dependencies für Vite 6 + Tailwind v4**:
- `vite`: ^6.0.0
- `@sveltejs/vite-plugin-svelte`: ^5.0.0 (nicht v4!)
- `@tailwindcss/postcss`: latest (für Tailwind v4)
- `tailwindcss`: ^4.0.0

**Warum?** Tailwind CSS v4 verwendet ein neues PostCSS-Plugin (`@tailwindcss/postcss`) statt der alten `tailwindcss`-Integration. Die SvelteKit Vite-Plugin-Version muss außerdem mit Vite 6 kompatibel sein.

### Problem: Frontend Build-Fehler (Allgemein)

**Symptom**:

```
Error: Cannot find module '@sveltejs/adapter-static'
```

**Lösung**:

```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
```

### Problem: Port bereits belegt

**Symptom**:

```
Error: listen tcp :8080: bind: address already in use
```

**Lösung**:

1. Finde Prozess:

   ```bash
   # macOS/Linux
   lsof -i :8080

   # Windows
   netstat -ano | findstr :8080
   ```

2. Kill Prozess oder verwende anderen Port:

   ```bash
   CGO_ENABLED=1 go run -tags "fts5" ./cmd/server -addr :8081
   ```

---

## Contributing

### Workflow

1. **Fork** das Repository
2. **Branch** erstellen: `git checkout -b feature/my-feature`
3. **Änderungen** committen: `git commit -m "Add feature X"`
4. **Tests** ausführen: `make test`
5. **Code formatieren**: `make fmt`
6. **Push** zu Fork: `git push origin feature/my-feature`
7. **Pull Request** erstellen

### Commit Messages

**Format**:

```
<type>: <subject>

<body>

<footer>
```

**Types**:

- `feat`: Neues Feature
- `fix`: Bugfix
- `docs`: Dokumentation
- `style`: Formatierung (keine Code-Änderung)
- `refactor`: Code-Refactoring
- `test`: Tests hinzufügen/ändern
- `chore`: Build, Dependencies

**Beispiel**:

```
feat: Add folder-based navigation

- Add GET /api/folders endpoint
- Update Sidebar component to show folder hierarchy
- Add folder filter to notes list

Closes #42
```

### Code Review Checkliste

- [ ] Tests schlagen nicht fehl (`make test`)
- [ ] Code ist formatiert (`make fmt`, `npm run format`)
- [ ] Keine Linter-Warnings (`make lint`)
- [ ] Neue Features haben Tests
- [ ] API-Änderungen sind dokumentiert
- [ ] Breaking Changes sind im Commit erwähnt
- [ ] Documentation ist aktualisiert

## Version History Feature

Xelanote enthält ein automatisches Version History System, das Snapshots von Notizen speichert und Zeitreisen ermöglicht.

### Funktionsweise

**Automatische Snapshots**:
- Snapshots werden erstellt wenn Content ODER Title einer Notiz geändert wird
- **Zeitbasierte Throttling**: Snapshot nur wenn >5 Minuten seit letztem Snapshot vergangen sind
- **Keine Snapshots für reine Umbenennungen**: Der `/api/notes/:id/rename` Endpoint triggert bewusst KEINE Snapshots

**Retention Policy**:
- Maximale 30 Versionen pro Notiz
- Täglicher Cleanup-Job (läuft beim Server-Start)
- Älteste Versionen werden zuerst gelöscht (FIFO)

**Non-Destructive Restore**:
- Vor jedem Restore wird die aktuelle Version als Snapshot gespeichert
- Versehentliche Restores können rückgängig gemacht werden
- Version wird nach Restore inkrementiert

### Datenbank-Schema

Migration `011_note_versions.sql` erstellt die `note_versions` Tabelle:

```sql
CREATE TABLE note_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    version INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    snapshot_at TEXT NOT NULL,
    FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

**Indizes**:
- `idx_note_versions_note_id`: Für schnelle Version-Abfragen
- `idx_note_versions_user_id`: Für User-spezifische Queries
- `idx_note_versions_snapshot_at`: Für zeitbasierte Sortierung
- `idx_note_versions_note_user`: Composite Index für Pagination

### API Endpoints

Siehe [API Dokumentation](api.md) für Details:

- `GET /api/notes/:id/versions` - Liste aller Versionen (paginiert)
- `GET /api/notes/:id/versions/:version` - Spezifische Version abrufen
- `GET /api/notes/:id/versions/compare?v1=X&v2=Y` - Zwei Versionen vergleichen
- `POST /api/notes/:id/versions/:version/restore` - Auf Version zurücksetzen

### Frontend-Komponenten

**VersionHistoryDialog.svelte**:
- Linke Sidebar: Timeline der Versionen + "Aktuell" (current state)
- Rechte Panel: Preview oder Diff-Ansicht
- Compare Mode: Line-Diff zwischen beliebigen zwei Versionen
- Restore Button mit Bestätigungsdialog

**Diff-Berechnung**:
- Client-seitig mit `diff` npm Package
- Line-by-Line Vergleich
- Highlighting für Added/Removed/Changed Lines

### Code-Locations

**Backend**:
- `backend/internal/api/versions.go` - API Endpoints
- `backend/internal/db/versions.go` - Database Operations
- `backend/internal/db/migrations/011_note_versions.sql` - Schema
- `backend/internal/service/notes.go` - Snapshot Logic

**Frontend**:
- `frontend/src/lib/components/VersionHistoryDialog.svelte` - UI Component
- `frontend/src/lib/api/versions.ts` - API Client Functions (Versionen)

### Testing Version History

**Manual Testing**:

1. Erstelle eine Notiz
2. Warte >5 Minuten
3. Ändere Content oder Title
4. Öffne Editor → Clock Icon
5. Verifiziere dass Snapshot erstellt wurde

**Snapshot Debugging**:

```bash
sqlite3 ./data/xelanote.db "SELECT version, title, snapshot_at FROM note_versions WHERE note_id = 'YOUR_NOTE_ID' ORDER BY version DESC;"
```

**Cleanup Testing**:

```bash
# Insert 35 versions for a note (exceeds retention limit)
# Wait for daily cleanup job
# Verify only 30 versions remain
sqlite3 ./data/xelanote.db "SELECT COUNT(*) FROM note_versions WHERE note_id = 'YOUR_NOTE_ID';"
```

### Encryption Support

**Status**: ✅ Fully supported as of 2026-01-22 (Commit `2804320`)

Version history fully supports encrypted notes with automatic client-side decryption:

**How it works**:
- Backend stores encrypted versions with `encrypted_content`, `wrapped_dek`, and encryption metadata
- Frontend `NoteVersion` interface includes all encryption fields
- `VersionHistoryDialog` automatically decrypts versions when loaded
- Decryption uses the same encryption store as the note editor

**User Experience**:
- Encrypted versions display decrypted content in history dialog
- Version comparison works for encrypted notes
- If encryption is locked (not logged in), shows "Unlock encryption first" message
- Seamless experience - users don't need to think about encryption

**Technical Implementation**:
- `decryptVersion()` function in `VersionHistoryDialog.svelte`
- Decryption triggered in `loadVersions()` and `loadMore()`
- Uses `encryptionStore.decrypt()` with version's wrapped DEK
- Graceful fallback when encryption is unavailable

**Files**:
- `frontend/src/lib/api.ts` - Extended `NoteVersion` interface with encryption fields
- `frontend/src/lib/components/VersionHistoryDialog.svelte` - Decryption logic

**Known Limitation**:
- Version snapshots for encrypted notes store full ciphertext (not encrypted diffs)
- This is by design for security - avoids potential plaintext leakage via diff patterns

### Performance Considerations

**Snapshot Creation**:
- Lightweight: Nur Title + Content werden kopiert
- Keine Links/Attachments werden gesnapshot
- ~1-2ms für typische Notiz (<10KB)

**Version Queries**:
- Cursor-basierte Pagination (wie Notes)
- Index auf `(note_id, user_id, version DESC)` für effiziente Queries
- Typisch <5ms für 50 Versionen

**Storage Impact**:
- Pro Notiz max 30 Versionen
- Bei 10KB Durchschnittsgröße: ~300KB pro Notiz
- 1000 Notizen mit voller History: ~300MB

### Zukunft: Mögliche Erweiterungen

- Automatische Snapshots bei bestimmten Events (z.B. vor Bulk-Operationen)
- Diff-Patches speichern statt Full Content (Storage-Optimierung)
- User-konfigurierbare Retention Policy
- Snapshot-Kommentare (warum wurde diese Version gespeichert?)

---

## Text Wrapping Feature

Xelanote implementiert automatisches Text-Wrapping für optimale Lesbarkeit auf mobilen Geräten, während Desktop-Nutzer nicht beeinträchtigt werden.

### Überblick

Das Text-Wrapping Feature behebt horizontales Scrollen auf mobilen Geräten durch intelligente Textumbruch-Strategien für verschiedene Content-Typen.

### Implementierungs-Details

#### CodeMirror Editor Wrapping

**Location**: `frontend/src/lib/editor/codemirror.ts:189`

```typescript
extensions: [
  EditorView.lineWrapping,  // Automatischer Zeilenumbruch
  // ... weitere Extensions
]
```

**Verhalten**:
- Aktiviert automatischen Zeilenumbruch im Editor
- Funktioniert auf Mobile und Desktop
- Erhält Syntax-Highlighting und Editor-Features

#### Markdown Preview Wrapping

**Location**: `frontend/src/app.css`

**Base Wrapping** (Zeile 365):
```css
.markdown-preview {
  overflow-wrap: anywhere;     /* Modern browsers */
  word-wrap: break-word;       /* Fallback für ältere Browser */
  word-break: normal;          /* Normale Wörter bleiben intakt */
}
```

**URL Wrapping** (Zeile 400):
```css
.markdown-preview a {
  word-break: break-all;       /* URLs brechen aggressiv um */
  overflow-wrap: anywhere;
}
```

**Inline Code Wrapping** (Zeile 418):
```css
.markdown-preview :not(pre) > code {
  white-space: pre-wrap;       /* Wrapping + Formatierung erhalten */
  overflow-wrap: break-word;
}
```

**Heading Wrapping** (Zeile 396):
```css
.markdown-preview h1, h2, h3, h4, h5, h6 {
  word-break: break-word;
  overflow-wrap: anywhere;
}
```

**Table Wrapping** (Zeile 478):
```css
.markdown-preview table {
  word-break: break-word;
  /* Vorbereitet für zukünftigen Table-Support */
}
```

**Editor CSS Fallback** (Zeile 314):
```css
.cm-editor .cm-content {
  white-space: pre-wrap;       /* Fallback für Browser ohne EditorView.lineWrapping */
}
```

### Content-Type Strategie

| Content-Type | Wrapping-Strategie | Grund |
|--------------|-------------------|-------|
| **Normal Text** | `overflow-wrap: anywhere` | Natürlicher Umbruch an Wortgrenzen |
| **URLs** | `word-break: break-all` | Lange URLs müssen aggressiv umbrechen |
| **Inline Code** | `white-space: pre-wrap` | Formatierung beibehalten, aber umbrechen |
| **Code Blocks** | Horizontal Scroll | Code-Struktur muss erhalten bleiben |
| **Headings** | `word-break: break-word` | Lange Titel müssen umbrechen |
| **Tables** | `word-break: break-word` | Zelleninhalte umbrechen (zukünftig) |

### Code Blocks: Bewusste Ausnahme

**Wichtig**: `<pre>` Elemente (Code Blocks) behalten **absichtlich** horizontales Scrollen:

```css
.markdown-preview pre {
  /* KEIN word-wrap, KEIN word-break */
  overflow-x: auto;  /* Horizontal Scroll erlaubt */
}
```

**Warum?**
- Code-Struktur ist semantisch wichtig
- Einrückung muss sichtbar bleiben
- Zeilenumbruch würde Code unleserlich machen

**Selector `:not(pre) > code`**: Schützt Code-Block-Code vor Wrapping, erlaubt aber Inline-Code-Wrapping.

### Browser-Kompatibilität

**Moderne Browser** (Chrome 80+, Firefox 75+, Safari 14+):
- `overflow-wrap: anywhere` (bevorzugt)

**Ältere Browser** (Safari < 14, Chromium < 80):
- `word-wrap: break-word` (Fallback)
- `word-break: break-word` (Fallback für Headings)

**Mobile Browser**:
- Getestet auf iOS Safari, Chrome Mobile, Firefox Mobile
- Funktioniert auf allen modernen Mobile Browsers

### Testing Text Wrapping

**Manual Testing**:

1. **Long URLs Test**:
   ```markdown
   [Very Long URL](https://example.com/very/long/path/that/should/wrap/on/mobile/devices/without/horizontal/scrolling)
   ```
   - Verifiziere: URL umbricht auf Mobile, kein horizontales Scrollen

2. **Long Heading Test**:
   ```markdown
   # This is a Very Long Heading That Should Wrap on Mobile Devices Without Causing Horizontal Scrolling
   ```
   - Verifiziere: Heading umbricht über mehrere Zeilen

3. **Inline Code Test**:
   ```markdown
   Use `this_is_a_very_long_inline_code_snippet_that_should_wrap_on_mobile_devices` in your code.
   ```
   - Verifiziere: Inline Code umbricht, behält aber Monospace-Font

4. **Code Block Test**:
   ```markdown
   ```javascript
   function veryLongFunctionNameThatShouldNotWrapButScrollHorizontally() {
     return "This line should cause horizontal scrolling on mobile";
   }
   ```
   ```
   - Verifiziere: Code Block scrollt horizontal, bricht NICHT um

5. **Mixed Content Test**:
   - Erstelle Note mit Mix aus Text, URLs, Headings, Code
   - Teste auf verschiedenen Mobile Devices (Portrait + Landscape)

**Responsive Testing**:

```bash
# Chrome DevTools
# 1. Toggle Device Toolbar (Cmd+Shift+M)
# 2. Teste verschiedene Devices:
#    - iPhone SE (375px width)
#    - iPhone 12 Pro (390px width)
#    - Pixel 5 (393px width)
#    - iPad Mini (768px width)
# 3. Rotiere zwischen Portrait/Landscape
```

**Edge Cases**:

- **Sehr lange Wörter ohne Trennzeichen**: `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`
  - Sollte mit `overflow-wrap: anywhere` umbrechen
- **Mathematische Formeln**: `f(x)=y+z+a+b+c+d+e+f+g+h+i+j+k+l+m+n+o+p+q+r+s+t+u+v+w+x+y+z`
  - Sollte an Operatoren umbrechen
- **Emojis in langen Strings**: `🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥`
  - Sollte ohne Fehler umbrechen

### Performance Considerations

**CSS Performance**:
- Wrapping-Regeln sind native CSS → Kein JavaScript overhead
- Browser-native Textumbruch-Engine (optimal performant)
- Kein Layout Thrashing

**Rendering**:
- Kein zusätzlicher Reflow beim Laden
- CSS-Properties sind nicht-animiert (kein GPU-Stress)
- Selector-Spezifität ist niedrig (schnelles Matching)

**Memory**:
- Keine zusätzlichen DOM-Elemente
- Keine JavaScript-basierte Wrapping-Library erforderlich

### Desktop Experience

**Wichtig**: Das Text-Wrapping Feature beeinträchtigt Desktop **nicht**:

- Desktop hat ausreichend Viewport-Breite
- Natürlicher Zeilenumbruch erfolgt selten
- Code-Blöcke scrollen horizontal wie gewohnt
- UX bleibt identisch zu vorher

### Accessibility Considerations

**Screen Reader**:
- Text-Wrapping ändert semantische Struktur **nicht**
- Screen Reader lesen Content unverändert
- Keine zusätzlichen ARIA-Attribute erforderlich

**Keyboard Navigation**:
- Tab-Reihenfolge bleibt gleich
- Cursor-Navigation in CodeMirror unverändert

**Zoom**:
- Text-Wrapping funktioniert korrekt bei Browser-Zoom
- Keine Überlappungen oder Clipping

### Zukunft: Mögliche Erweiterungen

**User Preferences**:
- Toggle für Editor-Wrapping (ON/OFF)
- Separate Einstellung für Preview-Wrapping
- Persistenz in LocalStorage

**Smart Wrapping**:
- Programmiersprachen-spezifische Wrap-Punkte
- Natural Language Processing für optimale Umbruchpunkte

**Advanced Code Block Handling**:
- Optional: Soft-Wrap für Code mit Visual Indicator
- Syntax-aware Wrapping (z.B. nach Operatoren)

---

## Mobile Sidebar Feature

Xelanote verfügt über eine responsive Sidebar, die sich auf mobilen Geräten als Overlay-Drawer verhält und auf Desktop-Geräten als normale Sidebar.

### Responsive Breakpoint

**Breakpoint**: `768px`

- **Mobile** (< 768px): Sidebar wird als Overlay-Drawer angezeigt
- **Desktop** (≥ 768px): Normale Inline-Sidebar mit Collapse-Funktion (256px ↔ 48px)

### Mobile Verhalten

**Drawer-Modus**:
- Sidebar gleitet von links über den Content (Overlay)
- Halbtransparenter Backdrop hinter dem Drawer
- Z-Index Hierarchie:
  - Backdrop: `z-40`
  - Drawer (Sidebar): `z-50`
  - QuickSwitcher/Dialogs: `z-50` (erscheint über Drawer durch DOM-Reihenfolge)

**Drawer Öffnen**:
- Hamburger-Menü in der `MobileHeader` Komponente
- Icon: `☰` (drei horizontale Linien)

**Drawer Schließen**:
- Klick auf Backdrop
- Escape-Taste drücken
- Notiz in der Sidebar auswählen
- Suche abschicken
- Graph/Trash Buttons klicken

**QuickSwitcher Verhalten**:
- Bleibt geöffnet auch wenn Drawer geschlossen wird
- Erscheint über dem Drawer (gleicher z-index, aber später im DOM)

### Desktop Verhalten

Auf Desktop-Geräten bleibt die Sidebar-Funktionalität **unverändert**:

- Inline-Sidebar (kein Overlay)
- Collapse/Expand Funktion (256px ↔ 48px)
- Kein Backdrop
- Kein Hamburger-Menü

### Responsive Resize-Verhalten

**Initial Load**:
- Mobile: Drawer startet geschlossen
- Desktop: Sidebar startet geöffnet

**Resize Events**:
- Desktop → Mobile: Drawer wird automatisch geschlossen
- Mobile → Desktop: Sidebar wird automatisch geöffnet

### Implementierungs-Details

#### UI Store (`frontend/src/lib/stores/ui.svelte.ts`)

**State Management**:

```typescript
class UIState {
  isMobile = $state(false);
  sidebarOpen = $state(false);

  getIsMobile(): boolean { return this.isMobile; }
  setIsMobile(value: boolean): void { this.isMobile = value; }

  closeSidebarOnMobile(): void {
    if (this.isMobile) {
      this.sidebarOpen = false;
    }
  }
}
```

**Verwendung**:
- `isMobile`: Wird via Resize-Listener in `+layout.svelte` aktualisiert
- `closeSidebarOnMobile()`: Wird von Komponenten aufgerufen, die den Drawer schließen sollen

#### MobileHeader (`frontend/src/lib/components/MobileHeader.svelte`)

Neue Komponente mit Hamburger-Menü:

```svelte
<button on:click={() => ui.sidebarOpen = !ui.sidebarOpen}>
  ☰
</button>
```

**Positionierung**:
- Fixed am oberen Bildschirmrand
- Nur auf Mobile sichtbar (< 768px)
- Z-Index: höher als Content, niedriger als Drawer

#### Layout (`frontend/src/routes/+layout.svelte`)

**Resize Listener**:

```typescript
function updateIsMobile() {
  const mobile = window.innerWidth < 768;
  ui.setIsMobile(mobile);

  if (mobile) {
    ui.sidebarOpen = false;  // Close drawer on mobile
  } else {
    ui.sidebarOpen = true;   // Open sidebar on desktop
  }
}

onMount(() => {
  updateIsMobile();
  window.addEventListener('resize', updateIsMobile);
});
```

**Backdrop Overlay**:

```svelte
{#if ui.isMobile && ui.sidebarOpen}
  <div class="fixed inset-0 bg-black/50 z-40"
       on:click={() => ui.sidebarOpen = false} />
{/if}
```

#### Sidebar (`frontend/src/lib/components/Sidebar.svelte`)

**Drawer CSS** (Conditional Styling):

```svelte
<aside class:drawer={ui.isMobile} class:open={ui.sidebarOpen}>
  <!-- Content -->
</aside>
```

```css
.drawer {
  position: fixed;
  left: 0;
  top: 0;
  height: 100vh;
  z-index: 50;
  transform: translateX(-100%);
  transition: transform 0.3s ease;
}

.drawer.open {
  transform: translateX(0);
}
```

**Escape Handler**:

```typescript
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && ui.isMobile && ui.sidebarOpen) {
    ui.sidebarOpen = false;
  }
}
```

#### UnifiedTree (`frontend/src/lib/components/UnifiedTree.svelte`)

**Auto-Close on Note Selection**:

```typescript
async function handleNoteClick(note: Note) {
  ui.closeSidebarOnMobile();
  await goto(`/notes/${note.id}`);
}
```

**Auto-Close on Search Submit**:

```typescript
function handleSearchSubmit() {
  ui.closeSidebarOnMobile();
  // ... search logic
}
```

**Auto-Close on Button Clicks** (Graph/Trash):

```typescript
function handleGraphClick() {
  ui.closeSidebarOnMobile();
  // ... graph logic
}

function handleTrashClick() {
  ui.closeSidebarOnMobile();
  // ... trash logic
}
```

### Z-Index Hierarchie

**Wichtig**: Komponenten mit gleichem Z-Index werden nach DOM-Reihenfolge gestapelt.

```
z-10:  Normal Content
z-40:  Backdrop
z-50:  Drawer (Sidebar)
z-50:  QuickSwitcher (erscheint über Drawer, da später im DOM)
z-50:  Dialogs (Version History, etc.)
```

**DOM-Reihenfolge in `+layout.svelte`**:

```svelte
<MobileHeader />         <!-- z-30 -->
<Sidebar />              <!-- z-50 (drawer mode) -->
<main>...</main>         <!-- z-10 -->
<QuickSwitcher />        <!-- z-50 (rendered last, appears on top) -->
{#if backdrop}...{/if}   <!-- z-40 -->
```

### Testing Mobile Sidebar

**Manual Testing**:

1. **Mobile Modus aktivieren**:
   - Chrome DevTools → Toggle Device Toolbar (Cmd+Shift+M)
   - Wähle Device mit < 768px Breite (z.B. iPhone)

2. **Drawer öffnen**:
   - Klicke Hamburger-Menü
   - Verifiziere: Drawer gleitet von links ein, Backdrop erscheint

3. **Drawer schließen**:
   - Klicke Backdrop → Drawer schließt
   - Drücke Escape → Drawer schließt
   - Klicke Notiz → Drawer schließt + Navigation

4. **Resize-Verhalten**:
   - Öffne Drawer auf Mobile
   - Resize zu Desktop (≥ 768px)
   - Verifiziere: Drawer verschwindet, normale Sidebar erscheint

5. **QuickSwitcher**:
   - Öffne Drawer
   - Drücke Cmd+K (QuickSwitcher)
   - Verifiziere: QuickSwitcher erscheint über Drawer
   - Verifiziere: Drawer bleibt geöffnet

**Responsive Breakpoint Testing**:

```bash
# Test verschiedene Viewport-Größen
# Mobile:  375px, 428px, 768px (Grenze)
# Desktop: 769px, 1024px, 1440px
```

**CSS Debugging**:

```css
/* Temporär hinzufügen für Debugging */
.drawer { border: 2px solid red; }
.backdrop { border: 2px solid blue; }
```

### Performance Considerations

**CSS Transitions**:
- Drawer Slide Animation: `transform 0.3s ease`
- Hardware-beschleunigt via `translateX()` (keine `left` Animation)
- ~60fps auf modernen Mobile Devices

**Event Listeners**:
- Resize Listener: Throttled (max 1x pro 100ms)
- Backdrop Click: Passive Event Listener

**DOM Updates**:
- Conditional Rendering für Backdrop (`{#if}`)
- CSS Classes statt Inline-Styles für besseres Caching

### Zukunft: Mögliche Erweiterungen

**Touch Gestures**:
- Swipe-to-Open von linkem Bildschirmrand
- Swipe-to-Close nach links

**Persistent State**:
- Sidebar-Status in localStorage speichern
- User-Präferenz für Default-Verhalten

**Animations**:
- Backdrop Fade-In Animation
- Spring-basierte Transitions (anstatt linear)

**Accessibility**:
- Focus Trap im geöffneten Drawer
- ARIA Labels für Screen Reader
- Touch-Target-Größe für Hamburger-Menü (min 44x44px)

---

## Graph View Mobile Touch Interaction

Die Graph View unterstützt auf mobilen Geräten (Touch-Displays) eine optimierte Interaktion mit Nodes, da das "Hovern" mit der Maus auf Touch-Geräten nicht möglich ist.

### Problem auf Mobile

**Desktop-Verhalten**:
- Mit der Maus über einen Node fahren → Tooltip mit Node-Titel erscheint
- Auf Node klicken → Notiz wird geöffnet

**Mobile-Problem**:
- Kein Hover-Event auf Touch-Geräten
- Benutzer können nicht sehen, welche Notiz ein Node repräsentiert, bevor sie ihn öffnen

### Lösung: Tap-to-Show, Tap-Again-to-Open

**Interaktionsmuster**:

1. **Erster Tap auf Node**:
   - Zeigt ein Tooltip mit dem Node-Titel
   - Gibt Hinweis "Tippe nochmal zum Öffnen"
   - Bei nicht aufgelösten Verknüpfungen: "Nicht aufgelöste Verknüpfung"

2. **Zweiter Tap auf denselben Node**:
   - Öffnet die Notiz (nur bei resolved Nodes)
   - Tooltip verschwindet

3. **Tap auf anderen Node**:
   - Wechselt den ausgewählten Node
   - Zeigt Tooltip für den neuen Node

4. **Tap auf leere Fläche**:
   - Schließt das Tooltip
   - Hebt Auswahl auf

### Implementierungs-Details

#### State Management

`frontend/src/lib/components/GraphCanvas.svelte`:

```typescript
let selectedNode = $state<any>(null);
let tooltipPosition = $state<{ x: number; y: number }>({ x: 0, y: 0 });
```

**selectedNode**:
- Speichert den aktuell ausgewählten Node
- `null` wenn kein Node ausgewählt

**tooltipPosition**:
- X/Y Koordinaten des Tap-Events
- Wird für Tooltip-Positionierung verwendet

#### Click Handler Logic

```typescript
function handleNodeClick(node: any, event: MouseEvent) {
  // Check if this is the already selected node
  if (selectedNode && selectedNode.id === node.id) {
    // Second tap - navigate to note if it's a resolved node
    if (node.is_resolved && !node.id.startsWith('unresolved:')) {
      goto(`/note/${node.id}`);
    }
    selectedNode = null;
  } else {
    // First tap - show tooltip
    selectedNode = node;
    tooltipPosition = {
      x: event.clientX,
      y: event.clientY
    };
  }
}
```

**Logik**:
- Vergleicht geklickten Node mit `selectedNode`
- Gleicher Node → Navigation oder Schließen
- Anderer Node → Tooltip anzeigen

#### Background Click Handler

```typescript
.onBackgroundClick(() => {
  // Close tooltip when clicking on empty space
  selectedNode = null;
})
```

Schließt das Tooltip beim Klick auf leere Fläche im Graph.

#### Tooltip UI Component

```svelte
{#if selectedNode}
  <div
    class="absolute z-10 bg-popover text-popover-foreground border rounded-lg shadow-lg px-4 py-3 max-w-xs pointer-events-none"
    style="left: {tooltipPosition.x}px; top: {tooltipPosition.y}px; transform: translate(-50%, -100%) translateY(-8px);"
  >
    <div class="font-medium">{selectedNode.title}</div>
    <div class="text-xs text-muted-foreground mt-1">
      {selectedNode.is_resolved ? 'Tippe nochmal zum Öffnen' : 'Nicht aufgelöste Verknüpfung'}
    </div>
  </div>
{/if}
```

**Styling**:
- Positioniert über dem Tap-Punkt (`transform: translate(-50%, -100%)`)
- Kleiner Offset nach oben (`translateY(-8px)`)
- `pointer-events-none`: Tooltip blockiert keine Touch-Events

### Cross-Platform Kompatibilität

Die Implementierung funktioniert **sowohl auf Mobile als auch Desktop**:

**Desktop (Maus)**:
- Hover zeigt weiterhin natives Tooltip (via `force-graph` Library)
- Click zeigt zusätzliches Tooltip
- Zweiter Click öffnet Notiz

**Mobile (Touch)**:
- Kein Hover verfügbar
- Tap zeigt Tooltip
- Zweiter Tap öffnet Notiz

**Warum beide Tooltips auf Desktop?**
- Das native Hover-Tooltip verschwindet beim Click
- Unser Custom-Tooltip bleibt nach dem Click sichtbar
- Gibt Feedback dass der Node ausgewählt ist

### Testing Mobile Touch Interaction

**Manual Testing**:

1. **Öffne Graph View auf Mobile Device**:
   - Chrome DevTools → Toggle Device Toolbar (Cmd+Shift+M)
   - Wähle Mobile Device (z.B. iPhone)
   - Navigiere zu `/graph`

2. **First Tap Test**:
   - Tap auf einen Node
   - Verifiziere: Tooltip erscheint über dem Node
   - Verifiziere: Titel und Hinweis sind sichtbar

3. **Second Tap Test**:
   - Tap nochmal auf denselben Node
   - Verifiziere: Navigation zu Notiz
   - Verifiziere: Tooltip verschwindet

4. **Node Switch Test**:
   - Tap auf Node A → Tooltip erscheint
   - Tap auf Node B → Tooltip wechselt zu Node B
   - Verifiziere: Kein doppeltes Tooltip

5. **Background Click Test**:
   - Tap auf Node → Tooltip erscheint
   - Tap auf leere Fläche → Tooltip verschwindet

6. **Unresolved Node Test**:
   - Tap auf nicht aufgelösten Node (rot)
   - Verifiziere: Tooltip zeigt "Nicht aufgelöste Verknüpfung"
   - Verifiziere: Zweiter Tap öffnet KEINE Notiz

**Desktop Testing**:

1. Öffne `/graph` auf Desktop
2. Hover über Node → Natives Tooltip erscheint
3. Click auf Node → Custom Tooltip erscheint
4. Click nochmal → Navigation zur Notiz

### Code Locations

**Frontend**:
- `frontend/src/lib/components/GraphCanvas.svelte:12-13` - State Management
- `frontend/src/lib/components/GraphCanvas.svelte:72-89` - Click Handler
- `frontend/src/lib/components/GraphCanvas.svelte:95-104` - Tooltip UI

**Dependencies**:
- `force-graph` Library für Graph Rendering
- Svelte 5 Runes (`$state`) für Reactive State

### Performance Considerations

**Event Handling**:
- Click Events werden direkt von `force-graph` Library gehandled
- Kein zusätzlicher Event Listener erforderlich
- ~1ms Latency für State Update

**Rendering**:
- Tooltip wird nur bei Bedarf gerendert (`{#if selectedNode}`)
- CSS Transitions für smooth Animation
- `pointer-events-none` verhindert Z-Index Probleme

**Memory**:
- `selectedNode` speichert nur Referenz zum Node (keine Kopie)
- `tooltipPosition` ist nur 2 Numbers (~16 bytes)
- Minimaler Memory Overhead

### Accessibility Considerations

**Touch Target Size**:
- Nodes haben ausreichende Größe für Touch (min ~44x44px)
- Definiert via `.nodeRelSize(6)` in force-graph Config

**Screen Reader**:
- Aktuell keine ARIA Labels (Graph ist rein visuell)
- Zukunft: Keyboard Navigation + ARIA Labels

**Contrast**:
- Tooltip verwendet Theme-Colors (`bg-popover`, `text-popover-foreground`)
- Dark Mode kompatibel

### Zukunft: Mögliche Erweiterungen

**Long Press Detection**:
- Alternative zu Tap: Long Press zeigt Tooltip + Context Menu
- Erfordert Touch Event Handling (nicht via force-graph)

**Gesture Support**:
- Pinch-to-Zoom für Graph
- Pan-Gesten für Navigation
- Erfordert Custom Event Handler

**Persistent Selection**:
- Ausgewählter Node bleibt highlighted bis Deselect
- Visual Feedback welcher Node gerade ausgewählt ist

**Tooltip Improvements**:
- Automatische Positionierung (verhindert Viewport Overflow)
- Zeige zusätzliche Infos (Anzahl Links, Last Modified)
- Animation beim Erscheinen (Fade-In, Scale)

---

### Areas for Contribution

**High Priority**:

- Graph View für Note-Netzwerk
- Tag System (Schema existiert bereits)
- Performance Optimierungen (Caching)
- E2E Tests (Playwright)

**Good First Issues**:

- UI Verbesserungen (CSS, Dark Mode Tweaks)
- Parser Edge Cases
- Error Messages verbessern
- Keyboard Shortcuts erweitern

**Long-term**:

- Plugin System (Lua/WASM)
- Real-time Collaboration (WebSockets)
- Mobile App (React Native)
- Git Sync Integration

---

## Development Tools

### Empfohlene Editor-Setup

**VSCode**:

Extensions:

- Go (golang.go)
- Svelte for VS Code (svelte.svelte-vscode)
- Prettier (esbenp.prettier-vscode)
- ESLint (dbaeumer.vscode-eslint)
- Tailwind CSS IntelliSense

**Settings** (`.vscode/settings.json`):

```json
{
  "go.formatTool": "gofmt",
  "go.lintTool": "golangci-lint",
  "editor.formatOnSave": true,
  "[svelte]": {
    "editor.defaultFormatter": "svelte.svelte-vscode"
  },
  "[typescript]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  }
}
```

**GoLand/IntelliJ**:

- Svelte Plugin
- Go Plugin (integriert)

**Vim/Neovim**:

- vim-go
- ale (Linter)
- coc-svelte

### Useful Commands

```bash
# Backend
make run-backend     # Start Backend Dev Server
make test            # Run all tests
make test-parser     # Parser tests only
make fmt             # Format Go code
make lint            # Lint Go code

# Frontend
make run-frontend    # Start Frontend Dev Server
cd frontend && npm run dev    # Same
cd frontend && npm run build  # Build for production

# Combined
make build           # Full build (Frontend + Backend)
make docker          # Build Docker image
make clean           # Remove all build artifacts
```

### Database Tools

**SQLite Browser**:

- [DB Browser for SQLite](https://sqlitebrowser.org/) (GUI)

**CLI**:

```bash
# Interactive shell
sqlite3 ./data/xelanote.db

# Execute query
sqlite3 ./data/xelanote.db "SELECT * FROM notes LIMIT 5;"

# Dump schema
sqlite3 ./data/xelanote.db .schema > schema_dump.sql

# Backup
sqlite3 ./data/xelanote.db ".backup backup.db"
```

---

## Performance Profiling

### Backend Profiling

**CPU Profile**:

```bash
cd backend
go test -cpuprofile=cpu.prof -bench=. ./internal/parser/...
go tool pprof cpu.prof
```

**Memory Profile**:

```bash
go test -memprofile=mem.prof -bench=. ./internal/parser/...
go tool pprof mem.prof
```

**Interactive Mode**:

```
(pprof) top 10
(pprof) list ParseWikiLink
(pprof) web
```

### Frontend Profiling

**Chrome DevTools**:

1. Performance Tab → Record
2. Interact mit App
3. Stop Recording
4. Analyze Flame Graph

**Lighthouse**:

```bash
npm install -g lighthouse
lighthouse http://localhost:5173 --view
```

---

## Weitere Ressourcen

- [Go Documentation](https://go.dev/doc/) - Offizielle Go Docs
- [SvelteKit Docs](https://kit.svelte.dev/docs) - SvelteKit Guide
- [SQLite FTS5](https://www.sqlite.org/fts5.html) - Full-Text Search
- [CodeMirror 6](https://codemirror.net/docs/) - Editor Documentation
- [Chi Router](https://github.com/go-chi/chi) - HTTP Router
- [Tailwind CSS v4](https://tailwindcss.com/docs) - Styling

---

## Performance-Optimierungen

XelaNote enthaelt Code-Splitting, Virtual Scrolling, PWA/Offline-Guard und WebSocket-Updates (Details und Belege unten).

### Übersicht der implementierten Optimierungen

| Feature | Status | Beleg (Repo) |
|---------|--------|--------------|
| Code Splitting/Lazy Loading | implementiert | `frontend/vite.config.ts`, `frontend/src/lib/editor/codemirror.ts`, `frontend/src/lib/components/GraphCanvas.svelte` |
| Virtual Scrolling | teilweise (Suche/Trash) | `frontend/src/routes/search/+page.svelte`, `frontend/src/routes/trash/+page.svelte`, `frontend/src/lib/stores/tree.svelte.ts` |
| Service Worker/PWA | implementiert | `frontend/vite.config.ts`, `frontend/src/lib/components/OfflineBanner.svelte`, `frontend/src/lib/stores/network.svelte.ts`, `frontend/src/lib/api.ts` |
| WebSocket Real-Time | implementiert | `backend/internal/api/websocket.go`, `frontend/src/lib/stores/websocket.svelte.ts` |

### Phase 1: Code Splitting & Lazy Loading

**Ziel:** Große Abhaengigkeiten in separate Chunks splitten und lazy laden.

**Implementierung:**

- **Vite Manual Chunks** (`frontend/vite.config.ts`):
  ```typescript
  manualChunks: (id) => {
    if (id.includes('@codemirror/view') || id.includes('@codemirror/state')) {
      return 'editor';
    }
    if (id.includes('@codemirror/commands') || ...) {
      return 'editor-extensions'; // Lazy loaded
    }
    if (id.includes('force-graph')) {
      return 'graph'; // Lazy loaded
    }
  }
  ```

- **CodeMirror Extensions Lazy Loading** (`frontend/src/lib/editor/codemirror.ts`):
  - Base Editor (View, State) → Sofort geladen
  - Extensions (Commands, Markdown, Syntax Highlighting) → Lazy geladen via Dynamic Import
  - Pattern: `import('@codemirror/commands').then(...)`

- **Force-Graph Lazy Loading** (`frontend/src/lib/components/GraphCanvas.svelte`):
  ```typescript
  const ForceGraph = (await import('force-graph')).default;
  ```

- **Loading Skeletons** (`frontend/src/lib/components/skeletons/`):
  - EditorSkeleton, GraphSkeleton, ListSkeleton

### Phase 2: Virtual Scrolling

**Ziel:** Lange Listen (Suche/Trash) virtualisieren; Tree/List bleibt separat.

**Implementierung:**

- **Dependency:** `@tanstack/svelte-virtual` (modern, Svelte 5 Runes support)

- **Search Results Virtualisiert** (`frontend/src/routes/search/+page.svelte`):
  ```typescript
  const virtualizer = createVirtualizer({
    count: results.length,
    getScrollElement: () => scrollElement,
    estimateSize: () => 80, // 80px per result
    overscan: 5
  });
  ```

- **Trash List Virtualisiert** (`frontend/src/routes/trash/+page.svelte`):
  - Von Grid-Layout zu Virtual List (120px per item)

- **Tree Flattening** (`frontend/src/lib/stores/tree.svelte.ts`):
  - `getFlattenedTree()` Funktion für zukünftige Tree-Virtualisierung
  - Rekursive Tree → Flache Liste mit Level-Tracking

### Phase 3: Service Worker & PWA

**Ziel:** Read-only Offline Mode mit Service Worker + UI-Hinweisen.

**Implementierung:**

- **Vite PWA Plugin** (`frontend/vite.config.ts`):
  ```typescript
  VitePWA({
    registerType: 'autoUpdate',
    workbox: {
      runtimeCaching: [
        {
          urlPattern: ({ url }) => url.pathname === '/api/notes',
          handler: 'NetworkFirst',
          options: {
            cacheName: 'api-notes',
            networkTimeoutSeconds: 5,
            expiration: { maxEntries: 100, maxAgeSeconds: 3600 }
          }
        }
      ]
    },
    manifest: { ... }
  })
  ```

- **Offline Detection** (`frontend/src/lib/api.ts`):
  ```typescript
  if (!navigator.onLine && ['POST', 'PUT', 'DELETE'].includes(method)) {
    throw new ApiError('Offline - Changes not allowed');
  }
  ```

- **Network Status Store** (`frontend/src/lib/stores/network.svelte.ts`):
  - `getIsOnline()`, `getShowOfflineBanner()`
  - Event Listeners: `online`, `offline`

- **PWA UI Components**:
  - `OfflineBanner.svelte` - Zeigt Offline-Status
  - `InstallPrompt.svelte` - PWA Installation

- **PWA Icons**: 192x192, 512x512 (`frontend/static/icon-192.png`, `frontend/static/icon-512.png`)

### Phase 4: WebSocket Real-Time Updates

**Ziel:** Realtime Updates via WebSocket fuer Create/Update/Delete.

**Backend Implementierung:**

- **WebSocket Manager** (`backend/internal/websocket/manager.go`):
  ```go
  type Manager struct {
    connections map[int][]*Connection  // userID -> connections
    broadcast   chan BroadcastMessage
    register    chan *Connection
    unregister  chan *Connection
  }
  ```
  - Ping/Pong Heartbeat: 50s Interval, 60s Timeout
  - Read/Write Pumps als Goroutines
  - Connection Lifecycle Management

- **WebSocket Handler** (`backend/internal/api/websocket.go`):
  - HTTP Upgrade zu WebSocket
  - JWT Token Validation (aus Query-Parameter)
  - Message Handling

- **Broadcasts** (`backend/internal/api/notes.go`):
  ```go
  // Nach UpdateNote/CreateNote/DeleteNote:
  s.wsManager.BroadcastToUser(userID, websocket.Message{
    Type:    "note.updated",
    Payload: payload,
  })
  ```

- **Route**: `GET /api/ws?token=<jwt>`

**Frontend Implementierung:**

- **WebSocket Store** (`frontend/src/lib/stores/websocket.svelte.ts`):
  ```typescript
  export function connect() {
    ws = new WebSocket(`${WS_URL}?token=${token}`);
    ws.onmessage = (event) => handleMessage(JSON.parse(event.data));
  }
  ```
  - Exponential Backoff Reconnect (1s → 30s max)
  - Page Visibility API für Auto-Reconnect

- **Remote Update Handler** (`frontend/src/lib/stores/notes.svelte.ts`):
  - `handleRemoteUpdate(remoteNote)` - Mit Conflict Detection
  - `handleRemoteCreate(note)` - Neue Note hinzufügen
  - `handleRemoteDelete(id)` - Note entfernen
  - Toast Notifications bei Konflikten

- **Connection Management** (`frontend/src/routes/+layout.svelte`):
  ```typescript
  onMount(() => {
    if (auth.isAuthenticated()) {
      websocket.connect();
    }
    return () => websocket.disconnect();
  });
  ```

### Testing

**Bundle Analyzer:**
```bash
cd frontend
ANALYZE=true npm run build
```

**Virtual Scrolling Performance:**
- Chrome DevTools → Performance Tab
- Scroll mit >100 Notes
- Ziel: gleichmaessiges Scrollen ohne spuerbare Ruckler

**PWA Testing:**
- DevTools → Application → Service Workers (registriert?)
- DevTools → Application → Manifest (gültig?)
- Offline-Test: Network → Offline → Navigation funktioniert

**WebSocket Testing:**
- Zwei Browser-Fenster öffnen
- Note in Fenster 1 bearbeiten
- Fenster 2 sollte Updates anzeigen
- DevTools → Network → WS Tab → Verbindung aktiv

### Deployment-Hinweise

**nginx-proxy-manager:** WebSocket Upgrade Headers konfigurieren:
```nginx
proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";
```

**Docker Container:**
```bash
docker run -d --name xelanote \
  -p 8081:8080 --network nginx_default \
  -v xelanote_xelanote-data:/app/data \
  -e JWT_SECRET=<secret> \
  -e XELANOTE_DB=/app/data/xelanote.db \
  -e XELANOTE_ENV=production \
  xelanote:latest
```

### Messung (nicht im Repo versioniert)

Wenn ihr Performance-Messungen dokumentieren wollt, fuehrt sie separat durch
und verlinkt die Ergebnisse in `docs/development.md` oder `CHANGELOG.md`.

---

## Editor TAB-Einrückung

Der Editor unterstützt TAB-basierte Einrückung für effizientes Markdown-Editing.

### Tastaturkürzel

| Taste | Aktion |
|-------|--------|
| **TAB** | Text einrücken (2 Spaces) |
| **Shift+TAB** | Einrückung entfernen |

### Implementierung

**Location**: `frontend/src/lib/editor/codemirror.ts`

```typescript
const [
  { defaultKeymap, history, historyKeymap, indentWithTab },
  // ...
] = await Promise.all([
  import('@codemirror/commands'),
  // ...
]);

return [
  // ...
  keymap.of([
    ...defaultKeymap,
    ...historyKeymap,
    indentWithTab,  // TAB/Shift+TAB Einrückung
    // ...
  ])
];
```

**Verhalten**:
- TAB fügt Einrückung ein (standardmäßig 2 Spaces)
- Shift+TAB entfernt Einrückung
- Bei Textauswahl: Gesamte Zeilen werden ein-/ausgerückt
- CodeMirror's `indentWithTab` Keymap aus `@codemirror/commands`

### Accessibility

- TAB-Einrücken überschreibt das normale Browser-Verhalten (Tab zum nächsten Element)
- Für Keyboard-Navigation: Escape drücken, dann Tab

---

## Markdown Preview Styling

### List-Marker Sichtbarkeit

Bullet-Points und nummerierte Listen werden in der Markdown-Preview korrekt angezeigt.

**Location**: `frontend/src/app.css`

```css
.markdown-preview ul {
  list-style-type: disc;
}

.markdown-preview ol {
  list-style-type: decimal;
}
```

**Warum nötig?** Tailwind CSS Reset (Preflight) entfernt standardmäßig alle List-Styles. Diese Regeln stellen sie explizit wieder her.

### Subtilere Trennlinien und List-Marker

Horizontale Linien (`---`) und List-Marker erscheinen in einer dezenten Farbe.

**Location**: `frontend/src/app.css`

```css
/* Horizontal rules - subtler appearance */
.markdown-preview hr {
  border: none;
  border-top: 1px solid var(--color-muted-foreground);
  margin: 1.5rem 0;
  opacity: 0.6;
}

/* List markers - muted color */
.markdown-preview ul li::marker,
.markdown-preview ol li::marker {
  color: var(--color-muted-foreground);
}
```

**Design-Entscheidungen**:
- `--color-muted-foreground` ist in allen 9 Themes definiert
- `opacity: 0.6` auf `<hr>` macht die Linie subtiler ohne Kind-Elemente zu beeinflussen
- `::marker` Pseudo-Element färbt nur die Bullet-Points/Nummern, nicht den Text

### Themes-Kompatibilität

Alle Styles verwenden CSS-Variablen und funktionieren mit allen 9 Themes:
- Standard (Light/Dark)
- Nord
- Solarized (Light/Dark)
- Dracula
- Catppuccin (Latte/Mocha)

---

## Support

**Issues**: [GitHub Issues](https://github.com/xela-io/xelanote/issues)

**Discussions**: [GitHub Discussions](https://github.com/xela-io/xelanote/discussions)

**Matrix**: (TODO: Setup Matrix Room)
