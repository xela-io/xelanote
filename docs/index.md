# xelanote Dokumentation

Zentraler Einstiegspunkt zu allen Dokumenten des Projekts.

## Wiki (Architektur-Überblick)

-   **[Wiki-Startseite](./wiki/Home.md)** — Übersicht und Einstieg ins Wiki
-   **[Architektur-Überblick](./wiki/Architektur-Überblick.md)** — Wie Backend und Frontend zusammenspielen
-   **[Backend](./wiki/Backend.md)** — Go-Server, Chi-Router, Middleware, Services, WebSocket, Jobs
-   **[Frontend](./wiki/Frontend.md)** — SvelteKit, Svelte 5 Runes, Editor, Offline-Modus
-   **[Datenbank](./wiki/Datenbank.md)** — SQLite-Schema, 59 Migrationen, FTS5-Volltextsuche
-   **[Notizen und Wikilinks](./wiki/Notizen-und-Wikilinks.md)** — Editor, Wikilinks, Graph, Versionshistorie
-   **[Auth und Sicherheit](./wiki/Authentifizierung-und-Sicherheit.md)** — Auth-Flow, 2FA, CSRF, Rate-Limiting
-   **[E2E-Verschlüsselung](./wiki/Verschlüsselung.md)** — KEK/DEK, XChaCha20, Security Levels
-   **[KI-Integration](./wiki/KI-Integration.md)** — BYOK LLM-Provider, alle KI-Features
-   **[Features im Detail](./wiki/Features-im-Detail.md)** — Journal, Rezepte, Einkaufslisten, Canvas, Suche
-   **[API-Referenz](./wiki/API-Referenz.md)** — Alle REST-Endpunkte auf einen Blick
-   **[Entwicklung Setup](./wiki/Entwicklung-Setup.md)** — Lokale Entwicklungsumgebung einrichten

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

-   **[Ende-zu-Ende-Verschlüsselung](./e2e-encryption.md)** — Argon2id + XChaCha20-Poly1305, inkl. aktueller E2EE-Grenzen
-   **[E2E Quickstart](./e2e-encryption-quickstart.md)** — Schnellanleitung für E2E-Verschlüsselung
-   **[E2E Deployment](./e2e-encryption-deployment.md)** — E2E-Verschlüsselung in Produktion
-   **[Encryption v2](./encryption-v2.md)** — Verschlüsselungs-Upgrade (v1 → v2)
-   **[Authentifizierung](./authentication.md)** — JWT, Token-Management, Session-Handling
-   **[Zwei-Faktor-Authentifizierung](./2fa.md)** — TOTP, WebAuthn/Passkeys, Backup-Codes
-   **[CAPTCHA](./captcha.md)** — Cloudflare Turnstile Integration
-   **[Signed URLs](./signed-urls.md)** — Authentifizierte Bild-Zugriffe
-   **[Security Audit](./security_audit_findings.md)** — Penetration-Test-Ergebnisse und Fixes
-   **[E2EE Audit Addendum (2026-02-28)](./security/E2EE-SECURITY-AUDIT-ADDENDUM-2026-02-28.md)** — Konsolidierte E2EE-Befunde inkl. Delta-Review
-   **[E2EE Remediation Plan (2026-02-28)](./security/E2EE-REMEDIATION-PLAN-2026-02-28.md)** — PR-basierter Umsetzungsplan für die Findings
-   **[E2EE Follow-up Roadmap (2026-02-28)](./security/E2EE-FOLLOW-UP-ROADMAP-2026-02-28.md)** — Offene P1/P2-Sicherheitsarbeiten nach den umgesetzten Fixes

## Deployment & Betrieb

-   **[Deployment Guide](./deployment.md)** — Docker-Deployment auf Homelab und Hetzner, inkl. CI/CD Auto-Deploy via Forgejo Actions
-   **[Forgejo Runner Setup](./forgejo-runner-setup.md)** — Forgejo Actions Runner auf Staging und Production (Installation, Registrierung, Workflows, Wartung)
-   **[Deployment Security](./deployment-security.md)** — Security-Checkliste für Produktion
-   **[Rollback-Anleitung](./deployment-rollback.md)** — Zurückkehren zu einer früheren Version (inkl. automatisches Rollback via CI/CD)

## Features

-   **[Editor-Funktionen](./editor-features.md)** — Live Preview, Wikilinks, Fokus-Modi, Task Drag&Drop, Inline Title
-   **[Admin Panel](./admin-panel.md)** — Benutzerverwaltung und Systemeinstellungen
-   **[Desktop-App](./desktop-app.md)** — Electron-App (Linux)
-   **[LLM Features](./llm-features.md)** — KI-Features: Tags, Links, Rechtschreibung, Zusammenfassungen, AI Recipe Import
-   **[Accessibility](./accessibility.md)** — WCAG 2.1, Keyboard-Navigation, Screen-Reader
-   **[Offline-Modus](./offline-mode.md)** — Offline Read + Write Mode mit IndexedDB-Queue und Konfliktloesung
-   **[Error Reporting](./error-reporting.md)** — Automatische Fehlerberichte und User-Feedback als Forgejo-Issues
-   **Rezepte** — Strukturiertes Rezept-Management mit Zutaten, Portionen, AI-Import, Kochbuch-Collections, Sharing; siehe [API Dokumentation](./api.md)
-   **Infinite Canvas** — Raeumliches Board (JSON Canvas spec v1.0) mit Text-Cards, eingebetteten Notizen, Links und Gruppen; siehe [Canvas RFC](./planning/canvas.md)
-   **Journal** — Dedizierte Journal-Seite mit Kalender und Eintraege-Liste
-   **Note Sharing** — Notizen mit anderen Benutzern teilen (Viewer/Editor-Rollen), siehe [API Dokumentation](./api.md#note-sharing)
-   **Folder Sharing** — Ganze Ordner teilen mit impliziter Permission-Vererbung, siehe [API Dokumentation](./api.md#folder-sharing)
-   **Collection Sharing** — Kochbuecher mit 3-Tier Prioritaets-Permission-Chain teilen
-   **Encryption Toggle** — Einzelne Notizen entschluesseln/verschluesseln, Folder Encryption Default, siehe [E2E-Verschluesselung](./e2e-encryption.md)

## Performance

-   **[Performance-Analyse](./performance-analysis.md)** — Umfassende Analyse aller Ebenen
-   **[Performance-Baseline](./performance-baseline.md)** — Baseline-Messungen
-   **[P0 Results](./performance/p0-results.md)** — Ergebnisse der CodeMirror Code-Splitting Optimierung
-   **[Graph-Optimierung](./graph-optimization-p3.md)** — Graph-Query-Optimierung mit Indizes

## Sonstiges

-   **[Logo Design](./logo_design.md)** — Design-Konzepte für das Projekt-Logo
-   **[Ideas](./ideas.md)** — Zukünftige Feature-Ideen

## Planung

Dokumente in `docs/planning/` beschreiben geplante oder kuerzlich implementierte Features:

-   **[Live Preview Optimization](./planning/live-preview-optimization.md)** — Shiki, KaTeX, Mermaid, Web Worker, Idiomorph, Scroll Sync (aktuell in Arbeit)
-   **[Design Improvements](./planning/design-improvements.md)** — UI-Redesign-Plan (grossteils umgesetzt)
-   **[Refactoring Report](./planning/refactoring-report.md)** — Ergebnisse des Refactoring-Sprints (2026-02-21)
-   **[Canvas RFC](./planning/canvas.md)** — Obsidian-Canvas-aehnliches Board-Feature (implementiert)
-   **[Claude API Integration](./planning/claude-api-integration.md)** — Claude API fuer KI-Features (implementiert)
-   **[Layer Violations Cleanup](./planning/layer-violations-cleanup.md)** — API->DB Layer-Bereinigung (abgeschlossen)
-   **[Modernization Plan](./planning/modernization-plan.md)** — Technische Modernisierung
-   **[Mobile App](./planning/mobile-app.md)** — Native Mobile App mit Capacitor
-   **[Mobile Improvements](./planning/mobile-improvements.md)** — Mobile UI-Verbesserungen
-   **[P0 Optimization Plan](./planning/p0-optimization.md)** — Performance-Optimierungsplan
-   **[List Types](./planning/list-types.md)** — Generisches Listenarten-System

## Postmortems & Incident Reports

-   `docs/postmortems/` — Analysen vergangener Incidents und deren Lösungen
-   `docs/bugs/` — Dokumentierte Bug-Analysen
-   `docs/security/` — Security-Reviews
