// WebSocket store for real-time updates
// Using Svelte 5 runes

import { getAccessToken } from './auth.svelte';
import * as notes from './notes.svelte';
import * as recipes from './recipes.svelte';
import * as tree from './tree.svelte';
import * as searchIndex from './search-index.svelte';
import * as encryption from './encryption.svelte';
import type { EncryptedPayload } from '$lib/crypto/e2e';
import { getWsBaseUrl } from '$lib/config';

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
      const message = JSON.parse(event.data);
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

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- WebSocket protocol payloads vary by message type
function handleMessage(message: { type: string; payload: any }) {
  console.log('WebSocket: Message received', message.type);

  switch (message.type) {
    case 'note.updated':
      notes.handleRemoteUpdate(message.payload);
      tree.loadTree();
      if (message.payload.content_encrypted && encryption.isEncryptionUnlocked()) {
        try {
          const payload: EncryptedPayload = {
            ciphertext: message.payload.encrypted_content,
            metadata: JSON.parse(message.payload.encryption_metadata || '{}'),
          };
          const decrypted = encryption.decryptNote(
            message.payload.encrypted_title || null,
            payload
          );
          searchIndex.updateInIndex(
            message.payload.id,
            decrypted.title || message.payload.title || '',
            decrypted.content
          );
        } catch {
          /* silent - index repairs on next build */
        }
      }
      break;
    case 'note.created':
      notes.handleRemoteCreate(message.payload);
      tree.loadTree();
      if (message.payload.content_encrypted && encryption.isEncryptionUnlocked()) {
        try {
          const payload: EncryptedPayload = {
            ciphertext: message.payload.encrypted_content,
            metadata: JSON.parse(message.payload.encryption_metadata || '{}'),
          };
          const decrypted = encryption.decryptNote(
            message.payload.encrypted_title || null,
            payload
          );
          searchIndex.addToIndex(
            message.payload.id,
            decrypted.title || message.payload.title || '',
            decrypted.content
          );
        } catch {
          /* silent */
        }
      }
      break;
    case 'note.deleted':
      notes.handleRemoteDelete(message.payload.id);
      tree.loadTree();
      searchIndex.removeFromIndex(message.payload.id);
      break;
    case 'recipe.metadata.updated':
      recipes.handleRemoteMetadataUpdate(message.payload);
      break;
    case 'recipe.ingredients.updated':
      recipes.handleRemoteIngredientsUpdate(message.payload);
      break;
    default:
      console.log('WebSocket: Unknown message type', message.type);
  }
}

// Exponential backoff reconnect
function scheduleReconnect() {
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }

  // Limit reconnect attempts
  if (reconnectAttempts >= 10) {
    console.log('WebSocket: Max reconnect attempts reached, giving up');
    return;
  }

  const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
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
