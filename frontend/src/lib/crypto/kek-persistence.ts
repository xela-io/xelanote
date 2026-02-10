/**
 * KEK (Key Encryption Key) Persistence Module
 *
 * Provides hybrid KEK persistence for xelanote:
 * - Paranoid: Memory-only (existing behavior)
 * - Balanced: JWK-wrapped KEK in IndexedDB + Auto-Lock
 * - Convenient: Balanced + WebAuthn Biometric
 *
 * Security Model:
 * - KEK wrapped with AES-GCM-256 wrapper key
 * - Wrapper key exported as JWK and stored in IndexedDB
 * - Re-imported as non-extractable on load
 * - XSS = game over (same as 1Password, Bitwarden in browser)
 */

import { warning } from '$lib/stores/toast.svelte';
import * as api from '$lib/api';
import * as encryption from '$lib/stores/encryption.svelte';

export type SecurityLevel = 'paranoid' | 'balanced' | 'convenient';

/**
 * IndexedDB schema for KEK storage
 */
export interface KEKStorageRecord {
  userId: number;
  wrapperKeyJWK: JsonWebKey; // Exported AES-GCM-256 key (serializable!)
  wrappedKEK: ArrayBuffer; // Encrypted libsodium KEK
  wrappingIV: ArrayBuffer; // 12-byte IV for AES-GCM
  wrappingAlgorithm: 'AES-GCM';
  wrappingVersion: 1;
  createdAt: number;
  lastUnwrapped: number;
  securityLevel: SecurityLevel;
}

const DB_NAME = 'xelanote-encryption';
const STORE_NAME = 'kek_storage';
const DB_VERSION = 1;

/**
 * Initialize IndexedDB database.
 * CRITICAL: Catch failures and force paranoid mode.
 *
 * @throws Error if IndexedDB is unavailable (e.g., private browsing)
 */
export async function initKEKDatabase(): Promise<void> {
  try {
    const request = indexedDB.open(DB_NAME, DB_VERSION);

    request.onupgradeneeded = (event) => {
      const db = (event.target as IDBOpenDBRequest).result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: 'userId' });
      }
    };

    return new Promise((resolve, reject) => {
      request.onsuccess = () => {
        request.result.close();
        resolve();
      };
      request.onerror = () => reject(request.error);
    });
  } catch (err) {
    console.error('IndexedDB unavailable (private browsing?):', err);

    // Force paranoid mode + sync backend
    await handleIndexedDBFailure();

    throw err; // Re-throw for caller
  }
}

/**
 * Handle IndexedDB initialization failure.
 * Force paranoid mode and sync to backend.
 *
 * ⚠️ REFINEMENT: Non-noisy error handling
 * Only show toast if user actually had balanced/convenient set.
 * Don't spam users already on paranoid mode.
 *
 * @returns true if toast should be shown, false otherwise
 */
export async function handleIndexedDBFailure(): Promise<boolean> {
  // Check current backend preference
  let shouldNotify = false;
  try {
    const prefs = await api.getUserPreferences();
    shouldNotify = prefs.security_level !== 'paranoid';
  } catch (_err) {
    // If can't check, assume we should notify
    shouldNotify = true;
  }

  // Update local state
  encryption.setSecurityLevel('paranoid');

  // Sync to backend
  try {
    await api.updateSecurityPreferences({
      security_level: 'paranoid',
    });
  } catch (err) {
    console.error('Failed to sync paranoid mode to backend:', err);
  }

  return shouldNotify;
}

/**
 * Generate wrapper key for KEK encryption.
 * Returns both CryptoKey (for immediate use) and JWK (for storage).
 */
async function generateWrapperKey(): Promise<{
  key: CryptoKey;
  jwk: JsonWebKey;
}> {
  // Generate EXTRACTABLE key (needed for JWK export)
  const key = await crypto.subtle.generateKey(
    { name: 'AES-GCM', length: 256 },
    true, // ✅ extractable = true (for JWK export)
    ['encrypt', 'decrypt'] // ✅ FIX: Use encrypt/decrypt for raw bytes, not wrapKey/unwrapKey
  );

  // Export as JWK for IndexedDB storage
  const jwk = await crypto.subtle.exportKey('jwk', key);

  return { key, jwk };
}

/**
 * Load wrapper key from JWK.
 * Re-imports as non-extractable for security.
 */
async function loadWrapperKey(jwk: JsonWebKey): Promise<CryptoKey> {
  return await crypto.subtle.importKey(
    'jwk',
    jwk,
    { name: 'AES-GCM' },
    false, // Import as non-extractable
    ['encrypt', 'decrypt'] // ✅ FIX: Use encrypt/decrypt for raw bytes
  );
}

/**
 * Wrap libsodium KEK with wrapper key using AES-GCM.
 */
async function wrapKEK(
  kek: Uint8Array,
  wrapperKey: CryptoKey
): Promise<{ wrappedKEK: ArrayBuffer; iv: ArrayBuffer }> {
  // Generate fresh IV for each wrap (CRITICAL for AES-GCM)
  const iv = crypto.getRandomValues(new Uint8Array(12));

  // Encrypt KEK (ensure proper ArrayBuffer for TypedArray views)
  // Using kek.buffer directly could encrypt wrong data if KEK is a TypedArray view
  // Create a new Uint8Array copy to ensure we have a proper ArrayBuffer
  const kekCopy = new Uint8Array(kek);
  const wrappedKEK = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, wrapperKey, kekCopy);

  return { wrappedKEK, iv: iv.buffer };
}

/**
 * Unwrap KEK using wrapper key.
 */
async function unwrapKEK(
  wrappedKEK: ArrayBuffer,
  iv: ArrayBuffer,
  wrapperKey: CryptoKey
): Promise<Uint8Array> {
  const decrypted = await crypto.subtle.decrypt({ name: 'AES-GCM', iv }, wrapperKey, wrappedKEK);

  return new Uint8Array(decrypted);
}

/**
 * Persist KEK to IndexedDB.
 * Called after successful login or KEK re-wrap.
 */
export async function persistKEK(
  userId: number,
  kek: Uint8Array,
  securityLevel: SecurityLevel
): Promise<void> {
  try {
    // Generate or load wrapper key
    const { key: wrapperKey, jwk: wrapperKeyJWK } = await generateWrapperKey();

    // Wrap KEK
    const { wrappedKEK, iv } = await wrapKEK(kek, wrapperKey);

    // Store in IndexedDB
    const db = await openDatabase();
    const tx = db.transaction(STORE_NAME, 'readwrite');
    const store = tx.objectStore(STORE_NAME);

    const record: KEKStorageRecord = {
      userId,
      wrapperKeyJWK, // ✅ FIX: JWK (serializable)
      wrappedKEK,
      wrappingIV: iv,
      wrappingAlgorithm: 'AES-GCM',
      wrappingVersion: 1,
      createdAt: Date.now(),
      lastUnwrapped: Date.now(),
      securityLevel,
    };

    await new Promise<void>((resolve, reject) => {
      const req = store.put(record);
      req.onsuccess = () => resolve();
      req.onerror = () => reject(req.error);
    });

    db.close();
  } catch (err) {
    console.error('Failed to persist KEK:', err);
    throw err;
  }
}

/**
 * Load and unwrap KEK from IndexedDB.
 * Returns null if not found or corruption detected.
 */
export async function loadPersistedKEK(userId: number): Promise<Uint8Array | null> {
  try {
    const db = await openDatabase();
    const tx = db.transaction(STORE_NAME, 'readonly');
    const store = tx.objectStore(STORE_NAME);

    const record = await new Promise<KEKStorageRecord | undefined>((resolve, reject) => {
      const req = store.get(userId);
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });

    db.close();

    if (!record) return null;

    // Load wrapper key from JWK
    const wrapperKey = await loadWrapperKey(record.wrapperKeyJWK);

    // Unwrap KEK
    const kek = await unwrapKEK(record.wrappedKEK, record.wrappingIV, wrapperKey);

    // Update last unwrapped timestamp
    await touchKEKAccess(userId);

    return kek;
  } catch (err) {
    console.error('Failed to load persisted KEK:', err);

    // ✅ FIX: Explicit cleanup on corruption
    await clearPersistedKEK(userId);

    return null;
  }
}

/**
 * Clear KEK from IndexedDB.
 * Called on logout or switch to paranoid mode.
 */
export async function clearPersistedKEK(userId: number): Promise<void> {
  try {
    const db = await openDatabase();
    const tx = db.transaction(STORE_NAME, 'readwrite');
    const store = tx.objectStore(STORE_NAME);

    await new Promise<void>((resolve, reject) => {
      const req = store.delete(userId);
      req.onsuccess = () => resolve();
      req.onerror = () => reject(req.error);
    });

    db.close();
  } catch (err) {
    console.error('Failed to clear persisted KEK:', err);
  }
}

/**
 * Check if persisted KEK exists for user.
 */
export async function hasPersistedKEK(userId: number): Promise<boolean> {
  try {
    const db = await openDatabase();
    const tx = db.transaction(STORE_NAME, 'readonly');
    const store = tx.objectStore(STORE_NAME);

    const record = await new Promise<KEKStorageRecord | undefined>((resolve, reject) => {
      const req = store.get(userId);
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });

    db.close();

    return !!record;
  } catch (_err) {
    return false;
  }
}

/**
 * Update last unwrapped timestamp.
 */
export async function touchKEKAccess(userId: number): Promise<void> {
  try {
    const db = await openDatabase();
    const tx = db.transaction(STORE_NAME, 'readwrite');
    const store = tx.objectStore(STORE_NAME);

    const record = await new Promise<KEKStorageRecord | undefined>((resolve, reject) => {
      const req = store.get(userId);
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });

    if (record) {
      record.lastUnwrapped = Date.now();
      await new Promise<void>((resolve, reject) => {
        const req = store.put(record);
        req.onsuccess = () => resolve();
        req.onerror = () => reject(req.error);
      });
    }

    db.close();
  } catch (err) {
    console.error('Failed to touch KEK access:', err);
  }
}

/**
 * Open IndexedDB database.
 * Handles errors with fallback to paranoid mode.
 */
async function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);

    request.onsuccess = () => resolve(request.result);

    request.onerror = async () => {
      console.error('IndexedDB operation failed:', request.error);

      // Force paranoid mode
      const shouldNotify = await handleIndexedDBFailure();
      if (shouldNotify) {
        warning('IndexedDB nicht verfügbar - Paranoid-Modus aktiv');
      }

      reject(request.error);
    };

    request.onupgradeneeded = (event) => {
      const db = (event.target as IDBOpenDBRequest).result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: 'userId' });
      }
    };
  });
}
