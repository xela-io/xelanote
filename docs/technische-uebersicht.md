# Technische Übersicht (vereinfacht)

Dieses Dokument erklärt die Technik von xelanote in einfacher Sprache. Es richtet sich an Leserinnen und Leser, die den Code verstehen wollen, aber (noch) wenig Erfahrung mit Go, TypeScript oder Web-Architektur haben.

## Inhaltsverzeichnis

1. Was ist xelanote technisch?
2. Hauptbestandteile (Frontend, Backend, Datenbank)
3. Wie Daten fließen (Beispiel: Notiz speichern)
4. Warum diese Technologien?
5. Sicherheit in einfachen Worten
6. Suchfunktion und Performance
7. Verschlüsselung (Kurzfassung)
8. Betrieb und Deployment (Kurzfassung)
9. Wo finde ich Details?

---

## 1. Was ist xelanote technisch?

xelanote ist eine klassische Web-Anwendung mit einem **Client-Server-Modell**:

- **Client**: Browser oder Desktop-App, zeigt die Oberfläche.
- **Server**: verarbeitet Requests, prüft Berechtigungen, speichert Daten.
- **Datenbank**: speichert alle Notizen, Benutzer, Ordner, Links usw.

---

## 2. Hauptbestandteile

```
Browser/Desktop (Frontend)  <—HTTP/JSON—>  Go-Server (Backend)  <—SQL—>  SQLite DB
```

### Frontend (Benutzeroberfläche)

- Technologie: **SvelteKit** + **TypeScript**
- Aufgaben:
  - UI anzeigen (Editor, Sidebar, Settings)
  - Daten vom Server abholen (API)
  - Eingaben in Markdown verarbeiten

### Backend (Server)

- Technologie: **Go**
- Aufgaben:
  - API-Endpunkte bereitstellen
  - Authentifizierung und Berechtigungen prüfen
  - Datenbankzugriffe durchführen

### Datenbank

- Technologie: **SQLite** (Datenbank in einer Datei)
- Zusatz: **FTS5** für Volltextsuche

---

## 3. Wie Daten fließen (Beispiel: Notiz speichern)

```
Editor tippt
  -> API Request (PUT /api/notes/:id)
  -> Auth-Check (JWT)
  -> DB-Update (notes, links, search)
  -> API Response (JSON)
  -> UI aktualisiert
```

1. Du schreibst im Editor.
2. Das Frontend sendet einen **API-Request** an den Server.
3. Der Server prüft deinen Login-Token.
4. Der Server schreibt die Notiz in die SQLite-Datenbank.
5. Der Server aktualisiert Links und Suchindex.
6. Das Frontend zeigt den neuen Stand an.

---

## 4. Warum diese Technologien?

- **Go**: Schnell, stabil, ein einziges Binary fuer Deployment.
- **SvelteKit**: Kleine Bundle, schnelle UI, moderne Web-Technik.
- **SQLite**: Keine extra Datenbank-Installation nötig, ideal für Self-Hosting.
- **FTS5**: Schnelle Volltextsuche direkt in SQLite.

---

## 5. Sicherheit in einfachen Worten

- **Login**: Nutzer melden sich mit Username/E-Mail + Passwort an.
- **JWT-Tokens**: Der Server gibt einen signierten Token aus, der wie ein digitaler Ausweis funktioniert.
- **Kurzlebige Tokens**: Access Tokens laufen schnell ab, Refresh Tokens erneuern sie.
- **2FA**: Optionaler zweiter Faktor (TOTP) für zusätzliche Sicherheit.

---

## 6. Suchfunktion und Performance

- Volltextsuche erfolgt über SQLite FTS5.
- Bei sehr vielen Notizen bleibt die Suche schnell, weil FTS5 dafür gebaut ist.
- Die App nutzt Caching und gezielte Datenbank-Queries.

---

## 7. Verschlüsselung (Kurzfassung)

- **Ende-zu-Ende-Verschlüsselung** bedeutet: Der Server sieht nie den Klartext.
- Schlüssel werden **im Browser** aus deinem Passwort abgeleitet.
- Moderne Kryptografie: **Argon2id** + **XChaCha20-Poly1305**.
- Optional: Titelverschlüsselung (mehr Datenschutz, weniger Suche).

---

## 8. Betrieb und Deployment (Kurzfassung)

### Lokal (Entwicklung)

- Backend starten: `make run-backend`
- Frontend starten: `make run-frontend`

### Docker

- `docker compose up -d --build`

### Wichtige Umgebungsvariablen

- `JWT_SECRET` (Pflicht)
- `XELANOTE_DB` (Pfad der Datenbank)
- `XELANOTE_ENV` (development/production)
- `CORS_ALLOWED_ORIGINS` (Production)

---

## 9. Wo finde ich Details?

- `docs/architecture.md` (Architektur)
- `docs/api.md` (API-Referenz)
- `docs/development.md` (Entwicklung)
- `docs/authentication.md` (Login/Token/2FA)
- `docs/e2e-encryption.md` (Verschlüsselung)
- `docs/benutzerhandbuch.md` (ausführliche Anleitung)
