# API-Referenz

Alle Endpunkte unter `/api/`. Antworten sind JSON. Auth via HttpOnly Cookie oder `Authorization: Bearer <token>`.

## Authentifizierung

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `POST` | `/auth/register` | Neuen User registrieren | Nein |
| `POST` | `/auth/login` | Login (Username + Passwort) | Nein |
| `POST` | `/auth/refresh` | Access Token erneuern | Cookie |
| `POST` | `/auth/logout` | Logout (Tokens widerrufen) | Cookie |
| `GET` | `/auth/me` | Aktuellen User abrufen | Ja |
| `POST` | `/auth/fido2/begin` | FIDO2-Login starten | Nein |
| `POST` | `/auth/fido2/finish` | FIDO2-Login abschließen | Nein |
| `POST` | `/auth/recovery/verify` | Recovery Key prüfen | Nein |
| `GET` | `/auth/recovery/encrypted-deks` | Recovery-Wrapper für verschlüsselte Notizen laden | Nein |
| `POST` | `/auth/recovery/reset-password` | Legacy-Reset (nur unverschlüsselte Accounts) | Nein |
| `POST` | `/auth/recovery/reset-password-v2` | Tokenisierter Reset mit DEK-Rewrap | Nein |

## Zwei-Faktor-Authentifizierung

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/2fa/status` | 2FA-Status abfragen |
| `POST` | `/2fa/setup` | TOTP-Secret generieren + QR-Code |
| `POST` | `/2fa/verify` | TOTP-Code verifizieren (Setup abschließen) |
| `DELETE` | `/2fa/` | 2FA deaktivieren |
| `POST` | `/2fa/backup-codes/regenerate` | Neue Backup-Codes generieren |
| `POST` | `/2fa/fido2/register/begin` | FIDO2-Key registrieren (Start) |
| `POST` | `/2fa/fido2/register/finish` | FIDO2-Key registrieren (Abschluss) |
| `GET` | `/2fa/fido2/credentials` | FIDO2-Keys auflisten |
| `DELETE` | `/2fa/fido2/credentials/{id}` | FIDO2-Key entfernen |

## Notizen

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/notes` | Alle Notizen auflisten |
| `POST` | `/notes` | Neue Notiz erstellen |
| `GET` | `/notes/titles` | Nur Titel aller Notizen |
| `GET` | `/notes/titles/ai-enabled` | Titel der KI-aktivierten Notizen |
| `GET` | `/notes/{id}` | Einzelne Notiz laden |
| `PUT` | `/notes/{id}` | Notiz aktualisieren |
| `DELETE` | `/notes/{id}` | Notiz in Papierkorb verschieben |
| `POST` | `/notes/{id}/rename` | Notiz umbenennen (Async-Job) |
| `GET` | `/notes/{id}/backlinks` | Backlinks der Notiz |
| `PUT` | `/notes/{id}/color` | Notiz-Farbe setzen |
| `GET` | `/notes/{id}/user-state` | Cursor/Scroll-Position laden |
| `PUT` | `/notes/{id}/user-state` | Cursor/Scroll-Position speichern |
| `GET` | `/notes/{id}/tags` | Tags der Notiz |
| `PUT` | `/notes/{id}/tags` | Tags setzen |
| `GET` | `/notes/{id}/ai-enabled` | KI-Status abfragen |
| `PUT` | `/notes/{id}/ai-enabled` | KI ein-/ausschalten |
| `POST` | `/notes/{id}/decrypt` | Verschlüsselte Notiz-Links auflösen |
| `POST` | `/notes/batch-reencrypt-deks` | DEKs nach Passwort-Änderung neu verschlüsseln |
| `POST` | `/notes/{id}/task-events` | Checkbox-Event loggen |

## Notiz-Versionen

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/notes/{id}/versions` | Versions-Historie (paginiert) |
| `GET` | `/notes/{id}/versions/compare` | Zwei Versionen vergleichen (Diff) |
| `POST` | `/notes/{id}/versions/{v}/restore` | Version wiederherstellen |
| `POST` | `/notes/{id}/versions/delta-summary` | KI-Zusammenfassung der Änderungen |

## Notiz-KI

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `POST` | `/notes/{id}/summarize` | Zusammenfassung generieren (+ SSE) |
| `POST` | `/notes/{id}/summarize/prepare` | Zusammenfassung vorbereiten |
| `POST` | `/notes/{id}/suggest-tags` | Tag-Vorschläge |
| `POST` | `/notes/{id}/suggest-links` | Link-Vorschläge |
| `POST` | `/notes/{id}/format-markdown` | Markdown formatieren |
| `POST` | `/notes/{id}/ai-transform` | Text transformieren |

## Notiz-Teilen

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/notes/{id}/shares` | Freigaben der Notiz |
| `POST` | `/notes/{id}/shares` | Notiz teilen |
| `PUT` | `/notes/{id}/shares` | Freigabe-Rolle ändern |
| `DELETE` | `/notes/{id}/shares` | Freigabe entfernen |

## Ordner

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/folders` | Alle Ordner auflisten |
| `POST` | `/folders` | Neuen Ordner erstellen |
| `GET` | `/folders/{id}` | Ordner laden |
| `PUT` | `/folders/{id}` | Ordner aktualisieren |
| `DELETE` | `/folders/{id}` | Ordner löschen |
| `PUT` | `/folders/{id}/reorder` | Reihenfolge ändern |
| `PUT` | `/folders/{id}/move` | Ordner verschieben |
| `PUT` | `/folders/{id}/rename` | Ordner umbenennen |
| `PUT` | `/folders/{id}/color` | Ordner-Farbe setzen |
| `GET` | `/folders/{id}/ai-enabled` | KI-Default des Ordners |
| `PUT` | `/folders/{id}/ai-enabled` | KI-Default setzen |
| `GET` | `/folders/{id}/encryption-default` | Verschlüsselungs-Default |
| `PUT` | `/folders/{id}/encryption-default` | Verschlüsselungs-Default setzen |

## Geteilte Inhalte

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/shared` | Mit mir geteilte Notizen |
| `GET` | `/shared/{id}` | Geteilte Notiz laden |
| `PUT` | `/shared/{id}` | Geteilte Notiz bearbeiten |
| `GET` | `/shared/folders` | Geteilte Ordner |
| `GET` | `/shared/folders/{id}/notes` | Notizen in geteiltem Ordner |
| `GET` | `/shared/collections` | Geteilte Sammlungen |
| `POST` | `/shared/collections` | Geteilte Sammlung erstellen |
| `DELETE` | `/shared/collections/{id}` | Geteilte Sammlung entfernen |

## Tags, Templates, Snippets

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/tags` | Alle Tags |
| `DELETE` | `/tags/{id}` | Tag löschen |
| `GET/POST` | `/templates` | Templates CRUD |
| `GET/PUT/DELETE` | `/templates/{id}` | Template bearbeiten |
| `GET/POST` | `/snippets` | Snippets CRUD |
| `GET/PUT/DELETE` | `/snippets/{id}` | Snippet bearbeiten |

## Journal

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/journal/` | Heutigen Eintrag laden |
| `GET` | `/journal/entries` | Alle Einträge (paginiert) |
| `GET` | `/journal/lookup?date=YYYY-MM-DD` | Eintrag für bestimmtes Datum |
| `GET` | `/journal/calendar?year=Y&month=M` | Tage mit Einträgen (Monat) |
| `GET` | `/journal/calendar/year?year=Y` | Jahres-Heatmap |

## Rezepte

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/recipes` | Alle Rezepte |
| `GET` | `/recipes/{id}` | Einzelnes Rezept |
| `PUT` | `/recipes/{id}/metadata` | Metadaten aktualisieren |
| `PUT` | `/recipes/{id}/ingredients` | Zutaten aktualisieren |
| `PUT` | `/recipes/{id}/images` | Bilder verwalten |
| `POST` | `/recipes/suggestions/similar` | Ähnliche Rezepte (KI) |
| `POST` | `/recipes/suggestions/by-ingredients` | Rezepte nach Zutaten (KI) |
| `POST` | `/recipes/save-generated` | KI-generiertes Rezept speichern |
| `POST` | `/recipes/extract-ingredients` | Zutaten aus Text/Bild extrahieren |
| `POST` | `/recipes/import-from-image` | Rezept von Foto importieren |
| `POST` | `/recipes/import-from-url` | Rezept von URL importieren |

## Rezept-Collections

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET/POST` | `/recipes/collections` | Collections CRUD |
| `GET/PUT/DELETE` | `/recipes/collections/{id}` | Collection bearbeiten |
| `POST/DELETE` | `/recipes/collections/{id}/items` | Rezepte hinzufügen/entfernen |

## Einkaufslisten

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET/POST` | `/shopping/lists` | Listen CRUD |
| `GET/PUT/DELETE` | `/shopping/lists/{id}` | Liste bearbeiten |
| `POST` | `/shopping/lists/{id}/archive` | Liste archivieren |
| `POST` | `/shopping/lists/{id}/items` | Item hinzufügen |
| `POST` | `/shopping/lists/{id}/items/batch` | Mehrere Items hinzufügen |
| `PUT` | `/shopping/items/{id}` | Item aktualisieren |
| `DELETE` | `/shopping/items/{id}` | Item löschen |
| `PUT` | `/shopping/items/{id}/check` | Item ab-/anhaken |
| `PUT` | `/shopping/lists/{id}/items/reorder` | Reihenfolge ändern |
| `POST` | `/shopping/lists/{id}/sort` | KI-Sortierung |
| `POST` | `/shopping/lists/{id}/import-recipe` | Rezept-Zutaten importieren |
| `GET/POST/DELETE` | `/shopping/favorites` | Favoriten verwalten |
| `GET/POST/DELETE` | `/shopping/lists/{id}/shares` | Liste teilen |

## Canvas

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET/POST` | `/canvas` | Canvas-Notizen CRUD |
| `GET/PUT/DELETE` | `/canvas/{id}` | Canvas bearbeiten |

## Suche

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/search?q=` | Volltextsuche (FTS5) |
| `GET` | `/quick-search?q=` | Schnellsuche (nur Titel) |

## Weitere Endpunkte

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/graph` | Wissensgraph (Nodes + Edges) |
| `GET` | `/due-dates` | Fällige Aufgaben |
| `GET` | `/export/markdown` | Alle Notizen als Markdown exportieren |
| `POST` | `/import/markdown` | Markdown-Dateien importieren |
| `POST` | `/uploads` | Datei hochladen |
| `GET` | `/uploads/{user}/{file}` | Datei abrufen |
| `GET` | `/ws` | WebSocket-Verbindung |
| `GET` | `/config` | CAPTCHA-Konfiguration |
| `GET` | `/changelog` | Changelog abrufen |
| `GET/PUT` | `/features/{feature}` | Feature-Flags |
| `GET` | `/jobs/{id}` | Job-Status abfragen |
| `POST` | `/llm/spell-check` | Rechtschreibprüfung (KI) |
| `POST` | `/llm/transcribe` | Audio-Transkription (Whisper) |
| `POST` | `/error-reports` | Fehlerbericht senden |

## Admin-Endpunkte (Admin-Rolle nötig)

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/admin/stats` | System-Statistiken |
| `GET` | `/admin/detailed-stats` | Detaillierte Stats + Charts |
| `GET` | `/admin/users` | User-Liste |
| `POST` | `/admin/users` | User erstellen |
| `PUT` | `/admin/users/{id}` | User bearbeiten |
| `DELETE` | `/admin/users/{id}` | User löschen |
| `GET` | `/admin/activity` | Aktivitätsprotokoll |
| `GET/PUT` | `/admin/settings` | System-Einstellungen |

## Papierkorb

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/trash` | Gelöschte Notizen |
| `GET` | `/trash/count` | Anzahl gelöschter Notizen |
| `DELETE` | `/trash` | Papierkorb leeren |
| `POST` | `/notes/{id}/restore` | Notiz wiederherstellen |
| `DELETE` | `/notes/{id}/permanent` | Notiz endgültig löschen |
