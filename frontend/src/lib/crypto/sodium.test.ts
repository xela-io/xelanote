import { beforeAll, describe, expect, it } from 'vitest';

import * as sodium from './sodium';

beforeAll(async () => {
  await sodium.initSodium();
});

describe('sodium initialization', () => {
  it('should be initialized after initSodium()', () => {
    expect(sodium.isInitialized()).toBe(true);
  });
});

// Note: deriveKey and encrypt/decrypt tests are skipped in jsdom because
// libsodium WASM doesn't fully work in jsdom. These should be tested in
// a real browser environment (e.g., Playwright e2e tests).
describe.skip('deriveKey (requires browser)', () => {
  it('derives deterministic key from password+salt', () => {
    const salt = sodium.generateSalt();
    const key1 = sodium.deriveKey('test123', salt);
    const key2 = sodium.deriveKey('test123', salt);
    expect(key1).toEqual(key2);
  });

  it('derives different keys for different passwords', () => {
    const salt = sodium.generateSalt();
    const key1 = sodium.deriveKey('pass1', salt);
    const key2 = sodium.deriveKey('pass2', salt);
    expect(key1).not.toEqual(key2);
  });

  it('derives different keys for different salts', () => {
    const salt1 = sodium.generateSalt();
    const salt2 = sodium.generateSalt();
    const key1 = sodium.deriveKey('password', salt1);
    const key2 = sodium.deriveKey('password', salt2);
    expect(key1).not.toEqual(key2);
  });

  it('handles unicode passwords correctly via NFC normalization', () => {
    const salt = sodium.generateSalt();
    // NFC vs NFD normalization test
    // café in NFC (precomposed)
    const keyNFC = sodium.deriveKey('café', salt);
    // café in NFD (decomposed: e + combining acute accent)
    const keyNFD = sodium.deriveKey('cafe\u0301', salt);
    expect(keyNFC).toEqual(keyNFD); // Both normalize to NFC
  });

  it('produces 32-byte keys', () => {
    const salt = sodium.generateSalt();
    const key = sodium.deriveKey('password', salt);
    expect(key.length).toBe(32);
  });
});

describe.skip('encrypt/decrypt (requires browser)', () => {
  it('roundtrips correctly', () => {
    const key = sodium.generateDEK();
    const plaintext = sodium.stringToBytes('Hello, World!');
    const encrypted = sodium.encrypt(plaintext, key);
    const decrypted = sodium.decrypt(encrypted, key);
    expect(decrypted).toEqual(plaintext);
  });

  it('returns null for wrong key', () => {
    const key1 = sodium.generateDEK();
    const key2 = sodium.generateDEK();
    const plaintext = sodium.stringToBytes('Secret');
    const encrypted = sodium.encrypt(plaintext, key1);
    const decrypted = sodium.decrypt(encrypted, key2);
    expect(decrypted).toBeNull();
  });

  it('returns null for tampered data (modified byte)', () => {
    const key = sodium.generateDEK();
    const encrypted = sodium.encrypt(sodium.stringToBytes('test'), key);
    // Tamper with the last byte (part of auth tag)
    encrypted[encrypted.length - 1] ^= 0xff;
    const decrypted = sodium.decrypt(encrypted, key);
    expect(decrypted).toBeNull();
  });

  it('returns null for truncated data', () => {
    const key = sodium.generateDEK();
    const encrypted = sodium.encrypt(sodium.stringToBytes('test'), key);
    // Truncate the ciphertext
    const truncated = encrypted.slice(0, 24); // Only nonce
    const decrypted = sodium.decrypt(truncated, key);
    expect(decrypted).toBeNull();
  });

  it('handles empty plaintext', () => {
    const key = sodium.generateDEK();
    const plaintext = sodium.stringToBytes('');
    const encrypted = sodium.encrypt(plaintext, key);
    const decrypted = sodium.decrypt(encrypted, key);
    expect(decrypted).toEqual(plaintext);
  });

  it('handles large plaintext', () => {
    const key = sodium.generateDEK();
    const plaintext = sodium.stringToBytes('A'.repeat(100000)); // 100KB
    const encrypted = sodium.encrypt(plaintext, key);
    const decrypted = sodium.decrypt(encrypted, key);
    expect(decrypted).toEqual(plaintext);
  });

  it('produces different ciphertexts for same plaintext (random nonce)', () => {
    const key = sodium.generateDEK();
    const plaintext = sodium.stringToBytes('same');
    const encrypted1 = sodium.encrypt(plaintext, key);
    const encrypted2 = sodium.encrypt(plaintext, key);
    // Ciphertexts should be different due to random nonces
    expect(encrypted1).not.toEqual(encrypted2);
    // But both should decrypt to the same plaintext
    expect(sodium.decrypt(encrypted1, key)).toEqual(plaintext);
    expect(sodium.decrypt(encrypted2, key)).toEqual(plaintext);
  });
});

describe('generateSalt', () => {
  it('produces 16-byte salt', () => {
    const salt = sodium.generateSalt();
    expect(salt.length).toBe(16);
  });

  it('produces unique salts', () => {
    const salt1 = sodium.generateSalt();
    const salt2 = sodium.generateSalt();
    expect(salt1).not.toEqual(salt2);
  });
});

describe('generateDEK', () => {
  it('produces 32-byte key', () => {
    const dek = sodium.generateDEK();
    expect(dek.length).toBe(32);
  });

  it('produces unique keys', () => {
    const dek1 = sodium.generateDEK();
    const dek2 = sodium.generateDEK();
    expect(dek1).not.toEqual(dek2);
  });
});

describe('string/bytes conversion', () => {
  it('converts string to bytes and back', () => {
    const original = 'Hello, World!';
    const bytes = sodium.stringToBytes(original);
    const result = sodium.bytesToString(bytes);
    expect(result).toBe(original);
  });

  it('handles unicode strings', () => {
    const original = 'Hëllö Wörld! 你好世界 🎉';
    const bytes = sodium.stringToBytes(original);
    const result = sodium.bytesToString(bytes);
    expect(result).toBe(original);
  });

  it('handles empty string', () => {
    const original = '';
    const bytes = sodium.stringToBytes(original);
    const result = sodium.bytesToString(bytes);
    expect(result).toBe(original);
  });
});

describe('base64 encoding', () => {
  it('encodes and decodes correctly', () => {
    const original = new Uint8Array([1, 2, 3, 4, 5, 255, 0, 128]);
    const encoded = sodium.toBase64(original);
    const decoded = sodium.fromBase64(encoded);
    expect(decoded).toEqual(original);
  });

  it('produces URL-safe output without padding', () => {
    const data = new Uint8Array([0xff, 0xfe, 0xfd]); // Would produce + and / in standard base64
    const encoded = sodium.toBase64(data);
    // URL-safe base64 uses - and _ instead of + and /
    expect(encoded).not.toContain('+');
    expect(encoded).not.toContain('/');
    expect(encoded).not.toContain('='); // No padding
  });

  it('handles empty array', () => {
    const original = new Uint8Array([]);
    const encoded = sodium.toBase64(original);
    const decoded = sodium.fromBase64(encoded);
    expect(decoded).toEqual(original);
  });
});

describe.skip('full encryption workflow (requires browser)', () => {
  it('encrypts and decrypts a note-like payload', () => {
    // Simulate what happens in e2e.ts
    const password = 'user-password-123';
    const salt = sodium.generateSalt();

    // Derive KEK from password
    const kek = sodium.deriveKey(password, salt);

    // Generate DEK for the note
    const dek = sodium.generateDEK();

    // Encrypt note content
    const noteContent = 'This is my secret note content!';
    const plaintext = sodium.stringToBytes(noteContent);
    const encryptedContent = sodium.encrypt(plaintext, dek);

    // Wrap DEK with KEK
    const wrappedDEK = sodium.encrypt(dek, kek);

    // Store as base64
    const storedContent = sodium.toBase64(encryptedContent);
    const storedDEK = sodium.toBase64(wrappedDEK);

    // --- Later, to decrypt ---

    // Derive KEK again from password
    const kek2 = sodium.deriveKey(password, salt);
    expect(kek2).toEqual(kek);

    // Unwrap DEK
    const wrappedDEKBytes = sodium.fromBase64(storedDEK);
    const unwrappedDEK = sodium.decrypt(wrappedDEKBytes, kek2);
    expect(unwrappedDEK).not.toBeNull();
    expect(unwrappedDEK).toEqual(dek);

    // Decrypt content
    const encryptedContentBytes = sodium.fromBase64(storedContent);
    const decryptedContent = sodium.decrypt(encryptedContentBytes, unwrappedDEK!);
    expect(decryptedContent).not.toBeNull();

    const result = sodium.bytesToString(decryptedContent!);
    expect(result).toBe(noteContent);
  });
});
