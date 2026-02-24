# Testing Manifest – Xelanote Frontend

Vollständige Übersicht aller zu testenden Seiten, Komponenten und User-Flows.

## Routen & Seiten

### Öffentliche Routen (keine Authentifizierung)

| Route                     | Beschreibung              | Priorität | Visual | E2E | A11y | Design | Vision |
| ------------------------- | ------------------------- | --------- | ------ | --- | ---- | ------ | ------ |
| `/login`                  | Login-Seite mit 2FA       | Critical  | ✅     | ✅  | ✅   | ✅     | ⬜     |
| `/register`               | Benutzer-Registrierung    | Critical  | ✅     | ✅  | ✅   | ✅     | ⬜     |
| `/about`                  | Info-Seite                | Low       | ✅     | ✅  | ✅   | ✅     | ⬜     |
| `/shared/collection/[id]` | Geteilte Rezeptsammlungen | Medium    | ✅     | ✅  | ✅   | ✅     | ⬜     |
| `/shared/folder/[id]`     | Geteilte Ordner           | Medium    | ✅     | ✅  | ✅   | ✅     | ⬜     |

### Geschützte Routen (Authentifizierung erforderlich)

| Route                  | Beschreibung     | Priorität | Visual | E2E | A11y | Design | Vision |
| ---------------------- | ---------------- | --------- | ------ | --- | ---- | ------ | ------ |
| `/`                    | Dashboard / Home | Critical  | ✅     | ✅  | ✅   | ✅     | ✅     |
| `/note/[id]`           | Notiz-Editor     | Critical  | ✅     | ✅  | ✅   | ✅     | ⬜     |
| `/search`              | Volltextsuche    | High      | ✅     | ✅  | ✅   | ✅     | ✅     |
| `/journal`             | Tagebuch         | High      | ✅     | ✅  | ✅   | ✅     | ✅     |
| `/graph`               | Wissensgraph     | Medium    | ✅     | ✅  | ✅   | ✅     | ✅     |
| `/recipes`             | Rezepte          | High      | ✅     | ✅  | ✅   | ✅     | ✅     |
| `/due-dates`           | Fälligkeiten     | Medium    | ✅     | ✅  | ✅   | ✅     | ⬜     |
| `/trash`               | Papierkorb       | Medium    | ✅     | ✅  | ✅   | ✅     | ⬜     |
| `/settings`            | Einstellungen    | High      | ✅     | ✅  | ✅   | ✅     | ✅     |
| `/settings/encryption` | Verschlüsselung  | High      | ✅     | ✅  | ✅   | ✅     | ⬜     |
| `/settings/migration`  | Datenmigration   | Low       | ✅     | ✅  | ✅   | ✅     | ⬜     |

> **Vision** = KI-gestützte Design-Bewertung via Claude Vision (`npm run test:design:vision`).
> Benötigt `claude` CLI mit Claude Max Abo. Wird automatisch übersprungen wenn nicht verfügbar.

### Admin-Routen

| Route    | Beschreibung    | Priorität | Visual | E2E | A11y | Design |
| -------- | --------------- | --------- | ------ | --- | ---- | ------ |
| `/admin` | Admin-Dashboard | Low       | ⬜     | ⬜  | ⬜   | ⬜     |

## Kritische User-Flows

### Authentication (Critical)

1. Registrierung → Login → Dashboard
2. Login mit 2FA (TOTP)
3. Login mit Backup-Code
4. Session-Timeout → Auto-Redirect zu `/login`
5. Logout → Redirect zu `/login`

### Notes CRUD (Critical)

1. Notiz erstellen → Titel eingeben → Inhalt schreiben → Auto-Save
2. Notiz öffnen → Bearbeiten → Speichern
3. Notiz in Ordner verschieben
4. Notiz löschen → Papierkorb → Wiederherstellen
5. Notiz suchen → Ergebnis klicken → Editor öffnet

### Ordner-Management (High)

1. Ordner erstellen → Benennen
2. Ordner umbenennen
3. Ordner verschachteln (Drag & Drop)
4. Ordner löschen (mit Bestätigung)
5. Ordner farblich markieren

### Editor (Critical)

1. Markdown eingeben → Live-Vorschau
2. Toolbar-Aktionen (Bold, Italic, Heading, Link, etc.)
3. Wikilinks `[[...]]` → Autocompletion
4. Tabellen einfügen
5. Task-Listen erstellen und abhaken
6. Bild hochladen

### Rezepte (High)

1. Rezept erstellen → Metadaten ausfüllen → Speichern
2. Rezept importieren (URL)
3. Rezeptsammlung erstellen → Rezepte hinzufügen
4. Rezept-Vorschau
5. Rezept teilen

### Einstellungen (High)

1. Theme wechseln (Light ↔ Dark)
2. Sprache ändern (DE ↔ EN)
3. 2FA aktivieren/deaktivieren
4. API-Key erstellen
5. Verschlüsselung aktivieren

## Komponenten-Testmatrix

### Dialoge

| Komponente             | States                    | Viewports       |
| ---------------------- | ------------------------- | --------------- |
| CreateNoteDialog       | Default, Validierung      | Desktop, Mobile |
| CreateFolderDialog     | Default, Validierung      | Desktop, Mobile |
| DeleteFolderDialog     | Default, Bestätigung      | Desktop, Mobile |
| RenameFolderDialog     | Default, Validierung      | Desktop, Mobile |
| RenameNoteDialog       | Default, Validierung      | Desktop, Mobile |
| MoveToFolderDialog     | Default, Ordnerliste      | Desktop, Mobile |
| ColorPickerDialog      | Default, Farbe gewählt    | Desktop, Mobile |
| FeedbackDialog         | Default, Gesendet         | Desktop, Mobile |
| ConflictDialog         | Default, Diff-Ansicht     | Desktop, Mobile |
| VersionHistoryDialog   | Default, Version gewählt  | Desktop, Mobile |
| TableInsertDialog      | Default, Größe gewählt    | Desktop, Mobile |
| RecipeImportDialog     | Default, URL eingegeben   | Desktop, Mobile |
| RecipeCollectionDialog | Default, Sammlung gewählt | Desktop, Mobile |
| AITransformDialog      | Default, Transformation   | Desktop, Mobile |

### Interaktive Elemente

| Komponente      | States                            |
| --------------- | --------------------------------- |
| Editor          | Leer, Mit Inhalt, Fokus, Vorschau |
| Sidebar         | Offen, Geschlossen, Mobile        |
| MobileBottomNav | Default, Aktiver Tab              |
| ThemeSelector   | Light, Dark                       |
| QuickSwitcher   | Offen, Suche, Ergebnis            |
| FilterBar       | Default, Filter aktiv             |
| TagEditor       | Default, Tags vorhanden           |
| Toast           | Success, Error, Warning, Info     |

## Viewports

| Name             | Breite | Höhe | Geräte-Typ    |
| ---------------- | ------ | ---- | ------------- |
| Desktop          | 1440   | 900  | Desktop       |
| Tablet Portrait  | 768    | 1024 | iPad          |
| Tablet Landscape | 1024   | 768  | iPad quer     |
| Mobile           | 393    | 852  | iPhone 14 Pro |

## Theme-Varianten

| Theme         | Klasse                | Variante |
| ------------- | --------------------- | -------- |
| Gruvbox Light | `theme-gruvbox-light` | `light`  |
| Gruvbox Dark  | `theme-gruvbox-dark`  | `dark`   |

## Sprachen

| Locale   | Datei                     |
| -------- | ------------------------- |
| Deutsch  | `src/lib/locales/de.json` |
| Englisch | `src/lib/locales/en.json` |
