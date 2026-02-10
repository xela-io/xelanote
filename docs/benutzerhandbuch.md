# xelanote Benutzerhandbuch (ausführlich)

Diese Anleitung richtet sich an eine gemischte Zielgruppe: Endnutzer, Admins und Leser, die den Code verstehen möchten, aber (noch) wenig Erfahrung mit den verwendeten Programmiersprachen haben. Deshalb sind die technischen Teile bewusst ausführlich und in einfacher Sprache erklärt.

Stand: Projektstatus im Repository; bitte bei Änderungen die verlinkten Detail-Dokumente prüfen.

## Inhaltsverzeichnis

1. Überblick
2. Schnellstart für Benutzer
3. Grundkonzepte (Notizen, Ordner, Links, Suche)
4. Benutzeroberfläche im Überblick
5. Arbeiten mit Notizen
6. Editor- und Markdown-Funktionen
7. Wikilinks, Backlinks und Graph
8. Suche, Filter, Tags
9. Versionen, Papierkorb, Wiederherstellung
10. Import/Export und Uploads
11. Einstellungen und Personalisierung
12. Sicherheit: Login, 2FA, Verschlüsselung, Recovery
13. Admin-Bereich
14. Desktop-Apps (Electron, Tauri)
15. Fehlerbehebung (FAQ)
16. Technische Dokumentation in einfacher Sprache
17. Glossar

---

## 1. Überblick

xelanote ist eine selbst gehostete Notiz-App mit Markdown-Editor, internen Wikilinks, Backlinks, Volltextsuche, optionaler Ende-zu-Ende-Verschlüsselung und Admin-Bereich. Die App besteht aus:

- **Frontend**: Benutzeroberfläche im Browser (SvelteKit/TypeScript)
- **Backend**: Server mit API und Datenbankzugriff (Go)
- **Datenbank**: SQLite mit Volltextsuche (FTS5)

---

## 2. Schnellstart für Benutzer

### Registrierung und Login

1. Öffne die App-URL im Browser.
2. Registriere dich (wenn Registrierung aktiviert ist).
3. Logge dich mit Username/E-Mail und Passwort ein.
4. Optional: CAPTCHA (Turnstile) wird angezeigt, wenn es vom Admin aktiviert wurde.

### Erste Notiz erstellen

1. Klicke in der Sidebar auf **Neue Notiz**.
2. Gib einen Titel ein und schreibe den Inhalt in Markdown.
3. Speichern passiert automatisch (Auto-Save); manuelles Speichern geht per `Ctrl/Cmd + S`.

### Erste Verlinkung

1. Tippe `[[` im Editor.
2. Wähle eine Notiz aus der Autovervollständigung oder tippe den Namen.
3. Es entsteht ein Link wie `[[Notiz-Titel]]`.

---

## 3. Grundkonzepte (Notizen, Ordner, Links, Suche)

### Notizen

- Jede Notiz hat **Titel**, **Inhalt** (Markdown), **Metadaten** (Erstellt/Geändert), ggf. **Farbe** und **Tags**.
- Notizen können **verschlüsselt** gespeichert werden (E2E). Dann kann der Server den Inhalt nicht lesen.

### Ordner

- Ordner strukturieren Notizen in einer Baumstruktur.
- Ordner können **Farben** bekommen (VS Code Stil-Farbmarkierung in der Sidebar).

### Wikilinks und Backlinks

- `[[Notiz-Titel]]` erzeugt einen Link zu einer Notiz.
- Wenn die Notiz nicht existiert, kann sie direkt erstellt werden.
- **Backlinks** zeigen alle Notizen, die auf die aktuelle Notiz verweisen.

### Volltextsuche

- Die Suche basiert auf SQLite FTS5.
- Bei verschlüsselten Notizen ist die Suchfähigkeit optional und kann Sicherheitsrisiken haben (siehe Abschnitt 12).

---

## 4. Benutzeroberfläche im Überblick

Typische Hauptbereiche:

- **Sidebar**: Ordner, Notizliste, Quick-Access, Einstellungen, Admin (wenn berechtigt)
- **Editor**: Markdown-Eingabe mit Toolbar
- **Preview**: Rendered Markdown (Split-View möglich)
- **Graph**: Visualisierung der Notiz-Verknüpfungen
- **Settings**: Account, Sicherheit, Verschlüsselung, UI-Optionen

---

## 5. Arbeiten mit Notizen

### Notiz erstellen

- Über Sidebar: `Neue Notiz`
- Optional in Ordnern erstellen
- Titel und Inhalt eingeben

### Notiz umbenennen

- Titel im Editor ändern oder per Kontextmenü
- Wikilinks werden in der Regel bei „Rename“ aktualisiert

### Notiz verschieben

- Drag & Drop in der Sidebar
- Oder über „Verschieben“ im Kontextmenü

### Notiz löschen

- Standardmäßig Soft-Delete in den Papierkorb
- Wiederherstellung möglich
- Permanentes Löschen nur über Papierkorb

---

## 6. Editor- und Markdown-Funktionen

### Editor-Modi

xelanote bietet drei Ansichten:

1. **Nur Editor** (Markdown-Quelltext)
2. **Split-View** (Editor + Preview)
3. **Nur Preview** (gerenderter Markdown)

### Markdown-Grundfunktionen

Unterstützt werden u. a.:

- Überschriften `#` bis `######`
- Fett/Kursiv/Strikethrough
- Listen und Checklisten
- Links und automatische Links
- Codeblöcke (mit Sprache)
- Zitate
- Bilder (Upload oder URL)

### Checklisten

- Markdown: `- [ ]` oder `- [x]`
- In der Preview anklickbar (automatische Synchronisation)

### Farbige Hervorhebungen

- Syntax: `{color:FARBE}Text{/color}`
- Farben: Theme-Farben oder Hex/RGB

### Fokus-Funktionen

- **Typewriter Mode**: Aktuelle Zeile bleibt in der Mitte
- **Dim Inactive Lines**: Andere Zeilen werden visuell abgedimmt

---

## 7. Wikilinks, Backlinks und Graph

### Wikilinks

- Einfach: `[[Notiz-Titel]]`
- Mit Alias: `[[Notiz-Titel|Anzeigetext]]`

### Backlinks

- Automatisch berechnet
- Zeigt alle Notizen, die auf die aktuelle verweisen

### Graph-Ansicht

- Visualisiert Verknüpfungen als Knoten und Kanten
- Hilft beim Explorieren von Wissensnetzwerken

---

## 8. Suche, Filter, Tags

### Suche

- Schnellzugriff über Suchfeld
- Optional: Quick Search für Autocomplete

### Tags

- Tags können Notizen zugeordnet werden
- Filterung nach Tags möglich

---

## 9. Versionen, Papierkorb, Wiederherstellung

### Version History

- Versionen erlauben den Zugriff auf frühere Stände einer Notiz
- Vergleich und Wiederherstellung möglich

### Papierkorb (Trash)

- Gelöschte Notizen landen im Papierkorb
- Wiederherstellung möglich
- Permanentes Löschen nur im Papierkorb

---

## 10. Import/Export und Uploads

### Markdown-Import

- Importiert bestehende Markdown-Dateien als Notizen

### Markdown-Export

- Exportiert Notizen als Markdown

### Uploads

- Bilder lassen sich per Drag & Drop oder Einfügen hochladen
- Bilder werden über `/api/uploads/...` ausgeliefert

---

## 11. Einstellungen und Personalisierung

### UI und Themes

- Mehrere Themes (hell und dunkel)
- Vorschau-Theme getrennt wählbar

### Auto-Save

- Standardmäßig aktiv, speichert nach kurzer Inaktivität

### Fokus-Optionen

- Typewriter Mode
- Dim Inactive Lines

---

## 12. Sicherheit: Login, 2FA, Verschlüsselung, Recovery

### Login und Tokens (einfach erklärt)

- Beim Login erhält der Client ein **Access Token** (kurzlebig) und ein **Refresh Token** (langlebig).
- Access Token wird für API-Requests genutzt; Refresh Token erneuert ihn automatisch.

### 2FA (TOTP)

- Optional aktivierbar in den Einstellungen.
- Authenticator-App generiert 6-stellige Codes.
- Backup-Codes für den Notfall.

### Ende-zu-Ende-Verschlüsselung (E2E)

- Verschlüsselung passiert im Browser, bevor Daten an den Server gehen.
- Server speichert nur verschlüsselten Text.
- Verwendet moderne Kryptografie (Argon2id + XChaCha20-Poly1305).

**Wichtige Optionen:**

- **Titel verschlüsseln**: maximaler Datenschutz, aber keine Titelsuche mehr.
- **Suchbare Keywords**: ermöglicht Suche, speichert aber Keywords unverschlüsselt (Risiko!).

### Recovery Key

- Recovery Key ist die einzige Möglichkeit, verschlüsselte Notizen bei Passwortverlust zu retten.
- **Hinweis:** Die Wiederherstellung per Recovery Key ist laut Doku noch nicht vollständig implementiert.

---

## 13. Admin-Bereich

### Zugriff

- Der erste registrierte Benutzer wird automatisch Admin.
- Admin-Bereich ist unter `/admin` erreichbar.

### Funktionen

- Benutzerverwaltung (Admins setzen, Nutzer löschen)
- Systemstatistiken
- Activity Logs (Audit-Trail)
- Systemeinstellungen (Registrierung, Limits, Maintenance)

---

## 14. Desktop-Apps (Electron, Tauri)

### Electron (stabil)

- Native App mit OS-Keychain Integration
- Automatischer Login, sichere Token-Speicherung
- CORS-Bypass für lokale Nutzung

### Tauri (experimentell)

- Kleinere Binary
- Rust-Backend für Token- und Schlüsselverwaltung
- Keyring-Integration, AES-Fallback

---

## 15. Fehlerbehebung (FAQ)

### Login klappt nicht

- Prüfen: CAPTCHA erforderlich?
- Passwort korrekt? 2FA aktiviert?
- Admin kann Activity Logs prüfen.

### Suche findet verschlüsselte Notizen nicht

- Suchbare Keywords sind optional und standardmäßig deaktiviert.

### Titel verschlüsselt und keine Suche möglich

- Das ist erwartetes Verhalten bei aktivierter Titelverschlüsselung.

### Wikilink zeigt rote Markierung

- Ziel-Notiz existiert noch nicht.

---

## 16. Technische Dokumentation in einfacher Sprache

### 16.1 Architektur (grob)

xelanote ist ein klassisches Client-Server-System:

```
Browser (Frontend)  <—HTTP/JSON—>  Go-Server (Backend)  <—SQL—>  SQLite DB
```

- **Frontend**: Zeigt die Oberfläche, sendet API-Requests.
- **Backend**: Verarbeitet Requests, prüft Berechtigungen, speichert Daten.
- **Datenbank**: Speichert Notizen, Ordner, Links, Benutzer, Einstellungen.

### 16.2 Tech Stack (einfach erklärt)

- **Go**: Programmiersprache für den Server. Schnell, zuverlässig, eine Datei als Binary.
- **SvelteKit/TypeScript**: Framework für die UI. TypeScript hilft, Fehler früh zu finden.
- **SQLite**: Datenbank in einer Datei, kein extra Server nötig.
- **FTS5**: Erweiterung für schnelle Volltextsuche.

### 16.3 Datenfluss (Beispiel: Notiz speichern)

1. Benutzer tippt im Editor.
2. Frontend schickt `PUT /api/notes/:id` mit JSON-Inhalt.
3. Backend prüft Token und `If-Match` (ETag) gegen Konflikte.
4. Backend speichert Notiz in SQLite.
5. Backend aktualisiert Links und Suchindex.
6. Frontend zeigt den neuen Stand.

### 16.4 Wichtige Backend-Schichten

- **API Layer**: Nimmt HTTP-Requests entgegen, gibt JSON zurück.
- **Service Layer**: Enthält die Geschäftslogik (z. B. Link-Updates).
- **DB Layer**: Enthält reine SQL-Operationen.

### 16.5 Datenbankschema (vereinfacht)

Wichtige Tabellen:

- `users`: Benutzerkonten
- `notes`: Notizen (Titel, Inhalt, Metadaten)
- `folders`: Ordnerstruktur
- `links`: Beziehungen zwischen Notizen (Wikilinks)
- `tags` + `note_tags`: Tagging
- `versions`: Version History
- `refresh_tokens`: Session-Management

### 16.6 API (REST)

Die API ist eine klassische REST-API mit JSON:

- `GET /api/notes` → Liste der Notizen
- `POST /api/notes` → neue Notiz
- `PUT /api/notes/:id` → Notiz aktualisieren
- `DELETE /api/notes/:id` → Notiz löschen

**Auth:** JWT im `Authorization` Header oder Cookie.

### 16.7 WebSocket (Live Updates)

- Endpoint: `/api/ws?token=...`
- Server sendet Events wie `note.updated`.
- Frontend aktualisiert Listen automatisch.

### 16.8 Verschlüsselung (E2E, v2)

- **Argon2id**: leitet Schlüssel aus Passwort ab.
- **XChaCha20-Poly1305**: verschlüsselt Notizen.
- **KEK/DEK**: Master-Key schützt einzelne Notiz-Schlüssel.

**Wichtig:** Schlüssel bleiben im Browser, der Server sieht nie Klartext.

### 16.9 Deployment (einfach)

**Lokale Entwicklung:**

- Backend: `make run-backend`
- Frontend: `make run-frontend`

**Docker:**

- `docker compose up -d --build`

**Wichtige Umgebungsvariablen:**

- `JWT_SECRET` (Pflicht)
- `XELANOTE_DB` (Datenbankpfad)
- `XELANOTE_ENV` (development/production)
- `CORS_ALLOWED_ORIGINS` (für Production)
- optional: `TURNSTILE_*`, `XELANOTE_DB_KEY`

### 16.10 Codebase-Lesefahrplan (fuer Anfaenger)

1. **Lies zuerst die README** und starte die App lokal, damit du sie nutzen kannst.
2. **Nimm dir eine Funktion** (z. B. Notiz speichern) und folge dem Weg von UI → API → DB.
3. **Suche nach dem API-Endpoint** in `backend/internal/api/`.
4. **Springe in den Service** in `backend/internal/service/` und schaue, welche DB-Funktionen gerufen werden.
5. **Pruefe das SQL** in `backend/internal/db/`.
6. **Gehe zur UI** in `frontend/src/routes/` und `frontend/src/lib/`.
7. **Wenn es um Editor oder Markdown geht**, schau in `frontend/src/lib/editor/`.

### 16.11 Backend-Ordner erklaert

- `backend/cmd/server/main.go`: Startpunkt. Laedt Konfiguration, setzt den HTTP-Server auf.
- `backend/internal/api/`: HTTP-Handler und Routing.
  - `notes.go`, `folders.go`, `search.go` etc. sind die eigentlichen API-Endpunkte.
  - `middleware.go` prueft Auth, CORS, Logging.
- `backend/internal/service/`: Geschaeftslogik (der "kluge" Teil).
  - Beispiel: Notiz speichern + Links neu berechnen.
- `backend/internal/db/`: Reine SQL-Operationen und Datenbankzugriff.
  - `migrations/` enthaelt Datenbank-Migrationen.
- `backend/internal/parser/`: Wikilink-Parser.
- `backend/internal/websocket/`: Realtime Updates (note.created, note.updated).
- `backend/internal/jobs/` und `backend/internal/cache/`: Hintergrundjobs und Cache (Performance).

### 16.12 Frontend-Ordner erklaert

- `frontend/src/routes/`: Seiten (z. B. `login`, `note/[id]`, `settings`).
- `frontend/src/lib/components/`: UI-Bausteine wie `Editor.svelte`, `Sidebar.svelte`.
- `frontend/src/lib/stores/`: Zentrale Zustandsverwaltung (Notizen, Auth, Settings).
- `frontend/src/lib/api.ts`: API-Wrapper fuer Requests.
- `frontend/src/lib/editor/`: CodeMirror- und Markdown-spezifische Logik.
- `frontend/src-electron/` und `frontend/src-tauri/`: Desktop-App-Code.

### 16.13 Feature-Wegweiser (Beispiele)

- **Notizen (CRUD):**
  - Backend: `backend/internal/api/notes.go` → `backend/internal/service/notes.go` → `backend/internal/db/notes.go`
  - Frontend: `frontend/src/lib/api.ts`, `frontend/src/lib/stores/notes.svelte.ts`, `frontend/src/routes/note/[id]/+page.svelte`
  - UI: `frontend/src/lib/components/Editor.svelte`
- **Wikilinks:**
  - Frontend: `frontend/src/lib/editor/wikilink-autocomplete.ts`, `frontend/src/lib/editor/markdown.ts`
  - Backend: `backend/internal/parser/wikilink.go`, `backend/internal/db/links.go`
- **Suche:**
  - Backend: `backend/internal/api/search.go`, `backend/internal/db/search.go`
  - Frontend: `frontend/src/lib/stores/search.svelte.ts`
- **2FA:**
  - Backend: `backend/internal/api/twofa.go`, `backend/internal/service/twofa.go`, `backend/internal/db/twofa.go`
  - Frontend: `frontend/src/lib/components/TwoFactorSetup.svelte`, `frontend/src/routes/login/+page.svelte`

### 16.14 Tests (wo sie liegen)

- Backend: `backend/internal/db/*_test.go`, `backend/internal/parser/*_test.go`, `backend/internal/service/*_test.go`.
- Frontend: `frontend/tests/e2e/*` (Playwright), `frontend/src/lib/**/*.test.ts`.

### 16.15 Schritt-fuer-Schritt: Notiz speichern (mit Codepfad)

Dieser Ablauf ist bewusst kleinteilig erklaert, damit du als Anfaenger den Weg durch die Codebase nachvollziehen kannst.

#### Uebersichtsdiagramm (Speicher-Flow)

```
Editor (UI)
  -> Notes Store (saveNote)
  -> API Client (PUT /notes/:id, If-Match)
  -> Backend API (updateNote)
  -> Service Layer (UpdateNote + Links)
  -> DB Layer (UPDATE notes ...)
  <- JSON Response (neue Version)
  <- UI aktualisiert
```

#### 1) UI-Event (Editor)

Datei: `frontend/src/lib/components/Editor.svelte`

```svelte
async function handleSave() {
	await notes.saveNote();
}
```

Der Editor ruft die Save-Funktion im Notes-Store auf. Der Editor selbst speichert nicht, sondern delegiert.

#### 2) Store-Logik (Notes Store)

Datei: `frontend/src/lib/stores/notes.svelte.ts`

```ts
const { encryptedTitle, encryptedContent, keywords } = encryption.encryptNote(
	currentNote.title,
	currentNote.content
);

const updated = await api.updateNote(currentNote.id, payload, currentNote.version);
```

Hier passiert das meiste:
- Wenn Verschluesselung aktiv ist, wird **vor dem Senden** verschluesselt.
- Es wird ein Payload gebaut und an die API geschickt.
- `currentNote.version` wird genutzt, um Konflikte zu erkennen (Optimistic Locking).

#### 3) API-Client (HTTP Request)

Datei: `frontend/src/lib/api.ts`

```ts
return request(`/notes/${id}`, {
	method: 'PUT',
	headers: { 'If-Match': version.toString() },
	body: JSON.stringify(data)
});
```

Die Funktion baut den HTTP-Request. `If-Match` ist wichtig: Es verhindert das Ueberschreiben, wenn jemand anders schon gespeichert hat.

#### 4) Backend-API (Request Handling)

Datei: `backend/internal/api/notes.go`

```go
ifMatch := r.Header.Get("If-Match")
version, err := strconv.Atoi(ifMatch)
note, err = s.noteService.UpdateNote(userID, id, req.Title, req.Content, req.FolderPath, version)
```

Das Backend liest den `If-Match` Header, prueft ihn und ruft den Service auf. Bei Versions-Konflikten kommt HTTP 409.

#### 5) Service-Layer (Geschaeftslogik)

Datei: `backend/internal/service/notes.go`

```go
note, err := s.db.UpdateNote(userID, id, title, content, folderPath, version)
if err := s.updateLinks(userID, id, content); err != nil { ... }
```

Der Service ruft die DB-Funktion auf und verarbeitet danach Links (Wikilinks/Backlinks). Ausserdem werden Snapshots fuer Versionen erstellt.

#### 6) DB-Layer (SQL Update)

Datei: `backend/internal/db/notes.go`

```go
UPDATE notes
SET title = ?, title_norm = ?, content = ?, version = version + 1, updated_at = ?
WHERE id = ? AND user_id = ? AND version = ? AND is_deleted = 0
```

Wichtig ist die Version im `WHERE`: Wenn die Version nicht passt, kommt **ErrVersionMismatch** zurueck.

#### 7) Rueckweg zum Client

- Das Backend gibt die aktualisierte Notiz als JSON zurueck.
- Der Store entschluesselt sie (falls noetig) und aktualisiert die UI.

#### Komponentenkarte (wo was passiert)

```
frontend/src/lib/components/Editor.svelte
  -> frontend/src/lib/stores/notes.svelte.ts
  -> frontend/src/lib/api.ts
backend/internal/api/notes.go
  -> backend/internal/service/notes.go
  -> backend/internal/db/notes.go
```

#### Diagramm: Auth + 2FA (vereinfacht)

```
Login (Passwort)
  -> Server prueft Passwort
  -> Falls 2FA aktiv: TOTP-Code abfragen
  -> Bei Erfolg: Access Token (kurz) + Refresh Token (lang)
  -> Client nutzt Access Token fuer API-Requests
  -> Refresh Token erneuert Access Token bei Ablauf
```

#### Diagramm: Ende-zu-Ende-Verschluesselung (vereinfacht)

```
Passwort + Salt (vom Server)
  -> KDF (Argon2id) erzeugt Master-Key (KEK)
  -> Jede Notiz bekommt eigenen DEK
  -> DEK verschluesselt Notizinhalt
  -> KEK verschluesselt DEK
  -> Server speichert nur Ciphertext + Wrapped DEK
```

#### Diagramm: Wikilink aufloesen + Backlinks (vereinfacht)

```
Editor speichert Inhalt mit [[Notiz-Titel]]
  -> Backend parst Wikilinks (parser/wikilink.go)
  -> Links werden in DB geschrieben (links table)
  -> Wenn Ziel existiert: resolved link
  -> Wenn Ziel fehlt: unresolved link
  -> Backlinks fuer Ziel werden aus links berechnet
  -> UI zeigt Backlinks unter der Notiz
```

#### Diagramm: Notiz laden (vereinfacht)

```
Route /note/[id]
  -> Page laedt Editor
  -> Notes Store loadNote(id)
  -> API Client GET /notes/:id
  -> Backend liefert Note (evtl. verschluesselt)
  -> Store entschluesselt (falls noetig)
  -> UI zeigt Titel, Inhalt, Backlinks
```

### 16.16 Schritt-fuer-Schritt: Wikilink erstellen

Diese Uebung zeigt den Weg eines Wikilinks vom Editor bis in die Datenbank.

#### 1) Im Editor tippen

Du tippst im Editor `[[` und suchst eine Notiz aus der Autovervollstaendigung.

Relevanter Code:
- `frontend/src/lib/editor/wikilink-autocomplete.ts`
- `frontend/src/lib/editor/markdown.ts`

Beispiel (Autocomplete Trigger):

```ts
// Trigger-Pattern fuer [[...]]
const match = /\[\[([^\]|]*)$/.exec(textBefore);
```

#### 2) Notiz speichern

Beim Speichern geht der normale Save-Flow los (siehe 16.15). Der Inhalt enthaelt jetzt den Wikilink-Text.

Relevanter Code:
- `frontend/src/lib/stores/notes.svelte.ts`
- `frontend/src/lib/api.ts`
- `backend/internal/api/notes.go`

#### 3) Backend parst Wikilinks

Nach dem Update ruft der Service den Parser auf und schreibt Links in die DB.

Relevanter Code:
- `backend/internal/service/notes.go` (updateLinks)
- `backend/internal/parser/wikilink.go`
- `backend/internal/db/links.go`

Beispiel (Parser-Aufruf):

```go
links := parser.ExtractWikilinks(content)
```

Beispiel (Links speichern):

```go
err := db.SetLinks(userID, noteID, links)
```

#### 4) Ergebnis in der UI

Die UI zeigt die Verknuepfung direkt im Editor und in der Vorschau. Backlinks werden geladen und unter der Notiz angezeigt.

Relevanter Code:
- `frontend/src/lib/components/Editor.svelte`
- `frontend/src/lib/stores/notes.svelte.ts` (getBacklinks)

Beispiel (Backlinks laden):

```ts
const result = await api.getBacklinks(id);
currentNoteBacklinks = result.backlinks;
```

### 16.17 Datenbank-Schema lesen (fuer Anfaenger)

Diese Sektion hilft dir, das Datenbankschema zu verstehen, ohne SQL-Profi zu sein.

#### 1) Einstieg: schema.sql

Datei: `backend/internal/db/schema.sql`

Hier stehen die wichtigsten Tabellen des Grundsystems, zum Beispiel:
- `notes` (Notizen)
- `links` und `unresolved_links` (Wikilinks)
- `tags` und `note_tags` (Tags)
- `notes_fts` (Volltextsuche)

Beispiel (Tabelle `notes`):

```sql
CREATE TABLE IF NOT EXISTS notes (
    id TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    folder_path TEXT DEFAULT '/',
    version INTEGER NOT NULL DEFAULT 1
);
```

**Wie lesen?**
- Spalten sind die Felder einer Notiz.
- `id` ist die UUID fuer die API.
- `version` dient fuer Konfliktschutz beim Speichern.

#### 2) Migrationen verstehen

Dateien: `backend/internal/db/migrations/*.sql`

Migrationen sind kleine Schritte, die das Schema weiterentwickeln. Beispiele:
- `004_add_users.sql` legt Benutzer und Refresh Tokens an.
- `002_folders_table.sql` legt die Ordner-Tabelle an.
- `011_note_versions.sql` legt Versionen fuer Notizen an.
- `017_activity_logs.sql` legt das Audit-Log an.

Beispiel (Benutzer + Tokens):

```sql
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    token TEXT UNIQUE NOT NULL,
    user_id INTEGER NOT NULL
);
```

**Wie lesen?**
- `FOREIGN KEY` zeigt Beziehungen zwischen Tabellen.
- `ON DELETE CASCADE` bedeutet: Wenn ein User geloescht wird, verschwinden seine Daten mit.

#### 3) Typische Beziehungen (vereinfacht)

```
users -> notes -> links
notes -> note_versions
users -> refresh_tokens
```

#### 4) Wo finde ich die SQL-Queries im Code?

- `backend/internal/db/notes.go` fuer Notizen
- `backend/internal/db/links.go` fuer Wikilinks
- `backend/internal/db/search.go` fuer Suche
- `backend/internal/db/users.go` fuer Benutzer


---

## 17. Glossar

- **HTTP**: Protokoll fuer Kommunikation zwischen Browser und Server.
- **Request/Response**: Anfrage an den Server / Antwort vom Server.
- **Endpoint**: Eine konkrete URL-Funktion der API (z. B. `GET /api/notes`).
- **REST**: Stil fuer Web-APIs mit klaren URLs und HTTP-Methoden.
- **JSON**: Textformat fuer strukturierte Daten (zwischen Client und Server).
- **Token**: Digitaler Ausweis fuer Anfragen (z. B. JWT).
- **Cookie**: Kleine Daten im Browser, die automatisch mitgesendet werden.
- **Auth**: Abkuerzung fuer Authentifizierung (Login/Identitaet pruefen).
- **CORS**: Browser-Sicherheitsregel fuer Zugriffe auf andere Domains.
- **SQL**: Sprache fuer Datenbankabfragen.
- **Schema**: Struktur/Tabellen-Definition einer Datenbank.
- **Migration**: Skript, das das Schema zwischen Versionen veraendert.
- **WebSocket**: Dauerhafte Verbindung fuer Live-Updates.
- **Hash**: Einweg-Umwandlung fuer sichere Speicherung (z. B. Passwoerter).
- **Salt**: Zusaetzliche Zufallsdaten fuer sichere Hashes/Schluessel.
- **Base64**: Kodierung fuer Binardaten als Text.
- **API**: Schnittstelle zwischen Frontend und Backend.
- **Backend**: Server-Teil einer Anwendung.
- **Frontend**: Benutzeroberfläche im Browser.
- **Go**: Programmiersprache fuer das Backend.
- **TypeScript**: JavaScript mit Typen fuer weniger Fehler.
- **SvelteKit**: Web-Framework fuer das Frontend.
- **JWT**: JSON Web Token, ein signierter Login-Token.
- **ETag**: Versionskennzeichnung zur Konfliktvermeidung.
- **FTS5**: SQLite-Volltextsuche.
- **SQLite**: Datenbank in einer Datei, ohne separaten Server.
- **E2E**: Ende-zu-Ende-Verschlüsselung.
- **KDF**: Key Derivation Function (z. B. Argon2id).
- **KEK/DEK**: Schlüsselhierarchie bei Verschlüsselung.
- **TOTP**: Time-based One-Time Password (2FA Codes).

---

### Weiterführende Dokumente

- `docs/architecture.md`
- `docs/api.md`
- `docs/development.md`
- `docs/e2e-encryption.md`
- `docs/authentication.md`
- `docs/admin-panel.md`
- `docs/markdown-guide.md`
