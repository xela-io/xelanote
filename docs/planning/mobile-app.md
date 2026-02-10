# Xelanote Mobile App - Implementierungsplan

## Ziel
Native Mobile App (Android + iOS) mit **Full Offline + Sync** via Capacitor.

---

## Architektur-Überblick

```
┌─────────────────────────────────────────────────────────────┐
│                    Capacitor App                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │            SvelteKit Frontend (angepasst)              │  │
│  └─────────────────────────┬─────────────────────────────┘  │
│                            │                                 │
│  ┌─────────────────────────▼─────────────────────────────┐  │
│  │                   Sync-Engine (NEU)                    │  │
│  │  - Push: Pending Changes → Server                      │  │
│  │  - Pull: Server Changes → Local DB                     │  │
│  │  - Konfliktauflösung (LWW)                             │  │
│  └─────────────────────────┬─────────────────────────────┘  │
│                            │                                 │
│  ┌─────────────────────────▼─────────────────────────────┐  │
│  │              Lokale SQLite DB (Capacitor)              │  │
│  │  - Notes, Folders (Spiegel der Server-DB)              │  │
│  │  - Pending Changes Queue                               │  │
│  │  - Sync State (last_sync_at)                           │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              Native Features (Capacitor)               │  │
│  │  - Biometrie (Fingerprint/FaceID)                      │  │
│  │  - Secure Storage (KEK in Keychain/Keystore)           │  │
│  │  - Network Status, App Lifecycle                       │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              ↕ Sync
┌─────────────────────────────────────────────────────────────┐
│                    Backend (Go/Chi)                          │
│  - NEU: GET /api/sync/changes (Delta-Sync)                  │
│  - NEU: POST /api/sync/push (Batched Changes)               │
│  - Bestehend: REST API, WebSocket                           │
└─────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Capacitor Setup + Read-Only Offline

### Ziel
App läuft, Notizen können offline gelesen werden.

### Tasks

1. **Capacitor initialisieren**
   ```bash
   cd frontend
   npm install @capacitor/core @capacitor/cli
   npx cap init xelanote com.xelanote.app
   npx cap add android
   npx cap add ios
   ```

2. **Plugins installieren**
   - `@capacitor-community/sqlite` - Lokale Datenbank
   - `@capacitor/network` - Online/Offline-Erkennung
   - `@capacitor/app` - App-Lifecycle
   - `@capacitor/preferences` - Key-Value Storage
   - `capacitor-native-biometric` - Biometrie
   - `capacitor-secure-storage-plugin` - Sichere Speicherung

3. **Lokale DB erstellen**
   - Datei: `frontend/src/lib/mobile/db.ts`
   - Schema: Notes, Folders, Sync-State (siehe unten)

4. **Initial-Sync bei Login**
   - Alle Notizen vom Server laden
   - In lokale DB speichern
   - Offline: Aus lokaler DB lesen

5. **Biometrie-Login (Basic)**
   - KEK in Secure Storage speichern
   - Biometric-Prompt bei App-Start

### Neue Dateien
```
frontend/
├── capacitor.config.ts
├── android/                    (auto-generiert)
├── ios/                        (auto-generiert)
└── src/lib/mobile/
    ├── index.ts                # Platform detection
    ├── db.ts                   # SQLite wrapper
    ├── migrations.ts           # DB migrations
    ├── kek-storage.ts          # Secure KEK storage
    └── biometric.ts            # Biometric wrapper
```

---

## Phase 2: Full Offline + Sync Engine

### Ziel
Notizen offline bearbeiten, automatischer Sync bei Verbindung.

### Backend-Änderungen

**Neuer Endpoint: `GET /api/sync/changes`**
```
Request:  ?since=2026-01-20T10:30:00Z&limit=100
Response: {
  changes: [{ id, action, data, version, timestamp }],
  folders: [...],
  server_time: "...",
  has_more: false
}
```

**Neuer Endpoint: `POST /api/sync/push`**
```
Request:  { changes: [{ id, action, data, client_version }] }
Response: {
  results: [
    { id, status: "ok", server_version },
    { id, status: "conflict", server_data }
  ]
}
```

**Migration: `023_sync_metadata.sql`**
```sql
CREATE TABLE sync_clients (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    device_id TEXT NOT NULL,
    last_sync_at TEXT,
    platform TEXT,
    UNIQUE(user_id, device_id)
);
```

### Frontend-Änderungen

**Sync-Engine** (`frontend/src/lib/mobile/sync-engine.ts`)
```
1. PUSH (Client → Server)
   - Pending Changes aus Queue lesen
   - POST /api/sync/push
   - Bei Erfolg: Queue leeren, lokale Version updaten
   - Bei Konflikt: Auflösung (siehe unten)

2. PULL (Server → Client)
   - GET /api/sync/changes?since=<last_sync>
   - Änderungen auf lokale DB anwenden
   - last_sync_at aktualisieren

3. Trigger
   - App wird aktiv (Vordergrund)
   - Netzwerk verfügbar
   - Manueller Refresh
   - Alle 5 Minuten (optional)
```

**Offline-Editing**
- `notes.svelte.ts` anpassen: Mobile + Offline erkennen
- In lokale DB speichern, in Queue eintragen
- Sync-Status anzeigen

### Konfliktauflösung: Last-Write-Wins (LWW)

```typescript
if (serverTime > localTime) {
  // Server gewinnt - lokale Version in History speichern
  await saveLocalAsVersion(local);
  return server;
} else {
  // Lokal gewinnt - wird beim nächsten Sync gepusht
  await saveServerAsVersion(server);
  return local;
}
```

**User-Feedback:** "Notiz synchronisiert. Vorherige Version in History gespeichert."

### Neue Dateien
```
frontend/src/lib/mobile/
├── sync-engine.ts              # Push/Pull Logik
├── sync-queue.ts               # Pending Changes Queue
└── network.ts                  # Enhanced Network Status

frontend/src/lib/stores/
├── sync.svelte.ts              # Sync State Store

frontend/src/lib/components/mobile/
├── SyncStatus.svelte           # Sync-Indikator
└── ConflictDialog.svelte       # Konflikt-UI

backend/internal/api/
└── sync.go                     # Sync Endpoints

backend/internal/db/
├── sync.go                     # Sync Queries
└── migrations/023_sync_metadata.sql
```

---

## Phase 3: Native Polish

### Tasks
- Pull-to-Refresh
- Swipe-Gesten (Löschen, Zurück)
- Optimiertes List-Rendering
- Background-Sync
- App-Icons, Splash Screen
- App Store Vorbereitung

---

## Lokales DB-Schema

```sql
-- Notes (Spiegel der Server-DB + Sync-Metadaten)
CREATE TABLE notes (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT,
    folder_path TEXT DEFAULT '/',
    version INTEGER DEFAULT 1,
    created_at TEXT,
    updated_at TEXT,
    deleted_at TEXT,
    is_deleted INTEGER DEFAULT 0,

    -- Encryption
    encrypted_content BLOB,
    wrapped_dek TEXT,
    encryption_version INTEGER DEFAULT 0,

    -- Sync (Client-only)
    sync_status TEXT DEFAULT 'synced',  -- synced | pending | conflict
    local_updated_at TEXT,
    server_version INTEGER
);

-- Pending Changes Queue
CREATE TABLE pending_changes (
    id INTEGER PRIMARY KEY,
    entity_type TEXT,           -- note | folder
    entity_id TEXT,
    action TEXT,                -- create | update | delete
    payload TEXT,               -- JSON
    created_at TEXT,
    retry_count INTEGER DEFAULT 0
);

-- Sync State
CREATE TABLE sync_state (
    key TEXT PRIMARY KEY,
    value TEXT
);
-- Keys: last_sync_at, device_id, user_id
```

---

## Capacitor Plugins

| Plugin | Zweck |
|--------|-------|
| `@capacitor-community/sqlite` | Lokale Datenbank |
| `@capacitor/network` | Online/Offline |
| `@capacitor/app` | App-Lifecycle |
| `@capacitor/preferences` | Key-Value Storage |
| `@capacitor/keyboard` | Keyboard-Handling |
| `@capacitor/status-bar` | Status-Bar Styling |
| `@capacitor/splash-screen` | Ladebildschirm |
| `capacitor-native-biometric` | Fingerprint/FaceID |
| `capacitor-secure-storage-plugin` | Keychain/Keystore |

---

## Kritische Dateien

| Datei | Änderung |
|-------|----------|
| `frontend/src/lib/stores/notes.svelte.ts` | Offline-Mode hinzufügen |
| `frontend/src/lib/api.ts` | Sync-Endpoints hinzufügen |
| `frontend/src/lib/crypto/e2e.ts` | Verify Capacitor-Kompatibilität |
| `backend/internal/api/api.go` | Sync-Routes registrieren |
| `backend/internal/db/notes.go` | GetChangedNotes Query |

---

## Risiken

| Risiko | Mitigation |
|--------|------------|
| SQLite-Plugin Limitationen | Früh testen, Fallback auf Preferences |
| Konflikt-Komplexität | LWW als Default, User-Prompt nur bei echten Konflikten |
| libsodium Performance | Bereits in Tauri/Electron getestet, sollte funktionieren |
| iOS App Store Review | Apple Guidelines strikt befolgen |

---

## Verifizierung

### Phase 1
1. `npm run build && npx cap sync`
2. Android Studio / Xcode öffnen
3. App auf Gerät installieren
4. Login → Notizen laden
5. Flugmodus aktivieren → Notizen noch sichtbar
6. Biometrie-Login testen

### Phase 2
1. Offline: Notiz bearbeiten
2. Online gehen → Sync startet automatisch
3. Auf anderem Gerät prüfen: Änderung sichtbar
4. Konflikt provozieren → LWW funktioniert
5. Version History: Alte Version noch vorhanden
