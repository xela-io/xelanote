# PWA-Integrationstests

Dedizierte E2E-Tests für die Progressive Web App-Funktionalität von xelanote.

## Voraussetzungen

- Node.js >= 22
- Go >= 1.25 (für Backend)
- Chromium (wird von Playwright installiert)

## Lokale Ausführung

```bash
# PWA-Tests (startet Backend + baut Frontend + führt Tests aus)
npm run test:pwa

# PWA-Tests mit sichtbarem Browser
npm run test:pwa:headed

# Nur Lighthouse CI
npm run test:lighthouse

# Alle PWA-Tests (Playwright + Lighthouse)
npm run test:pwa:all
```

Das Backend wird automatisch gestartet (in-memory SQLite), sofern nicht bereits ein Server auf Port 8080 läuft.

## Warum eine separate Playwright-Config?

Der Service Worker wird nur bei `vite build` generiert (`generateSW`-Strategie). Die bestehende
Playwright-Config (`playwright.config.ts`) nutzt `vite dev` — dort gibt es keinen Service Worker.

Die PWA-Tests verwenden daher `playwright.pwa.config.ts` mit:
- **Production Build** (`npm run build`) statt Dev-Server
- **`vite preview`** als Web-Server (Port 4173)
- **Nur Chromium** — SW-Testing ist nur in Chromium vollständig unterstützt
- **1 Worker, sequenziell** — SW-State ist global pro Browser

## Teststruktur

| Describe-Block | Was wird geprüft |
|---|---|
| Service Worker | SW-Registrierung, Aktivierung, Kontrolle der Seite |
| Offline-Fähigkeit | Offline-Laden, kritische Routen, navigateFallback, API-Denylist |
| Web App Manifest | Pflichtfelder, Icons (192/512/maskable), Farben, Apple Touch Icon |
| Caching & Sicherheit | Workbox-Precache, kein API-Caching, Cache-Clearing bei Logout |
| Navigation & UX | Ladezeit, Routen-Navigation, Zurück-Navigation |
| SW-Update-Lifecycle | Registration-State, Precache-Inhalte |
| HTTPS | HTTP→HTTPS Redirect (nur mit `TEST_BASE_URL`) |

## Neue Tests hinzufügen

Tests liegen in `tests/pwa/`. Auth-Helpers können aus den E2E-Tests importiert werden:

```typescript
import { createCredentials, registerViaApi, loginViaApi } from '../e2e/helpers/auth';
```

## Lighthouse-Konfiguration

Die Lighthouse-Config (`lighthouserc.js`) prüft die `/login`-Seite (öffentlich zugänglich).

**Hinweis**: Seit Lighthouse 12 (Nov 2024) existiert die PWA-Kategorie nicht mehr als
aggregierter Score. PWA-Audits werden als einzelne Assertions geprüft:
- `service-worker` — SW ist registriert
- `installable-manifest` — Manifest ist installierbar
- `apple-touch-icon` — Apple Touch Icon vorhanden

## Bewusste Testlücken

Diese Funktionen werden **nicht** in den PWA-E2E-Tests geprüft:

| Funktion | Grund | Abgedeckt durch |
|---|---|---|
| iOS Install Coach | Browser-spezifisch, nicht in Chromium testbar | Unit-Tests (`pwa.store.test.ts`, 580 Zeilen) |
| Share Target | Benötigt echte OS-Integration | Manuelles Testing |
| Offline Queue (IndexedDB) | Komplexes Setup mit Auth + Notiz + Reconnect | Separate E2E-Suite (geplant) |
| `beforeinstallprompt` | Browser-Event nicht zuverlässig automatisierbar | Unit-Tests |

## CI-Integration

Die PWA-Tests laufen in einem eigenen GitHub Actions Workflow (`.github/workflows/pwa-tests.yml`),
getrennt von der Haupt-Pipeline. Der Workflow wird nur bei Frontend-Änderungen getriggert und
enthält zwei Jobs:

1. **PWA Playwright Tests** — Service Worker, Offline, Manifest, Caching
2. **Lighthouse CI** — Performance, Accessibility, Best Practices, PWA-Audits
