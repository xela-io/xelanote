# xelanote Dokumentation

Zentraler Einstiegspunkt zu allen Dokumenten des Projekts.

## Für Benutzer

-   **[Benutzerhandbuch](./benutzerhandbuch.md)** — Ausführliche Anleitung mit Benutzer- und Adminsicht
-   **[Technische Übersicht (vereinfacht)](./technische-uebersicht.md)** — Wie xelanote unter der Haube funktioniert
-   **[Markdown-Anleitung](./markdown-guide.md)** — Markdown-Syntax in xelanote

## Für Entwickler

-   **[Development Guide](./development.md)** — Setup, Workflow, Build-Prozesse
-   **[Architektur](./architecture.md)** — Systemarchitektur, Datenbankschema, Design-Entscheidungen
-   **[Coding Conventions](./conventions.md)** — Architektur-Regeln und Patterns fuer neuen Code
-   **[API Dokumentation](./api.md)** — Vollständige REST-API-Referenz
-   **[Testing Guide](../TESTING.md)** — Tests ausführen und schreiben
-   **[Environment Variables](./environment-variables.md)** — Vollständige Konfigurationsreferenz
-   **[Editor-Funktionen](./editor-features.md)** — Wikilinks, Fokus-Modi, Syntax-Highlighting
-   **[Design System](./design-system.md)** — Themes, OKLch-Farbsystem, CSS-Variablen
-   **[Frontend Design System](../frontend/DESIGN_SYSTEM.md)** — Tailwind v4 CSS-Implementierung

## Sicherheit

-   **[Ende-zu-Ende-Verschlüsselung](./e2e-encryption.md)** — AES-256-GCM, Zero-Knowledge-Architektur
-   **[E2E Quickstart](./e2e-encryption-quickstart.md)** — Schnellanleitung für E2E-Verschlüsselung
-   **[E2E Deployment](./e2e-encryption-deployment.md)** — E2E-Verschlüsselung in Produktion
-   **[Encryption v2](./encryption-v2.md)** — Verschlüsselungs-Upgrade (v1 → v2)
-   **[Authentifizierung](./authentication.md)** — JWT, Token-Management, Session-Handling
-   **[Zwei-Faktor-Authentifizierung](./2fa.md)** — TOTP, WebAuthn/Passkeys, Backup-Codes
-   **[CAPTCHA](./captcha.md)** — Cloudflare Turnstile Integration
-   **[Signed URLs](./signed-urls.md)** — Authentifizierte Bild-Zugriffe
-   **[Security Audit](./security_audit_findings.md)** — Penetration-Test-Ergebnisse und Fixes

## Deployment & Betrieb

-   **[Deployment Guide](./deployment.md)** — Docker-Deployment auf Homelab und Hetzner, inkl. CI/CD Auto-Deploy via Forgejo Actions
-   **[Forgejo Runner Setup](./forgejo-runner-setup.md)** — Forgejo Actions Runner auf Staging und Production (Installation, Registrierung, Workflows, Wartung)
-   **[Deployment Security](./deployment-security.md)** — Security-Checkliste für Produktion
-   **[Rollback-Anleitung](./deployment-rollback.md)** — Zurückkehren zu einer früheren Version (inkl. automatisches Rollback via CI/CD)

## Features

-   **[Admin Panel](./admin-panel.md)** — Benutzerverwaltung und Systemeinstellungen
-   **[Desktop-App](./desktop-app.md)** — Electron-App (Linux)
-   **[LLM Features](./llm-features.md)** — KI-Features: Tags, Links, Rechtschreibung, Zusammenfassungen
-   **[Accessibility](./accessibility.md)** — WCAG 2.1, Keyboard-Navigation, Screen-Reader
-   **[Mobile Versionshistorie](./mobile-version-history.md)** — Mobile-optimierte Versionsansicht
-   **[Titlebar Modernisierung](./titlebar-modernization.md)** — Desktop-Titlebar-Design
-   **[Offline-Modus](./offline-mode.md)** — Offline Read + Write Mode mit IndexedDB-Queue und Konfliktloesung
-   **[Error Reporting](./error-reporting.md)** — Automatische Fehlerberichte und User-Feedback als Forgejo-Issues
-   **Note Sharing** — Notizen mit anderen Benutzern teilen (Viewer/Editor-Rollen), siehe [API Dokumentation](./api.md#note-sharing)
-   **Folder Sharing** — Ganze Ordner teilen mit impliziter Permission-Vererbung fuer alle Notizen, siehe [API Dokumentation](./api.md#folder-sharing)
-   **Shared Note Placements** — Geteilte Notizen in eigene Ordner einordnen, siehe [API Dokumentation](./api.md#shared-note-placements)
-   **Encryption Toggle** — Einzelne Notizen entschluesseln/verschluesseln, Folder Encryption Default, siehe [E2E-Verschluesselung](./e2e-encryption.md) und [API Dokumentation](./api.md#post-apinotesiddecrypt)

## Performance

-   **[Performance-Analyse](./performance-analysis.md)** — Umfassende Analyse aller Ebenen
-   **[Performance-Baseline](./performance-baseline.md)** — Baseline-Messungen
-   **[P0 Results](./performance/p0-results.md)** — Ergebnisse der CodeMirror Code-Splitting Optimierung
-   **[Graph-Optimierung](./graph-optimization-p3.md)** — Graph-Query-Optimierung mit Indizes

## Sonstiges

-   **[Logo Design](./logo_design.md)** — Design-Konzepte für das Projekt-Logo
-   **[Ideas](./ideas.md)** — Zukünftige Feature-Ideen

## Planung

Dokumente in `docs/planning/` beschreiben geplante, noch nicht implementierte Features:

-   **[Mobile App](./planning/mobile-app.md)** — Native Mobile App mit Capacitor
-   **[Claude API Integration](./planning/claude-api-integration.md)** — Claude API für KI-Features
-   **[Mobile Improvements](./planning/mobile-improvements.md)** — Mobile UI-Verbesserungen
-   **[P0 Optimization Plan](./planning/p0-optimization.md)** — Performance-Optimierungsplan
-   **[List Types](./planning/list-types.md)** — Generisches Listenarten-System

## Postmortems & Incident Reports

-   `docs/postmortems/` — Analysen vergangener Incidents und deren Lösungen
-   `docs/bugs/` — Dokumentierte Bug-Analysen
-   `docs/security/` — Security-Reviews
