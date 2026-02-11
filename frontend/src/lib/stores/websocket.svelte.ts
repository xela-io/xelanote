// WebSocket store for real-time updates
// Using Svelte 5 runes

import type { Note } from '$lib/api';
import { getWsBaseUrl } from '$lib/config';
import type { EncryptedPayload } from '$lib/crypto/e2e';
import { parseEncryptionMetadata } from '$lib/stores/encryption-metadata';

import { getAccessToken } from './auth.svelte';
import * as encryption from './encryption.svelte';
import * as notes from './notes.svelte';
import * as recipes from './recipes.svelte';
import * as searchIndex from './search-index.svelte';
import * as tree from './tree.svelte';

type WebSocketMessage = { type: string; payload: unknown };
type NoteEventPayload = {
  id: string;
  title?: string;
  content_encrypted?: boolean;
  encrypted_content?: string;
  encryption_metadata?: string | null;
  encrypted_title?: string | null;
};

type RecipeEventPayload = {
  note_id: string;
};

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object';
}

function parseMessage(raw: unknown): WebSocketMessage | null {
  if (!isObjectRecord(raw)) return null;
  if (typeof raw.type !== 'string') return null;
  return { type: raw.type, payload: raw.payload };
}

function parseNote(payload: unknown): Note | null {
  if (!isObjectRecord(payload)) return null;
  if (
    typeof payload.id !== 'string' ||
    typeof payload.title !== 'string' ||
    typeof payload.content !== 'string' ||
    typeof payload.folder_path !== 'string' ||
    typeof payload.version !== 'number' ||
    typeof payload.created_at !== 'string' ||
    typeof payload.updated_at !== 'string'
  ) {
    return null;
  }
  return {
    id: payload.id,
    title: payload.title,
    content: payload.content,
    folder_path: payload.folder_path,
    version: payload.version,
    created_at: payload.created_at,
    updated_at: payload.updated_at,
  };
}

function parseNoteEventPayload(payload: unknown): NoteEventPayload | null {
  if (!isObjectRecord(payload)) return null;
  if (typeof payload.id !== 'string') return null;
  return payload as NoteEventPayload;
}

function parseRecipeEventPayload(payload: unknown): RecipeEventPayload | null {
  if (!isObjectRecord(payload)) return null;
  if (typeof payload.note_id !== 'string') return null;
  return payload as RecipeEventPayload;
}

let ws: WebSocket | null = null;
let connected = $state(false);
let connecting = false; // Guard against concurrent connection attempts
let reconnectAttempts = 0;
let reconnectTimeout: number | null = null;
let intentionalDisconnect = false; // Flag to prevent reconnect on intentional disconnect

export function getConnected() {
  return connected;
}

function getWsUrl(): string {
  return getWsBaseUrl();
}

export function connect() {
  const token = getAccessToken();
  if (!token) {
    console.log('WebSocket: No token, skipping connection');
    return;
  }

  // Already connected or connecting - skip
  if (connected || connecting) {
    console.log('WebSocket: Already connected or connecting, skipping');
    return;
  }

  // Close existing connection cleanly if any
  if (ws) {
    intentionalDisconnect = true;
    ws.close(1000, 'Reconnecting');
    ws = null;
  }

  connecting = true;
  console.log('WebSocket: Connecting...');

  try {
    // Token is sent via HttpOnly cookie (set by auth flow), not query param
    ws = new WebSocket(getWsUrl());
  } catch (e) {
    console.error('WebSocket: Failed to create connection', e);
    connecting = false;
    return;
  }

  ws.onopen = () => {
    connecting = false;
    connected = true;
    reconnectAttempts = 0;
    intentionalDisconnect = false;
    console.log('WebSocket: Connected');
  };

  ws.onmessage = (event) => {
    try {
      const message = parseMessage(JSON.parse(event.data));
      if (!message) {
        console.warn('WebSocket: Invalid message payload shape');
        return;
      }
      handleMessage(message);
    } catch (e) {
      console.error('WebSocket: Failed to parse message', e);
    }
  };

  ws.onclose = (event) => {
    connecting = false;
    connected = false;
    ws = null;
    console.log('WebSocket: Disconnected', event.code, event.reason);

    // Don't reconnect on intentional disconnect, normal closure (1000) or unauthorized (1008)
    if (intentionalDisconnect || event.code === 1000 || event.code === 1008) {
      console.log('WebSocket: Not reconnecting (intentional or normal closure)');
      intentionalDisconnect = false;
      return;
    }

    // Schedule reconnect with exponential backoff
    scheduleReconnect();
  };

  ws.onerror = (error) => {
    console.error('WebSocket: Error', error);
  };
}

function handleMessage(message: WebSocketMessage) {
  console.log('WebSocket: Message received', message.type);

  switch (message.type) {
    case 'note.updated':
      {
        const note = parseNote(message.payload);
        if (!note) {
          console.warn('WebSocket: Invalid note.updated payload');
          return;
        }
        notes.handleRemoteUpdate(note);
      }
      tree.loadTree();
      {
        const notePayload = parseNoteEventPayload(message.payload);
        if (!notePayload) return;
        if (notePayload.content_encrypted && encryption.isEncryptionUnlocked()) {
          if (notePayload.encrypted_content) {
            try {
              const encryptedPayload: EncryptedPayload = {
                ciphertext: notePayload.encrypted_content,
                metadata: parseEncryptionMetadata(notePayload.encryption_metadata),
              };
              const decrypted = encryption.decryptNote(
                notePayload.encrypted_title || null,
                encryptedPayload
              );
              searchIndex.updateInIndex(
                notePayload.id,
                decrypted.title || notePayload.title || '',
                decrypted.content
              );
            } catch {
              /* silent - index repairs on next build */
            }
          }
        }
      }
      break;
    case 'note.created':
      {
        const note = parseNote(message.payload);
        if (!note) {
          console.warn('WebSocket: Invalid note.created payload');
          return;
        }
        notes.handleRemoteCreate(note);
      }
      tree.loadTree();
      {
        const notePayload = parseNoteEventPayload(message.payload);
        if (!notePayload) return;
        if (notePayload.content_encrypted && encryption.isEncryptionUnlocked()) {
          if (notePayload.encrypted_content) {
            try {
              const encryptedPayload: EncryptedPayload = {
                ciphertext: notePayload.encrypted_content,
                metadata: parseEncryptionMetadata(notePayload.encryption_metadata),
              };
              const decrypted = encryption.decryptNote(
                notePayload.encrypted_title || null,
                encryptedPayload
              );
              searchIndex.addToIndex(
                notePayload.id,
                decrypted.title || notePayload.title || '',
                decrypted.content
              );
            } catch {
              /* silent */
            }
          }
        }
      }
      break;
    case 'note.deleted':
      {
        const notePayload = parseNoteEventPayload(message.payload);
        if (!notePayload) {
          console.warn('WebSocket: Invalid note.deleted payload');
          return;
        }
        notes.handleRemoteDelete(notePayload.id);
        tree.loadTree();
        searchIndex.removeFromIndex(notePayload.id);
      }
      break;
    case 'recipe.metadata.updated':
      {
        const payload = parseRecipeEventPayload(message.payload);
        if (!payload) {
          console.warn('WebSocket: Invalid recipe.metadata.updated payload');
          return;
        }
        recipes.handleRemoteMetadataUpdate(payload);
      }
      break;
    case 'recipe.ingredients.updated':
      {
        const payload = parseRecipeEventPayload(message.payload);
        if (!payload) {
          console.warn('WebSocket: Invalid recipe.ingredients.updated payload');
          return;
        }
        recipes.handleRemoteIngredientsUpdate(payload);
      }
      break;
    default:
      console.log('WebSocket: Unknown message type', message.type);
  }
}

// Exponential backoff reconnect
function scheduleReconnect() {
  const BASE_RECONNECT_DELAY_MS = 1000;
  const MAX_RECONNECT_DELAY_MS = 30000;
  const MAX_RECONNECT_ATTEMPTS = 10;

  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }

  // Limit reconnect attempts
  if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
    console.log('WebSocket: Max reconnect attempts reached, giving up');
    return;
  }

  const delay = Math.min(BASE_RECONNECT_DELAY_MS * Math.pow(2, reconnectAttempts), MAX_RECONNECT_DELAY_MS);
  reconnectAttempts++;

  console.log(`WebSocket: Reconnecting in ${delay}ms (attempt ${reconnectAttempts})`);

  reconnectTimeout = window.setTimeout(() => {
    reconnectTimeout = null;
    connect();
  }, delay);
}

export function disconnect() {
  intentionalDisconnect = true;

  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }

  if (ws) {
    ws.close(1000, 'Client disconnect');
    ws = null;
  }

  connecting = false;
  connected = false;
  reconnectAttempts = 0;
}

// Page Visibility API - reconnect when page becomes visible
if (typeof document !== 'undefined') {
  document.addEventListener('visibilitychange', () => {
    // Only reconnect if we have a token and are not already connected/connecting
    if (document.visibilityState === 'visible' && !connected && !connecting && getAccessToken()) {
      console.log('WebSocket: Page visible, reconnecting');
      connect();
    }
  });
}
