# Modernisierungsplan: Pragmatische Dependency-Updates

## Zusammenfassung

Schlanker Modernisierungsplan mit Fokus auf risikoarme Updates mit realem Nutzen. **Vite 7 wird bewusst zurückgestellt**, bis das Plugin-Ökosystem stabil ist (voraussichtlich Q2 2026). Go wird auf 1.25 aktualisiert (nicht 1.26, da erst 2 Tage alt). **Geschätzte Hands-on-Zeit: 2 Stunden.**

### Abgrenzung: Was wir NICHT machen (und warum)

| Zurückgestellt | Grund |
|---|---|
| **Vite 6 → 7** | Minimaler Nutzen, aber `@sveltejs/vite-plugin-svelte` Major-Upgrade (5→6) nötig, `vite-plugin-pwa` Support unklar, `vite-plugin-top-level-await` bekannte Probleme. Warten bis Ökosystem stabil. |
| **Go 1.26** | Released am 10.02.2026, erst 2 Tage alt. Zu frisch für Production. Go 1.25 ist battle-tested und hat die gleichen Key-Features (`CrossOriginProtection`, `testing/synctest`). |
| **Svelte/SvelteKit Patch-Updates** | Reine Patch-Level-Updates, kommen automatisch via `^`-Ranges bei `npm install`. |

---

## Betroffene Bereiche

- **Backend:** `backend/go.mod`, `backend/go.sum`, `backend/.air.toml`, `Dockerfile`, `.forgejo/workflows/deploy-staging.yml`
- **Frontend:** `frontend/package.json`, `frontend/package-lock.json`
- **Dokumentation:** `CHANGELOG.md`

## Risiken

Alle Updates in diesem Plan sind **rückwärtskompatibel** (Minor/Patch-Level). Risiko ist gering.

| Update | Risiko | Mitigation |
|--------|--------|------------|
| Go 1.24 → 1.25 | Niedrig (Go 1 Kompatibilität) | `make quality` + `make test` + Race-Detector |
| go-chi v5.1 → v5.2.5 | Sehr niedrig (Drop-in) | `make test` |
| CodeMirror Patches | Sehr niedrig (keine Breaking Changes verifiziert) | `make test-frontend` + manueller Editor-Test |
| Air Build-Tags Fix | Keines (Bug-Fix) | `make dev` starten |

---

## Phase 1: Backend-Updates

### Task 1.1: Go 1.24 → 1.25 + go-chi v5.2.5

- **Beschreibung**: Go-Runtime und Dependencies aktualisieren. Go 1.25 ist seit August 2025 stabil und bringt `http.CrossOriginProtection`, `testing/synctest` und Container-Awareness (`GOMAXPROCS` beachtet cgroup-Limits).
- **Dateien**:
  - `backend/go.mod` (Zeile 3: `go 1.25.0`)
  - `backend/go.sum`
  - `Dockerfile` (`golang:1.25-alpine`)
  - `.forgejo/workflows/deploy-staging.yml` (Zeile 32: `golang:1.24-alpine` → `golang:1.25-alpine`)
- **Befehle**:
  ```bash
  # go.mod + Dockerfile + Workflow manuell anpassen, dann:
  cd backend && go get -u github.com/go-chi/chi/v5@v5.2.5
  cd backend && go mod tidy
  ```
- **Akzeptanzkriterien**:
  - [ ] `backend/go.mod`: `go 1.25.0` + chi `v5.2.5`
  - [ ] `Dockerfile`: `golang:1.25-alpine`
  - [ ] `.forgejo/workflows/deploy-staging.yml` Zeile 32: `golang:1.25-alpine`
  - [ ] `make quality` grün
  - [ ] `make test` grün
  - [ ] Race-Detector: `cd backend && go test -race -tags "fts5 sqlite_crypt" ./...`
  - [ ] `make docker` baut erfolgreich

### Task 1.2: Air Build-Tags synchronisieren

- **Beschreibung**: `.air.toml` Build-Tags mit Makefile synchronisieren. Behebt Inkonsistenz: Air baut nur mit `fts5`, Makefile mit `fts5 sqlite_crypt`.
- **Dateien**:
  - `backend/.air.toml` (Build-Command)
- **Akzeptanzkriterien**:
  - [ ] Air Build-Command enthält `-tags 'fts5 sqlite_crypt'`
  - [ ] `make dev` startet Backend mit Hot-Reload ohne Fehler

---

## Phase 2: Frontend-Updates

### Task 2.1: CodeMirror-Pakete aktualisieren

- **Beschreibung**: Alle CodeMirror-Dependencies auf neueste Patch-Versionen. Keine Breaking Changes (verifiziert via [codemirror.net/docs/changelog](https://codemirror.net/docs/changelog/), Stand 2026-02-12).
- **Dateien**:
  - `frontend/package.json`
  - `frontend/package-lock.json`
- **Befehle**:
  ```bash
  cd frontend && npm update @codemirror/view @codemirror/state @codemirror/autocomplete @codemirror/commands @codemirror/search @codemirror/lang-markdown @codemirror/language
  ```
- **Zielversionen** (verifiziert):
  - `@codemirror/view` ^6.35.0 → ^6.39.14
  - `@codemirror/state` ^6.4.1 → ^6.5.4
  - `@codemirror/autocomplete` ^6.18.3 → ^6.20.0
  - `@codemirror/commands` ^6.7.1 → ^6.10.2
- **Akzeptanzkriterien**:
  - [ ] `npm install` ohne Peer-Dependency-Warnungen
  - [ ] `make quality` grün (ESLint, Prettier, Typecheck)
  - [ ] `make test-frontend` grün
  - [ ] Manueller Editor-Test: Öffnen, Tippen, Autocomplete (`[[`), Find (Ctrl+F), Undo/Redo

---

## Phase 3: CHANGELOG + Deploy

### Task 3.1: CHANGELOG.md aktualisieren

- **Beschreibung**: Alle Änderungen im Keep-a-Changelog-Format dokumentieren. Pflicht vor Commit (lefthook pre-commit Hook).
- **Akzeptanzkriterien**:
  - [ ] `[Unreleased]` Sektion mit Einträgen:
    - **Changed:** Go 1.25.0, go-chi v5.2.5, CodeMirror-Pakete aktualisiert
    - **Fixed:** Air Build-Tags auf `fts5 sqlite_crypt` synchronisiert

### Task 3.2: Commit + Staging-Deploy

- **Beschreibung**: Änderungen committen und nach `main` pushen. Staging-Workflow triggert automatisch auf Push zu `main`.
- **Akzeptanzkriterien**:
  - [ ] Commit mit `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>`
  - [ ] `git push forgejo main` triggert Staging-Workflow
  - [ ] Staging-Workflow grün (inkl. Quality-Gates mit `golang:1.25-alpine`)
  - [ ] `curl https://notes.over-cloud.de/health` gibt 200
  - [ ] Manueller Smoke-Test auf Staging (Login, Notiz bearbeiten, Suche)

### Task 3.3: Production-Deploy (nach 24h Staging)

- **Beschreibung**: Nach erfolgreicher 24h Staging-Beobachtung: Tag erstellen, Production-Deploy triggern.
- **Akzeptanzkriterien**:
  - [ ] 24h Staging-Beobachtung ohne Issues
  - [ ] Semver-Version aus CHANGELOG ermitteln
  - [ ] `git tag -a v<VERSION> -m "Release <VERSION>"` + `git push forgejo v<VERSION>`
  - [ ] Production-Workflow grün
  - [ ] `curl https://xelanote.com/health` gibt 200

---

## Rollback-Strategie

Alle Änderungen sind in einem einzigen Commit. Rollback ist trivial:

**Automatisch:** Staging- und Production-Workflows haben eingebauten Auto-Rollback bei fehlgeschlagenem Health-Check.

**Manuell (falls nötig):**
```bash
git revert HEAD
git push forgejo main  # Staging
# Nach Staging-Verify:
git tag -a v<VERSION>-fix -m "Rollback"
git push forgejo v<VERSION>-fix  # Production
```

**Production workflow_dispatch (Alternative):**
Über Forgejo UI: Repository → Actions → "Deploy to Production" → "Run workflow" → Reason: "Rollback".

**Datenbank:** Keine Schema-Migrationen, kein DB-Rollback nötig.
Backups: `/var/lib/forgejo-runner/backups/` (automatisch via Production-Workflow).

---

## Zurückgestellte Modernisierungen (Backlog)

Diese Items werden separat geplant, wenn die Voraussetzungen erfüllt sind:

### Vite 7 Migration (voraussichtlich Q2 2026)

**Voraussetzungen (alle müssen erfüllt sein):**
- [ ] `@sveltejs/vite-plugin-svelte` v6.x ist mindestens 4 Wochen stabil
- [ ] `vite-plugin-pwa` hat offiziellen Vite 7 Support
- [ ] `vite-plugin-top-level-await` Kompatibilitätsproblem ([vitejs/vite#19160](https://github.com/vitejs/vite/issues/19160)) ist gelöst ODER Plugin kann entfernt werden
- [ ] Forgejo Runner auf Staging + Production haben Node.js 20.19+ oder 22.12+

**Scope bei Migration:**
- Vite ^6 → ^7
- `@sveltejs/vite-plugin-svelte` 5.x → 6.x (Major-Upgrade, Breaking Changes prüfen)
- `vite-plugin-top-level-await` evaluieren (bei `esnext` Target vermutlich entfernbar)

### Go 1.26 Upgrade (voraussichtlich Q2 2026)

**Voraussetzungen:**
- [ ] Go 1.26.x hat mindestens 2 Patch-Releases (Bug-Fixes nach Initial-Release)
- [ ] `mattn/go-sqlite3` bestätigt Go 1.26 Kompatibilität
- [ ] Community-Feedback zeigt keine kritischen Regressionen

**Zusätzlicher Nutzen gegenüber 1.25:**
- Green Tea GC als Default (Performance)
- 30% weniger cgo-Overhead
- `new()` mit Initialwert-Syntax

---

## Referenzen

Verifiziert via Context7 und Web-Recherche am 2026-02-12:

- Go 1.25 Release Notes: https://go.dev/doc/go1.25
- Go 1.26 Release Blog: https://go.dev/blog/go1.26
- go-chi Releases: https://github.com/go-chi/chi/releases
- CodeMirror Changelog: https://codemirror.net/docs/changelog/
- Vite 7.0 Announcement: https://vite.dev/blog/announcing-vite7
- @sveltejs/vite-plugin-svelte Releases: https://github.com/sveltejs/vite-plugin-svelte/releases
- vite-plugin-top-level-await Issue: https://github.com/vitejs/vite/issues/19160

---

**Plan erstellt:** 2026-02-12
**Status:** Bereit zur Umsetzung
**Verantwortlich:** xela-io + Claude Opus 4.6
**Nächster Schritt:** Task 1.1 starten
