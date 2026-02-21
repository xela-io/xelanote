import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { OfflineOperation } from './types';

// --- In-memory IndexedDB mock ---

type StoreRecord = Record<string, unknown>;

class MockIDBObjectStore {
  private data = new Map<string, StoreRecord>();
  private indexes = new Map<string, string>();
  private keyPath: string;

  constructor(keyPath: string) {
    this.keyPath = keyPath;
  }

  createIndex(name: string, keyPath: string, _options?: { unique: boolean }) {
    this.indexes.set(name, keyPath);
  }

  put(value: StoreRecord): MockIDBRequest<IDBValidKey> {
    const key = value[this.keyPath] as string;
    this.data.set(key, structuredClone(value));
    return MockIDBRequest.resolve(key);
  }

  get(key: string): MockIDBRequest<StoreRecord | undefined> {
    const value = this.data.get(key);
    return MockIDBRequest.resolve(value ? structuredClone(value) : undefined);
  }

  getAll(): MockIDBRequest<StoreRecord[]> {
    const results = Array.from(this.data.values()).map((v) => structuredClone(v));
    return MockIDBRequest.resolve(results);
  }

  delete(key: string): MockIDBRequest<undefined> {
    this.data.delete(key);
    return MockIDBRequest.resolve(undefined);
  }

  clear(): MockIDBRequest<undefined> {
    this.data.clear();
    return MockIDBRequest.resolve(undefined);
  }

  index(name: string): MockIDBIndex {
    const keyPath = this.indexes.get(name);
    if (!keyPath) throw new Error(`Index ${name} not found`);
    return new MockIDBIndex(this.data, keyPath);
  }

  _getData() {
    return this.data;
  }
}

class MockIDBIndex {
  private data: Map<string, StoreRecord>;
  private keyPath: string;

  constructor(data: Map<string, StoreRecord>, keyPath: string) {
    this.data = data;
    this.keyPath = keyPath;
  }

  getAll(query?: string): MockIDBRequest<StoreRecord[]> {
    let results = Array.from(this.data.values());
    if (query !== undefined) {
      results = results.filter((r) => r[this.keyPath] === query);
    }
    return MockIDBRequest.resolve(results.map((v) => structuredClone(v)));
  }
}

class MockIDBRequest<T> {
  result!: T;
  error: DOMException | null = null;
  onsuccess: (() => void) | null = null;
  onerror: (() => void) | null = null;

  static resolve<T>(value: T): MockIDBRequest<T> {
    const req = new MockIDBRequest<T>();
    req.result = value;
    // Defer to allow assignment of onsuccess
    queueMicrotask(() => req.onsuccess?.());
    return req;
  }

  static reject<T>(error: DOMException): MockIDBRequest<T> {
    const req = new MockIDBRequest<T>();
    req.error = error;
    queueMicrotask(() => req.onerror?.());
    return req;
  }
}

class MockIDBTransaction {
  private stores: Map<string, MockIDBObjectStore>;

  constructor(stores: Map<string, MockIDBObjectStore>) {
    this.stores = stores;
  }

  objectStore(name: string): MockIDBObjectStore {
    const store = this.stores.get(name);
    if (!store) throw new Error(`Store ${name} not found`);
    return store;
  }
}

class MockIDBDatabase {
  private stores = new Map<string, MockIDBObjectStore>();
  objectStoreNames = {
    contains: (name: string) => this.stores.has(name),
  };

  createObjectStore(name: string, options: { keyPath: string }): MockIDBObjectStore {
    const store = new MockIDBObjectStore(options.keyPath);
    this.stores.set(name, store);
    return store;
  }

  transaction(_storeNames: string | string[], _mode?: string): MockIDBTransaction {
    return new MockIDBTransaction(this.stores);
  }

  close() {
    // no-op
  }
}

// Shared DB instance per test (simulates persistent IndexedDB)
let mockDb: MockIDBDatabase;

function setupIndexedDBMock() {
  mockDb = new MockIDBDatabase();

  const mockOpen = (_name: string, _version?: number) => {
    const req = new MockIDBRequest<MockIDBDatabase>();
    req.result = mockDb;

    queueMicrotask(() => {
      // Trigger onupgradeneeded first if stores don't exist
      if (!mockDb.objectStoreNames.contains('operation_queue')) {
        const upgradeEvent = {
          target: { result: mockDb },
        } as unknown as IDBVersionChangeEvent;
        (req as unknown as IDBOpenDBRequest).onupgradeneeded?.(upgradeEvent);
      }
      req.onsuccess?.();
    });

    return req;
  };

  vi.stubGlobal('indexedDB', { open: mockOpen });
}

// --- Tests ---

describe('offline-queue', () => {
  beforeEach(() => {
    vi.resetModules();
    setupIndexedDBMock();
  });

  function makeOperation(overrides: Partial<OfflineOperation> = {}): OfflineOperation {
    return {
      id: 'op-1',
      type: 'create',
      noteId: 'note-1',
      tempId: 'temp_note-1',
      timestamp: Date.now(),
      status: 'pending',
      retryCount: 0,
      payload: {
        type: 'create',
        notePayload: { title: 'Test Note' },
        folderPath: '/',
      },
      ...overrides,
    };
  }

  it('enqueues and retrieves pending operations in FIFO order', async () => {
    const { enqueueOperation, getPendingOperations } = await import('./offline-queue');

    const op1 = makeOperation({ id: 'op-1', timestamp: 100 });
    const op2 = makeOperation({ id: 'op-2', noteId: 'note-2', timestamp: 200 });

    await enqueueOperation(op1);
    await enqueueOperation(op2);

    const pending = await getPendingOperations();
    expect(pending).toHaveLength(2);
    expect(pending[0].id).toBe('op-1');
    expect(pending[1].id).toBe('op-2');
  });

  it('dequeues an operation by id', async () => {
    const { enqueueOperation, dequeueOperation, getPendingOperations } =
      await import('./offline-queue');

    await enqueueOperation(makeOperation({ id: 'op-1' }));
    await enqueueOperation(makeOperation({ id: 'op-2', noteId: 'note-2' }));

    await dequeueOperation('op-1');

    const pending = await getPendingOperations();
    expect(pending).toHaveLength(1);
    expect(pending[0].id).toBe('op-2');
  });

  it('updates operation status', async () => {
    const { enqueueOperation, updateOperationStatus, getPendingOperations, getQueueCount } =
      await import('./offline-queue');

    await enqueueOperation(makeOperation({ id: 'op-1', status: 'pending' }));

    await updateOperationStatus('op-1', 'syncing');

    // getPendingOperations only returns 'pending' status
    const pending = await getPendingOperations();
    expect(pending).toHaveLength(0);

    // getQueueCount counts pending + syncing + conflict
    const count = await getQueueCount();
    expect(count).toBe(1);
  });

  it('getQueueCount excludes completed and failed operations', async () => {
    const { enqueueOperation, updateOperationStatus, getQueueCount } =
      await import('./offline-queue');

    await enqueueOperation(makeOperation({ id: 'op-1', status: 'pending' }));
    await enqueueOperation(makeOperation({ id: 'op-2', noteId: 'n2', status: 'pending' }));
    await enqueueOperation(makeOperation({ id: 'op-3', noteId: 'n3', status: 'pending' }));

    await updateOperationStatus('op-2', 'completed');
    await updateOperationStatus('op-3', 'failed');

    const count = await getQueueCount();
    expect(count).toBe(1); // Only op-1 (pending)
  });

  it('hasPendingForNote returns true for pending/syncing/conflict ops', async () => {
    const { enqueueOperation, updateOperationStatus, hasPendingForNote } =
      await import('./offline-queue');

    await enqueueOperation(makeOperation({ id: 'op-1', noteId: 'note-A', status: 'pending' }));

    expect(await hasPendingForNote('note-A')).toBe(true);
    expect(await hasPendingForNote('note-B')).toBe(false);

    await updateOperationStatus('op-1', 'syncing');
    expect(await hasPendingForNote('note-A')).toBe(true);

    await updateOperationStatus('op-1', 'failed');
    expect(await hasPendingForNote('note-A')).toBe(false);
  });

  describe('optimizeQueue', () => {
    it('collapses multiple updates into the last one with first expectedVersion', async () => {
      const { enqueueOperation, optimizeQueue, getPendingOperations } =
        await import('./offline-queue');

      await enqueueOperation(
        makeOperation({
          id: 'u1',
          type: 'update',
          noteId: 'note-X',
          timestamp: 100,
          payload: {
            type: 'update',
            notePayload: { title: 'v1', encrypted_content: 'enc-v1' },
            expectedVersion: 1,
          },
        })
      );
      await enqueueOperation(
        makeOperation({
          id: 'u2',
          type: 'update',
          noteId: 'note-X',
          timestamp: 200,
          payload: {
            type: 'update',
            notePayload: { title: 'v2', encrypted_content: 'enc-v2' },
            expectedVersion: 2,
          },
        })
      );
      await enqueueOperation(
        makeOperation({
          id: 'u3',
          type: 'update',
          noteId: 'note-X',
          timestamp: 300,
          payload: {
            type: 'update',
            notePayload: { title: 'v3', encrypted_content: 'enc-v3' },
            expectedVersion: 3,
          },
        })
      );

      await optimizeQueue();

      const pending = await getPendingOperations();
      expect(pending).toHaveLength(1);
      expect(pending[0].id).toBe('u3');
      // Last content but first expectedVersion
      const payload = pending[0].payload as {
        type: string;
        notePayload: { title: string; encrypted_content: string };
        expectedVersion: number;
      };
      expect(payload.notePayload.title).toBe('v3');
      expect(payload.expectedVersion).toBe(1);
    });

    it('cancels create + delete for the same note', async () => {
      const { enqueueOperation, optimizeQueue, getPendingOperations } =
        await import('./offline-queue');

      await enqueueOperation(
        makeOperation({
          id: 'c1',
          type: 'create',
          noteId: 'temp_abc',
          tempId: 'temp_abc',
          timestamp: 100,
          payload: {
            type: 'create',
            notePayload: { title: 'New Note' },
            folderPath: '/',
          },
        })
      );
      await enqueueOperation(
        makeOperation({
          id: 'd1',
          type: 'delete',
          noteId: 'temp_abc',
          timestamp: 200,
          payload: { type: 'delete' },
        })
      );

      await optimizeQueue();

      const pending = await getPendingOperations();
      expect(pending).toHaveLength(0);
    });

    it('keeps only delete when update + delete exist for same note', async () => {
      const { enqueueOperation, optimizeQueue, getPendingOperations } =
        await import('./offline-queue');

      await enqueueOperation(
        makeOperation({
          id: 'u1',
          type: 'update',
          noteId: 'note-Y',
          timestamp: 100,
          payload: {
            type: 'update',
            notePayload: { title: 'Updated', encrypted_content: 'enc-upd' },
            expectedVersion: 1,
          },
        })
      );
      await enqueueOperation(
        makeOperation({
          id: 'd1',
          type: 'delete',
          noteId: 'note-Y',
          timestamp: 200,
          payload: { type: 'delete' },
        })
      );

      await optimizeQueue();

      const pending = await getPendingOperations();
      expect(pending).toHaveLength(1);
      expect(pending[0].type).toBe('delete');
      expect(pending[0].id).toBe('d1');
    });
  });

  describe('note cache', () => {
    it('caches and retrieves a note', async () => {
      const { cacheNote, getCachedNote } = await import('./offline-queue');

      // We need to first initialize the DB so the cache store exists
      const { initOfflineDatabase } = await import('./offline-queue');
      await initOfflineDatabase();

      const note = {
        id: 'note-1',
        title: 'Cached',
        content: '',
        folder_path: '/',
        version: 1,
        created_at: '2024-01-01',
        updated_at: '2024-01-01',
      };

      await cacheNote('note-1', note);
      const cached = await getCachedNote('note-1');

      expect(cached).not.toBeNull();
      expect(cached?.title).toBe('Cached');
    });

    it('returns null for non-existent cached note', async () => {
      const { getCachedNote, initOfflineDatabase } = await import('./offline-queue');
      await initOfflineDatabase();

      const cached = await getCachedNote('nonexistent');
      expect(cached).toBeNull();
    });
  });
});
