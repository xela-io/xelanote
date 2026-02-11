# Refactoring Report (Phase 1)

Stand: 2026-02-11
Repository: `/projects/xelanote`

## 1. Analyse & Bestandsaufnahme

### 1.1 Technologien, Sprachen, Frameworks, Libraries

- Backend:
  - Sprache: Go (`backend/go.mod`, Go 1.24)
  - HTTP: `chi`
  - DB: SQLite (`mattn/go-sqlite3`, FTS5, optional SQLCipher-Tag)
  - Auth/Security: JWT, TOTP, WebAuthn, Rate Limiting
- Frontend:
  - Sprache: TypeScript + Svelte 5
  - Framework/Bundler: SvelteKit + Vite
  - Styling: Tailwind CSS 4
  - Tests: Vitest + Playwright
- Desktop:
  - Electron (`frontend/src-electron/*`)
  - Tauri/Rust (`frontend/src-tauri/*`)
- Infrastruktur:
  - Docker + Docker Compose
  - CI/CD über GitHub Actions (`.github/workflows/*`) und Forgejo Workflows (`.forgejo/workflows/*`)

### 1.2 Projektstruktur & Architektur-Pattern

- Monorepo mit klarer Trennung in `backend/` und `frontend/`.
- Backend folgt überwiegend einem Layering:
  - API-Handlers in `backend/internal/api`
  - Business-Logik in `backend/internal/service`
  - Persistenz in `backend/internal/db`
- Frontend ist feature-orientiert mit zentralen Stores (`frontend/src/lib/stores`), API-Client-Layer (`frontend/src/lib/api`) und vielen UI-Komponenten (`frontend/src/lib/components`).
- Architekturmix web + desktop (Web, Electron, Tauri) innerhalb derselben Frontend-Codebasis.

### 1.3 Einstiegspunkte

- Backend Server: `backend/cmd/server/main.go`
- Backend HTTP-Server Setup: `backend/internal/api/api.go`, `backend/internal/api/routes.go`
- Frontend App Shell: `frontend/src/routes/+layout.svelte`
- Frontend Root Page: `frontend/src/routes/+page.svelte`
- Electron Main Process: `frontend/src-electron/main.ts`
- Tauri Main: `frontend/src-tauri/src/main.rs`

### 1.4 Tests, Linting, CI/CD

- Testbestand:
  - Go-Testdateien: 63
  - Frontend-Testdateien (`.test.ts` + `.spec.ts`): 25
- Lokal ausgeführt:
  - `go test -tags 'fts5 sqlite_crypt' ./...` in `backend` -> erfolgreich
  - `npm run lint` in `frontend` -> erfolgreich
  - `npm run test` in `frontend` -> erfolgreich
- Lint/Format:
  - Go: `go vet`, `gofmt` checks (`Makefile`, CI)
  - Frontend: ESLint Flat Config (`frontend/eslint.config.js`), Prettier
  - Hooks: Lefthook (`lefthook.yml`)
- CI/CD:
  - GitHub: `ci.yml`, `quality.yml`, `security.yml`
  - Forgejo: staging/prod deploy workflows

## 2. Priorisierte Problembereiche

## 🔴 KRITISCH

### R-01: Electron umgeht serverseitige CORS-Schutzlogik
- Kategorie: Sicherheit
- Evidenz: `frontend/src-electron/main.ts:127`, `frontend/src-electron/main.ts:135`, `frontend/src-electron/main.ts:146`, `frontend/src-electron/main.ts:149`
- Problem:
  - `Origin`-Header wird aktiv entfernt.
  - Antwortheader werden mit `Access-Control-Allow-Origin: *` und gleichzeitig `Access-Control-Allow-Credentials: true` überschrieben.
- Risiko:
  - Sicherheitsgrenzen zwischen Client/Origin werden aufgeweicht; Verhalten kann nicht mehr auf Backend-CORS-Policy vertrauen.

### R-02: Kompiliertes Binary ist versioniert
- Kategorie: Tote/unerreichbare Artefakte, Wartbarkeit, Supply-Chain-Risiko
- Evidenz: `backend/cmd/server/server` (ELF-Binary, ~18 MB)
- Problem:
  - Build-Artefakt im VCS erhöht Repo-Risiko und erschwert reproduzierbare Builds.
- Risiko:
  - Intransparente Binäränderungen, unnötige Historien-Aufblähung.

### R-03: Trust-Proxy-Default ist sehr breit
- Kategorie: Sicherheit
- Evidenz: `backend/internal/api/middleware.go:13`, `backend/internal/api/middleware.go:17`, `backend/internal/api/middleware.go:87`
- Problem:
  - Standardmäßig werden gesamte private CIDR-Ranges als „trusted proxies“ akzeptiert.
- Risiko:
  - In fehlkonfigurierten Netzwerken kann IP-Spoofing über Forwarded-Header wahrscheinlicher werden.

## 🟡 WICHTIG

### Y-01: Sehr große UI-Module („God Components“)
- Kategorie: Code Smells, Performance, Wartbarkeit
- Evidenz:
  - `frontend/src/routes/settings/+page.svelte` (1208 LOC)
  - `frontend/src/lib/components/Editor.svelte` (1055 LOC)
  - `frontend/src/lib/components/UnifiedTree.svelte` (1051 LOC)
  - `frontend/src/lib/components/Sidebar.svelte` (1047 LOC)
  - `frontend/src/routes/+layout.svelte` (529 LOC)
- Problem:
  - Hohe Komplexität, schwere Testbarkeit, höheres Regression-Risiko.

### Y-02: Zentrale Route-Registrierung ist zu groß
- Kategorie: Code Smell (lange Methode/Datei)
- Evidenz: `backend/internal/api/routes.go` (~414 LOC)
- Problem:
  - Viele Endpoints in einer Datei; Änderungen sind schwer isolierbar.

### Y-03: Fehler werden an mehreren Stellen bewusst ignoriert
- Kategorie: Fehlerbehandlung
- Evidenz:
  - `backend/internal/db/recipes_images.go:88`, `backend/internal/db/recipes_images.go:129`, `backend/internal/db/recipes_images.go:153`
  - `backend/internal/db/search.go:284`, `backend/internal/db/search.go:320`
  - `backend/internal/db/folders_queries.go:45`, `backend/internal/db/folders_queries.go:93`
  - `backend/internal/db/versions.go:142`, `backend/internal/db/versions.go:185`
- Problem:
  - `_ =` bei relevanten DB-/Zeitoperationen verschleiert Laufzeitfehler.

### Y-04: Harte Konfigurationswerte in Code statt zentraler Config
- Kategorie: Hardcodierte Werte
- Evidenz:
  - `backend/internal/api/rate_limits.go:35`
  - `backend/internal/db/search.go:23`
  - `frontend/vite.config.ts` (mehrere fixe Cache/Timeout-Werte)
- Problem:
  - Tuning nur per Codeänderung; erschwert Betrieb in mehreren Umgebungen.

### Y-05: Duplizierte Logik bei Backup-Code-Formatierung
- Kategorie: Duplikation (DRY)
- Evidenz:
  - `backend/internal/service/twofa.go:102`
  - `backend/internal/service/twofa.go:320`
- Problem:
  - Gleiches Formatierungsmuster in mehreren Pfaden.

### Y-06: Dokumentationsbruch in README
- Kategorie: Dokumentationsqualität
- Evidenz: `README.md:148`, `README.md:149`
- Problem:
  - Tabellenstruktur ist unterbrochen; ein Env-Var-Block steht außerhalb der Tabelle.

### Y-07: Multi-Target-Frontend erhöht Kopplung
- Kategorie: Architektur/Kopplung
- Evidenz: `frontend/src`, `frontend/src-electron`, `frontend/src-tauri`
- Problem:
  - Web/Electron/Tauri-spezifische Anforderungen liegen eng beieinander; erhöht kognitiven Load.

### Y-08: Große Anzahl lokaler Build-Artefakte im Workspace
- Kategorie: Wartbarkeit/Performance der Entwicklerumgebung
- Evidenz: lokale Ordner `frontend/node_modules`, `frontend/.svelte-kit`, `frontend/build`, `frontend/release` etc.
- Problem:
  - Nicht als Git-Problem (korrekt ignoriert), aber Analyse-/Tooling-Last im Arbeitsverzeichnis hoch.

## 🟢 NICE-TO-HAVE

### G-01: Inkonsistente Benennung/Dateinamenslogik
- Kategorie: Namensgebung
- Evidenz:
  - Gemischte Singular/Plural-Muster, z. B. `backend/internal/service/user_apikey.go` und `backend/internal/service/user_apikeys.go`
  - API-Dateien mit unterschiedlichen Gruppierungsschemata (`notes_crud_*`, `notes_meta_*`, `notes_misc_*`)
- Problem:
  - Erschwert Orientierung.

### G-02: Sprachmischung in Kommentaren/Docs
- Kategorie: Lesbarkeit
- Evidenz:
  - Englisch + Deutsch gemischt, z. B. `frontend/vite.config.ts`.
- Problem:
  - Uneinheitliche Teamkommunikation in Code.

### G-03: Offene TODOs
- Kategorie: Technische Schulden
- Evidenz:
  - `backend/internal/db/search.go:22`
  - `frontend/src/lib/components/RecipeEditor.svelte:56`
  - `frontend/src/lib/stores/encryption.svelte.ts:390`
- Problem:
  - Bekannte Aufgaben sind nicht systematisch in Tickets überführt.

## 3. Abdeckung der geforderten Prüfpunkte

- Code Smells: vorhanden (große Dateien, Monolithen, Duplikate).
- Tote/unerreichbare Code-Abschnitte: kompiliertes Binary im Repo (`backend/cmd/server/server`).
- Inkonsistente Namensgebung: teilweise vorhanden (Dateimuster).
- Fehlerbehandlung: mehrere ignorierte Fehlerpfade.
- Hardcodierte Werte: Rate-Limits/Timeouts/Cachingwerte.
- Sicherheitsprobleme: Electron-CORS-Bypass, breites Trusted-Proxy-Default.
- Performance-Engpässe: große Komponenten/Stores mit erhöhtem Re-Render/Änderungsrisiko.
- Veraltete Dependencies/deprecated APIs:
  - Konnte in dieser Umgebung nicht verlässlich online geprüft werden (kein Zugriff auf `proxy.golang.org`/Registry-Auflösung für Updatestatus).

## 4. Phase-1-Fazit

- Test- und Lint-Basis ist solide (lokal grün), aber die strukturelle Komplexität ist hoch.
- Priorität für Phase 2:
  1. Sicherheitskorrekturen (R-01, R-03)
  2. Entfernung des versionierten Binaries (R-02)
  3. Modularisierung großer Frontend-/API-Module (Y-01, Y-02)
  4. Systematisches Error-Handling ohne Silent-Failures (Y-03)

