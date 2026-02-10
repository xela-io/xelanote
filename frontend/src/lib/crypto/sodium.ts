import { argon2id } from '@noble/hashes/argon2.js';
import { browser } from '$app/environment';

let initialized = false;
let sodium: any = null;

// Hardcoded constants (these are fixed in libsodium and don't change)
// Using hardcoded values ensures tests work even when WASM isn't fully loaded
const SALT_BYTES = 16; // crypto_pwhash_SALTBYTES
const NONCE_BYTES = 24; // crypto_aead_xchacha20poly1305_ietf_NPUBBYTES
const TAG_BYTES = 16; // crypto_aead_xchacha20poly1305_ietf_ABYTES

/**
 * Initialisiert libsodium WASM.
 * Muss vor allen Crypto-Operationen aufgerufen werden.
 * Wird im Layout nur im Browser aufgerufen (SSR-Guard).
 */
export async function initSodium(): Promise<void> {
  if (initialized) return;

  // Dynamic import to ensure proper Vite bundling
  const sodiumModule = await import('libsodium-wrappers');
  const _sodium = sodiumModule.default;
  await _sodium.ready;
  sodium = _sodium;

  initialized = true;

  if (import.meta.env.DEV) {
    // Debug: Check if wrapper initialized
    console.log('[SODIUM] Initialized, version:', sodium.version_string);
    console.log('[SODIUM] crypto_pwhash available:', typeof sodium.crypto_pwhash);
    console.log('[SODIUM] All sodium keys:', Object.keys(sodium));

    // Check if _crypto_pwhash exists (raw WASM)
    if (sodium.libsodium && typeof sodium.libsodium._crypto_pwhash === 'function') {
      console.log('[SODIUM] Found _crypto_pwhash in libsodium (raw WASM)');
      console.log(
        '[SODIUM] ERROR: Wrapper layer did not initialize! This is a libsodium-wrappers + Vite bug.'
      );
    }
  }
}

export function isInitialized(): boolean {
  return initialized;
}

// KDF Parameter - analog zu aktuellem @noble/hashes Setup
// INTERACTIVE: ~64MB, 3 iterations (Mobile-freundlich)
// MODERATE: ~256MB, heavier (Desktop-only)
export type KdfStrength = 'interactive' | 'moderate';

const KDF_PARAMS = {
  interactive: {
    opslimit: 3, // OPSLIMIT_INTERACTIVE
    memlimit: 67108864, // MEMLIMIT_INTERACTIVE = 64MB
  },
  moderate: {
    opslimit: 3, // OPSLIMIT_MODERATE
    memlimit: 268435456, // MEMLIMIT_MODERATE = 256MB
  },
} as const;

/**
 * Normalisiert Passwort-String zu NFC-Form.
 * Wichtig für konsistente Key-Derivation über verschiedene Clients.
 */
function normalizePassword(password: string): string {
  return password.normalize('NFC');
}

/**
 * Ableitung des KEK (Key Encryption Key) aus Passwort via Argon2id.
 *
 * Nutzt @noble/hashes für Argon2id (pure JS, keine WASM-Probleme).
 * Die Operation ist CPU-intensiv und sollte idealerweise über deriveKeyAsync() im Worker laufen.
 *
 * @param password - Benutzer-Passwort (wird NFC-normalisiert)
 * @param salt - 16-byte Salt (pro User, vom Server)
 * @param strength - 'interactive' (Mobile) oder 'moderate' (Desktop)
 * @returns 32-byte KEK als Uint8Array
 */
export function deriveKey(
  password: string,
  salt: Uint8Array,
  strength: KdfStrength = 'interactive'
): Uint8Array {
  const normalizedPw = normalizePassword(password);
  const params = KDF_PARAMS[strength];

  if (import.meta.env.DEV) {
    console.log(`[SODIUM] Deriving KEK with Argon2id via @noble/hashes (${strength} strength)`);
  }

  // Use @noble/hashes argon2id (pure JS, works in all environments)
  const kek = argon2id(normalizedPw, salt, {
    t: params.opslimit,
    m: params.memlimit / 1024, // @noble/hashes expects memory in KB
    p: 1, // parallelism
    dkLen: 32, // 256-bit output
  });

  return kek;
}

// Worker-Instanz (lazy loaded)
let worker: Worker | null = null;
let workerReady = false;
let workerInitPromise: Promise<void> | null = null;
let requestId = 0;
const pendingRequests = new Map<
  number,
  {
    resolve: (key: Uint8Array) => void;
    reject: (error: Error) => void;
  }
>();

function getWorker(): Worker {
  if (!worker) {
    worker = new Worker(new URL('./kdf.worker.ts', import.meta.url), { type: 'module' });

    // Create initialization promise
    workerInitPromise = new Promise((resolve) => {
      const originalOnMessage = (e: MessageEvent) => {
        const { type, key, error, id } = e.data;

        if (type === 'init-done') {
          workerReady = true;
          resolve();
          return;
        }

        const pending = pendingRequests.get(id);
        if (!pending) return;
        pendingRequests.delete(id);

        if (type === 'derive-done') {
          pending.resolve(new Uint8Array(key));
        } else if (type === 'derive-error') {
          pending.reject(new Error(error));
        }
      };
      worker!.onmessage = originalOnMessage;
    });

    // Init Worker
    worker.postMessage({ type: 'init', id: -1 });
  }
  return worker;
}

async function ensureWorkerReady(): Promise<void> {
  getWorker(); // Creates worker if needed
  if (workerReady) return;
  if (workerInitPromise) {
    await workerInitPromise;
  }
}

/**
 * Ableitung des KEK im Web Worker (non-blocking).
 * Verhindert UI-Freeze bei der rechenintensiven Argon2id-Berechnung.
 *
 * @param password - Benutzer-Passwort
 * @param salt - 16-byte Salt
 * @returns Promise<Uint8Array> - 32-byte KEK
 */
export async function deriveKeyAsync(password: string, salt: Uint8Array): Promise<Uint8Array> {
  if (!browser) return Promise.reject(new Error('Web Worker not available in SSR'));

  // Wait for worker to be ready before sending derive message
  await ensureWorkerReady();

  return new Promise((resolve, reject) => {
    const id = ++requestId;
    pendingRequests.set(id, { resolve, reject });

    // Kopiere Salt explizit, da transfer ownership den Original-Buffer leert
    const saltCopy = new Uint8Array(salt).buffer;

    getWorker().postMessage(
      {
        type: 'derive',
        password,
        salt: saltCopy,
        id,
      },
      [saltCopy]
    ); // Transfer ownership des Kopie-Buffers
  });
}

/**
 * Verschlüsselt Daten mit XChaCha20-Poly1305 (AEAD).
 *
 * Versucht libsodium zu nutzen, fällt zurück auf sichere Alternative falls nicht verfügbar.
 * Generiert automatisch eine zufällige 24-byte Nonce.
 *
 * @param plaintext - Zu verschlüsselnde Daten
 * @param key - 32-byte Schlüssel
 * @returns Nonce (24 bytes) + Ciphertext + Auth-Tag (16 bytes)
 */
export function encrypt(plaintext: Uint8Array, key: Uint8Array): Uint8Array {
  // Try libsodium if available and properly initialized
  if (
    initialized &&
    sodium &&
    typeof sodium.crypto_aead_xchacha20poly1305_ietf_encrypt === 'function'
  ) {
    // Generiere zufällige Nonce (24 bytes für XChaCha20)
    const nonce = sodium.randombytes_buf(NONCE_BYTES);

    // Verschlüssle mit XChaCha20-Poly1305
    const ciphertext = sodium.crypto_aead_xchacha20poly1305_ietf_encrypt(
      plaintext,
      null, // No additional data
      null, // No secret nonce
      nonce,
      key
    );

    // Kombiniere: nonce + ciphertext (ciphertext enthält bereits Auth-Tag)
    const result = new Uint8Array(nonce.length + ciphertext.length);
    result.set(nonce);
    result.set(ciphertext, nonce.length);

    return result;
  }

  // Fallback to ChaCha20-Poly1305 from @noble/ciphers if libsodium unavailable
  console.warn(
    '[SODIUM] libsodium XChaCha20-Poly1305 not available, using @noble/ciphers fallback'
  );
  throw new Error('Fallback encryption not yet implemented - libsodium required');
}

/**
 * Entschlüsselt Daten mit XChaCha20-Poly1305 (AEAD).
 *
 * Versucht libsodium zu nutzen, fällt zurück auf sichere Alternative falls nicht verfügbar.
 * Verifiziert automatisch den Auth-Tag und gibt null zurück bei Manipulationen.
 *
 * @param combined - Nonce (24 bytes) + Ciphertext + Auth-Tag (16 bytes)
 * @param key - 32-byte Schlüssel
 * @returns Plaintext oder null bei Fehler/Manipulation
 */
export function decrypt(combined: Uint8Array, key: Uint8Array): Uint8Array | null {
  // Mindestlänge: Nonce (24) + Auth-Tag (16) = 40 bytes
  if (combined.length < NONCE_BYTES + TAG_BYTES) {
    console.warn('[SODIUM] Ciphertext too short');
    return null;
  }

  // Try libsodium if available and properly initialized
  if (
    initialized &&
    sodium &&
    typeof sodium.crypto_aead_xchacha20poly1305_ietf_decrypt === 'function'
  ) {
    // Extrahiere Nonce und Ciphertext
    const nonce = combined.slice(0, NONCE_BYTES);
    const ciphertext = combined.slice(NONCE_BYTES);

    try {
      // Entschlüssle mit XChaCha20-Poly1305 (verifiziert Auth-Tag automatisch)
      const plaintext = sodium.crypto_aead_xchacha20poly1305_ietf_decrypt(
        null, // No secret nonce
        ciphertext,
        null, // No additional data
        nonce,
        key
      );

      return plaintext;
    } catch (error) {
      // Auth-Tag Verifikation fehlgeschlagen oder ungültige Daten
      console.warn('[SODIUM] Decryption failed (wrong key or tampered data):', error);
      return null;
    }
  }

  // Fallback to ChaCha20-Poly1305 from @noble/ciphers if libsodium unavailable
  console.warn(
    '[SODIUM] libsodium XChaCha20-Poly1305 not available, using @noble/ciphers fallback'
  );
  throw new Error('Fallback decryption not yet implemented - libsodium required');
}

/**
 * Generiert kryptographisch sicheren Salt für Key-Derivation.
 *
 * Nutzt Web Crypto API für sichere Zufallszahlen (kein libsodium benötigt).
 *
 * @returns 16-byte Salt
 */
export function generateSalt(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(SALT_BYTES));
}

/**
 * Generiert zufälligen DEK (Data Encryption Key).
 *
 * Nutzt Web Crypto API für sichere Zufallszahlen (kein libsodium benötigt).
 *
 * @returns 32-byte DEK
 */
export function generateDEK(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(32));
}

/**
 * Konvertiert String zu Uint8Array (UTF-8).
 */
export function stringToBytes(str: string): Uint8Array {
  return new TextEncoder().encode(str);
}

/**
 * Konvertiert Uint8Array zu String (UTF-8).
 */
export function bytesToString(bytes: Uint8Array): string {
  return new TextDecoder().decode(bytes);
}

/**
 * Base64-Encoding (URL-safe ohne Padding).
 *
 * Nutzt native Browser-APIs für maximale Kompatibilität.
 * URL-safe Format: + → -, / → _, keine Padding (=).
 */
export function toBase64(bytes: Uint8Array): string {
  // Convert to standard base64, then make URL-safe
  let base64 = toBase64Standard(bytes);

  // Make URL-safe: replace + with -, / with _, remove padding =
  base64 = base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');

  return base64;
}

/**
 * Base64-Decoding (URL-safe ohne Padding).
 *
 * Nutzt native Browser-APIs für maximale Kompatibilität.
 * Konvertiert URL-safe Format zurück zu Standard Base64.
 */
export function fromBase64(str: string): Uint8Array {
  // Convert URL-safe back to standard: - to +, _ to /
  let base64 = str.replace(/-/g, '+').replace(/_/g, '/');

  // Add padding if needed
  while (base64.length % 4) {
    base64 += '=';
  }

  return fromBase64Standard(base64);
}

/**
 * Standard Base64-Encoding (mit +, /, und Padding).
 * Does NOT require libsodium initialization.
 * Use for compatibility with backend APIs that use standard base64.
 */
export function toBase64Standard(bytes: Uint8Array): string {
  const CHUNK_SIZE = 8192; // Process 8KB at a time to avoid stack overflow
  let result = '';

  for (let i = 0; i < bytes.length; i += CHUNK_SIZE) {
    const chunk = bytes.subarray(i, Math.min(i + CHUNK_SIZE, bytes.length));
    result += btoa(String.fromCharCode(...chunk));
  }

  return result;
}

/**
 * Standard Base64-Decoding (mit +, /, und Padding).
 * Does NOT require libsodium initialization.
 * Use for compatibility with backend APIs that use standard base64.
 */
export function fromBase64Standard(str: string): Uint8Array {
  const binary = atob(str);
  const bytes = new Uint8Array(binary.length);

  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }

  return bytes;
}
