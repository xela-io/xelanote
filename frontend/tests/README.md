# Xelanote Testing-Architektur

Umfassendes Test-Framework basierend auf Playwright mit Visual Regression, Accessibility-Testing, Design-Validierung und Error-Monitoring.

## Verzeichnisstruktur

```
tests/
├── e2e/                    # Bestehende E2E-Tests (Auth, Notes, Folders, etc.)
│   ├── helpers/            # Auth-Hilfsfunktionen
│   └── *.spec.ts           # E2E-Testdateien
├── visual/                 # Visual Regression Tests
│   └── pages.visual.spec.ts
├── functional/             # Funktionale Tests
│   ├── navigation.spec.ts
│   ├── auth-flows.spec.ts
│   └── interactive-elements.spec.ts
├── accessibility/          # Barrierefreiheits-Tests (axe-core)
│   └── a11y.spec.ts
├── design/                 # Design-System-Validierung
│   ├── design-audit.spec.ts  # Regelbasiert (Layout, Kontrast, Hierarchie)
│   └── vision-review.spec.ts # KI-gestützt via Claude Vision
├── fixtures/               # Playwright-Fixtures
│   ├── auth.fixture.ts     # Authentifiziertes Kontext-Fixture
│   └── screenshot.fixture.ts # Screenshot-Stabilisierung
├── utils/                  # Hilfsbibliotheken
│   ├── error-collector.ts  # Runtime-Error-Monitoring
│   ├── design-validator.ts # Design-System-Validierung
│   ├── screenshot-comparator.ts # Pixel-Level-Vergleich
│   ├── report-generator.ts # HTML-Report-Generator
│   ├── generate-report.ts  # Report-Generierungs-Script
│   ├── vision-design-reviewer.ts # Claude-CLI-Integration für Vision Review
│   └── vision-report-generator.ts # HTML-Report für Vision Review
├── reports/                # Generierte Reports (gitignored)
│   └── vision/             # Vision Design Review Reports + Screenshots
├── results/                # Test-Artefakte (gitignored)
├── TESTING-MANIFEST.md     # Vollständige Test-Abdeckungsübersicht
└── README.md               # Diese Datei
```

## Setup

### Voraussetzungen

- Node.js >= 22
- Go >= 1.25 (für Backend in E2E-Tests)
- SQLite3 Entwicklungs-Headers

### Installation

```bash
cd frontend
npm install
npx playwright install --with-deps
```

### Backends starten (für lokale E2E-Tests)

```bash
# Backend wird automatisch von Playwright gestartet, alternativ manuell:
cd backend
JWT_SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
  XELANOTE_DB=:memory: XELANOTE_ENV=test go run -tags "fts5" ./cmd/server
```

## Test-Suiten

### E2E-Tests (bestehend)

Kernfunktionalität: Login, Notes CRUD, Folders, 2FA, Verschlüsselung.

```bash
npm run test:e2e
```

### Funktionale Tests

Navigation, Auth-Flows, interaktive Elemente, Broken Links, Layout-Integrität, Performance.

```bash
npm run test:functional
```

### Visual Regression

Screenshot-Vergleich aller Seiten in verschiedenen Viewports, Themes und Sprachen.

```bash
npm run test:e2e:visual          # Nur Desktop Chromium
npm run test:e2e:visual:all      # Alle Browser + Viewports
npm run test:e2e:visual:update   # Baselines aktualisieren
```

**Umgebungsvariablen:**

- `VISUAL_LOCALE` – `de` (Standard) oder `en`
- `VISUAL_THEME` – `gruvbox-light` (Standard) oder `gruvbox-dark`

### Accessibility-Tests

WCAG 2.1 Level AA mit axe-core + manuelle Prüfungen (Alt-Texte, Fokus-Indikatoren, ARIA).

```bash
npm run test:a11y
```

### Design-Validierung

Typografie, Layout-Overflow, Touch-Targets, Kontrast-Ratios, Heading-Hierarchie.

```bash
npm run test:design
```

### KI-gestützte Design-Bewertung (Vision Review)

Nutzt Claude Vision über den `claude` CLI, um Screenshots aller Kernseiten automatisch auf Design-Qualität zu bewerten. Bewertet werden: visuelle Hierarchie, Whitespace, Konsistenz, Lesbarkeit und Layout — jeweils mit Scores von 1-10.

**Voraussetzungen:**

- `claude` CLI installiert und authentifiziert (Claude Max Abo)
- Backend + Frontend laufen (wird automatisch via Playwright gestartet)

```bash
npm run test:design:vision
```

**Was passiert:**

1. Screenshots von 6 Desktop- und 5 Mobile-Seiten werden erfasst
2. Claude analysiert jede Seite einzeln (Scores, Issues, Empfehlungen)
3. Zusätzliche Cross-Page-Konsistenz-Analyse über alle Desktop-Seiten
4. HTML-Report mit Radar-Charts, eingebetteten Screenshots und Issue-Tabellen

**Report:**

```bash
# Nach dem Test-Lauf:
open tests/reports/vision/vision-design-review.html
```

**Ohne `claude` CLI:** Tests werden automatisch mit `test.skip()` übersprungen — kein Fehler, keine CI-Blockade.

**Fail-Kriterien:** Durchschnittlicher Score < 4 oder kritische Design-Issues.

### Alles ausführen

```bash
npm run test:all     # Alle Suiten
npm run test:ci      # CI-optimiert (e2e + functional + a11y)
```

## Playwright-Projekte

| Projekt                   | Browser  | Viewport  | Test-Verzeichnis       |
| ------------------------- | -------- | --------- | ---------------------- |
| `e2e`                     | Chromium | Desktop   | `tests/e2e/`           |
| `visual-desktop-chromium` | Chromium | 1440×900  | `tests/visual/`        |
| `visual-desktop-firefox`  | Firefox  | 1440×900  | `tests/visual/`        |
| `visual-desktop-webkit`   | WebKit   | 1440×900  | `tests/visual/`        |
| `visual-tablet-portrait`  | WebKit   | iPad      | `tests/visual/`        |
| `visual-tablet-landscape` | WebKit   | iPad quer | `tests/visual/`        |
| `visual-mobile-ios`       | WebKit   | iPhone 14 | `tests/visual/`        |
| `visual-mobile-android`   | Chromium | Pixel 7   | `tests/visual/`        |
| `functional`              | Chromium | Desktop   | `tests/functional/`    |
| `accessibility`           | Chromium | Desktop   | `tests/accessibility/` |
| `design`                  | Chromium | Desktop   | `tests/design/`        |

## Neue Tests erstellen

### Visual Test für eine neue Seite

1. Route in `tests/visual/pages.visual.spec.ts` zu `PROTECTED_ROUTES` oder `PUBLIC_ROUTES` hinzufügen
2. Baselines generieren: `npm run test:e2e:visual:update`
3. Baselines committen

### Funktionaler E2E-Test

```typescript
// tests/functional/mein-feature.spec.ts
import { expect } from '@playwright/test';
import { test } from '../fixtures/screenshot.fixture';

test.describe('Mein Feature @e2e', () => {
  test('Feature funktioniert', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/meine-route');
    // ... Test-Logik
  });
});
```

### Test mit Error-Monitoring

```typescript
import { ErrorCollector } from '../utils/error-collector';

test('keine Fehler auf der Seite', async ({ authenticatedPage }) => {
  const collector = new ErrorCollector();
  collector.attach(authenticatedPage);

  await authenticatedPage.goto('/meine-route');
  await authenticatedPage.waitForTimeout(2000);

  expect(collector.hasErrors('critical')).toBe(false);
});
```

## Baselines aktualisieren

```bash
# Alle Baselines aktualisieren (nach bewussten UI-Änderungen)
npm run test:e2e:visual:update

# Dann die neuen Snapshots committen
git add tests/visual/*.spec.ts-snapshots/
git add tests/e2e/ui-visual-regression.spec.ts-snapshots/
```

## Reports

```bash
# Playwright HTML-Report öffnen
npm run test:report

# Custom HTML-Report generieren (nach Test-Lauf)
npm run test:report:generate
```

## Troubleshooting

### Tests schlagen lokal fehl aber nicht im CI

- Prüfe die Viewport-Größe: `process.env.CI` ändert einige Defaults
- Schriftarten: Lokal installierte Fonts können Screenshots beeinflussen
- Lösung: `npx playwright install --with-deps` ausführen

### Visual Tests sind flaky

- Animationen werden automatisch deaktiviert (CSS `animation: none`)
- Falls nötig: `page.waitForTimeout(1000)` vor Screenshot
- Dynamische Inhalte (Timestamps) mit `mask` Option ausblenden

### Backend startet nicht

- Prüfe ob Port 8080 frei ist: `lsof -i :8080`
- SQLite-Header installiert: `sudo apt install libsqlite3-dev`
- Go-Version: mindestens 1.25.x

### axe-core meldet False Positives

- CodeMirror-Editor wird standardmäßig ausgeschlossen (`.cm-editor`)
- Canvas-Elemente werden ausgeschlossen (`canvas`)
- Weitere Ausschlüsse in `tests/accessibility/a11y.spec.ts` hinzufügen

## CI/CD

### GitHub Actions

- **`quality.yml`**: Visual Regression bei Push/PR (bestehend)
- **`playwright-tests.yml`**: Erweiterte Test-Suite bei Frontend-Änderungen
  - Matrix: e2e, functional, accessibility, design
  - Artefakte: Reports, Screenshots, Diffs (7 Tage)
  - Caching: Playwright-Browser, node_modules

### Lokaler CI-Test

```bash
CI=true npm run test:ci
```
