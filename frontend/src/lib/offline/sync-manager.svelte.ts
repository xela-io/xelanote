// Offline Write Mode - Sync Manager
// Replays queued offline operations when connectivity returns.
// Uses Svelte 5 runes for reactive state.

import type { Note, NotePayload } from '$lib/api';
import * as api from '$lib/api';
import { ApiError } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';
import * as encryption from '$lib/stores/encryption.svelte';
import * as notes from '$lib/stores/notes.svelte';
import * as toast from '$lib/stores/toast.svelte';

import {
  addTempIdMapping,
  clearTempIdMappings,
  dequeueOperation,
  getPendingOperations,
  getQueueCount,
  optimizeQueue,
  updateOperationStatus,
} from './offline-queue';
import type {
  ConflictData,
  OfflineCreatePayload,
  OfflineOperation,
  OfflineUpdatePayload,
  SyncProgress,
  TempIdMapping,
} from './types';

// --- Reactive State (Svelte 5 runes) ---

let isSyncing = $state(false);
let syncProgress = $state<SyncProgress>({ current: 0, total: 0 });
let conflicts = $state<ConflictData[]>([]);
let pendingCount = $state(0);

// Callback for URL rewriting after temp-ID resolution
let onTempIdResolved: ((tempId: string, realId: string) => void) | null = null;

export function setOnTempIdResolved(callback: ((tempId: string, realId: string) => void) | null) {
  onTempIdResolved = callback;
}

// --- Public Getters ---

export function getIsSyncing() {
  return isSyncing;
}

export function getSyncProgress() {
  return syncProgress;
}

export function getConflicts() {
  return conflicts;
}

export function getPendingCount() {
  return pendingCount;
}

// --- Initialization ---

export async function initSyncManager() {
  // Load initial pending count from IndexedDB
  try {
    pendingCount = await getQueueCount();
  } catch {
    pendingCount = 0;
  }
}

// Called by api.ts when an operation is enqueued
export function updatePendingCount(count: number) {
  pendingCount = count;
}

// Refresh pending count from IndexedDB (for multi-tab consistency)
export async function refreshPendingCount() {
  try {
    pendingCount = await getQueueCount();
  } catch {
    // Keep current value
  }
}

// --- Main Sync Function ---

export async function startSync(): Promise<void> {
  if (isSyncing) {
    console.log('[Sync] Already syncing, skipping');
    return;
  }

  // Check encryption state - sync needs KEK for conflict resolution
  if (!encryption.isEncryptionUnlocked()) {
    console.log('[Sync] Encryption locked, deferring sync');
    return;
  }

  // Tab safety: Use navigator.locks if available
  if (typeof navigator !== 'undefined' && 'locks' in navigator) {
    try {
      await navigator.locks.request('xelanote-sync', { ifAvailable: true }, async (lock) => {
        if (!lock) {
          console.log('[Sync] Another tab is syncing');
          return;
        }
        await _doSync();
      });
    } catch {
      // Fallback: just sync without lock
      await _doSync();
    }
  } else {
    await _doSync();
  }
}

async function _doSync(): Promise<void> {
  isSyncing = true;
  console.log('[Sync] Starting sync...');

  try {
    // Step 1: Optimize queue (merge redundant operations)
    await optimizeQueue();

    // Step 2: Get pending operations (sorted by timestamp, FIFO)
    const ops = await getPendingOperations();
    if (ops.length === 0) {
      console.log('[Sync] No pending operations');
      pendingCount = 0;
      return;
    }

    syncProgress = { current: 0, total: ops.length };

    // Track temp-ID → real-ID mappings for this sync session
    const tempIdMap = new Map<string, { realId: string; realVersion: number }>();

    // Step 3: Replay operations sequentially (order matters!)
    for (let i = 0; i < ops.length; i++) {
      const op = ops[i];
      syncProgress = { current: i + 1, total: ops.length };

      try {
        await updateOperationStatus(op.id, 'syncing');
        await replayOperation(op, tempIdMap);
        await dequeueOperation(op.id);
        pendingCount = Math.max(0, pendingCount - 1);
      } catch (err) {
        if (err instanceof ApiError && err.status === 409) {
          // Conflict: create ConflictData for dialog
          await handleConflict(op, tempIdMap);
        } else if (isTransientError(err)) {
          // Transient error: retry with backoff
          const retried = await retryWithBackoff(op, tempIdMap);
          if (!retried) {
            await updateOperationStatus(op.id, 'failed', String(err));
            console.error('[Sync] Operation failed after retries:', op.id, err);
          } else {
            await dequeueOperation(op.id);
            pendingCount = Math.max(0, pendingCount - 1);
          }
        } else {
          // Non-transient error (4xx except 409): mark as failed
          await updateOperationStatus(op.id, 'failed', String(err));
          console.error('[Sync] Operation permanently failed:', op.id, err);
        }
      }
    }

    // Step 4: Refresh state from server
    await notes.loadNotes();

    // Step 5: Show result
    const finalCount = await getQueueCount();
    pendingCount = finalCount;

    if (conflicts.length > 0) {
      toast.warning(
        `Sync abgeschlossen - ${conflicts.length} Konflikt(e) erfordern Aufmerksamkeit`
      );
    } else if (finalCount === 0) {
      toast.success('Alle Aenderungen synchronisiert');
    }

    // Clear temp-ID mappings after successful sync
    await clearTempIdMappings();
  } catch (err) {
    console.error('[Sync] Sync failed:', err);
    toast.error('Synchronisation fehlgeschlagen');
  } finally {
    isSyncing = false;
    syncProgress = { current: 0, total: 0 };
  }
}

// --- Operation Replay ---

async function replayOperation(
  op: OfflineOperation,
  tempIdMap: Map<string, { realId: string; realVersion: number }>
): Promise<void> {
  switch (op.type) {
    case 'create':
      await replayCreate(op, tempIdMap);
      break;
    case 'update':
      await replayUpdate(op, tempIdMap);
      break;
    case 'delete':
      await replayDelete(op, tempIdMap);
      break;
  }
}

async function replayCreate(
  op: OfflineOperation,
  tempIdMap: Map<string, { realId: string; realVersion: number }>
): Promise<void> {
  const payload = op.payload as OfflineCreatePayload;
  const serverNote = await api.createNote(payload.notePayload);

  // Store temp-ID mapping
  const mapping: TempIdMapping = {
    tempId: op.noteId,
    realId: serverNote.id,
    realVersion: serverNote.version,
  };
  tempIdMap.set(op.noteId, { realId: serverNote.id, realVersion: serverNote.version });
  await addTempIdMapping(mapping);

  // Decrypt for local state update
  const decryptedNote = decryptServerNote(serverNote);

  // Update local state: replace temp-ID with real ID
  notes.replaceTempId(op.noteId, decryptedNote);

  // Notify for URL rewriting (e.g. goto(/note/${realId}, { replaceState: true }))
  if (onTempIdResolved) {
    onTempIdResolved(op.noteId, serverNote.id);
  }

  console.log(`[Sync] Create synced: ${op.noteId} → ${serverNote.id}`);
}

async function replayUpdate(
  op: OfflineOperation,
  tempIdMap: Map<string, { realId: string; realVersion: number }>
): Promise<void> {
  const payload = op.payload as OfflineUpdatePayload;

  // Resolve temp-ID if this note was created offline
  let noteId = op.noteId;
  let expectedVersion = payload.expectedVersion;

  const mapping = tempIdMap.get(op.noteId);
  if (mapping) {
    noteId = mapping.realId;
    expectedVersion = mapping.realVersion;
  }

  const serverNote = await api.updateNote(noteId, payload.notePayload, expectedVersion);

  // Update mapping with new version
  if (mapping) {
    tempIdMap.set(op.noteId, { realId: noteId, realVersion: serverNote.version });
  }

  console.log(`[Sync] Update synced: ${noteId} v${expectedVersion} → v${serverNote.version}`);
}

async function replayDelete(
  op: OfflineOperation,
  tempIdMap: Map<string, { realId: string; realVersion: number }>
): Promise<void> {
  // Resolve temp-ID
  let noteId = op.noteId;
  const mapping = tempIdMap.get(op.noteId);
  if (mapping) {
    noteId = mapping.realId;
  }

  await api.deleteNote(noteId);
  console.log(`[Sync] Delete synced: ${noteId}`);
}

// --- Conflict Handling ---

async function handleConflict(
  op: OfflineOperation,
  tempIdMap: Map<string, { realId: string; realVersion: number }>
): Promise<void> {
  // Resolve real noteId
  let noteId = op.noteId;
  const mapping = tempIdMap.get(op.noteId);
  if (mapping) {
    noteId = mapping.realId;
  }

  try {
    // Fetch server version
    const serverNote = await api.getNote(noteId);
    const decryptedServer = decryptServerNote(serverNote);

    // Decrypt local payload
    const isDelete = op.type === 'delete';
    let localTitle = '';
    let localContent = '';

    if (!isDelete && (op.payload.type === 'update' || op.payload.type === 'create')) {
      const notePayload = op.payload.notePayload;
      const decrypted = decryptPayload(notePayload);
      localTitle = decrypted.title;
      localContent = decrypted.content;
    }

    const conflictData: ConflictData = {
      operationId: op.id,
      noteId: noteId,
      localTitle: isDelete ? '(Geloescht)' : localTitle,
      localContent: isDelete ? '' : localContent,
      remoteTitle: decryptedServer.title,
      remoteContent: decryptedServer.content,
      remoteVersion: serverNote.version,
      isDelete,
    };

    await updateOperationStatus(op.id, 'conflict');
    conflicts = [...conflicts, conflictData];
  } catch (err) {
    console.error('[Sync] Failed to build conflict data:', err);
    await updateOperationStatus(op.id, 'failed', 'Conflict resolution failed: ' + String(err));
  }
}

// --- Conflict Resolution ---

export async function resolveConflict(
  operationId: string,
  resolution: 'keep_local' | 'keep_remote' | 'keep_both'
): Promise<void> {
  const conflict = conflicts.find((c) => c.operationId === operationId);
  if (!conflict) return;

  // Ensure encryption is unlocked
  if (!encryption.isEncryptionUnlocked()) {
    toast.error('Verschluesselung gesperrt - bitte zuerst entsperren');
    return;
  }

  try {
    switch (resolution) {
      case 'keep_local': {
        if (conflict.isDelete) {
          // Delete on server
          await api.deleteNote(conflict.noteId);
        } else {
          // Re-encrypt local content and send to server with server version
          const { encryptedTitle, encryptedContent } = encryption.encryptNote(
            conflict.localTitle,
            conflict.localContent
          );

          const payload: NotePayload = {
            title: encryptedTitle ? '' : conflict.localTitle,
            encrypted_title: encryptedTitle,
            title_encrypted: !!encryptedTitle,
            encrypted_content: encryptedContent.ciphertext,
            wrapped_dek: encryptedContent.metadata.wrapped_dek,
            encryption_metadata: JSON.stringify(encryptedContent.metadata),
          };

          await api.updateNote(conflict.noteId, payload, conflict.remoteVersion);
        }
        break;
      }

      case 'keep_remote': {
        // Simply discard local operation - server version is already correct
        break;
      }

      case 'keep_both': {
        // Keep server version + create a copy with local content
        const copyTitle = conflict.localTitle + ' (Offline-Kopie)';
        const { encryptedTitle, encryptedContent } = encryption.encryptNote(
          copyTitle,
          conflict.localContent
        );

        const payload: NotePayload = {
          title: encryptedTitle ? '' : copyTitle,
          encrypted_title: encryptedTitle,
          title_encrypted: !!encryptedTitle,
          encrypted_content: encryptedContent.ciphertext,
          wrapped_dek: encryptedContent.metadata.wrapped_dek,
          encryption_metadata: JSON.stringify(encryptedContent.metadata),
        };

        await api.createNote(payload);
        break;
      }
    }

    // Remove from queue and conflicts
    await dequeueOperation(operationId);
    conflicts = conflicts.filter((c) => c.operationId !== operationId);
    pendingCount = Math.max(0, pendingCount - 1);

    // Refresh notes after resolution
    await notes.loadNotes();

    toast.success('Konflikt geloest');
  } catch (err) {
    console.error('[Sync] Conflict resolution failed:', err);
    toast.error(
      'Konfliktloesung fehlgeschlagen: ' + (err instanceof Error ? err.message : String(err))
    );
  }
}

// --- Helpers ---

function isTransientError(err: unknown): boolean {
  if (err instanceof ApiError) {
    return err.status >= 500 || err.status === 429;
  }
  // TypeError typically means network unreachable
  return err instanceof TypeError;
}

async function retryWithBackoff(
  op: OfflineOperation,
  tempIdMap: Map<string, { realId: string; realVersion: number }>,
  maxRetries = 5
): Promise<boolean> {
  for (let i = 0; i < maxRetries; i++) {
    const delay = Math.min(1000 * Math.pow(2, i), 30000); // 1s, 2s, 4s, 8s, 16s (max 30s)
    await new Promise((resolve) => setTimeout(resolve, delay));

    try {
      await replayOperation(op, tempIdMap);
      return true;
    } catch (err) {
      if (!isTransientError(err)) {
        return false; // Non-transient: stop retrying
      }
      console.log(`[Sync] Retry ${i + 1}/${maxRetries} for ${op.id}`);
    }
  }
  return false;
}

function decryptServerNote(note: Note): Note {
  if (!note.content_encrypted || !note.encrypted_content) return note;
  if (!encryption.isEncryptionUnlocked()) return note;

  try {
    const encryptedPayload: EncryptedPayload = {
      ciphertext: note.encrypted_content,
      metadata: JSON.parse(note.encryption_metadata || '{}'),
    };

    const { title, content } = encryption.decryptNote(
      note.encrypted_title || null,
      encryptedPayload
    );

    return {
      ...note,
      title: title || note.title,
      content,
    };
  } catch {
    return note;
  }
}

function decryptPayload(notePayload: NotePayload): { title: string; content: string } {
  if (!notePayload.encrypted_content) {
    return { title: notePayload.title || '', content: notePayload.content || '' };
  }

  if (!encryption.isEncryptionUnlocked()) {
    return { title: notePayload.title || 'Untitled', content: '' };
  }

  try {
    const encryptedPayload: EncryptedPayload = {
      ciphertext: notePayload.encrypted_content,
      metadata: JSON.parse(notePayload.encryption_metadata || '{}'),
    };

    const { title, content } = encryption.decryptNote(
      notePayload.encrypted_title || null,
      encryptedPayload
    );

    return {
      title: title || notePayload.title || 'Untitled',
      content,
    };
  } catch {
    return { title: notePayload.title || 'Untitled', content: '' };
  }
}
