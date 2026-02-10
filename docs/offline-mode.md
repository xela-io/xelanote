# Offline-Modus

xelanote unterstuetzt zwei Offline-Stufen: Read Mode (seit M0) und Write Mode Phase 1 (seit M2, 2026-02-06).

## Offline Read Mode

- Service Worker cached Notes und Folders fuer Offline-Lesezugriff
- Klare Hinweise im UI wenn offline (OfflineBanner)
- Kein Schreibzugriff moeglich

Refs: `frontend/vite.config.ts` (Service Worker Config), `frontend/src/lib/components/OfflineBanner.svelte`

## Offline Write Mode (Phase 1)

### Unterstuetzte Operationen

| Operation | Offline | Anmerkung |
|-----------|---------|-----------|
| Note erstellen | Ja | Temp-ID, nach Sync URL-Rewriting |
| Note bearbeiten | Ja | Verschluesselt in IndexedDB |
| Note verschieben | Ja | Folder-Zuweisung |
| Note loeschen | Ja | Soft-Delete |
| Tags zuweisen | Nein | Online-only |
| Folder erstellen/umbenennen/loeschen | Nein | Online-only |
| Note umbenennen | Nein | Online-only |
| Trash/Restore | Nein | Online-only |

### Architektur

```
Browser (offline)
  |
  +-- api.ts (Offline Interception)
  |     Erkennt fehlende Konnektivitaet, leitet Schreiboperationen an Queue
  |
  +-- offline-queue.ts (IndexedDB Queue)
  |     Speichert Operationen verschluesselt (kein Plaintext in IndexedDB)
  |     Queue-Optimierung vor Sync: merge updates, cancel create+delete, fold create+updates
  |
  +-- sync-manager.svelte.ts (Sync Engine)
  |     Svelte 5 Runes fuer reaktiven Status
  |     Automatischer Sync bei Reconnect (via network.svelte.ts)
  |     Tab-Safety: navigator.locks API verhindert gleichzeitigen Sync in mehreren Tabs
  |
  +-- diff-utils.ts (Diff Utilities)
  |     Line-based Diff fuer Conflict Resolution Dialog
  |
  +-- ConflictDialog.svelte (Conflict UI)
        Bei HTTP 409: Keep Local / Keep Remote / Keep Both (Kopie)
```

### Sicherheit

- **Kein Plaintext in IndexedDB** -- nur verschluesselte Payloads werden gespeichert
- **Keine Encryption-Imports in api.ts** -- verhindert zirkulaere Abhaengigkeiten
- **Paranoid Security Mode** -- blockiert Offline-Schreiboperationen komplett (bleibt read-only)

### Temp-ID-System

Offline erstellte Notizen erhalten eine temporaere ID. Nach erfolgreichem Sync wird die Server-ID zugewiesen und die URL im Browser umgeschrieben, sodass der User keine Unterbrechung bemerkt.

### Offline-Status UI

- **Editor-Toolbar**: Kompakte Pill zeigt Offline-Status und Anzahl ausstehender Aenderungen
- **Nicht-Editor-Seiten**: OfflineBanner am unteren Bildschirmrand

## Phase 2 (geplant)

Erweiterte Offline-Operationen:
- Tags offline zuweisen/entfernen
- Folder erstellen/umbenennen/loeschen offline
- Note Rename offline
- Trash/Restore offline
- Queue-UI mit Pending-Changes-Anzeige
- Ggf. Batch-Sync-Endpoint im Backend

## Referenzen

- `frontend/src/lib/offline/types.ts` -- TypeScript Types
- `frontend/src/lib/offline/offline-queue.ts` -- IndexedDB Queue
- `frontend/src/lib/offline/sync-manager.svelte.ts` -- Sync Engine
- `frontend/src/lib/offline/diff-utils.ts` -- Diff Utilities
- `frontend/src/lib/components/ConflictDialog.svelte` -- Conflict Resolution UI
- `frontend/src/lib/components/OfflineBanner.svelte` -- Offline-Banner (non-editor)
- `frontend/src/lib/components/Editor.svelte` -- Offline-Pill (editor)
- `frontend/src/lib/api.ts` -- Offline Interception + Queue Integration
- `frontend/src/lib/stores/notes.svelte.ts` -- Offline Gating + Temp-ID Handling
- `frontend/src/lib/stores/network.svelte.ts` -- Reconnect Trigger
- `frontend/src/routes/+layout.svelte` -- Offline DB Init + Sync Startup
