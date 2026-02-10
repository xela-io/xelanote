# Admin Panel

Das Admin Panel bietet Administratoren umfassende Kontroll- und Überwachungsfunktionen für die XelaNote-Instanz.

## Inhaltsverzeichnis

- [Überblick](#überblick)
- [Zugriff](#zugriff)
- [Features](#features)
  - [Dashboard](#dashboard)
  - [User Management](#user-management)
  - [Activity Logs](#activity-logs)
  - [Settings](#settings)
- [Datenbank](#datenbank)
- [API Endpoints](#api-endpoints)
- [Sicherheit](#sicherheit)
- [Erste Schritte](#erste-schritte)

---

## Überblick

Das Admin Panel ist eine Administrationsoberfläche, die folgende Funktionen bereitstellt:

- **System-Überwachung**: Echtzeit-Statistiken zu Benutzern, Notizen, Ordnern, Tags und Speicherverbrauch
- **Benutzerverwaltung**: Verwaltung von Benutzerkonten und Admin-Rechten
- **Aktivitätsverfolgung**: Audit-Trail aller relevanten Systemaktivitäten
- **Systemeinstellungen**: Konfiguration von Registrierung, Limits und Wartungsmodus

---

## Zugriff

### Erster Admin-Benutzer

Der erste registrierte Benutzer (niedrigste User-ID) wird **automatisch zum Administrator**. Dies wird durch die Datenbankmigration `016_add_admin_flag.sql` beim ersten Start sichergestellt.

### Admin Panel öffnen

Administratoren haben drei Möglichkeiten, das Admin Panel zu erreichen:

1. **Sidebar-Link**: Das Shield-Icon (🛡️) erscheint in der Sidebar unterhalb von "Einstellungen"
2. **Direkte URL**: Navigation zu `/admin`
3. **Browser-Lesezeichen**: URL-basierter Zugriff möglich

**Hinweis**: Nicht-Administratoren sehen weder das Shield-Icon noch können sie die `/admin`-Route aufrufen (403 Forbidden).

---

## Features

### Dashboard

Das Dashboard bietet einen Gesamtüberblick über die Systemaktivität.

#### System-Statistiken

Zeigt aktuelle Gesamtzahlen:

- **Total Users**: Anzahl registrierter Benutzer
- **Total Notes**: Anzahl aller Notizen (exklusive gelöschte)
- **Total Folders**: Anzahl aller Ordner
- **Total Tags**: Anzahl eindeutiger Tags
- **Storage Used**: Gesamtspeicherverbrauch in MB (Uploads)

#### Zeitreihen-Diagramme

Visuelle Darstellung von Trends:

1. **User Growth**: Tägliche Neuregistrierungen (letzte 30 Tage)
2. **Note Activity**: Tägliche Notizen-Erstellungen (letzte 30 Tage)
3. **Storage Trend**: Entwicklung des Speicherverbrauchs (letzte 30 Tage)

**API-Endpunkt**: `GET /api/admin/stats/detailed`

---

### User Management

Zentrale Verwaltung aller Benutzerkonten.

#### Benutzerliste

Tabellarische Übersicht aller User mit folgenden Informationen:

| Spalte | Beschreibung |
|--------|--------------|
| ID | User ID (eindeutig) |
| Username | Benutzername |
| Email | E-Mail-Adresse |
| Admin | Admin-Status (Badge) |
| Notes | Anzahl Notizen des Users |
| Storage | Speicherverbrauch in MB |
| Created | Registrierungsdatum |
| Actions | Aktionen (Admin toggle, Delete) |

#### Admin-Status ändern

Administratoren können anderen Benutzern Admin-Rechte geben oder entziehen:

1. Toggle-Button in der Actions-Spalte klicken
2. Bestätigung erfolgt automatisch
3. Änderung wird sofort wirksam und im Activity Log erfasst

**Einschränkungen**:
- ❌ Admins können sich selbst NICHT die Admin-Rechte entziehen
- ✅ Beliebig viele Admins möglich (kein Limit)

**API-Endpunkt**: `PUT /api/admin/users/{id}/admin`

#### Benutzer löschen

Admins können Benutzerkonten vollständig löschen:

1. Delete-Button klicken
2. Bestätigungsdialog erscheint
3. Bei Bestätigung: User wird gelöscht

**Was wird gelöscht**:
- User-Account (users-Tabelle)
- Alle Notizen des Users
- Alle Ordner des Users
- Alle Uploads des Users
- Alle Templates des Users
- Alle Snippets des Users
- Refresh Tokens
- Activity Logs (user_id wird auf NULL gesetzt)

**Einschränkungen**:
- ❌ Admins können sich selbst NICHT löschen
- ✅ Keine Wiederherstellung möglich (permanente Löschung)

**API-Endpunkt**: `DELETE /api/admin/users/{id}`

---

### Activity Logs

Vollständiger Audit-Trail aller relevanten Systemaktivitäten.

#### Erfasste Aktivitäten

Das System protokolliert folgende Aktionen:

| Action | Beschreibung | Target Type |
|--------|--------------|-------------|
| `login` | Benutzer-Login | - |
| `logout` | Benutzer-Logout | - |
| `register` | Neue Registrierung | - |
| `note_create` | Notiz erstellt | `note` |
| `note_update` | Notiz bearbeitet | `note` |
| `note_delete` | Notiz gelöscht | `note` |
| `note_restore` | Notiz wiederhergestellt | `note` |
| `folder_create` | Ordner erstellt | `folder` |
| `folder_delete` | Ordner gelöscht | `folder` |
| `user_admin_set` | Admin-Status geändert | `user` |
| `user_delete` | Benutzer gelöscht | `user` |
| `settings_change` | Systemeinstellungen geändert | `settings` |

#### Log-Details

Jeder Eintrag enthält:

- **Timestamp**: Wann die Aktion stattfand
- **User**: Welcher Benutzer die Aktion ausführte
- **Action**: Art der Aktion
- **Target**: Betroffenes Objekt (z.B. Note-Titel, Username)
- **IP Address**: Client-IP für Sicherheitsanalysen
- **User Agent**: Browser/Client-Information

#### Filterung und Pagination

- **Action-Filter**: Dropdown zur Filterung nach Action-Typ
- **Pagination**: 20 Einträge pro Seite (konfigurierbar)
- **Sortierung**: Neueste zuerst (DESC)

**API-Endpunkt**: `GET /api/admin/activity?limit=20&page=1&action=login`

#### Retention-Policy

Alte Logs werden automatisch gelöscht basierend auf der Einstellung `activity_retention_days` (Default: 90 Tage).

---

### Settings

Systemweite Konfigurationsoptionen.

#### Verfügbare Einstellungen

| Setting | Default | Beschreibung |
|---------|---------|--------------|
| `registration_enabled` | `true` | Öffentliche Registrierung erlauben |
| `max_notes_per_user` | `0` | Maximale Anzahl Notizen pro User (0 = unbegrenzt) |
| `max_storage_mb_per_user` | `0` | Maximaler Speicher in MB pro User (0 = unbegrenzt) |
| `maintenance_mode` | `false` | Wartungsmodus aktivieren |
| `activity_retention_days` | `90` | Aufbewahrungszeit für Activity Logs in Tagen |

#### Registrierung deaktivieren

Wenn `registration_enabled = false`:
- Die `/register`-Route gibt 403 Forbidden zurück
- Registrierungsformular wird deaktiviert
- Neue Benutzer können nur noch manuell angelegt werden (zukünftiges Feature)

#### Limits setzen

**Notes-Limit**:
```json
{
  "max_notes_per_user": "100"
}
```
Bei Erreichen des Limits schlägt `POST /api/notes` mit 403 Forbidden fehl.

**Storage-Limit**:
```json
{
  "max_storage_mb_per_user": "500"
}
```
Bei Überschreitung schlägt `POST /api/uploads` mit 413 Payload Too Large fehl.

#### Wartungsmodus

Wenn `maintenance_mode = true`:
- Alle API-Endpunkte (außer `/health` und `/api/auth/*`) geben 503 Service Unavailable zurück
- Admin-Endpoints bleiben verfügbar
- Benutzer sehen eine Wartungsmeldung

#### Einstellungen ändern

1. Wert im Formular anpassen
2. "Save Settings" klicken
3. Änderungen werden validiert und gespeichert
4. Activity Log erfasst alle geänderten Keys

**API-Endpunkt**: `PUT /api/admin/settings`

**Request**:
```json
{
  "registration_enabled": "false",
  "max_notes_per_user": "100"
}
```

**Response**: Alle aktuellen Settings
```json
{
  "registration_enabled": "false",
  "max_notes_per_user": "100",
  "max_storage_mb_per_user": "0",
  "maintenance_mode": "false",
  "activity_retention_days": "90"
}
```

---

## Datenbank

### Neue Tabellen

#### users.is_admin (Migration 016)

Neue Spalte in der `users`-Tabelle:

```sql
ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_users_is_admin ON users(is_admin);
```

- `is_admin = 1`: Administrator
- `is_admin = 0`: Normaler Benutzer

Der erste User wird automatisch zum Admin:
```sql
UPDATE users SET is_admin = 1 WHERE id = (SELECT MIN(id) FROM users);
```

#### activity_logs (Migration 017)

Neue Tabelle für Audit-Trail:

```sql
CREATE TABLE activity_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    action TEXT NOT NULL,
    target_type TEXT,
    target_id TEXT,
    details TEXT,
    ip_address TEXT,
    user_agent TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);
```

**Indizes**:
- `idx_activity_logs_user`: Schnelle User-Abfragen
- `idx_activity_logs_action`: Action-Filterung
- `idx_activity_logs_created`: Sortierung nach Datum
- `idx_activity_logs_target`: Target-basierte Suche

**Besonderheit**: `user_id` wird auf `NULL` gesetzt wenn der User gelöscht wird (Logs bleiben erhalten).

#### system_settings (Migration 018)

Neue Tabelle für System-Konfiguration:

```sql
CREATE TABLE system_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT DEFAULT (datetime('now'))
);
```

**Default-Werte**:
```sql
INSERT INTO system_settings (key, value) VALUES
    ('registration_enabled', 'true'),
    ('max_notes_per_user', '0'),
    ('max_storage_mb_per_user', '0'),
    ('maintenance_mode', 'false'),
    ('activity_retention_days', '90');
```

---

## API Endpoints

Alle Admin-Endpunkte sind unter `/api/admin/*` gruppiert und erfordern:
1. Gültige Authentifizierung (JWT oder Cookie)
2. Admin-Status (`is_admin = 1`)

### Statistiken

#### GET /api/admin/stats

Basis-Systemstatistiken abrufen.

**Response**:
```json
{
  "total_users": 10,
  "total_notes": 250,
  "total_folders": 35,
  "total_tags": 42,
  "storage_used_mb": 125.5
}
```

#### GET /api/admin/stats/detailed

Detaillierte Statistiken mit Zeitreihen.

**Response**:
```json
{
  "stats": {
    "total_users": 10,
    "total_notes": 250,
    "total_folders": 35,
    "total_tags": 42,
    "storage_used_mb": 125.5
  },
  "user_growth": [
    { "date": "2026-01-15", "count": 2 },
    { "date": "2026-01-16", "count": 1 }
  ],
  "note_growth": [
    { "date": "2026-01-15", "count": 15 },
    { "date": "2026-01-16", "count": 22 }
  ],
  "storage_trend": [
    { "date": "2026-01-15", "value": 120.0 },
    { "date": "2026-01-16", "value": 125.5 }
  ]
}
```

---

### Benutzerverwaltung

#### GET /api/admin/users

Alle Benutzer mit Statistiken abrufen.

**Response**:
```json
[
  {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "is_admin": true,
    "note_count": 50,
    "storage_mb": 25.5,
    "created_at": "2026-01-01T10:00:00Z"
  },
  {
    "id": 2,
    "username": "user1",
    "email": "user1@example.com",
    "is_admin": false,
    "note_count": 10,
    "storage_mb": 5.2,
    "created_at": "2026-01-02T14:30:00Z"
  }
]
```

#### GET /api/admin/users/{id}

Details zu einem einzelnen Benutzer abrufen.

**Response**:
```json
{
  "id": 2,
  "username": "user1",
  "email": "user1@example.com",
  "is_admin": false,
  "note_count": 10,
  "storage_mb": 5.2,
  "created_at": "2026-01-02T14:30:00Z"
}
```

**Error Codes**:
- `404`: User nicht gefunden
- `400`: Ungültige User-ID

#### PUT /api/admin/users/{id}/admin

Admin-Status eines Benutzers ändern.

**Request**:
```json
{
  "is_admin": true
}
```

**Response**: `204 No Content`

**Error Codes**:
- `403`: Versuch, eigenen Admin-Status zu entfernen
- `404`: User nicht gefunden
- `400`: Ungültige Request-Daten

**Sicherheit**: Es wird im Activity Log erfasst, wer wann welchem User Admin-Rechte gegeben/entzogen hat.

#### DELETE /api/admin/users/{id}

Benutzer vollständig löschen.

**Response**: `204 No Content`

**Error Codes**:
- `403`: Versuch, sich selbst zu löschen
- `404`: User nicht gefunden

**Sicherheit**: Aktion wird im Activity Log erfasst bevor der User gelöscht wird.

---

### Activity Logs

#### GET /api/admin/activity

Activity Logs mit Filterung und Pagination abrufen.

**Query Parameters**:

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `limit` | int | Einträge pro Seite (1-100, default: 50) |
| `page` | int | Seitennummer (default: 1) |
| `action` | string | Action-Typ filtern |
| `user_id` | int | Nach User-ID filtern |
| `target_type` | string | Target-Typ filtern |
| `date_from` | string | Startdatum (ISO 8601) |
| `date_to` | string | Enddatum (ISO 8601) |

**Beispiel**:
```
GET /api/admin/activity?limit=20&page=1&action=login&date_from=2026-01-15
```

**Response**:
```json
{
  "logs": [
    {
      "id": 123,
      "user_id": 2,
      "username": "user1",
      "action": "login",
      "target_type": null,
      "target_id": null,
      "details": "",
      "ip_address": "192.168.1.10",
      "user_agent": "Mozilla/5.0...",
      "created_at": "2026-01-16T10:30:00Z"
    }
  ],
  "total": 450
}
```

**Details-Feld**: JSON mit zusätzlichen Informationen (abhängig von Action-Typ):

- `user_admin_set`: `{"target_username": "user1", "is_admin": true}`
- `user_delete`: `{"deleted_username": "user1"}`
- `settings_change`: `{"changed_keys": ["registration_enabled"]}`

---

### Systemeinstellungen

#### GET /api/admin/settings

Alle Systemeinstellungen abrufen.

**Response**:
```json
{
  "registration_enabled": "true",
  "max_notes_per_user": "0",
  "max_storage_mb_per_user": "0",
  "maintenance_mode": "false",
  "activity_retention_days": "90"
}
```

#### PUT /api/admin/settings

Systemeinstellungen aktualisieren.

**Request**:
```json
{
  "registration_enabled": "false",
  "max_notes_per_user": "100"
}
```

**Response**: Alle aktuellen Settings (siehe GET)

**Validierung**:
- `registration_enabled`: Muss "true" oder "false" sein
- `maintenance_mode`: Muss "true" oder "false" sein
- Numerische Werte: Müssen >= 0 sein
- Unbekannte Keys werden abgelehnt (400 Bad Request)

**Error Codes**:
- `400`: Ungültige Einstellungen oder leere Request
- `500`: Speicherfehler

---

## Sicherheit

### Zweifacher Schutz

Alle Admin-Endpunkte sind durch zwei Sicherheitsschichten geschützt:

1. **Authentication Middleware**: Prüft JWT/Cookie und setzt `user_id` im Request-Context
2. **Admin Middleware**: Prüft `is_admin`-Flag in der Datenbank

```go
// In backend/internal/api/api.go
r.Route("/admin", func(r chi.Router) {
    r.Use(s.adminMiddleware)  // Prüft is_admin=1
    // ... Admin-Routen
})
```

```go
// In backend/internal/api/middleware.go
func (s *Server) adminMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID, ok := getUserID(r)
        if !ok {
            respondError(w, http.StatusUnauthorized, "authentication required")
            return
        }

        isAdmin, err := s.adminService.IsUserAdmin(userID)
        if err != nil {
            respondError(w, http.StatusInternalServerError, "failed to check admin status")
            return
        }

        if !isAdmin {
            respondError(w, http.StatusForbidden, "admin access required")
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### Selbstschutz-Maßnahmen

Das System verhindert gefährliche Selbst-Operationen:

#### 1. Eigene Admin-Rechte entziehen

```go
// In backend/internal/service/admin.go
func (s *AdminService) SetUserAdmin(adminID, targetID int, isAdmin bool) error {
    if adminID == targetID && !isAdmin {
        return errors.New("cannot demote yourself")
    }
    // ...
}
```

**Frontend-Check**:
```typescript
// In frontend/src/routes/admin/+page.svelte
async function handleToggleAdmin(user: AdminUser) {
    const currentUser = auth.getCurrentUser();
    if (currentUser && user.id === currentUser.id) {
        toast.error('Cannot change your own admin status');
        return;
    }
    // ...
}
```

#### 2. Sich selbst löschen

```go
// In backend/internal/service/admin.go
func (s *AdminService) DeleteUser(adminID, targetID int) error {
    if adminID == targetID {
        return errors.New("cannot delete yourself")
    }
    // ...
}
```

**Frontend-Check**:
```typescript
// In frontend/src/routes/admin/+page.svelte
function confirmDeleteUser(user: AdminUser) {
    const currentUser = auth.getCurrentUser();
    if (currentUser && user.id === currentUser.id) {
        toast.error('Cannot delete yourself');
        return;
    }
    // ...
}
```

### Activity Logging

Alle sensiblen Admin-Operationen werden im Activity Log erfasst:

- **Wer**: User-ID des ausführenden Admins
- **Was**: Aktion (user_admin_set, user_delete, settings_change)
- **Wann**: Timestamp (UTC)
- **Wo**: IP-Adresse des Clients
- **Womit**: User Agent (Browser/Client)
- **Details**: Zusätzliche Informationen (z.B. geänderte Settings-Keys)

Beispiel:
```go
// In backend/internal/api/admin.go
ipAddress := getClientIP(r)
userAgent := r.UserAgent()
s.activityService.LogUserAdminSet(adminID, targetID, req.IsAdmin, targetUsername, ipAddress, userAgent)
```

### Zugriffskontrolle im Frontend

Das Frontend versteckt Admin-Features vor Nicht-Admins:

```typescript
// In frontend/src/lib/stores/auth.svelte.ts
export function isAdmin(): boolean {
    return user?.is_admin || false;
}
```

```svelte
<!-- In Sidebar.svelte -->
{#if auth.isAdmin()}
    <button onclick={() => goto('/admin')}>
        <Shield size={18} />
        <span>Admin</span>
    </button>
{/if}
```

**Wichtig**: Dies ist nur UI-Convenience. Die echte Zugriffskontrolle erfolgt **immer** im Backend.

---

## Erste Schritte

### Als erster Benutzer

1. **Registrieren**: Erstelle einen Account (z.B. via `/register`)
2. **Automatischer Admin**: Du wirst automatisch zum Administrator
3. **Admin Panel öffnen**: Klicke auf das Shield-Icon in der Sidebar
4. **Erste Einstellungen**:
   - Optional: Registrierung deaktivieren (`registration_enabled = false`)
   - Optional: Limits setzen für neue Benutzer

### Weitere Admins hinzufügen

1. Im Admin Panel zu "Users" navigieren
2. Gewünschten User suchen
3. Toggle "Admin" aktivieren
4. Bestätigen

### Activity Logs überwachen

1. Im Admin Panel zu "Activity" navigieren
2. Optional: Filter nach Action-Typ setzen
3. Logs durchsuchen
4. Details im Details-Feld prüfen (JSON)

### Registrierung für neue Instanz schließen

Typischer Workflow für private Instanzen:

```bash
# 1. Erste Instanz starten
docker run -d --name xelanote ...

# 2. Browser öffnen und eigenen Account erstellen
# 3. Admin Panel öffnen (Shield-Icon)
# 4. Settings → registration_enabled = false
# 5. Save Settings

# Ab jetzt: Nur noch bekannte Benutzer können sich einloggen
```

---

## Best Practices

### Security

- ✅ Deaktiviere Registrierung nach Initial-Setup (`registration_enabled = false`)
- ✅ Vergebe Admin-Rechte sparsam (Principle of Least Privilege)
- ✅ Überprüfe Activity Logs regelmäßig auf ungewöhnliche Aktivitäten
- ✅ Setze `activity_retention_days` basierend auf Compliance-Anforderungen

### Performance

- ✅ Lösche inaktive Benutzer regelmäßig (freier Storage)
- ✅ Nutze Activity-Filter statt alle Logs zu laden
- ✅ Setze realistische Limits (`max_notes_per_user`, `max_storage_mb_per_user`)

### Maintenance

- ✅ Aktiviere `maintenance_mode` vor größeren Updates
- ✅ Teste Änderungen an Settings in einer Test-Instanz
- ✅ Sichere Activity Logs vor Ablauf der Retention-Period

---

## Troubleshooting

### Problem: "Admin access required" bei Admin-User

**Symptom**: Admin-User bekommt 403 bei `/api/admin/*`

**Ursache**: `is_admin`-Flag nicht gesetzt

**Lösung**:
```bash
# Datenbank direkt überprüfen
sqlite3 data/xelanote.db "SELECT id, username, is_admin FROM users;"

# Falls is_admin = 0: Manuell setzen
sqlite3 data/xelanote.db "UPDATE users SET is_admin = 1 WHERE username = 'admin';"

# Backend neu starten
```

### Problem: Shield-Icon erscheint nicht

**Symptom**: Sidebar zeigt kein Admin-Icon

**Ursache**: `user.is_admin` nicht im JWT/Response

**Lösung**:
1. Logout und erneuter Login (JWT wird neu erstellt)
2. Falls Problem bleibt: Datenbank prüfen (siehe oben)

### Problem: Activity Logs wachsen zu schnell

**Symptom**: Datenbank wird groß durch activity_logs

**Lösung**:
1. `activity_retention_days` reduzieren (z.B. auf 30)
2. Alte Logs manuell löschen:
```sql
DELETE FROM activity_logs
WHERE created_at < datetime('now', '-30 days');
```
3. SQLite vacuum ausführen:
```bash
sqlite3 data/xelanote.db "VACUUM;"
```

### Problem: Versehentlich letzten Admin degradiert

**Symptom**: Kein Admin mehr vorhanden

**Lösung**:
```bash
# User mit niedrigster ID wieder zum Admin machen
sqlite3 data/xelanote.db "UPDATE users SET is_admin = 1 WHERE id = (SELECT MIN(id) FROM users);"

# Backend neu starten
docker restart xelanote
```

---

## Zukünftige Features

Geplante Erweiterungen:

- [ ] Benutzer-Invites (Admin kann Einladungslinks generieren)
- [ ] Batch-Operationen (mehrere User gleichzeitig bearbeiten)
- [ ] Export von Activity Logs (CSV/JSON)
- [ ] E-Mail-Benachrichtigungen bei kritischen Events
- [ ] Granulare Permissions (nicht nur Admin/User, sondern Rollen)
- [ ] Audit-Report Generator (PDF mit Activity-Zusammenfassung)
- [ ] Speicher-Quota Visualisierung pro User
- [ ] Backup/Restore Funktionalität über UI

---

## Siehe auch

- [API Dokumentation](api.md) - Detaillierte API-Referenz
- [Authentication](authentication.md) - JWT und Cookie-basierte Auth
- [Development Guide](development.md) - Lokales Setup und Entwicklung
- [Deployment](deployment.md) - Produktiv-Deployment auf Servern
