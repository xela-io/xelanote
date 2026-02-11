import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { Note } from '$lib/api';

const handleRemoteUpdate = vi.fn();
const handleRemoteCreate = vi.fn();
const handleRemoteDelete = vi.fn();
const loadTree = vi.fn();
const updateInIndex = vi.fn();
const addToIndex = vi.fn();
const removeFromIndex = vi.fn();
const handleRemoteMetadataUpdate = vi.fn();
const handleRemoteIngredientsUpdate = vi.fn();
const isEncryptionUnlocked = vi.fn().mockReturnValue(false);
const decryptNote = vi.fn();
const getAccessToken = vi.fn().mockReturnValue('token');
const getWsBaseUrl = vi.fn().mockReturnValue('ws://test');

vi.mock('$lib/stores/notes.svelte', () => ({
  handleRemoteUpdate,
  handleRemoteCreate,
  handleRemoteDelete,
}));
vi.mock('$lib/stores/tree.svelte', () => ({ loadTree }));
vi.mock('$lib/stores/search-index.svelte', () => ({
  updateInIndex,
  addToIndex,
  removeFromIndex,
}));
vi.mock('$lib/stores/recipes.svelte', () => ({
  handleRemoteMetadataUpdate,
  handleRemoteIngredientsUpdate,
}));
vi.mock('$lib/stores/encryption.svelte', () => ({
  isEncryptionUnlocked,
  decryptNote,
}));
vi.mock('$lib/stores/auth.svelte', () => ({ getAccessToken }));
vi.mock('$lib/config', () => ({ getWsBaseUrl }));

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  url: string;
  onopen?: () => void;
  onmessage?: (event: { data: string }) => void;
  onclose?: (event: { code: number; reason: string }) => void;
  onerror?: () => void;
  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }
  close(code = 1000, reason = '') {
    this.onclose?.({ code, reason });
  }
  send() {}
}

describe('websocket store', () => {
  beforeEach(async () => {
    MockWebSocket.instances = [];
    vi.clearAllMocks();
    vi.stubGlobal('WebSocket', MockWebSocket);
    vi.resetModules();
    vi.useRealTimers();
  });

  it('should connect and set connected on open', async () => {
    const wsStore = await import('$lib/stores/websocket.svelte');

    wsStore.connect();
    expect(MockWebSocket.instances.length).toBe(1);
    const ws = MockWebSocket.instances[0];
    ws.onopen?.();

    expect(wsStore.getConnected()).toBe(true);
  });

  it('should handle note.deleted events', async () => {
    const wsStore = await import('$lib/stores/websocket.svelte');

    wsStore.connect();
    const ws = MockWebSocket.instances[0];

    const payload: Note = {
      id: 'note-1',
      title: 't',
      content: 'c',
      folder_path: '/',
      version: 1,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    ws.onmessage?.({ data: JSON.stringify({ type: 'note.deleted', payload }) });

    expect(handleRemoteDelete).toHaveBeenCalledWith('note-1');
    expect(loadTree).toHaveBeenCalled();
    expect(removeFromIndex).toHaveBeenCalledWith('note-1');
  });

  it('should reconnect on abnormal close', async () => {
    vi.useFakeTimers();
    vi.resetModules();
    const wsStore = await import('$lib/stores/websocket.svelte');

    wsStore.connect();
    expect(MockWebSocket.instances.length).toBe(1);

    const first = MockWebSocket.instances[0];
    first.onclose?.({ code: 1006, reason: 'abnormal' });

    vi.runAllTimers();

    expect(MockWebSocket.instances.length).toBe(2);
  });
});
