/// <reference lib="webworker" />
import { argon2id } from '@noble/hashes/argon2.js';

let initialized = false;

// KDF Parameters (same as main thread)
const KDF_OPSLIMIT = 3; // OPSLIMIT_INTERACTIVE
const KDF_MEMLIMIT = 67108864; // MEMLIMIT_INTERACTIVE = 64MB

self.onmessage = async (e: MessageEvent) => {
  const { type, password, salt, id } = e.data;

  if (type === 'init') {
    if (!initialized) {
      initialized = true;
      console.log('[WORKER] Argon2id via @noble/hashes initialized (pure JS)');
    }
    self.postMessage({ type: 'init-done', id });
    return;
  }

  if (type === 'derive') {
    try {
      // NFC-Normalisierung
      const normalizedPw = password.normalize('NFC');

      // Use @noble/hashes argon2id (pure JS, no WASM issues)
      const key = argon2id(normalizedPw, new Uint8Array(salt), {
        t: KDF_OPSLIMIT,
        m: KDF_MEMLIMIT / 1024, // @noble/hashes expects memory in KB
        p: 1, // parallelism
        dkLen: 32, // 256-bit output
      });

      // Transfer ownership für Performance
      self.postMessage({ type: 'derive-done', key: key.buffer, id }, [key.buffer]);
    } catch (error) {
      self.postMessage({
        type: 'derive-error',
        error: error instanceof Error ? error.message : 'Unknown error',
        id,
      });
    }
  }
};
