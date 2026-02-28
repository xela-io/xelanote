# xelanote Wiki

Willkommen zum xelanote Wiki! Hier findest du eine vollständige Dokumentation der Architektur, Features und des Codes.

## Was ist xelanote?

xelanote ist eine **selbst-gehostete Notiz-App** — eine Privacy-first Alternative zu Obsidian/Notion. Die App bietet:

- **Markdown-Notizen** mit Wiki-Links (`[[Notiz-Titel]]`) und Wissens-Graph
- **Ende-zu-Ende-Verschlüsselung** (Zero-Knowledge — der Server sieht nie den Klartext)
- **Journal** mit Kalender-Ansicht und Jahres-Heatmap
- **Rezeptverwaltung** mit KI-gestütztem Import (URL, Foto)
- **Einkaufslisten** mit Teilen, Favoriten und KI-Sortierung
- **Canvas** (unendliche Whiteboard-Leinwand, JSON Canvas Format)
- **Offline-Modus** (IndexedDB-Queue, automatische Synchronisation)
- **Desktop-App** via Tauri (+ PWA für Mobile)
- **KI-Features** (BYOK: eigene API-Keys für Claude, Gemini, ChatGPT)

## Tech-Stack

| Komponente | Technologie |
|------------|-------------|
| Backend | Go + Chi Router + SQLite (FTS5) |
| Frontend | SvelteKit + Svelte 5 + Tailwind v4 |
| Desktop | Tauri (Rust) |
| Verschlüsselung | XChaCha20-Poly1305 (libsodium) + Argon2id |
| Auth | JWT + HttpOnly Cookies + TOTP + FIDO2/WebAuthn |
| KI | Claude, Gemini, ChatGPT (BYOK) |

## Wiki-Seiten

### Architektur
- [Architektur-Überblick](Architektur-Überblick.md) — Wie Backend und Frontend zusammenspielen
- [Backend](Backend.md) — Go-Server, Router, Middleware, Services
- [Frontend](Frontend.md) — SvelteKit, State Management, Editor
- [Datenbank](Datenbank.md) — SQLite-Schema, Migrationen, FTS5

### Features
- [Notizen-und-Wikilinks](Notizen-und-Wikilinks.md) — Markdown-Editor, Links, Backlinks, Graph
- [Authentifizierung-und-Sicherheit](Authentifizierung-und-Sicherheit.md) — Auth-Flow, 2FA, CSRF, Rate-Limiting
- [Verschlüsselung](Verschlüsselung.md) — E2E-Encryption, KEK/DEK, Security Levels
- [KI-Integration](KI-Integration.md) — LLM-Provider, Zusammenfassungen, Rezept-Import
- [Features-im-Detail](Features-im-Detail.md) — Journal, Rezepte, Einkaufslisten, Canvas, Suche

### Entwicklung
- [API-Referenz](API-Referenz.md) — Alle REST-Endpunkte
- [Entwicklung-Setup](Entwicklung-Setup.md) — Lokale Entwicklungsumgebung einrichten
