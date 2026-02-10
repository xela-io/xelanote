// Offline Write Mode - IndexedDB Queue
// Stores offline operations with encrypted payloads only (no plaintext in IndexedDB).
// Pattern follows kek-persistence.ts for IndexedDB usage.

import type {
  OfflineOperation,
  OfflineOperationStatus,
  TempIdMapping,
  CachedNote,
  OfflineUpdatePayload,
  OfflineCreatePayload,
} from './types';
import type { Note } from '$lib/api';

const DB_NAME = 'xelanote-offline';
const DB_VERSION = 1;
const QUEUE_STORE = 'operation_queue';
const CACHE_STORE = 'local_note_cache';
const MAPPING_STORE = 'temp_id_mappings';

// --- Database Initialization ---

export async function initOfflineDatabase(): Promise<void> {
  const request = indexedDB.open(DB_NAME, DB_VERSION);

  request.onupgradeneeded = (event) => {
    const db = (event.target as IDBOpenDBRequest).result;
    ensureStores(db);
  };

  return new Promise((resolve, reject) => {
    request.onsuccess = () => {
      request.result.close();
      resolve();
    };
    request.onerror = () => reject(request.error);
  });
}

// --- Internal DB Helper ---

async function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
    request.onupgradeneeded = (event) => {
      const db = (event.target as IDBOpenDBRequest).result;
      ensureStores(db);
    };
  });
}

function ensureStores(db: IDBDatabase): void {
  if (!db.objectStoreNames.contains(QUEUE_STORE)) {
    const store = db.createObjectStore(QUEUE_STORE, { keyPath: 'id' });
    store.createIndex('noteId', 'noteId', { unique: false });
    store.createIndex('status', 'status', { unique: false });
    store.createIndex('timestamp', 'timestamp', { unique: false });
  }

  if (!db.objectStoreNames.contains(CACHE_STORE)) {
    db.createObjectStore(CACHE_STORE, { keyPath: 'id' });
  }

  if (!db.objectStoreNames.contains(MAPPING_STORE)) {
    db.createObjectStore(MAPPING_STORE, { keyPath: 'tempId' });
  }
}

// --- Queue Operations ---

export async function enqueueOperation(op: OfflineOperation): Promise<void> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(QUEUE_STORE, 'readwrite');
    const store = tx.objectStore(QUEUE_STORE);

    await new Promise<void>((resolve, reject) => {
      const req = store.put(op);
      req.onsuccess = () => resolve();
      req.onerror = () => reject(req.error);
    });
  } catch (err) {
    // Check for quota exceeded
    if (err instanceof DOMException && err.name === 'QuotaExceededError') {
      throw new Error('QUOTA_EXCEEDED');
    }
    throw err;
  } finally {
    db.close();
  }
}

export async function dequeueOperation(id: string): Promise<void> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(QUEUE_STORE, 'readwrite');
    const store = tx.objectStore(QUEUE_STORE);

    await new Promise<void>((resolve, reject) => {
      const req = store.delete(id);
      req.onsuccess = () => resolve();
      req.onerror = () => reject(req.error);
    });
  } finally {
    db.close();
  }
}

export async function updateOperationStatus(
  id: string,
  status: OfflineOperationStatus,
  error?: string
): Promise<void> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(QUEUE_STORE, 'readwrite');
    const store = tx.objectStore(QUEUE_STORE);

    const op = await new Promise<OfflineOperation | undefined>((resolve, reject) => {
      const req = store.get(id);
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });

    if (op) {
      op.status = status;
      if (error !== undefined) op.error = error;
      await new Promise<void>((resolve, reject) => {
        const req = store.put(op);
        req.onsuccess = () => resolve();
        req.onerror = () => reject(req.error);
      });
    }
  } finally {
    db.close();
  }
}

export async function getPendingOperations(): Promise<OfflineOperation[]> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(QUEUE_STORE, 'readonly');
    const store = tx.objectStore(QUEUE_STORE);

    const all = await new Promise<OfflineOperation[]>((resolve, reject) => {
      const req = store.getAll();
      req.onsuccess = () => resolve(req.result || []);
      req.onerror = () => reject(req.error);
    });

    // Filter pending and sort by timestamp (FIFO)
    return all.filter((op) => op.status === 'pending').sort((a, b) => a.timestamp - b.timestamp);
  } finally {
    db.close();
  }
}

export async function getQueueCount(): Promise<number> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(QUEUE_STORE, 'readonly');
    const store = tx.objectStore(QUEUE_STORE);

    const all = await new Promise<OfflineOperation[]>((resolve, reject) => {
      const req = store.getAll();
      req.onsuccess = () => resolve(req.result || []);
      req.onerror = () => reject(req.error);
    });

    // Count only active operations (not completed/failed)
    return all.filter(
      (op) => op.status === 'pending' || op.status === 'syncing' || op.status === 'conflict'
    ).length;
  } finally {
    db.close();
  }
}

export async function hasPendingForNote(noteId: string): Promise<boolean> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(QUEUE_STORE, 'readonly');
    const store = tx.objectStore(QUEUE_STORE);
    const index = store.index('noteId');

    const ops = await new Promise<OfflineOperation[]>((resolve, reject) => {
      const req = index.getAll(noteId);
      req.onsuccess = () => resolve(req.result || []);
      req.onerror = () => reject(req.error);
    });

    // Count only pending, syncing, or conflict operations
    // NOT failed - so failed ops don't permanently block remote updates
    return ops.some(
      (op) => op.status === 'pending' || op.status === 'syncing' || op.status === 'conflict'
    );
  } finally {
    db.close();
  }
}

// --- Queue Optimization ---
// Called before sync. Reduces redundant operations.
// INVARIANT: Payloads are always complete (encrypted_content, wrapped_dek, folder_path, etc.).
// saveNote() and moveNote() BOTH send full note content. "Last update wins" is safe.

export async function optimizeQueue(): Promise<void> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(QUEUE_STORE, 'readwrite');
    const store = tx.objectStore(QUEUE_STORE);

    const all = await new Promise<OfflineOperation[]>((resolve, reject) => {
      const req = store.getAll();
      req.onsuccess = () => resolve(req.result || []);
      req.onerror = () => reject(req.error);
    });

    const pending = all
      .filter((op) => op.status === 'pending')
      .sort((a, b) => a.timestamp - b.timestamp);

    // Group by noteId
    const byNote = new Map<string, OfflineOperation[]>();
    for (const op of pending) {
      const existing = byNote.get(op.noteId) || [];
      existing.push(op);
      byNote.set(op.noteId, existing);
    }

    const toDelete: string[] = [];

    for (const [, ops] of byNote) {
      if (ops.length <= 1) continue;

      const types = ops.map((op) => op.type);
      const hasCreate = types.includes('create');
      const hasDelete = types.includes('delete');

      // Rule 2: Create + Delete → both cancel out (note never existed on server)
      if (hasCreate && hasDelete) {
        for (const op of ops) {
          toDelete.push(op.id);
        }
        continue;
      }

      // Rule 3: Create + Updates → merge into single Create with final content
      if (hasCreate) {
        const createOp = ops.find((op) => op.type === 'create')!;
        const updates = ops.filter((op) => op.type === 'update');

        if (updates.length > 0) {
          const lastUpdate = updates[updates.length - 1];
          const updatePayload = lastUpdate.payload as OfflineUpdatePayload;
          const createPayload = createOp.payload as OfflineCreatePayload;

          // Assertion: payload is complete
          if (updatePayload.notePayload.encrypted_content === undefined) {
            console.error(
              '[OfflineQueue] Incomplete update payload detected, skipping optimization'
            );
            continue;
          }

          // Merge: Create with final content from last update
          createOp.payload = {
            type: 'create',
            notePayload: updatePayload.notePayload,
            folderPath: createPayload.folderPath,
          };

          // Update in store
          await new Promise<void>((resolve, reject) => {
            const req = store.put(createOp);
            req.onsuccess = () => resolve();
            req.onerror = () => reject(req.error);
          });

          // Delete all update operations
          for (const update of updates) {
            toDelete.push(update.id);
          }
        }
        continue;
      }

      // Rule 4: Update + Delete → only keep Delete
      if (hasDelete) {
        const deleteOp = ops.find((op) => op.type === 'delete')!;
        for (const op of ops) {
          if (op.id !== deleteOp.id) {
            toDelete.push(op.id);
          }
        }
        continue;
      }

      // Rule 1: Multiple Updates → keep only last, but use expectedVersion from first
      const updates = ops.filter((op) => op.type === 'update');
      if (updates.length > 1) {
        const firstUpdate = updates[0];
        const lastUpdate = updates[updates.length - 1];
        const firstPayload = firstUpdate.payload as OfflineUpdatePayload;
        const lastPayload = lastUpdate.payload as OfflineUpdatePayload;

        // Assertion: payload is complete
        if (lastPayload.notePayload.encrypted_content === undefined) {
          console.error('[OfflineQueue] Incomplete update payload detected, skipping optimization');
          continue;
        }

        // Keep last update but with first's expectedVersion (= last known server version)
        lastUpdate.payload = {
          type: 'update',
          notePayload: lastPayload.notePayload,
          expectedVersion: firstPayload.expectedVersion,
        };

        // Update in store
        await new Promise<void>((resolve, reject) => {
          const req = store.put(lastUpdate);
          req.onsuccess = () => resolve();
          req.onerror = () => reject(req.error);
        });

        // Delete all but last
        for (let i = 0; i < updates.length - 1; i++) {
          toDelete.push(updates[i].id);
        }
      }
    }

    // Delete optimized-away operations
    for (const id of toDelete) {
      await new Promise<void>((resolve, reject) => {
        const req = store.delete(id);
        req.onsuccess = () => resolve();
        req.onerror = () => reject(req.error);
      });
    }
  } finally {
    db.close();
  }
}

// --- Note Cache ---

export async function cacheNote(id: string, note: Note): Promise<void> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(CACHE_STORE, 'readwrite');
    const store = tx.objectStore(CACHE_STORE);
    const record: CachedNote = { id, note, cachedAt: Date.now() };

    await new Promise<void>((resolve, reject) => {
      const req = store.put(record);
      req.onsuccess = () => resolve();
      req.onerror = () => reject(req.error);
    });
  } finally {
    db.close();
  }
}

export async function getCachedNote(id: string): Promise<Note | null> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(CACHE_STORE, 'readonly');
    const store = tx.objectStore(CACHE_STORE);

    const record = await new Promise<CachedNote | undefined>((resolve, reject) => {
      const req = store.get(id);
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });

    return record?.note || null;
  } finally {
    db.close();
  }
}

export async function removeCachedNote(id: string): Promise<void> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(CACHE_STORE, 'readwrite');
    const store = tx.objectStore(CACHE_STORE);

    await new Promise<void>((resolve, reject) => {
      const req = store.delete(id);
      req.onsuccess = () => resolve();
      req.onerror = () => reject(req.error);
    });
  } finally {
    db.close();
  }
}

// --- Temp-ID Mappings ---

export async function addTempIdMapping(mapping: TempIdMapping): Promise<void> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(MAPPING_STORE, 'readwrite');
    const store = tx.objectStore(MAPPING_STORE);

    await new Promise<void>((resolve, reject) => {
      const req = store.put(mapping);
      req.onsuccess = () => resolve();
      req.onerror = () => reject(req.error);
    });
  } finally {
    db.close();
  }
}

export async function getTempIdMapping(tempId: string): Promise<TempIdMapping | null> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(MAPPING_STORE, 'readonly');
    const store = tx.objectStore(MAPPING_STORE);

    const record = await new Promise<TempIdMapping | undefined>((resolve, reject) => {
      const req = store.get(tempId);
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });

    return record || null;
  } finally {
    db.close();
  }
}

export async function clearTempIdMappings(): Promise<void> {
  const db = await openDatabase();
  try {
    const tx = db.transaction(MAPPING_STORE, 'readwrite');
    const store = tx.objectStore(MAPPING_STORE);

    await new Promise<void>((resolve, reject) => {
      const req = store.clear();
      req.onsuccess = () => resolve();
      req.onerror = () => reject(req.error);
    });
  } finally {
    db.close();
  }
}
